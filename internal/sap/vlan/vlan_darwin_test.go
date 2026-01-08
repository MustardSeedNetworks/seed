//go:build darwin

package vlan_test

import (
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
