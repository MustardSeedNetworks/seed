// Package sap_test provides extended tests for the sap module services.
// These tests focus on improving coverage for service methods and helper functions.
package sap_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/sap"
)

// =============================================================================
// Helper Function Tests - ConvertCableStatus
// =============================================================================

// TestConvertCableStatusWithType tests cable status conversion with typed inputs.
func TestConvertCableStatusWithType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    sap.CableStatusInput
		expected sap.CableStatus
	}{
		{"OK status", sap.CableStatusInputOK, sap.CableStatusOK},
		{"Open status", sap.CableStatusInputOpen, sap.CableStatusOpen},
		{"Short status", sap.CableStatusInputShort, sap.CableStatusShort},
		{"Impedance mismatch", sap.CableStatusInputImpedanceMismatch, sap.CableStatusImpedance},
		{"Crosstalk (maps to unknown)", sap.CableStatusInputCrosstalk, sap.CableStatusUnknown},
		{"Split pair (maps to unknown)", sap.CableStatusInputSplitPair, sap.CableStatusUnknown},
		{"Unknown status", sap.CableStatusInputUnknown, sap.CableStatusUnknown},
		{"Empty string", sap.CableStatusInput(""), sap.CableStatusUnknown},
		{"Invalid string", sap.CableStatusInput("invalid"), sap.CableStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sap.ConvertCableStatusWithType(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertCableStatusWithType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Helper Function Tests - ConvertPairResults
// =============================================================================

// TestConvertPairResultsExported tests pair result conversion.
func TestConvertPairResultsExported(t *testing.T) {
	t.Parallel()
	len25 := 25.0
	len10 := 10.0

	tests := []struct {
		name       string
		pairs      []sap.PairResultInput
		wantNil    bool
		wantLen    int
		checkFirst bool
		firstPair  int
	}{
		{
			name:    "empty slice",
			pairs:   []sap.PairResultInput{},
			wantNil: true,
		},
		{
			name:    "nil slice",
			pairs:   nil,
			wantNil: true,
		},
		{
			name: "single pair OK",
			pairs: []sap.PairResultInput{
				{Status: sap.CableStatusInputOK, LengthM: &len25},
			},
			wantLen:    1,
			checkFirst: true,
			firstPair:  1,
		},
		{
			name: "four pairs with mixed statuses",
			pairs: []sap.PairResultInput{
				{Status: sap.CableStatusInputOK, LengthM: &len25},
				{Status: sap.CableStatusInputOpen, LengthM: &len10},
				{Status: sap.CableStatusInputShort, LengthM: nil},
				{Status: sap.CableStatusInputUnknown, LengthM: nil},
			},
			wantLen:    4,
			checkFirst: true,
			firstPair:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sap.ConvertPairResultsExported(tt.pairs)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got %v", result)
				}
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("expected %d results, got %d", tt.wantLen, len(result))
				return
			}

			if tt.checkFirst && len(result) > 0 {
				if result[0].Pair != tt.firstPair {
					t.Errorf("expected first pair number %d, got %d", tt.firstPair, result[0].Pair)
				}
			}
		})
	}
}

// TestConvertPairResultsLength verifies length conversion.
func TestConvertPairResultsLength(t *testing.T) {
	t.Parallel()
	length := 25.5

	pairs := []sap.PairResultInput{
		{Status: sap.CableStatusInputOK, LengthM: &length},
	}

	result := sap.ConvertPairResultsExported(pairs)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	if result[0].Length != length {
		t.Errorf("expected length %f, got %f", length, result[0].Length)
	}
}

// TestConvertPairResultsNilLength verifies nil length handling.
func TestConvertPairResultsNilLength(t *testing.T) {
	t.Parallel()

	pairs := []sap.PairResultInput{
		{Status: sap.CableStatusInputOK, LengthM: nil},
	}

	result := sap.ConvertPairResultsExported(pairs)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	if result[0].Length != 0 {
		t.Errorf("expected length 0 for nil input, got %f", result[0].Length)
	}
}

