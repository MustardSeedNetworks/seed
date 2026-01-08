package link

import "time"

// ExportCheckLinkState exposes checkLinkState for testing.
func ExportCheckLinkState(interfaceName string) State {
	return checkLinkState(interfaceName)
}

// ExportCheckLinkStatePlatform exposes checkLinkStatePlatform for testing.
func ExportCheckLinkStatePlatform(interfaceName string) State {
	return checkLinkStatePlatform(interfaceName)
}

// ExportGetSpeedDuplex exposes getSpeedDuplex for testing.
func ExportGetSpeedDuplex(interfaceName string) (Speed, Duplex) {
	return getSpeedDuplex(interfaceName)
}

// ExportIsPhysicalInterfacePlatform exposes isPhysicalInterfacePlatform for testing.
func ExportIsPhysicalInterfacePlatform(name string) bool {
	return isPhysicalInterfacePlatform(name)
}

// ExportParseSpeedPlatform exposes parseSpeedPlatform for testing.
func ExportParseSpeedPlatform(s string) Speed {
	return parseSpeedPlatform(s)
}

// MonitorInterfaceName returns the interface name for testing.
func (m *Monitor) MonitorInterfaceName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.interfaceName
}

// MonitorState returns the current state for testing.
func (m *Monitor) MonitorState() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// MonitorCallbackCount returns the number of registered callbacks for testing.
func (m *Monitor) MonitorCallbackCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.callbacks)
}

// MonitorHistoryLen returns the history length for testing.
func (m *Monitor) MonitorHistoryLen() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.history)
}

// MonitorMaxHistory returns the max history setting for testing.
func (m *Monitor) MonitorMaxHistory() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.maxHistory
}

// MonitorPollInterval returns the poll interval for testing.
func (m *Monitor) MonitorPollInterval() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pollInterval.Milliseconds()
}

// SetState sets the state directly for testing.
func (m *Monitor) SetState(state State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
}

// AddHistoryEvent adds an event to history for testing.
func (m *Monitor) AddHistoryEvent(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = append(m.history, event)
	if len(m.history) > m.maxHistory {
		m.history = m.history[1:]
	}
}

// ExportCheckAndNotify exposes checkAndNotify for testing.
func (m *Monitor) ExportCheckAndNotify() {
	m.checkAndNotify()
}

// SetPollInterval sets the poll interval for testing.
func (m *Monitor) SetPollInterval(ms int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollInterval = time.Duration(ms) * time.Millisecond
}
