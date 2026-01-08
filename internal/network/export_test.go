package network

import (
	"sync"
	"time"
)

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

// SetLastState sets the lastState field for testing.
func (h *LinkMonitorTestHelper) SetLastState(state LinkState) {
	h.M.mu.Lock()
	defer h.M.mu.Unlock()
	h.M.lastState = state
}

// AddHistoryEvent adds an event to history for testing.
func (h *LinkMonitorTestHelper) AddHistoryEvent(event LinkEvent) {
	h.M.mu.Lock()
	defer h.M.mu.Unlock()
	h.M.history = append(h.M.history, event)
	if len(h.M.history) > h.M.maxHistory {
		h.M.history = h.M.history[1:]
	}
}

// GetHistory returns the history slice for testing.
func (h *LinkMonitorTestHelper) GetHistory() []LinkEvent {
	h.M.mu.RLock()
	defer h.M.mu.RUnlock()
	result := make([]LinkEvent, len(h.M.history))
	copy(result, h.M.history)
	return result
}

// SetMaxHistory sets the maxHistory field for testing.
func (h *LinkMonitorTestHelper) SetMaxHistory(maxSize int) {
	h.M.mu.Lock()
	defer h.M.mu.Unlock()
	h.M.maxHistory = maxSize
}

// GetMinCallbackGap returns the minCallbackGap field for testing.
func (h *LinkMonitorTestHelper) GetMinCallbackGap() int64 {
	h.M.mu.RLock()
	defer h.M.mu.RUnlock()
	return int64(h.M.minCallbackGap)
}

// SetMinCallbackGap sets the minCallbackGap field for testing.
func (h *LinkMonitorTestHelper) SetMinCallbackGap(gap int64) {
	h.M.mu.Lock()
	defer h.M.mu.Unlock()
	h.M.minCallbackGap = time.Duration(gap)
}

// GetLastCallbackTime returns the lastCallbackTime field for testing.
func (h *LinkMonitorTestHelper) GetLastCallbackTime() time.Time {
	h.M.mu.RLock()
	defer h.M.mu.RUnlock()
	return h.M.lastCallbackTime
}

// SetLastCallbackTime sets the lastCallbackTime field for testing.
func (h *LinkMonitorTestHelper) SetLastCallbackTime(t time.Time) {
	h.M.mu.Lock()
	defer h.M.mu.Unlock()
	h.M.lastCallbackTime = t
}

// ProcessStateChange exposes processStateChange for testing.
func (h *LinkMonitorTestHelper) ProcessStateChange(newState LinkState) (bool, bool) {
	result, changed := h.M.processStateChange(newState)
	return result.shouldNotify, changed
}

// RecordEvent exposes recordEvent for testing.
func (h *LinkMonitorTestHelper) RecordEvent(event LinkEvent) {
	h.M.mu.Lock()
	defer h.M.mu.Unlock()
	h.M.recordEvent(event)
}

// SafeInvokeCallback exposes safeInvokeCallback for testing.
func (h *LinkMonitorTestHelper) SafeInvokeCallback(callback LinkStateCallback, event LinkEvent) {
	h.M.safeInvokeCallback(callback, event)
}

// NotifyCallbacks exposes notifyCallbacks for testing.
func (h *LinkMonitorTestHelper) NotifyCallbacks(event LinkEvent, callbacks []LinkStateCallback) {
	h.M.notifyCallbacks(event, callbacks)
}

// CheckAndUpdateRateLimit exposes checkAndUpdateRateLimit for testing.
func (h *LinkMonitorTestHelper) CheckAndUpdateRateLimit() bool {
	h.M.mu.Lock()
	defer h.M.mu.Unlock()
	return h.M.checkAndUpdateRateLimit()
}

// CheckLinkState exposes checkLinkState for testing.
func (h *LinkMonitorTestHelper) CheckLinkState() LinkState {
	return h.M.checkLinkState()
}

// CreateInterfaceCandidates creates an interfaceCandidates struct for testing.
func CreateInterfaceCandidates(ethernetWithIP, wifiWithIP, ethernetUp, wifiUp []string) *interfaceCandidates {
	return &interfaceCandidates{
		ethernetWithIP: ethernetWithIP,
		wifiWithIP:     wifiWithIP,
		ethernetUp:     ethernetUp,
		wifiUp:         wifiUp,
	}
}

// SelectBest exposes selectBest for testing.
func (c *interfaceCandidates) SelectBest() string {
	return c.selectBest()
}
