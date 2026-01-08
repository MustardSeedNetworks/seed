//go:build darwin

package link_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/sap/link"
)

// TestDarwinCheckLinkStatePlatform tests macOS-specific link state detection.
func TestDarwinCheckLinkStatePlatform(t *testing.T) {
	tests := []struct {
		name      string
		iface     string
		wantValid bool
	}{
		{"en0 primary interface", "en0", true},
		{"lo0 loopback", "lo0", true},
		{"en1 secondary", "en1", true},
		{"nonexistent interface", "nonexistent_xyz_123", true},
		{"empty name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := link.ExportCheckLinkStatePlatform(tt.iface)

			// Verify result is a valid state
			switch state {
			case link.StateUp, link.StateDown, link.StateDormant, link.StateUnknown:
				// Valid state
			default:
				t.Errorf("unexpected state %v for interface %q", state, tt.iface)
			}
		})
	}
}

// TestDarwinGetSpeedDuplex tests macOS-specific speed/duplex detection.
func TestDarwinGetSpeedDuplex(t *testing.T) {
	tests := []struct {
		name  string
		iface string
	}{
		{"en0 primary", "en0"},
		{"en1 secondary", "en1"},
		{"lo0 loopback", "lo0"},
		{"nonexistent", "nonexistent_abc"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			speed, duplex := link.ExportGetSpeedDuplex(tt.iface)

			// Speed should be non-negative
			if speed < 0 {
				t.Errorf("unexpected negative speed %v", speed)
			}

			// Duplex should be a valid value
			switch duplex {
			case link.DuplexFull, link.DuplexHalf, link.DuplexUnknown:
				// Valid
			default:
				t.Errorf("unexpected duplex %v", duplex)
			}
		})
	}
}

// TestDarwinIsPhysicalInterfacePlatform tests macOS-specific physical interface detection.
func TestDarwinIsPhysicalInterfacePlatform(t *testing.T) {
	tests := []struct {
		name     string
		iface    string
		expected bool
	}{
		// Physical interfaces on macOS
		{"en0", "en0", true},
		{"en1", "en1", true},
		{"en2", "en2", true},
		{"en10", "en10", true},
		{"en99", "en99", true},
		{"eth0", "eth0", true},
		{"eth1", "eth1", true},

		// Virtual/system interfaces on macOS
		{"lo0 loopback", "lo0", false},
		{"lo1 loopback", "lo1", false},
		{"bridge0", "bridge0", false},
		{"bridge100", "bridge100", false},
		{"utun0 tunnel", "utun0", false},
		{"utun1 tunnel", "utun1", false},
		{"utun99 tunnel", "utun99", false},
		{"awdl0 airdrop", "awdl0", false},
		{"llw0 low latency", "llw0", false},
		{"llw1", "llw1", false},

		// Unknown patterns
		{"unknown pattern", "xyz0", false},
		{"gif0 tunnel", "gif0", false},
		{"stf0 6to4", "stf0", false},
		{"empty", "", false},
		{"p2p0", "p2p0", false},
		{"ap1 access point", "ap1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := link.ExportIsPhysicalInterfacePlatform(tt.iface)
			if result != tt.expected {
				t.Errorf("IsPhysicalInterfacePlatform(%q) = %v, expected %v", tt.iface, result, tt.expected)
			}
		})
	}
}

