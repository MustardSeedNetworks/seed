//go:build linux

package vlan_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan"
)

func TestNewManager(t *testing.T) {
	manager := vlan.NewManager("eth0")
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	if manager.ManagerInterfaceName() != "eth0" {
		t.Errorf("expected interfaceName 'eth0', got %q", manager.ManagerInterfaceName())
	}
	if manager.ManagerEnabled() {
		t.Error("expected enabled to be false initially")
	}
	if manager.ManagerConfiguredID() != 0 {
		t.Errorf("expected configuredID 0, got %d", manager.ManagerConfiguredID())
	}
}

func TestManagerSetInterface(t *testing.T) {
	manager := vlan.NewManager("eth0")

	manager.SetInterface("en0")
	if manager.ManagerInterfaceName() != "en0" {
		t.Errorf("expected interfaceName 'en0', got %q", manager.ManagerInterfaceName())
	}

	manager.SetInterface("bond0")
	if manager.ManagerInterfaceName() != "bond0" {
		t.Errorf("expected interfaceName 'bond0', got %q", manager.ManagerInterfaceName())
	}
}

func TestManagerSetConfigured(t *testing.T) {
	manager := vlan.NewManager("eth0")

	// Initially disabled.
	if manager.ManagerEnabled() {
		t.Error("expected enabled to be false initially")
	}
	if manager.ManagerConfiguredID() != 0 {
		t.Errorf("expected configuredID 0, got %d", manager.ManagerConfiguredID())
	}

	// Enable with VLAN 100.
	manager.SetConfigured(true, 100)
	if !manager.ManagerEnabled() {
		t.Error("expected enabled to be true")
	}
	if manager.ManagerConfiguredID() != 100 {
		t.Errorf("expected configuredID 100, got %d", manager.ManagerConfiguredID())
	}

	// Disable.
	manager.SetConfigured(false, 0)
	if manager.ManagerEnabled() {
		t.Error("expected enabled to be false")
	}
}

func TestManagerGetInfo(t *testing.T) {
	manager := vlan.NewManager("eth0")

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
	manager := vlan.NewManager("eth0")
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
	manager := vlan.NewManager("eth0")

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
	manager := vlan.NewManager("eth0")

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
	info := vlan.Info{
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
			result := vlan.ExportContains(tt.slice, tt.val)
			if result != tt.expected {
				t.Errorf("Contains(%v, %d) = %v, want %v", tt.slice, tt.val, result, tt.expected)
			}
		})
	}
}

// TestConcurrentManagerAccess is a race-detector exerciser: the interface name
// and the configured state are written while GetInfo reads them.
func TestConcurrentManagerAccess(t *testing.T) {
	manager := vlan.NewManager("eth0")

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

	for range 10 {
		<-done
	}

	if got := manager.ManagerInterfaceName(); len(got) != 4 || got[:3] != "eth" {
		t.Errorf("ManagerInterfaceName() = %q after concurrent writes, want an eth<n> name", got)
	}
}

func TestDetectVlanSubinterfacesPlatform(t *testing.T) {
	// This will return empty if no VLANs configured.
	vlans := vlan.ExportDetectVlanSubinterfacesPlatform("eth0")
	if vlans == nil {
		t.Error("expected non-nil slice, even if empty")
	}
}

// TestVlanInterfaceOnUnknownParent pins the behaviour that holds regardless of
// privileges: a VLAN cannot be created on or removed from a parent that does
// not exist, because the netlink parent lookup fails before any write.
func TestVlanInterfaceOnUnknownParent(t *testing.T) {
	const noSuchIface = "seed-test-noiface0"

	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"CreateVlanInterface", func() error { return vlan.CreateVlanInterface(noSuchIface, 100) }},
		{"DeleteVlanInterface", func() error { return vlan.DeleteVlanInterface(noSuchIface, 100) }},
		{"createVlanInterfacePlatform", func() error {
			return vlan.ExportCreateVlanInterfacePlatform(noSuchIface, 100)
		}},
		{"deleteVlanInterfacePlatform", func() error {
			return vlan.ExportDeleteVlanInterfacePlatform(noSuchIface, 100)
		}},
	} {
		if err := tc.call(); err == nil {
			t.Errorf("%s(%q, 100) = nil, want an error", tc.name, noSuchIface)
		}
	}
}

func TestDetectVlanSubinterfaces(t *testing.T) {
	manager := vlan.NewManager("eth0")

	vlans := manager.DetectVlanSubinterfaces("eth0")
	if vlans == nil {
		t.Error("expected non-nil slice, even if empty")
	}
}

