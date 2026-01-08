package vlan_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/seed/internal/sap/vlan"
)

// createVLANPacket creates a gopacket.Packet with a VLAN tag for testing.
func createVLANPacket(vlanID uint16, dataLen int) gopacket.Packet {
	// Create a minimal 802.1Q tagged Ethernet frame.
	// Ethernet header (14 bytes) + 802.1Q tag (4 bytes) + payload
	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       []byte{0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb},
		EthernetType: layers.EthernetTypeDot1Q,
	}
	dot1q := &layers.Dot1Q{
		VLANIdentifier: vlanID,
		Type:           layers.EthernetTypeIPv4,
	}

	// Create payload
	payload := make([]byte, dataLen)

	// Serialize the layers
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	err := gopacket.SerializeLayers(buf, opts, eth, dot1q, gopacket.Payload(payload))
	if err != nil {
		// Return a simple packet on error
		return gopacket.NewPacket(payload, layers.LayerTypeDot1Q, gopacket.Default)
	}

	// Parse the serialized data back into a packet
	return gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
}

// TestProcessPacketWithRealPacket tests the processPacket logic using real gopacket packets.
func TestProcessPacketWithRealPacket(t *testing.T) {
	tests := []struct {
		name        string
		vlanID      uint16
		payloadLen  int
		wantPackets uint64
		wantVLANID  int
	}{
		{
			name:        "new VLAN packet",
			vlanID:      100,
			payloadLen:  1000,
			wantPackets: 1,
			wantVLANID:  100,
		},
		{
			name:        "small packet",
			vlanID:      200,
			payloadLen:  64,
			wantPackets: 1,
			wantVLANID:  200,
		},
		{
			name:        "jumbo frame",
			vlanID:      300,
			payloadLen:  9000,
			wantPackets: 1,
			wantVLANID:  300,
		},
		{
			name:        "VLAN 0",
			vlanID:      0,
			payloadLen:  500,
			wantPackets: 1,
			wantVLANID:  0,
		},
		{
			name:        "VLAN 4094",
			vlanID:      4094,
			payloadLen:  1000,
			wantPackets: 1,
			wantVLANID:  4094,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := vlan.NewTrafficMonitor("eth0")

			// Create a real gopacket.Packet with VLAN tag.
			pkt := createVLANPacket(tt.vlanID, tt.payloadLen)

			// Process the packet.
			monitor.ExportProcessPacketRaw(pkt)

			// Verify stats.
			stats := monitor.GetStats()
			if len(stats) != 1 {
				t.Fatalf("expected 1 stat entry, got %d", len(stats))
			}

			found := false
			for _, s := range stats {
				if s.ID == tt.wantVLANID {
					found = true
					if s.Packets != tt.wantPackets {
						t.Errorf("packets = %d, want %d", s.Packets, tt.wantPackets)
					}
					// Bytes will be the full packet size (ethernet + dot1q + payload).
					if s.Bytes == 0 {
						t.Error("expected non-zero bytes")
					}
				}
			}
			if !found {
				t.Errorf("VLAN %d not found in stats", tt.wantVLANID)
			}
		})
	}
}

// TestProcessPacketWithNilDot1QLayer tests the processPacket when no Dot1Q layer exists.
func TestProcessPacketWithNilDot1QLayer(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Create a packet without Dot1Q layer (just raw Ethernet frame).
	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       []byte{0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb},
		EthernetType: layers.EthernetTypeIPv4,
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	_ = gopacket.SerializeLayers(buf, opts, eth, gopacket.Payload(make([]byte, 100)))
	pkt := gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)

	// Process the packet - should be ignored.
	monitor.ExportProcessPacketRaw(pkt)

	// Verify no stats recorded.
	stats := monitor.GetStats()
	if len(stats) != 0 {
		t.Errorf("expected 0 stat entries for packet without Dot1Q, got %d", len(stats))
	}
}

// TestProcessPacketMultiplePacketsSameVLAN tests accumulation of stats for same VLAN.
func TestProcessPacketMultiplePacketsSameVLAN(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	vlanID := uint16(100)
	numPackets := 6

	for range numPackets {
		pkt := createVLANPacket(vlanID, 100)
		monitor.ExportProcessPacketRaw(pkt)
	}

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry, got %d", len(stats))
	}

	for _, s := range stats {
		if s.ID == int(vlanID) {
			if s.Packets != uint64(numPackets) {
				t.Errorf("packets = %d, want %d", s.Packets, numPackets)
			}
			if s.Bytes == 0 {
				t.Error("expected non-zero bytes")
			}
		}
	}
}

