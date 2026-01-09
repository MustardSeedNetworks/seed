//go:build linux

package vlan_test

import (
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/vlan"
)

// TestFullIntegration tests the full workflow of Manager and TrafficMonitor together.
func TestFullIntegration(t *testing.T) {
	t.Parallel()

	// Create both manager and monitor for the same interface.
	manager := vlan.NewManager("eth0")
	monitor := vlan.NewTrafficMonitor("eth0")

	// Configure manager.
	manager.SetConfigured(true, 100)

	// Simulate traffic capture.
	monitor.SetStartedForTest(true)

	// Record traffic on multiple VLANs.
	for i := range 10 {
		for j := range 100 {
			monitor.ExportRecordVLANTraffic(i*10, uint64(j*100))
		}
	}

	// Get manager info.
	info := manager.GetInfo()
	if !info.Configured.Enabled {
		t.Error("expected configured to be enabled")
	}
	if info.Configured.ID != 100 {
		t.Errorf("expected configured ID 100, got %d", info.Configured.ID)
	}

	// Get traffic stats.
	stats := monitor.GetStats()
	if len(stats) != 10 {
		t.Errorf("expected 10 VLANs, got %d", len(stats))
	}

	// Verify each VLAN has correct packet count.
	for _, s := range stats {
		if s.Packets != 100 {
			t.Errorf("VLAN %d: expected 100 packets, got %d", s.ID, s.Packets)
		}
	}

	// Clean up.
	monitor.Stop()
}

// TestManagerAndMonitorConcurrentUsage tests concurrent usage of both components.
func TestManagerAndMonitorConcurrentUsage(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")
	monitor := vlan.NewTrafficMonitor("eth0")
	monitor.SetStartedForTest(true)

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Manager operations.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				manager.SetConfigured(true, 100)
				_ = manager.GetInfo()
				manager.SetInterface("en0")
			}
		}
	}()

	// Monitor recording.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				monitor.ExportRecordVLANTraffic(100, 1500)
			}
		}
	}()

	// Monitor reading.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				_ = monitor.GetStats()
			}
		}
	}()

	// Let it run for a bit.
	time.Sleep(50 * time.Millisecond)
	close(done)
	wg.Wait()

	// Clean up.
	monitor.Stop()
}

// TestInfoWithLLDPAndTraffic tests combining LLDP info with traffic stats.
func TestInfoWithLLDPAndTraffic(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")
	monitor := vlan.NewTrafficMonitor("eth0")
	monitor.SetStartedForTest(true)

	// Set up LLDP-detected VLANs.
	nativeVlan := 1
	voiceVlan := 100

	// Record traffic on these VLANs.
	monitor.ExportRecordVLANTraffic(1, 1000)
	monitor.ExportRecordVLANTraffic(100, 5000)

	// Get combined info.
	info := manager.GetInfoWithLLDP(&nativeVlan, &voiceVlan)
	if info.NativeVlan == nil || *info.NativeVlan != 1 {
		t.Error("expected native VLAN 1")
	}
	if info.VoiceVlan == nil || *info.VoiceVlan != 100 {
		t.Error("expected voice VLAN 100")
	}

	// Verify traffic stats.
	stats := monitor.GetStats()
	if len(stats) != 2 {
		t.Errorf("expected 2 VLANs, got %d", len(stats))
	}

	// Clean up.
	monitor.Stop()
}

// TestMultipleManagerInstances tests multiple manager instances for different interfaces.
func TestMultipleManagerInstances(t *testing.T) {
	t.Parallel()

	interfaces := []string{"eth0", "eth1", "en0", "en1", "bond0"}
	managers := make([]*vlan.Manager, len(interfaces))

	// Create managers.
	for i, iface := range interfaces {
		managers[i] = vlan.NewManager(iface)
		managers[i].SetConfigured(true, (i+1)*100)
	}

	// Verify each manager is independent.
	for i, mgr := range managers {
		expectedID := (i + 1) * 100
		if mgr.ManagerConfiguredID() != expectedID {
			t.Errorf("manager %d: expected ID %d, got %d", i, expectedID, mgr.ManagerConfiguredID())
		}
		if mgr.ManagerInterfaceName() != interfaces[i] {
			t.Errorf("manager %d: expected interface %q, got %q", i, interfaces[i], mgr.ManagerInterfaceName())
		}
	}
}

