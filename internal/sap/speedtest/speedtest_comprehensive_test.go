// Package speedtest_test provides comprehensive tests to improve coverage of the speedtest package.
// These tests focus on maximizing coverage of testable code paths without requiring network access.
package speedtest_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// TestRunTestConcurrentAlreadyRunningRace validates thread-safety of the running check.
func TestRunTestConcurrentAlreadyRunningRace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		goroutines int
		iterations int
	}{
		{"small scale", 5, 10},
		{"medium scale", 20, 50},
		{"large scale", 50, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			tester.SetRunning(true)

			var wg sync.WaitGroup
			errCount := make(chan int, 1)
			count := 0

			wg.Add(tt.goroutines)
			for range tt.goroutines {
				go func() {
					defer wg.Done()
					for range tt.iterations {
						_, err := tester.RunTest(context.Background())
						if err != nil && err.Error() == "test already in progress" {
							count++
						}
					}
				}()
			}

			wg.Wait()
			errCount <- count

			totalExpected := tt.goroutines * tt.iterations
			got := <-errCount
			if got != totalExpected {
				t.Errorf("expected %d errors, got %d", totalExpected, got)
			}
		})
	}
}

// TestRunTestPhaseStateValidation tests that phase states are valid after status updates.
func TestRunTestPhaseStateValidation(t *testing.T) {
	t.Parallel()

	validPhases := map[string]bool{
		"idle":             true,
		"finding_server":   true,
		"testing_latency":  true,
		"testing_download": true,
		"testing_upload":   true,
		"complete":         true,
	}

	tests := []struct {
		name     string
		phase    string
		progress float64
		valid    bool
	}{
		{"idle phase", "idle", 0, true},
		{"finding server phase", "finding_server", 10, true},
		{"testing latency phase", "testing_latency", 20, true},
		{"testing download phase", "testing_download", 40, true},
		{"testing upload phase", "testing_upload", 70, true},
		{"complete phase", "complete", 100, true},
		{"unknown phase", "unknown_phase", 50, false},
		{"empty phase", "", 0, false},
		{"numeric phase", "123", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			tester.SetStatus(tt.phase, tt.progress)
			status := tester.GetStatus()

			if status.Phase != tt.phase {
				t.Errorf("Phase: got %q, want %q", status.Phase, tt.phase)
			}
			if status.Progress != tt.progress {
				t.Errorf("Progress: got %v, want %v", status.Progress, tt.progress)
			}
			if validPhases[status.Phase] != tt.valid {
				t.Errorf("Phase validity: got %v, want %v", validPhases[status.Phase], tt.valid)
			}
		})
	}
}

// TestStatusAndSpeedsCoordination tests coordinated updates of status and speeds.
func TestStatusAndSpeedsCoordination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		phase    string
		progress float64
		download float64
		upload   float64
	}{
		{"idle with zero speeds", "idle", 0, 0, 0},
		{"download phase with download speed", "testing_download", 40, 500.0, 0},
		{"upload phase with both speeds", "testing_upload", 70, 945.0, 500.0},
		{"complete with final speeds", "complete", 100, 945.0, 880.0},
		{"high speed fiber", "testing_download", 50, 10000.0, 0},
		{"fractional speeds", "testing_download", 45, 123.456, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			tester.SetStatus(tt.phase, tt.progress)
			tester.SetCurrentSpeeds(tt.download, tt.upload)

			status := tester.GetStatus()
			if status.Phase != tt.phase {
				t.Errorf("Phase: got %q, want %q", status.Phase, tt.phase)
			}
			if status.Progress != tt.progress {
				t.Errorf("Progress: got %v, want %v", status.Progress, tt.progress)
			}
			if status.CurrentDownload != tt.download {
				t.Errorf("CurrentDownload: got %v, want %v", status.CurrentDownload, tt.download)
			}
			if status.CurrentUpload != tt.upload {
				t.Errorf("CurrentUpload: got %v, want %v", status.CurrentUpload, tt.upload)
			}
		})
	}
}

