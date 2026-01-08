// Package speedtest_test provides mock-based tests for the speedtest package.
// These tests use mock interfaces to test internal logic without network calls.
package speedtest_test

import (
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// TestRunTestInitialStatusCheck verifies that RunTest checks the running state correctly.
func TestRunTestInitialStatusCheck(t *testing.T) {
	tests := []struct {
		name         string
		initialState bool
		expectError  bool
		errorMessage string
	}{
		{
			name:         "not running initially - passes check",
			initialState: false,
			expectError:  false, // Would proceed to network calls, not relevant here
		},
		{
			name:         "already running - returns error",
			initialState: true,
			expectError:  true,
			errorMessage: "test already in progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			tester.SetRunning(tt.initialState)

			if !tt.initialState {
				// If not running, we can only verify the initial check passes
				// The test would fail at network level, so we just verify state
				status := tester.GetStatus()
				if status.Running {
					t.Error("expected not running initially")
				}
				return
			}

			// If running, verify error is returned
			result, err := tester.RunTest(nil)
			if tt.expectError {
				if err == nil {
					t.Error("expected error")
				} else if err.Error() != tt.errorMessage {
					t.Errorf("error message: got %q, want %q", err.Error(), tt.errorMessage)
				}
				if result != nil {
					t.Error("expected nil result on error")
				}
			}
		})
	}
}

// TestConcurrentRunTestAttempts tests that concurrent RunTest attempts are properly serialized.
func TestConcurrentRunTestAttempts(t *testing.T) {
	tester := speedtest.NewTester()

	// Simulate running state
	tester.SetRunning(true)

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errorChan := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			_, err := tester.RunTest(nil)
			if err != nil {
				errorChan <- err
			}
		}()
	}

	wg.Wait()
	close(errorChan)

	// All should have returned "already in progress" error
	errorCount := 0
	for err := range errorChan {
		if err.Error() != "test already in progress" {
			t.Errorf("unexpected error: %v", err)
		}
		errorCount++
	}

	if errorCount != numGoroutines {
		t.Errorf("expected %d errors, got %d", numGoroutines, errorCount)
	}
}

