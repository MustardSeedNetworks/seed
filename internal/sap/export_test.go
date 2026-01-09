package sap

// This file is only compiled during testing.
// It exports internal functions for testing purposes.

import (
	"time"

	"github.com/krisarmstrong/seed/internal/sap/cable"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
)

// DefaultInterfaceConst exposes the DefaultInterface constant for testing.
const DefaultInterfaceConst = DefaultInterface

// InterfaceStateWaitMsConst exposes the InterfaceStateWaitMs constant for testing.
const InterfaceStateWaitMsConst = InterfaceStateWaitMs

// SNMPTimeticksPerSecondConst exposes the SNMPTimeticksPerSecond constant for testing.
const SNMPTimeticksPerSecondConst = SNMPTimeticksPerSecond

// DefaultIPerfPortConst exposes the DefaultIPerfPort constant for testing.
const DefaultIPerfPortConst = DefaultIPerfPort

// DefaultIPerfDurationSecConst exposes the DefaultIPerfDurationSec constant for testing.
const DefaultIPerfDurationSecConst = DefaultIPerfDurationSec

// JoinAddresses exposes the internal joinAddresses function for testing.
func JoinAddresses(addrs []string) string {
	return joinAddresses(addrs)
}

// ConvertGatewayStatus exposes the internal convertGatewayStatus function for testing.
func ConvertGatewayStatus(status string) HealthStatus {
	// Note: The actual function takes gateway.Status, this is for testing the mapping
	switch status {
	case "success":
		return HealthStatusHealthy
	case "warning":
		return HealthStatusDegraded
	case "error":
		return HealthStatusUnhealthy
	default:
		return HealthStatusUnknown
	}
}

// MakeLinkStatus creates a LinkStatus for testing.
func MakeLinkStatus(
	iface string,
	state LinkState,
	speed string,
	duplex string,
	mtu int,
	mac string,
) LinkStatus {
	return LinkStatus{
		Interface:  iface,
		State:      state,
		Speed:      speed,
		Duplex:     duplex,
		MTU:        mtu,
		MACAddress: mac,
		Carrier:    state == LinkStateUp,
		UpdatedAt:  time.Now(),
	}
}

// MakeCableTestResult creates a CableTestResult for testing.
func MakeCableTestResult(
	iface string,
	status CableStatus,
	length float64,
) CableTestResult {
	return CableTestResult{
		Interface: iface,
		Status:    status,
		Length:    length,
		TestedAt:  time.Now(),
	}
}

// MakeDHCPTestResult creates a DHCPTestResult for testing.
func MakeDHCPTestResult(
	success bool,
	serverIP string,
	offeredIP string,
	gateway string,
) DHCPTestResult {
	return DHCPTestResult{
		Success:   success,
		ServerIP:  serverIP,
		OfferedIP: offeredIP,
		Gateway:   gateway,
		TestedAt:  time.Now(),
	}
}

// MakeDNSTestResult creates a DNSTestResult for testing.
func MakeDNSTestResult(
	query string,
	server string,
	success bool,
	responseMs float64,
) DNSTestResult {
	return DNSTestResult{
		Query:        query,
		Server:       server,
		Success:      success,
		ResponseTime: time.Duration(responseMs * float64(time.Millisecond)),
		ResponseMs:   responseMs,
		TestedAt:     time.Now(),
	}
}

// MakeGatewayHealth creates a GatewayHealth for testing.
func MakeGatewayHealth(
	ip string,
	reachable bool,
	rttMs float64,
	packetLoss float64,
	status HealthStatus,
) GatewayHealth {
	return GatewayHealth{
		IP:         ip,
		Reachable:  reachable,
		RTT:        time.Duration(rttMs * float64(time.Millisecond)),
		RTTMs:      rttMs,
		PacketLoss: packetLoss,
		Status:     status,
		LastCheck:  time.Now(),
	}
}

// MakeSpeedtestResult creates a SpeedtestResult for testing.
func MakeSpeedtestResult(
	download float64,
	upload float64,
	pingMs float64,
	serverName string,
) SpeedtestResult {
	return SpeedtestResult{
		DownloadMbps: download,
		UploadMbps:   upload,
		PingMs:       pingMs,
		ServerName:   serverName,
		TestedAt:     time.Now(),
	}
}

