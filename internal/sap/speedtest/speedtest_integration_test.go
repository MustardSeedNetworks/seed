//go:build integration

package speedtest_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// TestRunTestIntegration tests the full RunTest flow with actual network calls.
// This test requires network access and connects to speedtest.net servers.
func TestRunTestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := tester.RunTest(ctx)
	if err != nil {
		t.Fatalf("RunTest failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Validate result fields
	if result.Download <= 0 {
		t.Errorf("expected positive download speed, got %v", result.Download)
	}
	if result.Upload <= 0 {
		t.Errorf("expected positive upload speed, got %v", result.Upload)
	}
	if result.Latency <= 0 {
		t.Errorf("expected positive latency, got %v", result.Latency)
	}
	if result.Server == "" {
		t.Error("expected non-empty server name")
	}
	if result.Location == "" {
		t.Error("expected non-empty location")
	}
	if result.Host == "" {
		t.Error("expected non-empty host")
	}
	if result.TestDuration <= 0 {
		t.Errorf("expected positive test duration, got %v", result.TestDuration)
	}
	if result.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	// Validate last result is stored
	lastResult := tester.GetLastResult()
	if lastResult == nil {
		t.Error("expected last result to be stored")
	}
	if lastResult != result {
		t.Error("last result should match returned result")
	}

	t.Logf("Download: %.2f Mbps, Upload: %.2f Mbps, Latency: %.2f ms",
		result.Download, result.Upload, result.Latency)
	t.Logf("Server: %s at %s", result.Server, result.Location)
	t.Logf("Test duration: %.2f seconds", result.TestDuration)
}

// TestFindTestServerIntegration tests server discovery with actual network calls.
func TestFindTestServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	server, err := tester.FindTestServer()
	if err != nil {
		t.Fatalf("FindTestServer failed: %v", err)
	}

	if server == nil {
		t.Fatal("expected non-nil server")
	}

	if server.Name == "" {
		t.Error("expected non-empty server name")
	}
	if server.Host == "" {
		t.Error("expected non-empty server host")
	}

	t.Logf("Found server: %s (%s) at %.2f km", server.Name, server.Host, server.Distance)
}

// TestBuildTestResultWithRealServer tests buildTestResult with a real server object.
func TestBuildTestResultWithRealServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	server, err := tester.FindTestServer()
	if err != nil {
		t.Fatalf("FindTestServer failed: %v", err)
	}

	startTime := time.Now().Add(-15 * time.Second) // Simulate 15 second test
	result := tester.BuildTestResult(server, startTime)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Server != server.Name {
		t.Errorf("Server: got %q, want %q", result.Server, server.Name)
	}
	if result.Host != server.Host {
		t.Errorf("Host: got %q, want %q", result.Host, server.Host)
	}
	if result.Distance != server.Distance {
		t.Errorf("Distance: got %v, want %v", result.Distance, server.Distance)
	}

	// TestDuration should be approximately 15 seconds
	if result.TestDuration < 14.9 || result.TestDuration > 15.5 {
		t.Errorf("TestDuration: got %v, want ~15 seconds", result.TestDuration)
	}

	t.Logf("Result: Server=%s, Location=%s, Host=%s, Distance=%.2f km",
		result.Server, result.Location, result.Host, result.Distance)
}

// TestStatusProgressionIntegration tests that status progresses correctly during a real test.
func TestStatusProgressionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	// Track observed phases
	observedPhases := make(map[string]bool)

	// Start a goroutine to poll status
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				status := tester.GetStatus()
				observedPhases[status.Phase] = true
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	_, err := tester.RunTest(ctx)
	close(done)

	if err != nil {
		t.Fatalf("RunTest failed: %v", err)
	}

	// Verify key phases were observed
	expectedPhases := []string{"finding_server", "testing_latency", "testing_download", "testing_upload"}
	for _, phase := range expectedPhases {
		if !observedPhases[phase] {
			t.Errorf("phase %q was not observed during test", phase)
		}
	}

	t.Logf("Observed phases: %v", observedPhases)
}
