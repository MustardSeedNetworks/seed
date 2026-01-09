package sap_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/sap"
)

// TestNewTelemetryService verifies that NewTelemetryService creates a valid service.
func TestNewTelemetryService(t *testing.T) {
	cfg := config.DefaultConfig()
	service := sap.NewTelemetryService(cfg, nil)

	if service == nil {
		t.Fatal("expected non-nil TelemetryService")
	}
}

// TestTelemetryServiceStartStop verifies Start and Stop don't panic.
func TestTelemetryServiceStartStop(t *testing.T) {
	cfg := config.DefaultConfig()
	service := sap.NewTelemetryService(cfg, nil)

	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		t.Errorf("Start() returned error: %v", err)
	}

	// Stop should not panic.
	service.Stop()

	// Multiple stops should be safe.
	service.Stop()
}

// TestTelemetryServiceGetSnapshotNotImplemented verifies GetSnapshot returns ErrNotImplemented.
func TestTelemetryServiceGetSnapshotNotImplemented(t *testing.T) {
	cfg := config.DefaultConfig()
	service := sap.NewTelemetryService(cfg, nil)

	ctx := context.Background()
	snapshot, err := service.GetSnapshot(ctx)

	if !errors.Is(err, sap.ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
	if snapshot != nil {
		t.Errorf("expected nil snapshot, got %+v", snapshot)
	}
}

// TestTelemetryServiceGetHistoryNotImplemented verifies GetHistory returns ErrNotImplemented.
func TestTelemetryServiceGetHistoryNotImplemented(t *testing.T) {
	cfg := config.DefaultConfig()
	service := sap.NewTelemetryService(cfg, nil)

	ctx := context.Background()
	history, err := service.GetHistory(ctx, "", "")

	if !errors.Is(err, sap.ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
	if history != nil {
		t.Errorf("expected nil history, got %+v", history)
	}
}

// TestTelemetryServiceWithCanceledContext verifies behavior with canceled context.
func TestTelemetryServiceWithCanceledContext(t *testing.T) {
	cfg := config.DefaultConfig()
	service := sap.NewTelemetryService(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	// Start with canceled context should not panic.
	if err := service.Start(ctx); err != nil {
		t.Errorf("Start() with canceled context returned error: %v", err)
	}

	service.Stop()
}

// TestTelemetrySnapshotConstruction verifies TelemetrySnapshot can be constructed.
func TestTelemetrySnapshotConstruction(t *testing.T) {
	now := time.Now()

	snapshot := sap.TelemetrySnapshot{
		Timestamp: now,
		Links:     []sap.LinkStatus{},
		Gateway:   nil,
		DNS:       nil,
		DHCP:      nil,
		Bandwidth: nil,
	}

	if snapshot.Timestamp != now {
		t.Errorf("expected Timestamp %v, got %v", now, snapshot.Timestamp)
	}
	if snapshot.Links == nil {
		t.Error("expected non-nil Links slice")
	}
	if len(snapshot.Links) != 0 {
		t.Errorf("expected empty Links slice, got %d elements", len(snapshot.Links))
	}
	if snapshot.Gateway != nil {
		t.Error("expected nil Gateway")
	}
	if snapshot.DNS != nil {
		t.Error("expected nil DNS")
	}
	if snapshot.DHCP != nil {
		t.Error("expected nil DHCP")
	}
	if snapshot.Bandwidth != nil {
		t.Error("expected nil Bandwidth")
	}
}

// TestTelemetrySnapshotWithData verifies TelemetrySnapshot with populated fields.
func TestTelemetrySnapshotWithData(t *testing.T) {
	now := time.Now()

	linkStatus := sap.LinkStatus{
		Interface:  "eth0",
		State:      sap.LinkStateUp,
		Speed:      "1000Mbps",
		Duplex:     "full",
		MTU:        1500,
		MACAddress: "00:11:22:33:44:55",
		IPAddress:  "192.168.1.100",
		Carrier:    true,
		UpdatedAt:  now,
	}

	gatewayHealth := &sap.GatewayHealth{
		IP:         "192.168.1.1",
		Reachable:  true,
		RTT:        time.Millisecond * 5,
		RTTMs:      5.0,
		PacketLoss: 0.0,
		Status:     sap.HealthStatusHealthy,
		LastCheck:  now,
	}

	bandwidth := &sap.BandwidthSample{
		Interface:     "eth0",
		TxBytesPerSec: 1000000,
		RxBytesPerSec: 5000000,
		TxMbps:        8.0,
		RxMbps:        40.0,
		Utilization:   5.0,
		SampledAt:     now,
	}

	snapshot := sap.TelemetrySnapshot{
		Timestamp: now,
		Links:     []sap.LinkStatus{linkStatus},
		Gateway:   gatewayHealth,
		Bandwidth: bandwidth,
	}

	if len(snapshot.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(snapshot.Links))
	}
	if snapshot.Links[0].Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", snapshot.Links[0].Interface)
	}
	if snapshot.Links[0].State != sap.LinkStateUp {
		t.Errorf("expected State LinkStateUp, got %v", snapshot.Links[0].State)
	}
	if snapshot.Gateway == nil {
		t.Fatal("expected non-nil Gateway")
	}
	if snapshot.Gateway.IP != "192.168.1.1" {
		t.Errorf("expected Gateway IP '192.168.1.1', got %q", snapshot.Gateway.IP)
	}
	if snapshot.Bandwidth == nil {
		t.Fatal("expected non-nil Bandwidth")
	}
	if snapshot.Bandwidth.RxMbps != 40.0 {
		t.Errorf("expected RxMbps 40.0, got %v", snapshot.Bandwidth.RxMbps)
	}
}

// TestTelemetryBandwidthSampleTableDriven tests BandwidthSample with table-driven tests.
func TestTelemetryBandwidthSampleTableDriven(t *testing.T) {
	tests := []struct {
		name          string
		sample        sap.BandwidthSample
		wantInterface string
		wantTxBytes   float64
		wantRxBytes   float64
		wantTxMbps    float64
		wantRxMbps    float64
		wantUtil      float64
	}{
		{
			name: "zero values",
			sample: sap.BandwidthSample{
				Interface: "",
			},
			wantInterface: "",
			wantTxBytes:   0,
			wantRxBytes:   0,
			wantTxMbps:    0,
			wantRxMbps:    0,
			wantUtil:      0,
		},
		{
			name: "typical values",
			sample: sap.BandwidthSample{
				Interface:     "eth0",
				TxBytesPerSec: 125000000, // 1 Gbps
				RxBytesPerSec: 62500000,  // 500 Mbps
				TxMbps:        1000.0,
				RxMbps:        500.0,
				Utilization:   50.0,
				SampledAt:     time.Now(),
			},
			wantInterface: "eth0",
			wantTxBytes:   125000000,
			wantRxBytes:   62500000,
			wantTxMbps:    1000.0,
			wantRxMbps:    500.0,
			wantUtil:      50.0,
		},
		{
			name: "high utilization",
			sample: sap.BandwidthSample{
				Interface:     "enp0s31f6",
				TxBytesPerSec: 118750000, // 950 Mbps
				RxBytesPerSec: 118750000, // 950 Mbps
				TxMbps:        950.0,
				RxMbps:        950.0,
				Utilization:   95.0,
				SampledAt:     time.Now(),
			},
			wantInterface: "enp0s31f6",
			wantTxBytes:   118750000,
			wantRxBytes:   118750000,
			wantTxMbps:    950.0,
			wantRxMbps:    950.0,
			wantUtil:      95.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sample.Interface != tt.wantInterface {
				t.Errorf("Interface = %q, want %q", tt.sample.Interface, tt.wantInterface)
			}
			if tt.sample.TxBytesPerSec != tt.wantTxBytes {
				t.Errorf("TxBytesPerSec = %v, want %v", tt.sample.TxBytesPerSec, tt.wantTxBytes)
			}
			if tt.sample.RxBytesPerSec != tt.wantRxBytes {
				t.Errorf("RxBytesPerSec = %v, want %v", tt.sample.RxBytesPerSec, tt.wantRxBytes)
			}
			if tt.sample.TxMbps != tt.wantTxMbps {
				t.Errorf("TxMbps = %v, want %v", tt.sample.TxMbps, tt.wantTxMbps)
			}
			if tt.sample.RxMbps != tt.wantRxMbps {
				t.Errorf("RxMbps = %v, want %v", tt.sample.RxMbps, tt.wantRxMbps)
			}
			if tt.sample.Utilization != tt.wantUtil {
				t.Errorf("Utilization = %v, want %v", tt.sample.Utilization, tt.wantUtil)
			}
		})
	}
}

