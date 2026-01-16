// Test suite validates iperf3 client/server operations, result parsing, and bandwidth measurement.
package iperf_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/iperf"
)

func TestNewManager(t *testing.T) {
	manager := iperf.NewManager()

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	status := manager.GetClientStatus()
	if status.Phase != "idle" {
		t.Errorf("expected initial phase 'idle', got %q", status.Phase)
	}
	if status.Running {
		t.Error("expected Running to be false initially")
	}
}

func TestManagerGetServerStatus(t *testing.T) {
	manager := iperf.NewManager()

	status := manager.GetServerStatus()
	if status.Running {
		t.Error("expected server to not be running initially")
	}
	if status.Port != 0 {
		t.Errorf("expected port 0, got %d", status.Port)
	}
	if status.PID != 0 {
		t.Errorf("expected PID 0, got %d", status.PID)
	}
}

func TestManagerGetClientStatus(t *testing.T) {
	manager := iperf.NewManager()

	status := manager.GetClientStatus()
	if status.Running {
		t.Error("expected client to not be running initially")
	}
	if status.Phase != "idle" {
		t.Errorf("expected phase 'idle', got %q", status.Phase)
	}
	if status.Progress != 0 {
		t.Errorf("expected progress 0, got %v", status.Progress)
	}
}

func TestManagerGetLastResult(t *testing.T) {
	manager := iperf.NewManager()

	result := manager.GetLastResult()
	if result != nil {
		t.Error("expected nil result initially")
	}
}

func TestClientConfigDefaults(t *testing.T) {
	config := iperf.ClientConfig{
		Server: "localhost",
	}

	if config.Server != "localhost" {
		t.Errorf("expected server 'localhost', got %q", config.Server)
	}
	// Check zero values for unset fields
	if config.Port != 0 {
		t.Errorf("expected port 0 (default), got %d", config.Port)
	}
	if config.Duration != 0 {
		t.Errorf("expected duration 0 (default), got %d", config.Duration)
	}
	if config.Parallel != 0 {
		t.Errorf("expected parallel 0 (default), got %d", config.Parallel)
	}
	if config.Protocol != "" {
		t.Errorf("expected protocol '' (default), got %q", config.Protocol)
	}
	if config.Reverse {
		t.Error("expected reverse false (default)")
	}
}

func TestClientConfigWithValues(t *testing.T) {
	config := iperf.ClientConfig{
		Server:   "192.168.1.100",
		Port:     5201,
		Protocol: "tcp",
		Reverse:  true,
		Duration: 10,
		Parallel: 4,
	}

	if config.Server != "192.168.1.100" {
		t.Errorf("expected server '192.168.1.100', got %q", config.Server)
	}
	if config.Port != 5201 {
		t.Errorf("expected port 5201, got %d", config.Port)
	}
	if config.Protocol != "tcp" {
		t.Errorf("expected protocol 'tcp', got %q", config.Protocol)
	}
	if !config.Reverse {
		t.Error("expected reverse true")
	}
	if config.Duration != 10 {
		t.Errorf("expected duration 10, got %d", config.Duration)
	}
	if config.Parallel != 4 {
		t.Errorf("expected parallel 4, got %d", config.Parallel)
	}
}

func TestResultFields(t *testing.T) {
	now := time.Now()
	result := iperf.Result{
		BitsPerSecond: 100_000_000,
		Bandwidth:     100.0,
		Transfer:      125.0,
		Retransmits:   5,
		Jitter:        1.5,
		LostPackets:   2,
		LostPercent:   0.5,
		Protocol:      "tcp",
		Direction:     "download",
		Duration:      10.0,
		Server:        "speedtest.example.com",
		Port:          5201,
		Timestamp:     now,
	}

	if result.BitsPerSecond != 100_000_000 {
		t.Errorf("expected BitsPerSecond 100_000_000, got %v", result.BitsPerSecond)
	}
	if result.Bandwidth != 100.0 {
		t.Errorf("expected Bandwidth 100.0, got %v", result.Bandwidth)
	}
	if result.Transfer != 125.0 {
		t.Errorf("expected Transfer 125.0, got %v", result.Transfer)
	}
	if result.Retransmits != 5 {
		t.Errorf("expected Retransmits 5, got %d", result.Retransmits)
	}
	if result.Jitter != 1.5 {
		t.Errorf("expected Jitter 1.5, got %v", result.Jitter)
	}
	if result.LostPackets != 2 {
		t.Errorf("expected LostPackets 2, got %d", result.LostPackets)
	}
	if result.LostPercent != 0.5 {
		t.Errorf("expected LostPercent 0.5, got %v", result.LostPercent)
	}
	if result.Protocol != "tcp" {
		t.Errorf("expected Protocol 'tcp', got %q", result.Protocol)
	}
	if result.Direction != "download" {
		t.Errorf("expected Direction 'download', got %q", result.Direction)
	}
	if result.Duration != 10.0 {
		t.Errorf("expected Duration 10.0, got %v", result.Duration)
	}
	if result.Server != "speedtest.example.com" {
		t.Errorf("expected Server 'speedtest.example.com', got %q", result.Server)
	}
	if result.Port != 5201 {
		t.Errorf("expected Port 5201, got %d", result.Port)
	}
	if result.Timestamp != now {
		t.Errorf("expected Timestamp %v, got %v", now, result.Timestamp)
	}
}

func TestServerStatus(t *testing.T) {
	status := iperf.ServerStatus{
		Running: true,
		Port:    5201,
		PID:     12345,
		Error:   "",
	}

	if !status.Running {
		t.Error("expected Running true")
	}
	if status.Port != 5201 {
		t.Errorf("expected Port 5201, got %d", status.Port)
	}
	if status.PID != 12345 {
		t.Errorf("expected PID 12345, got %d", status.PID)
	}
	if status.Error != "" {
		t.Errorf("expected empty Error, got %q", status.Error)
	}
}

func TestClientStatus(t *testing.T) {
	status := iperf.ClientStatus{
		Running:  true,
		Phase:    "testing",
		Progress: 50.0,
	}

	if !status.Running {
		t.Error("expected Running true")
	}
	if status.Phase != "testing" {
		t.Errorf("expected Phase 'testing', got %q", status.Phase)
	}
	if status.Progress != 50.0 {
		t.Errorf("expected Progress 50.0, got %v", status.Progress)
	}
}

func TestManagerStopServerNotRunning(t *testing.T) {
	manager := iperf.NewManager()

	err := manager.StopServer()
	if err == nil {
		t.Error("expected error when stopping non-running server")
	}
	if err.Error() != "server not running" {
		t.Errorf("expected 'server not running' error, got %q", err.Error())
	}
}

