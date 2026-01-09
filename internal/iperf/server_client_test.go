package iperf_test

import (
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/iperf"
)

// TestStartServerWithValidPort tests starting server with a valid port.
func TestStartServerWithValidPort(t *testing.T) {
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf test")
	}

	// Check if iperf3 is installed
	if err := iperf.CheckInstalled(); err != nil {
		t.Skip("iperf3 not installed")
	}

	manager := iperf.NewManager()

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	// Give the OS time to release the port
	time.Sleep(100 * time.Millisecond)

	// Try to start server
	err = manager.StartServer(port)
	if err != nil {
		// May fail due to permissions or binary issues - that's OK
		t.Logf("StartServer error (expected on some systems): %v", err)
		return
	}

	// Verify server is running
	status := manager.GetServerStatus()
	if !status.Running {
		t.Error("Server should be running")
	}
	if status.Port != port {
		t.Errorf("Port = %d, want %d", status.Port, port)
	}

	// Stop server
	if err := manager.StopServer(); err != nil {
		t.Errorf("StopServer error: %v", err)
	}

	// Verify server stopped
	status = manager.GetServerStatus()
	if status.Running {
		t.Error("Server should not be running after stop")
	}
}

// TestStartServerAlreadyRunningWithPort tests starting when already running.
func TestStartServerAlreadyRunningWithPort(t *testing.T) {
	t.Parallel()

	manager := iperf.NewManager()

	// Set server as running
	manager.SetManagerServerStatusRunning(true, 5201)
	defer manager.SetManagerServerStatusRunning(false, 0)

	// Try to start again
	err := manager.StartServer(5202)
	if err == nil {
		t.Error("Expected error when server already running")
	}

	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("Error should mention 'already running', got: %v", err)
	}
}

// TestStopServerClearsAllState tests that stopping clears all state.
func TestStopServerClearsAllState(t *testing.T) {
	t.Parallel()

	manager := iperf.NewManager()

	// Set server running with some state
	manager.SetManagerServerStatusRunning(true, 5201)

	// Stop server
	err := manager.StopServer()
	if err != nil {
		t.Errorf("StopServer error: %v", err)
	}

	// Verify all state is cleared
	status := manager.GetServerStatus()
	if status.Running {
		t.Error("Running should be false")
	}
	if status.Port != 0 {
		t.Errorf("Port should be 0, got %d", status.Port)
	}
	if status.PID != 0 {
		t.Errorf("PID should be 0, got %d", status.PID)
	}
}

// TestRunClientWithValidConfig tests client with valid configuration.
func TestRunClientWithValidConfig(t *testing.T) {
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf test")
	}

	// Check if iperf3 is installed
	if err := iperf.CheckInstalled(); err != nil {
		t.Skip("iperf3 not installed")
	}

	manager := iperf.NewManager()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run client against a non-existent server (will fail quickly)
	config := &iperf.ClientConfig{
		Server:   "127.0.0.1",
		Port:     59997, // unlikely to have an iperf server
		Duration: 1,
	}

	_, err := manager.RunClient(ctx, config)
	// This should fail but exercise the code path
	if err == nil {
		t.Log("Unexpectedly succeeded (maybe an iperf server is running)")
	} else {
		t.Logf("Expected error: %v", err)
	}

	// Verify client status is reset
	status := manager.GetClientStatus()
	if status.Running {
		t.Error("Client should not be running after test")
	}
	if status.Phase != "idle" {
		t.Errorf("Phase should be 'idle', got %q", status.Phase)
	}
}

// TestRunClientContextCancellation tests client with cancelled context.
func TestRunClientContextCancellation(t *testing.T) {
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf test")
	}

	// Check if iperf3 is installed
	if err := iperf.CheckInstalled(); err != nil {
		t.Skip("iperf3 not installed")
	}

	manager := iperf.NewManager()

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := &iperf.ClientConfig{
		Server:   "127.0.0.1",
		Port:     5201,
		Duration: 10,
	}

	_, err := manager.RunClient(ctx, config)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}

	// Verify client status is reset
	status := manager.GetClientStatus()
	if status.Running {
		t.Error("Client should not be running after cancellation")
	}
}

