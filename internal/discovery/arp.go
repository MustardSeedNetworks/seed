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
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
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

// DefaultMaxHostsPerSubnet is the default limit for hosts scanned per subnet.
// This can be increased via SetMaxHostsPerSubnet for larger networks at the cost of longer scan times.
const DefaultMaxHostsPerSubnet = 254

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
	maxHostsPerSubnet int // Configurable limit (0 = use default)
}

// NewARPScanner creates a new ARP scanner for the given interface.
func NewARPScanner(interfaceName string, oui *OUIDatabase) *ARPScanner {
	return &ARPScanner{
		interfaceName:     interfaceName,
		oui:               oui,
		entries:           make(map[string]*ARPEntry),
		maxHostsPerSubnet: DefaultMaxHostsPerSubnet,
	}
}

// SetMaxHostsPerSubnet configures the maximum hosts to scan per subnet.
// Set to 0 to use DefaultMaxHostsPerSubnet (254).
// For larger subnets like /22 or /16, increase this at the cost of longer scan times.
// Example: /22 = 1022 hosts, /16 = 65534 hosts.
func (s *ARPScanner) SetMaxHostsPerSubnet(maxHosts int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if maxHosts <= 0 {
		s.maxHostsPerSubnet = DefaultMaxHostsPerSubnet
	} else {
		s.maxHostsPerSubnet = maxHosts
	}
}

// GetMaxHostsPerSubnet returns the current host limit per subnet.
func (s *ARPScanner) GetMaxHostsPerSubnet() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.maxHostsPerSubnet <= 0 {
		return DefaultMaxHostsPerSubnet
	}
	return s.maxHostsPerSubnet
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
			// Fixes #738: Report the network (subnet) instead of the host IP.
			// ipNet.IP contains the interface's host address; mask it to the
			// network address so status.subnet renders as 192.168.1.0/24 instead
			// of 192.168.1.123/24.
			maskedIP := ipNet.IP.Mask(ipNet.Mask)
			networkIP := make(net.IP, len(maskedIP))
			copy(networkIP, maskedIP)

			return &net.IPNet{
				IP:   networkIP,
				Mask: ipNet.Mask,
			}, ipNet.IP, nil
		}
	}

	return nil, nil, fmt.Errorf("no IPv4 address found on interface %s", s.interfaceName)
}