func TestManagerRunClientAlreadyRunning(t *testing.T) {
	manager := iperf.NewManager()

	// Manually set client as running using exported helper
	manager.SetManagerClientStatusRunning(true)

	ctx := context.Background()
	_, err := manager.RunClient(ctx, &iperf.ClientConfig{Server: "localhost"})
	if err == nil {
		t.Error("expected error when test already in progress")
	}
	if err.Error() != "test already in progress" {
		t.Errorf("expected 'test already in progress' error, got %q", err.Error())
	}

	// Clean up
	manager.SetManagerClientStatusRunning(false)
}

func TestManagerServerAlreadyRunning(t *testing.T) {
	manager := iperf.NewManager()

	// Manually set server as running using exported helper
	manager.SetManagerServerStatusRunning(true, 5201)

	err := manager.StartServer(5201)
	if err == nil {
		t.Error("expected error when server already running")
	}

	// Clean up
	manager.SetManagerServerStatusRunning(false, 0)
}

func TestCheckInstalled(_ *testing.T) {
	// This test may fail if iperf3 is not installed, which is okay
	err := iperf.CheckInstalled()
	// Just check it doesn't panic - the result depends on system configuration
	_ = err
}

func TestGetVersion(t *testing.T) {
	// Skip if iperf3 not installed
	if err := iperf.CheckInstalled(); err != nil {
		t.Skip("iperf3 not installed, skipping version test")
	}

	version, err := iperf.GetVersion()
	if err != nil {
		t.Skipf("could not get iperf3 version: %v", err)
	}
	if version == "" {
		t.Error("expected non-empty version string")
	}
}

func TestIperfJSONFields(t *testing.T) {
	// Test the IperfJSON structure
	output := iperf.IperfJSON{}
	if len(output.Start.Connected) != 0 {
		t.Error("expected empty Connected slice")
	}
	if output.End.SumSent.BitsPerSecond != 0 {
		t.Error("expected zero BitsPerSecond")
	}
}

func TestFindIperf3Binary(_ *testing.T) {
	// Reset the cached path to test finding
	originalPath := iperf.IperfBinaryPath()
	iperf.SetIperfBinaryPath("")
	defer func() { iperf.SetIperfBinaryPath(originalPath) }()

	path, err := iperf.FindIperf3Binary()
	// Just verify it doesn't panic - result depends on system
	_ = path
	_ = err
}

func TestFindIperf3BinaryCached(t *testing.T) {
	// Test that cached path is returned
	originalPath := iperf.IperfBinaryPath()
	iperf.SetIperfBinaryPath("/cached/path/iperf3")
	defer func() { iperf.SetIperfBinaryPath(originalPath) }()

	path, err := iperf.FindIperf3Binary()
	if err != nil {
		t.Errorf("unexpected error with cached path: %v", err)
	}
	if path != "/cached/path/iperf3" {
		t.Errorf("expected cached path, got %q", path)
	}
}

func TestClientConfigProtocols(t *testing.T) {
	tests := []struct {
		protocol string
		reverse  bool
		expected string
	}{
		{"tcp", false, "upload"},
		{"tcp", true, "download"},
		{"udp", false, "upload"},
		{"udp", true, "download"},
	}

	for _, tt := range tests {
		config := iperf.ClientConfig{
			Server:   "localhost",
			Protocol: tt.protocol,
			Reverse:  tt.reverse,
		}

		// Verify the config is set correctly
		if config.Server != "localhost" {
			t.Errorf("expected server 'localhost', got %q", config.Server)
		}
		if config.Protocol != tt.protocol {
			t.Errorf("expected protocol %q, got %q", tt.protocol, config.Protocol)
		}
		if config.Reverse != tt.reverse {
			t.Errorf("expected reverse %v, got %v", tt.reverse, config.Reverse)
		}
	}
}

func TestResultDirections(t *testing.T) {
	uploadResult := iperf.Result{Direction: "upload"}
	downloadResult := iperf.Result{Direction: "download"}

	if uploadResult.Direction != "upload" {
		t.Errorf("expected direction 'upload', got %q", uploadResult.Direction)
	}
	if downloadResult.Direction != "download" {
		t.Errorf("expected direction 'download', got %q", downloadResult.Direction)
	}
}

func TestResultUDPFields(t *testing.T) {
	result := iperf.Result{
		Protocol:    "udp",
		Jitter:      2.5,
		LostPackets: 10,
		LostPercent: 0.1,
	}

	if result.Protocol != "udp" {
		t.Errorf("expected protocol 'udp', got %q", result.Protocol)
	}
	if result.Jitter != 2.5 {
		t.Errorf("expected Jitter 2.5, got %v", result.Jitter)
	}
	if result.LostPackets != 10 {
		t.Errorf("expected LostPackets 10, got %d", result.LostPackets)
	}
	if result.LostPercent != 0.1 {
		t.Errorf("expected LostPercent 0.1, got %v", result.LostPercent)
	}
}

func TestServerStatusError(t *testing.T) {
	status := iperf.ServerStatus{
		Running: false,
		Port:    0,
		PID:     0,
		Error:   "server crashed",
	}

	if status.Running {
		t.Error("expected Running false")
	}
	if status.Port != 0 {
		t.Errorf("expected Port 0, got %d", status.Port)
	}
	if status.PID != 0 {
		t.Errorf("expected PID 0, got %d", status.PID)
	}
	if status.Error != "server crashed" {
		t.Errorf("expected Error 'server crashed', got %q", status.Error)
	}
}

func TestClientStatusPhases(t *testing.T) {
	phases := []string{"idle", "connecting", "testing", "complete"}

	for _, phase := range phases {
		status := iperf.ClientStatus{Phase: phase}
		if status.Phase != phase {
			t.Errorf("expected phase %q, got %q", phase, status.Phase)
		}
	}
}