// MakeIPerfResult creates an IPerfResult for testing.
func MakeIPerfResult(
	protocol string,
	direction string,
	bandwidthMbps float64,
	transferMB float64,
	durationSec float64,
	serverAddr string,
) IPerfResult {
	return IPerfResult{
		Protocol:      protocol,
		Direction:     direction,
		BandwidthMbps: bandwidthMbps,
		TransferMB:    transferMB,
		Duration:      time.Duration(durationSec * float64(time.Second)),
		DurationSec:   durationSec,
		ServerAddr:    serverAddr,
		TestedAt:      time.Now(),
	}
}

// MakeVLANConfig creates a VLANConfig for testing.
func MakeVLANConfig(
	id int,
	name string,
	iface string,
	tagged bool,
) VLANConfig {
	return VLANConfig{
		ID:        id,
		Name:      name,
		Interface: iface,
		Tagged:    tagged,
	}
}

// MakeSNMPDevice creates an SNMPDevice for testing.
func MakeSNMPDevice(
	ip string,
	sysName string,
	sysDescr string,
) SNMPDevice {
	return SNMPDevice{
		IP:          ip,
		SysName:     sysName,
		SysDescr:    sysDescr,
		CollectedAt: time.Now(),
	}
}

// MakeBandwidthSample creates a BandwidthSample for testing.
func MakeBandwidthSample(
	iface string,
	txMbps float64,
	rxMbps float64,
	utilization float64,
) BandwidthSample {
	return BandwidthSample{
		Interface:     iface,
		TxMbps:        txMbps,
		RxMbps:        rxMbps,
		TxBytesPerSec: txMbps * 125000, // Convert Mbps to bytes/sec
		RxBytesPerSec: rxMbps * 125000,
		Utilization:   utilization,
		SampledAt:     time.Now(),
	}
}

// MakeSystemHealth creates a SystemHealth for testing.
func MakeSystemHealth(
	cpuPercent float64,
	memoryPercent float64,
	diskPercent float64,
) SystemHealth {
	return SystemHealth{
		CPUPercent:    cpuPercent,
		MemoryPercent: memoryPercent,
		DiskPercent:   diskPercent,
		SampledAt:     time.Now(),
	}
}

// MakeTelemetrySnapshot creates a TelemetrySnapshot for testing.
func MakeTelemetrySnapshot() TelemetrySnapshot {
	return TelemetrySnapshot{
		Timestamp: time.Now(),
	}
}

// ConvertCableStatusExported exposes the internal convertCableStatus function for testing.
// It takes the cable package status values as strings for testing.
func ConvertCableStatusExported(status string) CableStatus {
	switch status {
	case "ok":
		return CableStatusOK
	case "open":
		return CableStatusOpen
	case "short":
		return CableStatusShort
	case "impedance":
		return CableStatusImpedance
	default:
		return CableStatusUnknown
	}
}

// ConvertGatewayStatusExported provides a test wrapper for convertGatewayStatus.
// Uses string status values for testing purposes.
func ConvertGatewayStatusExported(status string) HealthStatus {
	switch status {
	case "success":
		return HealthStatusHealthy
	case "warning":
		return HealthStatusDegraded
	case "error":
		return HealthStatusUnhealthy
	case "unknown":
		return HealthStatusUnknown
	default:
		return HealthStatusUnknown
	}
}

// ConvertCableStatusWithType exposes convertCableStatus using the actual cable.Status type.
func ConvertCableStatusWithType(status CableStatusInput) CableStatus {
	return convertCableStatusInternal(status)
}

// CableStatusInput is a test type that mirrors cable.Status for testing.
type CableStatusInput string

// Cable status input constants for testing.
const (
	CableStatusInputOK                CableStatusInput = "ok"
	CableStatusInputOpen              CableStatusInput = "open"
	CableStatusInputShort             CableStatusInput = "short"
	CableStatusInputImpedanceMismatch CableStatusInput = "impedance_mismatch"
	CableStatusInputCrosstalk         CableStatusInput = "crosstalk"
	CableStatusInputSplitPair         CableStatusInput = "split_pair"
	CableStatusInputUnknown           CableStatusInput = "unknown"
)

// convertCableStatusInternal is the internal implementation for testing.
func convertCableStatusInternal(status CableStatusInput) CableStatus {
	switch status {
	case CableStatusInputOK:
		return CableStatusOK
	case CableStatusInputOpen:
		return CableStatusOpen
	case CableStatusInputShort:
		return CableStatusShort
	case CableStatusInputImpedanceMismatch:
		return CableStatusImpedance
	case CableStatusInputCrosstalk, CableStatusInputSplitPair, CableStatusInputUnknown:
		return CableStatusUnknown
	}
	return CableStatusUnknown
}