// TestBuildTestResultFromParamsComprehensive tests result building with comprehensive parameter combinations.
func TestBuildTestResultFromParamsComprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		dlSpeed      float64
		ulSpeed      float64
		latencyMs    int64
		serverName   string
		sponsor      string
		country      string
		host         string
		distance     float64
		durationSecs int
	}{
		{
			name:         "typical residential fiber",
			dlSpeed:      945.67,
			ulSpeed:      923.45,
			latencyMs:    5,
			serverName:   "Fiber ISP Speed Test",
			sponsor:      "Fiber Provider",
			country:      "USA",
			host:         "speedtest.fiber.example.com:8080",
			distance:     12.5,
			durationSecs: 15,
		},
		{
			name:         "typical cable connection",
			dlSpeed:      250.0,
			ulSpeed:      25.0,
			latencyMs:    15,
			serverName:   "Cable ISP",
			sponsor:      "Cable Company",
			country:      "USA",
			host:         "speedtest.cable.example.com:8080",
			distance:     25.0,
			durationSecs: 20,
		},
		{
			name:         "slow DSL connection",
			dlSpeed:      10.5,
			ulSpeed:      1.5,
			latencyMs:    50,
			serverName:   "DSL Provider",
			sponsor:      "DSL Telecom",
			country:      "Germany",
			host:         "speedtest.dsl.example.de:8080",
			distance:     100.0,
			durationSecs: 30,
		},
		{
			name:         "mobile 5G connection",
			dlSpeed:      750.0,
			ulSpeed:      100.0,
			latencyMs:    8,
			serverName:   "Mobile Carrier",
			sponsor:      "Mobile Provider",
			country:      "Japan",
			host:         "speedtest.mobile.example.jp:8080",
			distance:     5.0,
			durationSecs: 12,
		},
		{
			name:         "satellite connection",
			dlSpeed:      50.0,
			ulSpeed:      10.0,
			latencyMs:    600,
			serverName:   "Satellite ISP",
			sponsor:      "Satellite Provider",
			country:      "Australia",
			host:         "speedtest.satellite.example.au:8080",
			distance:     5000.0,
			durationSecs: 45,
		},
		{
			name:         "10 gigabit datacenter",
			dlSpeed:      10000.0,
			ulSpeed:      10000.0,
			latencyMs:    1,
			serverName:   "10Gbps Server",
			sponsor:      "Datacenter Inc",
			country:      "Netherlands",
			host:         "speedtest.datacenter.nl:8080",
			distance:     0.5,
			durationSecs: 10,
		},
		{
			name:         "zero values",
			dlSpeed:      0,
			ulSpeed:      0,
			latencyMs:    0,
			serverName:   "",
			sponsor:      "",
			country:      "",
			host:         "",
			distance:     0,
			durationSecs: 1,
		},
		{
			name:         "unicode strings",
			dlSpeed:      100.0,
			ulSpeed:      50.0,
			latencyMs:    10,
			serverName:   "テストサーバー",
			sponsor:      "プロバイダー",
			country:      "日本",
			host:         "test.example.jp:8080",
			distance:     1000.0,
			durationSecs: 15,
		},
		{
			name:         "special characters",
			dlSpeed:      100.0,
			ulSpeed:      50.0,
			latencyMs:    10,
			serverName:   "Server <test> & 'special'",
			sponsor:      "Sponsor/Company",
			country:      "Country (Region)",
			host:         "host.example.com:8080",
			distance:     50.0,
			durationSecs: 15,
		},
		{
			name:         "very long strings",
			dlSpeed:      100.0,
			ulSpeed:      50.0,
			latencyMs:    10,
			serverName:   "This is a very long server name that might be used in some edge cases for testing purposes",
			sponsor:      "This is a very long sponsor name that might be used in some edge cases",
			country:      "United States of America",
			host:         "very-long-hostname.subdomain.example.com:8080",
			distance:     2500.0,
			durationSecs: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			startTime := time.Now().Add(-time.Duration(tt.durationSecs) * time.Second)

			result := tester.BuildTestResultFromParams(
				tt.dlSpeed,
				tt.ulSpeed,
				time.Duration(tt.latencyMs)*time.Millisecond,
				tt.serverName,
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
			if result.Latency != float64(tt.latencyMs) {
				t.Errorf("Latency: got %v, want %v", result.Latency, float64(tt.latencyMs))
			}
			if result.Server != tt.serverName {
				t.Errorf("Server: got %q, want %q", result.Server, tt.serverName)
			}

			expectedLocation := tt.sponsor + ", " + tt.country
			if result.Location != expectedLocation {
				t.Errorf("Location: got %q, want %q", result.Location, expectedLocation)
			}

			if result.Host != tt.host {
				t.Errorf("Host: got %q, want %q", result.Host, tt.host)
			}
			if result.Distance != tt.distance {
				t.Errorf("Distance: got %v, want %v", result.Distance, tt.distance)
			}

			// Allow 1 second tolerance for duration
			minDuration := float64(tt.durationSecs) - 0.5
			maxDuration := float64(tt.durationSecs) + 0.5
			if result.TestDuration < minDuration || result.TestDuration > maxDuration {
				t.Errorf("TestDuration: got %v, want between %v and %v",
					result.TestDuration, minDuration, maxDuration)
			}

			// Timestamp should be recent
			if time.Since(result.Timestamp) > time.Second {
				t.Errorf("Timestamp too old: %v", result.Timestamp)
			}
		})
	}
}

