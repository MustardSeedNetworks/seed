package dhcp

//
// The Monitor uses gopacket/pcap for real-time DHCP packet capture to measure
// transaction timing (DISCOVER→OFFER→REQUEST→ACK). Requires root/CAP_NET_RAW.

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/krisarmstrong/seed/internal/logging"
)

// Phase represents a DHCP transaction phase.
type Phase string

// DHCP transaction phase constants.
const (
	PhaseDiscover Phase = "discover"
	PhaseOffer    Phase = "offer"
	PhaseRequest  Phase = "request"
	PhaseAck      Phase = "ack"
)

// DHCP protocol constants.
const (
	// dhcpMagicCookie is the DHCP magic cookie (RFC 2131).
	// This value identifies a BOOTP/DHCP options field.
	dhcpMagicCookie = 0x63825363

	// dhcpMinPacketSize is the minimum DHCP packet size (header + magic cookie).
	dhcpMinPacketSize = 240

	// dhcpOptionEnd marks the end of DHCP options (RFC 2132).
	dhcpOptionEnd = 255

	// dhcpOptionPad is the padding option in DHCP (RFC 2132).
	dhcpOptionPad = 0

	// dhcpOptionMessageType is the DHCP message type option code (RFC 2132).
	dhcpOptionMessageType = 53
)

// DHCP message type constants (RFC 2132 section 9.6).
const (
	dhcpMsgTypeDiscover = 1
	dhcpMsgTypeOffer    = 2
	dhcpMsgTypeRequest  = 3
	dhcpMsgTypeAck      = 5
)

// Packet capture constants.
const (
	// pcapSnapshotLen is the snapshot length for pcap capture.
	// 1600 bytes is sufficient for DHCP packets (typically ~300-600 bytes).
	pcapSnapshotLen = 1600

	// pcapTimeout is the read timeout for pcap packet batching.
	pcapTimeout = 100 * time.Millisecond
)

// DHCP option parsing constants.
const (
	// dhcpOptionHeaderLen is the length of option type + length fields.
	dhcpOptionHeaderLen = 2

	// hexIPv4Len is the expected length of a hex-encoded IPv4 address.
	hexIPv4Len = 4
)

// Timing constants.
const (
	// transactionCleanupInterval is how often to clean up stale transactions.
	transactionCleanupInterval = 30 * time.Second

	// simulatedDiscoverTime is the simulated DHCP discover duration for testing.
	simulatedDiscoverTime = 50 * time.Millisecond

	// simulatedOfferTime is the simulated DHCP offer duration for testing.
	simulatedOfferTime = 10 * time.Millisecond

	// simulatedRequestTime is the simulated DHCP request duration for testing.
	simulatedRequestTime = 45 * time.Millisecond

	// simulatedTotalTime is the simulated total DHCP transaction time for testing.
	simulatedTotalTime = 105 * time.Millisecond
)

// Timing contains timing information for a complete DHCP transaction.
type Timing struct {
	Discover time.Duration `json:"discover"` // Time from Discover to Offer
	Offer    time.Duration `json:"offer"`    // Time from Offer to Request
	Request  time.Duration `json:"request"`  // Time from Request to Ack
	Total    time.Duration `json:"total"`    // Total transaction time
	Complete bool          `json:"complete"` // Whether all phases completed
}

// TimingMs contains timing in milliseconds for JSON serialization.
type TimingMs struct {
	Discover int64 `json:"discover"`
	Offer    int64 `json:"offer"`
	Request  int64 `json:"request"`
	Ack      int64 `json:"ack"`
	Total    int64 `json:"total"`
}

// ToMs converts Timing to milliseconds.
func (t *Timing) ToMs() TimingMs {
	return TimingMs{
		Discover: t.Discover.Milliseconds(),
		Offer:    t.Offer.Milliseconds(),
		Request:  t.Request.Milliseconds(),
		Total:    t.Total.Milliseconds(),
	}
}

// Transaction represents an in-progress DHCP transaction.
type Transaction struct {
	XID          uint32
	Started      time.Time
	DiscoverTime time.Time
	OfferTime    time.Time
	RequestTime  time.Time
	AckTime      time.Time
	Complete     bool
}