// =============================================================================
// Helper Function Tests - ConvertGatewayStatus
// =============================================================================

// TestConvertGatewayStatusWithType tests gateway status conversion with typed inputs.
func TestConvertGatewayStatusWithType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    sap.GatewayStatusInput
		expected sap.HealthStatus
	}{
		{"Success status", sap.GatewayStatusInputSuccess, sap.HealthStatusHealthy},
		{"Warning status", sap.GatewayStatusInputWarning, sap.HealthStatusDegraded},
		{"Error status", sap.GatewayStatusInputError, sap.HealthStatusUnhealthy},
		{"Unknown status", sap.GatewayStatusInputUnknown, sap.HealthStatusUnknown},
		{"Empty string", sap.GatewayStatusInput(""), sap.HealthStatusUnknown},
		{"Invalid string", sap.GatewayStatusInput("foo"), sap.HealthStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sap.ConvertGatewayStatusWithType(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertGatewayStatusWithType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// LinkService Extended Tests
// =============================================================================

// TestLinkServiceGetInterfaceStatusNonexistentExt tests getting status for a non-existent interface.
func TestLinkServiceGetInterfaceStatusNonexistentExt(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewLinkService(cfg)

	ctx := context.Background()
	// Test with a non-existent interface
	status, err := service.GetInterfaceStatus(ctx, "nonexistent_interface_xyz123")
	// May return error or status depending on platform
	if err != nil {
		t.Logf("GetInterfaceStatus for non-existent interface returned error (expected): %v", err)
	}
	if status != nil {
		t.Logf("GetInterfaceStatus returned status: %+v", status)
		if status.Interface != "nonexistent_interface_xyz123" {
			t.Errorf("expected interface 'nonexistent_interface_xyz123', got %q", status.Interface)
		}
	}
}

// TestLinkServiceGetInterfaceStatusKnownInterface tests with a likely-existing interface.
func TestLinkServiceGetInterfaceStatusKnownInterface(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewLinkService(cfg)

	ctx := context.Background()
	// Try common interface names
	interfaces := []string{"lo0", "lo", "en0", "eth0"}
	for _, iface := range interfaces {
		status, err := service.GetInterfaceStatus(ctx, iface)
		if err == nil && status != nil {
			t.Logf("Got status for %s: State=%s", iface, status.State)
			if status.Interface != iface {
				t.Errorf("expected interface %q, got %q", iface, status.Interface)
			}
			return // Found a working interface
		}
	}
	t.Log("No common interfaces found - this is OK for some environments")
}

// TestLinkServiceStartAndGetStatus tests the full lifecycle of LinkService.
func TestLinkServiceStartAndGetStatus(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Interface.Default = "lo0" // Use loopback on macOS
	service := sap.NewLinkService(cfg)

	ctx := context.Background()
	// Start may fail if interface doesn't exist
	err := service.Start(ctx)
	if err != nil {
		t.Logf("Start returned error (may be expected): %v", err)
		return
	}

	// Get status should now work
	statuses, getErr := service.GetStatus(ctx)
	if getErr != nil {
		t.Logf("GetStatus returned error: %v", getErr)
	}
	if statuses != nil {
		t.Logf("Got %d interface statuses", len(statuses))
	}

	service.Stop()
}

// =============================================================================
// CableService Extended Tests
// =============================================================================

// TestCableServiceTestTableDriven tests cable testing with various interfaces.
func TestCableServiceTestTableDriven(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewCableService(cfg)

	tests := []struct {
		name      string
		iface     string
		expectErr bool
	}{
		{"non-existent interface", "nonexistent_xyz123", true},
		{"eth0 (may not exist)", "eth0", true},
		{"en0 (may not exist)", "en0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := service.Test(ctx, tt.iface)
			if err != nil {
				t.Logf("Test returned error: %v", err)
			}
			if result != nil {
				t.Logf("Test result: Interface=%s, Status=%s", result.Interface, result.Status)
				if result.Interface != tt.iface {
					t.Errorf("expected interface %q, got %q", tt.iface, result.Interface)
				}
			}
		})
	}
}

// =============================================================================
// VLANService Extended Tests
// =============================================================================

// TestVLANServiceCreateDelete tests VLAN creation and deletion (will fail without privileges).
func TestVLANServiceCreateDelete(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewVLANService(cfg)

	ctx := context.Background()

	// Create will likely fail without root/admin privileges
	err := service.Create(ctx, "eth0", 100)
	if err != nil {
		t.Logf("Create VLAN returned error (expected without privileges): %v", err)
	}

	// Delete will also likely fail
	err = service.Delete(ctx, "eth0.100")
	if err != nil {
		t.Logf("Delete VLAN returned error (expected without privileges): %v", err)
	}
}

// TestVLANServiceListWithConfigs tests List with actual manager.
func TestVLANServiceListWithConfigs(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewVLANService(cfg)

	ctx := context.Background()
	configs, err := service.List(ctx)
	if err != nil {
		t.Logf("List returned error: %v", err)
	}
	// Result depends on system configuration
	t.Logf("List returned %d VLAN configs", len(configs))
}

// =============================================================================
// PerformanceService Extended Tests
// =============================================================================

// TestPerformanceServiceIPerfWithOptions tests IPerf with various options.
func TestPerformanceServiceIPerfWithOptions(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewPerformanceService(cfg)

	tests := []struct {
		name    string
		host    string
		options map[string]any
	}{
		{
			name: "basic TCP test",
			host: "localhost",
			options: map[string]any{
				"port":     5201,
				"duration": 1,
				"protocol": "tcp",
			},
		},
		{
			name: "UDP test with reverse",
			host: "127.0.0.1",
			options: map[string]any{
				"port":     5201,
				"duration": 1,
				"protocol": "udp",
				"reverse":  true,
			},
		},
		{
			name:    "empty options",
			host:    "localhost",
			options: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Will fail without iPerf server running
			result, err := service.IPerf(ctx, tt.host, tt.options)
			if err != nil {
				t.Logf("IPerf returned error (expected without server): %v", err)
			}
			if result != nil {
				t.Logf("IPerf result: Protocol=%s, Bandwidth=%.2f Mbps", result.Protocol, result.BandwidthMbps)
			}
		})
	}

	service.Stop()
}

// TestPerformanceServiceSpeedtest tests speedtest execution.
func TestPerformanceServiceSpeedtest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping speedtest in short mode")
	}
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewPerformanceService(cfg)
	defer service.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Will likely fail or timeout without network access
	result, err := service.Speedtest(ctx)
	if err != nil {
		t.Logf("Speedtest returned error (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("Speedtest result: Download=%.2f Mbps, Upload=%.2f Mbps", result.DownloadMbps, result.UploadMbps)
	}
}

