package speedtest_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// TestRunTestReturnsErrorWhenAlreadyRunning verifies that RunTest returns an error
// when a test is already in progress.
func TestRunTestReturnsErrorWhenAlreadyRunning(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"single attempt while running"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			tester.SetRunning(true)

			result, err := tester.RunTest(context.Background())

			if err == nil {
				t.Error("expected error when test already running")
			}
			if err != nil && err.Error() != "test already in progress" {
				t.Errorf("expected 'test already in progress' error, got %q", err.Error())
			}
			if result != nil {
				t.Error("expected nil result when error occurs")
			}
		})
	}
}

// TestRunTestNotRunningBeforeStart verifies that the tester is not running initially.
func TestRunTestNotRunningBeforeStart(t *testing.T) {
	tester := speedtest.NewTester()

	status := tester.GetStatus()
	if status.Running {
		t.Error("tester should not be running initially")
	}
	if status.Phase != "idle" {
		t.Errorf("expected phase 'idle', got %q", status.Phase)
	}
}

// TestBuildTestResultFromParamsLocationFormat tests the location format in buildTestResult.
func TestBuildTestResultFromParamsLocationFormat(t *testing.T) {
	tests := []struct {
		name             string
		sponsor          string
		country          string
		expectedLocation string
	}{
		{
			name:             "standard format",
			sponsor:          "Comcast",
			country:          "United States",
			expectedLocation: "Comcast, United States",
		},
		{
			name:             "empty sponsor",
			sponsor:          "",
			country:          "Germany",
			expectedLocation: ", Germany",
		},
		{
			name:             "empty country",
			sponsor:          "Vodafone",
			country:          "",
			expectedLocation: "Vodafone, ",
		},
		{
			name:             "both empty",
			sponsor:          "",
			country:          "",
			expectedLocation: ", ",
		},
		{
			name:             "special characters in sponsor",
			sponsor:          "AT&T Wireless",
			country:          "USA",
			expectedLocation: "AT&T Wireless, USA",
		},
		{
			name:             "unicode characters",
			sponsor:          "Deutsche Telekom",
			country:          "Deutschland",
			expectedLocation: "Deutsche Telekom, Deutschland",
		},
		{
			name:             "long names",
			sponsor:          "Very Long Internet Service Provider Name Inc.",
			country:          "United States of America",
			expectedLocation: "Very Long Internet Service Provider Name Inc., United States of America",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				100.0,
				50.0,
				10*time.Millisecond,
				"Test Server",
				tt.sponsor,
				tt.country,
				"host.example.com",
				50.0,
				time.Now(),
			)

			if result.Location != tt.expectedLocation {
				t.Errorf("Location: got %q, want %q", result.Location, tt.expectedLocation)
			}
		})
	}
}

// TestBuildTestResultFromParamsLatencyConversion tests latency conversion from duration to milliseconds.
func TestBuildTestResultFromParamsLatencyConversion(t *testing.T) {
	tests := []struct {
		name            string
		latency         time.Duration
		expectedLatency float64
	}{
		{
			name:            "1 millisecond",
			latency:         1 * time.Millisecond,
			expectedLatency: 1,
		},
		{
			name:            "10 milliseconds",
			latency:         10 * time.Millisecond,
			expectedLatency: 10,
		},
		{
			name:            "100 milliseconds",
			latency:         100 * time.Millisecond,
			expectedLatency: 100,
		},
		{
			name:            "1 second",
			latency:         1 * time.Second,
			expectedLatency: 1000,
		},
		{
			name:            "zero latency",
			latency:         0,
			expectedLatency: 0,
		},
		{
			name:            "fractional milliseconds (rounds down)",
			latency:         1500 * time.Microsecond,
			expectedLatency: 1, // 1.5ms rounds to 1ms with Milliseconds()
		},
		{
			name:            "sub-millisecond (rounds to zero)",
			latency:         500 * time.Microsecond,
			expectedLatency: 0, // 0.5ms rounds to 0ms
		},
		{
			name:            "high latency satellite",
			latency:         600 * time.Millisecond,
			expectedLatency: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				100.0,
				50.0,
				tt.latency,
				"Test Server",
				"Sponsor",
				"Country",
				"host.example.com",
				50.0,
				time.Now(),
			)

			if result.Latency != tt.expectedLatency {
				t.Errorf("Latency: got %v, want %v", result.Latency, tt.expectedLatency)
			}
		})
	}
}

