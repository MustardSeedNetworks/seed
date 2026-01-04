package dhcp

import "time"

// ExtractValue exports extractValue for testing.
func ExtractValue(line string) string {
	return extractValue(line)
}

// FindDHCPMessageType exports findDHCPMessageType for testing.
func FindDHCPMessageType(options []byte) byte {
	return findDHCPMessageType(options)
}

// GetLeaseInfoDarwin exports getLeaseInfoDarwin for testing.
func GetLeaseInfoDarwin(interfaceName string) (*LeaseInfo, error) {
	return getLeaseInfoDarwin(interfaceName)
}

// GetLeaseInfoLinux exports getLeaseInfoLinux for testing.
func GetLeaseInfoLinux(interfaceName string) (*LeaseInfo, error) {
	return getLeaseInfoLinux(interfaceName)
}

// ParseDHClientLeaseFile exports parseDHClientLeaseFile for testing.
func ParseDHClientLeaseFile(path, interfaceName string) *LeaseInfo {
	return parseDHClientLeaseFile(path, interfaceName)
}

// ParseNMLeaseFile exports parseNMLeaseFile for testing.
func ParseNMLeaseFile(path string) *LeaseInfo {
	return parseNMLeaseFile(path)
}

// ParseNetworkdLeaseFile exports parseNetworkdLeaseFile for testing.
func ParseNetworkdLeaseFile(path string) *LeaseInfo {
	return parseNetworkdLeaseFile(path)
}

// MonitorInterfaceName returns the interface name for testing.
func (m *Monitor) MonitorInterfaceName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.interfaceName
}

// MonitorRunning returns the running state for testing.
func (m *Monitor) MonitorRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// MonitorTransactions returns the transactions map for testing.
func (m *Monitor) MonitorTransactions() map[uint32]*Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.transactions
}

// MonitorLastTiming returns the lastTiming field for testing.
func (m *Monitor) MonitorLastTiming() *Timing {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastTiming
}

// CalculateTiming exports calculateTiming for testing.
func (m *Monitor) CalculateTiming(tx *Transaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calculateTiming(tx)
}

// AddTransaction adds a transaction for testing.
func (m *Monitor) AddTransaction(tx *Transaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions[tx.XID] = tx
}

// GetTransaction gets a transaction for testing.
func (m *Monitor) GetTransaction(xid uint32) (*Transaction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tx, exists := m.transactions[xid]
	return tx, exists
}

// TransactionExists checks if a transaction exists for testing.
func (m *Monitor) TransactionExists(xid uint32) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.transactions[xid]
	return exists
}

// RogueDetectorConfig returns the config for testing.
func (rd *RogueDetector) RogueConfig() *RogueDetectorConfig {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return rd.config
}

// RogueKnownServerSet returns the knownServerSet for testing.
func (rd *RogueDetector) RogueKnownServerSet() map[string]bool {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return rd.knownServerSet
}

// RogueRunning returns the running state for testing.
func (rd *RogueDetector) RogueRunning() bool {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return rd.running
}

// SetRogueRunning sets the running state for testing.
func (rd *RogueDetector) SetRogueRunning(running bool) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.running = running
}

// AddDetectedServer adds a server to the detected servers map for testing.
func (rd *RogueDetector) AddDetectedServer(server *RogueServer) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.detectedServers[server.IP] = server
}

// GetDetectedServer gets a server from the detected servers map for testing.
func (rd *RogueDetector) GetDetectedServer(ip string) (*RogueServer, bool) {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	server, exists := rd.detectedServers[ip]
	return server, exists
}

// RecordPhaseForTest is a test helper that records a phase and returns the transaction state.
func (m *Monitor) RecordPhaseForTest(xid uint32, phase Phase, timestamp time.Time) {
	m.RecordPhase(xid, phase, timestamp)
}