func TestConcurrentManagerAccess(_ *testing.T) {
	manager := iperf.NewManager()

	done := make(chan bool)
	for range 10 {
		go func() {
			for range 50 {
				_ = manager.GetServerStatus()
				_ = manager.GetClientStatus()
				_ = manager.GetLastResult()
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}

func TestManagerStoreResult(t *testing.T) {
	manager := iperf.NewManager()

	// Initially nil
	if manager.GetLastResult() != nil {
		t.Error("expected nil result initially")
	}

	// Manually set a result using exported helper
	manager.SetManagerLastResult(&iperf.Result{
		Bandwidth: 100.0,
		Direction: "download",
	})

	result := manager.GetLastResult()
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Bandwidth != 100.0 {
		t.Errorf("expected Bandwidth 100.0, got %v", result.Bandwidth)
	}
}

func TestManagerStartServerAlreadyRunning(t *testing.T) {
	manager := iperf.NewManager()

	// Manually set server as running using exported helper
	manager.SetManagerServerStatusRunning(true, 5201)

	err := manager.StartServer(5202)
	if err == nil {
		t.Error("expected error when server already running")
	}

	// Clean up
	manager.SetManagerServerStatusRunning(false, 0)
}

func TestResultAllFields(t *testing.T) {
	now := time.Now()
	result := iperf.Result{
		BitsPerSecond: 100_000_000,
		Bandwidth:     100.0,
		Transfer:      125.0,
		Retransmits:   5,
		Jitter:        1.5,
		LostPackets:   2,
		LostPercent:   0.5,
		Protocol:      "tcp",
		Direction:     "download",
		Duration:      10.0,
		Server:        "speedtest.example.com",
		Port:          5201,
		Timestamp:     now,
	}

	if result.BitsPerSecond != 100_000_000 {
		t.Errorf("expected BitsPerSecond 100_000_000, got %v", result.BitsPerSecond)
	}
	if result.Bandwidth != 100.0 {
		t.Errorf("expected Bandwidth 100.0, got %v", result.Bandwidth)
	}
	if result.Transfer != 125.0 {
		t.Errorf("expected Transfer 125.0, got %v", result.Transfer)
	}
	if result.Retransmits != 5 {
		t.Errorf("expected Retransmits 5, got %d", result.Retransmits)
	}
	if result.Jitter != 1.5 {
		t.Errorf("expected Jitter 1.5, got %v", result.Jitter)
	}
	if result.LostPackets != 2 {
		t.Errorf("expected LostPackets 2, got %d", result.LostPackets)
	}
	if result.LostPercent != 0.5 {
		t.Errorf("expected LostPercent 0.5, got %v", result.LostPercent)
	}
	if result.Protocol != "tcp" {
		t.Errorf("expected Protocol 'tcp', got %q", result.Protocol)
	}
	if result.Direction != "download" {
		t.Errorf("expected Direction 'download', got %q", result.Direction)
	}
	if result.Duration != 10.0 {
		t.Errorf("expected Duration 10.0, got %v", result.Duration)
	}
	if result.Server != "speedtest.example.com" {
		t.Errorf("expected Server 'speedtest.example.com', got %q", result.Server)
	}
	if result.Port != 5201 {
		t.Errorf("expected Port 5201, got %d", result.Port)
	}
	if result.Timestamp != now {
		t.Errorf("expected Timestamp %v, got %v", now, result.Timestamp)
	}
}

func TestServerStatusFields(t *testing.T) {
	status := iperf.ServerStatus{
		Running: true,
		Port:    5201,
		PID:     12345,
		Error:   "",
	}

	if !status.Running {
		t.Error("expected Running true")
	}
	if status.Port != 5201 {
		t.Errorf("expected Port 5201, got %d", status.Port)
	}
	if status.PID != 12345 {
		t.Errorf("expected PID 12345, got %d", status.PID)
	}
	if status.Error != "" {
		t.Errorf("expected empty Error, got %q", status.Error)
	}
}

func TestClientStatusFields(t *testing.T) {
	status := iperf.ClientStatus{
		Running:  true,
		Phase:    "testing",
		Progress: 50.0,
	}

	if !status.Running {
		t.Error("expected Running true")
	}
	if status.Phase != "testing" {
		t.Errorf("expected Phase 'testing', got %q", status.Phase)
	}
	if status.Progress != 50.0 {
		t.Errorf("expected Progress 50.0, got %v", status.Progress)
	}
}

func TestIperfJSONStructure(t *testing.T) {
	// Test the IperfJSON structure fields
	output := iperf.IperfJSON{}
	if len(output.Start.Connected) != 0 {
		t.Error("expected empty Connected slice")
	}
	if output.End.SumSent.BitsPerSecond != 0 {
		t.Error("expected zero BitsPerSecond")
	}
	if output.End.Sum.JitterMs != 0 {
		t.Error("expected zero JitterMs")
	}
}

func TestManagerStatusMethods(t *testing.T) {
	manager := iperf.NewManager()

	// Test initial server status
	serverStatus := manager.GetServerStatus()
	if serverStatus.Running {
		t.Error("expected server not running initially")
	}
	if serverStatus.Port != 0 {
		t.Errorf("expected port 0, got %d", serverStatus.Port)
	}

	// Test initial client status
	clientStatus := manager.GetClientStatus()
	if clientStatus.Running {
		t.Error("expected client not running initially")
	}
	if clientStatus.Phase != "idle" {
		t.Errorf("expected phase 'idle', got %q", clientStatus.Phase)
	}
}

func TestManagerConcurrentStatusAccess(_ *testing.T) {
	manager := iperf.NewManager()

	done := make(chan bool)
	for range 10 {
		go func() {
			for range 50 {
				_ = manager.GetServerStatus()
				_ = manager.GetClientStatus()
				_ = manager.GetLastResult()
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}

func TestClientConfigProtocol(t *testing.T) {
	tests := []struct {
		protocol string
		reverse  bool
		expected string
	}{
		{"tcp", false, "upload"},
		{"tcp", true, "download"},
		{"udp", false, "upload"},
		{"udp", true, "download"},
	}

	for _, tt := range tests {
		config := iperf.ClientConfig{
			Server:   "localhost",
			Protocol: tt.protocol,
			Reverse:  tt.reverse,
		}

		if config.Server != "localhost" {
			t.Errorf("expected server 'localhost', got %q", config.Server)
		}
		if config.Protocol != tt.protocol {
			t.Errorf("expected protocol %q, got %q", tt.protocol, config.Protocol)
		}
		if config.Reverse != tt.reverse {
			t.Errorf("expected reverse %v, got %v", tt.reverse, config.Reverse)
		}
	}
}

func TestServerStatusWithError(t *testing.T) {
	status := iperf.ServerStatus{
		Running: false,
		Port:    0,
		PID:     0,
		Error:   "server crashed",
	}

	if status.Running {
		t.Error("expected Running false")
	}
	if status.Port != 0 {
		t.Errorf("expected Port 0, got %d", status.Port)
	}
	if status.PID != 0 {
		t.Errorf("expected PID 0, got %d", status.PID)
	}
	if status.Error != "server crashed" {
		t.Errorf("expected Error 'server crashed', got %q", status.Error)
	}
}

func TestManagerSetResult(t *testing.T) {
	manager := iperf.NewManager()

	// Initially nil
	if manager.GetLastResult() != nil {
		t.Error("expected nil result initially")
	}

	// Set result manually using exported helper
	manager.SetManagerLastResult(&iperf.Result{
		Bandwidth: 500.0,
		Direction: "upload",
		Protocol:  "tcp",
	})

	result := manager.GetLastResult()
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Bandwidth != 500.0 {
		t.Errorf("expected Bandwidth 500.0, got %v", result.Bandwidth)
	}
	if result.Direction != "upload" {
		t.Errorf("expected Direction 'upload', got %q", result.Direction)
	}
}

// TestIperf3BinaryRequired validates that iperf3 binary can be found (build-time check).
func TestIperf3BinaryRequired(t *testing.T) {
	// Check if we're in CI or test environment where iperf3 may not be available
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf3 binary test (SKIP_IPERF_TEST=1)")
	}

	path, err := iperf.FindIperf3Binary()
	if err != nil {
		t.Fatalf(
			"iperf3 binary not found: %v\n\nTo fix this:\n1. Run scripts/build-iperf3.sh to build bundled binary\n2. Or install system-wide: apt-get install iperf3 (Ubuntu) or brew install iperf3 (macOS)\n3. Or set SKIP_IPERF_TEST=1 to skip this test in CI",
			err,
		)
	}

	t.Logf("Found iperf3 at: %s", path)

	// Verify it's executable
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat iperf3 binary: %v", err)
	}

	if info.Mode()&0o111 == 0 {
		t.Fatalf("iperf3 binary is not executable: %s", path)
	}
}

// TestIperf3VersionRequired validates the iperf3 version meets requirements (build-time check).
func TestIperf3VersionRequired(t *testing.T) {
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf3 version test (SKIP_IPERF_TEST=1)")
	}

	version, err := iperf.GetVersion()
	if err != nil {
		t.Fatalf("Failed to get iperf3 version: %v", err)
	}

	t.Logf("iperf3 version: %s", version)

	// Validate minimum version
	err = iperf.ValidateVersion()
	if err != nil {
		t.Fatalf(
			"iperf3 version validation failed: %v\n\nRequired version: %s or higher\nFound version: %s",
			err,
			iperf.MinSupportedVersion,
			version,
		)
	}
}

// TestVersionComparison tests the version comparison logic.
func TestVersionComparison(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"3.17", "3.17", 0},
		{"3.20", "3.17", 1},
		{"3.17", "3.20", -1},
		{"3.17.1", "3.17", 1},
		{"3.17", "3.17.1", -1},
		{"4.0", "3.20", 1},
		{"2.5", "3.0", -1},
		{"3.9", "3.10", -1}, // Test string vs numeric comparison
		{"3.10", "3.9", 1},
	}

	for _, tt := range tests {
		result := iperf.CompareVersions(tt.v1, tt.v2)
		if result != tt.expected {
			t.Errorf("CompareVersions(%q, %q) = %d; expected %d", tt.v1, tt.v2, result, tt.expected)
		}
	}
}

// TestValidateServer tests the server address validation.
func TestValidateServer(t *testing.T) {
	tests := []struct {
		name    string
		server  string
		wantErr bool
		errMsg  string
	}{
		// Valid IP addresses
		{"valid IPv4", "192.168.1.100", false, ""},
		{"valid IPv4 localhost", "127.0.0.1", false, ""},
		{"valid IPv4 zeros", "0.0.0.0", false, ""},
		{"valid IPv6", "::1", false, ""},
		{"valid IPv6 full", "2001:db8::1", false, ""},

		// Valid hostnames
		{"valid hostname simple", "localhost", false, ""},
		{"valid hostname domain", "example.com", false, ""},
		{"valid hostname subdomain", "test.example.com", false, ""},
		{"valid hostname with hyphen", "my-server.example.com", false, ""},
		{"valid hostname with numbers", "server1.example.com", false, ""},

		// Invalid inputs
		{"empty string", "", true, "server address is required"},
		{"hostname too long", string(make([]byte, 300)), true, "server hostname too long"},
		{"invalid hostname special chars", "test@server.com", true, "invalid server address"},
		{"invalid hostname space", "test server.com", true, "invalid server address"},
		{"invalid hostname underscore", "test_server.com", true, "invalid server address"},
		{"invalid hostname colon", "test:server", true, "invalid server address"},
		{"invalid hostname semicolon", "test;server", true, "invalid server address"},
		{
			"invalid command injection attempt",
			"localhost; rm -rf /",
			true,
			"invalid server address",
		},
		{"invalid pipe injection", "localhost | cat /etc/passwd", true, "invalid server address"},
		{"invalid backtick injection", "`whoami`", true, "invalid server address"},
		{"invalid shell variable", "$HOME", true, "invalid server address"},
		{"starts with hyphen", "-invalid.com", true, "invalid server address"},
		{"ends with hyphen", "invalid-.com", true, "invalid server address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := iperf.ValidateServer(tt.server)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateServer(%q) expected error, got nil", tt.server)
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf(
						"ValidateServer(%q) error = %q, want containing %q",
						tt.server,
						err.Error(),
						tt.errMsg,
					)
				}
			} else if err != nil {
				t.Errorf("ValidateServer(%q) unexpected error: %v", tt.server, err)
			}
		})
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSetClientDefaults tests the client configuration defaults.
func TestSetClientDefaults(t *testing.T) {
	tests := []struct {
		name          string
		input         iperf.ClientConfig
		expectedPort  int
		expectedDur   int
		expectedPar   int
		expectedProto string
	}{
		{
			name:          "all defaults",
			input:         iperf.ClientConfig{Server: "localhost"},
			expectedPort:  5201,
			expectedDur:   10,
			expectedPar:   1,
			expectedProto: "tcp",
		},
		{
			name:          "custom port preserved",
			input:         iperf.ClientConfig{Server: "localhost", Port: 9999},
			expectedPort:  9999,
			expectedDur:   10,
			expectedPar:   1,
			expectedProto: "tcp",
		},
		{
			name:          "custom duration preserved",
			input:         iperf.ClientConfig{Server: "localhost", Duration: 30},
			expectedPort:  5201,
			expectedDur:   30,
			expectedPar:   1,
			expectedProto: "tcp",
		},
		{
			name:          "custom parallel preserved",
			input:         iperf.ClientConfig{Server: "localhost", Parallel: 4},
			expectedPort:  5201,
			expectedDur:   10,
			expectedPar:   4,
			expectedProto: "tcp",
		},
		{
			name:          "custom protocol preserved",
			input:         iperf.ClientConfig{Server: "localhost", Protocol: "udp"},
			expectedPort:  5201,
			expectedDur:   10,
			expectedPar:   1,
			expectedProto: "udp",
		},
		{
			name: "all custom values preserved",
			input: iperf.ClientConfig{
				Server: "localhost", Port: 5555, Duration: 60, Parallel: 8, Protocol: "udp",
			},
			expectedPort:  5555,
			expectedDur:   60,
			expectedPar:   8,
			expectedProto: "udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.input
			iperf.SetClientDefaults(&config)

			if config.Port != tt.expectedPort {
				t.Errorf("Port = %d, want %d", config.Port, tt.expectedPort)
			}
			if config.Duration != tt.expectedDur {
				t.Errorf("Duration = %d, want %d", config.Duration, tt.expectedDur)
			}
			if config.Parallel != tt.expectedPar {
				t.Errorf("Parallel = %d, want %d", config.Parallel, tt.expectedPar)
			}
			if config.Protocol != tt.expectedProto {
				t.Errorf("Protocol = %q, want %q", config.Protocol, tt.expectedProto)
			}
		})
	}
}

