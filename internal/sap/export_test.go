package sap

// This file is only compiled during testing.
// It exports internal functions for testing purposes.

import "time"

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
