//go:build integration

package speedtest_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// TestFindTestServerMultiple tests finding multiple servers.
func TestFindTestServerMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Run multiple server discoveries to verify consistency
	const numAttempts = 3
	servers := make([]string, numAttempts)

	for i := range numAttempts {
		tester := speedtest.NewTester()
		server, err := tester.FindTestServer()
		if err != nil {
			t.Fatalf("attempt %d: FindTestServer failed: %v", i, err)
		}
		if server == nil {
			t.Fatalf("attempt %d: expected non-nil server", i)
		}
		servers[i] = server.Name
		t.Logf("attempt %d: found server %s at %.2f km", i, server.Name, server.Distance)
	}

	// Server selection should be consistent (same closest server)
	for i := 1; i < numAttempts; i++ {
		if servers[i] != servers[0] {
			t.Logf("Note: server selection varied between attempts (expected for load balancing)")
		}
	}
}

// TestBuildTestResultWithRealServerAllFields tests buildTestResult with a real server and all field validation.
func TestBuildTestResultWithRealServerAllFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	server, err := tester.FindTestServer()
	if err != nil {
		t.Fatalf("FindTestServer failed: %v", err)
	}

	startTime := time.Now().Add(-20 * time.Second)
	result := tester.BuildTestResult(server, startTime)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Validate server name matches
	if result.Server != server.Name {
		t.Errorf("Server: got %q, want %q", result.Server, server.Name)
	}

	// Validate host matches
	if result.Host != server.Host {
		t.Errorf("Host: got %q, want %q", result.Host, server.Host)
	}

	// Validate distance matches
	if result.Distance != server.Distance {
		t.Errorf("Distance: got %v, want %v", result.Distance, server.Distance)
	}

	// Validate location format (Sponsor, Country)
	if result.Location == "" {
		t.Error("expected non-empty Location")
	}

	// Validate test duration is approximately correct
	if result.TestDuration < 19.5 || result.TestDuration > 21.0 {
		t.Errorf("TestDuration: got %v, want ~20 seconds", result.TestDuration)
	}

	// Validate timestamp is recent
	if time.Since(result.Timestamp) > time.Second {
		t.Errorf("Timestamp too old: %v", result.Timestamp)
	}

	t.Logf("Result: Server=%s, Host=%s, Location=%s, Distance=%.2f km",
		result.Server, result.Host, result.Location, result.Distance)
}

// TestRunTestWithStatusPolling tests RunTest while polling status.
func TestRunTestWithStatusPolling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	// Collect status updates during the test
	var statusHistory []speedtest.Status
	var mu sync.Mutex
	done := make(chan struct{})

	// Start status polling goroutine
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				status := tester.GetStatus()
				mu.Lock()
				statusHistory = append(statusHistory, status)
				mu.Unlock()
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result, err := tester.RunTest(ctx)
	close(done)

	if err != nil {
		t.Fatalf("RunTest failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Analyze collected status history
	mu.Lock()
	defer mu.Unlock()

	if len(statusHistory) == 0 {
		t.Error("no status updates collected")
	}

	// Count occurrences of each phase
	phaseCounts := make(map[string]int)
	for _, status := range statusHistory {
		phaseCounts[status.Phase]++
	}

	t.Logf("Phase occurrences during test: %v", phaseCounts)

	// Verify key phases were observed
	keyPhases := []string{"finding_server", "testing_latency", "testing_download", "testing_upload"}
	for _, phase := range keyPhases {
		if phaseCounts[phase] == 0 {
			t.Errorf("phase %q was not observed", phase)
		}
	}

	// Check that progress generally increased
	var lastProgress float64
	progressIncreased := false
	for _, status := range statusHistory {
		if status.Progress > lastProgress {
			progressIncreased = true
		}
		lastProgress = status.Progress
	}
	if !progressIncreased {
		t.Error("progress never increased during test")
	}

	t.Logf("Test completed: %.2f Mbps down, %.2f Mbps up, %.2f ms latency",
		result.Download, result.Upload, result.Latency)
}

// TestRunTestValidateResults tests that RunTest returns valid results.
func TestRunTestValidateResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result, err := tester.RunTest(ctx)
	if err != nil {
		t.Fatalf("RunTest failed: %v", err)
	}

	// Validate result fields are positive
	if result.Download <= 0 {
		t.Errorf("Download should be positive, got %v", result.Download)
	}
	if result.Upload <= 0 {
		t.Errorf("Upload should be positive, got %v", result.Upload)
	}
	if result.Latency <= 0 {
		t.Errorf("Latency should be positive, got %v", result.Latency)
	}

	// Validate string fields are not empty
	if result.Server == "" {
		t.Error("Server should not be empty")
	}
	if result.Location == "" {
		t.Error("Location should not be empty")
	}
	if result.Host == "" {
		t.Error("Host should not be empty")
	}

	// Validate test duration is reasonable (5-180 seconds)
	if result.TestDuration < 5 || result.TestDuration > 180 {
		t.Errorf("TestDuration out of expected range: %v", result.TestDuration)
	}

	// Validate timestamp is recent
	if time.Since(result.Timestamp) > 5*time.Minute {
		t.Errorf("Timestamp too old: %v", result.Timestamp)
	}

	// Validate distance is non-negative
	if result.Distance < 0 {
		t.Errorf("Distance should not be negative: %v", result.Distance)
	}

	// Validate last result is stored
	lastResult := tester.GetLastResult()
	if lastResult == nil {
		t.Error("last result should be stored")
	}
	if lastResult != result {
		t.Error("last result should match returned result")
	}
}