// Monitor watches for DHCP transactions and records timing.
type Monitor struct {
	mu            sync.RWMutex
	running       bool
	interfaceName string
	lastTiming    *Timing
	transactions  map[uint32]*Transaction
	stopChan      chan struct{}
	handle        *pcap.Handle
	cleanupDone   chan struct{} // Signals cleanup goroutine exit (fixes #841)
}

// NewMonitor creates a new DHCP monitor.
func NewMonitor(interfaceName string) *Monitor {
	return &Monitor{
		interfaceName: interfaceName,
		transactions:  make(map[uint32]*Transaction),
	}
}

// Start begins monitoring for DHCP packets.
// Note: Requires root/CAP_NET_RAW for packet capture.
func (m *Monitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	// Open pcap handle on the interface
	// Snapshot length of pcapSnapshotLen bytes is enough for DHCP packets
	// Timeout of pcapTimeout for packet batching
	handle, err := pcap.OpenLive(m.interfaceName, pcapSnapshotLen, true, pcapTimeout)
	if err != nil {
		return err
	}

	// Set BPF filter for DHCP traffic (UDP ports 67 and 68)
	if filterErr := handle.SetBPFFilter("udp and (port 67 or port 68)"); filterErr != nil {
		handle.Close()
		return filterErr
	}

	m.handle = handle
	m.stopChan = make(chan struct{})
	m.cleanupDone = make(chan struct{})
	m.running = true

	// Start capture and cleanup goroutines. Both receive stopChan as a
	// parameter (not via m.stopChan) so Stop's m.stopChan = nil doesn't race
	// against the goroutines' reads. cleanupStaleTransactions already used
	// this pattern; capturePackets now matches.
	linkType := handle.LinkType()
	go m.capturePackets(handle, linkType, m.stopChan)
	go m.cleanupStaleTransactions(m.stopChan, m.cleanupDone)

	return nil
}

// capturePackets runs the packet capture loop. stopChan is passed as a
// parameter rather than read from m.stopChan so Stop's nil-assignment doesn't
// race against this goroutine.
func (m *Monitor) capturePackets(handle *pcap.Handle, linkType layers.LinkType, stopChan <-chan struct{}) {
	if handle == nil {
		return
	}

	packetSource := gopacket.NewPacketSource(handle, linkType)
	packets := packetSource.Packets()

	for {
		select {
		case <-stopChan:
			return
		case packet, ok := <-packets:
			if !ok {
				return
			}
			m.processPacket(packet)
		}
	}
}

// isDHCPPort checks if the port is a DHCP port (67 server, 68 client).
func isDHCPPort(port layers.UDPPort) bool {
	return port == 67 || port == 68
}

// extractDHCPPayload extracts DHCP payload from a packet, returns nil if invalid.
func extractDHCPPayload(packet gopacket.Packet) []byte {
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return nil
	}
	udp, ok := udpLayer.(*layers.UDP)
	if !ok || (!isDHCPPort(udp.DstPort) && !isDHCPPort(udp.SrcPort)) {
		return nil
	}

	appLayer := packet.ApplicationLayer()
	if appLayer == nil {
		return nil
	}
	payload := appLayer.Payload()
	// DHCP packets must be at least dhcpMinPacketSize bytes (minimum header + magic cookie)
	if len(payload) < dhcpMinPacketSize {
		return nil
	}

	// Magic cookie check at offset 236-239 (should be dhcpMagicCookie)
	magicCookie := binary.BigEndian.Uint32(payload[236:240])
	if magicCookie != dhcpMagicCookie {
		return nil
	}

	return payload
}

// msgTypeToPhase converts a DHCP message type to our Phase enum.
// Returns false if the message type should be ignored.
func msgTypeToPhase(msgType byte) (Phase, bool) {
	switch msgType {
	case dhcpMsgTypeDiscover:
		return PhaseDiscover, true
	case dhcpMsgTypeOffer:
		return PhaseOffer, true
	case dhcpMsgTypeRequest:
		return PhaseRequest, true
	case dhcpMsgTypeAck:
		return PhaseAck, true
	default:
		// Ignore other message types (DECLINE, NAK, RELEASE, INFORM)
		return "", false
	}
}

