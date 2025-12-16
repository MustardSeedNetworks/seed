// Package dhcp provides DHCP transaction timing and monitoring.
//
// The Monitor uses gopacket/pcap for real-time DHCP packet capture to measure
// transaction timing (DISCOVER→OFFER→REQUEST→ACK). Requires root/CAP_NET_RAW.
package dhcp

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
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
	// Snapshot length of 1600 bytes is enough for DHCP packets
	// Timeout of 100ms for packet batching
	handle, err := pcap.OpenLive(m.interfaceName, 1600, true, 100*time.Millisecond)
	if err != nil {
		return err
	}

	// Set BPF filter for DHCP traffic (UDP ports 67 and 68)
	if err := handle.SetBPFFilter("udp and (port 67 or port 68)"); err != nil {
		handle.Close()
		return err
	}

	m.handle = handle
	m.stopChan = make(chan struct{})
	m.running = true

	// Start capture goroutine
	go m.capturePackets()

	return nil
}

// capturePackets runs the packet capture loop.
func (m *Monitor) capturePackets() {
	packetSource := gopacket.NewPacketSource(m.handle, m.handle.LinkType())
	packets := packetSource.Packets()

	for {
		select {
		case <-m.stopChan:
			return
		case packet, ok := <-packets:
			if !ok {
				return
			}
			m.processPacket(packet)
		}
	}
}

// processPacket extracts DHCP information from a captured packet.
func (m *Monitor) processPacket(packet gopacket.Packet) {
	// Get UDP layer
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return
	}
	udp, _ := udpLayer.(*layers.UDP)

	// DHCP uses ports 67 (server) and 68 (client)
	if udp.DstPort != 67 && udp.DstPort != 68 && udp.SrcPort != 67 && udp.SrcPort != 68 {
		return
	}

	// Get application payload (DHCP data)
	appLayer := packet.ApplicationLayer()
	if appLayer == nil {
		return
	}
	payload := appLayer.Payload()
	if len(payload) < 240 {
		// DHCP packets must be at least 240 bytes (minimum header + magic cookie)
		return
	}

	// Parse DHCP packet
	// Transaction ID is at offset 4-7 (4 bytes, big endian)
	xid := binary.BigEndian.Uint32(payload[4:8])

	// Magic cookie check at offset 236-239 (should be 0x63825363)
	if len(payload) >= 240 {
		magicCookie := binary.BigEndian.Uint32(payload[236:240])
		if magicCookie != 0x63825363 {
			return
		}
	}

	// Find DHCP message type in options (starting at offset 240)
	msgType := findDHCPMessageType(payload[240:])
	if msgType == 0 {
		return
	}

	timestamp := time.Now()
	if packet.Metadata() != nil && !packet.Metadata().Timestamp.IsZero() {
		timestamp = packet.Metadata().Timestamp
	}

	// Convert DHCP message type to our phase
	var phase Phase
	switch msgType {
	case 1: // DHCP Discover
		phase = PhaseDiscover
	case 2: // DHCP Offer
		phase = PhaseOffer
	case 3: // DHCP Request
		phase = PhaseRequest
	case 5: // DHCP ACK
		phase = PhaseAck
	default:
		// Ignore other message types (DECLINE, NAK, RELEASE, INFORM)
		return
	}

	slog.Debug("DHCP captured", "phase", phase, "xid", fmt.Sprintf("0x%08x", xid))
	m.RecordPhase(xid, phase, timestamp)
}

// findDHCPMessageType searches DHCP options for message type (option 53).
func findDHCPMessageType(options []byte) byte {
	for i := 0; i < len(options)-1; {
		optionType := options[i]

		// End option
		if optionType == 255 {
			break
		}

		// Pad option
		if optionType == 0 {
			i++
			continue
		}

		// Check length
		if i+1 >= len(options) {
			break
		}
		optionLen := int(options[i+1])
		if i+2+optionLen > len(options) {
			break
		}

		// Option 53 is DHCP Message Type
		if optionType == 53 && optionLen >= 1 {
			return options[i+2]
		}

		i += 2 + optionLen
	}
	return 0
}

