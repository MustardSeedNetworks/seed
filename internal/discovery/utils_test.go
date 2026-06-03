package discovery_test

import (
	"net"
	"testing"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestExportIncrementIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		n        int
		expected string
	}{
		{"increment_by_1", "192.168.1.1", 1, "192.168.1.2"},
		{"increment_by_0", "192.168.1.1", 0, "192.168.1.1"},
		{"increment_by_10", "192.168.1.1", 10, "192.168.1.11"},
		{"increment_across_octet", "192.168.1.250", 10, "192.168.2.4"},
		{"increment_to_max", "192.168.1.254", 1, "192.168.1.255"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			result := discovery.ExportIncrementIP(ip, tt.n)
			if result == nil {
				t.Fatalf("incrementIP returned nil for %s + %d", tt.ip, tt.n)
			}
			if result.String() != tt.expected {
				t.Errorf("incrementIP(%s, %d) = %s, expected %s", tt.ip, tt.n, result, tt.expected)
			}
		})
	}
}

func TestExportNormalizeMac(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected string
	}{
		{"lowercase_colon", "aa:bb:cc:dd:ee:ff", "AA:BB:CC:DD:EE:FF"},
		{"uppercase_colon", "AA:BB:CC:DD:EE:FF", "AA:BB:CC:DD:EE:FF"},
		{"mixed_case", "Aa:Bb:Cc:Dd:Ee:Ff", "AA:BB:CC:DD:EE:FF"},
		{"dash_separator", "AA-BB-CC-DD-EE-FF", "AA:BB:CC:DD:EE:FF"},
		{"no_separator", "AABBCCDDEEFF", "AABBCCDDEEFF"},
		{"empty", "", ""},
		{"short_mac", "AA:BB", "AA:BB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := discovery.ExportNormalizeMac(tt.mac)
			if result != tt.expected {
				t.Errorf("normalizeMac(%q) = %q, expected %q", tt.mac, result, tt.expected)
			}
		})
	}
}

func TestExportGuessOSFromTTL(t *testing.T) {
	tests := []struct {
		name     string
		ttl      int
		expected string
	}{
		{"linux_64", 64, "Linux/macOS/Unix"},
		{"linux_63", 63, "Linux/macOS/Unix"},
		{"windows_128", 128, "Windows"},
		{"windows_127", 127, "Windows"},
		{"cisco_255", 255, "Network Device/Cisco"},
		{"cisco_254", 254, "Network Device/Cisco"},
		{"low_ttl_32", 32, "Network Device (Low TTL)"},
		{"windows_range_100", 100, "Windows"},
		{"low_ttl_0", 0, "Network Device (Low TTL)"},
		{"unknown_high", 256, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := discovery.ExportGuessOSFromTTL(tt.ttl)
			if result != tt.expected {
				t.Errorf("guessOSFromTTL(%d) = %q, expected %q", tt.ttl, result, tt.expected)
			}
		})
	}
}

func TestExportSplitSubnetIntoChunks(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		maxChunks   int
		minExpected int // At least this many chunks
		maxExpected int // At most this many chunks
	}{
		{"slash24_2chunks", "192.168.1.0/24", 2, 1, 2},
		{"slash24_4chunks", "192.168.1.0/24", 4, 1, 4},
		{"slash24_1chunk", "192.168.1.0/24", 1, 1, 1},
		{"slash16_16chunks", "10.0.0.0/16", 16, 1, 16},
		{"slash30_2chunks", "10.0.0.0/30", 2, 1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, subnet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("Failed to parse CIDR %s: %v", tt.cidr, err)
			}

			chunks := discovery.ExportSplitSubnetIntoChunks(subnet, tt.maxChunks)
			if len(chunks) < tt.minExpected || len(chunks) > tt.maxExpected {
				t.Errorf(
					"splitSubnetIntoChunks(%s, %d) returned %d chunks, expected %d-%d",
					tt.cidr,
					tt.maxChunks,
					len(chunks),
					tt.minExpected,
					tt.maxExpected,
				)
			}

			// Verify all chunks are valid subnets
			for i, chunk := range chunks {
				if chunk == nil {
					t.Errorf("Chunk %d is nil", i)
				}
			}
		})
	}
}

func TestNVDRateLimitConstants(t *testing.T) {
	// Test that NVD rate limit constants are accessible
	if discovery.NVDRateLimitNoKey <= 0 {
		t.Error("Expected NVDRateLimitNoKey to be positive")
	}
	if discovery.NVDRateLimitWithKey <= 0 {
		t.Error("Expected NVDRateLimitWithKey to be positive")
	}
	if discovery.NVDRateLimitWithKey <= discovery.NVDRateLimitNoKey {
		t.Error("Expected NVDRateLimitWithKey to be greater than NVDRateLimitNoKey")
	}
}
