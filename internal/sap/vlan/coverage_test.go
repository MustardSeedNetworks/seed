package vlan_test

import (
	"net"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/seed/internal/sap/vlan"
)

// TestTrafficMonitorHandleState tests handle state tracking.
func TestTrafficMonitorHandleState(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Initially no handle.
	if monitor.HasHandle() {
		t.Error("should not have handle initially")
	}

	// After SetStartedForTest, still no handle (no actual pcap).
	monitor.SetStartedForTest(true)
	if monitor.HasHandle() {
		t.Error("should not have handle after SetStartedForTest (no actual pcap)")
	}

	// After Stop, still no handle.
	monitor.Stop()
	if monitor.HasHandle() {
		t.Error("should not have handle after Stop")
	}
}

// TestTrafficMonitorSetInterfaceRestartPath tests the restart path more thoroughly.
func TestTrafficMonitorSetInterfaceRestartPath(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Start the monitor (simulated).
	monitor.SetStartedForTest(true)

	// Verify state.
	if !monitor.IsRunning() {
		t.Fatal("expected running")
	}
	if !monitor.HasCancelFunc() {
		t.Fatal("expected cancel func")
	}
	if !monitor.HasContext() {
		t.Fatal("expected context")
	}

	// SetInterface when running should stop and try to restart.
	originalInterface := monitor.TrafficMonitorInterfaceName()
	err := monitor.SetInterface("_invalid_for_restart_")

	// Restart fails because pcap can't open the interface.
	if err == nil {
		// If it somehow succeeded, clean up.
		monitor.Stop()
		t.Skip("SetInterface restart succeeded unexpectedly")
	}

	// Interface should be updated despite restart failure.
	if monitor.TrafficMonitorInterfaceName() == originalInterface {
		t.Error("interface should have been updated")
	}

	// After failed restart, monitor should not be running.
	if monitor.IsRunning() {
		t.Error("should not be running after failed restart")
	}
}

// TestTrafficMonitorStopWithSimulatedHandle tests Stop cleanup.
func TestTrafficMonitorStopWithSimulatedHandle(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Simulate full started state.
	monitor.SetStartedForTest(true)

	// Verify all state is set.
	if !monitor.IsRunning() {
		t.Error("expected running")
	}
	if !monitor.HasCancelFunc() {
		t.Error("expected cancel func")
	}
	if !monitor.HasContext() {
		t.Error("expected context")
	}
	// Handle is nil because we didn't actually start pcap.
	if monitor.HasHandle() {
		t.Error("should not have handle (no actual pcap)")
	}

	// Stop should clean up all state.
	monitor.Stop()

	// Verify cleanup.
	if monitor.IsRunning() {
		t.Error("should not be running after Stop")
	}
	if monitor.HasCancelFunc() {
		t.Error("should not have cancel func after Stop")
	}
	if monitor.TrafficMonitorStarted() {
		t.Error("should not be started after Stop")
	}
}

// TestProcessPacketDot1QTypeAssertion tests the Dot1Q type assertion path.
func TestProcessPacketDot1QTypeAssertion(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Create a packet with Dot1Q layer that should pass type assertion.
	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       []byte{0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb},
		EthernetType: layers.EthernetTypeDot1Q,
	}
	dot1q := &layers.Dot1Q{
		VLANIdentifier: 100,
		Type:           layers.EthernetTypeIPv4,
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}
	_ = gopacket.SerializeLayers(buf, opts, eth, dot1q, gopacket.Payload(make([]byte, 100)))
	pkt := gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)

	// Process and verify.
	monitor.ExportProcessPacketRaw(pkt)

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 VLAN, got %d", len(stats))
	}
	if stats[0].ID != 100 {
		t.Errorf("expected VLAN 100, got %d", stats[0].ID)
	}
}

// TestProcessPacketWithVariousPacketTypes tests processPacket with different packet types.
func TestProcessPacketWithVariousPacketTypes(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	// Test with nil packet.
	monitor.ExportProcessPacketRaw(nil)
	if len(monitor.GetStats()) != 0 {
		t.Error("nil packet should not add stats")
	}

	// Test with non-packet type.
	monitor.ExportProcessPacketRaw("not a packet")
	if len(monitor.GetStats()) != 0 {
		t.Error("string should not add stats")
	}

	// Test with int.
	monitor.ExportProcessPacketRaw(42)
	if len(monitor.GetStats()) != 0 {
		t.Error("int should not add stats")
	}

	// Test with struct.
	monitor.ExportProcessPacketRaw(struct{}{})
	if len(monitor.GetStats()) != 0 {
		t.Error("empty struct should not add stats")
	}
}

