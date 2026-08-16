// Package dhcp monitors DHCP transaction timing on the local network.
//
// The Monitor uses the capture port (internal/capture) for real-time DHCP packet
// capture to measure transaction timing (DISCOVER→OFFER→REQUEST→ACK). The
// libpcap-backed adapter requires root/CAP_NET_RAW; wire it via WithCapture.
package dhcp

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/logging"
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
	// dhcpMagicCookie is the DHCP options marker (RFC 2131).
	// This value identifies a BOOTP/DHCP options field.
	dhcpMagicCookie = 0x63825363

	// dhcpMinPacketSize is the minimum DHCP packet size (header + options marker).
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
	opener        capture.Opener
	handle        capture.Handle
	cleanupDone   chan struct{} // Signals cleanup goroutine exit (fixes #841)
}

// NewMonitor creates a new DHCP monitor. opts inject optional dependencies such
// as the live-capture Opener (WithCapture); the default is the CGO-free no-op.
func NewMonitor(interfaceName string, opts ...Option) *Monitor {
	return &Monitor{
		interfaceName: interfaceName,
		transactions:  make(map[uint32]*Transaction),
		opener:        resolveCapture(opts...),
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
	handle, err := m.opener.OpenLive(m.interfaceName, pcapSnapshotLen, true, pcapTimeout)
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
func (m *Monitor) capturePackets(handle capture.Handle, linkType layers.LinkType, stopChan <-chan struct{}) {
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
	// DHCP packets must be at least dhcpMinPacketSize bytes (minimum header + options marker)
	if len(payload) < dhcpMinPacketSize {
		return nil
	}

	// Validate the DHCP options marker at offset 236-239.
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