// TestStatusTransitionsDuringSimulatedTest tests status transitions during a simulated test flow.
func TestStatusTransitionsDuringSimulatedTest(t *testing.T) {
	tester := speedtest.NewTester()

	// Verify initial state
	status := tester.GetStatus()
	if status.Running {
		t.Error("should not be running initially")
	}
	if status.Phase != "idle" {
		t.Errorf("initial phase: got %q, want 'idle'", status.Phase)
	}

	// Simulate starting a test
	tester.SetRunning(true)
	status = tester.GetStatus()
	if !status.Running {
		t.Error("should be running after SetRunning(true)")
	}

	// Simulate finding server phase
	tester.SetStatus("finding_server", float64(speedtest.ProgressFindingServer))
	status = tester.GetStatus()
	if status.Phase != "finding_server" {
		t.Errorf("phase: got %q, want 'finding_server'", status.Phase)
	}
	if status.Progress != float64(speedtest.ProgressFindingServer) {
		t.Errorf("progress: got %v, want %d", status.Progress, speedtest.ProgressFindingServer)
	}

	// Simulate testing latency phase
	tester.SetStatus("testing_latency", float64(speedtest.ProgressTestingLatency))
	status = tester.GetStatus()
	if status.Phase != "testing_latency" {
		t.Errorf("phase: got %q, want 'testing_latency'", status.Phase)
	}

	// Simulate testing download phase with live speeds
	tester.SetStatus("testing_download", float64(speedtest.ProgressTestingDownload))
	downloadSpeeds := []float64{0, 100, 250, 500, 750, 945}
	for _, speed := range downloadSpeeds {
		tester.SetCurrentSpeeds(speed, 0)
		status = tester.GetStatus()
		if status.CurrentDownload != speed {
			t.Errorf("download speed: got %v, want %v", status.CurrentDownload, speed)
		}
	}

	// Simulate testing upload phase with live speeds
	finalDownload := 945.0
	tester.SetStatus("testing_upload", float64(speedtest.ProgressTestingUpload))
	uploadSpeeds := []float64{0, 50, 100, 200, 400, 850}
	for _, speed := range uploadSpeeds {
		tester.SetCurrentSpeeds(finalDownload, speed)
		status = tester.GetStatus()
		if status.CurrentDownload != finalDownload {
			t.Errorf("final download: got %v, want %v", status.CurrentDownload, finalDownload)
		}
		if status.CurrentUpload != speed {
			t.Errorf("upload speed: got %v, want %v", status.CurrentUpload, speed)
		}
	}

	// Simulate test completion
	tester.SetStatus("complete", float64(speedtest.ProgressComplete))
	status = tester.GetStatus()
	if status.Phase != "complete" {
		t.Errorf("phase: got %q, want 'complete'", status.Phase)
	}
	if status.Progress != 100 {
		t.Errorf("progress: got %v, want 100", status.Progress)
	}

	// Store result
	result := &speedtest.Result{
		Download:     finalDownload,
		Upload:       850,
		Latency:      10,
		Server:       "Test Server",
		Location:     "Test Sponsor, USA",
		Host:         "test.example.com",
		Distance:     50,
		Timestamp:    time.Now(),
		TestDuration: 25,
	}
	tester.SetLastResult(result)

	lastResult := tester.GetLastResult()
	if lastResult == nil {
		t.Fatal("expected non-nil result")
	}
	if lastResult.Download != finalDownload {
		t.Errorf("result download: got %v, want %v", lastResult.Download, finalDownload)
	}

	// Simulate reset to idle
	tester.SetRunning(false)
	tester.SetStatus("idle", 0)
	tester.SetCurrentSpeeds(0, 0)

	status = tester.GetStatus()
	if status.Running {
		t.Error("should not be running after reset")
	}
	if status.Phase != "idle" {
		t.Errorf("phase after reset: got %q, want 'idle'", status.Phase)
	}

	// Result should still be accessible
	lastResult = tester.GetLastResult()
	if lastResult == nil {
		t.Error("result should persist after reset")
	}
}

// TestSimulatedErrorAtEachPhase tests status handling when errors occur at each phase.
func TestSimulatedErrorAtEachPhase(t *testing.T) {
	tests := []struct {
		name          string
		errorPhase    string
		phasesReached []string
	}{
		{
			name:          "error during finding_server",
			errorPhase:    "finding_server",
			phasesReached: []string{"finding_server"},
		},
		{
			name:          "error during testing_latency",
			errorPhase:    "testing_latency",
			phasesReached: []string{"finding_server", "testing_latency"},
		},
		{
			name:          "error during testing_download",
			errorPhase:    "testing_download",
			phasesReached: []string{"finding_server", "testing_latency", "testing_download"},
		},
		{
			name:          "error during testing_upload",
			errorPhase:    "testing_upload",
			phasesReached: []string{"finding_server", "testing_latency", "testing_download", "testing_upload"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()

			// Simulate starting test
			tester.SetRunning(true)
			tester.SetStatus("idle", 0)

			// Progress through phases until error
			for _, phase := range tt.phasesReached {
				switch phase {
				case "finding_server":
					tester.SetStatus("finding_server", float64(speedtest.ProgressFindingServer))
				case "testing_latency":
					tester.SetStatus("testing_latency", float64(speedtest.ProgressTestingLatency))
				case "testing_download":
					tester.SetStatus("testing_download", float64(speedtest.ProgressTestingDownload))
				case "testing_upload":
					tester.SetStatus("testing_upload", float64(speedtest.ProgressTestingUpload))
				}

				// Verify we're at the expected phase
				status := tester.GetStatus()
				if status.Phase != phase {
					t.Errorf("expected phase %q, got %q", phase, status.Phase)
				}
			}

			// Simulate error recovery: reset to idle
			tester.SetRunning(false)
			tester.SetStatus("idle", 0)
			tester.SetCurrentSpeeds(0, 0)

			// Verify reset state
			status := tester.GetStatus()
			if status.Running {
				t.Error("should not be running after error reset")
			}
			if status.Phase != "idle" {
				t.Errorf("phase after error: got %q, want 'idle'", status.Phase)
			}
			if status.Progress != 0 {
				t.Errorf("progress after error: got %v, want 0", status.Progress)
			}
		})
	}
}

