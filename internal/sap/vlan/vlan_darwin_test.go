//go:build darwin

package vlan_test

import (
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/sap/vlan"
)

func TestGetVlanInfoNonVlanInterface(t *testing.T) {
	// Test with non-VLAN interface names.
	// These should either return empty or parse the name.
	tests := []struct {
		name     string
		ifname   string
		wantVlan int
	}{
		{"regular interface", "en0", 0},
		{"loopback", "lo0", 0},
		{"named vlan100", "vlan100", 100},
		{"named vlan200", "vlan200", 200},
		{"named vlan1", "vlan1", 1},
		{"named vlan4094", "vlan4094", 4094},
		{"invalid vlan name", "vlanABC", 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, vlanID := vlan.ExportGetVlanInfo(tt.ifname)
			if vlanID != tt.wantVlan {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVlan)
			}
		})
	}
}

func TestGetVlanInfoTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		ifname         string
		wantParent     string
		wantVlanID     int
		parentMayEmpty bool // true if parent being empty is acceptable
	}{
		// Non-VLAN interfaces - should return empty.
		{name: "empty interface", ifname: "", wantParent: "", wantVlanID: 0, parentMayEmpty: true},
		{name: "regular en0", ifname: "en0", wantParent: "", wantVlanID: 0, parentMayEmpty: true},
		{name: "loopback lo0", ifname: "lo0", wantParent: "", wantVlanID: 0, parentMayEmpty: true},
		{name: "bridge", ifname: "bridge0", wantParent: "", wantVlanID: 0, parentMayEmpty: true},
		{name: "utun", ifname: "utun0", wantParent: "", wantVlanID: 0, parentMayEmpty: true},
		{name: "awdl", ifname: "awdl0", wantParent: "", wantVlanID: 0, parentMayEmpty: true},

		// VLAN interfaces - parsed from name when ioctl fails.
		{name: "vlan1", ifname: "vlan1", wantParent: "", wantVlanID: 1, parentMayEmpty: true},
		{name: "vlan10", ifname: "vlan10", wantParent: "", wantVlanID: 10, parentMayEmpty: true},
		{name: "vlan100", ifname: "vlan100", wantParent: "", wantVlanID: 100, parentMayEmpty: true},
		{name: "vlan1000", ifname: "vlan1000", wantParent: "", wantVlanID: 1000, parentMayEmpty: true},
		{name: "vlan4094", ifname: "vlan4094", wantParent: "", wantVlanID: 4094, parentMayEmpty: true},
		{name: "vlan4095", ifname: "vlan4095", wantParent: "", wantVlanID: 4095, parentMayEmpty: true},

		// Invalid VLAN names.
		{name: "vlan only", ifname: "vlan", wantParent: "", wantVlanID: 0, parentMayEmpty: true},
		{name: "vlan with letters", ifname: "vlanABC", wantParent: "", wantVlanID: 0, parentMayEmpty: true},
		{name: "vlan with spaces", ifname: "vlan 100", wantParent: "", wantVlanID: 0, parentMayEmpty: true},
		{name: "vlan with dot", ifname: "vlan.100", wantParent: "", wantVlanID: 0, parentMayEmpty: true},
		{name: "vlan mixed", ifname: "vlan100abc", wantParent: "", wantVlanID: 0, parentMayEmpty: true},

		// Edge cases with negative numbers.
		{name: "negative vlan", ifname: "vlan-100", wantParent: "", wantVlanID: -100, parentMayEmpty: true},
		{name: "negative vlan -1", ifname: "vlan-1", wantParent: "", wantVlanID: -1, parentMayEmpty: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, vlanID := vlan.ExportGetVlanInfo(tt.ifname)

			if vlanID != tt.wantVlanID {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVlanID)
			}

			if !tt.parentMayEmpty && parent != tt.wantParent {
				t.Errorf("GetVlanInfo(%q) parent = %q, want %q", tt.ifname, parent, tt.wantParent)
			}
		})
	}
}

func TestGetVlanInfoWithVlanPrefix(t *testing.T) {
	// Test specifically the fallback parsing path when ioctl fails.
	tests := []struct {
		ifname       string
		wantVlanID   int
		wantParentOK bool
	}{
		{"vlan100", 100, false},
		{"vlan1", 1, false},
		{"vlan4094", 4094, false},
		{"vlan0", 0, false}, // 0 is parsed but returned as 0
		{"vlan", 0, false},  // No number after prefix
		{"vlanxyz", 0, false},
		{"vlan-100", -100, false}, // strconv.Atoi parses "-100" as -100
	}

	for _, tt := range tests {
		t.Run(tt.ifname, func(t *testing.T) {
			parent, vlanID := vlan.ExportGetVlanInfo(tt.ifname)
			if vlanID != tt.wantVlanID {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVlanID)
			}
			// Parent is typically empty when ioctl fails.
			if tt.wantParentOK && parent == "" {
				t.Errorf("GetVlanInfo(%q) expected non-empty parent", tt.ifname)
			}
		})
	}
}