// TestNormalizeDirection tests direction normalization logic.
func TestNormalizeDirection(t *testing.T) {
	tests := []struct {
		name              string
		inputDirection    string
		inputReverse      bool
		expectedDirection string
	}{
		// Empty direction infers from Reverse flag
		{"empty direction, reverse=false", "", false, "upload"},
		{"empty direction, reverse=true", "", true, "download"},

		// Explicit directions
		{"explicit upload", "upload", false, "upload"},
		{"explicit download", "download", false, "download"},
		{"explicit bidirectional", "bidirectional", false, "bidirectional"},

		// Case insensitivity
		{"uppercase UPLOAD", "UPLOAD", false, "upload"},
		{"uppercase DOWNLOAD", "DOWNLOAD", false, "download"},
		{"mixed case BiDiReCtIoNaL", "BiDiReCtIoNaL", false, "bidirectional"},

		// Invalid direction defaults to upload
		{"invalid direction", "invalid", false, "upload"},
		{"random string direction", "xyz", false, "upload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &iperf.ClientConfig{
				Server:    "localhost",
				Direction: tt.inputDirection,
				Reverse:   tt.inputReverse,
			}

			result := iperf.NormalizeDirection(config)

			if result != tt.expectedDirection {
				t.Errorf("NormalizeDirection() = %q, want %q", result, tt.expectedDirection)
			}
			if config.Direction != tt.expectedDirection {
				t.Errorf("config.Direction = %q, want %q", config.Direction, tt.expectedDirection)
			}
		})
	}
}