// TestLiveSpeedPollingSimulation simulates the live speed polling behavior.
func TestLiveSpeedPollingSimulation(t *testing.T) {
	tester := speedtest.NewTester()

	// Simulate download phase with speed updates over time
	tester.SetRunning(true)
	tester.SetStatus("testing_download", float64(speedtest.ProgressTestingDownload))

	// Simulate speed ramp-up
	speeds := []float64{0, 50, 150, 300, 500, 750, 900, 945}
	var prevSpeed float64

	for _, speed := range speeds {
		tester.SetCurrentSpeeds(speed, 0)
		status := tester.GetStatus()

		// Verify current speed is set
		if status.CurrentDownload != speed {
			t.Errorf("download speed: got %v, want %v", status.CurrentDownload, speed)
		}

		// Verify speed is increasing (or starting from 0)
		if speed < prevSpeed {
			t.Errorf("download speed should not decrease: was %v, now %v", prevSpeed, speed)
		}
		prevSpeed = speed
	}

	// Simulate transition to upload phase
	finalDownload := 945.0
	tester.SetStatus("testing_upload", float64(speedtest.ProgressTestingUpload))

	uploadSpeeds := []float64{0, 25, 75, 150, 300, 500, 800, 850}
	prevSpeed = 0

	for _, speed := range uploadSpeeds {
		tester.SetCurrentSpeeds(finalDownload, speed)
		status := tester.GetStatus()

		// Verify download speed is preserved
		if status.CurrentDownload != finalDownload {
			t.Errorf("download should be preserved: got %v, want %v", status.CurrentDownload, finalDownload)
		}

		// Verify upload speed is set
		if status.CurrentUpload != speed {
			t.Errorf("upload speed: got %v, want %v", status.CurrentUpload, speed)
		}

		// Verify upload speed is increasing
		if speed < prevSpeed {
			t.Errorf("upload speed should not decrease: was %v, now %v", prevSpeed, speed)
		}
		prevSpeed = speed
	}
}

// TestRapidStatusUpdates tests rapid status updates don't cause race conditions.
func TestRapidStatusUpdates(t *testing.T) {
	tester := speedtest.NewTester()

	const iterations = 1000
	var wg sync.WaitGroup
	wg.Add(4) // 4 concurrent updaters

	// Status phase updater
	go func() {
		defer wg.Done()
		phases := []string{"idle", "finding_server", "testing_latency", "testing_download", "testing_upload", "complete"}
		for i := range iterations {
			phase := phases[i%len(phases)]
			progress := float64((i % 100) + 1)
			tester.SetStatus(phase, progress)
		}
	}()

	// Running state updater
	go func() {
		defer wg.Done()
		for i := range iterations {
			tester.SetRunning(i%2 == 0)
		}
	}()

	// Speed updater
	go func() {
		defer wg.Done()
		for i := range iterations {
			dl := float64((i % 1000) + 1)
			ul := float64((i % 500) + 1)
			tester.SetCurrentSpeeds(dl, ul)
		}
	}()

	// Reader
	go func() {
		defer wg.Done()
		for range iterations {
			status := tester.GetStatus()
			// Just access all fields to detect any corruption
			_ = status.Phase
			_ = status.Progress
			_ = status.Running
			_ = status.CurrentDownload
			_ = status.CurrentUpload
		}
	}()

	wg.Wait()

	// Verify final state is valid
	status := tester.GetStatus()
	if status.Phase == "" {
		t.Error("phase should not be empty after updates")
	}
}