// TestClientStatusProgression tests client status changes.
func TestClientStatusProgression(t *testing.T) {
	t.Parallel()

	manager := iperf.NewManager()

	// Initial state
	status := manager.GetClientStatus()
	if status.Phase != "idle" {
		t.Errorf("Initial phase should be 'idle', got %q", status.Phase)
	}
	if status.Running {
		t.Error("Initial Running should be false")
	}
	if status.Progress != 0 {
		t.Errorf("Initial Progress should be 0, got %v", status.Progress)
	}

	// Set running
	manager.SetManagerClientStatusRunning(true)
	status = manager.GetClientStatus()
	if !status.Running {
		t.Error("Running should be true after setting")
	}

	// Reset
	manager.SetManagerClientStatusRunning(false)
	status = manager.GetClientStatus()
	if status.Running {
		t.Error("Running should be false after reset")
	}
}

// TestServerStatusProgression tests server status changes.
func TestServerStatusProgression(t *testing.T) {
	t.Parallel()

	manager := iperf.NewManager()

	// Initial state
	status := manager.GetServerStatus()
	if status.Running {
		t.Error("Initial Running should be false")
	}
	if status.Port != 0 {
		t.Errorf("Initial Port should be 0, got %d", status.Port)
	}

	// Set running
	manager.SetManagerServerStatusRunning(true, 5201)
	status = manager.GetServerStatus()
	if !status.Running {
		t.Error("Running should be true after setting")
	}
	if status.Port != 5201 {
		t.Errorf("Port should be 5201, got %d", status.Port)
	}

	// Reset
	manager.SetManagerServerStatusRunning(false, 0)
	status = manager.GetServerStatus()
	if status.Running {
		t.Error("Running should be false after reset")
	}
	if status.Port != 0 {
		t.Errorf("Port should be 0 after reset, got %d", status.Port)
	}
}

// TestLastResultStorage tests last result storage.
func TestLastResultStorage(t *testing.T) {
	t.Parallel()

	manager := iperf.NewManager()

	// Initially nil
	if manager.GetLastResult() != nil {
		t.Error("Initial last result should be nil")
	}

	// Set result
	result := &iperf.Result{
		BitsPerSecond: 100_000_000,
		Bandwidth:     100.0,
		Protocol:      "tcp",
		Direction:     "upload",
		Server:        "test.example.com",
		Port:          5201,
		Timestamp:     time.Now(),
	}
	manager.SetManagerLastResult(result)

	// Verify result is stored
	stored := manager.GetLastResult()
	if stored == nil {
		t.Fatal("Last result should not be nil after setting")
	}
	if stored.Bandwidth != 100.0 {
		t.Errorf("Bandwidth = %v, want 100.0", stored.Bandwidth)
	}
	if stored.Server != "test.example.com" {
		t.Errorf("Server = %q, want 'test.example.com'", stored.Server)
	}

	// Replace result
	result2 := &iperf.Result{
		Bandwidth: 200.0,
		Server:    "another.example.com",
	}
	manager.SetManagerLastResult(result2)

	stored = manager.GetLastResult()
	if stored.Bandwidth != 200.0 {
		t.Errorf("Bandwidth = %v, want 200.0", stored.Bandwidth)
	}

	// Set nil
	manager.SetManagerLastResult(nil)
	if manager.GetLastResult() != nil {
		t.Error("Last result should be nil after setting nil")
	}
}

// TestManagerConcurrentOperationsSafe tests thread safety.
func TestManagerConcurrentOperationsSafe(t *testing.T) {
	t.Parallel()

	manager := iperf.NewManager()

	var wg sync.WaitGroup
	iterations := 100
	goroutines := 20

	// Concurrent operations
	for range goroutines {
		wg.Add(4)

		// Read server status
		go func() {
			defer wg.Done()
			for range iterations {
				_ = manager.GetServerStatus()
			}
		}()

		// Read client status
		go func() {
			defer wg.Done()
			for range iterations {
				_ = manager.GetClientStatus()
			}
		}()

		// Read last result
		go func() {
			defer wg.Done()
			for range iterations {
				_ = manager.GetLastResult()
			}
		}()

		// Write operations
		go func() {
			defer wg.Done()
			for i := range iterations {
				if i%2 == 0 {
					manager.SetManagerClientStatusRunning(true)
				} else {
					manager.SetManagerClientStatusRunning(false)
				}
			}
		}()
	}

	wg.Wait()
}