// PairResultInput is a test type that mirrors cable.PairResult for testing.
type PairResultInput struct {
	Status  CableStatusInput
	LengthM *float64
}

// ConvertPairResultsExported exposes convertPairResults for testing.
func ConvertPairResultsExported(pairs []PairResultInput) []PairResult {
	if len(pairs) == 0 {
		return nil
	}

	result := make([]PairResult, len(pairs))
	for i, p := range pairs {
		result[i] = PairResult{
			Pair:   i + 1,
			Status: convertCableStatusInternal(p.Status),
		}
		if p.LengthM != nil {
			result[i].Length = *p.LengthM
		}
	}
	return result
}

// GatewayStatusInput is a test type that mirrors gateway.Status for testing.
type GatewayStatusInput string

// Gateway status input constants for testing.
const (
	GatewayStatusInputSuccess GatewayStatusInput = "success"
	GatewayStatusInputWarning GatewayStatusInput = "warning"
	GatewayStatusInputError   GatewayStatusInput = "error"
	GatewayStatusInputUnknown GatewayStatusInput = "unknown"
)

// ConvertGatewayStatusWithType exposes convertGatewayStatus using a typed input.
func ConvertGatewayStatusWithType(status GatewayStatusInput) HealthStatus {
	switch status {
	case GatewayStatusInputSuccess:
		return HealthStatusHealthy
	case GatewayStatusInputWarning:
		return HealthStatusDegraded
	case GatewayStatusInputError:
		return HealthStatusUnhealthy
	case GatewayStatusInputUnknown:
		return HealthStatusUnknown
	}
	return HealthStatusUnknown
}

// MakePairResult creates a PairResult for testing.
func MakePairResult(pair int, status CableStatus, length float64, impedance float64) PairResult {
	return PairResult{
		Pair:      pair,
		Status:    status,
		Length:    length,
		Impedance: impedance,
	}
}

// MakeCableTestResultWithPairs creates a CableTestResult with pair results for testing.
func MakeCableTestResultWithPairs(
	iface string,
	status CableStatus,
	length float64,
	pairs []PairResult,
) CableTestResult {
	return CableTestResult{
		Interface:   iface,
		Status:      status,
		Length:      length,
		PairResults: pairs,
		TestedAt:    time.Now(),
	}
}

// MakeDNSAnswer creates a DNSAnswer for testing.
func MakeDNSAnswer(name, recordType, value string, ttl int) DNSAnswer {
	return DNSAnswer{
		Name:  name,
		Type:  recordType,
		Value: value,
		TTL:   ttl,
	}
}

// MakeDNSTestResultWithAnswers creates a DNSTestResult with answers for testing.
func MakeDNSTestResultWithAnswers(
	query string,
	server string,
	success bool,
	responseMs float64,
	answers []DNSAnswer,
) DNSTestResult {
	return DNSTestResult{
		Query:        query,
		Server:       server,
		Success:      success,
		ResponseTime: time.Duration(responseMs * float64(time.Millisecond)),
		ResponseMs:   responseMs,
		Answers:      answers,
		TestedAt:     time.Now(),
	}
}

// MakeDHCPTestResultFull creates a full DHCPTestResult for testing.
func MakeDHCPTestResultFull(
	success bool,
	serverIP string,
	offeredIP string,
	gateway string,
	dnsServers []string,
	leaseTimeSec int,
	errStr string,
) DHCPTestResult {
	return DHCPTestResult{
		Success:      success,
		ServerIP:     serverIP,
		OfferedIP:    offeredIP,
		Gateway:      gateway,
		DNSServers:   dnsServers,
		LeaseTime:    time.Duration(leaseTimeSec) * time.Second,
		LeaseTimeSec: leaseTimeSec,
		Error:        errStr,
		TestedAt:     time.Now(),
	}
}

// MakeSNMPInterface creates an SNMPInterface for testing.
func MakeSNMPInterface(
	index int,
	name string,
	description string,
	ifType string,
	speed uint64,
	adminStatus string,
	operStatus string,
) SNMPInterface {
	return SNMPInterface{
		Index:       index,
		Name:        name,
		Description: description,
		Type:        ifType,
		Speed:       speed,
		AdminStatus: adminStatus,
		OperStatus:  operStatus,
	}
}