// TestTelemetrySystemHealthTableDriven tests SystemHealth with table-driven tests.
func TestTelemetrySystemHealthTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		health      sap.SystemHealth
		wantCPU     float64
		wantMemory  float64
		wantDisk    float64
		wantTemp    float64
		wantUptime  time.Duration
		wantLoadAvg []float64
	}{
		{
			name:        "zero values",
			health:      sap.SystemHealth{},
			wantCPU:     0,
			wantMemory:  0,
			wantDisk:    0,
			wantTemp:    0,
			wantUptime:  0,
			wantLoadAvg: nil,
		},
		{
			name: "typical server values",
			health: sap.SystemHealth{
				CPUPercent:    25.5,
				MemoryPercent: 60.0,
				DiskPercent:   45.0,
				Temperature:   55.0,
				Uptime:        24 * time.Hour * 30, // 30 days
				LoadAverage:   []float64{1.5, 2.0, 1.8},
				SampledAt:     time.Now(),
			},
			wantCPU:     25.5,
			wantMemory:  60.0,
			wantDisk:    45.0,
			wantTemp:    55.0,
			wantUptime:  24 * time.Hour * 30,
			wantLoadAvg: []float64{1.5, 2.0, 1.8},
		},
		{
			name: "high load values",
			health: sap.SystemHealth{
				CPUPercent:    95.0,
				MemoryPercent: 90.0,
				DiskPercent:   85.0,
				Temperature:   80.0,
				Uptime:        time.Hour * 2,
				LoadAverage:   []float64{8.0, 7.5, 6.0},
				SampledAt:     time.Now(),
			},
			wantCPU:     95.0,
			wantMemory:  90.0,
			wantDisk:    85.0,
			wantTemp:    80.0,
			wantUptime:  time.Hour * 2,
			wantLoadAvg: []float64{8.0, 7.5, 6.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.health.CPUPercent != tt.wantCPU {
				t.Errorf("CPUPercent = %v, want %v", tt.health.CPUPercent, tt.wantCPU)
			}
			if tt.health.MemoryPercent != tt.wantMemory {
				t.Errorf("MemoryPercent = %v, want %v", tt.health.MemoryPercent, tt.wantMemory)
			}
			if tt.health.DiskPercent != tt.wantDisk {
				t.Errorf("DiskPercent = %v, want %v", tt.health.DiskPercent, tt.wantDisk)
			}
			if tt.health.Temperature != tt.wantTemp {
				t.Errorf("Temperature = %v, want %v", tt.health.Temperature, tt.wantTemp)
			}
			if tt.health.Uptime != tt.wantUptime {
				t.Errorf("Uptime = %v, want %v", tt.health.Uptime, tt.wantUptime)
			}
			if len(tt.health.LoadAverage) != len(tt.wantLoadAvg) {
				t.Errorf("LoadAverage length = %d, want %d",
					len(tt.health.LoadAverage), len(tt.wantLoadAvg))
			}
		})
	}
}