// TestResultValidation tests that Result struct values are stored and retrieved correctly.
func TestResultValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result speedtest.Result
	}{
		{
			name: "complete result",
			result: speedtest.Result{
				Download:     945.67,
				Upload:       923.45,
				Latency:      5.0,
				Server:       "Test Server",
				Location:     "Test Location, USA",
				Host:         "test.example.com:8080",
				Distance:     50.0,
				Timestamp:    time.Now(),
				TestDuration: 15.0,
			},
		},
		{
			name: "zero result",
			result: speedtest.Result{
				Download:     0,
				Upload:       0,
				Latency:      0,
				Server:       "",
				Location:     "",
				Host:         "",
				Distance:     0,
				Timestamp:    time.Time{},
				TestDuration: 0,
			},
		},
		{
			name: "high values",
			result: speedtest.Result{
				Download:     100000.0,
				Upload:       100000.0,
				Latency:      1000.0,
				Server:       "Ultra High Speed Server",
				Location:     "Data Center, Netherlands",
				Host:         "ultra.speed.example.nl:8080",
				Distance:     10000.0,
				Timestamp:    time.Now(),
				TestDuration: 60.0,
			},
		},
		{
			name: "fractional values",
			result: speedtest.Result{
				Download:     123.456789,
				Upload:       78.901234,
				Latency:      5.5,
				Server:       "Precision Server",
				Location:     "Lab, USA",
				Host:         "precision.example.com:8080",
				Distance:     12.345,
				Timestamp:    time.Now(),
				TestDuration: 15.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			tester.SetLastResult(&tt.result)

			got := tester.GetLastResult()
			if got == nil {
				t.Fatal("expected non-nil result")
			}

			if got.Download != tt.result.Download {
				t.Errorf("Download: got %v, want %v", got.Download, tt.result.Download)
			}
			if got.Upload != tt.result.Upload {
				t.Errorf("Upload: got %v, want %v", got.Upload, tt.result.Upload)
			}
			if got.Latency != tt.result.Latency {
				t.Errorf("Latency: got %v, want %v", got.Latency, tt.result.Latency)
			}
			if got.Server != tt.result.Server {
				t.Errorf("Server: got %q, want %q", got.Server, tt.result.Server)
			}
			if got.Location != tt.result.Location {
				t.Errorf("Location: got %q, want %q", got.Location, tt.result.Location)
			}
			if got.Host != tt.result.Host {
				t.Errorf("Host: got %q, want %q", got.Host, tt.result.Host)
			}
			if got.Distance != tt.result.Distance {
				t.Errorf("Distance: got %v, want %v", got.Distance, tt.result.Distance)
			}
			if got.TestDuration != tt.result.TestDuration {
				t.Errorf("TestDuration: got %v, want %v", got.TestDuration, tt.result.TestDuration)
			}
		})
	}
}