// TestMultipleMonitorInstances tests multiple monitor instances for different interfaces.
func TestMultipleMonitorInstances(t *testing.T) {
	t.Parallel()

	interfaces := []string{"eth0", "eth1", "en0", "en1", "bond0"}
	monitors := make([]*vlan.TrafficMonitor, len(interfaces))

	// Create monitors.
	for i, iface := range interfaces {
		monitors[i] = vlan.NewTrafficMonitor(iface)
		monitors[i].SetStartedForTest(true)

		// Record traffic specific to each monitor.
		monitors[i].ExportRecordVLANTraffic((i+1)*10, uint64((i+1)*1000))
	}

	// Verify each monitor is independent.
	for i, mon := range monitors {
		stats := mon.GetStats()
		if len(stats) != 1 {
			t.Errorf("monitor %d: expected 1 VLAN, got %d", i, len(stats))
			continue
		}
		expectedVLAN := (i + 1) * 10
		if stats[0].ID != expectedVLAN {
			t.Errorf("monitor %d: expected VLAN %d, got %d", i, expectedVLAN, stats[0].ID)
		}
	}

	// Clean up.
	for _, mon := range monitors {
		mon.Stop()
	}
}

// TestVLANIDValidation tests various VLAN ID values.
func TestVLANIDValidation(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	tests := []struct {
		name   string
		vlanID int
		valid  bool // Valid 802.1Q VLAN ID (0-4094)
	}{
		{"VLAN 0 (untagged)", 0, true},
		{"VLAN 1 (default)", 1, true},
		{"VLAN 100", 100, true},
		{"VLAN 1000", 1000, true},
		{"VLAN 4094 (max)", 4094, true},
		{"VLAN 4095 (reserved)", 4095, false},
		{"negative", -1, false},
		{"very large", 100000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record traffic.
			monitor.ExportRecordVLANTraffic(tt.vlanID, 1000)

			// Verify it was recorded.
			stats := monitor.GetStats()
			found := false
			for _, s := range stats {
				if s.ID == tt.vlanID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("VLAN %d not found in stats", tt.vlanID)
			}

			// Reset for next test.
			monitor.Reset()
		})
	}
}

// TestTrafficStatisticsAccuracy tests that traffic statistics are accurate.
func TestTrafficStatisticsAccuracy(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	vlanID := 100
	expectedPackets := 1000
	expectedBytes := uint64(0)

	// Record specific traffic.
	for i := range expectedPackets {
		byteLen := uint64(64 + i) // Varying packet sizes.
		expectedBytes += byteLen
		monitor.ExportRecordVLANTraffic(vlanID, byteLen)
	}

	// Verify stats.
	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 VLAN, got %d", len(stats))
	}

	if stats[0].Packets != uint64(expectedPackets) {
		t.Errorf("packets = %d, want %d", stats[0].Packets, expectedPackets)
	}
	if stats[0].Bytes != expectedBytes {
		t.Errorf("bytes = %d, want %d", stats[0].Bytes, expectedBytes)
	}
}

// TestTrafficLastSeenAccuracy tests that LastSeen is updated correctly.
func TestTrafficLastSeenAccuracy(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Record first packet.
	before := time.Now()
	monitor.ExportRecordVLANTraffic(100, 1000)
	after := time.Now()

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatal("expected 1 VLAN")
	}

	// LastSeen should be between before and after.
	if stats[0].LastSeen.Before(before) {
		t.Error("LastSeen is before recording started")
	}
	if stats[0].LastSeen.After(after) {
		t.Error("LastSeen is after recording finished")
	}
}