// TestTelemetrySnapshotWithSystemHealth verifies TelemetrySnapshot with SystemHealth.
func TestTelemetrySnapshotWithSystemHealth(t *testing.T) {
	now := time.Now()

	systemHealth := &sap.SystemHealth{
		CPUPercent:    35.5,
		MemoryPercent: 55.0,
		DiskPercent:   40.0,
		Temperature:   45.0,
		Uptime:        time.Hour * 48,
		LoadAverage:   []float64{1.0, 0.8, 0.9},
		SampledAt:     now,
	}

	snapshot := sap.TelemetrySnapshot{
		Timestamp:    now,
		SystemHealth: systemHealth,
	}

	if snapshot.SystemHealth == nil {
		t.Fatal("expected non-nil SystemHealth")
	}
	if snapshot.SystemHealth.CPUPercent != 35.5 {
		t.Errorf("expected CPUPercent 35.5, got %v", snapshot.SystemHealth.CPUPercent)
	}
	if snapshot.SystemHealth.Uptime != time.Hour*48 {
		t.Errorf("expected Uptime 48h, got %v", snapshot.SystemHealth.Uptime)
	}
	if len(snapshot.SystemHealth.LoadAverage) != 3 {
		t.Errorf("expected 3 LoadAverage values, got %d", len(snapshot.SystemHealth.LoadAverage))
	}
}

