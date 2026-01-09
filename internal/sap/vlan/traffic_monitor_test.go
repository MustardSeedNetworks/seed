//go:build linux

package vlan_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/vlan"
)

// TestTrafficMonitorFullLifecycle tests the complete lifecycle.
func TestTrafficMonitorFullLifecycle(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Verify initial state.
	if monitor.IsRunning() {
		t.Error("should not be running initially")
	}
	if monitor.TrafficMonitorStarted() {
		t.Error("started should be false initially")
	}

	// Simulate started state.
	monitor.SetStartedForTest(true)
	if !monitor.IsRunning() {
		t.Error("should be running after SetStartedForTest(true)")
	}

	// Record some traffic.
	for i := range 100 {
		monitor.ExportRecordVLANTraffic(i%10, uint64(i*100))
	}

	// Verify stats.
	stats := monitor.GetStats()
	if len(stats) != 10 {
		t.Errorf("expected 10 VLANs, got %d", len(stats))
	}

	// Reset while running.
	monitor.Reset()
	stats = monitor.GetStats()
	if len(stats) != 0 {
		t.Errorf("expected 0 VLANs after reset, got %d", len(stats))
	}

	// Stop.
	monitor.Stop()
	if monitor.IsRunning() {
		t.Error("should not be running after Stop")
	}
}

// TestTrafficMonitorSetInterfaceVariations tests SetInterface with various names.
func TestTrafficMonitorSetInterfaceVariations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iface string
	}{
		{"ethernet", "eth0"},
		{"ethernet secondary", "eth1"},
		{"macos ethernet", "en0"},
		{"wireless", "wlan0"},
		{"bond", "bond0"},
		{"bridge", "br0"},
		{"docker", "docker0"},
		{"veth", "veth123"},
		{"loopback", "lo"},
		{"empty", ""},
		{"with dot", "eth0.100"},
		{"with colon", "eth0:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			monitor := vlan.NewTrafficMonitor("initial")

			err := monitor.SetInterface(tt.iface)
			// Error is expected if monitor is not running.
			_ = err

			if monitor.TrafficMonitorInterfaceName() != tt.iface {
				t.Errorf("interface = %q, want %q", monitor.TrafficMonitorInterfaceName(), tt.iface)
			}
		})
	}
}

// TestTrafficMonitorSetInterfaceWhileRunning tests interface change while running.
func TestTrafficMonitorSetInterfaceWhileRunning(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")
	monitor.SetStartedForTest(true)

	// Change interface while running.
	// This will try to restart, which will fail without pcap privileges.
	err := monitor.SetInterface("en0")
	if err == nil {
		// If it succeeded (has pcap access), clean up.
		monitor.Stop()
	}

	// Interface should be updated regardless.
	if monitor.TrafficMonitorInterfaceName() != "en0" {
		t.Errorf("interface = %q, want 'en0'", monitor.TrafficMonitorInterfaceName())
	}
}

// TestTrafficMonitorConcurrentRecordAndRead tests concurrent operations.
func TestTrafficMonitorConcurrentRecordAndRead(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Recorder goroutines.
	numRecorders := 5
	wg.Add(numRecorders)
	for i := range numRecorders {
		go func(id int) {
			defer wg.Done()
			vlanID := id * 10
			for {
				select {
				case <-done:
					return
				default:
					monitor.ExportRecordVLANTraffic(vlanID, 1000)
				}
			}
		}(i)
	}

	// Reader goroutines.
	numReaders := 5
	wg.Add(numReaders)
	for range numReaders {
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
	}

	// Reset goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				time.Sleep(time.Millisecond)
				monitor.Reset()
			}
		}
	}()

	// Let it run for a bit.
	time.Sleep(50 * time.Millisecond)
	close(done)
	wg.Wait()
}

// TestTrafficMonitorStopClearsHandle tests that Stop properly cleans up.
func TestTrafficMonitorStopClearsHandle(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Simulate started state with context.
	monitor.SetStartedForTest(true)
	if !monitor.HasContext() {
		t.Fatal("expected context after SetStartedForTest")
	}
	if !monitor.HasCancelFunc() {
		t.Fatal("expected cancel func after SetStartedForTest")
	}

	// Stop should clear everything.
	monitor.Stop()

	if monitor.HasCancelFunc() {
		t.Error("cancel func should be nil after Stop")
	}
	if monitor.IsRunning() {
		t.Error("should not be running after Stop")
	}
	if monitor.TrafficMonitorStarted() {
		t.Error("started should be false after Stop")
	}
}