// TestBuildClientArgs tests command line argument building.
func TestBuildClientArgs(t *testing.T) {
	tests := []struct {
		name          string
		config        iperf.ClientConfig
		direction     string
		expectedArgs  []string
		shouldContain []string
		shouldNotHave []string
	}{
		{
			name: "basic TCP upload",
			config: iperf.ClientConfig{
				Server:   "192.168.1.1",
				Port:     5201,
				Duration: 10,
				Parallel: 1,
				Protocol: "tcp",
			},
			direction:     "upload",
			shouldContain: []string{"-c", "192.168.1.1", "-p", "5201", "-t", "10", "-P", "1", "-J"},
			shouldNotHave: []string{"-R", "--bidir", "-u"},
		},
		{
			name: "TCP download (reverse)",
			config: iperf.ClientConfig{
				Server:   "192.168.1.1",
				Port:     5201,
				Duration: 10,
				Parallel: 1,
				Protocol: "tcp",
			},
			direction:     "download",
			shouldContain: []string{"-c", "192.168.1.1", "-R", "-J"},
			shouldNotHave: []string{"--bidir", "-u"},
		},
		{
			name: "bidirectional test",
			config: iperf.ClientConfig{
				Server:   "192.168.1.1",
				Port:     5201,
				Duration: 10,
				Parallel: 1,
				Protocol: "tcp",
			},
			direction:     "bidirectional",
			shouldContain: []string{"-c", "192.168.1.1", "--bidir", "-J"},
			shouldNotHave: []string{"-R", "-u"},
		},
		{
			name: "UDP test",
			config: iperf.ClientConfig{
				Server:   "192.168.1.1",
				Port:     5201,
				Duration: 10,
				Parallel: 1,
				Protocol: "udp",
			},
			direction:     "upload",
			shouldContain: []string{"-c", "192.168.1.1", "-u", "-b", "0", "-J"},
			shouldNotHave: []string{"-R", "--bidir"},
		},
		{
			name: "parallel streams",
			config: iperf.ClientConfig{
				Server:   "localhost",
				Port:     5201,
				Duration: 10,
				Parallel: 4,
				Protocol: "tcp",
			},
			direction:     "upload",
			shouldContain: []string{"-P", "4"},
		},
		{
			name: "custom port",
			config: iperf.ClientConfig{
				Server:   "localhost",
				Port:     9999,
				Duration: 10,
				Parallel: 1,
				Protocol: "tcp",
			},
			direction:     "upload",
			shouldContain: []string{"-p", "9999"},
		},
		{
			name: "custom duration",
			config: iperf.ClientConfig{
				Server:   "localhost",
				Port:     5201,
				Duration: 60,
				Parallel: 1,
				Protocol: "tcp",
			},
			direction:     "upload",
			shouldContain: []string{"-t", "60"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := iperf.BuildClientArgs(&tt.config, tt.direction)

			// Check that required args are present
			for _, expected := range tt.shouldContain {
				if !slices.Contains(args, expected) {
					t.Errorf("Expected arg %q not found in %v", expected, args)
				}
			}

			// Check that excluded args are not present
			for _, excluded := range tt.shouldNotHave {
				if slices.Contains(args, excluded) {
					t.Errorf("Unexpected arg %q found in %v", excluded, args)
				}
			}
		})
	}
}

// parseResultTestCase defines a test case for ParseClientResult.
type parseResultTestCase struct {
	name              string
	iperfOut          iperf.IperfJSON
	config            iperf.ClientConfig
	direction         string
	expectedBandwidth float64
	expectedDirection string
	expectedProtocol  string
}

// buildTCPUploadJSON creates iperf JSON for TCP upload test.
func buildTCPUploadJSON() iperf.IperfJSON {
	j := iperf.IperfJSON{}
	j.End.SumSent.BitsPerSecond = 100_000_000
	j.End.SumSent.Bytes = 125_000_000
	j.End.SumSent.Seconds = 10
	j.End.SumSent.Retransmits = 5
	return j
}

// buildTCPDownloadJSON creates iperf JSON for TCP download test.
func buildTCPDownloadJSON() iperf.IperfJSON {
	j := iperf.IperfJSON{}
	j.End.SumReceived.BitsPerSecond = 200_000_000
	j.End.SumReceived.Bytes = 250_000_000
	j.End.SumReceived.Seconds = 10
	return j
}

// buildBidirectionalJSON creates iperf JSON for bidirectional test.
func buildBidirectionalJSON() iperf.IperfJSON {
	j := iperf.IperfJSON{}
	j.End.SumReceived.BitsPerSecond = 150_000_000
	j.End.SumReceived.Bytes = 187_500_000
	j.End.SumSent.BitsPerSecond = 100_000_000
	j.End.SumSent.Bytes = 125_000_000
	j.End.SumSent.Retransmits = 2
	j.End.Sum.Seconds = 10
	return j
}

// buildUDPJSON creates iperf JSON for UDP test with jitter and loss.
func buildUDPJSON() iperf.IperfJSON {
	j := iperf.IperfJSON{}
	j.End.SumSent.BitsPerSecond = 50_000_000
	j.End.SumSent.Bytes = 62_500_000
	j.End.SumSent.Seconds = 10
	j.End.Sum.JitterMs = 1.5
	j.End.Sum.LostPackets = 10
	j.End.Sum.LostPercent = 0.5
	return j
}

