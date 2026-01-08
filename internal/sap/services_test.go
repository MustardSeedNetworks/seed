// Package sap_test provides tests for the sap module's services and types.
// These tests cover the PerformanceService, type structures, and helper functions.
package sap_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap"
)

// =============================================================================
// Constants Tests
// =============================================================================

func TestDefaultInterfaceConstant(t *testing.T) {
	if sap.DefaultInterfaceConst != "eth0" {
		t.Errorf("expected DefaultInterface 'eth0', got %q", sap.DefaultInterfaceConst)
	}
}

func TestInterfaceStateWaitMsConstant(t *testing.T) {
	if sap.InterfaceStateWaitMsConst != 100 {
		t.Errorf("expected InterfaceStateWaitMs 100, got %d", sap.InterfaceStateWaitMsConst)
	}
}

func TestSNMPTimeticksPerSecondConstant(t *testing.T) {
	if sap.SNMPTimeticksPerSecondConst != 100 {
		t.Errorf("expected SNMPTimeticksPerSecond 100, got %d", sap.SNMPTimeticksPerSecondConst)
	}
}

func TestDefaultIPerfPortConstant(t *testing.T) {
	if sap.DefaultIPerfPortConst != 5201 {
		t.Errorf("expected DefaultIPerfPort 5201, got %d", sap.DefaultIPerfPortConst)
	}
}

func TestDefaultIPerfDurationSecConstant(t *testing.T) {
	if sap.DefaultIPerfDurationSecConst != 10 {
		t.Errorf("expected DefaultIPerfDurationSec 10, got %d", sap.DefaultIPerfDurationSecConst)
	}
}

// =============================================================================
// LinkState Tests
// =============================================================================

func TestLinkStateValues(t *testing.T) {
	tests := []struct {
		name     string
		state    sap.LinkState
		expected string
	}{
		{"up", sap.LinkStateUp, "up"},
		{"down", sap.LinkStateDown, "down"},
		{"dormant", sap.LinkStateDormant, "dormant"},
		{"unknown", sap.LinkStateUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.state)
			}
		})
	}
}

// =============================================================================
// CableStatus Tests
// =============================================================================

func TestCableStatusValues(t *testing.T) {
	tests := []struct {
		name     string
		status   sap.CableStatus
		expected string
	}{
		{"ok", sap.CableStatusOK, "ok"},
		{"open", sap.CableStatusOpen, "open"},
		{"short", sap.CableStatusShort, "short"},
		{"impedance", sap.CableStatusImpedance, "impedance_mismatch"},
		{"unknown", sap.CableStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.status)
			}
		})
	}
}

// =============================================================================
// HealthStatus Tests
// =============================================================================

func TestHealthStatusValues(t *testing.T) {
	tests := []struct {
		name     string
		status   sap.HealthStatus
		expected string
	}{
		{"healthy", sap.HealthStatusHealthy, "healthy"},
		{"degraded", sap.HealthStatusDegraded, "degraded"},
		{"unhealthy", sap.HealthStatusUnhealthy, "unhealthy"},
		{"unknown", sap.HealthStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.status)
			}
		})
	}
}

// =============================================================================
// LinkStatus Tests
// =============================================================================

func TestLinkStatusAllFields(t *testing.T) {
	now := time.Now()
	status := sap.LinkStatus{
		Interface:  "eth0",
		State:      sap.LinkStateUp,
		Speed:      "1000",
		Duplex:     "full",
		MTU:        1500,
		MACAddress: "00:11:22:33:44:55",
		IPAddress:  "192.168.1.100",
		Gateway:    "192.168.1.1",
		Carrier:    true,
		TxBytes:    1000000,
		RxBytes:    2000000,
		TxPackets:  1000,
		RxPackets:  2000,
		TxErrors:   0,
		RxErrors:   0,
		TxDropped:  0,
		RxDropped:  0,
		UpdatedAt:  now,
	}

	if status.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", status.Interface)
	}
	if status.State != sap.LinkStateUp {
		t.Errorf("expected State 'up', got %q", status.State)
	}
	if status.Speed != "1000" {
		t.Errorf("expected Speed '1000', got %q", status.Speed)
	}
	if status.Duplex != "full" {
		t.Errorf("expected Duplex 'full', got %q", status.Duplex)
	}
	if status.MTU != 1500 {
		t.Errorf("expected MTU 1500, got %d", status.MTU)
	}
	if status.MACAddress != "00:11:22:33:44:55" {
		t.Errorf("expected MACAddress '00:11:22:33:44:55', got %q", status.MACAddress)
	}
	if !status.Carrier {
		t.Error("expected Carrier true")
	}
	if status.TxBytes != 1000000 {
		t.Errorf("expected TxBytes 1000000, got %d", status.TxBytes)
	}
	if status.RxBytes != 2000000 {
		t.Errorf("expected RxBytes 2000000, got %d", status.RxBytes)
	}
}

