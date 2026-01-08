package vlan

import (
	"context"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// ManagerInterfaceName returns the interface name for testing.
func (m *Manager) ManagerInterfaceName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.interfaceName
}

// ManagerEnabled returns the enabled state for testing.
func (m *Manager) ManagerEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// ManagerConfiguredID returns the configured ID for testing.
func (m *Manager) ManagerConfiguredID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.configuredID
}

// ExportContains exposes contains for testing.
func ExportContains(slice []int, val int) bool {
	return contains(slice, val)
}

// ExportDetectVlanSubinterfacesPlatform exposes detectVlanSubinterfacesPlatform for testing.
func ExportDetectVlanSubinterfacesPlatform(iface string) []int {
	return detectVlanSubinterfacesPlatform(iface)
}

// ExportCreateVlanInterfacePlatform exposes createVlanInterfacePlatform for testing.
func ExportCreateVlanInterfacePlatform(parentIface string, vlanID int) error {
	return createVlanInterfacePlatform(parentIface, vlanID)
}

// DetectVlanSubinterfaces is exported for testing.
func (m *Manager) DetectVlanSubinterfaces(iface string) []int {
	return m.detectVlanSubinterfaces(iface)
}

// ExportDeleteVlanInterfacePlatform exposes deleteVlanInterfacePlatform for testing.
func ExportDeleteVlanInterfacePlatform(parentIface string, vlanID int) error {
	return deleteVlanInterfacePlatform(parentIface, vlanID)
}

// TrafficMonitorInterfaceName returns the interface name for testing.
func (m *TrafficMonitor) TrafficMonitorInterfaceName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.interfaceName
}

// TrafficMonitorStarted returns whether the monitor is started for testing.
func (m *TrafficMonitor) TrafficMonitorStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

// TrafficMonitorStats returns the internal stats map for testing.
func (m *TrafficMonitor) TrafficMonitorStats() map[int]*Traffic {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// ExportProcessPacketRaw exposes processPacket logic for testing.
// This version works with raw gopacket.Packet.
func (m *TrafficMonitor) ExportProcessPacketRaw(packet any) {
	// Type assert to gopacket.Packet interface to test processPacket
	if pkt, ok := packet.(interface {
		Layer(gopacket.LayerType) gopacket.Layer
		Data() []byte
	}); ok {
		m.processPacketInternal(pkt)
	}
}

// processPacketInternal is the internal test helper that mirrors processPacket.
func (m *TrafficMonitor) processPacketInternal(packet interface {
	Layer(gopacket.LayerType) gopacket.Layer
	Data() []byte
}) {
	dot1qLayer := packet.Layer(layers.LayerTypeDot1Q)
	if dot1qLayer == nil {
		return
	}

	dot1q, ok := dot1qLayer.(*layers.Dot1Q)
	if !ok {
		return
	}

	vlanID := int(dot1q.VLANIdentifier)
	packetLen := uint64(len(packet.Data()))

	m.mu.Lock()
	if stats, exists := m.stats[vlanID]; exists {
		stats.Packets++
		stats.Bytes += packetLen
		stats.LastSeen = time.Now()
	} else {
		if len(m.stats) >= maxTrackedVLANs {
			m.mu.Unlock()
			return
		}
		m.stats[vlanID] = &Traffic{
			ID:       vlanID,
			Packets:  1,
			Bytes:    packetLen,
			LastSeen: time.Now(),
		}
	}
	m.mu.Unlock()
}

// ExportRecordVLANTraffic is a simplified test helper to simulate packet recording
// without needing actual gopacket objects. This simulates what processPacket does.
func (m *TrafficMonitor) ExportRecordVLANTraffic(vlanID int, packetLen uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stats, exists := m.stats[vlanID]; exists {
		stats.Packets++
		stats.Bytes += packetLen
		stats.LastSeen = time.Now()
	} else {
		if len(m.stats) >= maxTrackedVLANs {
			return
		}
		m.stats[vlanID] = &Traffic{
			ID:       vlanID,
			Packets:  1,
			Bytes:    packetLen,
			LastSeen: time.Now(),
		}
	}
}

// SetStats allows setting internal stats for testing.
func (m *TrafficMonitor) SetStats(stats map[int]*Traffic) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats = stats
}

// ExportMaxTrackedVLANs exposes the maxTrackedVLANs constant for testing.
const ExportMaxTrackedVLANs = maxTrackedVLANs

// ExportPcapSnapshotLen exposes the pcapSnapshotLen constant for testing.
const ExportPcapSnapshotLen = pcapSnapshotLen

// SetStartedForTest sets the started flag for testing.
// This allows testing Stop and SetInterface code paths that require a running monitor.
func (m *TrafficMonitor) SetStartedForTest(started bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = started
	if started && m.ctx == nil {
		m.ctx, m.cancel = context.WithCancel(context.Background())
	}
}

// HasCancelFunc returns true if the cancel function is set.
func (m *TrafficMonitor) HasCancelFunc() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cancel != nil
}

// HasContext returns true if the context is set.
func (m *TrafficMonitor) HasContext() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx != nil
}