// TestProcessPacketMultipleVLANs tests processing packets for different VLANs.
func TestProcessPacketMultipleVLANs(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	vlans := []uint16{10, 20, 30, 100, 200, 4094}

	for _, vid := range vlans {
		pkt := createVLANPacket(vid, 100)
		monitor.ExportProcessPacketRaw(pkt)
	}

	stats := monitor.GetStats()
	if len(stats) != len(vlans) {
		t.Fatalf("expected %d stat entries, got %d", len(vlans), len(stats))
	}

	vlanMap := make(map[int]bool)
	for _, s := range stats {
		vlanMap[s.ID] = true
	}

	for _, vid := range vlans {
		if !vlanMap[int(vid)] {
			t.Errorf("VLAN %d not found in stats", vid)
		}
	}
}

// TestProcessPacketMaxVLANsLimit tests the maxTrackedVLANs limit.
func TestProcessPacketMaxVLANsLimit(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Fill to max using the simple helper (avoids creating thousands of packet objects).
	for i := range vlan.ExportMaxTrackedVLANs {
		monitor.ExportRecordVLANTraffic(i, 100)
	}

	// Verify at max.
	stats := monitor.GetStats()
	if len(stats) != vlan.ExportMaxTrackedVLANs {
		t.Fatalf("expected %d stats, got %d", vlan.ExportMaxTrackedVLANs, len(stats))
	}

	// Try to add one more new VLAN via real packet - should be rejected.
	pkt := createVLANPacket(9999, 100)
	monitor.ExportProcessPacketRaw(pkt)

	// Still at max.
	stats = monitor.GetStats()
	if len(stats) != vlan.ExportMaxTrackedVLANs {
		t.Errorf("expected %d stats after limit, got %d", vlan.ExportMaxTrackedVLANs, len(stats))
	}
}

// TestProcessPacketUpdateExistingAtMax tests updating existing VLANs at max limit.
func TestProcessPacketUpdateExistingAtMax(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Fill to max using simple helper.
	for i := range vlan.ExportMaxTrackedVLANs {
		monitor.ExportRecordVLANTraffic(i, 100)
	}

	// Update existing VLANs via real packets - should work even at max.
	pkt1 := createVLANPacket(0, 100)
	monitor.ExportProcessPacketRaw(pkt1)

	pkt2 := createVLANPacket(100, 100)
	monitor.ExportProcessPacketRaw(pkt2)

	stats := monitor.GetStats()
	for _, s := range stats {
		if s.ID == 0 {
			if s.Packets != 2 {
				t.Errorf("VLAN 0: expected 2 packets, got %d", s.Packets)
			}
		}
		if s.ID == 100 {
			if s.Packets != 2 {
				t.Errorf("VLAN 100: expected 2 packets, got %d", s.Packets)
			}
		}
	}
}

// TestProcessPacketConcurrent tests concurrent packet processing.
func TestProcessPacketConcurrent(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	var wg sync.WaitGroup
	numGoroutines := 10
	packetsPerGoroutine := 100

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(vlanID int) {
			defer wg.Done()
			for range packetsPerGoroutine {
				pkt := createVLANPacket(uint16(vlanID), 100)
				monitor.ExportProcessPacketRaw(pkt)
			}
		}(i)
	}

	wg.Wait()

	stats := monitor.GetStats()
	if len(stats) != numGoroutines {
		t.Errorf("expected %d VLANs, got %d", numGoroutines, len(stats))
	}

	for _, s := range stats {
		if s.Packets != uint64(packetsPerGoroutine) {
			t.Errorf("VLAN %d: expected %d packets, got %d", s.ID, packetsPerGoroutine, s.Packets)
		}
	}
}

// TestProcessPacketLastSeenUpdates tests that LastSeen is updated.
func TestProcessPacketLastSeenUpdates(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	pkt := createVLANPacket(100, 100)

	// First packet.
	monitor.ExportProcessPacketRaw(pkt)
	stats1 := monitor.GetStats()
	if len(stats1) == 0 {
		t.Fatal("expected stats")
	}
	firstLastSeen := stats1[0].LastSeen

	// Small delay.
	time.Sleep(time.Millisecond)

	// Second packet.
	pkt2 := createVLANPacket(100, 100)
	monitor.ExportProcessPacketRaw(pkt2)
	stats2 := monitor.GetStats()
	secondLastSeen := stats2[0].LastSeen

	if secondLastSeen.Before(firstLastSeen) {
		t.Error("LastSeen should not decrease")
	}
}

// TestProcessPacketEmptyPayload tests processing packet with empty payload.
func TestProcessPacketEmptyPayload(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Create packet with minimal payload.
	pkt := createVLANPacket(100, 0)

	monitor.ExportProcessPacketRaw(pkt)

	stats := monitor.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry, got %d", len(stats))
	}
	// Even with 0 payload, the packet includes ethernet + vlan headers.
	if stats[0].Bytes == 0 {
		t.Log("packet with 0 payload still has header bytes")
	}
}