// TestTrafficMonitorMultipleStops tests idempotent Stop calls.
func TestTrafficMonitorMultipleStops(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")
	monitor.SetStartedForTest(true)

	// Multiple stops should not panic.
	for range 10 {
		monitor.Stop()
	}

	if monitor.IsRunning() {
		t.Error("should not be running after multiple stops")
	}
}

// TestTrafficMonitorResetPreservesRunningState tests Reset behavior.
func TestTrafficMonitorResetPreservesRunningState(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")
	monitor.SetStartedForTest(true)

	// Add some traffic.
	monitor.ExportRecordVLANTraffic(100, 1500)
	monitor.ExportRecordVLANTraffic(200, 2500)

	if len(monitor.GetStats()) != 2 {
		t.Fatal("expected 2 VLANs before reset")
	}

	// Reset.
	monitor.Reset()

	// Stats should be empty but still running.
	if len(monitor.GetStats()) != 0 {
		t.Error("expected empty stats after reset")
	}
	if !monitor.IsRunning() {
		t.Error("should still be running after reset")
	}

	// Clean up.
	monitor.Stop()
}

// TestTrafficJSONSerialization tests Traffic struct JSON serialization.
func TestTrafficJSONSerialization(t *testing.T) {
	t.Parallel()

	now := time.Now()
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
				ID:       1,
				Packets:  1,
				Bytes:    64,
				LastSeen: now,
			},
		},
		{
			name: "high volume traffic",
			traffic: vlan.Traffic{
				ID:       100,
				Packets:  1000000000,
				Bytes:    1500000000000,
				LastSeen: now,
			},
		},
		{
			name: "max VLAN ID",
			traffic: vlan.Traffic{
				ID:       4094,
				Packets:  999,
				Bytes:    999999,
				LastSeen: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Marshal to JSON.
			data, err := json.Marshal(tt.traffic)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			// Unmarshal back.
			var decoded vlan.Traffic
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			// Verify fields.
			if decoded.ID != tt.traffic.ID {
				t.Errorf("ID = %d, want %d", decoded.ID, tt.traffic.ID)
			}
			if decoded.Packets != tt.traffic.Packets {
				t.Errorf("Packets = %d, want %d", decoded.Packets, tt.traffic.Packets)
			}
			if decoded.Bytes != tt.traffic.Bytes {
				t.Errorf("Bytes = %d, want %d", decoded.Bytes, tt.traffic.Bytes)
			}
		})
	}
}

// TestTrafficMonitorMaxVLANsLimitExceeded tests the VLAN limit.
func TestTrafficMonitorMaxVLANsLimitExceeded(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Fill to max.
	for i := range vlan.ExportMaxTrackedVLANs {
		monitor.ExportRecordVLANTraffic(i, 100)
	}

	// Verify at max.
	if len(monitor.GetStats()) != vlan.ExportMaxTrackedVLANs {
		t.Fatalf("expected %d VLANs, got %d", vlan.ExportMaxTrackedVLANs, len(monitor.GetStats()))
	}

	// Try to add more - should be rejected.
	for i := range 100 {
		monitor.ExportRecordVLANTraffic(vlan.ExportMaxTrackedVLANs+i, 100)
	}

	// Should still be at max.
	if len(monitor.GetStats()) != vlan.ExportMaxTrackedVLANs {
		t.Errorf("expected %d VLANs after limit, got %d", vlan.ExportMaxTrackedVLANs, len(monitor.GetStats()))
	}

	// But updating existing VLANs should work.
	monitor.ExportRecordVLANTraffic(0, 200)
	monitor.ExportRecordVLANTraffic(100, 300)

	stats := monitor.GetStats()
	for _, s := range stats {
		if s.ID == 0 && s.Packets != 2 {
			t.Errorf("VLAN 0: expected 2 packets, got %d", s.Packets)
		}
		if s.ID == 100 && s.Packets != 2 {
			t.Errorf("VLAN 100: expected 2 packets, got %d", s.Packets)
		}
	}
}