// processPacket extracts DHCP information from a captured packet.
func (m *Monitor) processPacket(packet gopacket.Packet) {
	payload := extractDHCPPayload(packet)
	if payload == nil {
		return
	}

	// Transaction ID is at offset 4-7 (4 bytes, big endian)
	xid := binary.BigEndian.Uint32(payload[4:8])

	// Find DHCP message type in options (starting at offset 240)
	msgType := findDHCPMessageType(payload[240:])
	phase, ok := msgTypeToPhase(msgType)
	if !ok {
		return
	}

	timestamp := time.Now()
	// Fixes #924: Store metadata in variable to prevent multiple calls
	if meta := packet.Metadata(); meta != nil && !meta.Timestamp.IsZero() {
		timestamp = meta.Timestamp
	}

	logging.GetLogger().Debug("DHCP captured", "phase", phase, "xid", fmt.Sprintf("0x%08x", xid))
	m.RecordPhase(xid, phase, timestamp)
}

// findDHCPMessageType searches DHCP options for message type (option 53).
func findDHCPMessageType(options []byte) byte {
	for i := 0; i < len(options)-1; {
		optionType := options[i]

		// End option
		if optionType == dhcpOptionEnd {
			break
		}

		// Pad option
		if optionType == dhcpOptionPad {
			i++
			continue
		}

		// Check length
		if i+1 >= len(options) {
			break
		}
		optionLen := int(options[i+1])
		if i+dhcpOptionHeaderLen+optionLen > len(options) {
			break
		}

		// Option 53 is DHCP Message Type
		if optionType == dhcpOptionMessageType && optionLen >= 1 {
			return options[i+dhcpOptionHeaderLen]
		}

		i += dhcpOptionHeaderLen + optionLen
	}
	return 0
}

// Stop stops monitoring and releases resources.
func (m *Monitor) Stop() {
	m.mu.Lock()

	if !m.running {
		m.mu.Unlock()
		return
	}

	m.running = false

	// Fixes #942: Capture cleanupDone reference before releasing lock
	// to prevent deadlock (cleanup goroutine acquires lock in ticker loop)
	cleanupDone := m.cleanupDone
	m.cleanupDone = nil

	// Signal capture goroutine to stop
	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}

	// Close pcap handle
	if m.handle != nil {
		m.handle.Close()
		m.handle = nil
	}

	m.mu.Unlock()

	// Wait for cleanup goroutine to exit OUTSIDE the lock (fixes #841, #942)
	if cleanupDone != nil {
		<-cleanupDone
	}
}

// cleanupStaleTransactions periodically removes incomplete transactions older than 2 minutes.
// This prevents unbounded memory growth from incomplete DHCP transactions (fixes #841).
func (m *Monitor) cleanupStaleTransactions(stopChan <-chan struct{}, cleanupDone chan<- struct{}) {
	ticker := time.NewTicker(transactionCleanupInterval)
	defer ticker.Stop()
	defer close(cleanupDone)

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			m.mu.Lock()
			cutoff := time.Now().Add(-2 * time.Minute)
			for xid, tx := range m.transactions {
				if tx.Started.Before(cutoff) && !tx.Complete {
					delete(m.transactions, xid)
				}
			}
			m.mu.Unlock()
		}
	}
}

