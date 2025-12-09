// Package vlan provides VLAN detection and configuration functionality.
package vlan

import (
	"sync"
)

// Info contains VLAN information for an interface.
type Info struct {
	NativeVlan  *int  `json:"nativeVlan"`
	TaggedVlans []int `json:"taggedVlans"`
	VoiceVlan   *int  `json:"voiceVlan"`
	Configured  struct {
		Enabled bool `json:"enabled"`
		ID      int  `json:"id"`
	} `json:"configured"`
}

// Manager handles VLAN detection and configuration.
type Manager struct {
	interfaceName string
	configuredID  int
	enabled       bool
	mu            sync.RWMutex
}

// NewManager creates a new VLAN manager.
func NewManager(interfaceName string) *Manager {
	return &Manager{
		interfaceName: interfaceName,
	}
}

// SetInterface updates the interface to monitor.
func (m *Manager) SetInterface(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interfaceName = name
}

// SetConfigured sets the configured VLAN tagging.
func (m *Manager) SetConfigured(enabled bool, id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
	m.configuredID = id
}

// GetInfo returns current VLAN information.
func (m *Manager) GetInfo() *Info {
	m.mu.RLock()
	iface := m.interfaceName
	enabled := m.enabled
	configuredID := m.configuredID
	m.mu.RUnlock()

	info := &Info{
		TaggedVlans: make([]int, 0),
	}
	info.Configured.Enabled = enabled
	info.Configured.ID = configuredID

	// Detect VLAN subinterfaces
	taggedVlans := m.detectVlanSubinterfaces(iface)
	info.TaggedVlans = taggedVlans

	return info
}

// GetInfoWithLLDP returns VLAN info enriched with LLDP/CDP data.
func (m *Manager) GetInfoWithLLDP(nativeVlan, voiceVlan *int) *Info {
	info := m.GetInfo()
	info.NativeVlan = nativeVlan
	info.VoiceVlan = voiceVlan
	return info
}

// detectVlanSubinterfaces finds 802.1Q VLAN subinterfaces.
// Implementation is platform-specific (vlan_linux.go, vlan_darwin.go).
func (m *Manager) detectVlanSubinterfaces(iface string) []int {
	return detectVlanSubinterfacesPlatform(iface)
}

// CreateVlanInterface creates an 802.1Q VLAN subinterface.
// Implementation is platform-specific (vlan_linux.go, vlan_darwin.go).
func CreateVlanInterface(parentIface string, vlanID int) error {
	return createVlanInterfacePlatform(parentIface, vlanID)
}

// DeleteVlanInterface removes an 802.1Q VLAN subinterface.
// Implementation is platform-specific (vlan_linux.go, vlan_darwin.go).
func DeleteVlanInterface(parentIface string, vlanID int) error {
	return deleteVlanInterfacePlatform(parentIface, vlanID)
}

// contains checks if a slice contains a value.
func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
