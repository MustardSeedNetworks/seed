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
		{"ipv6_increment", "fe80::1", 1, "fe80::2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			result := discovery.ExportIncrementIP(ip, tt.n)
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
		{"lowercase_colon", "aa:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:ff"},
		{"uppercase_colon", "AA:BB:CC:DD:EE:FF", "aa:bb:cc:dd:ee:ff"},
		{"mixed_case", "Aa:Bb:Cc:Dd:Ee:Ff", "aa:bb:cc:dd:ee:ff"},
		{"dash_separator", "AA-BB-CC-DD-EE-FF", "aa:bb:cc:dd:ee:ff"},
		{"no_separator", "AABBCCDDEEFF", "aa:bb:cc:dd:ee:ff"},
		{"empty", "", ""},
		{"short_mac", "AA:BB", "aa:bb"},
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
		{"linux_64", 64, "Linux/Unix"},
		{"linux_63", 63, "Linux/Unix"},
		{"windows_128", 128, "Windows"},
		{"windows_127", 127, "Windows"},
		{"cisco_255", 255, "Cisco/Network"},
		{"cisco_254", 254, "Cisco/Network"},
		{"solaris_32", 32, "Solaris"},
		{"unknown_100", 100, ""},
		{"unknown_0", 0, ""},
		{"unknown_256", 256, ""},
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
		name          string
		cidr          string
		maxChunks     int
		expectedCount int
	}{
		{"slash24_2chunks", "192.168.1.0/24", 2, 2},
		{"slash24_4chunks", "192.168.1.0/24", 4, 4},
		{"slash24_1chunk", "192.168.1.0/24", 1, 1},
		{"slash16_16chunks", "10.0.0.0/16", 16, 16},
		{"slash30_2chunks", "10.0.0.0/30", 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, subnet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("Failed to parse CIDR %s: %v", tt.cidr, err)
			}

			chunks := discovery.ExportSplitSubnetIntoChunks(subnet, tt.maxChunks)
			if len(chunks) != tt.expectedCount {
				t.Errorf(
					"splitSubnetIntoChunks(%s, %d) returned %d chunks, expected %d",
					tt.cidr,
					tt.maxChunks,
					len(chunks),
					tt.expectedCount,
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
