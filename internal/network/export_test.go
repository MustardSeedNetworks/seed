// Package network exports internal functions for testing.
package network

import "sync"

// DetectInterfaceType exposes detectInterfaceType for testing.
func DetectInterfaceType(name string) InterfaceType {
	return detectInterfaceType(name)
}

// HasRoutableAddress exposes hasRoutableAddress for testing.
func HasRoutableAddress(addresses []string) bool {
	return hasRoutableAddress(addresses)
}

// ValidateIPConfig exposes validateIPConfig for testing.
func ValidateIPConfig(cfg *StaticIPConfig) error {
	return validateIPConfig(cfg)
}

// IsValidNetmask exposes isValidNetmask for testing.
func IsValidNetmask(netmask string) bool {
	return isValidNetmask(netmask)
}

// CIDRToNetmask exposes cidrToNetmask for testing.
func CIDRToNetmask(prefix int) string {
	return cidrToNetmask(prefix)
}

// ManagerTestHelper provides access to Manager internals for testing.
type ManagerTestHelper struct {
	M *Manager
}

// NewManagerTestHelper creates a new test helper.
func NewManagerTestHelper(m *Manager) *ManagerTestHelper {
	return &ManagerTestHelper{M: m}
}

// GetCurrentInterface returns the currentInterface field.
func (h *ManagerTestHelper) GetCurrentInterface() string {
	return h.M.currentInterface
}

// GetInterfaces returns the interfaces map.
func (h *ManagerTestHelper) GetInterfaces() map[string]*InterfaceInfo {
	return h.M.interfaces
}

// GetMutex returns the mutex for direct locking in tests.
func (h *ManagerTestHelper) GetMutex() *sync.RWMutex {
	return &h.M.mu
}

// SetInterface adds an interface directly to the map (for testing).
func (h *ManagerTestHelper) SetInterface(name string, info *InterfaceInfo) {
	h.M.mu.Lock()
	defer h.M.mu.Unlock()
	h.M.interfaces[name] = info
}

// LinkMonitorTestHelper provides access to LinkMonitor internals for testing.
type LinkMonitorTestHelper struct {
	M *LinkMonitor
}

// NewLinkMonitorTestHelper creates a new test helper.
func NewLinkMonitorTestHelper(m *LinkMonitor) *LinkMonitorTestHelper {
	return &LinkMonitorTestHelper{M: m}
}

// GetInterfaceName returns the interfaceName field.
func (h *LinkMonitorTestHelper) GetInterfaceName() string {
	return h.M.interfaceName
}

// GetLastState returns the lastState field.
func (h *LinkMonitorTestHelper) GetLastState() LinkState {
	return h.M.lastState
}

// GetPollInterval returns the pollInterval field.
func (h *LinkMonitorTestHelper) GetPollInterval() int64 {
	return int64(h.M.pollInterval)
}

// GetMaxHistory returns the maxHistory field.
func (h *LinkMonitorTestHelper) GetMaxHistory() int {
	return h.M.maxHistory
}

// GetCallbacksCount returns the number of registered callbacks.
func (h *LinkMonitorTestHelper) GetCallbacksCount() int {
	h.M.mu.RLock()
	defer h.M.mu.RUnlock()
	return len(h.M.callbacks)
}

// IsRunning returns whether the monitor is running.
func (h *LinkMonitorTestHelper) IsRunning() bool {
	h.M.mu.RLock()
	defer h.M.mu.RUnlock()
	return h.M.running
}

// CreateManagerWithInterfaces creates a Manager with custom interfaces for testing.
func CreateManagerWithInterfaces(interfaces map[string]*InterfaceInfo) *Manager {
	return &Manager{
		interfaces: interfaces,
	}
}