// TestBuildTestResultFromParamsTestDurationCalculation tests the test duration calculation.
func TestBuildTestResultFromParamsTestDurationCalculation(t *testing.T) {
	tests := []struct {
		name             string
		secondsAgo       int
		expectedMinDelta float64
		expectedMaxDelta float64
	}{
		{
			name:             "1 second test",
			secondsAgo:       1,
			expectedMinDelta: 0.9,
			expectedMaxDelta: 1.5,
		},
		{
			name:             "5 second test",
			secondsAgo:       5,
			expectedMinDelta: 4.9,
			expectedMaxDelta: 5.5,
		},
		{
			name:             "10 second test",
			secondsAgo:       10,
			expectedMinDelta: 9.9,
			expectedMaxDelta: 10.5,
		},
		{
			name:             "30 second test",
			secondsAgo:       30,
			expectedMinDelta: 29.9,
			expectedMaxDelta: 30.5,
		},
		{
			name:             "60 second test",
			secondsAgo:       60,
			expectedMinDelta: 59.9,
			expectedMaxDelta: 60.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			startTime := time.Now().Add(-time.Duration(tt.secondsAgo) * time.Second)
			result := tester.BuildTestResultFromParams(
				100.0,
				50.0,
				10*time.Millisecond,
				"Test Server",
				"Sponsor",
				"Country",
				"host.example.com",
				50.0,
				startTime,
			)

			if result.TestDuration < tt.expectedMinDelta || result.TestDuration > tt.expectedMaxDelta {
				t.Errorf("TestDuration: got %v, want between %v and %v",
					result.TestDuration, tt.expectedMinDelta, tt.expectedMaxDelta)
			}
		})
	}
}

// TestBuildTestResultFromParamsTimestamp tests that timestamp is set correctly.
func TestBuildTestResultFromParamsTimestamp(t *testing.T) {
	tester := speedtest.NewTester()
	beforeCall := time.Now()

	result := tester.BuildTestResultFromParams(
		100.0,
		50.0,
		10*time.Millisecond,
		"Test Server",
		"Sponsor",
		"Country",
		"host.example.com",
		50.0,
		time.Now().Add(-10*time.Second),
	)

	afterCall := time.Now()

	if result.Timestamp.Before(beforeCall) {
		t.Error("Timestamp should not be before the function call")
	}
	if result.Timestamp.After(afterCall) {
		t.Error("Timestamp should not be after the function call")
	}
}

// TestBuildTestResultFromParamsSpeedValues tests various speed values.
func TestBuildTestResultFromParamsSpeedValues(t *testing.T) {
	tests := []struct {
		name     string
		download float64
		upload   float64
	}{
		{"zero speeds", 0, 0},
		{"typical DSL", 25.0, 5.0},
		{"typical cable", 200.0, 20.0},
		{"symmetric fiber", 1000.0, 1000.0},
		{"asymmetric fiber", 940.0, 880.0},
		{"mobile 4G", 75.0, 25.0},
		{"mobile 5G", 500.0, 100.0},
		{"gigabit", 1000.0, 1000.0},
		{"10 gigabit", 10000.0, 10000.0},
		{"very slow", 0.1, 0.05},
		{"fractional", 123.456, 78.901},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				tt.download,
				tt.upload,
				10*time.Millisecond,
				"Test Server",
				"Sponsor",
				"Country",
				"host.example.com",
				50.0,
				time.Now(),
			)

			if result.Download != tt.download {
				t.Errorf("Download: got %v, want %v", result.Download, tt.download)
			}
			if result.Upload != tt.upload {
				t.Errorf("Upload: got %v, want %v", result.Upload, tt.upload)
			}
		})
	}
}

