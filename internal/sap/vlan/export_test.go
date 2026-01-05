// Package vlan exports internal functions for testing.
package vlan

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
