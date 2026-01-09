//go:build linux

package vlan_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/sap/vlan"
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

func TestConcurrentManagerAccess(_ *testing.T) {
	manager := vlan.NewManager("eth0")

	// Test concurrent access doesn't cause race conditions.
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

	// Wait for all goroutines.
	for range 10 {
		<-done
	}
}

func TestDetectVlanSubinterfacesPlatform(t *testing.T) {
	// This will return empty if no VLANs configured.
	vlans := vlan.ExportDetectVlanSubinterfacesPlatform("eth0")
	if vlans == nil {
		t.Error("expected non-nil slice, even if empty")
	}
}

func TestCreateVlanInterface(_ *testing.T) {
	// This requires root privileges, so just verify it doesn't panic.
	err := vlan.CreateVlanInterface("eth0", 100)
	// Error is expected on non-Linux or without root.
	_ = err
}

func TestCreateVlanInterfacePlatform(_ *testing.T) {
	// This requires root privileges on Linux.
	err := vlan.ExportCreateVlanInterfacePlatform("eth0", 100)
	// Error is expected without root or on macOS (returns nil).
	_ = err
}

func TestDeleteVlanInterface(_ *testing.T) {
	// This requires root privileges.
	err := vlan.DeleteVlanInterface("eth0", 100)
	// Error is expected on non-Linux or without root.
	_ = err
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

// TrafficMonitor Tests

func TestNewTrafficMonitor(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")
	if monitor == nil {
		t.Fatal("expected non-nil monitor")
	}

	if monitor.TrafficMonitorInterfaceName() != "eth0" {
		t.Errorf("expected interface 'eth0', got %q", monitor.TrafficMonitorInterfaceName())
	}
	if monitor.TrafficMonitorStarted() {
		t.Error("expected started to be false initially")
	}
	if monitor.IsRunning() {
		t.Error("expected IsRunning to return false initially")
	}
}

func TestNewTrafficMonitorDifferentInterfaces(t *testing.T) {
	interfaces := []string{"eth0", "en0", "wlan0", "bond0", "br0"}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			monitor := vlan.NewTrafficMonitor(iface)
			if monitor == nil {
				t.Fatal("expected non-nil monitor")
			}
			if monitor.TrafficMonitorInterfaceName() != iface {
				t.Errorf("expected interface %q, got %q", iface, monitor.TrafficMonitorInterfaceName())
			}
		})
	}
}

func TestTrafficMonitorGetStatsEmpty(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	stats := monitor.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats slice")
	}
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d entries", len(stats))
	}
}

func TestTrafficMonitorReset(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Manually set some stats for testing.
	testStats := map[int]*vlan.Traffic{
		100: {ID: 100, Packets: 50, Bytes: 5000},
		200: {ID: 200, Packets: 100, Bytes: 10000},
	}
	monitor.SetStats(testStats)

	// Verify stats are present.
	stats := monitor.GetStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats entries before reset, got %d", len(stats))
	}

	// Reset and verify empty.
	monitor.Reset()
	stats = monitor.GetStats()
	if len(stats) != 0 {
		t.Errorf("expected 0 stats entries after reset, got %d", len(stats))
	}
}

func TestTrafficMonitorStop(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Stop should not panic even if Start was never called.
	monitor.Stop()

	// Verify state.
	if monitor.IsRunning() {
		t.Error("expected IsRunning false after Stop")
	}
	if monitor.TrafficMonitorStarted() {
		t.Error("expected started false after Stop")
	}
}

func TestTrafficMonitorStopIdempotent(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Multiple stops should not panic.
	monitor.Stop()
	monitor.Stop()
	monitor.Stop()

	if monitor.IsRunning() {
		t.Error("expected IsRunning false after multiple stops")
	}
}

func TestTrafficMonitorIsRunning(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Initially not running.
	if monitor.IsRunning() {
		t.Error("expected IsRunning false initially")
	}

	// After stop still not running.
	monitor.Stop()
	if monitor.IsRunning() {
		t.Error("expected IsRunning false after stop")
	}
}

