package enumerate_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// TestIsLocallyAdministeredMAC covers the moved isLocallyAdministeredMAC helper
// (relocated from the discovery kernel with the wired collector, ADR-0018
// Phase 6). It consolidates the former kernel TestIsLocallyAdministeredMAC and
// TestIsLocallyAdministeredMAC_Comprehensive.
func TestIsLocallyAdministeredMAC(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected bool
	}{
		// LAA MACs (second bit of first octet set)
		{"laa_02", "02:00:00:00:00:00", true},
		{"laa_06", "06:00:00:00:00:00", true},
		{"laa_0A", "0A:11:22:33:44:55", true},
		{"laa_0E", "0E:00:00:00:00:00", true},
		{"laa_vmware", "02:50:56:00:00:01", true},
		{"laa_hyperv", "02:15:5D:00:00:01", true},
		{"laa_docker", "02:42:AC:11:00:02", true},
		{"laa_xen", "02:00:00:00:00:01", true},

		// UAA MACs (second bit NOT set)
		{"uaa_00", "00:11:22:33:44:55", false},
		{"uaa_04", "04:00:00:00:00:00", false},
		{"uaa_08", "08:00:00:00:00:00", false},
		{"uaa_apple", "A4:83:E7:12:34:56", false},
		{"uaa_dell", "D0:67:E5:12:34:56", false},
		{"uaa_cisco", "00:1A:2B:3C:4D:5E", false},

		// Edge cases
		{"empty", "", false},
		{"too_short", "0", false},
		{"single_char", "A", false},
		{"lowercase_laa", "0a:11:22:33:44:55", true},
		{"lowercase_uaa", "00:11:22:33:44:55", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enumerate.ExportIsLocallyAdministeredMAC(tt.mac)
			if result != tt.expected {
				t.Errorf(
					"isLocallyAdministeredMAC(%q) = %v, expected %v",
					tt.mac,
					result,
					tt.expected,
				)
			}
		})
	}
}
