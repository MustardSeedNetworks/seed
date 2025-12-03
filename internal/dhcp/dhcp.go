// Package dhcp provides DHCP transaction timing and monitoring.
package dhcp

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Phase represents a DHCP transaction phase.
type Phase string

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
	mu           sync.RWMutex
	running      bool
	interfaceName string
	lastTiming   *Timing
	transactions map[uint32]*Transaction
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

	// TODO: Implement actual packet capture using gopacket/pcap
	// For now, we just mark as running but don't capture
	// This requires root access and the gopacket library
	m.running = true

	return nil
}

// Stop stops monitoring.
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
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
func GetLeaseInfo(interfaceName string) (*LeaseInfo, error) {
	switch runtime.GOOS {
	case "darwin":
		return getLeaseInfoDarwin(interfaceName)
	case "linux":
		return getLeaseInfoLinux(interfaceName)
	default:
		return nil, nil
	}
}

// getLeaseInfoDarwin reads DHCP info on macOS using ipconfig.
func getLeaseInfoDarwin(interfaceName string) (*LeaseInfo, error) {
	cmd := exec.Command("ipconfig", "getpacket", interfaceName)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	info := &LeaseInfo{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// server_identifier (ip): 192.168.1.1
		if strings.HasPrefix(line, "server_identifier") {
			if idx := strings.LastIndex(line, ": "); idx != -1 {
				info.DHCPServer = strings.TrimSpace(line[idx+2:])
			}
		}

		// router (ip_mult): {192.168.1.1}
		if strings.HasPrefix(line, "router") {
			if idx := strings.LastIndex(line, ": "); idx != -1 {
				val := strings.TrimSpace(line[idx+2:])
				val = strings.Trim(val, "{}")
				if parts := strings.Split(val, ","); len(parts) > 0 {
					info.Gateway = strings.TrimSpace(parts[0])
				}
			}
		}

		// lease_time (uint32): 86400
		if strings.HasPrefix(line, "lease_time") {
			if idx := strings.LastIndex(line, ": "); idx != -1 {
				if lease, err := strconv.Atoi(strings.TrimSpace(line[idx+2:])); err == nil {
					info.LeaseTime = lease
				}
			}
		}

		// domain_name_server (ip_mult): {8.8.8.8, 8.8.4.4}
		if strings.HasPrefix(line, "domain_name_server") {
			if idx := strings.LastIndex(line, ": "); idx != -1 {
				val := strings.TrimSpace(line[idx+2:])
				val = strings.Trim(val, "{}")
				for _, dns := range strings.Split(val, ",") {
					dns = strings.TrimSpace(dns)
					if dns != "" {
						info.DNS = append(info.DNS, dns)
					}
				}
			}
		}
	}

	return info, nil
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

// parseDHClientLeaseFile parses a dhclient lease file.
func parseDHClientLeaseFile(path, interfaceName string) *LeaseInfo {
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
		}

		if !inLease {
			continue
		}

		if line == "}" {
			inLease = false
			continue
		}

		// option dhcp-server-identifier 192.168.1.1;
		if strings.HasPrefix(line, "option dhcp-server-identifier") {
			info.DHCPServer = extractValue(line)
		}

		// option routers 192.168.1.1;
		if strings.HasPrefix(line, "option routers") {
			val := extractValue(line)
			if parts := strings.Split(val, ","); len(parts) > 0 {
				info.Gateway = strings.TrimSpace(parts[0])
			}
		}

		// option dhcp-lease-time 86400;
		if strings.HasPrefix(line, "option dhcp-lease-time") {
			if lease, err := strconv.Atoi(extractValue(line)); err == nil {
				info.LeaseTime = lease
			}
		}

		// option domain-name-servers 8.8.8.8, 8.8.4.4;
		if strings.HasPrefix(line, "option domain-name-servers") {
			val := extractValue(line)
			for _, dns := range strings.Split(val, ",") {
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
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	info := &LeaseInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "DHCP4_SERVER_ID=") {
			info.DHCPServer = strings.TrimPrefix(line, "DHCP4_SERVER_ID=")
		}
		if strings.HasPrefix(line, "DHCP4_ROUTERS=") {
			val := strings.TrimPrefix(line, "DHCP4_ROUTERS=")
			if parts := strings.Split(val, " "); len(parts) > 0 {
				info.Gateway = parts[0]
			}
		}
		if strings.HasPrefix(line, "DHCP4_LEASE_TIME=") {
			if lease, err := strconv.Atoi(strings.TrimPrefix(line, "DHCP4_LEASE_TIME=")); err == nil {
				info.LeaseTime = lease
			}
		}
		if strings.HasPrefix(line, "DHCP4_DOMAIN_NAME_SERVERS=") {
			val := strings.TrimPrefix(line, "DHCP4_DOMAIN_NAME_SERVERS=")
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

// parseNetworkdLeaseFile parses systemd-networkd lease file.
func parseNetworkdLeaseFile(path string) *LeaseInfo {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	info := &LeaseInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "SERVER_ADDRESS=") {
			info.DHCPServer = strings.TrimPrefix(line, "SERVER_ADDRESS=")
		}
		if strings.HasPrefix(line, "ROUTER=") {
			val := strings.TrimPrefix(line, "ROUTER=")
			if parts := strings.Split(val, " "); len(parts) > 0 {
				info.Gateway = parts[0]
			}
		}
		if strings.HasPrefix(line, "LIFETIME=") {
			if lease, err := strconv.Atoi(strings.TrimPrefix(line, "LIFETIME=")); err == nil {
				info.LeaseTime = lease
			}
		}
		if strings.HasPrefix(line, "DNS=") {
			val := strings.TrimPrefix(line, "DNS=")
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