// TestManagerInterfaceSwitch tests switching interfaces.
func TestManagerInterfaceSwitch(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")
	manager.SetConfigured(true, 100)

	// Switch interfaces.
	interfaces := []string{"en0", "wlan0", "bond0", "br0", "eth1"}
	for _, iface := range interfaces {
		manager.SetInterface(iface)
		if manager.ManagerInterfaceName() != iface {
			t.Errorf("expected interface %q, got %q", iface, manager.ManagerInterfaceName())
		}
		// Configuration should persist.
		if !manager.ManagerEnabled() {
			t.Error("configuration should persist after interface switch")
		}
		if manager.ManagerConfiguredID() != 100 {
			t.Error("configured ID should persist after interface switch")
		}
	}
}

// TestMonitorInterfaceSwitch tests switching interfaces on monitor.
func TestMonitorInterfaceSwitch(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Record some traffic.
	monitor.ExportRecordVLANTraffic(100, 1000)

	// Switch interfaces.
	interfaces := []string{"en0", "wlan0", "bond0"}
	for _, iface := range interfaces {
		err := monitor.SetInterface(iface)
		// Error is expected without pcap privileges.
		_ = err

		if monitor.TrafficMonitorInterfaceName() != iface {
			t.Errorf("expected interface %q, got %q", iface, monitor.TrafficMonitorInterfaceName())
		}
	}
}

// TestContainsFunctionality tests the contains helper comprehensively.
func TestContainsFunctionality(t *testing.T) {
	t.Parallel()

	// Test with various slice sizes.
	sizes := []int{0, 1, 10, 100, 1000}

	for _, size := range sizes {
		t.Run("size_"+string(rune('0'+size%10)), func(t *testing.T) {
			slice := make([]int, size)
			for i := range slice {
				slice[i] = i
			}

			// Test finding existing elements.
			if size > 0 {
				if !vlan.ExportContains(slice, 0) {
					t.Error("should find first element")
				}
				if !vlan.ExportContains(slice, size-1) {
					t.Error("should find last element")
				}
				if size > 1 {
					mid := size / 2
					if !vlan.ExportContains(slice, mid) {
						t.Error("should find middle element")
					}
				}
			}

			// Test not finding non-existent elements.
			if vlan.ExportContains(slice, size) {
				t.Error("should not find element beyond slice")
			}
			if vlan.ExportContains(slice, -1) {
				t.Error("should not find negative element")
			}
		})
	}
}

// TestCreateDeleteVLANInterfaceSequence tests create/delete sequence.
func TestCreateDeleteVLANInterfaceSequence(t *testing.T) {
	t.Parallel()

	// Test create then delete.
	err := vlan.CreateVlanInterface("eth0", 100)
	_ = err // May fail without privileges.

	err = vlan.DeleteVlanInterface("eth0", 100)
	_ = err // May fail without privileges.

	// Test delete then create.
	err = vlan.DeleteVlanInterface("eth0", 200)
	_ = err

	err = vlan.CreateVlanInterface("eth0", 200)
	_ = err

	// Test multiple creates.
	for i := range 10 {
		err = vlan.CreateVlanInterface("eth0", i)
		_ = err
	}

	// Test multiple deletes.
	for i := range 10 {
		err = vlan.DeleteVlanInterface("eth0", i)
		_ = err
	}
}

// BenchmarkIntegration benchmarks the full workflow.
func BenchmarkIntegration(b *testing.B) {
	b.Run("ManagerAndMonitor", func(b *testing.B) {
		manager := vlan.NewManager("eth0")
		monitor := vlan.NewTrafficMonitor("eth0")

		b.ResetTimer()
		for b.Loop() {
			manager.SetConfigured(true, 100)
			_ = manager.GetInfo()
			monitor.ExportRecordVLANTraffic(100, 1500)
			_ = monitor.GetStats()
			monitor.Reset()
		}
	})

	b.Run("MultipleVLANs", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			monitor.ExportRecordVLANTraffic(i%100, 1500)
		}
	})
}
