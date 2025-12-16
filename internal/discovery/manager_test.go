// Package discovery provides manager tests.
package discovery

import (
	"testing"
	"time"
)

func TestProtocolConstants(t *testing.T) {
	if ProtocolLLDP != "lldp" {
		t.Errorf("Expected ProtocolLLDP='lldp', got %q", ProtocolLLDP)
	}
	if ProtocolCDP != "cdp" {
		t.Errorf("Expected ProtocolCDP='cdp', got %q", ProtocolCDP)
	}
	if ProtocolEDP != "edp" {
		t.Errorf("Expected ProtocolEDP='edp', got %q", ProtocolEDP)
	}
}

func TestNeighbor_Fields(t *testing.T) {
	n := Neighbor{
		Protocol:          ProtocolLLDP,
		ChassisID:         "00:11:22:33:44:55",
		PortID:            "Gi0/1",
		PortDescription:   "Uplink to Core",
		SystemName:        "switch01",
		SystemDescription: "Cisco IOS",
		Capabilities:      []string{"Bridge", "Router"},
		ManagementAddress: "192.168.1.1",
		VLAN:              100,
		NativeVLAN:        1,
		VoiceVLAN:         200,
		TTL:               120,
		LastSeen:          time.Now(),
		SourceMAC:         "AA:BB:CC:DD:EE:FF",
	}

	if n.Protocol != ProtocolLLDP {
		t.Errorf("Expected Protocol LLDP, got %v", n.Protocol)
	}
	if n.ChassisID != "00:11:22:33:44:55" {
		t.Errorf("Expected ChassisID '00:11:22:33:44:55', got %q", n.ChassisID)
	}
	if n.PortID != "Gi0/1" {
		t.Errorf("Expected PortID 'Gi0/1', got %q", n.PortID)
	}
	if n.PortDescription != "Uplink to Core" {
		t.Errorf("Expected PortDescription 'Uplink to Core', got %q", n.PortDescription)
	}
	if n.SystemName != "switch01" {
		t.Errorf("Expected SystemName 'switch01', got %q", n.SystemName)
	}
	if n.SystemDescription != "Cisco IOS" {
		t.Errorf("Expected SystemDescription 'Cisco IOS', got %q", n.SystemDescription)
	}
	if len(n.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(n.Capabilities))
	}
	if n.ManagementAddress != "192.168.1.1" {
		t.Errorf("Expected ManagementAddress '192.168.1.1', got %q", n.ManagementAddress)
	}
	if n.VLAN != 100 {
		t.Errorf("Expected VLAN 100, got %d", n.VLAN)
	}
	if n.NativeVLAN != 1 {
		t.Errorf("Expected NativeVLAN 1, got %d", n.NativeVLAN)
	}
	if n.VoiceVLAN != 200 {
		t.Errorf("Expected VoiceVLAN 200, got %d", n.VoiceVLAN)
	}
	if n.TTL != 120 {
		t.Errorf("Expected TTL 120, got %d", n.TTL)
	}
	if n.LastSeen.IsZero() {
		t.Error("Expected LastSeen to be set")
	}
	if n.SourceMAC != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("Expected SourceMAC 'AA:BB:CC:DD:EE:FF', got %q", n.SourceMAC)
	}
}

func TestNewManager(t *testing.T) {
	mgr := NewManager("lo")

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.IsRunning() {
		t.Error("Manager should not be running initially")
	}
}

func TestManager_IsRunning(t *testing.T) {
	mgr := NewManager("lo")

	if mgr.IsRunning() {
		t.Error("Manager should not be running initially")
	}
}

func TestManager_GetNeighbors_Empty(t *testing.T) {
	mgr := NewManager("lo")

	neighbors := mgr.GetNeighbors()
	if len(neighbors) != 0 {
		t.Errorf("Expected 0 neighbors initially, got %d", len(neighbors))
	}
}

func TestManager_GetLLDPNeighbors_Empty(t *testing.T) {
	mgr := NewManager("lo")

	neighbors := mgr.GetLLDPNeighbors()
	if len(neighbors) != 0 {
		t.Errorf("Expected 0 LLDP neighbors initially, got %d", len(neighbors))
	}
}

func TestManager_GetCDPNeighbors_Empty(t *testing.T) {
	mgr := NewManager("lo")

	neighbors := mgr.GetCDPNeighbors()
	if len(neighbors) != 0 {
		t.Errorf("Expected 0 CDP neighbors initially, got %d", len(neighbors))
	}
}

func TestManager_GetEDPNeighbors_Empty(t *testing.T) {
	mgr := NewManager("lo")

	neighbors := mgr.GetEDPNeighbors()
	if len(neighbors) != 0 {
		t.Errorf("Expected 0 EDP neighbors initially, got %d", len(neighbors))
	}
}

func TestManager_Stop_WhenNotRunning(t *testing.T) {
	mgr := NewManager("lo")

	// Stop when not running should be safe (no-op)
	mgr.Stop()

	if mgr.IsRunning() {
		t.Error("Manager should still not be running after Stop")
	}
}