// validateParseResultBasic validates basic result fields.
func validateParseResultBasic(t *testing.T, result *iperf.Result, tc *parseResultTestCase) {
	t.Helper()
	if result.Bandwidth != tc.expectedBandwidth {
		t.Errorf("Bandwidth = %v, want %v", result.Bandwidth, tc.expectedBandwidth)
	}
	if result.Direction != tc.expectedDirection {
		t.Errorf("Direction = %q, want %q", result.Direction, tc.expectedDirection)
	}
	if result.Protocol != tc.expectedProtocol {
		t.Errorf("Protocol = %q, want %q", result.Protocol, tc.expectedProtocol)
	}
	if result.Server != tc.config.Server {
		t.Errorf("Server = %q, want %q", result.Server, tc.config.Server)
	}
	if result.Port != tc.config.Port {
		t.Errorf("Port = %d, want %d", result.Port, tc.config.Port)
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

// validateUDPFields validates UDP-specific result fields.
func validateUDPFields(t *testing.T, result *iperf.Result, iperfOut *iperf.IperfJSON) {
	t.Helper()
	if result.Jitter != iperfOut.End.Sum.JitterMs {
		t.Errorf("Jitter = %v, want %v", result.Jitter, iperfOut.End.Sum.JitterMs)
	}
	if result.LostPackets != iperfOut.End.Sum.LostPackets {
		t.Errorf("LostPackets = %d, want %d", result.LostPackets, iperfOut.End.Sum.LostPackets)
	}
	if result.LostPercent != iperfOut.End.Sum.LostPercent {
		t.Errorf("LostPercent = %v, want %v", result.LostPercent, iperfOut.End.Sum.LostPercent)
	}
}

// validateBidirectionalFields validates bidirectional result fields.
func validateBidirectionalFields(t *testing.T, result *iperf.Result) {
	t.Helper()
	if result.UploadBandwidth == 0 {
		t.Error("UploadBandwidth should not be zero for bidirectional")
	}
	if result.DownloadBandwidth == 0 {
		t.Error("DownloadBandwidth should not be zero for bidirectional")
	}
}

// TestParseClientResult tests result parsing from iperf3 JSON output.
func TestParseClientResult(t *testing.T) {
	tests := []parseResultTestCase{
		{
			name:              "TCP upload result",
			iperfOut:          buildTCPUploadJSON(),
			config:            iperf.ClientConfig{Server: "localhost", Port: 5201, Protocol: "tcp"},
			direction:         "upload",
			expectedBandwidth: 100.0,
			expectedDirection: "upload",
			expectedProtocol:  "tcp",
		},
		{
			name:              "TCP download result",
			iperfOut:          buildTCPDownloadJSON(),
			config:            iperf.ClientConfig{Server: "localhost", Port: 5201, Protocol: "tcp"},
			direction:         "download",
			expectedBandwidth: 200.0,
			expectedDirection: "download",
			expectedProtocol:  "tcp",
		},
		{
			name:              "bidirectional result",
			iperfOut:          buildBidirectionalJSON(),
			config:            iperf.ClientConfig{Server: "localhost", Port: 5201, Protocol: "tcp"},
			direction:         "bidirectional",
			expectedBandwidth: 150.0,
			expectedDirection: "bidirectional",
			expectedProtocol:  "tcp",
		},
		{
			name:              "UDP result with jitter and loss",
			iperfOut:          buildUDPJSON(),
			config:            iperf.ClientConfig{Server: "localhost", Port: 5201, Protocol: "udp"},
			direction:         "upload",
			expectedBandwidth: 50.0,
			expectedDirection: "upload",
			expectedProtocol:  "udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := iperf.ParseClientResult(&tt.iperfOut, &tt.config, tt.direction)
			if result == nil {
				t.Fatal("ParseClientResult returned nil")
			}
			validateParseResultBasic(t, result, &tt)
			if tt.config.Protocol == "udp" {
				validateUDPFields(t, result, &tt.iperfOut)
			}
			if tt.direction == "bidirectional" {
				validateBidirectionalFields(t, result)
			}
		})
	}
}

// TestParseClientResultRetransmits tests TCP retransmit parsing.
func TestParseClientResultRetransmits(t *testing.T) {
	iperfOut := iperf.IperfJSON{}
	iperfOut.End.SumSent.BitsPerSecond = 100_000_000
	iperfOut.End.SumSent.Bytes = 125_000_000
	iperfOut.End.SumSent.Seconds = 10
	iperfOut.End.SumSent.Retransmits = 42

	config := &iperf.ClientConfig{
		Server:   "localhost",
		Port:     5201,
		Protocol: "tcp",
	}

	result := iperf.ParseClientResult(&iperfOut, config, "upload")

	if result.Retransmits != 42 {
		t.Errorf("Retransmits = %d, want 42", result.Retransmits)
	}
}

// TestGetLegacyPaths tests the legacy path lookup.
func TestGetLegacyPaths(t *testing.T) {
	paths := iperf.GetLegacyPaths()

	// Should return at least some paths
	if len(paths) == 0 {
		t.Error("GetLegacyPaths() returned empty slice")
	}

	// All paths should be absolute
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			t.Errorf("Path %q is not absolute", path)
		}
	}

	// Should contain expected path patterns
	foundBinIperf := false
	for _, path := range paths {
		if filepath.Base(path) == "iperf3" {
			foundBinIperf = true
			break
		}
	}
	if !foundBinIperf {
		t.Error("Expected path ending in 'iperf3' not found")
	}
}

