//go:build darwin

package vlan_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/krisarmstrong/seed/internal/sap/vlan"
)

// TestDetectVlanSubinterfacesPlatformConcurrent tests concurrent detection.
func TestDetectVlanSubinterfacesPlatformConcurrent(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	numGoroutines := 20
	interfaces := []string{"en0", "en1", "lo0", "bridge0", "awdl0"}

	wg.Add(numGoroutines * len(interfaces))
	for _, iface := range interfaces {
		for range numGoroutines {
			go func(ifName string) {
				defer wg.Done()
				vlans := vlan.ExportDetectVlanSubinterfacesPlatform(ifName)
				if vlans == nil {
					t.Errorf("ExportDetectVlanSubinterfacesPlatform(%q) returned nil", ifName)
				}
			}(iface)
		}
	}

	wg.Wait()
}

// TestGetVlanInfoConcurrent tests concurrent getVlanInfo calls.
func TestGetVlanInfoConcurrent(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	numGoroutines := 20
	interfaces := []string{"en0", "vlan100", "vlan200", "lo0", "bridge0"}

	wg.Add(numGoroutines * len(interfaces))
	for _, iface := range interfaces {
		for range numGoroutines {
			go func(ifName string) {
				defer wg.Done()
				_, vlanID := vlan.ExportGetVlanInfo(ifName)
				// Just ensure no panic or race condition.
				_ = vlanID
			}(iface)
		}
	}

	wg.Wait()
}

// TestGetVlanInfoSystemInterfaces tests with real system interface names.
func TestGetVlanInfoSystemInterfaces(t *testing.T) {
	t.Parallel()

	// Common macOS interface names.
	tests := []struct {
		name        string
		ifname      string
		wantVLANID  int
		checkParent bool
	}{
		{"primary ethernet en0", "en0", 0, false},
		{"secondary ethernet en1", "en1", 0, false},
		{"wifi en2", "en2", 0, false},
		{"thunderbolt en3", "en3", 0, false},
		{"loopback lo0", "lo0", 0, false},
		{"bridge0", "bridge0", 0, false},
		{"bridge100", "bridge100", 0, false},
		{"awdl0 (airdrop)", "awdl0", 0, false},
		{"llw0 (low-latency wlan)", "llw0", 0, false},
		{"utun0 (vpn tunnel)", "utun0", 0, false},
		{"utun1", "utun1", 0, false},
		{"utun2", "utun2", 0, false},
		{"gif0 (generic tunnel)", "gif0", 0, false},
		{"stf0 (6to4 tunnel)", "stf0", 0, false},
		{"ap1 (access point)", "ap1", 0, false},
		{"XHC0 (USB host controller)", "XHC0", 0, false},
		{"XHC1", "XHC1", 0, false},
		{"anpi0", "anpi0", 0, false},
		{"anpi1", "anpi1", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parent, vlanID := vlan.ExportGetVlanInfo(tt.ifname)

			if vlanID != tt.wantVLANID {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVLANID)
			}

			if tt.checkParent && parent == "" {
				t.Errorf("GetVlanInfo(%q) expected non-empty parent", tt.ifname)
			}
		})
	}
}

// TestGetVlanInfoNumericVLANNames tests parsing of vlanN interface names.
func TestGetVlanInfoNumericVLANNames(t *testing.T) {
	t.Parallel()

	// All valid numeric VLAN interface names.
	for i := range 4096 {
		ifname := "vlan" + strings.TrimPrefix(
			"000"+string(rune('0'+i/1000%10)+rune('0'+i/100%10)+rune('0'+i/10%10)+rune('0'+i%10)),
			"000",
		)
		// Use strconv for proper formatting.
		ifname = "vlan" + itoa(i)

		_, vlanID := vlan.ExportGetVlanInfo(ifname)
		if vlanID != i {
			t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", ifname, vlanID, i)
		}
	}
}

// itoa is a simple int to string converter.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