// TestBuildTestResultFromParamsDistanceValues tests various distance values.
func TestBuildTestResultFromParamsDistanceValues(t *testing.T) {
	tests := []struct {
		name     string
		distance float64
	}{
		{"zero distance", 0},
		{"local server", 1.5},
		{"nearby server", 25.0},
		{"regional server", 100.0},
		{"distant server", 500.0},
		{"cross-country", 2500.0},
		{"international", 8000.0},
		{"fractional", 123.456},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				100.0,
				50.0,
				10*time.Millisecond,
				"Test Server",
				"Sponsor",
				"Country",
				"host.example.com",
				tt.distance,
				time.Now(),
			)

			if result.Distance != tt.distance {
				t.Errorf("Distance: got %v, want %v", result.Distance, tt.distance)
			}
		})
	}
}

// TestBuildTestResultFromParamsServerInfo tests server information fields.
func TestBuildTestResultFromParamsServerInfo(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		host       string
	}{
		{"empty server", "", ""},
		{"typical server", "Comcast Speed Test", "speedtest.comcast.net:8080"},
		{"with port", "Test Server", "speedtest.example.com:5201"},
		{"IP address host", "Local Server", "192.168.1.100:8080"},
		{"IPv6 host", "IPv6 Server", "[2001:db8::1]:8080"},
		{"long server name", "Very Long Server Name For Testing Purposes", "long.host.example.com:8080"},
		{"unicode server", "テストサーバー", "test.example.jp:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				100.0,
				50.0,
				10*time.Millisecond,
				tt.serverName,
				"Sponsor",
				"Country",
				tt.host,
				50.0,
				time.Now(),
			)

			if result.Server != tt.serverName {
				t.Errorf("Server: got %q, want %q", result.Server, tt.serverName)
			}
			if result.Host != tt.host {
				t.Errorf("Host: got %q, want %q", result.Host, tt.host)
			}
		})
	}
}

// TestTesterStatusIsolation tests that multiple tester instances have isolated status.
func TestTesterStatusIsolation(t *testing.T) {
	testers := make([]*speedtest.Tester, 5)
	for i := range testers {
		testers[i] = speedtest.NewTester()
	}

	// Set different statuses for each tester
	phases := []string{"idle", "finding_server", "testing_latency", "testing_download", "testing_upload"}
	for i, tester := range testers {
		tester.SetStatus(phases[i], float64(i*20))
		tester.SetCurrentSpeeds(float64(i*100), float64(i*50))
		tester.SetRunning(i%2 == 0)
	}

	// Verify each tester has its own isolated state
	for i, tester := range testers {
		status := tester.GetStatus()
		if status.Phase != phases[i] {
			t.Errorf("tester %d: Phase got %q, want %q", i, status.Phase, phases[i])
		}
		if status.Progress != float64(i*20) {
			t.Errorf("tester %d: Progress got %v, want %v", i, status.Progress, float64(i*20))
		}
		if status.CurrentDownload != float64(i*100) {
			t.Errorf("tester %d: CurrentDownload got %v, want %v", i, status.CurrentDownload, float64(i*100))
		}
		if status.CurrentUpload != float64(i*50) {
			t.Errorf("tester %d: CurrentUpload got %v, want %v", i, status.CurrentUpload, float64(i*50))
		}
		if status.Running != (i%2 == 0) {
			t.Errorf("tester %d: Running got %v, want %v", i, status.Running, i%2 == 0)
		}
	}
}

// TestTesterResultIsolation tests that multiple tester instances have isolated results.
func TestTesterResultIsolation(t *testing.T) {
	testers := make([]*speedtest.Tester, 3)
	for i := range testers {
		testers[i] = speedtest.NewTester()
	}

	// Set different results for each tester
	for i, tester := range testers {
		result := &speedtest.Result{
			Download:  float64((i + 1) * 100),
			Upload:    float64((i + 1) * 50),
			Latency:   float64((i + 1) * 10),
			Server:    "Server" + string(rune('A'+i)),
			Timestamp: time.Now(),
		}
		tester.SetLastResult(result)
	}

	// Verify each tester has its own isolated result
	for i, tester := range testers {
		result := tester.GetLastResult()
		if result == nil {
			t.Errorf("tester %d: expected non-nil result", i)
			continue
		}
		expectedDownload := float64((i + 1) * 100)
		if result.Download != expectedDownload {
			t.Errorf("tester %d: Download got %v, want %v", i, result.Download, expectedDownload)
		}
	}
}