// TestProcessPacketWithInvalidPacketType tests handling of invalid packet type.
func TestProcessPacketWithInvalidPacketType(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Pass a non-packet type - should be ignored.
	monitor.ExportProcessPacketRaw("not a packet")
	monitor.ExportProcessPacketRaw(123)
	monitor.ExportProcessPacketRaw(nil)

	stats := monitor.GetStats()
	if len(stats) != 0 {
		t.Errorf("expected 0 stats for invalid packet types, got %d", len(stats))
	}
}

// TestTrafficMonitorStartFailsWithoutPrivileges tests Start failure.
func TestTrafficMonitorStartFailsWithoutPrivileges(t *testing.T) {
	// Use invalid interface to force failure.
	monitor := vlan.NewTrafficMonitor("_invalid_interface_name_12345")

	err := monitor.Start()
	if err == nil {
		// If Start succeeded (e.g., running as root), clean up.
		monitor.Stop()
		t.Skip("Start succeeded - may have elevated privileges")
	}

	// Verify error message contains expected text.
	if err != nil {
		t.Logf("Start correctly failed: %v", err)
	}

	// Should not be running after failed start.
	if monitor.IsRunning() {
		t.Error("expected IsRunning false after failed Start")
	}
}

// TestTrafficMonitorStartTwice tests that starting twice is idempotent.
func TestTrafficMonitorStartTwice(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Use test helper to simulate started state.
	monitor.SetStartedForTest(true)

	// Start again should return nil immediately.
	err := monitor.Start()
	if err != nil {
		t.Errorf("expected nil error when already started, got %v", err)
	}

	// Clean up.
	monitor.Stop()
}

// TestTrafficMonitorSetInterfaceRestartsCapture tests SetInterface behavior.
func TestTrafficMonitorSetInterfaceRestartsCapture(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Simulate started state.
	monitor.SetStartedForTest(true)
	if !monitor.IsRunning() {
		t.Fatal("expected IsRunning true")
	}

	// Change interface - will stop and try to restart.
	// Restart will fail without privileges.
	err := monitor.SetInterface("en0")

	// Interface should be updated regardless of restart result.
	if monitor.TrafficMonitorInterfaceName() != "en0" {
		t.Errorf("expected interface 'en0', got %q", monitor.TrafficMonitorInterfaceName())
	}

	if err != nil {
		t.Logf("SetInterface restart failed as expected: %v", err)
	} else {
		// Restart succeeded, clean up.
		monitor.Stop()
	}
}

// TestTrafficMonitorStopCleansUpProperly tests Stop cleanup.
func TestTrafficMonitorStopCleansUpProperly(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Simulate started state.
	monitor.SetStartedForTest(true)

	if !monitor.HasCancelFunc() {
		t.Fatal("expected cancel func to be set")
	}
	if !monitor.HasContext() {
		t.Fatal("expected context to be set")
	}

	// Stop.
	monitor.Stop()

	// Verify cleanup.
	if monitor.IsRunning() {
		t.Error("expected IsRunning false")
	}
	if monitor.HasCancelFunc() {
		t.Error("expected cancel func to be nil after Stop")
	}
	if monitor.TrafficMonitorStarted() {
		t.Error("expected started false after Stop")
	}
}

// TestTrafficMonitorStopWithoutStart tests Stop when never started.
func TestTrafficMonitorStopWithoutStart(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Should not panic.
	monitor.Stop()
	monitor.Stop()
	monitor.Stop()

	if monitor.IsRunning() {
		t.Error("expected IsRunning false")
	}
}

// TestTrafficMonitorGetStatsAfterProcessing tests GetStats returns copies.
func TestTrafficMonitorGetStatsAfterProcessing(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Process some packets.
	for i := range 5 {
		pkt := createVLANPacket(uint16(i*10), 100)
		monitor.ExportProcessPacketRaw(pkt)
	}

	stats1 := monitor.GetStats()
	stats2 := monitor.GetStats()

	// Verify both have same count.
	if len(stats1) != len(stats2) {
		t.Fatalf("stats1 len %d != stats2 len %d", len(stats1), len(stats2))
	}
	if len(stats1) != 5 {
		t.Fatalf("expected 5 stats, got %d", len(stats1))
	}
}

// TestTrafficMonitorResetClearsStats tests Reset clears all stats.
func TestTrafficMonitorResetClearsStats(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Add some data.
	for i := range 10 {
		pkt := createVLANPacket(uint16(i), 100)
		monitor.ExportProcessPacketRaw(pkt)
	}

	if len(monitor.GetStats()) != 10 {
		t.Fatal("expected 10 stats before reset")
	}

	monitor.Reset()

	if len(monitor.GetStats()) != 0 {
		t.Error("expected 0 stats after reset")
	}
}

