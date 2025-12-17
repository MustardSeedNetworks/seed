// Package discovery implements active network scanning via ARP and ICMP.
//
// This file provides active Layer 2 and Layer 3 network scanning to discover
// devices on local and remote subnets. Unlike passive discovery protocols (LLDP/CDP),
// ARP scanning actively probes the network to find all responsive devices.
//
// Scanning modes:
//   - ARP scanning: Local subnet discovery using Address Resolution Protocol (Layer 2)
//   - ICMP scanning: Remote subnet discovery using ICMP Echo Request/Reply (Layer 3)
//
// Local subnet scanning (ARP):
//   - Sends ARP "who-has" requests for each IP in the subnet
//   - Parses ARP replies to build device list with MAC addresses
//   - Requires Layer 2 connectivity (same broadcast domain)
//   - Fast and reliable for local networks
//   - Identifies vendor from MAC OUI (Organizationally Unique Identifier)
//
// Remote subnet scanning (ICMP):
//   - Sends ICMP Echo Request (ping) for each IP
//   - Uses raw sockets for concurrent pinging (requires elevated privileges)
//   - Examines TTL values to estimate operating system
//   - Works across routers (Layer 3)
//   - Slower than ARP but supports remote networks
//
// Additional subnets:
//   - Configure via Discovery.AdditionalSubnets in config
//   - Automatically selects ICMP for remote subnets (beyond local broadcast domain)
//   - Results are merged with local ARP results
//   - Marked with IsLocal flag to distinguish origin
//
// OS detection heuristics (based on initial TTL):
//   - TTL 64: Linux/Unix (decremented from 64)
//   - TTL 128: Windows (decremented from 128)
//   - TTL 255: Network device (decremented from 255)
//
// Performance:
//   - Concurrent scanning with rate limiting to avoid network flooding
//   - Results cached with LastSeen timestamps
//   - Hostname resolution performed asynchronously to avoid blocking
//   - Vendor lookup via OUI database (IEEE MA-L assignments)
//
// Security considerations:
//   - Active scanning generates network traffic (visible to IDS/IPS)
//   - May trigger security alerts in monitored environments
//   - Requires CAP_NET_RAW on Linux for ICMP scanning
//   - Should be rate-limited in production environments
package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

// ARPEntry represents a discovered device from ARP or ICMP scanning.
//
// Entries are created from:
//   - ARP replies during local subnet scanning (has MAC address)
//   - ICMP Echo Replies during remote subnet scanning (may lack MAC)
//   - System ARP cache queries (platform-specific implementation)
//
// The entry contains both Layer 2 (MAC, vendor) and Layer 3 (IP, hostname)
// information when available. Remote devices discovered via ICMP may not
// have MAC addresses.
type ARPEntry struct {
	IP           string    `json:"ip"`
	MAC          string    `json:"mac"`
	Vendor       string    `json:"vendor,omitempty"`
	Hostname     string    `json:"hostname,omitempty"`
	Interface    string    `json:"interface,omitempty"`
	State        string    `json:"state,omitempty"` // REACHABLE, STALE, etc.
	TTL          int       `json:"ttl,omitempty"`
	OSGuess      string    `json:"osGuess,omitempty"`
	LastSeen     time.Time `json:"lastSeen"`
	ResponseTime int64     `json:"responseTime,omitempty"` // in milliseconds
	IsLocal      bool      `json:"isLocal"`                // true if on local subnet, false for additional subnets
}

// ARPScanner performs active network discovery via ARP.
type ARPScanner struct {
	interfaceName     string
	oui               *OUIDatabase
	mu                sync.RWMutex
	entries           map[string]*ARPEntry // Key by IP
	subnet            *net.IPNet
	localIP           net.IP
	additionalSubnets []*net.IPNet          // Additional subnets to scan
	pingResponders    []string              // IPs that responded to ping (for remote subnets)
	pingResults       map[string]PingResult // Cached ping results with TTL info
	pinger            *ICMPPinger           // Raw socket ICMP pinger
	scanning          bool
	lastScan          time.Time
}

// NewARPScanner creates a new ARP scanner for the given interface.
func NewARPScanner(interfaceName string, oui *OUIDatabase) *ARPScanner {
	return &ARPScanner{
		interfaceName: interfaceName,
		oui:           oui,
		entries:       make(map[string]*ARPEntry),
	}
}

// SetInterface updates the interface to scan.
func (s *ARPScanner) SetInterface(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interfaceName = name
	s.subnet = nil
	s.localIP = nil
}

// SetAdditionalSubnets configures extra subnets to scan.
func (s *ARPScanner) SetAdditionalSubnets(cidrs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.additionalSubnets = nil
	for _, cidr := range cidrs {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid CIDR %s: %w", cidr, err)
		}
		s.additionalSubnets = append(s.additionalSubnets, subnet)
	}
	return nil
}

