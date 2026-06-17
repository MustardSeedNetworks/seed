package enumerate

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

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/resolve"
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

// IP address and byte manipulation constants.
const (
	ipv4ByteSize          = 4    // Size of an IPv4 address in bytes
	byteMask              = 0xff // Mask for extracting a single byte
	percentMultiplier     = 100  // Multiplier for percentage calculations
	byteShift24           = 24   // Bit shift for first byte of IPv4
	byteShift16           = 16   // Bit shift for second byte of IPv4
	byteShift8            = 8    // Bit shift for third byte of IPv4
	cidrMask24            = 24   // CIDR prefix length for /24 subnet
	cidrBits32            = 32   // Total bits in IPv4 address
	pingSweepWorkers      = 50   // Number of concurrent workers for ping sweep
	hostnameResolveMs     = 500  // Timeout in milliseconds for hostname resolution
	hostsPerSubnet24      = 254  // Usable hosts in a /24 subnet (excluding network/broadcast)
	subnetExcludeCount    = 2    // Number of addresses excluded (network + broadcast)
	roundUpAdjust         = 253  // Value added before division for chunk count rounding
	hostsPerSubnet24Block = 256  // Total IPs in a /24 block (including network/broadcast)
)

// TTL threshold constants for OS detection heuristics.
const (
	ttlThresholdLow     = 32  // TTL threshold for low-TTL network devices
	ttlThresholdLinux   = 64  // Default TTL for Linux/macOS/Unix systems
	ttlThresholdWindows = 128 // Default TTL for Windows systems
	ttlThresholdNetwork = 255 // Default TTL for network devices/Cisco
)

// ARPScanner performs active network discovery via ARP.
type ARPScanner struct {
	interfaceName     string
	oui               *resolve.OUIDatabase
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
func NewARPScanner(interfaceName string, oui *resolve.OUIDatabase) *ARPScanner {
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
	result := make(net.IP, ipv4ByteSize)
	copy(result, ip)

	carry := n
	for i := 3; i >= 0 && carry > 0; i-- {
		sum := int(result[i]) + (carry & byteMask)
		result[i] = byte(sum & byteMask)
		carry = (carry >> byteShift8) + (sum >> byteShift8)
	}

	return result
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