func TestDetectVlanSubinterfacesDarwin(t *testing.T) {
	// Test with various interface names.
	// On a typical macOS system without VLAN interfaces, this returns empty.
	interfaces := []string{"en0", "en1", "lo0", "bridge0", "awdl0", "utun0"}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			vlans := vlan.ExportDetectVlanSubinterfacesPlatform(iface)
			if vlans == nil {
				t.Error("expected non-nil slice even if empty")
			}
		})
	}
}

func TestCreateVlanInterfaceDarwin(t *testing.T) {
	// On macOS, this currently returns nil (not implemented).
	err := vlan.ExportCreateVlanInterfacePlatform("en0", 100)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestDeleteVlanInterfaceDarwin(t *testing.T) {
	// On macOS, this currently returns nil (not implemented).
	err := vlan.ExportDeleteVlanInterfacePlatform("en0", 100)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestGetVlanInfoEdgeCases(t *testing.T) {
	// Test edge cases.
	tests := []struct {
		name   string
		ifname string
	}{
		{"long interface name", "verylonginterfacenamethatexceedsnormallength"},
		{"unicode chars", "vlan\u0000"},
		{"special chars", "vlan/100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Just ensure it doesn't panic.
			_, _ = vlan.ExportGetVlanInfo(tt.ifname)
		})
	}
}

func TestDetectVlanSubinterfacesPlatformTableDriven(t *testing.T) {
	tests := []struct {
		name  string
		iface string
	}{
		{"en0 - common ethernet", "en0"},
		{"en1 - secondary ethernet", "en1"},
		{"en2 - tertiary ethernet", "en2"},
		{"lo0 - loopback", "lo0"},
		{"bridge0 - bridge interface", "bridge0"},
		{"bridge100 - numbered bridge", "bridge100"},
		{"awdl0 - Apple Wireless Direct Link", "awdl0"},
		{"utun0 - user tunnel", "utun0"},
		{"utun1 - second user tunnel", "utun1"},
		{"llw0 - low-latency WLAN", "llw0"},
		{"gif0 - gif interface", "gif0"},
		{"stf0 - 6to4 tunnel", "stf0"},
		{"ap1 - access point", "ap1"},
		{"XHC0 - USB host controller", "XHC0"},
		{"anpi0 - Apple network interface", "anpi0"},
		{"empty interface", ""},
		{"nonexistent interface", "nonexistent0"},
		{"interface with dot", "en0.100"},
		{"interface with hyphen", "en-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vlans := vlan.ExportDetectVlanSubinterfacesPlatform(tt.iface)
			if vlans == nil {
				t.Error("expected non-nil slice even if empty")
			}
			// On a typical macOS system without VLAN interfaces configured,
			// this should return an empty slice.
		})
	}
}

func TestCreateDeleteVlanInterfaceTableDriven(t *testing.T) {
	tests := []struct {
		name       string
		parentIf   string
		vlanID     int
		wantCreate error
		wantDelete error
	}{
		{"typical VLAN 100 on en0", "en0", 100, nil, nil},
		{"VLAN 1 on en0", "en0", 1, nil, nil},
		{"max VLAN 4094 on en0", "en0", 4094, nil, nil},
		{"VLAN on en1", "en1", 200, nil, nil},
		{"VLAN 0", "en0", 0, nil, nil},
		{"empty parent interface", "", 100, nil, nil},
		{"VLAN on nonexistent", "nonexistent0", 100, nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// On macOS, these return nil (not implemented).
			err := vlan.ExportCreateVlanInterfacePlatform(tt.parentIf, tt.vlanID)
			if !errors.Is(err, tt.wantCreate) {
				t.Errorf("CreateVlanInterfacePlatform() error = %v, want %v", err, tt.wantCreate)
			}

			err = vlan.ExportDeleteVlanInterfacePlatform(tt.parentIf, tt.vlanID)
			if !errors.Is(err, tt.wantDelete) {
				t.Errorf("DeleteVlanInterfacePlatform() error = %v, want %v", err, tt.wantDelete)
			}
		})
	}
}