// GetAdditionalSubnets returns the configured additional subnets.
func (s *ARPScanner) GetAdditionalSubnets() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, len(s.additionalSubnets))
	for i, subnet := range s.additionalSubnets {
		result[i] = subnet.String()
	}
	return result
}

// getSubnet determines the subnet for the interface.
func (s *ARPScanner) getSubnet() (*net.IPNet, net.IP, error) {
	iface, err := net.InterfaceByName(s.interfaceName)
	if err != nil {
		return nil, nil, fmt.Errorf("interface %s not found: %w", s.interfaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get addresses: %w", err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		// Only use IPv4
		if ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() {
			return ipNet, ipNet.IP, nil
		}
	}

	return nil, nil, fmt.Errorf("no IPv4 address found on interface %s", s.interfaceName)
}

// incrementIP adds n to an IP address.
func incrementIP(ip net.IP, n int) net.IP {
	ip = ip.To4()
	if ip == nil {
		return nil
	}
	result := make(net.IP, 4)
	copy(result, ip)

	carry := n
	for i := 3; i >= 0 && carry > 0; i-- {
		sum := int(result[i]) + (carry & 0xff)
		result[i] = byte(sum & 0xff)
		carry = (carry >> 8) + (sum >> 8)
	}

	return result
}

// Scan performs an active ARP scan of the network.
func (s *ARPScanner) Scan(ctx context.Context) error {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		return fmt.Errorf("scan already in progress")
	}
	s.scanning = true
	s.pingResponders = nil                   // Clear previous ping responders
	additionalSubnets := s.additionalSubnets // Copy while holding lock
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.lastScan = time.Now()
		s.mu.Unlock()
	}()

	// Get subnet info
	subnet, localIP, err := s.getSubnet()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.subnet = subnet
	s.localIP = localIP
	s.mu.Unlock()

	// Perform ping sweep on primary subnet
	if err := s.pingSweep(ctx, subnet); err != nil {
		return fmt.Errorf("ping sweep failed: %w", err)
	}

	// Perform ping sweep on additional subnets
	for _, additionalSubnet := range additionalSubnets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue scanning additional subnets even if some fail
			if err := s.pingSweep(ctx, additionalSubnet); err != nil {
				slog.Error("ping sweep failed for subnet", "subnet", additionalSubnet, "error", err)
			}
		}
	}

	// Read ARP table (will include entries from all scanned subnets)
	entries, err := s.readARPTable()
	if err != nil {
		return fmt.Errorf("failed to read ARP table: %w", err)
	}

	// Mark ARP entries as local (ARP only works on local subnet)
	for _, entry := range entries {
		entry.IsLocal = true
	}

	// Merge ping responders that aren't in ARP table (remote subnets)
	s.mu.RLock()
	responders := make([]string, len(s.pingResponders))
	copy(responders, s.pingResponders)
	s.mu.RUnlock()

	// Create a map of existing IPs from ARP
	existingIPs := make(map[string]bool)
	for _, entry := range entries {
		existingIPs[entry.IP] = true
	}

	// Add ping responders not in ARP table (these are from remote/additional subnets)
	for _, ip := range responders {
		if !existingIPs[ip] {
			entries = append(entries, &ARPEntry{
				IP:       ip,
				MAC:      "", // No MAC for remote hosts (ARP doesn't work across routers)
				State:    "PING_ONLY",
				LastSeen: time.Now(),
				IsLocal:  false, // Remote subnet - not local
			})
		}
	}

	// Enrich entries with OUI lookup and hostname resolution
	s.enrichEntries(ctx, entries)

	return nil
}

// pingSweep sends ICMP echo requests to all hosts in the subnet using raw sockets.
func (s *ARPScanner) pingSweep(ctx context.Context, subnet *net.IPNet) error {
	ones, bits := subnet.Mask.Size()
	numHosts := 1<<(bits-ones) - 2 // Exclude network and broadcast

	// Limit to reasonable size (/24 = 254 hosts)
	if numHosts > 254 {
		numHosts = 254
	}

	// Generate host IPs
	baseIP := subnet.IP.Mask(subnet.Mask).To4()
	if baseIP == nil {
		return fmt.Errorf("invalid subnet")
	}

	// Build list of IPs to ping
	var ips []net.IP
	for i := 1; i <= numHosts; i++ {
		ip := incrementIP(baseIP, i)
		if ip != nil && !ip.Equal(s.localIP) {
			ips = append(ips, ip)
		}
	}

	// Initialize pinger if needed
	if s.pinger == nil {
		pinger, err := NewICMPPinger(time.Second)
		if err != nil {
			slog.Warn("Failed to create ICMP pinger", "error", err)
			return err
		}
		s.pinger = pinger
	}

	// Perform ping sweep using raw ICMP sockets (50 workers)
	results := s.pinger.PingSweep(ctx, ips, 50)

	// Store results and track responders
	s.mu.Lock()
	if s.pingResults == nil {
		s.pingResults = make(map[string]PingResult)
	}
	for _, result := range results {
		if result.Reachable {
			s.pingResponders = append(s.pingResponders, result.IP)
		}
		s.pingResults[result.IP] = result
	}
	s.mu.Unlock()

	return nil
}