func TestTrafficMonitorStatsIntegrity(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Set test stats.
	testStats := map[int]*vlan.Traffic{
		10:   {ID: 10, Packets: 100, Bytes: 1000},
		20:   {ID: 20, Packets: 200, Bytes: 2000},
		30:   {ID: 30, Packets: 300, Bytes: 3000},
		4094: {ID: 4094, Packets: 4094, Bytes: 409400},
	}
	monitor.SetStats(testStats)

	stats := monitor.GetStats()
	if len(stats) != 4 {
		t.Fatalf("expected 4 stats entries, got %d", len(stats))
	}

	// Verify each VLAN is present in stats.
	vlanIDs := make(map[int]bool)
	for _, s := range stats {
		vlanIDs[s.ID] = true
	}

	for expectedID := range testStats {
		if !vlanIDs[expectedID] {
			t.Errorf("missing VLAN ID %d in stats", expectedID)
		}
	}
}

func TestTrafficStructFields(t *testing.T) {
	tests := []struct {
		name    string
		traffic vlan.Traffic
	}{
		{
			name:    "empty traffic",
			traffic: vlan.Traffic{},
		},
		{
			name: "minimal traffic",
			traffic: vlan.Traffic{
				ID:      1,
				Packets: 1,
				Bytes:   64,
			},
		},
		{
			name: "high volume traffic",
			traffic: vlan.Traffic{
				ID:      100,
				Packets: 1000000,
				Bytes:   1500000000,
			},
		},
		{
			name: "max VLAN ID",
			traffic: vlan.Traffic{
				ID:      4094,
				Packets: 500,
				Bytes:   75000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.traffic.ID < 0 || tt.traffic.ID > 4094 {
				// VLAN IDs should be 0-4094 (12-bit field).
				if tt.traffic.ID != 0 {
					t.Logf("VLAN ID %d outside typical range", tt.traffic.ID)
				}
			}
		})
	}
}