func TestLinkStatusMakeHelper(t *testing.T) {
	status := sap.MakeLinkStatus("eth0", sap.LinkStateUp, "1000", "full", 1500, "00:11:22:33:44:55")

	if status.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", status.Interface)
	}
	if status.State != sap.LinkStateUp {
		t.Errorf("expected State 'up', got %q", status.State)
	}
	if !status.Carrier {
		t.Error("expected Carrier true for LinkStateUp")
	}
}

// =============================================================================
// CableTestResult Tests
// =============================================================================

func TestCableTestResultFields(t *testing.T) {
	now := time.Now()
	result := sap.CableTestResult{
		Interface: "eth0",
		Status:    sap.CableStatusOK,
		Length:    25.5,
		PairResults: []sap.PairResult{
			{Pair: 1, Status: sap.CableStatusOK, Length: 25.0},
			{Pair: 2, Status: sap.CableStatusOK, Length: 25.2},
			{Pair: 3, Status: sap.CableStatusOK, Length: 25.3},
			{Pair: 4, Status: sap.CableStatusOK, Length: 25.8},
		},
		TestedAt: now,
	}

	if result.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", result.Interface)
	}
	if result.Status != sap.CableStatusOK {
		t.Errorf("expected Status 'ok', got %q", result.Status)
	}
	if result.Length != 25.5 {
		t.Errorf("expected Length 25.5, got %v", result.Length)
	}
	if len(result.PairResults) != 4 {
		t.Errorf("expected 4 PairResults, got %d", len(result.PairResults))
	}
}

func TestCableTestResultMakeHelper(t *testing.T) {
	result := sap.MakeCableTestResult("eth0", sap.CableStatusOK, 25.5)

	if result.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", result.Interface)
	}
	if result.Status != sap.CableStatusOK {
		t.Errorf("expected Status 'ok', got %q", result.Status)
	}
	if result.Length != 25.5 {
		t.Errorf("expected Length 25.5, got %v", result.Length)
	}
	if result.TestedAt.IsZero() {
		t.Error("expected TestedAt to be set")
	}
}

func TestPairResultFields(t *testing.T) {
	pair := sap.PairResult{
		Pair:      1,
		Status:    sap.CableStatusOK,
		Length:    25.0,
		Impedance: 100.0,
	}

	if pair.Pair != 1 {
		t.Errorf("expected Pair 1, got %d", pair.Pair)
	}
	if pair.Status != sap.CableStatusOK {
		t.Errorf("expected Status 'ok', got %q", pair.Status)
	}
	if pair.Length != 25.0 {
		t.Errorf("expected Length 25.0, got %v", pair.Length)
	}
	if pair.Impedance != 100.0 {
		t.Errorf("expected Impedance 100.0, got %v", pair.Impedance)
	}
}

// =============================================================================
// DHCPTestResult Tests
// =============================================================================

func TestDHCPTestResultAllFields(t *testing.T) {
	now := time.Now()
	result := sap.DHCPTestResult{
		Success:      true,
		ServerIP:     "192.168.1.1",
		OfferedIP:    "192.168.1.100",
		SubnetMask:   "255.255.255.0",
		Gateway:      "192.168.1.1",
		DNSServers:   []string{"8.8.8.8", "8.8.4.4"},
		LeaseTime:    86400 * time.Second,
		LeaseTimeSec: 86400,
		ResponseTime: 50 * time.Millisecond,
		ResponseMs:   50.0,
		Error:        "",
		TestedAt:     now,
	}

	if !result.Success {
		t.Error("expected Success true")
	}
	if result.ServerIP != "192.168.1.1" {
		t.Errorf("expected ServerIP '192.168.1.1', got %q", result.ServerIP)
	}
	if result.OfferedIP != "192.168.1.100" {
		t.Errorf("expected OfferedIP '192.168.1.100', got %q", result.OfferedIP)
	}
	if result.Gateway != "192.168.1.1" {
		t.Errorf("expected Gateway '192.168.1.1', got %q", result.Gateway)
	}
	if len(result.DNSServers) != 2 {
		t.Errorf("expected 2 DNSServers, got %d", len(result.DNSServers))
	}
	if result.LeaseTimeSec != 86400 {
		t.Errorf("expected LeaseTimeSec 86400, got %d", result.LeaseTimeSec)
	}
}