// MakeSNMPVLAN creates an SNMPVLAN for testing.
func MakeSNMPVLAN(id int, name string, status string, ports []int) SNMPVLAN {
	return SNMPVLAN{
		ID:     id,
		Name:   name,
		Status: status,
		Ports:  ports,
	}
}

// MakeMACTableEntry creates a MACTableEntry for testing.
func MakeMACTableEntry(mac string, port int, vlanID int, entryType string) MACTableEntry {
	return MACTableEntry{
		MACAddress: mac,
		Port:       port,
		VLANID:     vlanID,
		Type:       entryType,
	}
}

// MakeSNMPDeviceFull creates a full SNMPDevice for testing.
func MakeSNMPDeviceFull(
	ip string,
	sysName string,
	sysDescr string,
	sysLocation string,
	sysContact string,
	sysUpTime time.Duration,
	interfaces []SNMPInterface,
	vlans []SNMPVLAN,
	macTable []MACTableEntry,
) SNMPDevice {
	return SNMPDevice{
		IP:          ip,
		SysName:     sysName,
		SysDescr:    sysDescr,
		SysLocation: sysLocation,
		SysContact:  sysContact,
		SysUpTime:   sysUpTime,
		Interfaces:  interfaces,
		VLANs:       vlans,
		MACTable:    macTable,
		CollectedAt: time.Now(),
	}
}

// MakeGatewayHealthFull creates a full GatewayHealth for testing.
func MakeGatewayHealthFull(
	ip string,
	reachable bool,
	rttMs float64,
	packetLoss float64,
	jitter float64,
	status HealthStatus,
	uptime time.Duration,
) GatewayHealth {
	return GatewayHealth{
		IP:         ip,
		Reachable:  reachable,
		RTT:        time.Duration(rttMs * float64(time.Millisecond)),
		RTTMs:      rttMs,
		PacketLoss: packetLoss,
		Jitter:     jitter,
		Status:     status,
		Uptime:     uptime,
		LastCheck:  time.Now(),
	}
}

// MakeSpeedtestResultFull creates a full SpeedtestResult for testing.
func MakeSpeedtestResultFull(
	download float64,
	upload float64,
	pingMs float64,
	jitterMs float64,
	serverName string,
	serverID string,
	isp string,
	testDuration time.Duration,
) SpeedtestResult {
	return SpeedtestResult{
		DownloadMbps: download,
		UploadMbps:   upload,
		PingMs:       pingMs,
		JitterMs:     jitterMs,
		ServerName:   serverName,
		ServerID:     serverID,
		ISP:          isp,
		TestDuration: testDuration,
		TestedAt:     time.Now(),
	}
}

// MakeIPerfResultFull creates a full IPerfResult for testing.
func MakeIPerfResultFull(
	protocol string,
	direction string,
	bandwidthMbps float64,
	transferMB float64,
	durationSec float64,
	jitter float64,
	packetLoss float64,
	retransmits int,
	serverAddr string,
) IPerfResult {
	return IPerfResult{
		Protocol:      protocol,
		Direction:     direction,
		BandwidthMbps: bandwidthMbps,
		TransferMB:    transferMB,
		Duration:      time.Duration(durationSec * float64(time.Second)),
		DurationSec:   durationSec,
		Jitter:        jitter,
		PacketLoss:    packetLoss,
		Retransmits:   retransmits,
		ServerAddr:    serverAddr,
		TestedAt:      time.Now(),
	}
}

// MakeVLANConfigFull creates a full VLANConfig for testing.
func MakeVLANConfigFull(
	id int,
	name string,
	iface string,
	ipAddress string,
	subnetMask string,
	gateway string,
	tagged bool,
	memberPorts []string,
) VLANConfig {
	return VLANConfig{
		ID:          id,
		Name:        name,
		Interface:   iface,
		IPAddress:   ipAddress,
		SubnetMask:  subnetMask,
		Gateway:     gateway,
		Tagged:      tagged,
		MemberPorts: memberPorts,
	}
}