// TestStatusValidation tests that Status struct values are stored and retrieved correctly.
func TestStatusValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status speedtest.Status
	}{
		{
			name: "idle status",
			status: speedtest.Status{
				Running:         false,
				Phase:           "idle",
				Progress:        0,
				CurrentDownload: 0,
				CurrentUpload:   0,
			},
		},
		{
			name: "finding server status",
			status: speedtest.Status{
				Running:         true,
				Phase:           "finding_server",
				Progress:        10,
				CurrentDownload: 0,
				CurrentUpload:   0,
			},
		},
		{
			name: "download in progress",
			status: speedtest.Status{
				Running:         true,
				Phase:           "testing_download",
				Progress:        50,
				CurrentDownload: 500.0,
				CurrentUpload:   0,
			},
		},
		{
			name: "upload in progress",
			status: speedtest.Status{
				Running:         true,
				Phase:           "testing_upload",
				Progress:        80,
				CurrentDownload: 945.0,
				CurrentUpload:   500.0,
			},
		},
		{
			name: "complete status",
			status: speedtest.Status{
				Running:         true,
				Phase:           "complete",
				Progress:        100,
				CurrentDownload: 945.0,
				CurrentUpload:   880.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			tester.SetRunning(tt.status.Running)
			tester.SetStatus(tt.status.Phase, tt.status.Progress)
			tester.SetCurrentSpeeds(tt.status.CurrentDownload, tt.status.CurrentUpload)

			got := tester.GetStatus()
			if got.Running != tt.status.Running {
				t.Errorf("Running: got %v, want %v", got.Running, tt.status.Running)
			}
			if got.Phase != tt.status.Phase {
				t.Errorf("Phase: got %q, want %q", got.Phase, tt.status.Phase)
			}
			if got.Progress != tt.status.Progress {
				t.Errorf("Progress: got %v, want %v", got.Progress, tt.status.Progress)
			}
			if got.CurrentDownload != tt.status.CurrentDownload {
				t.Errorf("CurrentDownload: got %v, want %v", got.CurrentDownload, tt.status.CurrentDownload)
			}
			if got.CurrentUpload != tt.status.CurrentUpload {
				t.Errorf("CurrentUpload: got %v, want %v", got.CurrentUpload, tt.status.CurrentUpload)
			}
		})
	}
}

// TestTesterIsolation tests that multiple tester instances are completely isolated.
func TestTesterIsolation(t *testing.T) {
	t.Parallel()

	const numTesters = 10

	testers := make([]*speedtest.Tester, numTesters)
	for i := range testers {
		if i%2 == 0 {
			testers[i] = speedtest.NewTester()
		} else {
			testers[i] = speedtest.NewTesterWithConfig("server-" + string(rune('0'+i)))
		}
	}

	// Set different states for each tester
	for i, tester := range testers {
		tester.SetStatus("phase"+string(rune('0'+i)), float64(i*10))
		tester.SetRunning(i%2 == 0)
		tester.SetCurrentSpeeds(float64(i*100), float64(i*50))
		tester.SetLastResult(&speedtest.Result{
			Download: float64((i + 1) * 100),
			Upload:   float64((i + 1) * 50),
		})
	}

	// Verify each tester has its own isolated state
	for i, tester := range testers {
		status := tester.GetStatus()
		expectedPhase := "phase" + string(rune('0'+i))
		if status.Phase != expectedPhase {
			t.Errorf("tester %d: Phase got %q, want %q", i, status.Phase, expectedPhase)
		}
		if status.Progress != float64(i*10) {
			t.Errorf("tester %d: Progress got %v, want %v", i, status.Progress, float64(i*10))
		}
		if status.Running != (i%2 == 0) {
			t.Errorf("tester %d: Running got %v, want %v", i, status.Running, i%2 == 0)
		}
		if status.CurrentDownload != float64(i*100) {
			t.Errorf("tester %d: CurrentDownload got %v, want %v", i, status.CurrentDownload, float64(i*100))
		}
		if status.CurrentUpload != float64(i*50) {
			t.Errorf("tester %d: CurrentUpload got %v, want %v", i, status.CurrentUpload, float64(i*50))
		}

		result := tester.GetLastResult()
		if result == nil {
			t.Errorf("tester %d: expected non-nil result", i)
			continue
		}
		if result.Download != float64((i+1)*100) {
			t.Errorf("tester %d: Download got %v, want %v", i, result.Download, float64((i+1)*100))
		}
	}
}