// IsRunning returns whether the monitor is active.
func (m *Monitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// SetInterface changes the monitored interface.
// Fixes #935: Restarts monitoring if it was previously running (like vlan/traffic.go).
func (m *Monitor) SetInterface(name string) error {
	m.mu.Lock()
	wasRunning := m.running

	// Stop inline if running (while holding lock to prevent TOCTOU)
	if wasRunning {
		m.running = false
		cleanupDone := m.cleanupDone
		m.cleanupDone = nil

		if m.stopChan != nil {
			close(m.stopChan)
			m.stopChan = nil
		}
		if m.handle != nil {
			m.handle.Close()
			m.handle = nil
		}

		m.mu.Unlock()

		// Wait for cleanup goroutine outside lock
		if cleanupDone != nil {
			<-cleanupDone
		}
	} else {
		m.mu.Unlock()
	}

	// Update interface (no lock needed for this atomic update)
	m.mu.Lock()
	m.interfaceName = name
	m.mu.Unlock()

	// Restart if was previously running
	if wasRunning {
		return m.Start()
	}
	return nil
}

// GetLastTiming returns the most recent complete DHCP transaction timing.
func (m *Monitor) GetLastTiming() *Timing {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastTiming
}

// RecordPhase records a DHCP phase timestamp (used by packet capture).
func (m *Monitor) RecordPhase(xid uint32, phase Phase, timestamp time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tx, exists := m.transactions[xid]
	if !exists {
		tx = &Transaction{
			XID:     xid,
			Started: timestamp,
		}
		m.transactions[xid] = tx
	}

	switch phase {
	case PhaseDiscover:
		tx.DiscoverTime = timestamp
	case PhaseOffer:
		tx.OfferTime = timestamp
	case PhaseRequest:
		tx.RequestTime = timestamp
	case PhaseAck:
		tx.AckTime = timestamp
		tx.Complete = true
		m.calculateTiming(tx)
	}
}

// calculateTiming computes the timing from a completed transaction.
func (m *Monitor) calculateTiming(tx *Transaction) {
	if !tx.Complete {
		return
	}

	timing := &Timing{
		Complete: true,
	}

	// Calculate phase durations
	if !tx.OfferTime.IsZero() && !tx.DiscoverTime.IsZero() {
		timing.Discover = tx.OfferTime.Sub(tx.DiscoverTime)
	}
	if !tx.RequestTime.IsZero() && !tx.OfferTime.IsZero() {
		timing.Offer = tx.RequestTime.Sub(tx.OfferTime)
	}
	if !tx.AckTime.IsZero() && !tx.RequestTime.IsZero() {
		timing.Request = tx.AckTime.Sub(tx.RequestTime)
	}

	// Total time
	if !tx.AckTime.IsZero() && !tx.DiscoverTime.IsZero() {
		timing.Total = tx.AckTime.Sub(tx.DiscoverTime)
	}

	m.lastTiming = timing

	// Cleanup old transaction
	delete(m.transactions, tx.XID)
}

// SimulateTiming creates simulated timing data for testing.
// This is useful when packet capture isn't available.
func SimulateTiming() *Timing {
	return &Timing{
		Discover: simulatedDiscoverTime,
		Offer:    simulatedOfferTime,
		Request:  simulatedRequestTime,
		Total:    simulatedTotalTime,
		Complete: true,
	}
}

// LeaseInfo contains DHCP lease information from the system.
type LeaseInfo struct {
	DHCPServer string
	Gateway    string
	LeaseTime  int // seconds
	DNS        []string
}

// GetLeaseInfo retrieves DHCP lease information for an interface.
// Returns (nil, nil) for unsupported platforms - this is not an error.
func GetLeaseInfo(interfaceName string) (*LeaseInfo, error) {
	switch runtime.GOOS {
	case "darwin":
		return getLeaseInfoDarwin(interfaceName)
	case "linux":
		return getLeaseInfoLinux(interfaceName)
	default:
		//nolint:nilnil // Unsupported platform returns no info, not an error
		return nil, nil
	}
}

// getLeaseInfoDarwin reads DHCP info on macOS from lease files.
// macOS stores DHCP leases in /var/db/dhcpclient/leases/ as plist-like files.
// The filename format is: ifname-1,<hardware_address>.
func getLeaseInfoDarwin(interfaceName string) (*LeaseInfo, error) {
	// Get hardware address for the interface to find the lease file
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	// macOS lease files are named like: en0-1,aa:bb:cc:dd:ee:ff
	hwAddr := iface.HardwareAddr.String()

	// Try both old and new lease file locations
	// Note: Using filepath.Join with absolute path is intentional to construct full paths
	leasePaths := []string{
		"/var/db/dhcpclient/leases/" + interfaceName + "-1," + hwAddr,
		"/private/var/db/dhcpclient/leases/" + interfaceName + "-1," + hwAddr,
	}

	for _, path := range leasePaths {
		if info := parseDarwinLeaseFile(path); info != nil {
			return info, nil
		}
	}

	// Fallback: scan the leases directory for any file matching the interface
	leaseDir := "/var/db/dhcpclient/leases"
	entries, err := os.ReadDir(leaseDir)
	if err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), interfaceName+"-") {
				if info := parseDarwinLeaseFile(filepath.Join(leaseDir, entry.Name())); info != nil {
					return info, nil
				}
			}
		}
	}

	return &LeaseInfo{}, nil
}