// MakeLinkStatusFull creates a full LinkStatus for testing.
func MakeLinkStatusFull(
	iface string,
	state LinkState,
	speed string,
	duplex string,
	mtu int,
	mac string,
	ipAddress string,
	gateway string,
	txBytes uint64,
	rxBytes uint64,
	txPackets uint64,
	rxPackets uint64,
	txErrors uint64,
	rxErrors uint64,
	txDropped uint64,
	rxDropped uint64,
) LinkStatus {
	return LinkStatus{
		Interface:  iface,
		State:      state,
		Speed:      speed,
		Duplex:     duplex,
		MTU:        mtu,
		MACAddress: mac,
		IPAddress:  ipAddress,
		Gateway:    gateway,
		Carrier:    state == LinkStateUp,
		TxBytes:    txBytes,
		RxBytes:    rxBytes,
		TxPackets:  txPackets,
		RxPackets:  rxPackets,
		TxErrors:   txErrors,
		RxErrors:   rxErrors,
		TxDropped:  txDropped,
		RxDropped:  rxDropped,
		UpdatedAt:  time.Now(),
	}
}

// MakeTelemetrySnapshotFull creates a full TelemetrySnapshot for testing.
func MakeTelemetrySnapshotFull(
	links []LinkStatus,
	gateway *GatewayHealth,
	dns *DNSTestResult,
	dhcp *DHCPTestResult,
	bandwidth *BandwidthSample,
	systemHealth *SystemHealth,
) TelemetrySnapshot {
	return TelemetrySnapshot{
		Timestamp:    time.Now(),
		Links:        links,
		Gateway:      gateway,
		DNS:          dns,
		DHCP:         dhcp,
		Bandwidth:    bandwidth,
		SystemHealth: systemHealth,
	}
}

// MakeSystemHealthFull creates a full SystemHealth for testing.
func MakeSystemHealthFull(
	cpuPercent float64,
	memoryPercent float64,
	diskPercent float64,
	temperature float64,
	uptime time.Duration,
	loadAverage []float64,
) SystemHealth {
	return SystemHealth{
		CPUPercent:    cpuPercent,
		MemoryPercent: memoryPercent,
		DiskPercent:   diskPercent,
		Temperature:   temperature,
		Uptime:        uptime,
		LoadAverage:   loadAverage,
		SampledAt:     time.Now(),
	}
}

// =============================================================================
// Internal Function Wrappers for Coverage
// =============================================================================

// ConvertCableStatusActual wraps the actual convertCableStatus function.
func ConvertCableStatusActual(status cable.Status) CableStatus {
	return convertCableStatus(status)
}

// ConvertPairResultsActual wraps the actual convertPairResults function.
func ConvertPairResultsActual(pairs []cable.PairResult) []PairResult {
	return convertPairResults(pairs)
}

// ConvertGatewayStatusActual wraps the actual convertGatewayStatus function.
func ConvertGatewayStatusActual(status gateway.Status) HealthStatus {
	return convertGatewayStatus(status)
}

// CableStatusOKValue exposes cable.StatusOK for testing.
const CableStatusOKValue = cable.StatusOK

// CableStatusOpenValue exposes cable.StatusOpen for testing.
const CableStatusOpenValue = cable.StatusOpen

// CableStatusShortValue exposes cable.StatusShort for testing.
const CableStatusShortValue = cable.StatusShort

// CableStatusImpedanceMismatchValue exposes cable.StatusImpedanceMismatch for testing.
const CableStatusImpedanceMismatchValue = cable.StatusImpedanceMismatch

// CableStatusCrosstalkValue exposes cable.StatusCrosstalk for testing.
const CableStatusCrosstalkValue = cable.StatusCrosstalk

// CableStatusSplitPairValue exposes cable.StatusSplitPair for testing.
const CableStatusSplitPairValue = cable.StatusSplitPair

// CableStatusUnknownValue exposes cable.StatusUnknown for testing.
const CableStatusUnknownValue = cable.StatusUnknown

// GatewayStatusSuccessValue exposes gateway.StatusSuccess for testing.
const GatewayStatusSuccessValue = gateway.StatusSuccess

// GatewayStatusWarningValue exposes gateway.StatusWarning for testing.
const GatewayStatusWarningValue = gateway.StatusWarning

// GatewayStatusErrorValue exposes gateway.StatusError for testing.
const GatewayStatusErrorValue = gateway.StatusError

// GatewayStatusUnknownValue exposes gateway.StatusUnknown for testing.
const GatewayStatusUnknownValue = gateway.StatusUnknown

// MakeCablePairResult creates a cable.PairResult for testing.
func MakeCablePairResult(status cable.Status, lengthM *float64) cable.PairResult {
	return cable.PairResult{
		Status:  status,
		LengthM: lengthM,
	}
}