// =============================================================================
// GatewayService Extended Tests
// =============================================================================

// TestGatewayServiceGetHealth tests health retrieval after start.
func TestGatewayServiceGetHealth(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	service := sap.NewGatewayService(cfg)

	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		t.Logf("Start returned error: %v", err)
	}

	health, err := service.GetHealth(ctx)
	if err != nil {
		t.Logf("GetHealth returned error: %v", err)
	}
	if health != nil {
		t.Logf("Gateway health: IP=%s, Reachable=%v, Status=%s", health.IP, health.Reachable, health.Status)
	}

	service.Stop()
}

// =============================================================================
// Module Extended Tests
// =============================================================================

// TestModuleStartSuccessful tests successful module start with valid config.
func TestModuleStartSuccessful(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	// Use loopback interface which should exist
	cfg.Interface.Default = "lo0"
	cfg.Interface.Fallbacks = []string{"lo", "lo0"}

	module := sap.New(cfg, nil)
	ctx := context.Background()

	// Start may succeed or fail depending on platform
	err := module.Start(ctx)
	if err != nil {
		t.Logf("Start returned error (may be expected): %v", err)
	}

	// Stop should always succeed
	if stopErr := module.Stop(); stopErr != nil {
		t.Errorf("Stop returned error: %v", stopErr)
	}
}