// TestValidateBinary tests binary validation.
func TestValidateBinary(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"non-existent path", "/non/existent/path/iperf3", false},
		{"empty path", "", false},
		{"directory path", "/tmp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := iperf.ValidateBinary(tt.path)
			if result != tt.expected {
				t.Errorf("ValidateBinary(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestNeedsSudo tests the sudo requirement detection.
func TestNeedsSudo(t *testing.T) {
	tests := []struct {
		name           string
		packageManager string
		expected       bool
	}{
		{"homebrew", "homebrew", false},
		{"scoop", "scoop", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := iperf.NeedsSudo(tt.packageManager)
			if result != tt.expected {
				t.Errorf("NeedsSudo(%q) = %v, want %v", tt.packageManager, result, tt.expected)
			}
		})
	}
}

// TestGetCacheDir tests cache directory retrieval.
func TestGetCacheDir(t *testing.T) {
	dir, err := iperf.GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}

	if dir == "" {
		t.Error("GetCacheDir() returned empty string")
	}

	// Should be an absolute path
	if !filepath.IsAbs(dir) {
		t.Errorf("GetCacheDir() returned non-absolute path: %q", dir)
	}

	// Should contain "seed" in the path
	if !findSubstring(dir, "seed") {
		t.Errorf("GetCacheDir() path should contain 'seed': %q", dir)
	}
}

// binaryTestCase defines a test case for binary validation.
type binaryTestCase struct {
	name        string
	binaryPerm  os.FileMode // 0 means don't create binary
	versionPerm os.FileMode // 0 means don't create version file
	version     string
	expected    bool
}

// getBinaryTestCases returns test cases for IsValidExtractedBinary.
func getBinaryTestCases() []binaryTestCase {
	return []binaryTestCase{
		{name: "non-existent binary", expected: false},
		{name: "missing version file", binaryPerm: 0o755, expected: false},
		{
			name:        "wrong version",
			binaryPerm:  0o755,
			versionPerm: 0o600,
			version:     "1.0.0",
			expected:    false,
		},
		{
			name:        "non-executable binary",
			binaryPerm:  0o600,
			versionPerm: 0o600,
			version:     iperf.EmbeddedVersion,
			expected:    false,
		},
		{
			name:        "valid binary and version",
			binaryPerm:  0o755,
			versionPerm: 0o600,
			version:     iperf.EmbeddedVersion,
			expected:    true,
		},
	}
}

// TestIsValidExtractedBinary tests the extracted binary validation.
func TestIsValidExtractedBinary(t *testing.T) {
	for i, tc := range getBinaryTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			binaryPath := filepath.Join(tempDir, "binary")
			versionFile := filepath.Join(tempDir, "version")

			if tc.binaryPerm != 0 {
				if err := os.WriteFile(binaryPath, []byte("test"), tc.binaryPerm); err != nil {
					t.Fatalf("Failed to create test binary: %v", err)
				}
			}
			if tc.versionPerm != 0 {
				if err := os.WriteFile(versionFile, []byte(tc.version), tc.versionPerm); err != nil {
					t.Fatalf("Failed to create version file: %v", err)
				}
			}

			result := iperf.IsValidExtractedBinary(binaryPath, versionFile)
			if result != tc.expected {
				t.Errorf(
					"test case %d: IsValidExtractedBinary() = %v, want %v",
					i,
					result,
					tc.expected,
				)
			}
		})
	}
}

// TestGetPlatformBinaryMap tests the platform binary mapping.
func TestGetPlatformBinaryMap(t *testing.T) {
	platformMap := iperf.GetPlatformBinaryMap()

	// Should have entries for major platforms
	expectedPlatforms := []string{
		"linux-amd64",
		"linux-arm64",
		"darwin-amd64",
		"darwin-arm64",
	}

	for _, platform := range expectedPlatforms {
		if _, ok := platformMap[platform]; !ok {
			t.Errorf("Platform %q not found in platform binary map", platform)
		}
	}

	// Each entry should have a valid binary name
	for platform, binaryName := range platformMap {
		if binaryName == "" {
			t.Errorf("Platform %q has empty binary name", platform)
		}
		if !findSubstring(binaryName, "iperf3") {
			t.Errorf("Binary name %q for platform %q should contain 'iperf3'", binaryName, platform)
		}
	}
}

// TestNotFoundError tests the NotFoundError type.
func TestNotFoundError(t *testing.T) {
	tests := []struct {
		name          string
		err           *iperf.NotFoundError
		shouldContain []string
	}{
		{
			name: "basic error",
			err: &iperf.NotFoundError{
				SearchedPaths: []string{"/usr/bin/iperf3", "/usr/local/bin/iperf3"},
				SystemError:   nil,
				EmbeddedError: nil,
			},
			shouldContain: []string{"iperf3 not found", "/usr/bin/iperf3", "/usr/local/bin/iperf3"},
		},
		{
			name: "with system error",
			err: &iperf.NotFoundError{
				SearchedPaths: []string{"/usr/bin/iperf3"},
				SystemError:   os.ErrNotExist,
				EmbeddedError: nil,
			},
			shouldContain: []string{"iperf3 not found", "System PATH"},
		},
		{
			name: "with embedded error",
			err: &iperf.NotFoundError{
				SearchedPaths: []string{},
				SystemError:   nil,
				EmbeddedError: os.ErrNotExist,
			},
			shouldContain: []string{"iperf3 not found", "Embedded binary"},
		},
		{
			name: "with both errors",
			err: &iperf.NotFoundError{
				SearchedPaths: []string{"/usr/bin/iperf3"},
				SystemError:   os.ErrNotExist,
				EmbeddedError: os.ErrNotExist,
			},
			shouldContain: []string{"iperf3 not found", "System PATH", "Embedded binary"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			for _, substr := range tt.shouldContain {
				if !findSubstring(errMsg, substr) {
					t.Errorf("Error message should contain %q, got: %s", substr, errMsg)
				}
			}
		})
	}
}

// TestHasEmbeddedBinary tests the embedded binary detection.
func TestHasEmbeddedBinary(t *testing.T) {
	// This just tests the function doesn't panic
	// The actual result depends on whether binaries are embedded
	result := iperf.HasEmbeddedBinary()
	t.Logf("HasEmbeddedBinary() = %v", result)
}

// TestGetInstallInstructions tests install instructions generation.
func TestGetInstallInstructions(t *testing.T) {
	instructions := iperf.GetInstallInstructions()

	if instructions == "" {
		t.Error("GetInstallInstructions() returned empty string")
	}

	// Should always contain some install guidance
	if !findSubstring(instructions, "iperf3") {
		t.Error("Instructions should mention iperf3")
	}

	// Should contain source build instructions
	if !findSubstring(instructions, "github.com/esnet/iperf") {
		t.Error("Instructions should include GitHub source")
	}
}

// TestDetectPackageManager tests package manager detection.
func TestDetectPackageManager(t *testing.T) {
	pm := iperf.DetectPackageManager()

	// This test is system-dependent
	// Just verify the function doesn't panic and returns consistent type
	if pm != nil {
		if pm.Name == "" {
			t.Error("Detected package manager should have a name")
		}
		if len(pm.InstallCommand) == 0 {
			t.Error("Detected package manager should have install command")
		}
		t.Logf("Detected package manager: %s", pm.Name)
	} else {
		t.Log("No package manager detected (may be expected on some systems)")
	}
}

