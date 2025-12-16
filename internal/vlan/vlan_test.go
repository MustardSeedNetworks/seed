// Package vlan provides VLAN detection and configuration functionality.
// Test suite validates VLAN detection, traffic analysis, and configuration operations.
package vlan

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("eth0")
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	if manager.interfaceName != "eth0" {
		t.Errorf("expected interfaceName 'eth0', got %q", manager.interfaceName)
	}
	if manager.enabled {
		t.Error("expected enabled to be false initially")
	}
	if manager.configuredID != 0 {
		t.Errorf("expected configuredID 0, got %d", manager.configuredID)
	}
}

func TestManagerSetInterface(t *testing.T) {
	manager := NewManager("eth0")

	manager.SetInterface("en0")
	if manager.interfaceName != "en0" {
		t.Errorf("expected interfaceName 'en0', got %q", manager.interfaceName)
	}

	manager.SetInterface("bond0")
	if manager.interfaceName != "bond0" {
		t.Errorf("expected interfaceName 'bond0', got %q", manager.interfaceName)
	}
}

func TestManagerSetConfigured(t *testing.T) {
	manager := NewManager("eth0")

	// Initially disabled
	if manager.enabled {
		t.Error("expected enabled to be false initially")
	}
	if manager.configuredID != 0 {
		t.Errorf("expected configuredID 0, got %d", manager.configuredID)
	}

	// Enable with VLAN 100
	manager.SetConfigured(true, 100)
	if !manager.enabled {
		t.Error("expected enabled to be true")
	}
	if manager.configuredID != 100 {
		t.Errorf("expected configuredID 100, got %d", manager.configuredID)
	}

	// Disable
	manager.SetConfigured(false, 0)
	if manager.enabled {
		t.Error("expected enabled to be false")
	}
}

func TestManagerGetInfo(t *testing.T) {
	manager := NewManager("eth0")

	info := manager.GetInfo()
	if info == nil {
		t.Fatal("expected non-nil info")
	}

	if info.TaggedVlans == nil {
		t.Error("expected non-nil TaggedVlans slice")
	}

	if info.Configured.Enabled {
		t.Error("expected Configured.Enabled to be false initially")
	}
	if info.Configured.ID != 0 {
		t.Errorf("expected Configured.ID 0, got %d", info.Configured.ID)
	}
}

func TestManagerGetInfoWithConfigured(t *testing.T) {
	manager := NewManager("eth0")
	manager.SetConfigured(true, 200)

	info := manager.GetInfo()
	if !info.Configured.Enabled {
		t.Error("expected Configured.Enabled to be true")
	}
	if info.Configured.ID != 200 {
		t.Errorf("expected Configured.ID 200, got %d", info.Configured.ID)
	}
}

func TestManagerGetInfoWithLLDP(t *testing.T) {
	manager := NewManager("eth0")

	nativeVlan := 10
	voiceVlan := 50

	info := manager.GetInfoWithLLDP(&nativeVlan, &voiceVlan)
	if info == nil {
		t.Fatal("expected non-nil info")
	}

	if info.NativeVlan == nil {
		t.Fatal("expected non-nil NativeVlan")
	}
	if *info.NativeVlan != 10 {
		t.Errorf("expected NativeVlan 10, got %d", *info.NativeVlan)
	}
	if info.VoiceVlan == nil {
		t.Fatal("expected non-nil VoiceVlan")
	}
	if *info.VoiceVlan != 50 {
		t.Errorf("expected VoiceVlan 50, got %d", *info.VoiceVlan)
	}
}

func TestManagerGetInfoWithLLDPNilValues(t *testing.T) {
	manager := NewManager("eth0")

	info := manager.GetInfoWithLLDP(nil, nil)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.NativeVlan != nil {
		t.Error("expected nil NativeVlan")
	}
	if info.VoiceVlan != nil {
		t.Error("expected nil VoiceVlan")
	}
}

