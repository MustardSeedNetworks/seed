package discovery

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ARPEntry represents a discovered device from ARP scanning.
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
}

// ARPScanner performs active network discovery via ARP.
type ARPScanner struct {
	interfaceName string
	oui           *OUIDatabase
	mu            sync.RWMutex
	entries       map[string]*ARPEntry // Key by IP
	subnet        *net.IPNet
	localIP       net.IP
	scanning      bool
	lastScan      time.Time
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

// generateHosts returns all host IPs in the subnet (excluding network and broadcast).
func (s *ARPScanner) generateHosts(subnet *net.IPNet) []net.IP {
	var hosts []net.IP
	ip := subnet.IP.Mask(subnet.Mask)

	// Calculate max hosts to scan (limit to /16 = 65534 hosts)
	ones, bits := subnet.Mask.Size()
	maxHosts := 1 << (bits - ones)
	if maxHosts > 65536 {
		maxHosts = 65536
	}

	for i := 1; i < maxHosts-1; i++ {
		hostIP := make(net.IP, len(ip))
		copy(hostIP, ip)

		// Add offset
		for j := len(hostIP) - 1; j >= 0 && i > 0; j-- {
			hostIP[j] += byte(i & 0xff)
			i >>= 8
			i = 0 // Only do this once
		}

		// Calculate host IP properly
		hostIP = incrementIP(ip, i)
		if hostIP != nil && !hostIP.Equal(s.localIP) {
			hosts = append(hosts, hostIP)
		}
	}

	return hosts
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

	// Perform ping sweep
	if err := s.pingSweep(ctx, subnet); err != nil {
		return fmt.Errorf("ping sweep failed: %w", err)
	}

	// Read ARP table
	entries, err := s.readARPTable()
	if err != nil {
		return fmt.Errorf("failed to read ARP table: %w", err)
	}

	// Enrich entries with OUI lookup and hostname resolution
	s.enrichEntries(ctx, entries)

	return nil
}

// pingSweep sends ICMP echo requests to all hosts in the subnet.
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

	// Use worker pool for concurrent pings
	const workers = 50
	jobs := make(chan net.IP, numHosts)
	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					s.pingHost(ctx, ip)
				}
			}
		}()
	}

	// Queue jobs
	for i := 1; i <= numHosts; i++ {
		ip := incrementIP(baseIP, i)
		if ip != nil && !ip.Equal(s.localIP) {
			select {
			case jobs <- ip:
			case <-ctx.Done():
				close(jobs)
				wg.Wait()
				return ctx.Err()
			}
		}
	}

	close(jobs)
	wg.Wait()

	return nil
}

// pingHost sends a single ping to a host.
func (s *ARPScanner) pingHost(ctx context.Context, ip net.IP) {
	var cmd *exec.Cmd
	ipStr := ip.String()

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "100", ipStr)
	case "linux":
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ipStr)
	default:
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ipStr)
	}

	// We don't care about the result, just populate the ARP cache
	_ = cmd.Run()
}

// readARPTable reads the system ARP table.
func (s *ARPScanner) readARPTable() ([]*ARPEntry, error) {
	switch runtime.GOOS {
	case "darwin":
		return s.readARPTableDarwin()
	case "linux":
		return s.readARPTableLinux()
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// readARPTableDarwin reads ARP table on macOS.
func (s *ARPScanner) readARPTableDarwin() ([]*ARPEntry, error) {
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var entries []*ARPEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		// Format: hostname (IP) at MAC on interface [ifscope ...]
		// Example: ? (192.168.1.1) at 00:11:22:33:44:55 on en0 ifscope [ethernet]

		entry := s.parseARPLineDarwin(line)
		if entry != nil && s.isInSubnet(entry.IP) {
			entries = append(entries, entry)
		}
	}

	return entries, scanner.Err()
}

// parseARPLineDarwin parses a single ARP line on macOS.
func (s *ARPScanner) parseARPLineDarwin(line string) *ARPEntry {
	// Find IP in parentheses
	start := strings.Index(line, "(")
	end := strings.Index(line, ")")
	if start < 0 || end < 0 || end <= start {
		return nil
	}
	ip := line[start+1 : end]

	// Find MAC after "at "
	atIdx := strings.Index(line, " at ")
	if atIdx < 0 {
		return nil
	}
	rest := line[atIdx+4:]

	// MAC is next token
	parts := strings.Fields(rest)
	if len(parts) < 1 {
		return nil
	}
	mac := parts[0]

	// Skip incomplete entries
	if mac == "(incomplete)" {
		return nil
	}

	// Find interface after "on "
	var iface string
	onIdx := strings.Index(rest, " on ")
	if onIdx >= 0 {
		ifaceParts := strings.Fields(rest[onIdx+4:])
		if len(ifaceParts) > 0 {
			iface = ifaceParts[0]
		}
	}

	// Normalize MAC format
	mac = normalizeMac(mac)

	return &ARPEntry{
		IP:        ip,
		MAC:       mac,
		Interface: iface,
		State:     "REACHABLE",
		LastSeen:  time.Now(),
	}
}

// readARPTableLinux reads ARP table on Linux.
func (s *ARPScanner) readARPTableLinux() ([]*ARPEntry, error) {
	// Use ip neigh for more detailed info
	cmd := exec.Command("ip", "neigh", "show")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to arp -a
		return s.readARPTableLinuxFallback()
	}

	var entries []*ARPEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		// Format: IP dev INTERFACE lladdr MAC STATE
		// Example: 192.168.1.1 dev eth0 lladdr 00:11:22:33:44:55 REACHABLE

		entry := s.parseIPNeighLine(line)
		if entry != nil && s.isInSubnet(entry.IP) {
			entries = append(entries, entry)
		}
	}

	return entries, scanner.Err()
}