func TestDHCPTestResultMakeHelper(t *testing.T) {
	result := sap.MakeDHCPTestResult(true, "192.168.1.1", "192.168.1.100", "192.168.1.1")

	if !result.Success {
		t.Error("expected Success true")
	}
	if result.ServerIP != "192.168.1.1" {
		t.Errorf("expected ServerIP '192.168.1.1', got %q", result.ServerIP)
	}
	if result.TestedAt.IsZero() {
		t.Error("expected TestedAt to be set")
	}
}

// =============================================================================
// DNSTestResult Tests
// =============================================================================

func TestDNSTestResultAllFields(t *testing.T) {
	now := time.Now()
	result := sap.DNSTestResult{
		Query:   "google.com",
		Server:  "8.8.8.8",
		Success: true,
		Answers: []sap.DNSAnswer{
			{Name: "google.com", Type: "A", Value: "142.250.80.14", TTL: 300},
		},
		ResponseTime:  10 * time.Millisecond,
		ResponseMs:    10.0,
		DNSSEC:        true,
		Authoritative: false,
		Error:         "",
		TestedAt:      now,
	}

	if result.Query != "google.com" {
		t.Errorf("expected Query 'google.com', got %q", result.Query)
	}
	if result.Server != "8.8.8.8" {
		t.Errorf("expected Server '8.8.8.8', got %q", result.Server)
	}
	if !result.Success {
		t.Error("expected Success true")
	}
	if len(result.Answers) != 1 {
		t.Errorf("expected 1 Answer, got %d", len(result.Answers))
	}
	if result.ResponseMs != 10.0 {
		t.Errorf("expected ResponseMs 10.0, got %v", result.ResponseMs)
	}
	if !result.DNSSEC {
		t.Error("expected DNSSEC true")
	}
}

func TestDNSTestResultMakeHelper(t *testing.T) {
	result := sap.MakeDNSTestResult("google.com", "8.8.8.8", true, 10.0)

	if result.Query != "google.com" {
		t.Errorf("expected Query 'google.com', got %q", result.Query)
	}
	if result.Server != "8.8.8.8" {
		t.Errorf("expected Server '8.8.8.8', got %q", result.Server)
	}
	if !result.Success {
		t.Error("expected Success true")
	}
	if result.ResponseMs != 10.0 {
		t.Errorf("expected ResponseMs 10.0, got %v", result.ResponseMs)
	}
}

func TestDNSAnswerAllFields(t *testing.T) {
	answer := sap.DNSAnswer{
		Name:  "google.com",
		Type:  "A",
		Value: "142.250.80.14",
		TTL:   300,
	}

	if answer.Name != "google.com" {
		t.Errorf("expected Name 'google.com', got %q", answer.Name)
	}
	if answer.Type != "A" {
		t.Errorf("expected Type 'A', got %q", answer.Type)
	}
	if answer.Value != "142.250.80.14" {
		t.Errorf("expected Value '142.250.80.14', got %q", answer.Value)
	}
	if answer.TTL != 300 {
		t.Errorf("expected TTL 300, got %d", answer.TTL)
	}
}

// =============================================================================
// GatewayHealth Tests
// =============================================================================

func TestGatewayHealthAllFields(t *testing.T) {
	now := time.Now()
	health := sap.GatewayHealth{
		IP:         "192.168.1.1",
		Reachable:  true,
		RTT:        5 * time.Millisecond,
		RTTMs:      5.0,
		PacketLoss: 0.0,
		Jitter:     0.5,
		Status:     sap.HealthStatusHealthy,
		Uptime:     86400 * time.Second,
		LastCheck:  now,
	}

	if health.IP != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %q", health.IP)
	}
	if !health.Reachable {
		t.Error("expected Reachable true")
	}
	if health.RTTMs != 5.0 {
		t.Errorf("expected RTTMs 5.0, got %v", health.RTTMs)
	}
	if health.PacketLoss != 0.0 {
		t.Errorf("expected PacketLoss 0.0, got %v", health.PacketLoss)
	}
	if health.Status != sap.HealthStatusHealthy {
		t.Errorf("expected Status 'healthy', got %q", health.Status)
	}
}