// TestTesterServerIDIsolation tests that multiple tester instances have isolated server IDs.
func TestTesterServerIDIsolation(t *testing.T) {
	tester1 := speedtest.NewTesterWithConfig("server-1")
	tester2 := speedtest.NewTesterWithConfig("server-2")
	tester3 := speedtest.NewTester()

	if tester1.TesterServerID() != "server-1" {
		t.Errorf("tester1: got %q, want 'server-1'", tester1.TesterServerID())
	}
	if tester2.TesterServerID() != "server-2" {
		t.Errorf("tester2: got %q, want 'server-2'", tester2.TesterServerID())
	}
	if tester3.TesterServerID() != "" {
		t.Errorf("tester3: got %q, want empty", tester3.TesterServerID())
	}

	// Modify one and verify others are unchanged
	tester1.SetServerID("modified-1")
	if tester1.TesterServerID() != "modified-1" {
		t.Errorf("tester1 after modification: got %q, want 'modified-1'", tester1.TesterServerID())
	}
	if tester2.TesterServerID() != "server-2" {
		t.Errorf("tester2 should be unchanged: got %q, want 'server-2'", tester2.TesterServerID())
	}
}

// TestRunTestConcurrentAlreadyRunning tests concurrent RunTest calls when already running.
func TestRunTestConcurrentAlreadyRunning(t *testing.T) {
	tester := speedtest.NewTester()
	tester.SetRunning(true)

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			_, err := tester.RunTest(context.Background())
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// All should have returned errors
	errorCount := 0
	for err := range errors {
		if err.Error() != "test already in progress" {
			t.Errorf("unexpected error: %v", err)
		}
		errorCount++
	}

	if errorCount != numGoroutines {
		t.Errorf("expected %d errors, got %d", numGoroutines, errorCount)
	}
}

// TestPhaseProgressMapping tests that phases have expected progress values.
func TestPhaseProgressMapping(t *testing.T) {
	tests := []struct {
		phase            string
		expectedProgress int
	}{
		{"finding_server", speedtest.ProgressFindingServer},
		{"testing_latency", speedtest.ProgressTestingLatency},
		{"testing_download", speedtest.ProgressTestingDownload},
		{"testing_upload", speedtest.ProgressTestingUpload},
		{"complete", speedtest.ProgressComplete},
	}

	tester := speedtest.NewTester()

	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			tester.SetStatus(tt.phase, float64(tt.expectedProgress))
			status := tester.GetStatus()

			if status.Phase != tt.phase {
				t.Errorf("Phase: got %q, want %q", status.Phase, tt.phase)
			}
			if status.Progress != float64(tt.expectedProgress) {
				t.Errorf("Progress: got %v, want %v", status.Progress, float64(tt.expectedProgress))
			}
		})
	}
}

// TestSpeedUpdatesDuringDownloadPhase tests live speed updates during download phase.
func TestSpeedUpdatesDuringDownloadPhase(t *testing.T) {
	tester := speedtest.NewTester()
	tester.SetStatus("testing_download", float64(speedtest.ProgressTestingDownload))
	tester.SetRunning(true)

	// Simulate download speed progression
	speeds := []float64{0, 25.5, 100.0, 250.5, 500.0, 750.0, 945.5}

	for _, speed := range speeds {
		tester.SetCurrentSpeeds(speed, 0)
		status := tester.GetStatus()

		if status.CurrentDownload != speed {
			t.Errorf("CurrentDownload: got %v, want %v", status.CurrentDownload, speed)
		}
		if status.CurrentUpload != 0 {
			t.Errorf("CurrentUpload should be 0 during download, got %v", status.CurrentUpload)
		}
	}
}