// Stop stops monitoring and releases resources.
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false

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
}

// IsRunning returns whether the monitor is active.
func (m *Monitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// SetInterface changes the monitored interface.
func (m *Monitor) SetInterface(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interfaceName = name
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
		Discover: 50 * time.Millisecond,
		Offer:    10 * time.Millisecond,
		Request:  45 * time.Millisecond,
		Total:    105 * time.Millisecond,
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
	//nolint:gosec // G304: path is from known DHCP lease file locations in system directories
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
	idx := strings.Index(content, keyTag)
	if idx == -1 {
		return ""
	}

	// Look for <data> tag after the key
	remaining := content[idx+len(keyTag):]
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
	idx := strings.Index(content, keyTag)
	if idx == -1 {
		return 0
	}

	remaining := content[idx+len(keyTag):]
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

// extractPlistIPArray extracts array of IPs from plist.
func extractPlistIPArray(content, key string) []string {
	var ips []string
	keyTag := "<key>" + key + "</key>"
	idx := strings.Index(content, keyTag)
	if idx == -1 {
		return ips
	}

	remaining := content[idx+len(keyTag):]

	// Try array format first
	arrayStart := strings.Index(remaining, "<array>")
	if arrayStart != -1 {
		arrayEnd := strings.Index(remaining[arrayStart:], "</array>")
		if arrayEnd != -1 {
			arrayContent := remaining[arrayStart+7 : arrayStart+arrayEnd]
			// Find all <data> tags within array
			for {
				dataStart := strings.Index(arrayContent, "<data>")
				if dataStart == -1 {
					break
				}
				dataEnd := strings.Index(arrayContent[dataStart:], "</data>")
				if dataEnd == -1 {
					break
				}
				hexData := strings.TrimSpace(arrayContent[dataStart+6 : dataStart+dataEnd])
				if ip := hexToIP(hexData); ip != "" {
					ips = append(ips, ip)
				}
				arrayContent = arrayContent[dataStart+dataEnd+7:]
			}
		}
	}

	// Try single data format
	if len(ips) == 0 {
		if ip := extractPlistIP(content, key); ip != "" {
			ips = append(ips, ip)
		}
	}

	return ips
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
	if len(hexStr) == 4 {
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
		for _, dns := range strings.Split(val, ",") {
			dns = strings.TrimSpace(dns)
			if dns != "" {
				info.DNS = append(info.DNS, dns)
			}
		}
	}
}

// parseDHClientLeaseFile parses a dhclient lease file.
func parseDHClientLeaseFile(path, _ string) *LeaseInfo {
	//nolint:gosec // G304: path is from known dhclient lease file locations in system directories
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

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

// parseLeaseFileWithMapping parses a lease file using the given field mappings.
func parseLeaseFileWithMapping(path string, mapping leaseFieldMapping) *LeaseInfo {
	//nolint:gosec // G304: path is from known DHCP lease file locations in system directories
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	info := &LeaseInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, mapping.serverKey) {
			info.DHCPServer = strings.TrimPrefix(line, mapping.serverKey)
		}
		if strings.HasPrefix(line, mapping.routerKey) {
			val := strings.TrimPrefix(line, mapping.routerKey)
			if parts := strings.Split(val, " "); len(parts) > 0 {
				info.Gateway = parts[0]
			}
		}
		if strings.HasPrefix(line, mapping.leaseTimeKey) {
			if lease, err := strconv.Atoi(strings.TrimPrefix(line, mapping.leaseTimeKey)); err == nil {
				info.LeaseTime = lease
			}
		}
		if strings.HasPrefix(line, mapping.dnsKey) {
			val := strings.TrimPrefix(line, mapping.dnsKey)
			for _, dns := range strings.Split(val, " ") {
				dns = strings.TrimSpace(dns)
				if dns != "" {
					info.DNS = append(info.DNS, dns)
				}
			}
		}
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
