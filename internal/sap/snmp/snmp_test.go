package snmp_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/sap/snmp"
)

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   snmp.Status
		expected string
	}{
		{"StatusSuccess", snmp.StatusSuccess, "success"},
		{"StatusWarning", snmp.StatusWarning, "warning"},
		{"StatusError", snmp.StatusError, "error"},
		{"StatusUnknown", snmp.StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
			}
		})
	}
}

func TestDefaultThresholds(t *testing.T) {
	thresholds := snmp.DefaultThresholds()

	if thresholds.Warning != 500*time.Millisecond {
		t.Errorf("expected Warning threshold 500ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 2000*time.Millisecond {
		t.Errorf("expected Critical threshold 2000ms, got %v", thresholds.Critical)
	}
}

func TestNewCollector(t *testing.T) {
	cfg := snmp.DefaultConfig()
	collector := snmp.NewCollector(cfg)

	if collector == nil {
		t.Fatal("NewCollector returned nil")
	}

	if collector.CollectorConfig() != cfg {
		t.Error("expected config to be set correctly")
	}

	thresholds := collector.CollectorThresholds()
	if thresholds.Warning != snmp.DefaultThresholds().Warning {
		t.Errorf("expected default warning threshold, got %v", thresholds.Warning)
	}
	if thresholds.Critical != snmp.DefaultThresholds().Critical {
		t.Errorf("expected default critical threshold, got %v", thresholds.Critical)
	}
}

func TestNewCollectorWithNilConfig(t *testing.T) {
	collector := snmp.NewCollector(nil)

	if collector == nil {
		t.Fatal("NewCollector returned nil")
	}

	if collector.CollectorConfig() != nil {
		t.Error("expected nil config")
	}
}

func TestCollectorSetGetThresholds(t *testing.T) {
	collector := snmp.NewCollector(snmp.DefaultConfig())

	newThresholds := snmp.Thresholds{
		Warning:  100 * time.Millisecond,
		Critical: 500 * time.Millisecond,
	}

	collector.SetThresholds(newThresholds)
	got := collector.GetThresholds()

	if got.Warning != newThresholds.Warning {
		t.Errorf("expected Warning %v, got %v", newThresholds.Warning, got.Warning)
	}
	if got.Critical != newThresholds.Critical {
		t.Errorf("expected Critical %v, got %v", newThresholds.Critical, got.Critical)
	}
}

func TestDetermineStatus(t *testing.T) {
	collector := snmp.NewCollector(snmp.DefaultConfig())
	thresholds := snmp.Thresholds{
		Warning:  100 * time.Millisecond,
		Critical: 500 * time.Millisecond,
	}

	tests := []struct {
		name     string
		elapsed  time.Duration
		expected snmp.Status
	}{
		{"success - fast", 50 * time.Millisecond, snmp.StatusSuccess},
		{"success - just below warning", 99 * time.Millisecond, snmp.StatusSuccess},
		{"warning - at threshold", 100 * time.Millisecond, snmp.StatusWarning},
		{"warning - between thresholds", 300 * time.Millisecond, snmp.StatusWarning},
		{"warning - just below critical", 499 * time.Millisecond, snmp.StatusWarning},
		{"error - at critical", 500 * time.Millisecond, snmp.StatusError},
		{"error - above critical", 1000 * time.Millisecond, snmp.StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := collector.DetermineStatus(tt.elapsed, thresholds)
			if status != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, status)
			}
		})
	}
}

func TestExportIsValidIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv4 - localhost", "127.0.0.1", true},
		{"valid IPv4 - broadcast", "255.255.255.255", true},
		{"valid IPv4 - zeros", "0.0.0.0", true},
		{"valid IPv6 - localhost", "::1", true},
		{"valid IPv6 - full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"valid IPv6 - compressed", "2001:db8::8a2e:370:7334", true},
		{"valid IPv6 - link-local", "fe80::1", true},
		{"invalid - empty", "", false},
		{"invalid - hostname", "example.com", false},
		{"invalid - not an IP", "not-an-ip", false},
		{"invalid - partial IPv4", "192.168.1", false},
		{"invalid - out of range", "256.256.256.256", false},
		{"invalid - with port", "192.168.1.1:161", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := snmp.ExportIsValidIP(tt.ip)
			if result != tt.expected {
				t.Errorf("isValidIP(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.SNMPConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &config.SNMPConfig{
				Port:        161,
				Timeout:     5 * time.Second,
				Retries:     2,
				Communities: []string{"public"},
			},
			wantErr: false,
		},
		{
			name: "valid config with v3",
			cfg: &config.SNMPConfig{
				Port:    161,
				Timeout: 5 * time.Second,
				Retries: 2,
				V3Credentials: []config.SNMPv3Credential{
					{Username: "admin"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port - zero",
			cfg: &config.SNMPConfig{
				Port:        0,
				Timeout:     5 * time.Second,
				Communities: []string{"public"},
			},
			wantErr: true,
		},
		{
			name: "invalid port - negative",
			cfg: &config.SNMPConfig{
				Port:        -1,
				Timeout:     5 * time.Second,
				Communities: []string{"public"},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			cfg: &config.SNMPConfig{
				Port:        70000,
				Timeout:     5 * time.Second,
				Communities: []string{"public"},
			},
			wantErr: true,
		},
		{
			name: "invalid timeout - zero",
			cfg: &config.SNMPConfig{
				Port:        161,
				Timeout:     0,
				Communities: []string{"public"},
			},
			wantErr: true,
		},
		{
			name: "invalid timeout - negative",
			cfg: &config.SNMPConfig{
				Port:        161,
				Timeout:     -1 * time.Second,
				Communities: []string{"public"},
			},
			wantErr: true,
		},
		{
			name: "no credentials",
			cfg: &config.SNMPConfig{
				Port:    161,
				Timeout: 5 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := snmp.ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := snmp.DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.Port != 161 {
		t.Errorf("expected port 161, got %d", cfg.Port)
	}

	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", cfg.Timeout)
	}

	if cfg.Retries != 2 {
		t.Errorf("expected retries 2, got %d", cfg.Retries)
	}

	if len(cfg.Communities) != 1 || cfg.Communities[0] != "public" {
		t.Errorf("expected communities ['public'], got %v", cfg.Communities)
	}

	// Validate the default config
	if err := snmp.ValidateConfig(cfg); err != nil {
		t.Errorf("DefaultConfig() produced invalid config: %v", err)
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"seconds only", 45 * time.Second, "45s"},
		{"minutes and seconds", 5*time.Minute + 30*time.Second, "5m 30s"},
		{"hours minutes seconds", 2*time.Hour + 15*time.Minute + 30*time.Second, "2h 15m 30s"},
		{"days hours minutes seconds", 3*24*time.Hour + 5*time.Hour + 30*time.Minute + 15*time.Second, "3d 5h 30m 15s"},
		{"just minutes", 10 * time.Minute, "10m 0s"},
		{"just hours", 3 * time.Hour, "3h 0m 0s"},
		{"just days", 7 * 24 * time.Hour, "7d 0h 0m 0s"},
		{"negative", -1 * time.Second, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := snmp.FormatUptime(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatUptime(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestCollectDeviceInvalidIP(t *testing.T) {
	cfg := snmp.DefaultConfig()
	collector := snmp.NewCollector(cfg)
	ctx := context.Background()

	tests := []struct {
		name string
		ip   string
	}{
		{"empty", ""},
		{"hostname", "example.com"},
		{"invalid", "not-an-ip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.CollectDevice(ctx, tt.ip)

			if result.Success {
				t.Error("expected failure for invalid IP")
			}
			if result.Device.Status != snmp.StatusError {
				t.Errorf("expected StatusError, got %v", result.Device.Status)
			}
			if result.Error == "" {
				t.Error("expected error message")
			}
			if result.Device.Error == "" {
				t.Error("expected device error message")
			}
		})
	}
}

func TestCollectDeviceNilConfig(t *testing.T) {
	collector := snmp.NewCollector(nil)
	ctx := context.Background()

	result := collector.CollectDevice(ctx, "192.168.1.1")

	if result.Success {
		t.Error("expected failure for nil config")
	}
	if result.Device.Status != snmp.StatusError {
		t.Errorf("expected StatusError, got %v", result.Device.Status)
	}
	if result.Error != snmp.ErrNilConfig.Error() {
		t.Errorf("expected '%s' error, got %q", snmp.ErrNilConfig.Error(), result.Error)
	}
}

func TestCollectDeviceWithCommunityInvalidIP(t *testing.T) {
	cfg := snmp.DefaultConfig()
	collector := snmp.NewCollector(cfg)
	ctx := context.Background()

	result := collector.CollectDeviceWithCommunity(ctx, "", "public")

	if result.Success {
		t.Error("expected failure for invalid IP")
	}
	if result.Device.Status != snmp.StatusError {
		t.Errorf("expected StatusError, got %v", result.Device.Status)
	}
}

func TestCollectDeviceWithCommunityNilConfig(t *testing.T) {
	collector := snmp.NewCollector(nil)
	ctx := context.Background()

	result := collector.CollectDeviceWithCommunity(ctx, "192.168.1.1", "public")

	if result.Success {
		t.Error("expected failure for nil config")
	}
	if result.Device.Status != snmp.StatusError {
		t.Errorf("expected StatusError, got %v", result.Device.Status)
	}
}

func TestQueryInvalidIP(t *testing.T) {
	cfg := snmp.DefaultConfig()
	collector := snmp.NewCollector(cfg)
	ctx := context.Background()

	_, err := collector.Query(ctx, "", "1.3.6.1.2.1.1.1.0")
	if err == nil {
		t.Error("expected error for empty IP")
	}

	_, err = collector.Query(ctx, "invalid", "1.3.6.1.2.1.1.1.0")
	if err == nil {
		t.Error("expected error for invalid IP")
	}
}

func TestQueryNilConfig(t *testing.T) {
	collector := snmp.NewCollector(nil)
	ctx := context.Background()

	_, err := collector.Query(ctx, "192.168.1.1", "1.3.6.1.2.1.1.1.0")
	if err == nil {
		t.Error("expected error for nil config")
	}
	if !errors.Is(err, snmp.ErrNilConfig) {
		t.Errorf("expected ErrNilConfig, got %v", err)
	}
}

func TestQueryMultipleInvalidIP(t *testing.T) {
	cfg := snmp.DefaultConfig()
	collector := snmp.NewCollector(cfg)
	ctx := context.Background()

	oids := []string{"1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.5.0"}

	_, err := collector.QueryMultiple(ctx, "", oids)
	if err == nil {
		t.Error("expected error for empty IP")
	}

	_, err = collector.QueryMultiple(ctx, "invalid", oids)
	if err == nil {
		t.Error("expected error for invalid IP")
	}
}

func TestQueryMultipleNilConfig(t *testing.T) {
	collector := snmp.NewCollector(nil)
	ctx := context.Background()

	oids := []string{"1.3.6.1.2.1.1.1.0"}

	_, err := collector.QueryMultiple(ctx, "192.168.1.1", oids)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestDeviceInfoFields(t *testing.T) {
	now := time.Now()
	device := snmp.DeviceInfo{
		IP:           "192.168.1.1",
		SysName:      "test-switch",
		SysDescr:     "Cisco IOS Software",
		SysLocation:  "Data Center 1",
		SysContact:   "admin@example.com",
		SysUpTime:    24 * time.Hour,
		SysUpTimeSec: 86400,
		Status:       snmp.StatusSuccess,
		ResponseMs:   125.5,
		CollectedAt:  now,
		Error:        "",
	}

	if device.IP != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %q", device.IP)
	}
	if device.SysName != "test-switch" {
		t.Errorf("expected SysName 'test-switch', got %q", device.SysName)
	}
	if device.SysDescr != "Cisco IOS Software" {
		t.Errorf("expected SysDescr 'Cisco IOS Software', got %q", device.SysDescr)
	}
	if device.SysLocation != "Data Center 1" {
		t.Errorf("expected SysLocation 'Data Center 1', got %q", device.SysLocation)
	}
	if device.SysContact != "admin@example.com" {
		t.Errorf("expected SysContact 'admin@example.com', got %q", device.SysContact)
	}
	if device.SysUpTime != 24*time.Hour {
		t.Errorf("expected SysUpTime 24h, got %v", device.SysUpTime)
	}
	if device.SysUpTimeSec != 86400 {
		t.Errorf("expected SysUpTimeSec 86400, got %d", device.SysUpTimeSec)
	}
	if device.Status != snmp.StatusSuccess {
		t.Errorf("expected StatusSuccess, got %v", device.Status)
	}
	if device.ResponseMs != 125.5 {
		t.Errorf("expected ResponseMs 125.5, got %v", device.ResponseMs)
	}
	if device.CollectedAt != now {
		t.Errorf("expected CollectedAt %v, got %v", now, device.CollectedAt)
	}
	if device.Error != "" {
		t.Errorf("expected empty Error, got %q", device.Error)
	}
}

func TestInterfaceInfoFields(t *testing.T) {
	iface := snmp.InterfaceInfo{
		Index:       1,
		Name:        "eth0",
		Description: "Ethernet port 0",
		Type:        "ethernet-csmacd",
		Speed:       1000000000,
		AdminStatus: "up",
		OperStatus:  "up",
		InOctets:    1234567890,
		OutOctets:   987654321,
		InErrors:    5,
		OutErrors:   3,
	}

	if iface.Index != 1 {
		t.Errorf("expected Index 1, got %d", iface.Index)
	}
	if iface.Name != "eth0" {
		t.Errorf("expected Name 'eth0', got %q", iface.Name)
	}
	if iface.Description != "Ethernet port 0" {
		t.Errorf("expected Description 'Ethernet port 0', got %q", iface.Description)
	}
	if iface.Type != "ethernet-csmacd" {
		t.Errorf("expected Type 'ethernet-csmacd', got %q", iface.Type)
	}
	if iface.Speed != 1000000000 {
		t.Errorf("expected Speed 1000000000, got %d", iface.Speed)
	}
	if iface.AdminStatus != "up" {
		t.Errorf("expected AdminStatus 'up', got %q", iface.AdminStatus)
	}
	if iface.OperStatus != "up" {
		t.Errorf("expected OperStatus 'up', got %q", iface.OperStatus)
	}
	if iface.InOctets != 1234567890 {
		t.Errorf("expected InOctets 1234567890, got %d", iface.InOctets)
	}
	if iface.OutOctets != 987654321 {
		t.Errorf("expected OutOctets 987654321, got %d", iface.OutOctets)
	}
	if iface.InErrors != 5 {
		t.Errorf("expected InErrors 5, got %d", iface.InErrors)
	}
	if iface.OutErrors != 3 {
		t.Errorf("expected OutErrors 3, got %d", iface.OutErrors)
	}
}

func TestVLANInfoFields(t *testing.T) {
	vlan := snmp.VLANInfo{
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
		t.Errorf("expected 4 ports, got %d", len(vlan.Ports))
	}
}

func TestMACEntryFields(t *testing.T) {
	entry := snmp.MACEntry{
		MACAddress: "00:11:22:33:44:55",
		Port:       5,
		VLANID:     100,
		Type:       "dynamic",
	}

	if entry.MACAddress != "00:11:22:33:44:55" {
		t.Errorf("expected MACAddress '00:11:22:33:44:55', got %q", entry.MACAddress)
	}
	if entry.Port != 5 {
		t.Errorf("expected Port 5, got %d", entry.Port)
	}
	if entry.VLANID != 100 {
		t.Errorf("expected VLANID 100, got %d", entry.VLANID)
	}
	if entry.Type != "dynamic" {
		t.Errorf("expected Type 'dynamic', got %q", entry.Type)
	}
}

func TestCollectResultFields(t *testing.T) {
	device := &snmp.DeviceInfo{
		IP:     "192.168.1.1",
		Status: snmp.StatusSuccess,
	}

	result := snmp.CollectResult{
		Device:  device,
		Success: true,
		Error:   "",
	}

	if result.Device == nil {
		t.Fatal("expected non-nil Device")
	}
	if !result.Success {
		t.Error("expected Success true")
	}
	if result.Error != "" {
		t.Errorf("expected empty Error, got %q", result.Error)
	}
}

func TestThresholdsFields(t *testing.T) {
	thresholds := snmp.Thresholds{
		Warning:  250 * time.Millisecond,
		Critical: 1000 * time.Millisecond,
	}

	if thresholds.Warning != 250*time.Millisecond {
		t.Errorf("expected Warning 250ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 1000*time.Millisecond {
		t.Errorf("expected Critical 1000ms, got %v", thresholds.Critical)
	}
}

func TestThresholdsZeroValues(t *testing.T) {
	thresholds := snmp.Thresholds{}

	if thresholds.Warning != 0 {
		t.Error("expected Warning 0")
	}
	if thresholds.Critical != 0 {
		t.Error("expected Critical 0")
	}
}

func TestCollectorWithZeroThresholds(t *testing.T) {
	collector := snmp.NewCollector(snmp.DefaultConfig())

	zeroThresholds := snmp.Thresholds{
		Warning:  0,
		Critical: 0,
	}
	collector.SetThresholds(zeroThresholds)

	// With zero thresholds, any elapsed time >= 0 should be critical.
	status := collector.DetermineStatus(1*time.Millisecond, zeroThresholds)
	if status != snmp.StatusError {
		t.Errorf("expected StatusError with zero thresholds, got %v", status)
	}
}

func TestConcurrentCollectorAccess(t *testing.T) {
	t.Parallel()
	collector := snmp.NewCollector(snmp.DefaultConfig())

	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			for range 50 {
				thresholds := snmp.Thresholds{
					Warning:  time.Duration(id*100) * time.Millisecond,
					Critical: time.Duration(id*200) * time.Millisecond,
				}
				collector.SetThresholds(thresholds)
				_ = collector.GetThresholds()
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}
}

func TestDeviceInfoWithInterfaces(t *testing.T) {
	interfaces := []snmp.InterfaceInfo{
		{Index: 1, Name: "eth0", OperStatus: "up"},
		{Index: 2, Name: "eth1", OperStatus: "down"},
	}

	device := snmp.DeviceInfo{
		IP:         "192.168.1.1",
		Interfaces: interfaces,
	}

	if device.IP != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %q", device.IP)
	}
	if len(device.Interfaces) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(device.Interfaces))
	}
	if device.Interfaces[0].Name != "eth0" {
		t.Errorf("expected first interface 'eth0', got %q", device.Interfaces[0].Name)
	}
	if device.Interfaces[1].OperStatus != "down" {
		t.Errorf("expected second interface status 'down', got %q", device.Interfaces[1].OperStatus)
	}
}

func TestCollectResultWithError(t *testing.T) {
	device := &snmp.DeviceInfo{
		IP:     "192.168.1.1",
		Status: snmp.StatusError,
		Error:  "connection timeout",
	}

	result := snmp.CollectResult{
		Device:  device,
		Success: false,
		Error:   "connection timeout",
	}

	if result.Device.IP != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %q", result.Device.IP)
	}
	if result.Success {
		t.Error("expected Success false")
	}
	if result.Error != "connection timeout" {
		t.Errorf("expected Error 'connection timeout', got %q", result.Error)
	}
	if result.Device.Status != snmp.StatusError {
		t.Errorf("expected Device.Status StatusError, got %v", result.Device.Status)
	}
}

func TestDeviceInfoCollectedAt(t *testing.T) {
	before := time.Now()
	device := snmp.DeviceInfo{
		IP:          "192.168.1.1",
		CollectedAt: time.Now(),
	}
	after := time.Now()

	if device.IP != "192.168.1.1" {
		t.Errorf("expected IP '192.168.1.1', got %q", device.IP)
	}
	if device.CollectedAt.Before(before) {
		t.Error("CollectedAt should not be before the test started")
	}
	if device.CollectedAt.After(after) {
		t.Error("CollectedAt should not be after the test ended")
	}
}

func TestVLANInfoEmptyPorts(t *testing.T) {
	vlan := snmp.VLANInfo{
		ID:     10,
		Name:   "Empty VLAN",
		Status: "active",
		Ports:  nil,
	}

	if vlan.Ports != nil {
		t.Error("expected nil Ports")
	}
	if vlan.ID != 10 {
		t.Errorf("expected ID 10, got %d", vlan.ID)
	}
	if vlan.Name != "Empty VLAN" {
		t.Errorf("expected Name 'Empty VLAN', got %q", vlan.Name)
	}
	if vlan.Status != "active" {
		t.Errorf("expected Status 'active', got %q", vlan.Status)
	}
}

func TestMACEntryStaticType(t *testing.T) {
	entry := snmp.MACEntry{
		MACAddress: "AA:BB:CC:DD:EE:FF",
		Port:       1,
		VLANID:     1,
		Type:       "static",
	}

	if entry.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("expected MACAddress 'AA:BB:CC:DD:EE:FF', got %q", entry.MACAddress)
	}
	if entry.Port != 1 {
		t.Errorf("expected Port 1, got %d", entry.Port)
	}
	if entry.VLANID != 1 {
		t.Errorf("expected VLANID 1, got %d", entry.VLANID)
	}
	if entry.Type != "static" {
		t.Errorf("expected Type 'static', got %q", entry.Type)
	}
}

func TestInterfaceInfoZeroValues(t *testing.T) {
	iface := snmp.InterfaceInfo{}

	if iface.Index != 0 {
		t.Error("expected Index 0")
	}
	if iface.Name != "" {
		t.Error("expected empty Name")
	}
	if iface.Speed != 0 {
		t.Error("expected Speed 0")
	}
	if iface.InOctets != 0 {
		t.Error("expected InOctets 0")
	}
	if iface.OutOctets != 0 {
		t.Error("expected OutOctets 0")
	}
}

func TestCollectorMu(t *testing.T) {
	collector := snmp.NewCollector(snmp.DefaultConfig())
	mu := collector.CollectorMu()

	if mu == nil {
		t.Fatal("expected non-nil mutex")
	}

	// Verify the mutex works correctly by using it with a shared variable.
	var counter int
	mu.Lock()
	counter++
	mu.Unlock()

	mu.RLock()
	if counter != 1 {
		t.Errorf("expected counter 1, got %d", counter)
	}
	mu.RUnlock()
}

func TestDetermineStatusWithCustomThresholds(t *testing.T) {
	collector := snmp.NewCollector(snmp.DefaultConfig())

	// Test with very high thresholds.
	highThresholds := snmp.Thresholds{
		Warning:  1 * time.Hour,
		Critical: 2 * time.Hour,
	}

	status := collector.DetermineStatus(30*time.Second, highThresholds)
	if status != snmp.StatusSuccess {
		t.Errorf("expected StatusSuccess with high thresholds, got %v", status)
	}

	// Test with very low thresholds.
	lowThresholds := snmp.Thresholds{
		Warning:  1 * time.Nanosecond,
		Critical: 2 * time.Nanosecond,
	}

	status = collector.DetermineStatus(1*time.Millisecond, lowThresholds)
	if status != snmp.StatusError {
		t.Errorf("expected StatusError with low thresholds, got %v", status)
	}
}

func TestFormatUptimeLargeValues(t *testing.T) {
	// Test with very large uptime (1 year).
	oneYear := 365 * 24 * time.Hour
	result := snmp.FormatUptime(oneYear)
	if result == "" || result == "unknown" {
		t.Error("expected valid uptime string for 1 year")
	}

	// Test with 30 days.
	thirtyDays := 30 * 24 * time.Hour
	result = snmp.FormatUptime(thirtyDays)
	if result == "" || result == "unknown" {
		t.Error("expected valid uptime string for 30 days")
	}
}

func TestValidateConfigEdgeCases(t *testing.T) {
	// Test port at boundaries.
	cfg := &config.SNMPConfig{
		Port:        1,
		Timeout:     1 * time.Second,
		Communities: []string{"public"},
	}
	if err := snmp.ValidateConfig(cfg); err != nil {
		t.Errorf("port 1 should be valid: %v", err)
	}

	cfg.Port = 65535
	if err := snmp.ValidateConfig(cfg); err != nil {
		t.Errorf("port 65535 should be valid: %v", err)
	}
}

func TestCollectDeviceValidIPButNoConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := &config.SNMPConfig{
		Port:        161,
		Timeout:     100 * time.Millisecond, // Short timeout for faster test.
		Retries:     0,
		Communities: []string{"public"},
	}
	collector := snmp.NewCollector(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Use TEST-NET-1 which is not routable.
	result := collector.CollectDevice(ctx, "192.0.2.1")

	// Should fail to connect.
	if result.Success {
		t.Log("unexpectedly connected - skip this check")
		return
	}

	if result.Device.Status != snmp.StatusError {
		t.Errorf("expected StatusError for unreachable device, got %v", result.Device.Status)
	}
	if result.Device.IP != "192.0.2.1" {
		t.Errorf("expected IP '192.0.2.1', got %q", result.Device.IP)
	}
}