// TestTelemetryLinkStatusTableDriven tests LinkStatus with table-driven tests.
func TestTelemetryLinkStatusTableDriven(t *testing.T) {
	tests := []struct {
		name   string
		status sap.LinkStatus
	}{
		{
			name: "up interface with full data",
			status: sap.LinkStatus{
				Interface:  "eth0",
				State:      sap.LinkStateUp,
				Speed:      "1000Mbps",
				Duplex:     "full",
				MTU:        1500,
				MACAddress: "00:11:22:33:44:55",
				IPAddress:  "192.168.1.100",
				Gateway:    "192.168.1.1",
				Carrier:    true,
				TxBytes:    1000000,
				RxBytes:    5000000,
				TxPackets:  1000,
				RxPackets:  5000,
				TxErrors:   0,
				RxErrors:   0,
				TxDropped:  0,
				RxDropped:  0,
				UpdatedAt:  time.Now(),
			},
		},
		{
			name: "down interface",
			status: sap.LinkStatus{
				Interface:  "eth1",
				State:      sap.LinkStateDown,
				Speed:      "",
				Duplex:     "",
				MTU:        1500,
				MACAddress: "00:11:22:33:44:66",
				Carrier:    false,
				UpdatedAt:  time.Now(),
			},
		},
		{
			name: "interface with errors",
			status: sap.LinkStatus{
				Interface:  "eth2",
				State:      sap.LinkStateUp,
				Speed:      "100Mbps",
				Duplex:     "half",
				MTU:        1500,
				MACAddress: "00:11:22:33:44:77",
				Carrier:    true,
				TxBytes:    500000,
				RxBytes:    2500000,
				TxPackets:  500,
				RxPackets:  2500,
				TxErrors:   10,
				RxErrors:   5,
				TxDropped:  2,
				RxDropped:  1,
				UpdatedAt:  time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status.Interface == "" {
				t.Error("expected non-empty Interface")
			}
			if tt.status.State == "" {
				t.Error("expected non-empty State")
			}
			if tt.status.UpdatedAt.IsZero() {
				t.Error("expected non-zero UpdatedAt")
			}
		})
	}
}

// TestTelemetryLinkStateConstants verifies LinkState constant values.
func TestTelemetryLinkStateConstants(t *testing.T) {
	tests := []struct {
		state    sap.LinkState
		expected string
	}{
		{sap.LinkStateUp, "up"},
		{sap.LinkStateDown, "down"},
		{sap.LinkStateDormant, "dormant"},
		{sap.LinkStateUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.state))
			}
		})
	}
}

// TestTelemetryHealthStatusConstants verifies HealthStatus constant values.
func TestTelemetryHealthStatusConstants(t *testing.T) {
	tests := []struct {
		status   sap.HealthStatus
		expected string
	}{
		{sap.HealthStatusHealthy, "healthy"},
		{sap.HealthStatusDegraded, "degraded"},
		{sap.HealthStatusUnhealthy, "unhealthy"},
		{sap.HealthStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
			}
		})
	}
}

// TestTelemetryCableStatusConstants verifies CableStatus constant values.
func TestTelemetryCableStatusConstants(t *testing.T) {
	tests := []struct {
		status   sap.CableStatus
		expected string
	}{
		{sap.CableStatusOK, "ok"},
		{sap.CableStatusOpen, "open"},
		{sap.CableStatusShort, "short"},
		{sap.CableStatusImpedance, "impedance_mismatch"},
		{sap.CableStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
			}
		})
	}
}

// TestTelemetryGatewayHealthTableDriven tests GatewayHealth with table-driven tests.
func TestTelemetryGatewayHealthTableDriven(t *testing.T) {
	tests := []struct {
		name   string
		health sap.GatewayHealth
	}{
		{
			name: "healthy gateway",
			health: sap.GatewayHealth{
				IP:         "192.168.1.1",
				Reachable:  true,
				RTT:        time.Millisecond * 2,
				RTTMs:      2.0,
				PacketLoss: 0.0,
				Jitter:     0.5,
				Status:     sap.HealthStatusHealthy,
				Uptime:     time.Hour * 24 * 7,
				LastCheck:  time.Now(),
			},
		},
		{
			name: "degraded gateway",
			health: sap.GatewayHealth{
				IP:         "10.0.0.1",
				Reachable:  true,
				RTT:        time.Millisecond * 100,
				RTTMs:      100.0,
				PacketLoss: 5.0,
				Jitter:     10.0,
				Status:     sap.HealthStatusDegraded,
				LastCheck:  time.Now(),
			},
		},
		{
			name: "unreachable gateway",
			health: sap.GatewayHealth{
				IP:         "172.16.0.1",
				Reachable:  false,
				RTT:        0,
				RTTMs:      0,
				PacketLoss: 100.0,
				Jitter:     0,
				Status:     sap.HealthStatusUnhealthy,
				LastCheck:  time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.health.IP == "" {
				t.Error("expected non-empty IP")
			}
			if tt.health.LastCheck.IsZero() {
				t.Error("expected non-zero LastCheck")
			}
			if tt.health.Status == "" {
				t.Error("expected non-empty Status")
			}
		})
	}
}