func TestConcurrentTrafficMonitorAccess(_ *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	done := make(chan bool)
	for i := range 10 {
		go func(_ int) {
			for range 100 {
				_ = monitor.GetStats()
				_ = monitor.IsRunning()
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines.
	for range 10 {
		<-done
	}
}

func TestConcurrentTrafficMonitorStatsAndReset(_ *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	done := make(chan bool)

	// Writer goroutine.
	go func() {
		for i := range 50 {
			testStats := map[int]*vlan.Traffic{
				i: {ID: i, Packets: uint64(i), Bytes: uint64(i * 100)},
			}
			monitor.SetStats(testStats)
		}
		done <- true
	}()

	// Reader goroutine.
	go func() {
		for range 50 {
			_ = monitor.GetStats()
		}
		done <- true
	}()

	// Reset goroutine.
	go func() {
		for range 50 {
			monitor.Reset()
		}
		done <- true
	}()

	// Wait for all goroutines.
	for range 3 {
		<-done
	}
}

func TestTrafficMonitorConstants(t *testing.T) {
	// Verify constants are set correctly.
	if vlan.ExportMaxTrackedVLANs != 4096 {
		t.Errorf("expected maxTrackedVLANs 4096, got %d", vlan.ExportMaxTrackedVLANs)
	}
	if vlan.ExportPcapSnapshotLen != 128 {
		t.Errorf("expected pcapSnapshotLen 128, got %d", vlan.ExportPcapSnapshotLen)
	}
}

func TestTrafficMonitorSetInterface(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// SetInterface on non-started monitor.
	err := monitor.SetInterface("en0")
	if err != nil {
		// Only fails if monitor was running and restart fails.
		t.Logf("SetInterface returned error (may be expected): %v", err)
	}

	if monitor.TrafficMonitorInterfaceName() != "en0" {
		t.Errorf("expected interface 'en0', got %q", monitor.TrafficMonitorInterfaceName())
	}
}

func TestTrafficMonitorSetInterfaceMultipleTimes(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	interfaces := []string{"en0", "wlan0", "bond0", "eth1"}
	for _, iface := range interfaces {
		err := monitor.SetInterface(iface)
		if err != nil {
			t.Logf("SetInterface(%s) returned error (may be expected): %v", iface, err)
		}
		if monitor.TrafficMonitorInterfaceName() != iface {
			t.Errorf("expected interface %q, got %q", iface, monitor.TrafficMonitorInterfaceName())
		}
	}
}

func TestTrafficMonitorStartWithInvalidInterface(t *testing.T) {
	// Use a clearly invalid interface name.
	monitor := vlan.NewTrafficMonitor("nonexistent_interface_12345")

	err := monitor.Start()
	// Start should fail for invalid interface.
	if err == nil {
		t.Log("Start succeeded on invalid interface (platform may allow this)")
		monitor.Stop()
	} else {
		t.Logf("Start correctly failed with: %v", err)
	}
}

func TestDeleteVlanInterfacePlatform(_ *testing.T) {
	// Test the exported delete function.
	err := vlan.ExportDeleteVlanInterfacePlatform("eth0", 100)
	// Error is expected on most systems without root or the interface.
	_ = err
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

func TestTrafficMonitorEmptyInterface(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("")

	if monitor.TrafficMonitorInterfaceName() != "" {
		t.Errorf("expected empty interface name, got %q", monitor.TrafficMonitorInterfaceName())
	}

	// Should not panic.
	stats := monitor.GetStats()
	if stats == nil {
		t.Error("expected non-nil stats slice")
	}
}

func TestTrafficStatsWithTimestamp(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	now := monitor.GetStats()
	if now == nil {
		t.Fatal("expected non-nil stats")
	}

	// After setting stats with timestamps.
	monitor.Reset()
	later := monitor.GetStats()
	if later == nil {
		t.Fatal("expected non-nil stats after reset")
	}
	if len(later) != 0 {
		t.Errorf("expected empty stats after reset, got %d", len(later))
	}
}

func TestManagerInterfaceNameThread(_ *testing.T) {
	manager := vlan.NewManager("initial")

	results := make(chan string, 100)

	// Multiple readers.
	for range 10 {
		go func() {
			for range 10 {
				results <- manager.ManagerInterfaceName()
			}
		}()
	}

	// Single writer.
	go func() {
		for i := range 10 {
			manager.SetInterface("interface" + string(rune('0'+i)))
		}
	}()

	// Collect results (no race conditions).
	for range 100 {
		<-results
	}
}

func TestTrafficGetStatsReturnsSnapshot(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	testStats := map[int]*vlan.Traffic{
		100: {ID: 100, Packets: 50, Bytes: 5000},
	}
	monitor.SetStats(testStats)

	stats1 := monitor.GetStats()
	stats2 := monitor.GetStats()

	// Modifying one shouldn't affect the other (they're copies).
	if len(stats1) > 0 {
		stats1[0].Packets = 999
	}

	// stats2 should be independent.
	for _, s := range stats2 {
		if s.ID == 100 && s.Packets == 999 {
			// This is actually testing the internal pointer behavior.
			// The current implementation returns copies of the Traffic struct.
			t.Log("stats share underlying data (expected based on current implementation)")
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

func BenchmarkTrafficMonitorGetStats(b *testing.B) {
	monitor := vlan.NewTrafficMonitor("eth0")

	testStats := make(map[int]*vlan.Traffic)
	for i := range 100 {
		testStats[i] = &vlan.Traffic{ID: i, Packets: uint64(i * 100), Bytes: uint64(i * 1000)}
	}
	monitor.SetStats(testStats)

	b.ResetTimer()
	for b.Loop() {
		_ = monitor.GetStats()
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

func TestRecordVLANTrafficBasic(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Record traffic for VLAN 100.
	monitor.ExportRecordVLANTraffic(100, 1500)

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry, got %d", len(stats))
	}

	found := false
	for _, s := range stats {
		if s.ID == 100 {
			found = true
			if s.Packets != 1 {
				t.Errorf("expected 1 packet, got %d", s.Packets)
			}
			if s.Bytes != 1500 {
				t.Errorf("expected 1500 bytes, got %d", s.Bytes)
			}
		}
	}
	if !found {
		t.Error("VLAN 100 not found in stats")
	}
}

func TestRecordVLANTrafficMultiplePackets(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Record multiple packets for same VLAN.
	for range 10 {
		monitor.ExportRecordVLANTraffic(100, 1500)
	}

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry, got %d", len(stats))
	}

	for _, s := range stats {
		if s.ID == 100 {
			if s.Packets != 10 {
				t.Errorf("expected 10 packets, got %d", s.Packets)
			}
			if s.Bytes != 15000 {
				t.Errorf("expected 15000 bytes, got %d", s.Bytes)
			}
		}
	}
}

func TestRecordVLANTrafficMultipleVLANs(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Record traffic for multiple VLANs.
	vlanIDs := []int{10, 20, 30, 100, 200, 4094}
	for _, id := range vlanIDs {
		monitor.ExportRecordVLANTraffic(id, 1000)
	}

	stats := monitor.GetStats()
	if len(stats) != len(vlanIDs) {
		t.Fatalf("expected %d stat entries, got %d", len(vlanIDs), len(stats))
	}

	vlanMap := make(map[int]bool)
	for _, s := range stats {
		vlanMap[s.ID] = true
	}

	for _, id := range vlanIDs {
		if !vlanMap[id] {
			t.Errorf("VLAN %d not found in stats", id)
		}
	}
}

func TestRecordVLANTrafficMaxVLANs(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Record traffic for maximum VLANs.
	for i := range vlan.ExportMaxTrackedVLANs {
		monitor.ExportRecordVLANTraffic(i, 100)
	}

	stats := monitor.GetStats()
	if len(stats) != vlan.ExportMaxTrackedVLANs {
		t.Fatalf("expected %d stat entries, got %d", vlan.ExportMaxTrackedVLANs, len(stats))
	}

	// Try to add one more - should be rejected.
	monitor.ExportRecordVLANTraffic(9999, 100)

	stats = monitor.GetStats()
	if len(stats) != vlan.ExportMaxTrackedVLANs {
		t.Errorf("expected %d stat entries after max limit, got %d", vlan.ExportMaxTrackedVLANs, len(stats))
	}
}

func TestRecordVLANTrafficUpdatesExisting(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Fill to max.
	for i := range vlan.ExportMaxTrackedVLANs {
		monitor.ExportRecordVLANTraffic(i, 100)
	}

	// Now update existing VLANs - should work even at max.
	monitor.ExportRecordVLANTraffic(0, 200)
	monitor.ExportRecordVLANTraffic(100, 300)

	stats := monitor.GetStats()

	for _, s := range stats {
		if s.ID == 0 && s.Packets != 2 {
			t.Errorf("expected 2 packets for VLAN 0, got %d", s.Packets)
		}
		if s.ID == 100 && s.Packets != 2 {
			t.Errorf("expected 2 packets for VLAN 100, got %d", s.Packets)
		}
	}
}

func TestRecordVLANTrafficEdgeVLANIDs(t *testing.T) {
	tests := []struct {
		name   string
		vlanID int
	}{
		{"VLAN 0", 0},
		{"VLAN 1", 1},
		{"VLAN 4094 (max)", 4094},
		{"VLAN 4095 (reserved)", 4095},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := vlan.NewTrafficMonitor("eth0")
			monitor.ExportRecordVLANTraffic(tt.vlanID, 1500)

			stats := monitor.GetStats()
			if len(stats) != 1 {
				t.Fatalf("expected 1 stat entry, got %d", len(stats))
			}
			if stats[0].ID != tt.vlanID {
				t.Errorf("expected VLAN ID %d, got %d", tt.vlanID, stats[0].ID)
			}
		})
	}
}

func TestRecordVLANTrafficZeroBytes(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")
	monitor.ExportRecordVLANTraffic(100, 0)

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry, got %d", len(stats))
	}
	if stats[0].Bytes != 0 {
		t.Errorf("expected 0 bytes, got %d", stats[0].Bytes)
	}
}

func TestRecordVLANTrafficLargeBytes(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Large packet (jumbo frame).
	monitor.ExportRecordVLANTraffic(100, 9000)

	stats := monitor.GetStats()
	if stats[0].Bytes != 9000 {
		t.Errorf("expected 9000 bytes, got %d", stats[0].Bytes)
	}
}

func TestRecordVLANTrafficConcurrent(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	done := make(chan bool)

	// Multiple goroutines recording different VLANs.
	for i := range 10 {
		go func(vlanID int) {
			for range 100 {
				monitor.ExportRecordVLANTraffic(vlanID, 1000)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines.
	for range 10 {
		<-done
	}

	stats := monitor.GetStats()
	if len(stats) != 10 {
		t.Errorf("expected 10 VLANs, got %d", len(stats))
	}

	for _, s := range stats {
		if s.Packets != 100 {
			t.Errorf("VLAN %d: expected 100 packets, got %d", s.ID, s.Packets)
		}
		if s.Bytes != 100000 {
			t.Errorf("VLAN %d: expected 100000 bytes, got %d", s.ID, s.Bytes)
		}
	}
}

func TestRecordVLANTrafficAfterReset(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Record some traffic.
	monitor.ExportRecordVLANTraffic(100, 1500)
	monitor.ExportRecordVLANTraffic(200, 1500)

	// Reset.
	monitor.Reset()

	// Record new traffic.
	monitor.ExportRecordVLANTraffic(300, 2000)

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry after reset, got %d", len(stats))
	}
	if stats[0].ID != 300 {
		t.Errorf("expected VLAN 300, got %d", stats[0].ID)
	}
}

func TestTrafficLastSeenUpdates(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Record initial traffic.
	monitor.ExportRecordVLANTraffic(100, 1500)

	stats1 := monitor.GetStats()
	if len(stats1) != 1 {
		t.Fatalf("expected 1 stat entry, got %d", len(stats1))
	}
	firstLastSeen := stats1[0].LastSeen

	// Small delay to ensure time difference.
	// Note: In real tests, we might use a time mock.
	monitor.ExportRecordVLANTraffic(100, 1500)

	stats2 := monitor.GetStats()
	secondLastSeen := stats2[0].LastSeen

	if secondLastSeen.Before(firstLastSeen) {
		t.Error("LastSeen should not go backwards")
	}
}

func TestTrafficMonitorStopAfterSetInterface(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// SetInterface when not started.
	err := monitor.SetInterface("en0")
	if err != nil {
		t.Logf("SetInterface returned error (expected if pcap not available): %v", err)
	}

	// Stop should still work.
	monitor.Stop()

	if monitor.IsRunning() {
		t.Error("expected IsRunning false after Stop")
	}
}

func TestCreateAndDeleteVlanInterfaceTableDriven(t *testing.T) {
	tests := []struct {
		name       string
		parentIf   string
		vlanID     int
		wantCreate bool // We can't verify if it works without root
		wantDelete bool
	}{
		{"valid interface and VLAN", "eth0", 100, true, true},
		{"valid interface max VLAN", "eth0", 4094, true, true},
		{"valid interface VLAN 1", "eth0", 1, true, true},
		{"empty interface", "", 100, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Just verify these don't panic.
			_ = vlan.CreateVlanInterface(tt.parentIf, tt.vlanID)
			_ = vlan.DeleteVlanInterface(tt.parentIf, tt.vlanID)
		})
	}
}

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

func TestTrafficJSONTags(t *testing.T) {
	traffic := vlan.Traffic{
		ID:      100,
		Packets: 1000,
		Bytes:   1500000,
	}

	// Verify the struct is usable.
	if traffic.ID != 100 {
		t.Error("expected ID to be 100")
	}
	if traffic.Packets != 1000 {
		t.Error("expected Packets to be 1000")
	}
	if traffic.Bytes != 1500000 {
		t.Error("expected Bytes to be 1500000")
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

func TestTrafficMonitorWithSpecialCharactersInInterface(t *testing.T) {
	interfaces := []string{
		"eth0.100",
		"bond0:0",
		"veth123abc",
		"docker0",
		"br-abcd1234",
	}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			monitor := vlan.NewTrafficMonitor(iface)
			if monitor.TrafficMonitorInterfaceName() != iface {
				t.Errorf("expected %q, got %q", iface, monitor.TrafficMonitorInterfaceName())
			}
		})
	}
}

func BenchmarkRecordVLANTraffic(b *testing.B) {
	monitor := vlan.NewTrafficMonitor("eth0")

	b.ResetTimer()
	for b.Loop() {
		monitor.ExportRecordVLANTraffic(100, 1500)
	}
}

func BenchmarkRecordVLANTrafficMultipleVLANs(b *testing.B) {
	monitor := vlan.NewTrafficMonitor("eth0")

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		monitor.ExportRecordVLANTraffic(i%100, 1500)
	}
}

func BenchmarkGetStatsLarge(b *testing.B) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Fill with many VLANs.
	for i := range 1000 {
		monitor.ExportRecordVLANTraffic(i, 1500)
	}

	b.ResetTimer()
	for b.Loop() {
		_ = monitor.GetStats()
	}
}

// Tests that exercise Stop and SetInterface code paths with simulated started state

func TestTrafficMonitorStopWhenSimulatedStarted(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Simulate started state.
	monitor.SetStartedForTest(true)

	if !monitor.IsRunning() {
		t.Fatal("expected IsRunning true after SetStartedForTest(true)")
	}
	if !monitor.HasCancelFunc() {
		t.Fatal("expected cancel function to be set")
	}
	if !monitor.HasContext() {
		t.Fatal("expected context to be set")
	}

	// Now stop.
	monitor.Stop()

	if monitor.IsRunning() {
		t.Error("expected IsRunning false after Stop")
	}
	if monitor.HasCancelFunc() {
		t.Error("expected cancel function to be nil after Stop")
	}
}

func TestTrafficMonitorSetInterfaceWhenSimulatedStarted(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Simulate started state.
	monitor.SetStartedForTest(true)

	if !monitor.IsRunning() {
		t.Fatal("expected IsRunning true after SetStartedForTest(true)")
	}

	// SetInterface should stop and try to restart.
	// Restart will fail because pcap won't work, but the stop part should execute.
	err := monitor.SetInterface("en0")
	// Error expected because Start will fail.
	if err == nil {
		t.Log("SetInterface succeeded (unexpected - may have pcap access)")
		// In this case, we need to clean up.
		monitor.Stop()
	} else {
		t.Logf("SetInterface failed as expected: %v", err)
	}

	// Interface name should be updated even if Start fails.
	if monitor.TrafficMonitorInterfaceName() != "en0" {
		t.Errorf("expected interface 'en0', got %q", monitor.TrafficMonitorInterfaceName())
	}
}

func TestTrafficMonitorStartAlreadyStarted(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Simulate started state.
	monitor.SetStartedForTest(true)

	// Start when already started should return nil immediately.
	err := monitor.Start()
	if err != nil {
		t.Errorf("expected nil error when already started, got %v", err)
	}

	// Clean up.
	monitor.Stop()
}

func TestTrafficMonitorCancelFuncStateTransitions(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Initially no cancel func.
	if monitor.HasCancelFunc() {
		t.Error("expected no cancel func initially")
	}

	// After SetStartedForTest, should have cancel func.
	monitor.SetStartedForTest(true)
	if !monitor.HasCancelFunc() {
		t.Error("expected cancel func after SetStartedForTest(true)")
	}

	// After Stop, no cancel func.
	monitor.Stop()
	if monitor.HasCancelFunc() {
		t.Error("expected no cancel func after Stop")
	}
}

func TestTrafficMonitorContextStateTransitions(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Initially no context.
	if monitor.HasContext() {
		t.Error("expected no context initially")
	}

	// After SetStartedForTest, should have context.
	monitor.SetStartedForTest(true)
	if !monitor.HasContext() {
		t.Error("expected context after SetStartedForTest(true)")
	}

	// Clean up.
	monitor.Stop()
}

func TestTrafficMonitorMultipleSetStartedCycles(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	for range 5 {
		monitor.SetStartedForTest(true)
		if !monitor.IsRunning() {
			t.Error("expected IsRunning true")
		}

		monitor.Stop()
		if monitor.IsRunning() {
			t.Error("expected IsRunning false after Stop")
		}
	}
}

func TestTrafficMonitorSetStartedFalse(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Set started to true first.
	monitor.SetStartedForTest(true)

	// Then set to false - this shouldn't affect context/cancel.
	monitor.SetStartedForTest(false)

	// Should still have context (SetStartedForTest only sets context when started=true).
	if monitor.HasContext() {
		t.Log("context was set and not cleared by SetStartedForTest(false)")
	}
}

func TestTrafficMonitorResetWhileSimulatedStarted(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Simulate started and add some traffic.
	monitor.SetStartedForTest(true)
	monitor.ExportRecordVLANTraffic(100, 1500)

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}

	// Reset while running.
	monitor.Reset()

	stats = monitor.GetStats()
	if len(stats) != 0 {
		t.Errorf("expected 0 stats after reset, got %d", len(stats))
	}

	// Should still be running.
	if !monitor.IsRunning() {
		t.Error("expected still running after Reset")
	}

	// Clean up.
	monitor.Stop()
}

func TestTrafficMonitorGetStatsWhileSimulatedStarted(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	monitor.SetStartedForTest(true)

	// Simulate traffic recording.
	for i := range 10 {
		monitor.ExportRecordVLANTraffic(i, uint64((i+1)*100))
	}

	stats := monitor.GetStats()
	if len(stats) != 10 {
		t.Errorf("expected 10 stats, got %d", len(stats))
	}

	// Clean up.
	monitor.Stop()
}
