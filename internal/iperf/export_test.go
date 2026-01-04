// Package iperf exports internal items for testing.
// This file is only compiled during testing.
package iperf

// IperfBinaryPath returns the current cached iperf binary path for testing.
func IperfBinaryPath() string {
	return iperfBinaryPath
}

// SetIperfBinaryPath sets the cached iperf binary path for testing.
func SetIperfBinaryPath(path string) {
	iperfBinaryPath = path
}

// FindIperf3Binary exposes the internal findIperf3Binary function for testing.
func FindIperf3Binary() (string, error) {
	return findIperf3Binary()
}

// CompareVersions exposes the internal compareVersions function for testing.
func CompareVersions(v1, v2 string) int {
	return compareVersions(v1, v2)
}

// MinSupportedVersion exposes the minimum supported version constant for testing.
const MinSupportedVersion = minSupportedVersion

// IperfJSON exposes the internal iperfJSON type for testing.
type IperfJSON = iperfJSON

// SetManagerClientStatusRunning sets the client status running flag for testing.
func (m *Manager) SetManagerClientStatusRunning(running bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientStatus.Running = running
}

// SetManagerServerStatusRunning sets the server status running flag for testing.
func (m *Manager) SetManagerServerStatusRunning(running bool, port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serverStatus.Running = running
	m.serverStatus.Port = port
}

// SetManagerLastResult sets the last result for testing.
func (m *Manager) SetManagerLastResult(result *Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastResult = result
}
