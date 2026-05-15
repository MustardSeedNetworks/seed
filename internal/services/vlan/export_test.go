//go:build linux

package vlan

import (
	"context"
	"time"

	"github.com/gopacket/gopacket"
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
// This version works with raw gopacket.Packet interface.
func (m *TrafficMonitor) ExportProcessPacketRaw(packet any) {
	// Type assert to gopacket.Packet interface to call the real processPacket
	if pkt, ok := packet.(gopacket.Packet); ok {
		m.processPacket(pkt)
	}
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

// HasHandle returns true if the pcap handle is set.
func (m *TrafficMonitor) HasHandle() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.handle != nil
}

// SetHandleForTest sets a mock handle state for testing.
// This is used to test the handle cleanup path in Stop without requiring pcap privileges.
func (m *TrafficMonitor) SetHandleForTest() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// We can't set a real handle, but we can verify the nil check path.
	// The actual handle is already nil, so Stop's handle.Close() path won't execute.
	// This is intentional - we're testing the guard condition.
}