// parseDarwinLeaseFile parses a macOS DHCP lease file (plist-like format).
func parseDarwinLeaseFile(path string) *LeaseInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	info := &LeaseInfo{}
	content := string(data)

	// The lease file is XML plist format
	// Extract values using simple string parsing (avoiding plist dependency)

	// <key>ServerIdentifier</key> followed by <data>hex_encoded_ip</data>
	if server := extractPlistIP(content, "ServerIdentifier"); server != "" {
		info.DHCPServer = server
	}

	// <key>RouterIPAddress</key> followed by <data>hex_encoded_ip</data>
	if router := extractPlistIP(content, "RouterIPAddress"); router != "" {
		info.Gateway = router
	}
	// Also try Router key
	if info.Gateway == "" {
		if router := extractPlistIP(content, "Router"); router != "" {
			info.Gateway = router
		}
	}

	// <key>LeaseLength</key> followed by <integer>value</integer>
	if lease := extractPlistInteger(content, "LeaseLength"); lease > 0 {
		info.LeaseTime = lease
	}

	// DNS servers - may be in array format
	if dns := extractPlistIPArray(content, "DomainNameServer"); len(dns) > 0 {
		info.DNS = dns
	}

	if info.DHCPServer != "" || info.Gateway != "" {
		return info
	}
	return nil
}

// extractPlistIP extracts an IP address from a plist data field.
func extractPlistIP(content, key string) string {
	keyTag := "<key>" + key + "</key>"
	_, remaining, found := strings.Cut(content, keyTag)
	if !found {
		return ""
	}

	// Look for <data> tag after the key
	dataStart := strings.Index(remaining, "<data>")
	if dataStart == -1 {
		return ""
	}
	dataEnd := strings.Index(remaining[dataStart:], "</data>")
	if dataEnd == -1 {
		return ""
	}

	hexData := strings.TrimSpace(remaining[dataStart+6 : dataStart+dataEnd])
	return hexToIP(hexData)
}

// extractPlistInteger extracts an integer value from a plist.
func extractPlistInteger(content, key string) int {
	keyTag := "<key>" + key + "</key>"
	_, remaining, found := strings.Cut(content, keyTag)
	if !found {
		return 0
	}

	intStart := strings.Index(remaining, "<integer>")
	if intStart == -1 {
		return 0
	}
	intEnd := strings.Index(remaining[intStart:], "</integer>")
	if intEnd == -1 {
		return 0
	}

	valStr := strings.TrimSpace(remaining[intStart+9 : intStart+intEnd])
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0
	}
	return val
}

// parseDataTagsFromArray extracts IP addresses from <data> tags within array content.
func parseDataTagsFromArray(arrayContent string) []string {
	var ips []string
	content := arrayContent

	for {
		dataStart := strings.Index(content, "<data>")
		if dataStart == -1 {
			break
		}
		dataEnd := strings.Index(content[dataStart:], "</data>")
		if dataEnd == -1 {
			break
		}
		hexData := strings.TrimSpace(content[dataStart+6 : dataStart+dataEnd])
		if ip := hexToIP(hexData); ip != "" {
			ips = append(ips, ip)
		}
		content = content[dataStart+dataEnd+7:]
	}

	return ips
}

// extractArrayContent finds and extracts the content between <array> and </array> tags.
func extractArrayContent(remaining string) (string, bool) {
	arrayStart := strings.Index(remaining, "<array>")
	if arrayStart == -1 {
		return "", false
	}

	arrayEnd := strings.Index(remaining[arrayStart:], "</array>")
	if arrayEnd == -1 {
		return "", false
	}

	return remaining[arrayStart+7 : arrayStart+arrayEnd], true
}