// TestModuleConcurrentStartStopParallel tests concurrent start/stop operations in parallel.
func TestModuleConcurrentStartStopParallel(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()

	const iterations = 5
	var wg sync.WaitGroup
	wg.Add(iterations)

	for range iterations {
		go func() {
			defer wg.Done()
			module := sap.New(cfg, nil)
			ctx := context.Background()
			_ = module.Start(ctx)
			_ = module.Stop()
		}()
	}

	wg.Wait()
}

// =============================================================================
// Make Helper Tests - Extended
// =============================================================================

// TestMakePairResult tests PairResult creation helper.
func TestMakePairResult(t *testing.T) {
	t.Parallel()
	result := sap.MakePairResult(1, sap.CableStatusOK, 25.5, 100.0)

	if result.Pair != 1 {
		t.Errorf("expected Pair 1, got %d", result.Pair)
	}
	if result.Status != sap.CableStatusOK {
		t.Errorf("expected Status %q, got %q", sap.CableStatusOK, result.Status)
	}
	if result.Length != 25.5 {
		t.Errorf("expected Length 25.5, got %f", result.Length)
	}
	if result.Impedance != 100.0 {
		t.Errorf("expected Impedance 100.0, got %f", result.Impedance)
	}
}

// TestMakeCableTestResultWithPairs tests CableTestResult with pairs helper.
func TestMakeCableTestResultWithPairs(t *testing.T) {
	t.Parallel()
	pairs := []sap.PairResult{
		sap.MakePairResult(1, sap.CableStatusOK, 25.0, 100.0),
		sap.MakePairResult(2, sap.CableStatusOK, 25.2, 100.0),
		sap.MakePairResult(3, sap.CableStatusOpen, 10.0, 75.0),
		sap.MakePairResult(4, sap.CableStatusOK, 25.1, 100.0),
	}

	result := sap.MakeCableTestResultWithPairs("eth0", sap.CableStatusOpen, 25.0, pairs)

	if result.Interface != "eth0" {
		t.Errorf("expected Interface %q, got %q", "eth0", result.Interface)
	}
	if len(result.PairResults) != 4 {
		t.Errorf("expected 4 pair results, got %d", len(result.PairResults))
	}
	if result.TestedAt.IsZero() {
		t.Error("expected non-zero TestedAt")
	}
}

// TestMakeDNSAnswer tests DNSAnswer creation helper.
func TestMakeDNSAnswer(t *testing.T) {
	t.Parallel()
	answer := sap.MakeDNSAnswer("google.com", "A", "142.250.80.14", 300)

	if answer.Name != "google.com" {
		t.Errorf("expected Name %q, got %q", "google.com", answer.Name)
	}
	if answer.Type != "A" {
		t.Errorf("expected Type %q, got %q", "A", answer.Type)
	}
	if answer.Value != "142.250.80.14" {
		t.Errorf("expected Value %q, got %q", "142.250.80.14", answer.Value)
	}
	if answer.TTL != 300 {
		t.Errorf("expected TTL 300, got %d", answer.TTL)
	}
}

// TestMakeDNSTestResultWithAnswers tests DNSTestResult with answers helper.
func TestMakeDNSTestResultWithAnswers(t *testing.T) {
	t.Parallel()
	answers := []sap.DNSAnswer{
		sap.MakeDNSAnswer("google.com", "A", "142.250.80.14", 300),
		sap.MakeDNSAnswer("google.com", "A", "142.250.80.46", 300),
	}

	result := sap.MakeDNSTestResultWithAnswers("google.com", "8.8.8.8", true, 15.0, answers)

	if result.Query != "google.com" {
		t.Errorf("expected Query %q, got %q", "google.com", result.Query)
	}
	if len(result.Answers) != 2 {
		t.Errorf("expected 2 answers, got %d", len(result.Answers))
	}
}