// TestTrafficMonitorSetStatsDirectly tests the SetStats helper.
func TestTrafficMonitorSetStatsDirectly(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	now := time.Now()
	testStats := map[int]*vlan.Traffic{
		10:  {ID: 10, Packets: 100, Bytes: 10000, LastSeen: now},
		20:  {ID: 20, Packets: 200, Bytes: 20000, LastSeen: now},
		30:  {ID: 30, Packets: 300, Bytes: 30000, LastSeen: now},
		100: {ID: 100, Packets: 1000, Bytes: 100000, LastSeen: now},
	}

	monitor.SetStats(testStats)

	stats := monitor.GetStats()
	if len(stats) != 4 {
		t.Fatalf("expected 4 VLANs, got %d", len(stats))
	}

	vlanMap := make(map[int]vlan.Traffic)
	for _, s := range stats {
		vlanMap[s.ID] = s
	}

	for id, expected := range testStats {
		got, ok := vlanMap[id]
		if !ok {
			t.Errorf("VLAN %d not found", id)
			continue
		}
		if got.Packets != expected.Packets {
			t.Errorf("VLAN %d: packets = %d, want %d", id, got.Packets, expected.Packets)
		}
		if got.Bytes != expected.Bytes {
			t.Errorf("VLAN %d: bytes = %d, want %d", id, got.Bytes, expected.Bytes)
		}
	}
}

// TestTrafficMonitorStartIdempotent tests that Start is idempotent.
func TestTrafficMonitorStartIdempotent(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")
	monitor.SetStartedForTest(true)

	// Start when already started should return nil.
	err := monitor.Start()
	if err != nil {
		t.Errorf("expected nil error for already-started monitor, got %v", err)
	}

	// Clean up.
	monitor.Stop()
}

// TestTrafficMonitorStartFailure tests Start failure path.
func TestTrafficMonitorStartFailure(t *testing.T) {
	t.Parallel()

	// Use invalid interface to force failure.
	monitor := vlan.NewTrafficMonitor("_invalid_interface_xyz_12345")

	err := monitor.Start()
	if err == nil {
		// If it succeeded (unlikely), clean up.
		monitor.Stop()
		t.Skip("Start succeeded unexpectedly")
	}

	// Should not be running after failed start.
	if monitor.IsRunning() {
		t.Error("should not be running after failed Start")
	}
	if monitor.TrafficMonitorStarted() {
		t.Error("started should be false after failed Start")
	}
}

// TestTrafficMonitorContextLifecycle tests context management.
func TestTrafficMonitorContextLifecycle(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Initially no context.
	if monitor.HasContext() {
		t.Error("should not have context initially")
	}
	if monitor.HasCancelFunc() {
		t.Error("should not have cancel func initially")
	}

	// After SetStartedForTest, should have context.
	monitor.SetStartedForTest(true)
	if !monitor.HasContext() {
		t.Error("should have context after SetStartedForTest")
	}
	if !monitor.HasCancelFunc() {
		t.Error("should have cancel func after SetStartedForTest")
	}

	// After Stop, no cancel func (context might still exist but is cancelled).
	monitor.Stop()
	if monitor.HasCancelFunc() {
		t.Error("should not have cancel func after Stop")
	}
}

// TestTrafficMonitorRecordZeroPacketLength tests recording with zero bytes.
func TestTrafficMonitorRecordZeroPacketLength(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	monitor.ExportRecordVLANTraffic(100, 0)

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 VLAN, got %d", len(stats))
	}
	if stats[0].Bytes != 0 {
		t.Errorf("expected 0 bytes, got %d", stats[0].Bytes)
	}
	if stats[0].Packets != 1 {
		t.Errorf("expected 1 packet, got %d", stats[0].Packets)
	}
}

// TestTrafficMonitorRecordLargePacketLength tests recording jumbo frames.
func TestTrafficMonitorRecordLargePacketLength(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Jumbo frame size.
	monitor.ExportRecordVLANTraffic(100, 9000)

	stats := monitor.GetStats()
	if stats[0].Bytes != 9000 {
		t.Errorf("expected 9000 bytes, got %d", stats[0].Bytes)
	}
}

// TestTrafficMonitorRecordOverflow tests byte accumulation with large values.
func TestTrafficMonitorRecordOverflow(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Record large byte values to test overflow behavior.
	largeBytes := uint64(1<<62) / 2 // Large but won't overflow with a few additions.
	monitor.ExportRecordVLANTraffic(100, largeBytes)
	monitor.ExportRecordVLANTraffic(100, largeBytes)

	stats := monitor.GetStats()
	if stats[0].Bytes != largeBytes*2 {
		t.Errorf("expected %d bytes, got %d", largeBytes*2, stats[0].Bytes)
	}
	if stats[0].Packets != 2 {
		t.Errorf("expected 2 packets, got %d", stats[0].Packets)
	}
}