// TestInstallOptions tests install options struct.
func TestInstallOptions(t *testing.T) {
	opts := iperf.InstallOptions{
		Method:     iperf.InstallMethodPackageManager,
		Version:    "3.17",
		InstallDir: "/usr/local",
		UseSudo:    true,
		Verbose:    true,
	}

	if opts.Method != iperf.InstallMethodPackageManager {
		t.Errorf("Method = %q, want %q", opts.Method, iperf.InstallMethodPackageManager)
	}
	if opts.Version != "3.17" {
		t.Errorf("Version = %q, want %q", opts.Version, "3.17")
	}
	if opts.InstallDir != "/usr/local" {
		t.Errorf("InstallDir = %q, want %q", opts.InstallDir, "/usr/local")
	}
	if !opts.UseSudo {
		t.Error("UseSudo should be true")
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
}

// TestInstallResult tests install result struct.
func TestInstallResult(t *testing.T) {
	result := iperf.InstallResult{
		Success: true,
		Path:    "/usr/local/bin/iperf3",
		Version: "3.17",
		Method:  iperf.InstallMethodGitHub,
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Path != "/usr/local/bin/iperf3" {
		t.Errorf("Path = %q, want %q", result.Path, "/usr/local/bin/iperf3")
	}
	if result.Version != "3.17" {
		t.Errorf("Version = %q, want %q", result.Version, "3.17")
	}
	if result.Method != iperf.InstallMethodGitHub {
		t.Errorf("Method = %q, want %q", result.Method, iperf.InstallMethodGitHub)
	}
	if result.Error != nil {
		t.Errorf("Error should be nil, got %v", result.Error)
	}
	if result.NeedsSudo {
		t.Error("NeedsSudo should be false")
	}
	if result.SudoCommand != "" {
		t.Errorf("SudoCommand should be empty, got %q", result.SudoCommand)
	}
}

// TestInstallMethodConstants tests install method constant values.
func TestInstallMethodConstants(t *testing.T) {
	tests := []struct {
		method   iperf.InstallMethod
		expected string
	}{
		{iperf.InstallMethodPackageManager, "package_manager"},
		{iperf.InstallMethodGitHub, "github"},
		{iperf.InstallMethodManual, "manual"},
	}

	for _, tt := range tests {
		if string(tt.method) != tt.expected {
			t.Errorf("InstallMethod constant = %q, want %q", tt.method, tt.expected)
		}
	}
}

// TestPackageManagerInfo tests package manager info struct.
func TestPackageManagerInfo(t *testing.T) {
	info := iperf.PackageManagerInfo{
		Name:           "apt",
		InstallCommand: []string{"apt", "install", "-y", "iperf3"},
		UpdateCommand:  []string{"apt", "update"},
		Available:      true,
	}

	if info.Name != "apt" {
		t.Errorf("Name = %q, want %q", info.Name, "apt")
	}
	if len(info.InstallCommand) != 4 {
		t.Errorf("InstallCommand length = %d, want 4", len(info.InstallCommand))
	}
	if len(info.UpdateCommand) != 2 {
		t.Errorf("UpdateCommand length = %d, want 2", len(info.UpdateCommand))
	}
	if !info.Available {
		t.Error("Available should be true")
	}
}

// TestCheckBuildDependencies tests build dependency checking.
func TestCheckBuildDependencies(t *testing.T) {
	missing := iperf.CheckBuildDependencies()

	// This is system-dependent - just verify it returns a slice
	t.Logf("Missing build dependencies: %v", missing)
}

// TestGetBuildDependencyInstallCommand tests build dependency install command.
func TestGetBuildDependencyInstallCommand(t *testing.T) {
	cmd := iperf.GetBuildDependencyInstallCommand()

	if cmd == "" {
		t.Error("GetBuildDependencyInstallCommand() returned empty string")
	}

	// Should mention some common build tools
	if !findSubstring(cmd, "make") && !findSubstring(cmd, "gcc") && !findSubstring(cmd, "Install") {
		t.Errorf("Command should mention build tools: %q", cmd)
	}
}

// TestManagerRunClientEmptyServer tests running client with empty server.
func TestManagerRunClientEmptyServer(t *testing.T) {
	manager := iperf.NewManager()
	ctx := context.Background()

	_, err := manager.RunClient(ctx, &iperf.ClientConfig{Server: ""})
	if err == nil {
		t.Error("Expected error for empty server")
	}
	if !findSubstring(err.Error(), "server address is required") {
		t.Errorf("Error should mention server address, got: %v", err)
	}
}

// TestManagerRunClientInvalidServer tests running client with invalid server.
func TestManagerRunClientInvalidServer(t *testing.T) {
	manager := iperf.NewManager()
	ctx := context.Background()

	_, err := manager.RunClient(ctx, &iperf.ClientConfig{Server: "invalid;server"})
	if err == nil {
		t.Error("Expected error for invalid server")
	}
	if !findSubstring(err.Error(), "invalid server address") {
		t.Errorf("Error should mention invalid server address, got: %v", err)
	}
}

// TestResultBidirectionalFields tests bidirectional result fields.
func TestResultBidirectionalFields(t *testing.T) {
	result := iperf.Result{
		Direction:             "bidirectional",
		UploadBitsPerSecond:   100_000_000,
		DownloadBitsPerSecond: 200_000_000,
		UploadBandwidth:       100.0,
		DownloadBandwidth:     200.0,
		UploadTransfer:        125.0,
		DownloadTransfer:      250.0,
	}

	if result.Direction != "bidirectional" {
		t.Errorf("Direction = %q, want bidirectional", result.Direction)
	}
	if result.UploadBitsPerSecond != 100_000_000 {
		t.Errorf("UploadBitsPerSecond = %v, want 100000000", result.UploadBitsPerSecond)
	}
	if result.DownloadBitsPerSecond != 200_000_000 {
		t.Errorf("DownloadBitsPerSecond = %v, want 200000000", result.DownloadBitsPerSecond)
	}
	if result.UploadBandwidth != 100.0 {
		t.Errorf("UploadBandwidth = %v, want 100.0", result.UploadBandwidth)
	}
	if result.DownloadBandwidth != 200.0 {
		t.Errorf("DownloadBandwidth = %v, want 200.0", result.DownloadBandwidth)
	}
	if result.UploadTransfer != 125.0 {
		t.Errorf("UploadTransfer = %v, want 125.0", result.UploadTransfer)
	}
	if result.DownloadTransfer != 250.0 {
		t.Errorf("DownloadTransfer = %v, want 250.0", result.DownloadTransfer)
	}
}

// TestManagerStartServerInvalidPort tests starting server with invalid port.
func TestManagerStartServerInvalidPort(t *testing.T) {
	manager := iperf.NewManager()

	// Test with invalid port numbers
	invalidPorts := []int{-1, 0, 65536, 100000}

	for _, port := range invalidPorts {
		err := manager.StartServer(port)
		if err == nil {
			t.Errorf("Expected error for invalid port %d", port)
		}
	}
}

// TestEmbeddedVersion tests the embedded version constant.
func TestEmbeddedVersion(t *testing.T) {
	if iperf.EmbeddedVersion == "" {
		t.Error("EmbeddedVersion should not be empty")
	}

	// Should be a valid version format
	parts := splitString(iperf.EmbeddedVersion, ".")
	if len(parts) < 2 {
		t.Errorf(
			"EmbeddedVersion should have at least major.minor format: %q",
			iperf.EmbeddedVersion,
		)
	}
}

// splitString splits a string by a separator.
func splitString(s, sep string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