// TestMakeDHCPTestResultFull tests full DHCPTestResult helper.
func TestMakeDHCPTestResultFull(t *testing.T) {
	t.Parallel()
	dnsServers := []string{"8.8.8.8", "8.8.4.4"}
	result := sap.MakeDHCPTestResultFull(
		true,
		"192.168.1.1",
		"192.168.1.100",
		"192.168.1.1",
		dnsServers,
		86400,
		"",
	)

	if !result.Success {
		t.Error("expected Success true")
	}
	if len(result.DNSServers) != 2 {
		t.Errorf("expected 2 DNS servers, got %d", len(result.DNSServers))
	}
	if result.LeaseTimeSec != 86400 {
		t.Errorf("expected LeaseTimeSec 86400, got %d", result.LeaseTimeSec)
	}
	if result.Error != "" {
		t.Errorf("expected empty Error, got %q", result.Error)
	}
}

// TestMakeDHCPTestResultFullWithError tests DHCPTestResult with error.
func TestMakeDHCPTestResultFullWithError(t *testing.T) {
	t.Parallel()
	result := sap.MakeDHCPTestResultFull(
		false,
		"",
		"",
		"",
		nil,
		0,
		"DHCP timeout",
	)

	if result.Success {
		t.Error("expected Success false")
	}
	if result.Error != "DHCP timeout" {
		t.Errorf("expected Error 'DHCP timeout', got %q", result.Error)
	}
}

// TestMakeSNMPInterface tests SNMPInterface creation helper.
func TestMakeSNMPInterface(t *testing.T) {
	t.Parallel()
	iface := sap.MakeSNMPInterface(
		1,
		"GigabitEthernet0/1",
		"Uplink to Core",
		"ethernetCsmacd",
		1000000000,
		"up",
		"up",
	)

	if iface.Index != 1 {
		t.Errorf("expected Index 1, got %d", iface.Index)
	}
	if iface.Name != "GigabitEthernet0/1" {
		t.Errorf("expected Name %q, got %q", "GigabitEthernet0/1", iface.Name)
	}
	if iface.Speed != 1000000000 {
		t.Errorf("expected Speed 1000000000, got %d", iface.Speed)
	}
}

// TestMakeSNMPVLAN tests SNMPVLAN creation helper.
func TestMakeSNMPVLAN(t *testing.T) {
	t.Parallel()
	vlan := sap.MakeSNMPVLAN(100, "Management", "active", []int{1, 2, 3, 4})

	if vlan.ID != 100 {
		t.Errorf("expected ID 100, got %d", vlan.ID)
	}
	if vlan.Name != "Management" {
		t.Errorf("expected Name %q, got %q", "Management", vlan.Name)
	}
	if len(vlan.Ports) != 4 {
		t.Errorf("expected 4 ports, got %d", len(vlan.Ports))
	}
}

// TestMakeMACTableEntry tests MACTableEntry creation helper.
func TestMakeMACTableEntry(t *testing.T) {
	t.Parallel()
	entry := sap.MakeMACTableEntry("00:11:22:33:44:55", 1, 100, "dynamic")

	if entry.MACAddress != "00:11:22:33:44:55" {
		t.Errorf("expected MACAddress %q, got %q", "00:11:22:33:44:55", entry.MACAddress)
	}
	if entry.Port != 1 {
		t.Errorf("expected Port 1, got %d", entry.Port)
	}
	if entry.VLANID != 100 {
		t.Errorf("expected VLANID 100, got %d", entry.VLANID)
	}
	if entry.Type != "dynamic" {
		t.Errorf("expected Type %q, got %q", "dynamic", entry.Type)
	}
}