// incrementIP adds n to an IP address.
// n must be non-negative and at most 0xFFFFFF (max hosts in /8 subnet).
// Returns nil if n is out of bounds or ip is not IPv4. (fixes #839).
func incrementIP(ip net.IP, n int) net.IP {
	ip = ip.To4()
	if ip == nil {
		return nil
	}
	// Validate n is within reasonable bounds for IP increment (fixes #839)
	// Max reasonable increment for a /8 subnet is 16777214 (2^24 - 2)
	if n < 0 || n > 0xFFFFFF {
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
		return errors.New("scan already in progress")
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

	// Perform ping sweep on primary subnet with retry logic
	// Note: ping sweep may fail without CAP_NET_RAW - continue with ARP table
	result := RetryWithBackoff(ctx, NetworkRetryConfig(), func() error {
		return s.pingSweep(ctx, subnet)
	})
	if !result.Successful {
		logging.GetLogger().Warn("Ping sweep failed after retries (continuing with ARP table only)",
			"subnet", subnet,
			"attempts", result.Attempts,
			"duration", result.TotalTime,
			"error", result.LastError)
	}

	// Perform ping sweep on additional subnets with retry logic
	for _, additionalSubnet := range additionalSubnets {
		select {
		case <-ctx.Done():
			return fmt.Errorf("ping sweep cancelled: %w", ctx.Err())
		default:
			// Retry logic for additional subnets - continue even if some fail
			subnetCopy := additionalSubnet // Capture for closure
			subnetResult := RetryWithBackoff(ctx, NetworkRetryConfig(), func() error {
				return s.pingSweep(ctx, subnetCopy)
			})
			if !subnetResult.Successful {
				logging.GetLogger().Warn("Ping sweep failed for additional subnet after retries",
					"subnet", additionalSubnet,
					"attempts", subnetResult.Attempts,
					"duration", subnetResult.TotalTime,
					"error", subnetResult.LastError)
			}
		}
	}

	// Read ARP table (will include entries from all scanned subnets)
	entries, err := s.readARPTable()
	if err != nil {
		return fmt.Errorf("failed to read ARP table: %w", err)
	}

	// Mark ARP entries based on whether they're in the primary subnet
	// Note: ARP can capture entries from additional subnets if they're routed through us
	for _, entry := range entries {
		entry.IsLocal = s.isInLocalSubnet(entry.IP)
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

	// Add ping responders not in ARP table
	// These could be local (in primary subnet) or from additional/extended subnets
	for _, ip := range responders {
		if !existingIPs[ip] {
			entries = append(entries, &ARPEntry{
				IP:       ip,
				MAC:      "", // No MAC - either not in ARP cache or remote host
				State:    "PING_ONLY",
				LastSeen: time.Now(),
				IsLocal:  s.isInLocalSubnet(ip), // Only primary subnet is "local"
			})
		}
	}

	// Enrich entries with OUI lookup and hostname resolution
	s.enrichEntries(ctx, entries)

	return nil
}

// MaxChunksDefault is the default maximum number of /24 chunks to scan.
// This provides a safety guardrail for very large CIDRs like /8.
// 256 chunks = /16 subnet = 65,534 hosts (reasonable upper bound).
const MaxChunksDefault = 256

// splitSubnetIntoChunks splits a large subnet into /24 chunks for manageable scanning.
// Returns the original subnet in a slice if it's already /24 or smaller.
// maxChunks limits the number of chunks to prevent memory/time issues with huge CIDRs.
// Pass 0 for maxChunks to use MaxChunksDefault.
func splitSubnetIntoChunks(subnet *net.IPNet, maxChunks int) []*net.IPNet {
	ones, bits := subnet.Mask.Size()
	if bits != 32 {
		// IPv6 or invalid - return as-is
		return []*net.IPNet{subnet}
	}

	// /24 or smaller - no need to chunk
	if ones >= 24 {
		return []*net.IPNet{subnet}
	}

	// Calculate number of /24 chunks needed
	numChunks := 1 << (24 - ones) // e.g., /22 = 4 chunks, /16 = 256 chunks

	// Apply safety cap
	if maxChunks <= 0 {
		maxChunks = MaxChunksDefault
	}
	if numChunks > maxChunks {
		logging.GetLogger().Warn("Subnet too large - capping chunk count",
			"subnet", subnet.String(),
			"totalChunks", numChunks,
			"maxChunks", maxChunks,
			"coverage", fmt.Sprintf("%.1f%%", float64(maxChunks)/float64(numChunks)*100))
		numChunks = maxChunks
	}

	chunks := make([]*net.IPNet, 0, numChunks)
	baseIP := subnet.IP.Mask(subnet.Mask).To4()
	if baseIP == nil {
		return []*net.IPNet{subnet}
	}

	// Convert base IP to uint32 for proper arithmetic
	baseUint := uint32(baseIP[0])<<24 | uint32(baseIP[1])<<16 | uint32(baseIP[2])<<8 | uint32(baseIP[3])

	for i := range numChunks {
		// Calculate the starting IP for this /24 chunk
		// Explicit bounds check for safe uint32 conversion (i is bounded by maxChunks ≤ 256)
		if i < 0 || i > 256 {
			continue
		}
		offset := uint32(i) * 256
		chunkUint := baseUint + offset

		chunkIP := net.IP{
			byte(chunkUint >> 24),
			byte(chunkUint >> 16),
			byte(chunkUint >> 8),
			0, // Start of /24 block
		}

		chunk := &net.IPNet{
			IP:   chunkIP,
			Mask: net.CIDRMask(24, 32),
		}
		chunks = append(chunks, chunk)
	}

	return chunks
}

// pingSweep sends ICMP echo requests to all hosts in the subnet using raw sockets.
// For subnets larger than /24, automatically splits into /24 chunks and scans sequentially.
// Respects maxHostsPerSubnet configuration to cap total hosts scanned.
func (s *ARPScanner) pingSweep(ctx context.Context, subnet *net.IPNet) error {
	ones, bits := subnet.Mask.Size()
	totalHosts := 1<<(bits-ones) - 2 // Exclude network and broadcast

	// Calculate max chunks based on configured host limit
	// maxHostsPerSubnet / 254 = max /24 chunks to scan
	maxHosts := s.GetMaxHostsPerSubnet()
	maxChunks := (maxHosts + 253) / 254 // Round up

	// For large subnets, split into /24 chunks and scan sequentially
	chunks := splitSubnetIntoChunks(subnet, maxChunks)
	if len(chunks) > 1 {
		logging.GetLogger().Info("Large subnet detected - scanning in chunks",
			"subnet", subnet.String(),
			"totalHosts", totalHosts,
			"chunks", len(chunks),
			"maxHosts", maxHosts)

		for i, chunk := range chunks {
			select {
			case <-ctx.Done():
				return fmt.Errorf("chunk scan cancelled: %w", ctx.Err())
			default:
				logging.GetLogger().Debug("Scanning chunk",
					"chunk", fmt.Sprintf("%d/%d", i+1, len(chunks)),
					"subnet", chunk.String())

				if err := s.pingSweepChunk(ctx, chunk); err != nil {
					logging.GetLogger().Warn("Chunk scan failed - continuing with remaining chunks",
						"chunk", chunk.String(),
						"error", err)
				}
			}
		}
		return nil
	}

	// Small subnet - scan directly
	return s.pingSweepChunk(ctx, subnet)
}

// pingSweepChunk scans a single /24 or smaller subnet chunk.
func (s *ARPScanner) pingSweepChunk(ctx context.Context, subnet *net.IPNet) error {
	ones, bits := subnet.Mask.Size()
	numHosts := 1<<(bits-ones) - 2 // Exclude network and broadcast

	// Generate host IPs
	baseIP := subnet.IP.Mask(subnet.Mask).To4()
	if baseIP == nil {
		return errors.New("invalid subnet")
	}

	// Build list of IPs to ping
	var ips []net.IP
	for i := 1; i <= numHosts; i++ {
		ip := incrementIP(baseIP, i)
		if ip != nil && !ip.Equal(s.localIP) {
			ips = append(ips, ip)
		}
	}

	// Initialize pinger if needed (fixes #822 - check under lock)
	s.mu.Lock()
	if s.pinger == nil {
		pinger, err := NewICMPPinger(time.Second)
		if err != nil {
			s.mu.Unlock()
			logging.GetLogger().Warn("Failed to create ICMP pinger", "error", err)
			return err
		}
		s.pinger = pinger
	}
	pinger := s.pinger // Copy reference under lock
	s.mu.Unlock()

	// Perform ping sweep using raw ICMP sockets (50 workers)
	results := pinger.PingSweep(ctx, ips, 50)

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
	// Access pinger under lock to avoid race with pingSweep (fixes #826)
	s.mu.Lock()
	pinger := s.pinger
	s.pinger = nil
	s.mu.Unlock()

	if pinger != nil {
		return pinger.Close()
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

// isInLocalSubnet checks if an IP is in the PRIMARY subnet only (not additional subnets).
// This is used to determine if a device should be shown in "Local Network" vs "Extended Networks".
func (s *ARPScanner) isInLocalSubnet(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Only check primary subnet - additional subnets are "extended" networks
	return s.subnet != nil && s.subnet.Contains(ip)
}

// enrichEntries adds OUI lookups, hostname resolution, and TTL-based OS guessing.
func (s *ARPScanner) enrichEntries(ctx context.Context, entries []*ARPEntry) {
	s.mu.Lock()

	// Copy pingResults under lock to avoid data race (fixes #819)
	pingResults := s.pingResults

	// Clear old entries not in current scan
	newEntries := make(map[string]*ARPEntry)

	for _, entry := range entries {
		// OUI lookup - but first check if it's a locally administered address
		if s.oui != nil {
			if isLocallyAdministeredMAC(entry.MAC) {
				entry.Vendor = "LAA"
			} else {
				entry.Vendor = s.oui.LookupWithDefault(entry.MAC, "Unknown")
			}
		}

		// Use TTL from cached ping results (already collected during ping sweep)
		if pr, ok := pingResults[entry.IP]; ok && pr.Reachable && pr.TTL > 0 {
			entry.TTL = pr.TTL
			entry.OSGuess = guessOSFromTTL(pr.TTL)
			entry.ResponseTime = pr.RTT.Milliseconds()
		}

		newEntries[entry.IP] = entry
	}

	s.entries = newEntries
	s.mu.Unlock()

	// Hostname resolution with WaitGroup to prevent goroutine leak (fixes #823)
	var wg sync.WaitGroup
	for _, entry := range entries {
		wg.Add(1)
		go func(e *ARPEntry) {
			defer wg.Done()

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
	}
	wg.Wait()
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
func (s *ARPScanner) GetSubnetInfo() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var subnet, localIP string
	if s.subnet != nil {
		subnet = s.subnet.String()
	}
	if s.localIP != nil {
		localIP = s.localIP.String()
	}
	return subnet, localIP
}