func TestInfoFields(t *testing.T) {
	nativeVlan := 1
	voiceVlan := 100
	info := Info{
		NativeVlan:  &nativeVlan,
		TaggedVlans: []int{10, 20, 30},
		VoiceVlan:   &voiceVlan,
	}
	info.Configured.Enabled = true
	info.Configured.ID = 50

	if info.NativeVlan == nil || *info.NativeVlan != 1 {
		t.Error("expected NativeVlan 1")
	}
	if len(info.TaggedVlans) != 3 {
		t.Errorf("expected 3 tagged VLANs, got %d", len(info.TaggedVlans))
	}
	if info.TaggedVlans[0] != 10 || info.TaggedVlans[1] != 20 || info.TaggedVlans[2] != 30 {
		t.Errorf("unexpected tagged VLANs: %v", info.TaggedVlans)
	}
	if info.VoiceVlan == nil || *info.VoiceVlan != 100 {
		t.Error("expected VoiceVlan 100")
	}
	if !info.Configured.Enabled {
		t.Error("expected Configured.Enabled true")
	}
	if info.Configured.ID != 50 {
		t.Errorf("expected Configured.ID 50, got %d", info.Configured.ID)
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		val      int
		expected bool
	}{
		{"empty slice", []int{}, 1, false},
		{"not found", []int{1, 2, 3}, 4, false},
		{"found at start", []int{1, 2, 3}, 1, true},
		{"found in middle", []int{1, 2, 3}, 2, true},
		{"found at end", []int{1, 2, 3}, 3, true},
		{"single element found", []int{42}, 42, true},
		{"single element not found", []int{42}, 1, false},
		{"negative numbers", []int{-1, -2, -3}, -2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.val)
			if result != tt.expected {
				t.Errorf("contains(%v, %d) = %v, want %v", tt.slice, tt.val, result, tt.expected)
			}
		})
	}
}

func TestConcurrentManagerAccess(t *testing.T) {
	manager := NewManager("eth0")

	// Test concurrent access doesn't cause race conditions
	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			for j := range 100 {
				manager.SetInterface("eth" + string(rune('0'+id)))
				_ = manager.GetInfo()
				manager.SetConfigured(j%2 == 0, j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}
}

func TestDetectVlanSubinterfacesPlatform(t *testing.T) {
	// This will return empty if no VLANs configured
	vlans := detectVlanSubinterfacesPlatform("eth0")
	if vlans == nil {
		t.Error("expected non-nil slice, even if empty")
	}
}

func TestCreateVlanInterface(t *testing.T) {
	// This requires root privileges, so just verify it doesn't panic
	err := CreateVlanInterface("eth0", 100)
	// Error is expected on non-Linux or without root
	_ = err
}

func TestCreateVlanInterfacePlatform(t *testing.T) {
	// This requires root privileges on Linux
	err := createVlanInterfacePlatform("eth0", 100)
	// Error is expected without root or on macOS (returns nil)
	_ = err
}

func TestDeleteVlanInterface(t *testing.T) {
	// This requires root privileges
	err := DeleteVlanInterface("eth0", 100)
	// Error is expected on non-Linux or without root
	_ = err
}

func TestDetectVlanSubinterfaces(t *testing.T) {
	manager := NewManager("eth0")

	vlans := manager.detectVlanSubinterfaces("eth0")
	if vlans == nil {
		t.Error("expected non-nil slice, even if empty")
	}
}

func TestInfoNilPointers(t *testing.T) {
	info := Info{
		NativeVlan:  nil,
		TaggedVlans: []int{},
		VoiceVlan:   nil,
	}

	if info.NativeVlan != nil {
		t.Error("expected nil NativeVlan")
	}
	if info.VoiceVlan != nil {
		t.Error("expected nil VoiceVlan")
	}
	if len(info.TaggedVlans) != 0 {
		t.Error("expected empty TaggedVlans")
	}
}

func TestInfoConfiguredStruct(t *testing.T) {
	info := Info{}
	info.Configured.Enabled = true
	info.Configured.ID = 150

	if !info.Configured.Enabled {
		t.Error("expected Configured.Enabled true")
	}
	if info.Configured.ID != 150 {
		t.Errorf("expected Configured.ID 150, got %d", info.Configured.ID)
	}
}

func TestContainsEdgeCases(t *testing.T) {
	// Test with large slice
	largeSlice := make([]int, 1000)
	for i := range largeSlice {
		largeSlice[i] = i
	}

	if !contains(largeSlice, 500) {
		t.Error("expected to find 500 in large slice")
	}
	if !contains(largeSlice, 999) {
		t.Error("expected to find 999 in large slice")
	}
	if contains(largeSlice, 1000) {
		t.Error("did not expect to find 1000 in slice 0-999")
	}
}

func TestManagerGetInfoReturnsNewSlice(t *testing.T) {
	manager := NewManager("eth0")

	info1 := manager.GetInfo()
	info2 := manager.GetInfo()

	// Modifying one shouldn't affect the other
	info1.TaggedVlans = append(info1.TaggedVlans, 100)
	if len(info2.TaggedVlans) != 0 {
		t.Error("expected info2 to have empty TaggedVlans")
	}
}

func TestSetConfiguredMultipleTimes(t *testing.T) {
	manager := NewManager("eth0")

	manager.SetConfigured(true, 100)
	manager.SetConfigured(true, 200)
	manager.SetConfigured(false, 0)
	manager.SetConfigured(true, 300)

	if !manager.enabled {
		t.Error("expected enabled true after final SetConfigured")
	}
	if manager.configuredID != 300 {
		t.Errorf("expected configuredID 300, got %d", manager.configuredID)
	}
}