// TestConstantsIntegrity tests that constants have expected values and relationships.
func TestConstantsIntegrity(t *testing.T) {
	t.Parallel()

	// Test progress constants are in valid range [0, 100]
	progressConstants := []struct {
		name  string
		value int
	}{
		{"ProgressFindingServer", speedtest.ProgressFindingServer},
		{"ProgressTestingLatency", speedtest.ProgressTestingLatency},
		{"ProgressTestingDownload", speedtest.ProgressTestingDownload},
		{"ProgressTestingUpload", speedtest.ProgressTestingUpload},
		{"ProgressComplete", speedtest.ProgressComplete},
	}

	for _, c := range progressConstants {
		if c.value < 0 || c.value > 100 {
			t.Errorf("%s = %d: should be in range [0, 100]", c.name, c.value)
		}
	}

	// Test progress constants are in ascending order
	for i := 1; i < len(progressConstants); i++ {
		if progressConstants[i].value <= progressConstants[i-1].value {
			t.Errorf("%s (%d) should be greater than %s (%d)",
				progressConstants[i].name, progressConstants[i].value,
				progressConstants[i-1].name, progressConstants[i-1].value)
		}
	}

	// Test timing constants are positive
	if speedtest.SpeedPollInterval <= 0 {
		t.Errorf("SpeedPollInterval should be positive, got %v", speedtest.SpeedPollInterval)
	}
	if speedtest.IdleResetDelay <= 0 {
		t.Errorf("IdleResetDelay should be positive, got %v", speedtest.IdleResetDelay)
	}

	// Test BytesToMbps conversion factor
	// 1 Mbps = 1,000,000 bits/sec = 125,000 bytes/sec
	if speedtest.BytesToMbps != 125000.0 {
		t.Errorf("BytesToMbps should be 125000.0, got %v", speedtest.BytesToMbps)
	}
}

// TestBytesToMbpsConversions tests various byte to Mbps conversions.
func TestBytesToMbpsConversions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		bytesPerSec  float64
		expectedMbps float64
		tolerance    float64
	}{
		{"1 Mbps", 125000, 1.0, 0.001},
		{"10 Mbps", 1250000, 10.0, 0.001},
		{"100 Mbps", 12500000, 100.0, 0.001},
		{"1 Gbps", 125000000, 1000.0, 0.001},
		{"10 Gbps", 1250000000, 10000.0, 0.001},
		{"500 Kbps", 62500, 0.5, 0.001},
		{"zero", 0, 0, 0},
		{"very small", 125, 0.001, 0.0001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mbps := tt.bytesPerSec / speedtest.BytesToMbps
			diff := mbps - tt.expectedMbps
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("got %.6f Mbps, want %.6f Mbps (tolerance %.6f)",
					mbps, tt.expectedMbps, tt.tolerance)
			}
		})
	}
}

// TestRunTestWithNilContext tests RunTest with nil context.
func TestRunTestWithNilContext(t *testing.T) {
	tester := speedtest.NewTester()

	// Set running to test the early return
	tester.SetRunning(true)

	// Should still check running state even with nil context
	result, err := tester.RunTest(nil)
	if err == nil {
		t.Error("expected error when test already in progress")
	}
	if err != nil && err.Error() != "test already in progress" {
		t.Errorf("expected 'test already in progress' error, got %q", err.Error())
	}
	if result != nil {
		t.Error("expected nil result when error occurs")
	}
}