// TestTrafficMonitorLastSeenProgression tests LastSeen updates.
func TestTrafficMonitorLastSeenProgression(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// First packet.
	monitor.ExportRecordVLANTraffic(100, 1000)
	stats1 := monitor.GetStats()
	firstSeen := stats1[0].LastSeen

	// Small delay.
	time.Sleep(time.Millisecond)

	// Second packet.
	monitor.ExportRecordVLANTraffic(100, 1000)
	stats2 := monitor.GetStats()
	secondSeen := stats2[0].LastSeen

	if secondSeen.Before(firstSeen) {
		t.Error("LastSeen should not go backwards")
	}
}

// TestTrafficMonitorEdgeVLANIDs tests edge case VLAN IDs.
func TestTrafficMonitorEdgeVLANIDs(t *testing.T) {
	t.Parallel()

	tests := []int{0, 1, 100, 1000, 4093, 4094, 4095}

	for _, vlanID := range tests {
		t.Run("VLAN_"+string(rune('0'+vlanID%10)), func(t *testing.T) {
			t.Parallel()

			monitor := vlan.NewTrafficMonitor("eth0")
			monitor.ExportRecordVLANTraffic(vlanID, 1500)

			stats := monitor.GetStats()
			if len(stats) != 1 {
				t.Fatalf("expected 1 VLAN, got %d", len(stats))
			}
			if stats[0].ID != vlanID {
				t.Errorf("expected VLAN ID %d, got %d", vlanID, stats[0].ID)
			}
		})
	}
}

// TestTrafficMonitorConcurrentModification tests concurrent modification safety.
func TestTrafficMonitorConcurrentModification(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	var wg sync.WaitGroup
	numGoroutines := 10
	iterations := 1000

	// Concurrent recorders.
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				monitor.ExportRecordVLANTraffic(id, uint64(j))
			}
		}(i)
	}

	wg.Wait()

	// Verify results.
	stats := monitor.GetStats()
	if len(stats) != numGoroutines {
		t.Errorf("expected %d VLANs, got %d", numGoroutines, len(stats))
	}

	for _, s := range stats {
		if s.Packets != uint64(iterations) {
			t.Errorf("VLAN %d: expected %d packets, got %d", s.ID, iterations, s.Packets)
		}
	}
}

// BenchmarkTrafficMonitorOperations benchmarks various operations.
func BenchmarkTrafficMonitorOperations(b *testing.B) {
	b.Run("NewTrafficMonitor", func(b *testing.B) {
		for b.Loop() {
			_ = vlan.NewTrafficMonitor("eth0")
		}
	})

	b.Run("RecordSingleVLAN", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		b.ResetTimer()
		for b.Loop() {
			monitor.ExportRecordVLANTraffic(100, 1500)
		}
	})

	b.Run("RecordMultipleVLANs", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			monitor.ExportRecordVLANTraffic(i%100, 1500)
		}
	})

	b.Run("GetStatsSmall", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		for i := range 10 {
			monitor.ExportRecordVLANTraffic(i, 1000)
		}
		b.ResetTimer()
		for b.Loop() {
			_ = monitor.GetStats()
		}
	})

	b.Run("GetStatsLarge", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		for i := range 1000 {
			monitor.ExportRecordVLANTraffic(i, 1000)
		}
		b.ResetTimer()
		for b.Loop() {
			_ = monitor.GetStats()
		}
	})

	b.Run("Reset", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		for i := range 100 {
			monitor.ExportRecordVLANTraffic(i, 1000)
		}
		b.ResetTimer()
		for b.Loop() {
			monitor.Reset()
		}
	})

	b.Run("IsRunning", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		b.ResetTimer()
		for b.Loop() {
			_ = monitor.IsRunning()
		}
	})

	b.Run("SetInterface", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		b.ResetTimer()
		for b.Loop() {
			_ = monitor.SetInterface("en0")
		}
	})
}

// BenchmarkTrafficMonitorConcurrent benchmarks concurrent access.
func BenchmarkTrafficMonitorConcurrent(b *testing.B) {
	b.Run("RecordParallel", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				monitor.ExportRecordVLANTraffic(i%100, 1500)
				i++
			}
		})
	})

	b.Run("GetStatsParallel", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		for i := range 100 {
			monitor.ExportRecordVLANTraffic(i, 1000)
		}
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = monitor.GetStats()
			}
		})
	})

	b.Run("MixedParallel", func(b *testing.B) {
		monitor := vlan.NewTrafficMonitor("eth0")
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%3 == 0 {
					monitor.ExportRecordVLANTraffic(i%100, 1500)
				} else {
					_ = monitor.GetStats()
				}
				i++
			}
		})
	})
}