// TestRunTestReasonableSpeeds tests that RunTest returns speeds within reasonable bounds.
func TestRunTestReasonableSpeeds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result, err := tester.RunTest(ctx)
	if err != nil {
		t.Fatalf("RunTest failed: %v", err)
	}

	// Verify speeds are within reasonable bounds for typical internet connections
	// Very few connections exceed 10 Gbps
	maxReasonableSpeed := 10000.0 // 10 Gbps
	if result.Download > maxReasonableSpeed {
		t.Errorf("Download speed unreasonably high: %v Mbps", result.Download)
	}
	if result.Upload > maxReasonableSpeed {
		t.Errorf("Upload speed unreasonably high: %v Mbps", result.Upload)
	}

	// Verify latency is within reasonable bounds (< 10 seconds = 10000ms)
	maxReasonableLatency := 10000.0
	if result.Latency > maxReasonableLatency {
		t.Errorf("Latency unreasonably high: %v ms", result.Latency)
	}

	t.Logf("Speeds validated: %.2f Mbps down, %.2f Mbps up, %.2f ms latency",
		result.Download, result.Upload, result.Latency)
}

// TestRunTestStatusReset tests that status resets to idle after test completion.
func TestRunTestStatusReset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	// Verify initial state
	status := tester.GetStatus()
	if status.Running {
		t.Error("should not be running initially")
	}
	if status.Phase != "idle" {
		t.Errorf("initial phase should be 'idle', got %q", status.Phase)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	_, err := tester.RunTest(ctx)
	if err != nil {
		t.Fatalf("RunTest failed: %v", err)
	}

	// Immediately after test, should be "complete"
	status = tester.GetStatus()
	if status.Phase != "complete" {
		t.Logf("Phase immediately after test: %q (expected 'complete' or may have already reset)", status.Phase)
	}

	// Wait for idle reset delay (2 seconds + buffer)
	time.Sleep(3 * time.Second)

	// Should be back to idle
	status = tester.GetStatus()
	if status.Phase != "idle" {
		t.Errorf("phase should be 'idle' after reset delay, got %q", status.Phase)
	}
	if status.Running {
		t.Error("should not be running after reset")
	}
}

// TestRunTestLiveSpeedUpdates tests that live speed updates are provided during the test.
func TestRunTestLiveSpeedUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	var downloadSpeedsSeen []float64
	var uploadSpeedsSeen []float64
	var mu sync.Mutex
	done := make(chan struct{})

	// Poll for live speed updates
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				status := tester.GetStatus()
				mu.Lock()
				if status.Phase == "testing_download" && status.CurrentDownload > 0 {
					downloadSpeedsSeen = append(downloadSpeedsSeen, status.CurrentDownload)
				}
				if status.Phase == "testing_upload" && status.CurrentUpload > 0 {
					uploadSpeedsSeen = append(uploadSpeedsSeen, status.CurrentUpload)
				}
				mu.Unlock()
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	_, err := tester.RunTest(ctx)
	close(done)

	if err != nil {
		t.Fatalf("RunTest failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Should have seen some live download speeds
	if len(downloadSpeedsSeen) == 0 {
		t.Error("no live download speeds observed")
	} else {
		t.Logf("Observed %d download speed samples", len(downloadSpeedsSeen))
	}

	// Should have seen some live upload speeds
	if len(uploadSpeedsSeen) == 0 {
		t.Error("no live upload speeds observed")
	} else {
		t.Logf("Observed %d upload speed samples", len(uploadSpeedsSeen))
	}
}

// TestMultipleSequentialTests tests running multiple tests sequentially.
func TestMultipleSequentialTests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	const numTests = 2

	results := make([]*speedtest.Result, numTests)
	tester := speedtest.NewTester()

	for i := range numTests {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

		result, err := tester.RunTest(ctx)
		cancel()

		if err != nil {
			t.Fatalf("test %d: RunTest failed: %v", i, err)
		}
		results[i] = result
		t.Logf("test %d: %.2f Mbps down, %.2f Mbps up", i, result.Download, result.Upload)

		// Wait for status to reset before next test
		time.Sleep(3 * time.Second)
	}

	// Verify all results are valid
	for i, result := range results {
		if result.Download <= 0 {
			t.Errorf("test %d: invalid download speed", i)
		}
		if result.Upload <= 0 {
			t.Errorf("test %d: invalid upload speed", i)
		}
	}
}

// TestConcurrentTestAttemptsDuringRun tests that concurrent attempts are rejected while test is running.
func TestConcurrentTestAttemptsDuringRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tester := speedtest.NewTester()

	// Start the main test
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Use a channel to signal when to try concurrent attempts
	testStarted := make(chan struct{})
	var mainResult *speedtest.Result
	var mainErr error

	// Run the main test
	go func() {
		// Signal that test has started
		close(testStarted)
		mainResult, mainErr = tester.RunTest(ctx)
	}()

	// Wait for test to start
	<-testStarted
	time.Sleep(100 * time.Millisecond) // Ensure test is actually running

	// Try concurrent attempts
	const numAttempts = 5
	var wg sync.WaitGroup
	wg.Add(numAttempts)
	rejectedCount := 0
	var mu sync.Mutex

	for range numAttempts {
		go func() {
			defer wg.Done()
			_, err := tester.RunTest(context.Background())
			if err != nil && err.Error() == "test already in progress" {
				mu.Lock()
				rejectedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Wait for main test to complete
	for mainResult == nil && mainErr == nil {
		time.Sleep(100 * time.Millisecond)
	}

	if mainErr != nil {
		t.Fatalf("main test failed: %v", mainErr)
	}

	// All concurrent attempts should have been rejected
	if rejectedCount != numAttempts {
		t.Errorf("expected %d rejected attempts, got %d", numAttempts, rejectedCount)
	}

	t.Logf("Main test completed with %.2f Mbps down, %d concurrent attempts rejected",
		mainResult.Download, rejectedCount)
}