// TestMakeSNMPDeviceFull tests full SNMPDevice creation helper.
func TestMakeSNMPDeviceFull(t *testing.T) {
	t.Parallel()
	interfaces := []sap.SNMPInterface{
		sap.MakeSNMPInterface(1, "Gi0/1", "Uplink", "ethernetCsmacd", 1000000000, "up", "up"),
	}
	vlans := []sap.SNMPVLAN{
		sap.MakeSNMPVLAN(1, "default", "active", nil),
	}
	macTable := []sap.MACTableEntry{
		sap.MakeMACTableEntry("00:11:22:33:44:55", 1, 1, "dynamic"),
	}

	device := sap.MakeSNMPDeviceFull(
		"192.168.1.1",
		"switch-01",
		"Cisco IOS",
		"Server Room",
		"admin@example.com",
		24*time.Hour,
		interfaces,
		vlans,
		macTable,
	)

	if device.IP != "192.168.1.1" {
		t.Errorf("expected IP %q, got %q", "192.168.1.1", device.IP)
	}
	if device.SysLocation != "Server Room" {
		t.Errorf("expected SysLocation %q, got %q", "Server Room", device.SysLocation)
	}
	if len(device.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(device.Interfaces))
	}
	if len(device.VLANs) != 1 {
		t.Errorf("expected 1 VLAN, got %d", len(device.VLANs))
	}
	if len(device.MACTable) != 1 {
		t.Errorf("expected 1 MAC entry, got %d", len(device.MACTable))
	}
}

// TestMakeGatewayHealthFull tests full GatewayHealth creation helper.
func TestMakeGatewayHealthFull(t *testing.T) {
	t.Parallel()
	health := sap.MakeGatewayHealthFull(
		"192.168.1.1",
		true,
		5.0,
		0.0,
		0.5,
		sap.HealthStatusHealthy,
		24*time.Hour,
	)

	if health.IP != "192.168.1.1" {
		t.Errorf("expected IP %q, got %q", "192.168.1.1", health.IP)
	}
	if health.Jitter != 0.5 {
		t.Errorf("expected Jitter 0.5, got %f", health.Jitter)
	}
	if health.Uptime != 24*time.Hour {
		t.Errorf("expected Uptime 24h, got %v", health.Uptime)
	}
}

// TestMakeSpeedtestResultFull tests full SpeedtestResult creation helper.
func TestMakeSpeedtestResultFull(t *testing.T) {
	t.Parallel()
	result := sap.MakeSpeedtestResultFull(
		100.0,
		50.0,
		10.0,
		2.0,
		"Comcast Speed",
		"12345",
		"Comcast",
		30*time.Second,
	)

	if result.DownloadMbps != 100.0 {
		t.Errorf("expected DownloadMbps 100.0, got %f", result.DownloadMbps)
	}
	if result.JitterMs != 2.0 {
		t.Errorf("expected JitterMs 2.0, got %f", result.JitterMs)
	}
	if result.ServerID != "12345" {
		t.Errorf("expected ServerID %q, got %q", "12345", result.ServerID)
	}
	if result.ISP != "Comcast" {
		t.Errorf("expected ISP %q, got %q", "Comcast", result.ISP)
	}
	if result.TestDuration != 30*time.Second {
		t.Errorf("expected TestDuration 30s, got %v", result.TestDuration)
	}
}

// TestMakeIPerfResultFull tests full IPerfResult creation helper.
func TestMakeIPerfResultFull(t *testing.T) {
	t.Parallel()
	result := sap.MakeIPerfResultFull(
		"tcp",
		"upload",
		950.0,
		1187.5,
		10.0,
		1.5,
		0.1,
		5,
		"192.168.1.100",
	)

	if result.Protocol != "tcp" {
		t.Errorf("expected Protocol %q, got %q", "tcp", result.Protocol)
	}
	if result.Jitter != 1.5 {
		t.Errorf("expected Jitter 1.5, got %f", result.Jitter)
	}
	if result.PacketLoss != 0.1 {
		t.Errorf("expected PacketLoss 0.1, got %f", result.PacketLoss)
	}
	if result.Retransmits != 5 {
		t.Errorf("expected Retransmits 5, got %d", result.Retransmits)
	}
}