func TestGatewayHealthMakeHelper(t *testing.T) {
	health := sap.MakeGatewayHealth("192.168.1.1", true, 5.0, 0.0, sap.HealthStatusHealthy)

	if health.IP != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %q", health.IP)
	}
	if !health.Reachable {
		t.Error("expected Reachable true")
	}
	if health.RTTMs != 5.0 {
		t.Errorf("expected RTTMs 5.0, got %v", health.RTTMs)
	}
}

// =============================================================================
// SpeedtestResult Tests
// =============================================================================

func TestSpeedtestResultAllFields(t *testing.T) {
	now := time.Now()
	result := sap.SpeedtestResult{
		DownloadMbps: 100.0,
		UploadMbps:   50.0,
		PingMs:       10.0,
		JitterMs:     2.0,
		ServerName:   "Comcast Speed Test",
		ServerID:     "12345",
		ISP:          "Comcast",
		TestDuration: 30 * time.Second,
		TestedAt:     now,
	}

	if result.DownloadMbps != 100.0 {
		t.Errorf("expected DownloadMbps 100.0, got %v", result.DownloadMbps)
	}
	if result.UploadMbps != 50.0 {
		t.Errorf("expected UploadMbps 50.0, got %v", result.UploadMbps)
	}
	if result.PingMs != 10.0 {
		t.Errorf("expected PingMs 10.0, got %v", result.PingMs)
	}
	if result.ServerName != "Comcast Speed Test" {
		t.Errorf("expected ServerName 'Comcast Speed Test', got %q", result.ServerName)
	}
}

func TestSpeedtestResultMakeHelper(t *testing.T) {
	result := sap.MakeSpeedtestResult(100.0, 50.0, 10.0, "Test Server")

	if result.DownloadMbps != 100.0 {
		t.Errorf("expected DownloadMbps 100.0, got %v", result.DownloadMbps)
	}
	if result.UploadMbps != 50.0 {
		t.Errorf("expected UploadMbps 50.0, got %v", result.UploadMbps)
	}
	if result.ServerName != "Test Server" {
		t.Errorf("expected ServerName 'Test Server', got %q", result.ServerName)
	}
}

// =============================================================================
// IPerfResult Tests
// =============================================================================

func TestIPerfResultAllFields(t *testing.T) {
	now := time.Now()
	result := sap.IPerfResult{
		Protocol:      "tcp",
		Direction:     "upload",
		BandwidthMbps: 500.0,
		TransferMB:    625.0,
		Duration:      10 * time.Second,
		DurationSec:   10.0,
		Jitter:        1.0,
		PacketLoss:    0.1,
		Retransmits:   5,
		ServerAddr:    "192.168.1.100",
		TestedAt:      now,
	}

	if result.Protocol != "tcp" {
		t.Errorf("expected Protocol 'tcp', got %q", result.Protocol)
	}
	if result.Direction != "upload" {
		t.Errorf("expected Direction 'upload', got %q", result.Direction)
	}
	if result.BandwidthMbps != 500.0 {
		t.Errorf("expected BandwidthMbps 500.0, got %v", result.BandwidthMbps)
	}
	if result.TransferMB != 625.0 {
		t.Errorf("expected TransferMB 625.0, got %v", result.TransferMB)
	}
	if result.DurationSec != 10.0 {
		t.Errorf("expected DurationSec 10.0, got %v", result.DurationSec)
	}
	if result.Retransmits != 5 {
		t.Errorf("expected Retransmits 5, got %d", result.Retransmits)
	}
	if result.ServerAddr != "192.168.1.100" {
		t.Errorf("expected ServerAddr '192.168.1.100', got %q", result.ServerAddr)
	}
}

func TestIPerfResultMakeHelper(t *testing.T) {
	result := sap.MakeIPerfResult("tcp", "download", 500.0, 625.0, 10.0, "192.168.1.100")

	if result.Protocol != "tcp" {
		t.Errorf("expected Protocol 'tcp', got %q", result.Protocol)
	}
	if result.Direction != "download" {
		t.Errorf("expected Direction 'download', got %q", result.Direction)
	}
	if result.BandwidthMbps != 500.0 {
		t.Errorf("expected BandwidthMbps 500.0, got %v", result.BandwidthMbps)
	}
}