func TestGetVlanInfoParentExtraction(t *testing.T) {
	// Test the parent extraction logic by testing with various interface name patterns.
	// Since we can't easily simulate a real VLAN interface, we test the fallback behavior.
	tests := []struct {
		name       string
		ifname     string
		wantVlanID int
	}{
		// These should all trigger the fallback path (ioctl fails).
		{"vlan0", "vlan0", 0},
		{"vlan1", "vlan1", 1},
		{"vlan10", "vlan10", 10},
		{"vlan99", "vlan99", 99},
		{"vlan100", "vlan100", 100},
		{"vlan999", "vlan999", 999},
		{"vlan4094", "vlan4094", 4094},
		{"vlan4095", "vlan4095", 4095},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, vlanID := vlan.ExportGetVlanInfo(tt.ifname)
			if vlanID != tt.wantVlanID {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVlanID)
			}
		})
	}
}

func TestGetVlanInfoParentStringExtraction(t *testing.T) {
	// Test with interface names that don't start with "vlan".
	// These should return (empty, 0) since ioctl will fail and
	// the fallback parsing only works for "vlan" prefix.
	tests := []struct {
		ifname     string
		wantVlanID int
	}{
		{"", 0},
		{"en0", 0},
		{"eth0", 0},
		{"bond0", 0},
		{"br0", 0},
		{"docker0", 0},
		{"virbr0", 0},
		{"wlan0", 0},
		{"wlp3s0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.ifname, func(t *testing.T) {
			parent, vlanID := vlan.ExportGetVlanInfo(tt.ifname)
			if vlanID != tt.wantVlanID {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVlanID)
			}
			// Parent should be empty for non-VLAN interfaces.
			if parent != "" {
				t.Errorf("GetVlanInfo(%q) parent = %q, want empty", tt.ifname, parent)
			}
		})
	}
}

func TestGetVlanInfoLargeVlanNumbers(t *testing.T) {
	// Test parsing of large VLAN numbers (beyond 4094).
	tests := []struct {
		ifname     string
		wantVlanID int
	}{
		{"vlan5000", 5000},
		{"vlan10000", 10000},
		{"vlan65535", 65535},
		{"vlan100000", 100000},
		{"vlan999999", 999999},
	}

	for _, tt := range tests {
		t.Run(tt.ifname, func(t *testing.T) {
			_, vlanID := vlan.ExportGetVlanInfo(tt.ifname)
			if vlanID != tt.wantVlanID {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVlanID)
			}
		})
	}
}

func TestGetVlanInfoSpecialCharacters(t *testing.T) {
	// Test with special characters that might cause parsing issues.
	// Note: strconv.Atoi can parse "+100" as 100, so we expect 100 for that case.
	tests := []struct {
		name       string
		ifname     string
		wantVlanID int
	}{
		{"null byte in middle", "vlan\x00100", 0},
		{"tab character", "vlan\t100", 0},
		{"newline", "vlan\n100", 0},
		{"carriage return", "vlan\r100", 0},
		{"space", "vlan 100", 0},
		{"plus sign", "vlan+100", 100}, // strconv.Atoi parses "+100" as 100
		{"equals sign", "vlan=100", 0},
		{"at sign", "vlan@100", 0},
		{"hash", "vlan#100", 0},
		{"percent", "vlan%100", 0},
		{"caret", "vlan^100", 0},
		{"ampersand", "vlan&100", 0},
		{"asterisk", "vlan*100", 0},
		{"parentheses open", "vlan(100", 0},
		{"parentheses close", "vlan)100", 0},
		{"brackets", "vlan[100]", 0},
		{"braces", "vlan{100}", 0},
		{"pipe", "vlan|100", 0},
		{"backslash", "vlan\\100", 0},
		{"colon", "vlan:100", 0},
		{"semicolon", "vlan;100", 0},
		{"quote single", "vlan'100", 0},
		{"quote double", "vlan\"100", 0},
		{"less than", "vlan<100", 0},
		{"greater than", "vlan>100", 0},
		{"comma", "vlan,100", 0},
		{"question mark", "vlan?100", 0},
		{"exclamation", "vlan!100", 0},
		{"tilde", "vlan~100", 0},
		{"backtick", "vlan`100", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, vlanID := vlan.ExportGetVlanInfo(tt.ifname)
			if vlanID != tt.wantVlanID {
				t.Errorf("GetVlanInfo(%q) vlanID = %d, want %d", tt.ifname, vlanID, tt.wantVlanID)
			}
		})
	}
}

func BenchmarkGetVlanInfo(b *testing.B) {
	b.Run("non-vlan interface", func(b *testing.B) {
		for b.Loop() {
			_, _ = vlan.ExportGetVlanInfo("en0")
		}
	})

	b.Run("vlan interface", func(b *testing.B) {
		for b.Loop() {
			_, _ = vlan.ExportGetVlanInfo("vlan100")
		}
	})
}

func BenchmarkDetectVlanSubinterfacesPlatform(b *testing.B) {
	for b.Loop() {
		_ = vlan.ExportDetectVlanSubinterfacesPlatform("en0")
	}
}