// extractPlistIPArray extracts array of IPs from plist.
func extractPlistIPArray(content, key string) []string {
	keyTag := "<key>" + key + "</key>"
	_, remaining, found := strings.Cut(content, keyTag)
	if !found {
		return nil
	}

	// Try array format first
	if arrayContent, arrayFound := extractArrayContent(remaining); arrayFound {
		if ips := parseDataTagsFromArray(arrayContent); len(ips) > 0 {
			return ips
		}
	}

	// Try single data format
	if ip := extractPlistIP(content, key); ip != "" {
		return []string{ip}
	}

	return nil
}

// hexToIP converts hex-encoded IP address to string.
func hexToIP(hexStr string) string {
	// Remove any whitespace/newlines from base64-ish encoding
	hexStr = strings.ReplaceAll(hexStr, "\n", "")
	hexStr = strings.ReplaceAll(hexStr, "\t", "")
	hexStr = strings.ReplaceAll(hexStr, " ", "")

	// Try decoding as hex
	if bytes, err := hex.DecodeString(hexStr); err == nil && len(bytes) == 4 {
		return net.IP(bytes).String()
	}

	// macOS sometimes uses raw bytes in base64 - try that too
	// The data might actually be raw bytes interpreted as string
	if len(hexStr) == hexIPv4Len {
		return net.IP([]byte(hexStr)).String()
	}

	return ""
}

// getLeaseInfoLinux reads DHCP info on Linux from lease files.
func getLeaseInfoLinux(interfaceName string) (*LeaseInfo, error) {
	info := &LeaseInfo{}

	// Try NetworkManager lease file first
	nmLeasePath := "/var/lib/NetworkManager/internal-" + interfaceName + ".lease"
	if _, err := os.Stat(nmLeasePath); err == nil {
		if lease := parseNMLeaseFile(nmLeasePath); lease != nil {
			return lease, nil
		}
	}

	// Try dhclient lease file
	dhclientPaths := []string{
		"/var/lib/dhcp/dhclient." + interfaceName + ".leases",
		"/var/lib/dhclient/dhclient." + interfaceName + ".leases",
		"/var/lib/dhcp/dhclient.leases",
	}

	for _, path := range dhclientPaths {
		if _, err := os.Stat(path); err == nil {
			if lease := parseDHClientLeaseFile(path, interfaceName); lease != nil {
				return lease, nil
			}
		}
	}

	// Try systemd-networkd lease file
	networkdPath := "/run/systemd/netif/leases/"
	if entries, err := os.ReadDir(networkdPath); err == nil {
		for _, entry := range entries {
			if lease := parseNetworkdLeaseFile(networkdPath + entry.Name()); lease != nil {
				return lease, nil
			}
		}
	}

	return info, nil
}

// parseDHClientLeaseLine parses a single line from a dhclient lease file.
func parseDHClientLeaseLine(line string, info *LeaseInfo) {
	switch {
	case strings.HasPrefix(line, "option dhcp-server-identifier"):
		info.DHCPServer = extractValue(line)
	case strings.HasPrefix(line, "option routers"):
		val := extractValue(line)
		if parts := strings.Split(val, ","); len(parts) > 0 {
			info.Gateway = strings.TrimSpace(parts[0])
		}
	case strings.HasPrefix(line, "option dhcp-lease-time"):
		if lease, err := strconv.Atoi(extractValue(line)); err == nil {
			info.LeaseTime = lease
		}
	case strings.HasPrefix(line, "option domain-name-servers"):
		val := extractValue(line)
		for dns := range strings.SplitSeq(val, ",") {
			dns = strings.TrimSpace(dns)
			if dns != "" {
				info.DNS = append(info.DNS, dns)
			}
		}
	}
}

// parseDHClientLeaseFile parses a dhclient lease file.
func parseDHClientLeaseFile(path, _ string) *LeaseInfo {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	info := &LeaseInfo{}
	scanner := bufio.NewScanner(file)
	inLease := false

	// Parse the last lease block for the interface
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "lease {") {
			inLease = true
			info = &LeaseInfo{} // Reset for each lease block
			continue
		}
		if !inLease {
			continue
		}
		if line == "}" {
			inLease = false
			continue
		}
		parseDHClientLeaseLine(line, info)
	}

	if info.DHCPServer != "" || info.Gateway != "" {
		return info
	}
	return nil
}