// TestBuildTestResultFromParamsEdgeCases tests edge cases in BuildTestResultFromParams.
func TestBuildTestResultFromParamsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		dlSpeed  float64
		ulSpeed  float64
		latency  time.Duration
		server   string
		sponsor  string
		country  string
		host     string
		distance float64
	}{
		{
			name:     "all zeros",
			dlSpeed:  0,
			ulSpeed:  0,
			latency:  0,
			server:   "",
			sponsor:  "",
			country:  "",
			host:     "",
			distance: 0,
		},
		{
			name:     "negative-like values (treated as positive)",
			dlSpeed:  0.001,
			ulSpeed:  0.001,
			latency:  1 * time.Nanosecond, // Will round to 0ms
			server:   "x",
			sponsor:  "y",
			country:  "z",
			host:     "h",
			distance: 0.001,
		},
		{
			name:     "very large values",
			dlSpeed:  100000,
			ulSpeed:  100000,
			latency:  10 * time.Second,
			server:   "Ultra Fast Server",
			sponsor:  "Major ISP",
			country:  "Country",
			host:     "ultra.fast.example.com:8080",
			distance: 50000, // km
		},
		{
			name:     "unicode in all strings",
			dlSpeed:  100,
			ulSpeed:  50,
			latency:  10 * time.Millisecond,
			server:   "サーバー名",
			sponsor:  "スポンサー",
			country:  "日本",
			host:     "example.co.jp:8080",
			distance: 5000,
		},
		{
			name:     "special characters",
			dlSpeed:  100,
			ulSpeed:  50,
			latency:  10 * time.Millisecond,
			server:   "Server <with> 'special' \"chars\" & more",
			sponsor:  "Sponsor/Provider",
			country:  "Country (Region)",
			host:     "host.example.com:8080",
			distance: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			startTime := time.Now().Add(-10 * time.Second)

			result := tester.BuildTestResultFromParams(
				tt.dlSpeed,
				tt.ulSpeed,
				tt.latency,
				tt.server,
				tt.sponsor,
				tt.country,
				tt.host,
				tt.distance,
				startTime,
			)

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Download != tt.dlSpeed {
				t.Errorf("Download: got %v, want %v", result.Download, tt.dlSpeed)
			}
			if result.Upload != tt.ulSpeed {
				t.Errorf("Upload: got %v, want %v", result.Upload, tt.ulSpeed)
			}
			if result.Server != tt.server {
				t.Errorf("Server: got %q, want %q", result.Server, tt.server)
			}
			if result.Host != tt.host {
				t.Errorf("Host: got %q, want %q", result.Host, tt.host)
			}
			if result.Distance != tt.distance {
				t.Errorf("Distance: got %v, want %v", result.Distance, tt.distance)
			}

			expectedLocation := tt.sponsor + ", " + tt.country
			if result.Location != expectedLocation {
				t.Errorf("Location: got %q, want %q", result.Location, expectedLocation)
			}
		})
	}
}