// TestSimulatedTestWorkflow simulates a complete test workflow without network.
func TestSimulatedTestWorkflow(t *testing.T) {
	tester := speedtest.NewTester()

	// Initial state
	status := tester.GetStatus()
	if status.Running {
		t.Error("should not be running initially")
	}
	if status.Phase != "idle" {
		t.Errorf("initial phase should be 'idle', got %q", status.Phase)
	}

	// Simulate test start
	tester.SetRunning(true)

	// Phase 1: Finding server
	tester.SetStatus("finding_server", float64(speedtest.ProgressFindingServer))
	status = tester.GetStatus()
	if !status.Running {
		t.Error("should be running")
	}
	if status.Phase != "finding_server" {
		t.Errorf("phase should be 'finding_server', got %q", status.Phase)
	}

	// Phase 2: Testing latency
	tester.SetStatus("testing_latency", float64(speedtest.ProgressTestingLatency))
	status = tester.GetStatus()
	if status.Phase != "testing_latency" {
		t.Errorf("phase should be 'testing_latency', got %q", status.Phase)
	}

	// Phase 3: Testing download with speed updates
	tester.SetStatus("testing_download", float64(speedtest.ProgressTestingDownload))
	downloadSpeeds := []float64{0, 100, 300, 600, 850, 945}
	for _, speed := range downloadSpeeds {
		tester.SetCurrentSpeeds(speed, 0)
		status = tester.GetStatus()
		if status.CurrentDownload != speed {
			t.Errorf("CurrentDownload: got %v, want %v", status.CurrentDownload, speed)
		}
	}
	finalDownload := 945.0

	// Phase 4: Testing upload with speed updates
	tester.SetStatus("testing_upload", float64(speedtest.ProgressTestingUpload))
	uploadSpeeds := []float64{0, 50, 150, 350, 600, 850}
	for _, speed := range uploadSpeeds {
		tester.SetCurrentSpeeds(finalDownload, speed)
		status = tester.GetStatus()
		if status.CurrentDownload != finalDownload {
			t.Errorf("CurrentDownload should be %v, got %v", finalDownload, status.CurrentDownload)
		}
		if status.CurrentUpload != speed {
			t.Errorf("CurrentUpload: got %v, want %v", status.CurrentUpload, speed)
		}
	}
	finalUpload := 850.0

	// Phase 5: Complete
	tester.SetStatus("complete", float64(speedtest.ProgressComplete))
	status = tester.GetStatus()
	if status.Phase != "complete" {
		t.Errorf("phase should be 'complete', got %q", status.Phase)
	}
	if status.Progress != 100 {
		t.Errorf("progress should be 100, got %v", status.Progress)
	}

	// Store result
	startTime := time.Now().Add(-20 * time.Second)
	result := tester.BuildTestResultFromParams(
		finalDownload,
		finalUpload,
		10*time.Millisecond,
		"Test Server",
		"Test Provider",
		"USA",
		"test.example.com:8080",
		50.0,
		startTime,
	)
	tester.SetLastResult(result)

	// Verify result stored
	lastResult := tester.GetLastResult()
	if lastResult == nil {
		t.Fatal("expected non-nil result")
	}
	if lastResult.Download != finalDownload {
		t.Errorf("result Download: got %v, want %v", lastResult.Download, finalDownload)
	}
	if lastResult.Upload != finalUpload {
		t.Errorf("result Upload: got %v, want %v", lastResult.Upload, finalUpload)
	}

	// Reset to idle
	tester.SetRunning(false)
	tester.SetStatus("idle", 0)
	tester.SetCurrentSpeeds(0, 0)

	status = tester.GetStatus()
	if status.Running {
		t.Error("should not be running after reset")
	}
	if status.Phase != "idle" {
		t.Errorf("phase should be 'idle', got %q", status.Phase)
	}
	if status.CurrentDownload != 0 {
		t.Errorf("CurrentDownload should be 0, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("CurrentUpload should be 0, got %v", status.CurrentUpload)
	}

	// Result should still be available after reset
	lastResult = tester.GetLastResult()
	if lastResult == nil {
		t.Error("result should persist after reset")
	}
}

// TestConcurrentResultUpdates tests concurrent result updates.
func TestConcurrentResultUpdates(t *testing.T) {
	t.Parallel()

	tester := speedtest.NewTester()
	var wg sync.WaitGroup

	const numWriters = 10
	const iterations = 100

	wg.Add(numWriters)
	for i := range numWriters {
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				result := &speedtest.Result{
					Download:     float64(id*100 + j),
					Upload:       float64(j * 10),
					Latency:      float64(id),
					Server:       "Server",
					Location:     "Location",
					Host:         "host.example.com",
					Distance:     float64(id),
					Timestamp:    time.Now(),
					TestDuration: float64(j),
				}
				tester.SetLastResult(result)
			}
		}(i)
	}

	// Concurrent readers
	wg.Add(numWriters)
	for range numWriters {
		go func() {
			defer wg.Done()
			for range iterations {
				result := tester.GetLastResult()
				if result != nil {
					_ = result.Download
					_ = result.Upload
					_ = result.Server
				}
			}
		}()
	}

	wg.Wait()

	// Final result should exist
	result := tester.GetLastResult()
	if result == nil {
		t.Error("expected non-nil result after concurrent writes")
	}
}