// TestTelemetryVLANConfigTableDriven tests VLANConfig with table-driven tests.
func TestTelemetryVLANConfigTableDriven(t *testing.T) {
	tests := []struct {
		name   string
		config sap.VLANConfig
	}{
		{
			name: "tagged VLAN",
			config: sap.VLANConfig{
				ID:          100,
				Name:        "Management",
				Interface:   "eth0",
				IPAddress:   "192.168.100.1",
				SubnetMask:  "255.255.255.0",
				Gateway:     "192.168.100.254",
				Tagged:      true,
				MemberPorts: []string{"eth0", "eth1"},
			},
		},
		{
			name: "native VLAN",
			config: sap.VLANConfig{
				ID:        1,
				Name:      "Native",
				Interface: "eth0",
				Tagged:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.ID < 1 || tt.config.ID > 4094 {
				t.Errorf("expected VLAN ID between 1-4094, got %d", tt.config.ID)
			}
			if tt.config.Interface == "" {
				t.Error("expected non-empty Interface")
			}
		})
	}
}

// TestTelemetryCableTestResultTableDriven tests CableTestResult with table-driven tests.
func TestTelemetryCableTestResultTableDriven(t *testing.T) {
	tests := []struct {
		name   string
		result sap.CableTestResult
	}{
		{
			name: "cable OK",
			result: sap.CableTestResult{
				Interface: "eth0",
				Status:    sap.CableStatusOK,
				Length:    25.5,
				PairResults: []sap.PairResult{
					{Pair: 1, Status: sap.CableStatusOK, Length: 25.5, Impedance: 100.0},
					{Pair: 2, Status: sap.CableStatusOK, Length: 25.5, Impedance: 100.0},
					{Pair: 3, Status: sap.CableStatusOK, Length: 25.5, Impedance: 100.0},
					{Pair: 4, Status: sap.CableStatusOK, Length: 25.5, Impedance: 100.0},
				},
				TestedAt: time.Now(),
			},
		},
		{
			name: "cable with open pair",
			result: sap.CableTestResult{
				Interface: "eth1",
				Status:    sap.CableStatusOpen,
				Length:    10.0,
				PairResults: []sap.PairResult{
					{Pair: 1, Status: sap.CableStatusOK, Length: 10.0},
					{Pair: 2, Status: sap.CableStatusOpen, Length: 10.0},
					{Pair: 3, Status: sap.CableStatusOK, Length: 10.0},
					{Pair: 4, Status: sap.CableStatusOK, Length: 10.0},
				},
				TestedAt: time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Interface == "" {
				t.Error("expected non-empty Interface")
			}
			if tt.result.Status == "" {
				t.Error("expected non-empty Status")
			}
			if tt.result.TestedAt.IsZero() {
				t.Error("expected non-zero TestedAt")
			}
		})
	}
}

// TestTelemetryPairResultTableDriven tests PairResult with table-driven tests.
func TestTelemetryPairResultTableDriven(t *testing.T) {
	tests := []struct {
		name   string
		result sap.PairResult
	}{
		{
			name:   "OK pair with full data",
			result: sap.PairResult{Pair: 1, Status: sap.CableStatusOK, Length: 30.0, Impedance: 100.0},
		},
		{
			name:   "shorted pair",
			result: sap.PairResult{Pair: 2, Status: sap.CableStatusShort, Length: 5.0},
		},
		{
			name:   "impedance mismatch",
			result: sap.PairResult{Pair: 3, Status: sap.CableStatusImpedance, Length: 20.0, Impedance: 75.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Pair < 1 || tt.result.Pair > 4 {
				t.Errorf("expected Pair between 1-4, got %d", tt.result.Pair)
			}
			if tt.result.Status == "" {
				t.Error("expected non-empty Status")
			}
		})
	}
}