// TestTesterFieldsAfterResultStorage tests that tester fields are properly set after storing a result.
func TestTesterFieldsAfterResultStorage(t *testing.T) {
	tester := speedtest.NewTester()

	// Create multiple results with different values
	results := []*speedtest.Result{
		{
			Download:     100,
			Upload:       50,
			Latency:      10,
			Server:       "Server 1",
			Location:     "Location 1",
			Host:         "host1.example.com",
			Distance:     25,
			Timestamp:    time.Now(),
			TestDuration: 15,
		},
		{
			Download:     200,
			Upload:       100,
			Latency:      8,
			Server:       "Server 2",
			Location:     "Location 2",
			Host:         "host2.example.com",
			Distance:     50,
			Timestamp:    time.Now(),
			TestDuration: 20,
		},
		{
			Download:     500,
			Upload:       250,
			Latency:      5,
			Server:       "Server 3",
			Location:     "Location 3",
			Host:         "host3.example.com",
			Distance:     100,
			Timestamp:    time.Now(),
			TestDuration: 25,
		},
	}

	for i, expected := range results {
		tester.SetLastResult(expected)

		actual := tester.GetLastResult()
		if actual == nil {
			t.Fatalf("iteration %d: expected non-nil result", i)
		}

		if actual.Download != expected.Download {
			t.Errorf("iteration %d: Download got %v, want %v", i, actual.Download, expected.Download)
		}
		if actual.Upload != expected.Upload {
			t.Errorf("iteration %d: Upload got %v, want %v", i, actual.Upload, expected.Upload)
		}
		if actual.Latency != expected.Latency {
			t.Errorf("iteration %d: Latency got %v, want %v", i, actual.Latency, expected.Latency)
		}
		if actual.Server != expected.Server {
			t.Errorf("iteration %d: Server got %q, want %q", i, actual.Server, expected.Server)
		}
		if actual.Location != expected.Location {
			t.Errorf("iteration %d: Location got %q, want %q", i, actual.Location, expected.Location)
		}
		if actual.Host != expected.Host {
			t.Errorf("iteration %d: Host got %q, want %q", i, actual.Host, expected.Host)
		}
		if actual.Distance != expected.Distance {
			t.Errorf("iteration %d: Distance got %v, want %v", i, actual.Distance, expected.Distance)
		}
		if actual.TestDuration != expected.TestDuration {
			t.Errorf("iteration %d: TestDuration got %v, want %v", i, actual.TestDuration, expected.TestDuration)
		}
	}
}

// TestProgressMonotonicity tests that progress values should be monotonically increasing during a test.
func TestProgressMonotonicity(t *testing.T) {
	tester := speedtest.NewTester()

	progressValues := []struct {
		phase    string
		progress int
	}{
		{"finding_server", speedtest.ProgressFindingServer},
		{"testing_latency", speedtest.ProgressTestingLatency},
		{"testing_download", speedtest.ProgressTestingDownload},
		{"testing_upload", speedtest.ProgressTestingUpload},
		{"complete", speedtest.ProgressComplete},
	}

	var prevProgress int
	for _, pv := range progressValues {
		if pv.progress <= prevProgress && prevProgress != 0 {
			t.Errorf("progress should increase: %d is not greater than %d", pv.progress, prevProgress)
		}
		prevProgress = pv.progress

		tester.SetStatus(pv.phase, float64(pv.progress))
		status := tester.GetStatus()

		if status.Phase != pv.phase {
			t.Errorf("phase: got %q, want %q", status.Phase, pv.phase)
		}
		if status.Progress != float64(pv.progress) {
			t.Errorf("progress: got %v, want %v", status.Progress, float64(pv.progress))
		}
	}
}

// TestServerIDPersistence tests that server ID persists across status changes.
func TestServerIDPersistence(t *testing.T) {
	tester := speedtest.NewTesterWithConfig("persistent-server-id")

	// Change various status values
	tester.SetStatus("finding_server", 10)
	if tester.TesterServerID() != "persistent-server-id" {
		t.Errorf("serverID should persist: got %q", tester.TesterServerID())
	}

	tester.SetRunning(true)
	if tester.TesterServerID() != "persistent-server-id" {
		t.Errorf("serverID should persist: got %q", tester.TesterServerID())
	}

	tester.SetCurrentSpeeds(500, 250)
	if tester.TesterServerID() != "persistent-server-id" {
		t.Errorf("serverID should persist: got %q", tester.TesterServerID())
	}

	tester.SetLastResult(&speedtest.Result{Download: 100})
	if tester.TesterServerID() != "persistent-server-id" {
		t.Errorf("serverID should persist: got %q", tester.TesterServerID())
	}

	// Only SetServerID should change it
	tester.SetServerID("new-server-id")
	if tester.TesterServerID() != "new-server-id" {
		t.Errorf("serverID should change: got %q", tester.TesterServerID())
	}
}