// =============================================================================
// VLANConfig Tests
// =============================================================================

func TestVLANConfigFields(t *testing.T) {
	config := sap.VLANConfig{
		ID:          100,
		Name:        "Management",
		Interface:   "eth0",
		IPAddress:   "192.168.100.1",
		SubnetMask:  "255.255.255.0",
		Gateway:     "192.168.100.254",
		Tagged:      true,
		MemberPorts: []string{"eth1", "eth2", "eth3"},
	}

	if config.ID != 100 {
		t.Errorf("expected ID 100, got %d", config.ID)
	}
	if config.Name != "Management" {
		t.Errorf("expected Name 'Management', got %q", config.Name)
	}
	if config.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", config.Interface)
	}
	if !config.Tagged {
		t.Error("expected Tagged true")
	}
	if len(config.MemberPorts) != 3 {
		t.Errorf("expected 3 MemberPorts, got %d", len(config.MemberPorts))
	}
}

func TestVLANConfigMakeHelper(t *testing.T) {
	config := sap.MakeVLANConfig(100, "Management", "eth0", true)

	if config.ID != 100 {
		t.Errorf("expected ID 100, got %d", config.ID)
	}
	if config.Name != "Management" {
		t.Errorf("expected Name 'Management', got %q", config.Name)
	}
	if !config.Tagged {
		t.Error("expected Tagged true")
	}
}

// =============================================================================
// SNMPDevice Tests
// =============================================================================

func TestSNMPDeviceFields(t *testing.T) {
	now := time.Now()
	device := sap.SNMPDevice{
		IP:          "192.168.1.1",
		SysName:     "switch-core-01",
		SysDescr:    "Cisco IOS Software",
		SysLocation: "Data Center Rack 1",
		SysContact:  "admin@example.com",
		SysUpTime:   86400 * time.Second,
		Interfaces: []sap.SNMPInterface{
			{Index: 1, Name: "GigabitEthernet0/1", Speed: 1000000000},
		},
		VLANs: []sap.SNMPVLAN{
			{ID: 100, Name: "Management", Status: "active"},
		},
		MACTable: []sap.MACTableEntry{
			{MACAddress: "00:11:22:33:44:55", Port: 1, VLANID: 100, Type: "dynamic"},
		},
		Custom:      map[string]any{"vendor": "Cisco"},
		CollectedAt: now,
	}

	if device.IP != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %q", device.IP)
	}
	if device.SysName != "switch-core-01" {
		t.Errorf("expected SysName 'switch-core-01', got %q", device.SysName)
	}
	if len(device.Interfaces) != 1 {
		t.Errorf("expected 1 Interface, got %d", len(device.Interfaces))
	}
	if len(device.VLANs) != 1 {
		t.Errorf("expected 1 VLAN, got %d", len(device.VLANs))
	}
	if len(device.MACTable) != 1 {
		t.Errorf("expected 1 MACTableEntry, got %d", len(device.MACTable))
	}
}

func TestSNMPDeviceMakeHelper(t *testing.T) {
	device := sap.MakeSNMPDevice("192.168.1.1", "switch-01", "Cisco IOS")

	if device.IP != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %q", device.IP)
	}
	if device.SysName != "switch-01" {
		t.Errorf("expected SysName 'switch-01', got %q", device.SysName)
	}
	if device.CollectedAt.IsZero() {
		t.Error("expected CollectedAt to be set")
	}
}

func TestSNMPInterfaceFields(t *testing.T) {
	iface := sap.SNMPInterface{
		Index:       1,
		Name:        "GigabitEthernet0/1",
		Description: "Uplink to Core",
		Type:        "ethernetCsmacd",
		Speed:       1000000000,
		AdminStatus: "up",
		OperStatus:  "up",
		InOctets:    1000000000,
		OutOctets:   500000000,
		InErrors:    0,
		OutErrors:   0,
	}

	if iface.Index != 1 {
		t.Errorf("expected Index 1, got %d", iface.Index)
	}
	if iface.Name != "GigabitEthernet0/1" {
		t.Errorf("expected Name 'GigabitEthernet0/1', got %q", iface.Name)
	}
	if iface.Speed != 1000000000 {
		t.Errorf("expected Speed 1000000000, got %d", iface.Speed)
	}
	if iface.AdminStatus != "up" {
		t.Errorf("expected AdminStatus 'up', got %q", iface.AdminStatus)
	}
}