// leaseFieldMapping defines how to extract DHCP lease fields from a file format.
type leaseFieldMapping struct {
	serverKey    string
	routerKey    string
	leaseTimeKey string
	dnsKey       string
}

// parseLeaseLineServer extracts the DHCP server from a line if it matches the server key.
func parseLeaseLineServer(line, serverKey string) (string, bool) {
	after, ok := strings.CutPrefix(line, serverKey)
	if !ok {
		return "", false
	}
	return after, true
}

// parseLeaseLineRouter extracts the gateway from a line if it matches the router key.
func parseLeaseLineRouter(line, routerKey string) (string, bool) {
	after, ok := strings.CutPrefix(line, routerKey)
	if !ok {
		return "", false
	}
	parts := strings.Split(after, " ")
	if len(parts) == 0 {
		return "", false
	}
	return parts[0], true
}

// parseLeaseLineTime extracts the lease time from a line if it matches the lease time key.
func parseLeaseLineTime(line, leaseTimeKey string) (int, bool) {
	after, ok := strings.CutPrefix(line, leaseTimeKey)
	if !ok {
		return 0, false
	}
	lease, err := strconv.Atoi(after)
	if err != nil {
		return 0, false
	}
	return lease, true
}

// parseLeaseLineDNS extracts DNS servers from a line if it matches the DNS key.
func parseLeaseLineDNS(line, dnsKey string) []string {
	after, ok := strings.CutPrefix(line, dnsKey)
	if !ok {
		return nil
	}
	var servers []string
	for dns := range strings.SplitSeq(after, " ") {
		dns = strings.TrimSpace(dns)
		if dns != "" {
			servers = append(servers, dns)
		}
	}
	return servers
}

// processLeaseLine processes a single lease file line and updates the LeaseInfo.
func processLeaseLine(line string, mapping leaseFieldMapping, info *LeaseInfo) {
	if server, ok := parseLeaseLineServer(line, mapping.serverKey); ok {
		info.DHCPServer = server
	}
	if gateway, ok := parseLeaseLineRouter(line, mapping.routerKey); ok {
		info.Gateway = gateway
	}
	if leaseTime, ok := parseLeaseLineTime(line, mapping.leaseTimeKey); ok {
		info.LeaseTime = leaseTime
	}
	if dnsServers := parseLeaseLineDNS(line, mapping.dnsKey); dnsServers != nil {
		info.DNS = append(info.DNS, dnsServers...)
	}
}

// parseLeaseFileWithMapping parses a lease file using the given field mappings.
func parseLeaseFileWithMapping(path string, mapping leaseFieldMapping) *LeaseInfo {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	info := &LeaseInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		processLeaseLine(line, mapping, info)
	}

	if info.DHCPServer != "" || info.Gateway != "" {
		return info
	}
	return nil
}

// parseNMLeaseFile parses NetworkManager internal lease file.
func parseNMLeaseFile(path string) *LeaseInfo {
	return parseLeaseFileWithMapping(path, leaseFieldMapping{
		serverKey:    "DHCP4_SERVER_ID=",
		routerKey:    "DHCP4_ROUTERS=",
		leaseTimeKey: "DHCP4_LEASE_TIME=",
		dnsKey:       "DHCP4_DOMAIN_NAME_SERVERS=",
	})
}

// parseNetworkdLeaseFile parses systemd-networkd lease file.
func parseNetworkdLeaseFile(path string) *LeaseInfo {
	return parseLeaseFileWithMapping(path, leaseFieldMapping{
		serverKey:    "SERVER_ADDRESS=",
		routerKey:    "ROUTER=",
		leaseTimeKey: "LIFETIME=",
		dnsKey:       "DNS=",
	})
}

// extractValue extracts value from dhclient option line.
func extractValue(line string) string {
	// Remove trailing semicolon
	line = strings.TrimSuffix(line, ";")
	// Get last space-separated value
	parts := strings.Fields(line)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