// TestTelemetrySNMPDeviceTableDriven tests SNMPDevice with table-driven tests.
func TestTelemetrySNMPDeviceTableDriven(t *testing.T) {
	tests := []struct {
		name   string
		device sap.SNMPDevice
	}{
		{
			name: "switch with full data",
			device: sap.SNMPDevice{
				IP:          "192.168.1.2",
				SysName:     "core-switch-01",
				SysDescr:    "Cisco IOS Software, Version 15.2",
				SysLocation: "Server Room A",
				SysContact:  "admin@example.com",
				SysUpTime:   time.Hour * 24 * 365,
				Interfaces: []sap.SNMPInterface{
					{Index: 1, Name: "GigabitEthernet0/1", Type: "ethernetCsmacd", Speed: 1000000000},
				},
				VLANs: []sap.SNMPVLAN{
					{ID: 1, Name: "default", Status: "active"},
					{ID: 100, Name: "Management", Status: "active"},
				},
				MACTable: []sap.MACTableEntry{
					{MACAddress: "00:11:22:33:44:55", Port: 1, VLANID: 1, Type: "dynamic"},
				},
				CollectedAt: time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.device.IP == "" {
				t.Error("expected non-empty IP")
			}
			if tt.device.CollectedAt.IsZero() {
				t.Error("expected non-zero CollectedAt")
			}
		})
	}
}

// TestTelemetrySNMPInterfaceConstruction tests SNMPInterface struct construction.
func TestTelemetrySNMPInterfaceConstruction(t *testing.T) {
	iface := sap.SNMPInterface{
		Index:       1,
		Name:        "GigabitEthernet0/1",
		Description: "Uplink to Router",
		Type:        "ethernetCsmacd",
		Speed:       1000000000,
		AdminStatus: "up",
		OperStatus:  "up",
		InOctets:    1000000000,
		OutOctets:   500000000,
		InErrors:    0,
		OutErrors:   0,
	}

	if iface.Index < 1 {
		t.Errorf("expected positive Index, got %d", iface.Index)
	}
	if iface.Name == "" {
		t.Error("expected non-empty Name")
	}
	if iface.Type == "" {
		t.Error("expected non-empty Type")
	}
}

// TestTelemetrySNMPVLANConstruction tests SNMPVLAN struct construction.
func TestTelemetrySNMPVLANConstruction(t *testing.T) {
	vlan := sap.SNMPVLAN{
		ID:     100,
		Name:   "Management",
		Status: "active",
		Ports:  []int{1, 2, 3, 4},
	}

	if vlan.ID < 1 || vlan.ID > 4094 {
		t.Errorf("expected VLAN ID between 1-4094, got %d", vlan.ID)
	}
	if vlan.Name == "" {
		t.Error("expected non-empty Name")
	}
}

// TestTelemetryMACTableEntryTableDriven tests MACTableEntry with table-driven tests.
func TestTelemetryMACTableEntryTableDriven(t *testing.T) {
	tests := []struct {
		name  string
		entry sap.MACTableEntry
	}{
		{
			name:  "dynamic entry",
			entry: sap.MACTableEntry{MACAddress: "00:11:22:33:44:55", Port: 1, VLANID: 1, Type: "dynamic"},
		},
		{
			name:  "static entry",
			entry: sap.MACTableEntry{MACAddress: "66:77:88:99:AA:BB", Port: 24, VLANID: 100, Type: "static"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.entry.MACAddress == "" {
				t.Error("expected non-empty MACAddress")
			}
			if tt.entry.Port < 1 {
				t.Errorf("expected positive Port, got %d", tt.entry.Port)
			}
			if tt.entry.Type != "dynamic" && tt.entry.Type != "static" {
				t.Errorf("expected Type 'dynamic' or 'static', got %q", tt.entry.Type)
			}
		})
	}
}

// TestTelemetryErrorConstants verifies error constant values.
func TestTelemetryErrorConstants(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotImplemented", sap.ErrNotImplemented},
		{"ErrNotInitialized", sap.ErrNotInitialized},
		{"ErrNotSupported", sap.ErrNotSupported},
		{"ErrTestFailed", sap.ErrTestFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s should have non-empty message", tt.name)
			}
		})
	}
}