func TestSNMPVLANFields(t *testing.T) {
	vlan := sap.SNMPVLAN{
		ID:     100,
		Name:   "Management",
		Status: "active",
		Ports:  []int{1, 2, 3, 4},
	}

	if vlan.ID != 100 {
		t.Errorf("expected ID 100, got %d", vlan.ID)
	}
	if vlan.Name != "Management" {
		t.Errorf("expected Name 'Management', got %q", vlan.Name)
	}
	if vlan.Status != "active" {
		t.Errorf("expected Status 'active', got %q", vlan.Status)
	}
	if len(vlan.Ports) != 4 {
		t.Errorf("expected 4 Ports, got %d", len(vlan.Ports))
	}
}

func TestMACTableEntryFields(t *testing.T) {
	entry := sap.MACTableEntry{
		MACAddress: "00:11:22:33:44:55",
		Port:       1,
		VLANID:     100,
		Type:       "dynamic",
	}

	if entry.MACAddress != "00:11:22:33:44:55" {
		t.Errorf("expected MACAddress '00:11:22:33:44:55', got %q", entry.MACAddress)
	}
	if entry.Port != 1 {
		t.Errorf("expected Port 1, got %d", entry.Port)
	}
	if entry.VLANID != 100 {
		t.Errorf("expected VLANID 100, got %d", entry.VLANID)
	}
	if entry.Type != "dynamic" {
		t.Errorf("expected Type 'dynamic', got %q", entry.Type)
	}
}

// =============================================================================
// BandwidthSample Tests
// =============================================================================

func TestBandwidthSampleAllFields(t *testing.T) {
	now := time.Now()
	sample := sap.BandwidthSample{
		Interface:     "eth0",
		TxBytesPerSec: 125000000, // 1 Gbps
		RxBytesPerSec: 62500000,  // 500 Mbps
		TxMbps:        1000.0,
		RxMbps:        500.0,
		Utilization:   75.0,
		SampledAt:     now,
	}

	if sample.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", sample.Interface)
	}
	if sample.TxMbps != 1000.0 {
		t.Errorf("expected TxMbps 1000.0, got %v", sample.TxMbps)
	}
	if sample.RxMbps != 500.0 {
		t.Errorf("expected RxMbps 500.0, got %v", sample.RxMbps)
	}
	if sample.Utilization != 75.0 {
		t.Errorf("expected Utilization 75.0, got %v", sample.Utilization)
	}
}

func TestBandwidthSampleMakeHelper(t *testing.T) {
	sample := sap.MakeBandwidthSample("eth0", 1000.0, 500.0, 75.0)

	if sample.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", sample.Interface)
	}
	if sample.TxMbps != 1000.0 {
		t.Errorf("expected TxMbps 1000.0, got %v", sample.TxMbps)
	}
	if sample.RxMbps != 500.0 {
		t.Errorf("expected RxMbps 500.0, got %v", sample.RxMbps)
	}
	// Verify bytes/sec conversion
	if sample.TxBytesPerSec != 125000000 {
		t.Errorf("expected TxBytesPerSec 125000000, got %v", sample.TxBytesPerSec)
	}
}

// =============================================================================
// SystemHealth Tests
// =============================================================================

func TestSystemHealthAllFields(t *testing.T) {
	now := time.Now()
	health := sap.SystemHealth{
		CPUPercent:    25.0,
		MemoryPercent: 50.0,
		DiskPercent:   75.0,
		Temperature:   45.0,
		Uptime:        86400 * time.Second,
		LoadAverage:   []float64{0.5, 0.7, 0.9},
		SampledAt:     now,
	}

	if health.CPUPercent != 25.0 {
		t.Errorf("expected CPUPercent 25.0, got %v", health.CPUPercent)
	}
	if health.MemoryPercent != 50.0 {
		t.Errorf("expected MemoryPercent 50.0, got %v", health.MemoryPercent)
	}
	if health.DiskPercent != 75.0 {
		t.Errorf("expected DiskPercent 75.0, got %v", health.DiskPercent)
	}
	if health.Temperature != 45.0 {
		t.Errorf("expected Temperature 45.0, got %v", health.Temperature)
	}
	if len(health.LoadAverage) != 3 {
		t.Errorf("expected 3 LoadAverage values, got %d", len(health.LoadAverage))
	}
}