// TestSpeedUpdatesDuringUploadPhase tests live speed updates during upload phase.
func TestSpeedUpdatesDuringUploadPhase(t *testing.T) {
	tester := speedtest.NewTester()
	tester.SetStatus("testing_upload", float64(speedtest.ProgressTestingUpload))
	tester.SetRunning(true)

	// Set final download speed (from previous phase)
	finalDownload := 945.5

	// Simulate upload speed progression while maintaining download speed
	uploadSpeeds := []float64{0, 10.5, 50.0, 100.0, 250.0, 500.0, 880.0}

	for _, uploadSpeed := range uploadSpeeds {
		tester.SetCurrentSpeeds(finalDownload, uploadSpeed)
		status := tester.GetStatus()

		if status.CurrentDownload != finalDownload {
			t.Errorf("CurrentDownload: got %v, want %v", status.CurrentDownload, finalDownload)
		}
		if status.CurrentUpload != uploadSpeed {
			t.Errorf("CurrentUpload: got %v, want %v", status.CurrentUpload, uploadSpeed)
		}
	}
}

// TestBytesToMbpsConversionExamples tests the bytes to Mbps conversion with real-world examples.
func TestBytesToMbpsConversionExamples(t *testing.T) {
	tests := []struct {
		name         string
		bytesPerSec  float64
		expectedMbps float64
		description  string
	}{
		{
			name:         "1 Mbps",
			bytesPerSec:  125000,
			expectedMbps: 1.0,
			description:  "Basic DSL upload",
		},
		{
			name:         "10 Mbps",
			bytesPerSec:  1250000,
			expectedMbps: 10.0,
			description:  "Typical DSL download",
		},
		{
			name:         "25 Mbps",
			bytesPerSec:  3125000,
			expectedMbps: 25.0,
			description:  "Entry-level cable",
		},
		{
			name:         "100 Mbps",
			bytesPerSec:  12500000,
			expectedMbps: 100.0,
			description:  "Fast cable",
		},
		{
			name:         "1 Gbps",
			bytesPerSec:  125000000,
			expectedMbps: 1000.0,
			description:  "Gigabit fiber",
		},
		{
			name:         "10 Gbps",
			bytesPerSec:  1250000000,
			expectedMbps: 10000.0,
			description:  "10 gigabit enterprise",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mbps := tt.bytesPerSec / speedtest.BytesToMbps
			if mbps != tt.expectedMbps {
				t.Errorf("%s: got %v Mbps, want %v Mbps", tt.description, mbps, tt.expectedMbps)
			}
		})
	}
}

// TestStatusResetAfterTest tests that status is reset correctly after test completion.
func TestStatusResetAfterTest(t *testing.T) {
	tester := speedtest.NewTester()

	// Simulate test in progress
	tester.SetRunning(true)
	tester.SetStatus("testing_upload", float64(speedtest.ProgressTestingUpload))
	tester.SetCurrentSpeeds(500.0, 250.0)

	status := tester.GetStatus()
	if !status.Running {
		t.Error("should be running during test")
	}

	// Simulate test completion
	tester.SetStatus("complete", float64(speedtest.ProgressComplete))
	status = tester.GetStatus()
	if status.Phase != "complete" {
		t.Errorf("Phase: got %q, want 'complete'", status.Phase)
	}
	if status.Progress != 100 {
		t.Errorf("Progress: got %v, want 100", status.Progress)
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
		t.Errorf("Phase: got %q, want 'idle'", status.Phase)
	}
	if status.Progress != 0 {
		t.Errorf("Progress: got %v, want 0", status.Progress)
	}
	if status.CurrentDownload != 0 {
		t.Errorf("CurrentDownload: got %v, want 0", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("CurrentUpload: got %v, want 0", status.CurrentUpload)
	}
}

// TestErrorRecoveryFlow tests status handling during error recovery.
func TestErrorRecoveryFlow(t *testing.T) {
	tests := []struct {
		name       string
		errorAt    string
		phases     []string
		finalPhase string
	}{
		{
			name:       "error at finding server",
			errorAt:    "finding_server",
			phases:     []string{"idle", "finding_server", "idle"},
			finalPhase: "idle",
		},
		{
			name:       "error at testing latency",
			errorAt:    "testing_latency",
			phases:     []string{"idle", "finding_server", "testing_latency", "idle"},
			finalPhase: "idle",
		},
		{
			name:       "error at testing download",
			errorAt:    "testing_download",
			phases:     []string{"idle", "finding_server", "testing_latency", "testing_download", "idle"},
			finalPhase: "idle",
		},
		{
			name:    "error at testing upload",
			errorAt: "testing_upload",
			phases: []string{
				"idle",
				"finding_server",
				"testing_latency",
				"testing_download",
				"testing_upload",
				"idle",
			},
			finalPhase: "idle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()

			// Simulate the phase progression
			for _, phase := range tt.phases {
				tester.SetStatus(phase, 0)
				if phase != "idle" {
					tester.SetRunning(true)
				} else {
					tester.SetRunning(false)
				}
			}

			status := tester.GetStatus()
			if status.Phase != tt.finalPhase {
				t.Errorf("final Phase: got %q, want %q", status.Phase, tt.finalPhase)
			}
			if status.Running {
				t.Error("should not be running after error")
			}
		})
	}
}

