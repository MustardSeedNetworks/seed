package dhcp

import (
	"time"

	"github.com/gopacket/gopacket/layers"
)

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

// IsDHCPPort exports isDHCPPort for testing.
func IsDHCPPort(port uint16) bool {
	return isDHCPPort(layers.UDPPort(port))
}

// MsgTypeToPhase exports msgTypeToPhase for testing.
func MsgTypeToPhase(msgType byte) (Phase, bool) {
	return msgTypeToPhase(msgType)
}

// HexToIP exports hexToIP for testing.
func HexToIP(hexStr string) string {
	return hexToIP(hexStr)
}

// ExtractPlistIP exports extractPlistIP for testing.
func ExtractPlistIP(content, key string) string {
	return extractPlistIP(content, key)
}

// ExtractPlistInteger exports extractPlistInteger for testing.
func ExtractPlistInteger(content, key string) int {
	return extractPlistInteger(content, key)
}

// ExtractPlistIPArray exports extractPlistIPArray for testing.
func ExtractPlistIPArray(content, key string) []string {
	return extractPlistIPArray(content, key)
}

// ExtractArrayContent exports extractArrayContent for testing.
func ExtractArrayContent(remaining string) (string, bool) {
	return extractArrayContent(remaining)
}

// ParseDataTagsFromArray exports parseDataTagsFromArray for testing.
func ParseDataTagsFromArray(arrayContent string) []string {
	return parseDataTagsFromArray(arrayContent)
}

// ParseDHClientLeaseLine exports parseDHClientLeaseLine for testing.
func ParseDHClientLeaseLine(line string, info *LeaseInfo) {
	parseDHClientLeaseLine(line, info)
}

// ParseLeaseLineServer exports parseLeaseLineServer for testing.
func ParseLeaseLineServer(line, serverKey string) (string, bool) {
	return parseLeaseLineServer(line, serverKey)
}

// ParseLeaseLineRouter exports parseLeaseLineRouter for testing.
func ParseLeaseLineRouter(line, routerKey string) (string, bool) {
	return parseLeaseLineRouter(line, routerKey)
}

// ParseLeaseLineTime exports parseLeaseLineTime for testing.
func ParseLeaseLineTime(line, leaseTimeKey string) (int, bool) {
	return parseLeaseLineTime(line, leaseTimeKey)
}

// ParseLeaseLineDNS exports parseLeaseLineDNS for testing.
func ParseLeaseLineDNS(line, dnsKey string) []string {
	return parseLeaseLineDNS(line, dnsKey)
}

// ProcessLeaseLine exports processLeaseLine for testing.
func ProcessLeaseLine(line string, mapping LeaseFieldMapping, info *LeaseInfo) {
	processLeaseLine(line, leaseFieldMapping{
		serverKey:    mapping.ServerKey,
		routerKey:    mapping.RouterKey,
		leaseTimeKey: mapping.LeaseTimeKey,
		dnsKey:       mapping.DNSKey,
	}, info)
}

// LeaseFieldMapping is an exported type for testing processLeaseLine.
type LeaseFieldMapping struct {
	ServerKey    string
	RouterKey    string
	LeaseTimeKey string
	DNSKey       string
}

// ParseLeaseFileWithMapping exports parseLeaseFileWithMapping for testing.
func ParseLeaseFileWithMapping(path string, mapping LeaseFieldMapping) *LeaseInfo {
	return parseLeaseFileWithMapping(path, leaseFieldMapping{
		serverKey:    mapping.ServerKey,
		routerKey:    mapping.RouterKey,
		leaseTimeKey: mapping.LeaseTimeKey,
		dnsKey:       mapping.DNSKey,
	})
}

// ParseDarwinLeaseFile exports parseDarwinLeaseFile for testing.
func ParseDarwinLeaseFile(path string) *LeaseInfo {
	return parseDarwinLeaseFile(path)
}

// GetDHCPMessageType exports getDHCPMessageType for testing on RogueDetector.
// This is a placeholder - we can't easily export this without the DHCPv4 type.
func (rd *RogueDetector) GetDHCPMessageType(_ any) byte {
	return 0
}

// SetDetectedServers sets the detected servers map directly for testing.
func (rd *RogueDetector) SetDetectedServers(servers map[string]*RogueServer) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.detectedServers = servers
}

// DetectedServersCount returns the count of detected servers for testing.
func (rd *RogueDetector) DetectedServersCount() int {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return len(rd.detectedServers)
}

// PruneExpiredServers exports pruneExpiredServers for testing.
func (rd *RogueDetector) PruneExpiredServers(now time.Time) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.pruneExpiredServers(now)
}

// AddNewServer exports addNewServer for testing.
func (rd *RogueDetector) AddNewServer(serverIP, serverMAC string, now time.Time) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.addNewServer(serverIP, serverMAC, now)
}

// UpdateExistingServer exports updateExistingServer for testing.
func (rd *RogueDetector) UpdateExistingServer(server *RogueServer, serverMAC string, now time.Time) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.updateExistingServer(server, serverMAC, now)
}

// RecordDetectedServer exports recordDetectedServer for testing.
func (rd *RogueDetector) RecordDetectedServer(serverIP, serverMAC string) {
	rd.recordDetectedServer(serverIP, serverMAC)
}

// StopLocked exports stopLocked for testing (used for partial coverage).
func (rd *RogueDetector) StopLocked() {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.stopLocked()
}

// StartLocked exports startLocked for testing.
func (rd *RogueDetector) StartLocked() error {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	return rd.startLocked()
}

// GetServerIdentifier exports getServerIdentifier for testing.
func (rd *RogueDetector) ExportGetServerIdentifier(dhcp *layers.DHCPv4) string {
	return rd.getServerIdentifier(dhcp)
}

// ExportGetDHCPMessageType exports getDHCPMessageType for testing.
func (rd *RogueDetector) ExportGetDHCPMessageType(dhcp *layers.DHCPv4) layers.DHCPMsgType {
	return rd.getDHCPMessageType(dhcp)
}

// MonitorStop exports Stop without lock for testing internal behavior.
func (m *Monitor) MonitorStopLocked() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}

	m.running = false
	cleanupDone := m.cleanupDone
	m.cleanupDone = nil

	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}

	if m.handle != nil {
		m.handle.Close()
		m.handle = nil
	}

	m.mu.Unlock()

	if cleanupDone != nil {
		<-cleanupDone
	}
}

// MockDHCPOption represents a DHCP option for testing.
type MockDHCPOption struct {
	Type byte
	Data []byte
}

// MockDHCPv4 is a mock DHCP packet for testing.
type MockDHCPv4 struct {
	Options []MockDHCPOption
}

// ToLayers converts MockDHCPv4 to a layers.DHCPv4 for testing.
func (m *MockDHCPv4) ToLayers() *layers.DHCPv4 {
	dhcp := &layers.DHCPv4{
		Options: make(layers.DHCPOptions, len(m.Options)),
	}
	for i, opt := range m.Options {
		dhcp.Options[i] = layers.DHCPOption{
			Type:   layers.DHCPOpt(opt.Type),
			Length: uint8(len(opt.Data)),
			Data:   opt.Data,
		}
	}
	return dhcp
}