// TestMakeVLANConfigFull tests full VLANConfig creation helper.
func TestMakeVLANConfigFull(t *testing.T) {
	t.Parallel()
	config := sap.MakeVLANConfigFull(
		100,
		"Management",
		"eth0",
		"192.168.100.1",
		"255.255.255.0",
		"192.168.100.254",
		true,
		[]string{"eth0", "eth1", "eth2"},
	)

	if config.ID != 100 {
		t.Errorf("expected ID 100, got %d", config.ID)
	}
	if config.IPAddress != "192.168.100.1" {
		t.Errorf("expected IPAddress %q, got %q", "192.168.100.1", config.IPAddress)
	}
	if config.SubnetMask != "255.255.255.0" {
		t.Errorf("expected SubnetMask %q, got %q", "255.255.255.0", config.SubnetMask)
	}
	if config.Gateway != "192.168.100.254" {
		t.Errorf("expected Gateway %q, got %q", "192.168.100.254", config.Gateway)
	}
	if len(config.MemberPorts) != 3 {
		t.Errorf("expected 3 member ports, got %d", len(config.MemberPorts))
	}
}

// TestMakeLinkStatusFull tests full LinkStatus creation helper.
func TestMakeLinkStatusFull(t *testing.T) {
	t.Parallel()
	status := sap.MakeLinkStatusFull(
		"eth0",
		sap.LinkStateUp,
		"1000Mbps",
		"full",
		1500,
		"00:11:22:33:44:55",
		"192.168.1.100",
		"192.168.1.1",
		1000000,
		5000000,
		1000,
		5000,
		0,
		0,
		0,
		0,
	)

	if status.Interface != "eth0" {
		t.Errorf("expected Interface %q, got %q", "eth0", status.Interface)
	}
	if status.IPAddress != "192.168.1.100" {
		t.Errorf("expected IPAddress %q, got %q", "192.168.1.100", status.IPAddress)
	}
	if status.Gateway != "192.168.1.1" {
		t.Errorf("expected Gateway %q, got %q", "192.168.1.1", status.Gateway)
	}
	if status.TxBytes != 1000000 {
		t.Errorf("expected TxBytes 1000000, got %d", status.TxBytes)
	}
	if status.RxBytes != 5000000 {
		t.Errorf("expected RxBytes 5000000, got %d", status.RxBytes)
	}
	if !status.Carrier {
		t.Error("expected Carrier true for LinkStateUp")
	}
}

// TestMakeLinkStatusFullDown tests full LinkStatus for down interface.
func TestMakeLinkStatusFullDown(t *testing.T) {
	t.Parallel()
	status := sap.MakeLinkStatusFull(
		"eth1",
		sap.LinkStateDown,
		"",
		"",
		1500,
		"00:11:22:33:44:66",
		"",
		"",
		0, 0, 0, 0, 0, 0, 0, 0,
	)

	if status.State != sap.LinkStateDown {
		t.Errorf("expected State %q, got %q", sap.LinkStateDown, status.State)
	}
	if status.Carrier {
		t.Error("expected Carrier false for LinkStateDown")
	}
}