// TestLastResultPersistence tests that last result persists after status reset.
func TestLastResultPersistence(t *testing.T) {
	tester := speedtest.NewTester()

	// Set a result
	result := &speedtest.Result{
		Download:     500.0,
		Upload:       250.0,
		Latency:      15.0,
		Server:       "Test Server",
		Location:     "Test Location",
		Host:         "test.example.com",
		Distance:     50.0,
		Timestamp:    time.Now(),
		TestDuration: 20.0,
	}
	tester.SetLastResult(result)

	// Reset status
	tester.SetStatus("idle", 0)
	tester.SetRunning(false)
	tester.SetCurrentSpeeds(0, 0)

	// Result should still be available
	lastResult := tester.GetLastResult()
	if lastResult == nil {
		t.Fatal("expected result to persist after status reset")
	}
	if lastResult.Download != 500.0 {
		t.Errorf("Download: got %v, want 500.0", lastResult.Download)
	}
	if lastResult.Upload != 250.0 {
		t.Errorf("Upload: got %v, want 250.0", lastResult.Upload)
	}
}

// TestMultipleTestRuns tests that multiple test results are stored correctly.
func TestMultipleTestRuns(t *testing.T) {
	tester := speedtest.NewTester()

	// Simulate multiple test runs
	for i := 1; i <= 5; i++ {
		result := &speedtest.Result{
			Download:     float64(i * 100),
			Upload:       float64(i * 50),
			Latency:      float64(i * 5),
			Server:       "Test Server",
			TestDuration: float64(i * 10),
		}
		tester.SetLastResult(result)

		lastResult := tester.GetLastResult()
		if lastResult.Download != float64(i*100) {
			t.Errorf("run %d: Download got %v, want %v", i, lastResult.Download, float64(i*100))
		}
	}

	// Only last result should be stored
	lastResult := tester.GetLastResult()
	if lastResult.Download != 500.0 {
		t.Errorf("final result Download: got %v, want 500.0", lastResult.Download)
	}
}

// TestConcurrentTesterCreation tests creating many testers concurrently.
func TestConcurrentTesterCreation(t *testing.T) {
	const numTesters = 100
	testers := make(chan *speedtest.Tester, numTesters)

	var wg sync.WaitGroup
	wg.Add(numTesters)

	for range numTesters {
		go func() {
			defer wg.Done()
			tester := speedtest.NewTester()
			testers <- tester
		}()
	}

	wg.Wait()
	close(testers)

	// Verify all testers are valid
	count := 0
	for tester := range testers {
		count++
		if tester == nil {
			t.Error("got nil tester")
			continue
		}
		status := tester.GetStatus()
		if status.Phase != "idle" {
			t.Errorf("expected Phase 'idle', got %q", status.Phase)
		}
	}

	if count != numTesters {
		t.Errorf("expected %d testers, got %d", numTesters, count)
	}
}

// TestStatusCopySemantics tests that GetStatus returns a copy, not a reference.
func TestStatusCopySemantics(t *testing.T) {
	tester := speedtest.NewTester()
	tester.SetStatus("testing", 50)

	// Get status
	status1 := tester.GetStatus()

	// Modify the tester
	tester.SetStatus("complete", 100)

	// status1 should still have old values
	if status1.Phase != "testing" {
		t.Errorf("status1.Phase should be 'testing', got %q", status1.Phase)
	}
	if status1.Progress != 50 {
		t.Errorf("status1.Progress should be 50, got %v", status1.Progress)
	}

	// New status should have new values
	status2 := tester.GetStatus()
	if status2.Phase != "complete" {
		t.Errorf("status2.Phase should be 'complete', got %q", status2.Phase)
	}
	if status2.Progress != 100 {
		t.Errorf("status2.Progress should be 100, got %v", status2.Progress)
	}
}

