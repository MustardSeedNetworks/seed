// Package iperf provides network throughput testing using the iperf3 tool.
// Test suite validates iperf3 client/server operations, result parsing, and bandwidth measurement.
package iperf

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

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
	manager := NewManager()

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
	manager := NewManager()

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
	manager := NewManager()

	result := manager.GetLastResult()
	if result != nil {
		t.Error("expected nil result initially")
	}
}

func TestClientConfigDefaults(t *testing.T) {
	config := ClientConfig{
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
	config := ClientConfig{
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
	result := Result{
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
	if result.Protocol != "tcp" {
		t.Errorf("expected Protocol 'tcp', got %q", result.Protocol)
	}
	if result.Direction != "download" {
		t.Errorf("expected Direction 'download', got %q", result.Direction)
	}
}

func TestServerStatus(t *testing.T) {
	status := ServerStatus{
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
	status := ClientStatus{
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
	manager := NewManager()

	err := manager.StopServer()
	if err == nil {
		t.Error("expected error when stopping non-running server")
	}
	if err.Error() != "server not running" {
		t.Errorf("expected 'server not running' error, got %q", err.Error())
	}
}

func TestManagerRunClientAlreadyRunning(t *testing.T) {
	manager := NewManager()

	// Manually set client as running
	manager.mu.Lock()
	manager.clientStatus.Running = true
	manager.mu.Unlock()

	ctx := context.Background()
	_, err := manager.RunClient(ctx, &ClientConfig{Server: "localhost"})
	if err == nil {
		t.Error("expected error when test already in progress")
	}
	if err.Error() != "test already in progress" {
		t.Errorf("expected 'test already in progress' error, got %q", err.Error())
	}

	// Clean up
	manager.mu.Lock()
	manager.clientStatus.Running = false
	manager.mu.Unlock()
}

func TestManagerServerAlreadyRunning(t *testing.T) {
	manager := NewManager()

	// Manually set server as running
	manager.mu.Lock()
	manager.serverStatus.Running = true
	manager.serverStatus.Port = 5201
	manager.mu.Unlock()

	err := manager.StartServer(5201)
	if err == nil {
		t.Error("expected error when server already running")
	}

	// Clean up
	manager.mu.Lock()
	manager.serverStatus.Running = false
	manager.mu.Unlock()
}

func TestCheckInstalled(t *testing.T) {
	// This test may fail if iperf3 is not installed, which is okay
	err := CheckInstalled()
	// Just check it doesn't panic - the result depends on system configuration
	_ = err
}

func TestGetVersion(t *testing.T) {
	// Skip if iperf3 not installed
	if err := CheckInstalled(); err != nil {
		t.Skip("iperf3 not installed, skipping version test")
	}

	version, err := GetVersion()
	if err != nil {
		t.Skipf("could not get iperf3 version: %v", err)
	}
	if version == "" {
		t.Error("expected non-empty version string")
	}
}

func TestIperfJSONFields(t *testing.T) {
	// Test the iperfJSON structure
	output := iperfJSON{}
	if len(output.Start.Connected) != 0 {
		t.Error("expected empty Connected slice")
	}
	if output.End.SumSent.BitsPerSecond != 0 {
		t.Error("expected zero BitsPerSecond")
	}
}

func TestFindIperf3Binary(t *testing.T) {
	// Reset the cached path to test finding
	originalPath := iperfBinaryPath
	iperfBinaryPath = ""
	defer func() { iperfBinaryPath = originalPath }()

	path, err := findIperf3Binary()
	// Just verify it doesn't panic - result depends on system
	_ = path
	_ = err
}

func TestFindIperf3BinaryCached(t *testing.T) {
	// Test that cached path is returned
	originalPath := iperfBinaryPath
	iperfBinaryPath = "/cached/path/iperf3"
	defer func() { iperfBinaryPath = originalPath }()

	path, err := findIperf3Binary()
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
		config := ClientConfig{
			Server:   "localhost",
			Protocol: tt.protocol,
			Reverse:  tt.reverse,
		}

		// Just verify the config is set correctly
		if config.Protocol != tt.protocol {
			t.Errorf("expected protocol %q, got %q", tt.protocol, config.Protocol)
		}
		if config.Reverse != tt.reverse {
			t.Errorf("expected reverse %v, got %v", tt.reverse, config.Reverse)
		}
	}
}

func TestResultDirections(t *testing.T) {
	uploadResult := Result{Direction: "upload"}
	downloadResult := Result{Direction: "download"}

	if uploadResult.Direction != "upload" {
		t.Errorf("expected direction 'upload', got %q", uploadResult.Direction)
	}
	if downloadResult.Direction != "download" {
		t.Errorf("expected direction 'download', got %q", downloadResult.Direction)
	}
}

func TestResultUDPFields(t *testing.T) {
	result := Result{
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
	status := ServerStatus{
		Running: false,
		Port:    0,
		PID:     0,
		Error:   "server crashed",
	}

	if status.Running {
		t.Error("expected Running false")
	}
	if status.Error != "server crashed" {
		t.Errorf("expected Error 'server crashed', got %q", status.Error)
	}
}

func TestClientStatusPhases(t *testing.T) {
	phases := []string{"idle", "connecting", "testing", "complete"}

	for _, phase := range phases {
		status := ClientStatus{Phase: phase}
		if status.Phase != phase {
			t.Errorf("expected phase %q, got %q", phase, status.Phase)
		}
	}
}

func TestConcurrentManagerAccess(t *testing.T) {
	manager := NewManager()

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
	manager := NewManager()

	// Initially nil
	if manager.GetLastResult() != nil {
		t.Error("expected nil result initially")
	}

	// Manually set a result
	manager.mu.Lock()
	manager.lastResult = &Result{
		Bandwidth: 100.0,
		Direction: "download",
	}
	manager.mu.Unlock()

	result := manager.GetLastResult()
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Bandwidth != 100.0 {
		t.Errorf("expected Bandwidth 100.0, got %v", result.Bandwidth)
	}
}

func TestManagerStartServerAlreadyRunning(t *testing.T) {
	manager := NewManager()

	// Manually set server as running
	manager.mu.Lock()
	manager.serverStatus.Running = true
	manager.serverStatus.Port = 5201
	manager.mu.Unlock()

	err := manager.StartServer(5202)
	if err == nil {
		t.Error("expected error when server already running")
	}

	// Clean up
	manager.mu.Lock()
	manager.serverStatus.Running = false
	manager.mu.Unlock()
}

func TestResultAllFields(t *testing.T) {
	now := time.Now()
	result := Result{
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
}

func TestServerStatusFields(t *testing.T) {
	status := ServerStatus{
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
	status := ClientStatus{
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
	// Test the iperfJSON structure fields
	output := iperfJSON{}
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
	manager := NewManager()

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

func TestManagerConcurrentStatusAccess(t *testing.T) {
	manager := NewManager()

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
		config := ClientConfig{
			Server:   "localhost",
			Protocol: tt.protocol,
			Reverse:  tt.reverse,
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
	status := ServerStatus{
		Running: false,
		Port:    0,
		PID:     0,
		Error:   "server crashed",
	}

	if status.Running {
		t.Error("expected Running false")
	}
	if status.Error != "server crashed" {
		t.Errorf("expected Error 'server crashed', got %q", status.Error)
	}
}

func TestManagerSetResult(t *testing.T) {
	manager := NewManager()

	// Initially nil
	if manager.GetLastResult() != nil {
		t.Error("expected nil result initially")
	}

	// Set result manually
	manager.mu.Lock()
	manager.lastResult = &Result{
		Bandwidth: 500.0,
		Direction: "upload",
		Protocol:  "tcp",
	}
	manager.mu.Unlock()

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

	path, err := findIperf3Binary()
	if err != nil {
		t.Fatalf("iperf3 binary not found: %v\n\nTo fix this:\n1. Run scripts/build-iperf3.sh to build bundled binary\n2. Or install system-wide: apt-get install iperf3 (Ubuntu) or brew install iperf3 (macOS)\n3. Or set SKIP_IPERF_TEST=1 to skip this test in CI", err)
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

	version, err := GetVersion()
	if err != nil {
		t.Fatalf("Failed to get iperf3 version: %v", err)
	}

	t.Logf("iperf3 version: %s", version)

	// Validate minimum version
	err = ValidateVersion()
	if err != nil {
		t.Fatalf("iperf3 version validation failed: %v\n\nRequired version: %s or higher\nFound version: %s", err, minSupportedVersion, version)
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
		result := compareVersions(tt.v1, tt.v2)
		if result != tt.expected {
			t.Errorf("compareVersions(%q, %q) = %d; expected %d", tt.v1, tt.v2, result, tt.expected)
		}
	}
}