// TestGetVlanInfoPrefixVariations tests various "vlan" prefix variations.
func TestGetVlanInfoPrefixVariations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ifname     string
		wantVLANID int
	}{
		// Standard vlan prefix.
		{"vlan0", 0},
		{"vlan1", 1},
		{"vlan10", 10},
		{"vlan100", 100},
		{"vlan1000", 1000},
		{"vlan4094", 4094},

		// Should not parse as VLAN (different prefix).
		{"VLAN100", 0},     // Uppercase.
		{"Vlan100", 0},     // Mixed case.
		{"vLAN100", 0},     // Mixed case.
		{"vlaan100", 0},    // Typo.
		{"vla100", 0},      // Missing 'n'.
		{"vln100", 0},      // Missing 'a'.
		{"vlan_100", 0},    // Underscore.
		{"vlan-100", -100}, // Hyphen (strconv parses as negative).
		{"vlan.100", 0},    // Dot.
	}

	for _, tt := range tests {
		t.Run(tt.ifname, func(t *testing.T) {
			t.Parallel()

			_, vlanID := vlan.ExportGetVlanInfo(tt.ifname)
			if vlanID != tt.wantVLANID {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVLANID)
			}
		})
	}
}

// TestDetectVlanSubinterfacesPlatformWithRealInterfaces tests with real interfaces.
func TestDetectVlanSubinterfacesPlatformWithRealInterfaces(t *testing.T) {
	t.Parallel()

	// Test with interfaces that typically exist on macOS.
	interfaces := []string{
		"en0",     // Primary ethernet/wifi.
		"en1",     // Secondary network.
		"lo0",     // Loopback.
		"bridge0", // Bridge interface.
		"awdl0",   // Apple Wireless Direct Link.
		"llw0",    // Low-latency WLAN.
		"utun0",   // User tunnel.
		"utun1",
		"utun2",
		"gif0", // Generic tunnel.
		"stf0", // 6to4 tunnel.
		"XHC0", // USB host controller.
		"XHC1",
		"anpi0", // Apple network interface.
		"anpi1",
	}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			t.Parallel()

			vlans := vlan.ExportDetectVlanSubinterfacesPlatform(iface)
			if vlans == nil {
				t.Error("expected non-nil slice")
			}
			// On typical macOS without VLAN configuration, this returns empty.
			// We just verify it doesn't panic.
		})
	}
}

// TestCreateDeleteVlanInterfacePlatformMultiple tests multiple create/delete calls.
func TestCreateDeleteVlanInterfacePlatformMultiple(t *testing.T) {
	t.Parallel()

	// On macOS, these return nil (not fully implemented).
	interfaces := []string{"en0", "en1", "bridge0"}
	vlanIDs := []int{1, 10, 100, 1000, 4094}

	for _, iface := range interfaces {
		for _, vlanID := range vlanIDs {
			t.Run(iface+"_"+itoa(vlanID), func(t *testing.T) {
				t.Parallel()

				err := vlan.ExportCreateVlanInterfacePlatform(iface, vlanID)
				if err != nil {
					t.Errorf("CreateVlanInterfacePlatform(%q, %d) = %v, want nil", iface, vlanID, err)
				}

				err = vlan.ExportDeleteVlanInterfacePlatform(iface, vlanID)
				if err != nil {
					t.Errorf("DeleteVlanInterfacePlatform(%q, %d) = %v, want nil", iface, vlanID, err)
				}
			})
		}
	}
}

// TestGetVlanInfoWithInterfaceNameLengths tests various name lengths.
func TestGetVlanInfoWithInterfaceNameLengths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		ifname     string
		wantVLANID int
	}{
		{"empty", "", 0},
		{"single char", "v", 0},
		{"two chars", "vl", 0},
		{"three chars", "vla", 0},
		{"four chars (vlan)", "vlan", 0},
		{"five chars", "vlan0", 0},
		{"six chars", "vlan10", 10},
		{"seven chars", "vlan100", 100},
		{"eight chars", "vlan1000", 1000},
		{"sixteen chars (IFNAMSIZ limit)", "vlan123456789012", 123456789012}, // Parses as number.
		{"exactly IFNAMSIZ", "vlan12345678901", 12345678901},                 // Parses as number.
		{"longer than IFNAMSIZ", "verylonginterfacenamethatexceedsifnamsizlimit", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, vlanID := vlan.ExportGetVlanInfo(tt.ifname)
			if vlanID != tt.wantVLANID {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVLANID)
			}
		})
	}
}