func TestSystemHealthMakeHelper(t *testing.T) {
	health := sap.MakeSystemHealth(25.0, 50.0, 75.0)

	if health.CPUPercent != 25.0 {
		t.Errorf("expected CPUPercent 25.0, got %v", health.CPUPercent)
	}
	if health.MemoryPercent != 50.0 {
		t.Errorf("expected MemoryPercent 50.0, got %v", health.MemoryPercent)
	}
	if health.DiskPercent != 75.0 {
		t.Errorf("expected DiskPercent 75.0, got %v", health.DiskPercent)
	}
	if health.SampledAt.IsZero() {
		t.Error("expected SampledAt to be set")
	}
}

// =============================================================================
// TelemetrySnapshot Tests
// =============================================================================

func TestTelemetrySnapshotAllFields(t *testing.T) {
	now := time.Now()
	gwHealth := &sap.GatewayHealth{IP: "192.168.1.1", Reachable: true}
	dnsResult := &sap.DNSTestResult{Success: true}
	dhcpResult := &sap.DHCPTestResult{Success: true}
	bandwidth := &sap.BandwidthSample{Interface: "eth0", TxMbps: 100.0}
	sysHealth := &sap.SystemHealth{CPUPercent: 25.0}

	snapshot := sap.TelemetrySnapshot{
		Timestamp: now,
		Links: []sap.LinkStatus{
			{Interface: "eth0", State: sap.LinkStateUp},
		},
		Gateway:      gwHealth,
		DNS:          dnsResult,
		DHCP:         dhcpResult,
		Bandwidth:    bandwidth,
		SystemHealth: sysHealth,
	}

	if snapshot.Timestamp != now {
		t.Errorf("expected Timestamp %v, got %v", now, snapshot.Timestamp)
	}
	if len(snapshot.Links) != 1 {
		t.Errorf("expected 1 Link, got %d", len(snapshot.Links))
	}
	if snapshot.Gateway == nil {
		t.Error("expected Gateway to be set")
	}
	if snapshot.DNS == nil {
		t.Error("expected DNS to be set")
	}
	if snapshot.DHCP == nil {
		t.Error("expected DHCP to be set")
	}
	if snapshot.Bandwidth == nil {
		t.Error("expected Bandwidth to be set")
	}
	if snapshot.SystemHealth == nil {
		t.Error("expected SystemHealth to be set")
	}
}