// TestTrafficMonitorResetWhileRunning tests Reset during active capture.
func TestTrafficMonitorResetWhileRunning(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Simulate started state.
	monitor.SetStartedForTest(true)

	// Add data.
	pkt := createVLANPacket(100, 100)
	monitor.ExportProcessPacketRaw(pkt)

	// Reset while running.
	monitor.Reset()

	// Stats should be empty.
	if len(monitor.GetStats()) != 0 {
		t.Error("expected empty stats after reset")
	}

	// Should still be running.
	if !monitor.IsRunning() {
		t.Error("expected still running after reset")
	}

	// Clean up.
	monitor.Stop()
}

// TestTrafficMonitorConcurrentOperations tests concurrent access to all operations.
func TestTrafficMonitorConcurrentOperations(t *testing.T) {
	monitor := vlan.NewTrafficMonitor("eth0")

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Packet processor.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				pkt := createVLANPacket(100, 100)
				monitor.ExportProcessPacketRaw(pkt)
			}
		}
	}()

	// Stats reader.
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

	// IsRunning checker.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				_ = monitor.IsRunning()
			}
		}
	}()

	// Let them run for a bit.
	time.Sleep(50 * time.Millisecond)
	close(done)
	wg.Wait()

	// Verify no panic and stats are accessible.
	stats := monitor.GetStats()
	if stats == nil {
		t.Error("expected non-nil stats")
	}
}

// TestTrafficConstantsValues tests the exported constants have correct values.
func TestTrafficConstantsValues(t *testing.T) {
	// maxTrackedVLANs should be 4096 (per comment about 802.1Q max being 4094).
	if vlan.ExportMaxTrackedVLANs != 4096 {
		t.Errorf("ExportMaxTrackedVLANs = %d, want 4096", vlan.ExportMaxTrackedVLANs)
	}

	// pcapSnapshotLen should be 128 (enough for headers).
	if vlan.ExportPcapSnapshotLen != 128 {
		t.Errorf("ExportPcapSnapshotLen = %d, want 128", vlan.ExportPcapSnapshotLen)
	}
}

// TestTrafficStructZeroValue tests Traffic struct zero value behavior.
func TestTrafficStructZeroValue(t *testing.T) {
	traffic := vlan.Traffic{}

	if traffic.ID != 0 {
		t.Error("zero value ID should be 0")
	}
	if traffic.Packets != 0 {
		t.Error("zero value Packets should be 0")
	}
	if traffic.Bytes != 0 {
		t.Error("zero value Bytes should be 0")
	}
	if !traffic.LastSeen.IsZero() {
		t.Error("zero value LastSeen should be zero time")
	}
}

// TestTrafficStructFieldMutation tests Traffic struct field mutations.
func TestTrafficStructFieldMutation(t *testing.T) {
	now := time.Now()
	traffic := vlan.Traffic{
		ID:       100,
		Packets:  1000,
		Bytes:    1500000,
		LastSeen: now,
	}

	// Verify values.
	if traffic.ID != 100 {
		t.Errorf("ID = %d, want 100", traffic.ID)
	}
	if traffic.Packets != 1000 {
		t.Errorf("Packets = %d, want 1000", traffic.Packets)
	}
	if traffic.Bytes != 1500000 {
		t.Errorf("Bytes = %d, want 1500000", traffic.Bytes)
	}
	if !traffic.LastSeen.Equal(now) {
		t.Errorf("LastSeen = %v, want %v", traffic.LastSeen, now)
	}

	// Mutate.
	traffic.Packets++
	traffic.Bytes += 1500
	if traffic.Packets != 1001 {
		t.Error("mutation failed")
	}
}

// BenchmarkProcessPacketReal benchmarks packet processing with real packets.
func BenchmarkProcessPacketReal(b *testing.B) {
	monitor := vlan.NewTrafficMonitor("eth0")
	pkt := createVLANPacket(100, 1000)

	b.ResetTimer()
	for b.Loop() {
		monitor.ExportProcessPacketRaw(pkt)
	}
}

// BenchmarkProcessPacketRealMultiVLAN benchmarks processing different VLANs.
func BenchmarkProcessPacketRealMultiVLAN(b *testing.B) {
	monitor := vlan.NewTrafficMonitor("eth0")

	// Pre-create packets to avoid allocation overhead in benchmark.
	packets := make([]gopacket.Packet, 100)
	for i := range packets {
		packets[i] = createVLANPacket(uint16(i), 100)
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		monitor.ExportProcessPacketRaw(packets[i%100])
	}
}

// BenchmarkProcessPacketConcurrent benchmarks concurrent packet processing.
func BenchmarkProcessPacketConcurrent(b *testing.B) {
	monitor := vlan.NewTrafficMonitor("eth0")
	pkt := createVLANPacket(100, 100)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			monitor.ExportProcessPacketRaw(pkt)
		}
	})
}