// TestGetVlanInfoWithBinaryData tests with binary data in interface name.
func TestGetVlanInfoWithBinaryData(t *testing.T) {
	t.Parallel()

	// Test with various binary patterns.
	tests := []struct {
		name   string
		ifname string
	}{
		{"null byte", "vlan\x00100"},
		{"tab", "vlan\t100"},
		{"newline", "vlan\n100"},
		{"carriage return", "vlan\r100"},
		{"bell", "vlan\a100"},
		{"backspace", "vlan\b100"},
		{"form feed", "vlan\f100"},
		{"vertical tab", "vlan\v100"},
		{"high ascii", "vlan\xff100"},
		{"unicode replacement", "vlan\ufffd100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Just ensure no panic.
			_, _ = vlan.ExportGetVlanInfo(tt.ifname)
		})
	}
}

// TestDetectVlanSubinterfacesPlatformEmptyInterface tests with empty interface.
func TestDetectVlanSubinterfacesPlatformEmptyInterface(t *testing.T) {
	t.Parallel()

	vlans := vlan.ExportDetectVlanSubinterfacesPlatform("")
	if vlans == nil {
		t.Error("expected non-nil slice for empty interface")
	}
	if len(vlans) != 0 {
		t.Errorf("expected empty slice for empty interface, got %v", vlans)
	}
}

// TestGetVlanInfoReturnValueConsistency tests return value consistency.
func TestGetVlanInfoReturnValueConsistency(t *testing.T) {
	t.Parallel()

	ifname := "vlan100"

	// Call multiple times and verify consistent results.
	results := make([]struct {
		parent string
		vlanID int
	}, 100)

	for i := range results {
		results[i].parent, results[i].vlanID = vlan.ExportGetVlanInfo(ifname)
	}

	// All results should be identical.
	for i := 1; i < len(results); i++ {
		if results[i].parent != results[0].parent || results[i].vlanID != results[0].vlanID {
			t.Errorf("inconsistent results: call %d got (%q, %d), call 0 got (%q, %d)",
				i, results[i].parent, results[i].vlanID, results[0].parent, results[0].vlanID)
		}
	}
}

// BenchmarkGetVlanInfoVariations benchmarks various interface name patterns.
func BenchmarkGetVlanInfoVariations(b *testing.B) {
	interfaces := []string{"en0", "vlan100", "vlan4094", "lo0", "bridge0"}

	for _, iface := range interfaces {
		b.Run(iface, func(b *testing.B) {
			for b.Loop() {
				_, _ = vlan.ExportGetVlanInfo(iface)
			}
		})
	}
}

// BenchmarkDetectVlanSubinterfacesPlatformVariations benchmarks interface detection.
func BenchmarkDetectVlanSubinterfacesPlatformVariations(b *testing.B) {
	interfaces := []string{"en0", "en1", "lo0", "bridge0", "awdl0"}

	for _, iface := range interfaces {
		b.Run(iface, func(b *testing.B) {
			for b.Loop() {
				_ = vlan.ExportDetectVlanSubinterfacesPlatform(iface)
			}
		})
	}
}

// BenchmarkCreateDeleteVlanInterface benchmarks create/delete operations.
func BenchmarkCreateDeleteVlanInterface(b *testing.B) {
	b.Run("Create", func(b *testing.B) {
		for b.Loop() {
			_ = vlan.ExportCreateVlanInterfacePlatform("en0", 100)
		}
	})

	b.Run("Delete", func(b *testing.B) {
		for b.Loop() {
			_ = vlan.ExportDeleteVlanInterfacePlatform("en0", 100)
		}
	})
}