func TestTelemetrySnapshotMakeHelper(t *testing.T) {
	snapshot := sap.MakeTelemetrySnapshot()

	if snapshot.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set")
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestJoinAddresses(t *testing.T) {
	tests := []struct {
		name     string
		addrs    []string
		expected string
	}{
		{"empty slice", []string{}, ""},
		{"single address", []string{"192.168.1.1"}, "192.168.1.1"},
		{"multiple addresses", []string{"192.168.1.1", "192.168.1.2"}, "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sap.JoinAddresses(tt.addrs)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConvertGatewayStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected sap.HealthStatus
	}{
		{"success", "success", sap.HealthStatusHealthy},
		{"warning", "warning", sap.HealthStatusDegraded},
		{"error", "error", sap.HealthStatusUnhealthy},
		{"unknown", "unknown", sap.HealthStatusUnknown},
		{"invalid", "invalid", sap.HealthStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sap.ConvertGatewayStatus(tt.status)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// Error Tests
// =============================================================================

func TestSapErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrNotImplemented", sap.ErrNotImplemented, "not implemented: pending migration"},
		{"ErrNotInitialized", sap.ErrNotInitialized, "service not initialized"},
		{"ErrNotSupported", sap.ErrNotSupported, "feature not supported on this platform"},
		{"ErrTestFailed", sap.ErrTestFailed, "test failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
			}
		})
	}
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestConcurrentLinkStatusAccess(_ *testing.T) {
	done := make(chan bool)

	for range 10 {
		go func() {
			for range 50 {
				_ = sap.MakeLinkStatus("eth0", sap.LinkStateUp, "1000", "full", 1500, "00:11:22:33:44:55")
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}

func TestConcurrentSpeedtestResultAccess(_ *testing.T) {
	done := make(chan bool)

	for range 10 {
		go func() {
			for range 50 {
				_ = sap.MakeSpeedtestResult(100.0, 50.0, 10.0, "Test Server")
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}

// =============================================================================
// Table-Driven Type Tests
// =============================================================================

func TestLinkStatusTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		iface    string
		state    sap.LinkState
		speed    string
		mtu      int
		expectUp bool
	}{
		{"up interface", "eth0", sap.LinkStateUp, "1000", 1500, true},
		{"down interface", "eth1", sap.LinkStateDown, "0", 1500, false},
		{"dormant interface", "eth2", sap.LinkStateDormant, "1000", 1500, false},
		{"unknown interface", "eth3", sap.LinkStateUnknown, "", 1500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := sap.MakeLinkStatus(tt.iface, tt.state, tt.speed, "full", tt.mtu, "00:00:00:00:00:00")

			if status.Interface != tt.iface {
				t.Errorf("expected Interface %q, got %q", tt.iface, status.Interface)
			}
			if status.State != tt.state {
				t.Errorf("expected State %q, got %q", tt.state, status.State)
			}
			if status.Carrier != tt.expectUp {
				t.Errorf("expected Carrier %v, got %v", tt.expectUp, status.Carrier)
			}
		})
	}
}

func TestSpeedtestResultTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		download float64
		upload   float64
		ping     float64
	}{
		{"fast fiber", 950.0, 950.0, 3.0},
		{"cable", 200.0, 20.0, 15.0},
		{"dsl", 25.0, 5.0, 30.0},
		{"mobile 4g", 50.0, 25.0, 40.0},
		{"mobile 5g", 500.0, 100.0, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sap.MakeSpeedtestResult(tt.download, tt.upload, tt.ping, "Test Server")

			if result.DownloadMbps != tt.download {
				t.Errorf("expected DownloadMbps %v, got %v", tt.download, result.DownloadMbps)
			}
			if result.UploadMbps != tt.upload {
				t.Errorf("expected UploadMbps %v, got %v", tt.upload, result.UploadMbps)
			}
			if result.PingMs != tt.ping {
				t.Errorf("expected PingMs %v, got %v", tt.ping, result.PingMs)
			}
		})
	}
}

func TestIPerfResultTableDriven(t *testing.T) {
	tests := []struct {
		name      string
		protocol  string
		direction string
		bandwidth float64
	}{
		{"tcp upload", "tcp", "upload", 500.0},
		{"tcp download", "tcp", "download", 800.0},
		{"udp upload", "udp", "upload", 100.0},
		{"udp download", "udp", "download", 150.0},
		{"tcp bidirectional", "tcp", "bidirectional", 450.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sap.MakeIPerfResult(tt.protocol, tt.direction, tt.bandwidth, 125.0, 10.0, "localhost")

			if result.Protocol != tt.protocol {
				t.Errorf("expected Protocol %q, got %q", tt.protocol, result.Protocol)
			}
			if result.Direction != tt.direction {
				t.Errorf("expected Direction %q, got %q", tt.direction, result.Direction)
			}
			if result.BandwidthMbps != tt.bandwidth {
				t.Errorf("expected BandwidthMbps %v, got %v", tt.bandwidth, result.BandwidthMbps)
			}
		})
	}
}

func TestGatewayHealthTableDriven(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		reachable  bool
		rtt        float64
		packetLoss float64
		status     sap.HealthStatus
	}{
		{"healthy gateway", "192.168.1.1", true, 2.0, 0.0, sap.HealthStatusHealthy},
		{"degraded gateway", "192.168.1.1", true, 50.0, 5.0, sap.HealthStatusDegraded},
		{"unhealthy gateway", "192.168.1.1", false, 0.0, 100.0, sap.HealthStatusUnhealthy},
		{"unknown gateway", "10.0.0.1", false, 0.0, 0.0, sap.HealthStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := sap.MakeGatewayHealth(tt.ip, tt.reachable, tt.rtt, tt.packetLoss, tt.status)

			if health.IP != tt.ip {
				t.Errorf("expected IP %q, got %q", tt.ip, health.IP)
			}
			if health.Reachable != tt.reachable {
				t.Errorf("expected Reachable %v, got %v", tt.reachable, health.Reachable)
			}
			if health.RTTMs != tt.rtt {
				t.Errorf("expected RTTMs %v, got %v", tt.rtt, health.RTTMs)
			}
			if health.Status != tt.status {
				t.Errorf("expected Status %q, got %q", tt.status, health.Status)
			}
		})
	}
}