// TestProgressConstantsValues tests the exact values of progress constants.
func TestProgressConstantsValues(t *testing.T) {
	// These tests ensure the constants have the expected values as documented
	if speedtest.ProgressFindingServer != 10 {
		t.Errorf("ProgressFindingServer: got %d, want 10", speedtest.ProgressFindingServer)
	}
	if speedtest.ProgressTestingLatency != 20 {
		t.Errorf("ProgressTestingLatency: got %d, want 20", speedtest.ProgressTestingLatency)
	}
	if speedtest.ProgressTestingDownload != 40 {
		t.Errorf("ProgressTestingDownload: got %d, want 40", speedtest.ProgressTestingDownload)
	}
	if speedtest.ProgressTestingUpload != 70 {
		t.Errorf("ProgressTestingUpload: got %d, want 70", speedtest.ProgressTestingUpload)
	}
	if speedtest.ProgressComplete != 100 {
		t.Errorf("ProgressComplete: got %d, want 100", speedtest.ProgressComplete)
	}
}

// TestTimingConstantsValues tests the exact values of timing constants.
func TestTimingConstantsValues(t *testing.T) {
	if speedtest.SpeedPollInterval != 100*time.Millisecond {
		t.Errorf("SpeedPollInterval: got %v, want 100ms", speedtest.SpeedPollInterval)
	}
	if speedtest.IdleResetDelay != 2*time.Second {
		t.Errorf("IdleResetDelay: got %v, want 2s", speedtest.IdleResetDelay)
	}
	if speedtest.BytesToMbps != 125000.0 {
		t.Errorf("BytesToMbps: got %v, want 125000.0", speedtest.BytesToMbps)
	}
}

// TestNewTesterWithConfigVariousIDs tests NewTesterWithConfig with various server IDs.
func TestNewTesterWithConfigVariousIDs(t *testing.T) {
	tests := []struct {
		name     string
		serverID string
	}{
		{"empty", ""},
		{"numeric", "12345"},
		{"alphanumeric", "server123abc"},
		{"with hyphens", "server-123-abc-456"},
		{"with underscores", "server_123_abc"},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000"},
		{"url-like", "speedtest.example.com:8080"},
		{"path-like", "/api/v1/servers/123"},
		{"with spaces", "Server Name With Spaces"},
		{"unicode", "サーバー123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTesterWithConfig(tt.serverID)

			if tester == nil {
				t.Fatal("expected non-nil tester")
			}

			if tester.TesterServerID() != tt.serverID {
				t.Errorf("TesterServerID: got %q, want %q", tester.TesterServerID(), tt.serverID)
			}

			// Verify other fields are initialized correctly
			status := tester.GetStatus()
			if status.Phase != "idle" {
				t.Errorf("Phase: got %q, want 'idle'", status.Phase)
			}
			if status.Running {
				t.Error("should not be running initially")
			}
		})
	}
}

// TestSetServerIDReplace tests replacing server ID.
func TestSetServerIDReplace(t *testing.T) {
	tester := speedtest.NewTesterWithConfig("initial-server")

	if tester.TesterServerID() != "initial-server" {
		t.Fatalf("initial serverID: got %q, want 'initial-server'", tester.TesterServerID())
	}

	// Replace with new value
	tester.SetServerID("replacement-server")
	if tester.TesterServerID() != "replacement-server" {
		t.Errorf("after replacement: got %q, want 'replacement-server'", tester.TesterServerID())
	}

	// Replace with empty
	tester.SetServerID("")
	if tester.TesterServerID() != "" {
		t.Errorf("after empty: got %q, want empty", tester.TesterServerID())
	}

	// Replace empty with new value
	tester.SetServerID("new-server")
	if tester.TesterServerID() != "new-server" {
		t.Errorf("after new: got %q, want 'new-server'", tester.TesterServerID())
	}
}