// parseIPNeighLine parses a line from `ip neigh show`.
func (s *ARPScanner) parseIPNeighLine(line string) *ARPEntry {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return nil
	}

	ip := parts[0]
	var mac, iface, state string

	for i := 1; i < len(parts); i++ {
		switch parts[i] {
		case "dev":
			if i+1 < len(parts) {
				iface = parts[i+1]
				i++
			}
		case "lladdr":
			if i+1 < len(parts) {
				mac = parts[i+1]
				i++
			}
		case "REACHABLE", "STALE", "DELAY", "PROBE", "PERMANENT", "NOARP", "INCOMPLETE", "FAILED":
			state = parts[i]
		}
	}

	// Skip entries without MAC
	if mac == "" || state == "INCOMPLETE" || state == "FAILED" {
		return nil
	}

	mac = normalizeMac(mac)

	return &ARPEntry{
		IP:        ip,
		MAC:       mac,
		Interface: iface,
		State:     state,
		LastSeen:  time.Now(),
	}
}

// readARPTableLinuxFallback reads ARP table using arp -a on Linux.
func (s *ARPScanner) readARPTableLinuxFallback() ([]*ARPEntry, error) {
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var entries []*ARPEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		entry := s.parseARPLineDarwin(line) // Similar format
		if entry != nil && s.isInSubnet(entry.IP) {
			entries = append(entries, entry)
		}
	}

	return entries, scanner.Err()
}

// isInSubnet checks if an IP is in the current subnet.
func (s *ARPScanner) isInSubnet(ipStr string) bool {
	if s.subnet == nil {
		return true // No filter if subnet unknown
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return s.subnet.Contains(ip)
}

// enrichEntries adds OUI lookups, hostname resolution, and TTL-based OS guessing.
func (s *ARPScanner) enrichEntries(ctx context.Context, entries []*ARPEntry) {
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

		// TTL-based OS guess via ping
		ttl := s.getTTL(ctx, entry.IP)
		if ttl > 0 {
			entry.TTL = ttl
			entry.OSGuess = guessOSFromTTL(ttl)
		}

		newEntries[entry.IP] = entry
	}

	s.entries = newEntries
}

// getTTL gets the TTL from a ping response.
func (s *ARPScanner) getTTL(ctx context.Context, ip string) int {
	var cmd *exec.Cmd

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(pingCtx, "ping", "-c", "1", "-W", "1000", ip)
	case "linux":
		cmd = exec.CommandContext(pingCtx, "ping", "-c", "1", "-W", "1", ip)
	default:
		return 0
	}

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse TTL from output
	// macOS: 64 bytes from 192.168.1.1: icmp_seq=0 ttl=64 time=1.234 ms
	// Linux: 64 bytes from 192.168.1.1: icmp_seq=1 ttl=64 time=1.23 ms
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "ttl=") {
			var ttl int
			if _, err := fmt.Sscanf(line[strings.Index(line, "ttl="):], "ttl=%d", &ttl); err == nil {
				return ttl
			}
		}
	}

	return 0
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
func (s *ARPScanner) GetSubnetInfo() (subnet string, localIP string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.subnet != nil {
		subnet = s.subnet.String()
	}
	if s.localIP != nil {
		localIP = s.localIP.String()
	}
	return
}