// TestMakeTelemetrySnapshotFull tests full TelemetrySnapshot creation helper.
func TestMakeTelemetrySnapshotFull(t *testing.T) {
	t.Parallel()
	links := []sap.LinkStatus{
		sap.MakeLinkStatus("eth0", sap.LinkStateUp, "1000", "full", 1500, "00:11:22:33:44:55"),
	}
	gateway := &sap.GatewayHealth{IP: "192.168.1.1", Reachable: true, Status: sap.HealthStatusHealthy}
	dns := &sap.DNSTestResult{Query: "google.com", Success: true}
	dhcp := &sap.DHCPTestResult{Success: true}
	bandwidth := &sap.BandwidthSample{Interface: "eth0", TxMbps: 100.0, RxMbps: 50.0}
	systemHealth := &sap.SystemHealth{CPUPercent: 25.0, MemoryPercent: 50.0}

	snapshot := sap.MakeTelemetrySnapshotFull(links, gateway, dns, dhcp, bandwidth, systemHealth)

	if len(snapshot.Links) != 1 {
		t.Errorf("expected 1 link, got %d", len(snapshot.Links))
	}
	if snapshot.Gateway == nil {
		t.Error("expected non-nil Gateway")
	}
	if snapshot.DNS == nil {
		t.Error("expected non-nil DNS")
	}
	if snapshot.DHCP == nil {
		t.Error("expected non-nil DHCP")
	}
	if snapshot.Bandwidth == nil {
		t.Error("expected non-nil Bandwidth")
	}
	if snapshot.SystemHealth == nil {
		t.Error("expected non-nil SystemHealth")
	}
	if snapshot.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}

// TestMakeSystemHealthFull tests full SystemHealth creation helper.
func TestMakeSystemHealthFull(t *testing.T) {
	t.Parallel()
	health := sap.MakeSystemHealthFull(
		25.0,
		50.0,
		60.0,
		45.0,
		7*24*time.Hour,
		[]float64{1.0, 0.8, 0.9},
	)

	if health.CPUPercent != 25.0 {
		t.Errorf("expected CPUPercent 25.0, got %f", health.CPUPercent)
	}
	if health.Temperature != 45.0 {
		t.Errorf("expected Temperature 45.0, got %f", health.Temperature)
	}
	if health.Uptime != 7*24*time.Hour {
		t.Errorf("expected Uptime 7 days, got %v", health.Uptime)
	}
	if len(health.LoadAverage) != 3 {
		t.Errorf("expected 3 load average values, got %d", len(health.LoadAverage))
	}
	if health.SampledAt.IsZero() {
		t.Error("expected non-zero SampledAt")
	}
}

// =============================================================================
// Benchmark Tests - Extended
// =============================================================================

// BenchmarkConvertCableStatus benchmarks cable status conversion.
func BenchmarkConvertCableStatus(b *testing.B) {
	statuses := []sap.CableStatusInput{
		sap.CableStatusInputOK,
		sap.CableStatusInputOpen,
		sap.CableStatusInputShort,
		sap.CableStatusInputImpedanceMismatch,
		sap.CableStatusInputUnknown,
	}
	b.ResetTimer()

	for b.Loop() {
		for _, s := range statuses {
			_ = sap.ConvertCableStatusWithType(s)
		}
	}
}

// BenchmarkConvertPairResults benchmarks pair results conversion.
func BenchmarkConvertPairResults(b *testing.B) {
	len25 := 25.0
	pairs := []sap.PairResultInput{
		{Status: sap.CableStatusInputOK, LengthM: &len25},
		{Status: sap.CableStatusInputOK, LengthM: &len25},
		{Status: sap.CableStatusInputOK, LengthM: &len25},
		{Status: sap.CableStatusInputOK, LengthM: &len25},
	}
	b.ResetTimer()

	for b.Loop() {
		_ = sap.ConvertPairResultsExported(pairs)
	}
}

// BenchmarkMakeLinkStatusFull benchmarks full link status creation.
func BenchmarkMakeLinkStatusFull(b *testing.B) {
	b.ResetTimer()

	for b.Loop() {
		_ = sap.MakeLinkStatusFull(
			"eth0", sap.LinkStateUp, "1000", "full", 1500, "00:11:22:33:44:55",
			"192.168.1.100", "192.168.1.1",
			1000000, 5000000, 1000, 5000, 0, 0, 0, 0,
		)
	}
}
