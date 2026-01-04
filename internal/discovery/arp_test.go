// Package discovery_test provides ARP scanner tests.
package discovery_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestNewARPScanner(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	if scanner == nil {
		t.Fatal("NewARPScanner returned nil")
	}

	accessor := &discovery.ARPScannerTestAccessor{Scanner: scanner}
	if accessor.GetInterfaceName() != "lo" {
		t.Errorf("Expected interface 'lo', got %q", accessor.GetInterfaceName())
	}
	if accessor.GetOUI() != oui {
		t.Error("OUI database not set correctly")
	}
	if accessor.GetEntries() == nil {
		t.Error("entries map should be initialized")
	}
}

func TestARPScanner_SetInterface(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	scanner.SetInterface("eth0")

	accessor := &discovery.ARPScannerTestAccessor{Scanner: scanner}
	if accessor.GetInterfaceName() != "eth0" {
		t.Errorf("Expected interface 'eth0', got %q", accessor.GetInterfaceName())
	}
	// subnet and localIP should be reset
	if accessor.GetSubnet() != nil {
		t.Error("subnet should be nil after SetInterface")
	}
	if accessor.GetLocalIP() != nil {
		t.Error("localIP should be nil after SetInterface")
	}
}

func TestARPScanner_SetAdditionalSubnets(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	// Valid CIDRs
	err := scanner.SetAdditionalSubnets([]string{"192.168.1.0/24", "10.0.0.0/8"})
	if err != nil {
		t.Fatalf("SetAdditionalSubnets failed: %v", err)
	}

	subnets := scanner.GetAdditionalSubnets()
	if len(subnets) != 2 {
		t.Errorf("Expected 2 subnets, got %d", len(subnets))
	}

	// Invalid CIDR
	err = scanner.SetAdditionalSubnets([]string{"invalid"})
	if err == nil {
		t.Error("Expected error for invalid CIDR")
	}
}

func TestARPScanner_GetAdditionalSubnets(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	// Empty initially
	subnets := scanner.GetAdditionalSubnets()
	if len(subnets) != 0 {
		t.Errorf("Expected 0 subnets initially, got %d", len(subnets))
	}

	// After setting
	_ = scanner.SetAdditionalSubnets([]string{"172.16.0.0/16"})
	subnets = scanner.GetAdditionalSubnets()
	if len(subnets) != 1 {
		t.Errorf("Expected 1 subnet, got %d", len(subnets))
	}
	if subnets[0] != "172.16.0.0/16" {
		t.Errorf("Expected '172.16.0.0/16', got %q", subnets[0])
	}
}

func TestIncrementIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		n        int
		expected string
	}{
		{"increment by 1", "192.168.1.1", 1, "192.168.1.2"},
		{"increment by 10", "192.168.1.1", 10, "192.168.1.11"},
		{"increment across octet", "192.168.1.254", 2, "192.168.2.0"},
		{"increment by 0", "192.168.1.1", 0, "192.168.1.1"},
		{"increment at boundary", "192.168.1.255", 1, "192.168.2.0"},
		{"increment from zero", "0.0.0.0", 1, "0.0.0.1"},
		{"large increment", "192.168.0.0", 256, "192.168.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip).To4()
			result := discovery.IncrementIP(ip, tt.n)
			if result.String() != tt.expected {
				t.Errorf("incrementIP(%s, %d) = %s, want %s", tt.ip, tt.n, result.String(), tt.expected)
			}
		})
	}
}

func TestIncrementIP_Nil(t *testing.T) {
	// IPv6 address should return nil (only IPv4 supported)
	ip := net.ParseIP("::1")
	result := discovery.IncrementIP(ip, 1)
	if result != nil {
		t.Errorf("Expected nil for IPv6, got %v", result)
	}

	// nil input
	result = discovery.IncrementIP(nil, 1)
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}
}