// TestDetectVlanSubinterfacesPlatformWithSystemInterfaces tests with real system interfaces.
func TestDetectVlanSubinterfacesPlatformWithSystemInterfaces(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test")
	}

	t.Parallel()

	// Get actual system interfaces.
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("failed to get interfaces: %v", err)
	}

	for _, iface := range interfaces {
		t.Run(iface.Name, func(t *testing.T) {
			t.Parallel()

			vlans := vlan.ExportDetectVlanSubinterfacesPlatform(iface.Name)
			if vlans == nil {
				t.Error("expected non-nil slice")
			}
			// Most interfaces won't have VLANs, so we just verify no panic.
		})
	}
}

// TestGetVlanInfoWithRealInterfaces tests getVlanInfo with real interface names on darwin.
func TestGetVlanInfoWithRealInterfaces(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test")
	}

	t.Parallel()

	// Get actual system interfaces.
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("failed to get interfaces: %v", err)
	}

	for _, iface := range interfaces {
		t.Run(iface.Name, func(t *testing.T) {
			t.Parallel()

			parent, vlanID := vlan.ExportGetVlanInfo(iface.Name)

			// For non-VLAN interfaces, both should be zero/empty.
			if !strings.HasPrefix(iface.Name, "vlan") {
				if parent != "" {
					t.Logf("interface %s has parent %s", iface.Name, parent)
				}
				if vlanID != 0 {
					t.Logf("interface %s has VLAN ID %d", iface.Name, vlanID)
				}
			}
		})
	}
}

// TestManagerConcurrentGetInfoWithLLDP tests concurrent GetInfoWithLLDP calls.
func TestManagerConcurrentGetInfoWithLLDP(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")
	manager.SetConfigured(true, 100)

	var wg sync.WaitGroup
	numGoroutines := 50

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			nv := id * 10
			vv := id*10 + 5
			for range 100 {
				info := manager.GetInfoWithLLDP(&nv, &vv)
				if info == nil {
					t.Error("GetInfoWithLLDP returned nil")
				}
				if info.NativeVlan == nil || *info.NativeVlan != nv {
					t.Error("NativeVlan mismatch")
				}
				if info.VoiceVlan == nil || *info.VoiceVlan != vv {
					t.Error("VoiceVlan mismatch")
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestTrafficMonitorConcurrentRecordAndReset tests concurrent record and reset.
func TestTrafficMonitorConcurrentRecordAndReset(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Recorder.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				monitor.ExportRecordVLANTraffic(100, 1000)
			}
		}
	}()

	// Resetter.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				monitor.Reset()
			}
		}
	}()

	// Reader.
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

	time.Sleep(50 * time.Millisecond)
	close(done)
	wg.Wait()
}

// TestTrafficMonitorMaxVLANsBoundaryConditions tests boundary conditions at max VLANs.
func TestTrafficMonitorMaxVLANsBoundaryConditions(t *testing.T) {
	t.Parallel()

	monitor := vlan.NewTrafficMonitor("eth0")
	maxVLANs := vlan.ExportMaxTrackedVLANs

	// Fill to exactly max - 1.
	for i := range maxVLANs - 1 {
		monitor.ExportRecordVLANTraffic(i, 100)
	}
	if len(monitor.GetStats()) != maxVLANs-1 {
		t.Fatalf("expected %d VLANs, got %d", maxVLANs-1, len(monitor.GetStats()))
	}

	// Add one more to reach exactly max.
	monitor.ExportRecordVLANTraffic(maxVLANs-1, 100)
	if len(monitor.GetStats()) != maxVLANs {
		t.Fatalf("expected %d VLANs, got %d", maxVLANs, len(monitor.GetStats()))
	}

	// Try to add one more - should be rejected.
	monitor.ExportRecordVLANTraffic(maxVLANs, 100)
	if len(monitor.GetStats()) != maxVLANs {
		t.Errorf("expected %d VLANs after limit, got %d", maxVLANs, len(monitor.GetStats()))
	}

	// Verify we can still update existing VLANs.
	monitor.ExportRecordVLANTraffic(0, 100)
	stats := monitor.GetStats()
	for _, s := range stats {
		if s.ID == 0 && s.Packets != 2 {
			t.Errorf("VLAN 0 should have 2 packets, got %d", s.Packets)
		}
	}
}