// Close releases resources held by the ARPScanner.
func (s *ARPScanner) Close() error {
	if s.pinger != nil {
		return s.pinger.Close()
	}
	return nil
}

// readARPTable reads the system ARP table.
// Uses netlink on Linux, exec commands on macOS.
func (s *ARPScanner) readARPTable() ([]*ARPEntry, error) {
	return s.readARPTablePlatform()
}

// isInSubnet checks if an IP is in the current subnet or any additional subnets.
func (s *ARPScanner) isInSubnet(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check primary subnet
	if s.subnet != nil && s.subnet.Contains(ip) {
		return true
	}

	// Check additional subnets
	for _, subnet := range s.additionalSubnets {
		if subnet.Contains(ip) {
			return true
		}
	}

	// If no subnets configured, accept all (fallback)
	return s.subnet == nil && len(s.additionalSubnets) == 0
}

// enrichEntries adds OUI lookups, hostname resolution, and TTL-based OS guessing.
func (s *ARPScanner) enrichEntries(ctx context.Context, entries []*ARPEntry) {
	// Note: pingResults already populated by pingSweep, no lock needed for reading
	pingResults := s.pingResults

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear old entries not in current scan
	newEntries := make(map[string]*ARPEntry)

	for _, entry := range entries {
		// OUI lookup
		if s.oui != nil {
			entry.Vendor = s.oui.LookupWithDefault(entry.MAC, "Unknown")
		}

		// Hostname resolution (with timeout)
		go func(e *ARPEntry) {
			resolveCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			defer cancel()

			names, err := net.DefaultResolver.LookupAddr(resolveCtx, e.IP)
			if err == nil && len(names) > 0 {
				s.mu.Lock()
				if existing, ok := s.entries[e.IP]; ok {
					existing.Hostname = strings.TrimSuffix(names[0], ".")
				}
				s.mu.Unlock()
			}
		}(entry)

		// Use TTL from cached ping results (already collected during ping sweep)
		if pr, ok := pingResults[entry.IP]; ok && pr.Reachable && pr.TTL > 0 {
			entry.TTL = pr.TTL
			entry.OSGuess = guessOSFromTTL(pr.TTL)
			entry.ResponseTime = pr.RTT.Milliseconds()
		}

		newEntries[entry.IP] = entry
	}

	s.entries = newEntries
}

// guessOSFromTTL makes a rough OS guess based on default TTL values.
func guessOSFromTTL(ttl int) string {
	// Common default TTL values:
	// 64: Linux, macOS, iOS, Android, FreeBSD
	// 128: Windows
	// 255: Cisco IOS, Solaris, some network devices
	// 60: HP-UX
	// 30: Some older network devices

	switch {
	case ttl <= 32:
		return "Network Device (Low TTL)"
	case ttl <= 64:
		return "Linux/macOS/Unix"
	case ttl <= 128:
		return "Windows"
	case ttl <= 255:
		return "Network Device/Cisco"
	default:
		return "Unknown"
	}
}

// normalizeMac converts MAC to uppercase colon-separated format.
func normalizeMac(mac string) string {
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", ":")
	return mac
}

// GetEntries returns all discovered ARP entries.
func (s *ARPScanner) GetEntries() []*ARPEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]*ARPEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, entry)
	}
	return entries
}

// GetEntry returns a specific ARP entry by IP.
func (s *ARPScanner) GetEntry(ip string) *ARPEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entries[ip]
}

// IsScanning returns true if a scan is in progress.
func (s *ARPScanner) IsScanning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanning
}

// LastScan returns the time of the last completed scan.
func (s *ARPScanner) LastScan() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScan
}

// Count returns the number of discovered entries.
func (s *ARPScanner) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// GetSubnetInfo returns the current subnet and local IP.
func (s *ARPScanner) GetSubnetInfo() (subnet, localIP string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.subnet != nil {
		subnet = s.subnet.String()
	}
	if s.localIP != nil {
		localIP = s.localIP.String()
	}
	return subnet, localIP
}