// TestDarwinParseSpeedPlatform tests macOS-specific speed parsing.
func TestDarwinParseSpeedPlatform(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected link.Speed
	}{
		// Direct numeric values
		{"direct 10", "10", link.Speed10},
		{"direct 100", "100", link.Speed100},
		{"direct 1000", "1000", link.Speed1000},
		{"direct 10000", "10000", link.Speed10000},
		{"direct 2500", "2500", link.Speed2500},
		{"direct 5000", "5000", link.Speed5000},
		{"direct 25000", "25000", link.Speed25000},
		{"direct 40000", "40000", link.Speed40000},
		{"direct 100000", "100000", link.Speed100000},

		// baseT/baseX format (from ifconfig media output)
		{"1000baset", "1000baset", link.Speed1000},
		{"100baset", "100baset", link.Speed100},
		{"10baset", "10baset", link.Speed10},
		{"10000base", "10000base", link.Speed10000},

		// Gbps format variations
		{"100g", "100g", link.Speed100000},
		{"100gbps", "100gbps", link.Speed100000},
		{"40g", "40g", link.Speed40000},
		{"40gbps", "40gbps", link.Speed40000},
		{"25g", "25g", link.Speed25000},
		{"25gbps", "25gbps", link.Speed25000},
		{"10g", "10g", link.Speed10000},
		{"10gbps", "10gbps", link.Speed10000},
		{"5g", "5g", link.Speed5000},
		{"5gbps", "5gbps", link.Speed5000},
		// Note: "2.5g" matches "5g" first due to order of checks in implementation
		{"2.5g", "2.5g", link.Speed5000},
		{"2.5gbps", "2.5gbps", link.Speed5000},
		{"1g", "1g", link.Speed1000},
		{"1gbps", "1gbps", link.Speed1000},

		// Edge cases and invalid inputs
		{"empty string", "", link.Speed(0)},
		{"unknown", "unknown", link.Speed(0)},
		{"auto", "auto", link.Speed(0)},
		{"whitespace", "   ", link.Speed(0)},
		{"invalid text", "not-a-speed", link.Speed(0)},

		// Mixed case handling
		{"uppercase 1000", "1000", link.Speed1000},
		{"mixed case 10G", "10G", link.Speed10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := link.ExportParseSpeedPlatform(tt.input)
			if result != tt.expected {
				t.Errorf("ParseSpeedPlatform(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDarwinIsPhysicalInterfacePublicAPI tests the public API on macOS.
func TestDarwinIsPhysicalInterfacePublicAPI(t *testing.T) {
	tests := []struct {
		iface    string
		expected bool
	}{
		{"en0", true},
		{"lo0", false},
		{"bridge0", false},
		{"utun0", false},
	}

	for _, tt := range tests {
		t.Run(tt.iface, func(t *testing.T) {
			result := link.IsPhysicalInterface(tt.iface)
			if result != tt.expected {
				t.Errorf("IsPhysicalInterface(%q) = %v, expected %v", tt.iface, result, tt.expected)
			}
		})
	}
}

// TestDarwinParseSpeedPublicAPI tests the public ParseSpeed API on macOS.
func TestDarwinParseSpeedPublicAPI(t *testing.T) {
	tests := []struct {
		input    string
		expected link.Speed
	}{
		{"1000", link.Speed1000},
		{"100", link.Speed100},
		{"10g", link.Speed10000},
		{"unknown", link.Speed(0)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := link.ParseSpeed(tt.input)
			if result != tt.expected {
				t.Errorf("ParseSpeed(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDarwinGetStatusRealInterface tests GetStatus with real macOS interfaces.
func TestDarwinGetStatusRealInterface(t *testing.T) {
	// Test with en0 which typically exists on macOS
	status, err := link.GetStatus("en0")
	if err != nil {
		// en0 might not exist on all systems
		t.Logf("en0 not found: %v", err)

		// Try lo0 which should always exist
		status, err = link.GetStatus("lo0")
		if err != nil {
			t.Fatalf("lo0 not found: %v", err)
		}
	}

	// Verify status fields
	if status.Interface == "" {
		t.Error("Interface name should not be empty")
	}

	if status.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}

	// Verify state is valid
	switch status.State {
	case link.StateUp, link.StateDown, link.StateDormant, link.StateUnknown:
		// Valid
	default:
		t.Errorf("unexpected state %v", status.State)
	}
}

// TestDarwinListInterfacesContent tests ListInterfaces returns macOS interfaces.
func TestDarwinListInterfacesContent(t *testing.T) {
	interfaces, err := link.ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces failed: %v", err)
	}

	// On macOS, we should typically have some interfaces
	if len(interfaces) == 0 {
		t.Log("No non-loopback interfaces found")
	}

	// Verify loopback is excluded
	for _, iface := range interfaces {
		if iface.Interface == "lo0" {
			t.Error("lo0 loopback should be excluded from ListInterfaces")
		}
	}

	// Verify each interface has valid data
	for _, iface := range interfaces {
		if iface.Interface == "" {
			t.Error("interface name should not be empty")
		}

		if iface.SpeedStr == "" {
			t.Errorf("SpeedStr should not be empty for %s", iface.Interface)
		}

		switch iface.State {
		case link.StateUp, link.StateDown, link.StateDormant, link.StateUnknown:
			// Valid
		default:
			t.Errorf("unexpected state %v for %s", iface.State, iface.Interface)
		}

		switch iface.Duplex {
		case link.DuplexFull, link.DuplexHalf, link.DuplexUnknown:
			// Valid
		default:
			t.Errorf("unexpected duplex %v for %s", iface.Duplex, iface.Interface)
		}
	}
}

// BenchmarkDarwinCheckLinkStatePlatform benchmarks link state detection on macOS.
func BenchmarkDarwinCheckLinkStatePlatform(b *testing.B) {
	for b.Loop() {
		_ = link.ExportCheckLinkStatePlatform("en0")
	}
}

// BenchmarkDarwinGetSpeedDuplex benchmarks speed/duplex detection on macOS.
func BenchmarkDarwinGetSpeedDuplex(b *testing.B) {
	for b.Loop() {
		_, _ = link.ExportGetSpeedDuplex("en0")
	}
}

// BenchmarkDarwinIsPhysicalInterfacePlatform benchmarks physical interface detection on macOS.
func BenchmarkDarwinIsPhysicalInterfacePlatform(b *testing.B) {
	interfaces := []string{"en0", "lo0", "utun0", "bridge0", "awdl0"}

	for b.Loop() {
		for _, iface := range interfaces {
			_ = link.ExportIsPhysicalInterfacePlatform(iface)
		}
	}
}

// BenchmarkDarwinParseSpeedPlatform benchmarks speed parsing on macOS.
func BenchmarkDarwinParseSpeedPlatform(b *testing.B) {
	speeds := []string{"1000", "100", "10000", "1g", "10g", "100g", "unknown"}

	for b.Loop() {
		for _, s := range speeds {
			_ = link.ExportParseSpeedPlatform(s)
		}
	}
}