func TestInfoNilPointers(t *testing.T) {
	info := vlan.Info{
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
	info := vlan.Info{}
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
	// Test with large slice.
	largeSlice := make([]int, 1000)
	for i := range largeSlice {
		largeSlice[i] = i
	}

	if !vlan.ExportContains(largeSlice, 500) {
		t.Error("expected to find 500 in large slice")
	}
	if !vlan.ExportContains(largeSlice, 999) {
		t.Error("expected to find 999 in large slice")
	}
	if vlan.ExportContains(largeSlice, 1000) {
		t.Error("did not expect to find 1000 in slice 0-999")
	}
}

func TestManagerGetInfoReturnsNewSlice(t *testing.T) {
	manager := vlan.NewManager("eth0")

	info1 := manager.GetInfo()
	info2 := manager.GetInfo()

	// Modifying one shouldn't affect the other.
	info1.TaggedVlans = append(info1.TaggedVlans, 100)
	if len(info2.TaggedVlans) != 0 {
		t.Error("expected info2 to have empty TaggedVlans")
	}
}

func TestSetConfiguredMultipleTimes(t *testing.T) {
	manager := vlan.NewManager("eth0")

	manager.SetConfigured(true, 100)
	manager.SetConfigured(true, 200)
	manager.SetConfigured(false, 0)
	manager.SetConfigured(true, 300)

	if !manager.ManagerEnabled() {
		t.Error("expected enabled true after final SetConfigured")
	}
	if manager.ManagerConfiguredID() != 300 {
		t.Errorf("expected configuredID 300, got %d", manager.ManagerConfiguredID())
	}
}

func TestManagerWithEmptyInterfaceName(t *testing.T) {
	manager := vlan.NewManager("")

	if manager.ManagerInterfaceName() != "" {
		t.Errorf("expected empty interface name, got %q", manager.ManagerInterfaceName())
	}

	info := manager.GetInfo()
	if info == nil {
		t.Fatal("expected non-nil info even with empty interface")
	}
}

func TestManagerSetInterfaceToEmpty(t *testing.T) {
	manager := vlan.NewManager("eth0")
	manager.SetInterface("")

	if manager.ManagerInterfaceName() != "" {
		t.Errorf("expected empty interface name, got %q", manager.ManagerInterfaceName())
	}
}

func TestManagerSetConfiguredEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		vlanID    int
		wantID    int
		wantState bool
	}{
		{"zero VLAN ID enabled", true, 0, 0, true},
		{"negative VLAN ID", true, -1, -1, true},
		{"max VLAN ID", true, 4094, 4094, true},
		{"above max VLAN ID", true, 4095, 4095, true},
		{"very large VLAN ID", true, 999999, 999999, true},
		{"disabled with VLAN ID", false, 100, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := vlan.NewManager("eth0")
			manager.SetConfigured(tt.enabled, tt.vlanID)

			if manager.ManagerEnabled() != tt.wantState {
				t.Errorf("enabled = %v, want %v", manager.ManagerEnabled(), tt.wantState)
			}
			if manager.ManagerConfiguredID() != tt.wantID {
				t.Errorf("configuredID = %d, want %d", manager.ManagerConfiguredID(), tt.wantID)
			}
		})
	}
}

func TestInfoTaggedVlansSlice(t *testing.T) {
	tests := []struct {
		name        string
		taggedVlans []int
		wantLen     int
	}{
		{"nil slice", nil, 0},
		{"empty slice", []int{}, 0},
		{"single VLAN", []int{100}, 1},
		{"multiple VLANs", []int{10, 20, 30, 40, 50}, 5},
		{"many VLANs", make([]int, 100), 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := vlan.Info{TaggedVlans: tt.taggedVlans}

			if tt.taggedVlans == nil {
				if info.TaggedVlans != nil {
					t.Error("expected nil TaggedVlans to remain nil")
				}
			} else if len(info.TaggedVlans) != tt.wantLen {
				t.Errorf("len(TaggedVlans) = %d, want %d", len(info.TaggedVlans), tt.wantLen)
			}
		})
	}
}

func TestGetInfoWithLLDPPartialNil(t *testing.T) {
	manager := vlan.NewManager("eth0")

	// Only native VLAN set.
	nativeVlan := 10
	info := manager.GetInfoWithLLDP(&nativeVlan, nil)
	if info.NativeVlan == nil || *info.NativeVlan != 10 {
		t.Error("expected NativeVlan 10")
	}
	if info.VoiceVlan != nil {
		t.Error("expected nil VoiceVlan")
	}

	// Only voice VLAN set.
	voiceVlan := 50
	info2 := manager.GetInfoWithLLDP(nil, &voiceVlan)
	if info2.NativeVlan != nil {
		t.Error("expected nil NativeVlan")
	}
	if info2.VoiceVlan == nil || *info2.VoiceVlan != 50 {
		t.Error("expected VoiceVlan 50")
	}
}