// TestContainsWithDuplicates tests contains with duplicate values in slice.
func TestContainsWithDuplicates(t *testing.T) {
	t.Parallel()

	slice := []int{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}

	// All values should be found.
	for _, v := range []int{1, 2, 3, 4} {
		if !vlan.ExportContains(slice, v) {
			t.Errorf("expected to find %d", v)
		}
	}

	// Non-existent values should not be found.
	for _, v := range []int{0, 5, 10, -1} {
		if vlan.ExportContains(slice, v) {
			t.Errorf("did not expect to find %d", v)
		}
	}
}

// TestInfoFieldMutability tests that Info struct fields can be modified.
func TestInfoFieldMutability(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")

	info := manager.GetInfo()

	// Modify fields.
	nv := 100
	vv := 200
	info.NativeVlan = &nv
	info.VoiceVlan = &vv
	info.TaggedVlans = append(info.TaggedVlans, 10, 20, 30)
	info.Configured.Enabled = true
	info.Configured.ID = 999

	// Get new info and verify original manager state is unchanged.
	info2 := manager.GetInfo()
	if info2.NativeVlan != nil {
		t.Error("original NativeVlan should be nil")
	}
	if info2.VoiceVlan != nil {
		t.Error("original VoiceVlan should be nil")
	}
	if len(info2.TaggedVlans) != 0 {
		t.Error("original TaggedVlans should be empty")
	}
	if info2.Configured.Enabled {
		t.Error("original Configured.Enabled should be false")
	}
}

// TestTrafficStructCopy tests that Traffic struct copies are independent.
func TestTrafficStructCopy(t *testing.T) {
	t.Parallel()

	original := vlan.Traffic{
		ID:       100,
		Packets:  1000,
		Bytes:    150000,
		LastSeen: time.Now(),
	}

	// Create copy.
	copy := original

	// Modify copy.
	copy.Packets = 9999
	copy.Bytes = 999999

	// Original should be unchanged.
	if original.Packets != 1000 {
		t.Error("original Packets should not change")
	}
	if original.Bytes != 150000 {
		t.Error("original Bytes should not change")
	}
}

// TestStartWithBPFFilterFailure documents the BPF filter failure path.
// This path is covered when Start fails after opening pcap but before setting filter.
// Since we can't easily simulate this, we document what would trigger it.
func TestStartWithBPFFilterFailure(t *testing.T) {
	t.Parallel()

	// The BPF filter failure path (lines 67-75 in traffic.go) is triggered when:
	// 1. pcap.OpenLive succeeds (requires valid interface and privileges)
	// 2. handle.SetBPFFilter("vlan") fails (rare, requires kernel/interface issues)
	//
	// This is difficult to test without mocking pcap, but the code path exists
	// for robustness against edge cases.
	//
	// We can verify the error handling works by checking that Start failure
	// doesn't leave the monitor in an inconsistent state.

	monitor := vlan.NewTrafficMonitor("_invalid_interface_")
	err := monitor.Start()

	// Expected to fail.
	if err == nil {
		monitor.Stop()
		t.Skip("Start succeeded unexpectedly")
	}

	// Monitor should be in clean state.
	if monitor.IsRunning() {
		t.Error("should not be running after failed Start")
	}
	if monitor.HasCancelFunc() {
		t.Error("should not have cancel func after failed Start")
	}
	if monitor.HasContext() {
		t.Error("should not have context after failed Start")
	}
	if monitor.HasHandle() {
		t.Error("should not have handle after failed Start")
	}
}

// BenchmarkVLANCoverage benchmarks the main code paths.
func BenchmarkVLANCoverage(b *testing.B) {
	b.Run("FullManagerCycle", func(b *testing.B) {
		for b.Loop() {
			m := vlan.NewManager("eth0")
			m.SetConfigured(true, 100)
			m.SetInterface("en0")
			_ = m.GetInfo()
			nv := 1
			vv := 100
			_ = m.GetInfoWithLLDP(&nv, &vv)
			_ = m.DetectVlanSubinterfaces("eth0")
		}
	})

	b.Run("FullMonitorCycle", func(b *testing.B) {
		for b.Loop() {
			m := vlan.NewTrafficMonitor("eth0")
			m.ExportRecordVLANTraffic(100, 1500)
			_ = m.GetStats()
			_ = m.IsRunning()
			m.Reset()
		}
	})

	b.Run("ConcurrentManagerAccess", func(b *testing.B) {
		m := vlan.NewManager("eth0")
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.SetConfigured(true, 100)
				_ = m.GetInfo()
			}
		})
	})

	b.Run("ConcurrentMonitorAccess", func(b *testing.B) {
		m := vlan.NewTrafficMonitor("eth0")
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				m.ExportRecordVLANTraffic(i%100, 1500)
				_ = m.GetStats()
				i++
			}
		})
	})
}