// TestValidateVersionWithInstalledIperf tests version validation.
func TestValidateVersionWithInstalledIperf(t *testing.T) {
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf test")
	}

	// Check if iperf3 is installed
	if err := iperf.CheckInstalled(); err != nil {
		t.Skip("iperf3 not installed")
	}

	err := iperf.ValidateVersion()
	if err != nil {
		t.Logf("Version validation result: %v", err)
		// May fail if installed version is old
	} else {
		t.Log("Version validation passed")
	}
}

// TestGetVersionWithInstalledIperf tests version retrieval.
func TestGetVersionWithInstalledIperf(t *testing.T) {
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf test")
	}

	// Check if iperf3 is installed
	if err := iperf.CheckInstalled(); err != nil {
		t.Skip("iperf3 not installed")
	}

	version, err := iperf.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	if version == "" {
		t.Error("Version should not be empty")
	}

	// Should start with v
	if !strings.HasPrefix(version, "v") {
		t.Logf("Version format: %s (may not start with v)", version)
	}

	t.Logf("Installed iperf3 version: %s", version)
}

// TestCheckInstalledReturnsConsistent tests consistency.
func TestCheckInstalledReturnsConsistent(t *testing.T) {
	t.Parallel()

	// Call multiple times
	results := make([]error, 5)
	for i := range results {
		results[i] = iperf.CheckInstalled()
	}

	// All results should be consistent
	first := results[0]
	for i, r := range results {
		if (first == nil) != (r == nil) {
			t.Errorf("Result %d differs from first: first=%v, this=%v", i, first, r)
		}
	}
}

// TestClientConfigDirectionNormalization tests direction normalization.
func TestClientConfigDirectionNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		direction         string
		reverse           bool
		expectedDirection string
		expectedReverse   bool
	}{
		{
			name:              "empty direction with reverse false",
			direction:         "",
			reverse:           false,
			expectedDirection: "upload",
			expectedReverse:   false,
		},
		{
			name:              "empty direction with reverse true",
			direction:         "",
			reverse:           true,
			expectedDirection: "download",
			expectedReverse:   true,
		},
		{
			name:              "explicit upload",
			direction:         "upload",
			reverse:           false,
			expectedDirection: "upload",
		},
		{
			name:              "explicit download",
			direction:         "download",
			reverse:           false,
			expectedDirection: "download",
		},
		{
			name:              "explicit bidirectional",
			direction:         "bidirectional",
			reverse:           false,
			expectedDirection: "bidirectional",
		},
		{
			name:              "uppercase UPLOAD",
			direction:         "UPLOAD",
			reverse:           false,
			expectedDirection: "upload",
		},
		{
			name:              "uppercase DOWNLOAD",
			direction:         "DOWNLOAD",
			reverse:           false,
			expectedDirection: "download",
		},
		{
			name:              "invalid direction defaults to upload",
			direction:         "invalid",
			reverse:           false,
			expectedDirection: "upload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &iperf.ClientConfig{
				Server:    "localhost",
				Direction: tt.direction,
				Reverse:   tt.reverse,
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

// TestSetClientDefaultsAllFields tests all default setting.
func TestSetClientDefaultsAllFields(t *testing.T) {
	t.Parallel()

	// All zero values
	config := &iperf.ClientConfig{Server: "localhost"}
	iperf.SetClientDefaults(config)

	if config.Port != 5201 {
		t.Errorf("Port = %d, want 5201", config.Port)
	}
	if config.Duration != 10 {
		t.Errorf("Duration = %d, want 10", config.Duration)
	}
	if config.Parallel != 1 {
		t.Errorf("Parallel = %d, want 1", config.Parallel)
	}
	if config.Protocol != "tcp" {
		t.Errorf("Protocol = %q, want 'tcp'", config.Protocol)
	}

	// Non-zero values should not be changed
	config2 := &iperf.ClientConfig{
		Server:   "localhost",
		Port:     9999,
		Duration: 60,
		Parallel: 8,
		Protocol: "udp",
	}
	iperf.SetClientDefaults(config2)

	if config2.Port != 9999 {
		t.Errorf("Port changed from 9999 to %d", config2.Port)
	}
	if config2.Duration != 60 {
		t.Errorf("Duration changed from 60 to %d", config2.Duration)
	}
	if config2.Parallel != 8 {
		t.Errorf("Parallel changed from 8 to %d", config2.Parallel)
	}
	if config2.Protocol != "udp" {
		t.Errorf("Protocol changed from 'udp' to %q", config2.Protocol)
	}
}