// TestEmptyAndNilHandling tests handling of empty and nil values.
func TestEmptyAndNilHandling(t *testing.T) {
	t.Parallel()

	tester := speedtest.NewTester()

	// Set nil result
	tester.SetLastResult(nil)
	if tester.GetLastResult() != nil {
		t.Error("expected nil result after setting nil")
	}

	// Set empty server ID
	tester.SetServerID("")
	if tester.TesterServerID() != "" {
		t.Errorf("expected empty server ID, got %q", tester.TesterServerID())
	}

	// Set empty phase
	tester.SetStatus("", 0)
	status := tester.GetStatus()
	if status.Phase != "" {
		t.Errorf("expected empty phase, got %q", status.Phase)
	}

	// Set zero speeds
	tester.SetCurrentSpeeds(0, 0)
	status = tester.GetStatus()
	if status.CurrentDownload != 0 {
		t.Errorf("expected zero CurrentDownload, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("expected zero CurrentUpload, got %v", status.CurrentUpload)
	}
}

// TestProgressBoundaryValues tests progress values at boundaries.
func TestProgressBoundaryValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		progress float64
	}{
		{"zero", 0},
		{"minimum positive", 0.0001},
		{"one percent", 1},
		{"ten percent", 10},
		{"fifty percent", 50},
		{"ninety nine percent", 99},
		{"hundred percent", 100},
		{"over hundred", 150}, // Should still work, no validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			tester.SetStatus("testing", tt.progress)
			status := tester.GetStatus()
			if status.Progress != tt.progress {
				t.Errorf("Progress: got %v, want %v", status.Progress, tt.progress)
			}
		})
	}
}

// TestSpeedBoundaryValues tests speed values at boundaries.
func TestSpeedBoundaryValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		download float64
		upload   float64
	}{
		{"zero speeds", 0, 0},
		{"tiny speeds", 0.001, 0.001},
		{"typical speeds", 100, 50},
		{"gigabit speeds", 1000, 1000},
		{"10 gigabit speeds", 10000, 10000},
		{"100 gigabit speeds", 100000, 100000},
		{"asymmetric download", 1000, 10},
		{"asymmetric upload", 10, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			tester.SetCurrentSpeeds(tt.download, tt.upload)
			status := tester.GetStatus()
			if status.CurrentDownload != tt.download {
				t.Errorf("CurrentDownload: got %v, want %v", status.CurrentDownload, tt.download)
			}
			if status.CurrentUpload != tt.upload {
				t.Errorf("CurrentUpload: got %v, want %v", status.CurrentUpload, tt.upload)
			}
		})
	}
}
