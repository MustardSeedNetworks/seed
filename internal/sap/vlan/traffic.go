// Package vlan provides VLAN detection and configuration functionality.
package vlan

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// Fixes #915: Maximum number of tracked VLANs (802.1Q max is 4094).
const maxTrackedVLANs = 4096

// Traffic represents per-VLAN traffic statistics.
type Traffic struct {
	ID       int       `json:"id"`
	Packets  uint64    `json:"packets"`
	Bytes    uint64    `json:"bytes"`
	LastSeen time.Time `json:"lastSeen"`
}

// TrafficMonitor captures and tracks 802.1Q VLAN-tagged traffic.
type TrafficMonitor struct {
	interfaceName string
	handle        *pcap.Handle
	stats         map[int]*Traffic // keyed by VLAN ID
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
}

// NewTrafficMonitor creates a new VLAN traffic monitor.
// Fixes #916: Context is created in Start() to prevent leaks if Start() is never called.
func NewTrafficMonitor(interfaceName string) *TrafficMonitor {
	return &TrafficMonitor{
		interfaceName: interfaceName,
		stats:         make(map[int]*Traffic),
	}
}

// Start begins capturing VLAN-tagged traffic.
func (m *TrafficMonitor) Start() error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}

	// Open capture handle
	handle, err := pcap.OpenLive(m.interfaceName, 128, true, pcap.BlockForever)
	if err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to open capture: %w", err)
	}

	// Set BPF filter for 802.1Q tagged frames (EtherType 0x8100)
	if filterErr := handle.SetBPFFilter("vlan"); filterErr != nil {
		// Fixes #941: Log BPF filter failures for debugging (kernel/interface issues)
		slog.Error("Failed to set VLAN BPF filter",
			"interface", m.interfaceName,
			"error", filterErr)
		handle.Close()
		m.mu.Unlock()
		return fmt.Errorf("failed to set BPF filter: %w", filterErr)
	}

	m.handle = handle
	m.started = true
	// Fixes #916: Create context here instead of in constructor
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.mu.Unlock()

	go m.captureLoop()
	return nil
}

// Stop stops capturing VLAN traffic.
func (m *TrafficMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Fixes #916: Check cancel is not nil (Start() may not have been called)
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	if m.handle != nil {
		m.handle.Close()
		m.handle = nil
	}
	m.started = false
}

// SetInterface updates the interface to monitor.
// Fixes #852: Keep lock held throughout operation to prevent TOCTOU race condition.
func (m *TrafficMonitor) SetInterface(name string) error {
	m.mu.Lock()
	wasRunning := m.started

	// Stop capture if running (while holding lock)
	if wasRunning {
		m.cancel()
		if m.handle != nil {
			m.handle.Close()
			m.handle = nil
		}
		m.started = false
	}

	// Fixes #933: Don't create context here - Start() will create its own.
	// Creating context here and in Start() causes the one here to be orphaned.
	m.interfaceName = name
	m.mu.Unlock()

	// Restart if was previously running
	if wasRunning {
		return m.Start()
	}
	return nil
}

// GetStats returns traffic statistics for all observed VLANs.
func (m *TrafficMonitor) GetStats() []Traffic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Traffic, 0, len(m.stats))
	for _, t := range m.stats {
		result = append(result, *t)
	}
	return result
}

// Reset clears all collected statistics.
func (m *TrafficMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats = make(map[int]*Traffic)
}

// captureLoop continuously captures and processes VLAN-tagged frames.
func (m *TrafficMonitor) captureLoop() {
	packetSource := gopacket.NewPacketSource(m.handle, m.handle.LinkType())

	for {
		select {
		case <-m.ctx.Done():
			return
		case packet, ok := <-packetSource.Packets():
			if !ok {
				return
			}
			m.processPacket(packet)
		}
	}
}

// processPacket extracts VLAN information from a captured packet.
func (m *TrafficMonitor) processPacket(packet gopacket.Packet) {
	// Look for 802.1Q (Dot1Q) layer
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
		// Fixes #915: Limit tracked VLANs to prevent unbounded memory growth
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

// IsRunning returns true if capture is active.
func (m *TrafficMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}