func TestNormalizeMac(t *testing.T) {
	// normalizeMac only handles colon and hyphen formats (uppercase + hyphen to colon)
	// It does NOT convert cisco format (aabb.ccdd.eeff) or compact format (aabbccddeeff)
	tests := []struct {
		input    string
		expected string
	}{
		{"AA:BB:CC:DD:EE:FF", "AA:BB:CC:DD:EE:FF"},
		{"aa:bb:cc:dd:ee:ff", "AA:BB:CC:DD:EE:FF"},
		{"AA-BB-CC-DD-EE-FF", "AA:BB:CC:DD:EE:FF"},
		{"aa-bb-cc-dd-ee-ff", "AA:BB:CC:DD:EE:FF"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := discovery.NormalizeMac(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeMac(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGuessOSFromTTL(t *testing.T) {
	tests := []struct {
		ttl      int
		expected string
	}{
		{64, "Linux/macOS/Unix"},
		{63, "Linux/macOS/Unix"},
		{60, "Linux/macOS/Unix"},
		{33, "Linux/macOS/Unix"},
		{128, "Windows"},
		{127, "Windows"},
		{100, "Windows"},
		{65, "Windows"},
		{255, "Network Device/Cisco"},
		{254, "Network Device/Cisco"},
		{200, "Network Device/Cisco"},
		{129, "Network Device/Cisco"},
		{32, "Network Device (Low TTL)"},
		{30, "Network Device (Low TTL)"},
		{1, "Network Device (Low TTL)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := discovery.GuessOSFromTTL(tt.ttl)
			if result != tt.expected {
				t.Errorf("guessOSFromTTL(%d) = %q, want %q", tt.ttl, result, tt.expected)
			}
		})
	}
}

func TestARPScanner_IsScanning(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	// Initially not scanning
	if scanner.IsScanning() {
		t.Error("Should not be scanning initially")
	}
}

func TestARPScanner_LastScan(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	// Initially zero
	if !scanner.LastScan().IsZero() {
		t.Error("LastScan should be zero initially")
	}
}

func TestARPScanner_Count(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	// Initially zero
	if scanner.Count() != 0 {
		t.Errorf("Expected count 0, got %d", scanner.Count())
	}
}

func TestARPScanner_GetEntries(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	// Initially empty
	entries := scanner.GetEntries()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestARPScanner_GetEntry(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	// Non-existent entry
	entry := scanner.GetEntry("192.168.1.1")
	if entry != nil {
		t.Error("Expected nil for non-existent entry")
	}
}

func TestARPScanner_ScanAlreadyInProgress(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	// Set scanning to true manually
	accessor := &discovery.ARPScannerTestAccessor{Scanner: scanner}
	accessor.Lock()
	accessor.SetScanning(true)
	accessor.Unlock()

	ctx := context.Background()
	err := scanner.Scan(ctx)
	if err == nil {
		t.Error("Expected error when scan already in progress")
	}
	if err.Error() != "scan already in progress" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestARPScanner_ScanInvalidInterface(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("nonexistent_interface_xyz", oui)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := scanner.Scan(ctx)
	if err == nil {
		t.Error("Expected error for invalid interface")
	}
}

func TestARPEntry_Fields(t *testing.T) {
	entry := discovery.ARPEntry{
		IP:           "192.168.1.1",
		MAC:          "AA:BB:CC:DD:EE:FF",
		Vendor:       "Test Vendor",
		Hostname:     "test-host",
		Interface:    "eth0",
		State:        "REACHABLE",
		TTL:          64,
		OSGuess:      "Linux/Unix",
		LastSeen:     time.Now(),
		ResponseTime: 5,
		IsLocal:      true,
	}

	if entry.IP != "192.168.1.1" {
		t.Errorf("Expected IP '192.168.1.1', got %q", entry.IP)
	}
	if entry.MAC != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("Expected MAC 'AA:BB:CC:DD:EE:FF', got %q", entry.MAC)
	}
	if entry.Vendor != "Test Vendor" {
		t.Errorf("Expected Vendor 'Test Vendor', got %q", entry.Vendor)
	}
	if entry.Hostname != "test-host" {
		t.Errorf("Expected Hostname 'test-host', got %q", entry.Hostname)
	}
	if entry.Interface != "eth0" {
		t.Errorf("Expected Interface 'eth0', got %q", entry.Interface)
	}
	if entry.State != "REACHABLE" {
		t.Errorf("Expected State 'REACHABLE', got %q", entry.State)
	}
	if entry.TTL != 64 {
		t.Errorf("Expected TTL 64, got %d", entry.TTL)
	}
	if entry.OSGuess != "Linux/Unix" {
		t.Errorf("Expected OSGuess 'Linux/Unix', got %q", entry.OSGuess)
	}
	if entry.LastSeen.IsZero() {
		t.Error("Expected LastSeen to be set")
	}
	if entry.ResponseTime != 5 {
		t.Errorf("Expected ResponseTime 5, got %d", entry.ResponseTime)
	}
	if !entry.IsLocal {
		t.Error("Expected IsLocal to be true")
	}
}

func TestARPScanner_isInSubnet(t *testing.T) {
	oui := discovery.NewOUIDatabase()
	scanner := discovery.NewARPScanner("lo", oui)

	// Set up a subnet on the scanner
	_, subnet, _ := net.ParseCIDR("192.168.1.0/24")
	accessor := &discovery.ARPScannerTestAccessor{Scanner: scanner}
	accessor.Lock()
	accessor.SetSubnet(subnet)
	accessor.Unlock()

	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.168.1.1", true},
		{"192.168.1.255", true},
		{"192.168.2.1", false},
		{"10.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := scanner.IsInSubnet(tt.ip)
			if result != tt.expected {
				t.Errorf("isInSubnet(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestSplitSubnetIntoChunks(t *testing.T) {
	tests := []struct {
		name           string
		cidr           string
		maxChunks      int // 0 = use default
		expectedChunks int
		firstChunk     string
		lastChunk      string
	}{
		{
			name:           "/24 - no chunking needed",
			cidr:           "192.168.1.0/24",
			expectedChunks: 1,
			firstChunk:     "192.168.1.0/24",
			lastChunk:      "192.168.1.0/24",
		},
		{
			name:           "/25 - smaller than /24",
			cidr:           "192.168.1.0/25",
			expectedChunks: 1,
			firstChunk:     "192.168.1.0/25",
			lastChunk:      "192.168.1.0/25",
		},
		{
			name:           "/23 - 2 chunks",
			cidr:           "192.168.0.0/23",
			expectedChunks: 2,
			firstChunk:     "192.168.0.0/24",
			lastChunk:      "192.168.1.0/24",
		},
		{
			name:           "/22 - 4 chunks",
			cidr:           "10.0.0.0/22",
			expectedChunks: 4,
			firstChunk:     "10.0.0.0/24",
			lastChunk:      "10.0.3.0/24",
		},
		{
			name:           "/20 - 16 chunks",
			cidr:           "172.16.0.0/20",
			expectedChunks: 16,
			firstChunk:     "172.16.0.0/24",
			lastChunk:      "172.16.15.0/24",
		},
		{
			name:           "/16 - 256 chunks (capped by default)",
			cidr:           "10.0.0.0/16",
			expectedChunks: discovery.MaxChunksDefault, // 256 - capped by default
			firstChunk:     "10.0.0.0/24",
			lastChunk:      "10.0.255.0/24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, subnet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("Invalid CIDR %s: %v", tt.cidr, err)
			}

			chunks := discovery.SplitSubnetIntoChunks(subnet, tt.maxChunks)

			if len(chunks) != tt.expectedChunks {
				t.Errorf("Expected %d chunks, got %d", tt.expectedChunks, len(chunks))
			}

			if len(chunks) > 0 {
				if chunks[0].String() != tt.firstChunk {
					t.Errorf("First chunk: expected %s, got %s", tt.firstChunk, chunks[0].String())
				}
				if chunks[len(chunks)-1].String() != tt.lastChunk {
					t.Errorf("Last chunk: expected %s, got %s", tt.lastChunk, chunks[len(chunks)-1].String())
				}
			}

			// Verify all chunks are /24 for large subnets
			if tt.expectedChunks > 1 {
				for i, chunk := range chunks {
					ones, _ := chunk.Mask.Size()
					if ones != 24 {
						t.Errorf("Chunk %d: expected /24 mask, got /%d", i, ones)
					}
				}
			}

			// Verify chunks are contiguous (no gaps)
			if tt.expectedChunks > 1 {
				for i := 1; i < len(chunks); i++ {
					prevIP := chunks[i-1].IP.To4()
					currIP := chunks[i].IP.To4()
					prevUint := uint32(prevIP[0])<<24 | uint32(prevIP[1])<<16 | uint32(prevIP[2])<<8 | uint32(prevIP[3])
					currUint := uint32(currIP[0])<<24 | uint32(currIP[1])<<16 | uint32(currIP[2])<<8 | uint32(currIP[3])
					if currUint-prevUint != 256 {
						t.Errorf("Chunks %d and %d are not contiguous: %s -> %s (gap: %d)",
							i-1, i, chunks[i-1].IP, chunks[i].IP, currUint-prevUint)
					}
				}
			}
		})
	}
}

func TestSplitSubnetIntoChunks_MaxChunksCap(t *testing.T) {
	// Test /8 supernet with explicit cap
	_, subnet, _ := net.ParseCIDR("10.0.0.0/8")

	// With cap of 16, should only get 16 chunks
	chunks := discovery.SplitSubnetIntoChunks(subnet, 16)
	if len(chunks) != 16 {
		t.Errorf("/8 with maxChunks=16: expected 16 chunks, got %d", len(chunks))
	}

	// First chunk should still be correct
	if chunks[0].String() != "10.0.0.0/24" {
		t.Errorf("First chunk: expected 10.0.0.0/24, got %s", chunks[0].String())
	}

	// Last chunk should be 10.0.15.0/24 (16th chunk, 0-indexed)
	if chunks[len(chunks)-1].String() != "10.0.15.0/24" {
		t.Errorf("Last chunk: expected 10.0.15.0/24, got %s", chunks[len(chunks)-1].String())
	}
}

func TestSplitSubnetIntoChunks_DefaultCap(t *testing.T) {
	// Test /8 supernet with default cap (MaxChunksDefault = 256)
	_, subnet, _ := net.ParseCIDR("10.0.0.0/8")

	// With default cap, should get MaxChunksDefault chunks
	chunks := discovery.SplitSubnetIntoChunks(subnet, 0)
	if len(chunks) != discovery.MaxChunksDefault {
		t.Errorf("/8 with default cap: expected %d chunks, got %d", discovery.MaxChunksDefault, len(chunks))
	}

	// First chunk should still be correct
	if chunks[0].String() != "10.0.0.0/24" {
		t.Errorf("First chunk: expected 10.0.0.0/24, got %s", chunks[0].String())
	}
}