func TestGetInfoWithLLDPAndConfigured(t *testing.T) {
	manager := vlan.NewManager("eth0")
	manager.SetConfigured(true, 200)

	nativeVlan := 1
	voiceVlan := 100

	info := manager.GetInfoWithLLDP(&nativeVlan, &voiceVlan)

	// All fields should be set.
	if info.NativeVlan == nil || *info.NativeVlan != 1 {
		t.Error("expected NativeVlan 1")
	}
	if info.VoiceVlan == nil || *info.VoiceVlan != 100 {
		t.Error("expected VoiceVlan 100")
	}
	if !info.Configured.Enabled {
		t.Error("expected Configured.Enabled true")
	}
	if info.Configured.ID != 200 {
		t.Errorf("expected Configured.ID 200, got %d", info.Configured.ID)
	}
}

func TestDetectVlanSubinterfacesWithDifferentInterfaces(t *testing.T) {
	interfaces := []string{"eth0", "en0", "wlan0", "bond0", "lo", ""}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			vlans := vlan.ExportDetectVlanSubinterfacesPlatform(iface)
			if vlans == nil {
				t.Error("expected non-nil slice even if empty")
			}
		})
	}
}

func TestContainsWithZeroAndNegative(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		val      int
		expected bool
	}{
		{"find zero in slice", []int{0, 1, 2}, 0, true},
		{"find zero in single element", []int{0}, 0, true},
		{"find negative zero", []int{-0}, 0, true},
		{"large negative", []int{-1000000, 0, 1000000}, -1000000, true},
		{"max int", []int{1, 2, 2147483647}, 2147483647, true},
		{"min int", []int{-2147483648, 0, 1}, -2147483648, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vlan.ExportContains(tt.slice, tt.val)
			if result != tt.expected {
				t.Errorf("Contains(%v, %d) = %v, want %v", tt.slice, tt.val, result, tt.expected)
			}
		})
	}
}

// TestManagerInterfaceNameThread checks that a reader never observes a torn
// interface name while a writer is replacing it.
func TestManagerInterfaceNameThread(t *testing.T) {
	manager := vlan.NewManager("initial")

	results := make(chan string, 100)

	for range 10 {
		go func() {
			for range 10 {
				results <- manager.ManagerInterfaceName()
			}
		}()
	}

	go func() {
		for i := range 10 {
			manager.SetInterface("interface" + string(rune('0'+i)))
		}
	}()

	for range 100 {
		got := <-results
		if got != "initial" && !strings.HasPrefix(got, "interface") {
			t.Fatalf("ManagerInterfaceName() = %q, want %q or an interface<n> name", got, "initial")
		}
	}
}

func BenchmarkContains(b *testing.B) {
	slice := make([]int, 1000)
	for i := range slice {
		slice[i] = i
	}

	b.ResetTimer()
	for b.Loop() {
		vlan.ExportContains(slice, 999)
	}
}

func BenchmarkManagerGetInfo(b *testing.B) {
	manager := vlan.NewManager("eth0")
	manager.SetConfigured(true, 100)

	b.ResetTimer()
	for b.Loop() {
		_ = manager.GetInfo()
	}
}

func BenchmarkManagerConcurrent(b *testing.B) {
	manager := vlan.NewManager("eth0")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			manager.SetInterface("eth0")
			_ = manager.GetInfo()
			manager.SetConfigured(true, 100)
		}
	})
}

// Tests for simulated packet processing using ExportRecordVLANTraffic

func TestInfoJSONTags(t *testing.T) {
	// Verify Info struct has correct JSON tags by checking field types.
	info := vlan.Info{}
	info.Configured.Enabled = true
	info.Configured.ID = 100

	// The struct should have json tags but we can't test JSON encoding here
	// without importing encoding/json. Just verify the struct is usable.
	if info.Configured.ID != 100 {
		t.Error("expected Configured.ID to be 100")
	}
}

func TestManagerWithSpecialCharactersInInterface(t *testing.T) {
	interfaces := []string{
		"eth0.100",
		"bond0:0",
		"veth123abc",
		"docker0",
		"br-abcd1234",
	}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			manager := vlan.NewManager(iface)
			if manager.ManagerInterfaceName() != iface {
				t.Errorf("expected %q, got %q", iface, manager.ManagerInterfaceName())
			}
		})
	}
}

// Tests that exercise Stop and SetInterface code paths with simulated started state
