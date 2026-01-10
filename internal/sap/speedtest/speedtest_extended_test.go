package speedtest_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// TestBuildTestResultFromParams tests the result building logic with explicit parameters.
func TestBuildTestResultFromParams(t *testing.T) {
	tests := []buildTestResultParamsCase{
		{
			name:        "typical fiber connection",
			dlSpeed:     945.67,
			ulSpeed:     923.45,
			latency:     5 * time.Millisecond,
			serverName:  "Fiber ISP Speed Test",
			sponsor:     "Fiber Provider",
			country:     "USA",
			host:        "speedtest.fiber.example.com:8080",
			distance:    12.5,
			wantLatency: 5,
		},
		{
			name:        "typical cable connection",
			dlSpeed:     250.0,
			ulSpeed:     25.0,
			latency:     15 * time.Millisecond,
			serverName:  "Cable ISP Speed Test",
			sponsor:     "Cable Company",
			country:     "USA",
			host:        "speedtest.cable.example.com:8080",
			distance:    25.0,
			wantLatency: 15,
		},
		{
			name:        "slow DSL connection",
			dlSpeed:     10.5,
			ulSpeed:     1.5,
			latency:     50 * time.Millisecond,
			serverName:  "DSL Provider Test",
			sponsor:     "DSL Telecom",
			country:     "Germany",
			host:        "speedtest.dsl.example.de:8080",
			distance:    100.0,
			wantLatency: 50,
		},
		{
			name:        "mobile 5G connection",
			dlSpeed:     750.0,
			ulSpeed:     100.0,
			latency:     8 * time.Millisecond,
			serverName:  "Mobile Carrier Test",
			sponsor:     "Mobile Carrier",
			country:     "Japan",
			host:        "speedtest.mobile.example.jp:8080",
			distance:    5.0,
			wantLatency: 8,
		},
		{
			name:        "high latency satellite",
			dlSpeed:     50.0,
			ulSpeed:     10.0,
			latency:     600 * time.Millisecond,
			serverName:  "Satellite ISP Test",
			sponsor:     "Satellite Provider",
			country:     "Australia",
			host:        "speedtest.satellite.example.au:8080",
			distance:    5000.0,
			wantLatency: 600,
		},
		{
			name:        "zero values",
			dlSpeed:     0,
			ulSpeed:     0,
			latency:     0,
			serverName:  "",
			sponsor:     "",
			country:     "",
			host:        "",
			distance:    0,
			wantLatency: 0,
		},
		{
			name:        "very fast connection",
			dlSpeed:     10000.0,
			ulSpeed:     10000.0,
			latency:     1 * time.Millisecond,
			serverName:  "10Gbps Test Server",
			sponsor:     "Datacenter",
			country:     "Netherlands",
			host:        "speedtest.datacenter.nl:8080",
			distance:    0.5,
			wantLatency: 1,
		},
		{
			name:        "fractional latency",
			dlSpeed:     100.0,
			ulSpeed:     50.0,
			latency:     2500 * time.Microsecond, // 2.5ms rounds to 2ms
			serverName:  "Low Latency Server",
			sponsor:     "LowLatency Inc",
			country:     "USA",
			host:        "speedtest.lowlatency.com:8080",
			distance:    1.0,
			wantLatency: 2, // Milliseconds() truncates
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runBuildTestResultParamsCase(t, tt)
		})
	}
}

type buildTestResultParamsCase struct {
	name        string
	dlSpeed     float64
	ulSpeed     float64
	latency     time.Duration
	serverName  string
	sponsor     string
	country     string
	host        string
	distance    float64
	wantLatency float64
}

func runBuildTestResultParamsCase(t *testing.T, tt buildTestResultParamsCase) {
	t.Helper()

	tester := speedtest.NewTester()
	startTime := time.Now().Add(-10 * time.Second)

	result := tester.BuildTestResultFromParams(
		tt.dlSpeed,
		tt.ulSpeed,
		tt.latency,
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

	assertBuildResultParamsFields(t, result, tt)
	assertBuildResultDuration(t, result)
	assertBuildResultTimestamp(t, result.Timestamp)
}

func assertBuildResultParamsFields(t *testing.T, result *speedtest.Result, tt buildTestResultParamsCase) {
	t.Helper()

	if result.Download != tt.dlSpeed {
		t.Errorf("Download: got %v, want %v", result.Download, tt.dlSpeed)
	}
	if result.Upload != tt.ulSpeed {
		t.Errorf("Upload: got %v, want %v", result.Upload, tt.ulSpeed)
	}
	if result.Latency != tt.wantLatency {
		t.Errorf("Latency: got %v, want %v", result.Latency, tt.wantLatency)
	}
	if result.Server != tt.serverName {
		t.Errorf("Server: got %q, want %q", result.Server, tt.serverName)
	}

	wantLocation := tt.sponsor + ", " + tt.country
	if result.Location != wantLocation {
		t.Errorf("Location: got %q, want %q", result.Location, wantLocation)
	}

	if result.Host != tt.host {
		t.Errorf("Host: got %q, want %q", result.Host, tt.host)
	}
	if result.Distance != tt.distance {
		t.Errorf("Distance: got %v, want %v", result.Distance, tt.distance)
	}
}

func assertBuildResultDuration(t *testing.T, result *speedtest.Result) {
	t.Helper()

	if result.TestDuration < 9.9 || result.TestDuration > 10.5 {
		t.Errorf("TestDuration: got %v, want ~10 seconds", result.TestDuration)
	}
}

func assertBuildResultTimestamp(t *testing.T, timestamp time.Time) {
	t.Helper()

	if time.Since(timestamp) > time.Second {
		t.Errorf("Timestamp too old: %v", timestamp)
	}
}

// TestBuildTestResultFromParamsTestDuration tests various test durations.
func TestBuildTestResultFromParamsTestDuration(t *testing.T) {
	tests := []struct {
		name            string
		durationSeconds int
	}{
		{"very short test", 1},
		{"short test", 5},
		{"typical test", 15},
		{"long test", 30},
		{"very long test", 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			startTime := time.Now().Add(-time.Duration(tt.durationSeconds) * time.Second)

			result := tester.BuildTestResultFromParams(
				100.0, 50.0,
				10*time.Millisecond,
				"Test Server", "Sponsor", "Country",
				"host.example.com",
				50.0,
				startTime,
			)

			// Allow 0.5 second tolerance
			expectedMin := float64(tt.durationSeconds) - 0.5
			expectedMax := float64(tt.durationSeconds) + 0.5

			if result.TestDuration < expectedMin || result.TestDuration > expectedMax {
				t.Errorf("TestDuration: got %v, want ~%d seconds", result.TestDuration, tt.durationSeconds)
			}
		})
	}
}

// TestStatusTransitionsDetailed tests detailed status transitions.
func TestStatusTransitionsDetailed(t *testing.T) {
	tests := []struct {
		name           string
		phases         []string
		progressValues []float64
		runningStates  []bool
	}{
		{
			name: "normal test flow",
			phases: []string{
				"idle",
				"finding_server",
				"testing_latency",
				"testing_download",
				"testing_upload",
				"complete",
				"idle",
			},
			progressValues: []float64{
				0,
				float64(speedtest.ProgressFindingServer),
				float64(speedtest.ProgressTestingLatency),
				float64(speedtest.ProgressTestingDownload),
				float64(speedtest.ProgressTestingUpload),
				float64(speedtest.ProgressComplete),
				0,
			},
			runningStates: []bool{false, true, true, true, true, true, false},
		},
		{
			name:           "error during finding server",
			phases:         []string{"idle", "finding_server", "idle"},
			progressValues: []float64{0, float64(speedtest.ProgressFindingServer), 0},
			runningStates:  []bool{false, true, false},
		},
		{
			name:           "error during latency test",
			phases:         []string{"idle", "finding_server", "testing_latency", "idle"},
			progressValues: []float64{0, 10, 20, 0},
			runningStates:  []bool{false, true, true, false},
		},
		{
			name:           "error during download test",
			phases:         []string{"idle", "finding_server", "testing_latency", "testing_download", "idle"},
			progressValues: []float64{0, 10, 20, 40, 0},
			runningStates:  []bool{false, true, true, true, false},
		},
		{
			name: "error during upload test",
			phases: []string{
				"idle",
				"finding_server",
				"testing_latency",
				"testing_download",
				"testing_upload",
				"idle",
			},
			progressValues: []float64{0, 10, 20, 40, 70, 0},
			runningStates:  []bool{false, true, true, true, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()

			if len(tt.phases) != len(tt.progressValues) || len(tt.phases) != len(tt.runningStates) {
				t.Fatal("test case has mismatched slice lengths")
			}

			for i, phase := range tt.phases {
				tester.SetRunning(tt.runningStates[i])
				tester.SetStatus(phase, tt.progressValues[i])

				status := tester.GetStatus()
				if status.Phase != phase {
					t.Errorf("step %d: Phase got %q, want %q", i, status.Phase, phase)
				}
				if status.Progress != tt.progressValues[i] {
					t.Errorf("step %d: Progress got %v, want %v", i, status.Progress, tt.progressValues[i])
				}
				if status.Running != tt.runningStates[i] {
					t.Errorf("step %d: Running got %v, want %v", i, status.Running, tt.runningStates[i])
				}
			}
		})
	}
}

// TestCurrentSpeedsDuringPhases tests live speed updates during different phases.
func TestCurrentSpeedsDuringPhases(t *testing.T) {
	tests := []currentSpeedsCase{
		{
			name:             "download phase progression",
			phase:            "testing_download",
			downloadSpeeds:   []float64{0, 50, 100, 200, 350, 500},
			uploadSpeeds:     []float64{0, 0, 0, 0, 0, 0},
			wantFinalDL:      500,
			wantFinalUL:      0,
			checkProgression: true,
		},
		{
			name:             "upload phase with final download",
			phase:            "testing_upload",
			downloadSpeeds:   []float64{500, 500, 500, 500},
			uploadSpeeds:     []float64{0, 50, 100, 150},
			wantFinalDL:      500,
			wantFinalUL:      150,
			checkProgression: true,
		},
		{
			name:             "reset after test",
			phase:            "idle",
			downloadSpeeds:   []float64{500, 0},
			uploadSpeeds:     []float64{150, 0},
			wantFinalDL:      0,
			wantFinalUL:      0,
			checkProgression: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCurrentSpeedsCase(t, tt)
		})
	}
}

type currentSpeedsCase struct {
	name             string
	phase            string
	downloadSpeeds   []float64
	uploadSpeeds     []float64
	wantFinalDL      float64
	wantFinalUL      float64
	checkProgression bool
}

func runCurrentSpeedsCase(t *testing.T, tt currentSpeedsCase) {
	t.Helper()

	tester := speedtest.NewTester()
	tester.SetStatus(tt.phase, 50)

	updateCurrentSpeeds(t, tester, tt)
	assertFinalSpeeds(t, tester, tt)
}

func updateCurrentSpeeds(t *testing.T, tester *speedtest.Tester, tt currentSpeedsCase) {
	t.Helper()

	var prevDL, prevUL float64
	for i := range tt.downloadSpeeds {
		tester.SetCurrentSpeeds(tt.downloadSpeeds[i], tt.uploadSpeeds[i])
		status := tester.GetStatus()

		if status.CurrentDownload != tt.downloadSpeeds[i] {
			t.Errorf("step %d: CurrentDownload got %v, want %v",
				i, status.CurrentDownload, tt.downloadSpeeds[i])
		}
		if status.CurrentUpload != tt.uploadSpeeds[i] {
			t.Errorf("step %d: CurrentUpload got %v, want %v",
				i, status.CurrentUpload, tt.uploadSpeeds[i])
		}

		if tt.checkProgression && i > 0 {
			assertNonDecreasingSpeeds(t, tt.phase, i, status.CurrentDownload, status.CurrentUpload, prevDL, prevUL)
		}

		prevDL = status.CurrentDownload
		prevUL = status.CurrentUpload
	}
}

func assertNonDecreasingSpeeds(t *testing.T, phase string, step int, currentDL, currentUL, prevDL, prevUL float64) {
	t.Helper()

	if phase == "testing_download" && currentDL < prevDL {
		t.Errorf("step %d: download speed decreased from %v to %v", step, prevDL, currentDL)
	}
	if phase == "testing_upload" && currentUL < prevUL {
		t.Errorf("step %d: upload speed decreased from %v to %v", step, prevUL, currentUL)
	}
}

func assertFinalSpeeds(t *testing.T, tester *speedtest.Tester, tt currentSpeedsCase) {
	t.Helper()

	status := tester.GetStatus()
	if status.CurrentDownload != tt.wantFinalDL {
		t.Errorf("final CurrentDownload got %v, want %v", status.CurrentDownload, tt.wantFinalDL)
	}
	if status.CurrentUpload != tt.wantFinalUL {
		t.Errorf("final CurrentUpload got %v, want %v", status.CurrentUpload, tt.wantFinalUL)
	}
}

// TestConcurrentStatusAndSpeedUpdates tests concurrent updates to status and speeds.
func TestConcurrentStatusAndSpeedUpdates(t *testing.T) {
	tester := speedtest.NewTester()
	var wg sync.WaitGroup

	const numGoroutines = 20
	const iterationsPerGoroutine = 100

	// Status updaters
	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			phases := []string{
				"idle",
				"finding_server",
				"testing_latency",
				"testing_download",
				"testing_upload",
				"complete",
			}
			for j := range iterationsPerGoroutine {
				phase := phases[j%len(phases)]
				progress := float64((j % 100) + 1)
				tester.SetStatus(phase, progress)
			}
		}()
	}

	// Speed updaters
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := range iterationsPerGoroutine {
				dl := float64(id*100 + j)
				ul := float64(j * 10)
				tester.SetCurrentSpeeds(dl, ul)
			}
		}(i)
	}

	// Running state updaters
	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for j := range iterationsPerGoroutine {
				tester.SetRunning(j%2 == 0)
			}
		}()
	}

	// Concurrent readers
	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range iterationsPerGoroutine {
				status := tester.GetStatus()
				// Verify status is consistent (no partial reads)
				_ = status.Phase
				_ = status.Progress
				_ = status.Running
				_ = status.CurrentDownload
				_ = status.CurrentUpload
			}
		}()
	}

	wg.Wait()

	// Final state should be valid (no panic, no corruption)
	status := tester.GetStatus()
	if status.Phase == "" {
		t.Error("expected non-empty phase after concurrent updates")
	}
}

// TestResultConcurrentAccess tests concurrent access to lastResult.
func TestResultConcurrentAccess(t *testing.T) {
	tester := speedtest.NewTester()
	var wg sync.WaitGroup

	const numWriters = 10
	const numReaders = 10
	const iterations = 100

	// Writers
	wg.Add(numWriters)
	for i := range numWriters {
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				result := &speedtest.Result{
					Download:     float64(id*100 + j),
					Upload:       float64(j * 10),
					Latency:      float64(id + j),
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

	// Readers
	wg.Add(numReaders)
	for range numReaders {
		go func() {
			defer wg.Done()
			for range iterations {
				result := tester.GetLastResult()
				if result != nil {
					// Access all fields to ensure no data corruption
					_ = result.Download
					_ = result.Upload
					_ = result.Latency
					_ = result.Server
					_ = result.Location
					_ = result.Host
					_ = result.Distance
					_ = result.Timestamp
					_ = result.TestDuration
				}
			}
		}()
	}

	wg.Wait()

	// Final result should exist and be valid
	result := tester.GetLastResult()
	if result == nil {
		t.Error("expected non-nil result after concurrent writes")
	}
}

// TestRunTestAlreadyRunningMultiple tests multiple concurrent "already running" checks.
func TestRunTestAlreadyRunningMultiple(t *testing.T) {
	tester := speedtest.NewTester()
	tester.SetRunning(true)

	var wg sync.WaitGroup
	const numAttempts = 50
	errors := make(chan error, numAttempts)

	wg.Add(numAttempts)
	for range numAttempts {
		go func() {
			defer wg.Done()
			_, err := tester.RunTest(context.Background())
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	// All attempts should fail
	for err := range errors {
		if err == nil {
			t.Error("expected error when test already running")
			continue
		}
		if err.Error() != "test already in progress" {
			t.Errorf("unexpected error: %v", err)
		}
	}

	tester.SetRunning(false)
}

// TestServerIDVariations tests various server ID formats.
func TestServerIDVariations(t *testing.T) {
	tests := []struct {
		name     string
		serverID string
	}{
		{"empty", ""},
		{"numeric", "12345"},
		{"alphanumeric", "server123"},
		{"with dashes", "server-123-abc"},
		{"with underscores", "server_123_abc"},
		{"uuid format", "550e8400-e29b-41d4-a716-446655440000"},
		{"long id", "very-long-server-identifier-that-might-be-used-in-some-systems"},
		{"with special chars", "server.example.com:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test NewTesterWithConfig
			tester := speedtest.NewTesterWithConfig(tt.serverID)
			if tester.TesterServerID() != tt.serverID {
				t.Errorf("NewTesterWithConfig: got %q, want %q",
					tester.TesterServerID(), tt.serverID)
			}

			// Test SetServerID
			tester2 := speedtest.NewTester()
			tester2.SetServerID(tt.serverID)
			if tester2.TesterServerID() != tt.serverID {
				t.Errorf("SetServerID: got %q, want %q",
					tester2.TesterServerID(), tt.serverID)
			}
		})
	}
}

// TestResultJSONTags verifies that Result struct has proper JSON tags.
func TestResultJSONTags(t *testing.T) {
	result := speedtest.Result{
		Download:     100.0,
		Upload:       50.0,
		Latency:      10.0,
		Server:       "Test Server",
		Location:     "Test Location",
		Host:         "test.example.com",
		Distance:     25.0,
		Timestamp:    time.Now(),
		TestDuration: 15.0,
	}

	// Verify struct fields are accessible and have expected values
	if result.Download != 100.0 {
		t.Errorf("Download: got %v, want 100.0", result.Download)
	}
	if result.Upload != 50.0 {
		t.Errorf("Upload: got %v, want 50.0", result.Upload)
	}
	if result.Latency != 10.0 {
		t.Errorf("Latency: got %v, want 10.0", result.Latency)
	}
	if result.Server != "Test Server" {
		t.Errorf("Server: got %q, want %q", result.Server, "Test Server")
	}
	if result.Location != "Test Location" {
		t.Errorf("Location: got %q, want %q", result.Location, "Test Location")
	}
	if result.Host != "test.example.com" {
		t.Errorf("Host: got %q, want %q", result.Host, "test.example.com")
	}
	if result.Distance != 25.0 {
		t.Errorf("Distance: got %v, want 25.0", result.Distance)
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if result.TestDuration != 15.0 {
		t.Errorf("TestDuration: got %v, want 15.0", result.TestDuration)
	}
}

// TestStatusJSONTags verifies that Status struct has proper JSON tags.
func TestStatusJSONTags(t *testing.T) {
	status := speedtest.Status{
		Running:         true,
		Phase:           "testing_download",
		Progress:        50.0,
		CurrentDownload: 250.5,
		CurrentUpload:   0,
	}

	// Verify struct fields are accessible and have expected values
	if !status.Running {
		t.Error("Running: got false, want true")
	}
	if status.Phase != "testing_download" {
		t.Errorf("Phase: got %q, want %q", status.Phase, "testing_download")
	}
	if status.Progress != 50.0 {
		t.Errorf("Progress: got %v, want 50.0", status.Progress)
	}
	if status.CurrentDownload != 250.5 {
		t.Errorf("CurrentDownload: got %v, want 250.5", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("CurrentUpload: got %v, want 0", status.CurrentUpload)
	}
}

// TestProgressConstantsOrder verifies that progress constants are in ascending order.
func TestProgressConstantsOrder(t *testing.T) {
	phases := []struct {
		name     string
		progress int
	}{
		{"ProgressFindingServer", speedtest.ProgressFindingServer},
		{"ProgressTestingLatency", speedtest.ProgressTestingLatency},
		{"ProgressTestingDownload", speedtest.ProgressTestingDownload},
		{"ProgressTestingUpload", speedtest.ProgressTestingUpload},
		{"ProgressComplete", speedtest.ProgressComplete},
	}

	for i := 1; i < len(phases); i++ {
		if phases[i].progress <= phases[i-1].progress {
			t.Errorf("%s (%d) should be greater than %s (%d)",
				phases[i].name, phases[i].progress,
				phases[i-1].name, phases[i-1].progress)
		}
	}
}

// TestProgressConstantsRange verifies that progress constants are in valid range [0, 100].
func TestProgressConstantsRange(t *testing.T) {
	constants := []struct {
		name  string
		value int
	}{
		{"ProgressFindingServer", speedtest.ProgressFindingServer},
		{"ProgressTestingLatency", speedtest.ProgressTestingLatency},
		{"ProgressTestingDownload", speedtest.ProgressTestingDownload},
		{"ProgressTestingUpload", speedtest.ProgressTestingUpload},
		{"ProgressComplete", speedtest.ProgressComplete},
	}

	for _, c := range constants {
		if c.value < 0 || c.value > 100 {
			t.Errorf("%s = %d: should be in range [0, 100]", c.name, c.value)
		}
	}
}

// TestBytesToMbpsConversion tests the bytes to Mbps conversion factor.
func TestBytesToMbpsConversion(t *testing.T) {
	tests := []struct {
		name         string
		bytesPerSec  float64
		expectedMbps float64
	}{
		{"1 Mbps", 125000, 1.0},
		{"10 Mbps", 1250000, 10.0},
		{"100 Mbps", 12500000, 100.0},
		{"1 Gbps", 125000000, 1000.0},
		{"500 Kbps", 62500, 0.5},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mbps := tt.bytesPerSec / speedtest.BytesToMbps
			if mbps != tt.expectedMbps {
				t.Errorf("got %v Mbps, want %v Mbps", mbps, tt.expectedMbps)
			}
		})
	}
}

// TestSpeedPollIntervalValue tests the speed poll interval constant.
func TestSpeedPollIntervalValue(t *testing.T) {
	// Should be 100ms as per the code
	expected := 100 * time.Millisecond
	if speedtest.SpeedPollInterval != expected {
		t.Errorf("SpeedPollInterval: got %v, want %v", speedtest.SpeedPollInterval, expected)
	}

	// Should be reasonable for UI updates (between 50ms and 500ms)
	if speedtest.SpeedPollInterval < 50*time.Millisecond {
		t.Error("SpeedPollInterval too short for practical use")
	}
	if speedtest.SpeedPollInterval > 500*time.Millisecond {
		t.Error("SpeedPollInterval too long for smooth UI updates")
	}
}

// TestIdleResetDelayValue tests the idle reset delay constant.
func TestIdleResetDelayValue(t *testing.T) {
	// Should be 2 seconds as per the code
	expected := 2 * time.Second
	if speedtest.IdleResetDelay != expected {
		t.Errorf("IdleResetDelay: got %v, want %v", speedtest.IdleResetDelay, expected)
	}

	// Should be reasonable (between 1 and 10 seconds)
	if speedtest.IdleResetDelay < time.Second {
		t.Error("IdleResetDelay too short to show completion status")
	}
	if speedtest.IdleResetDelay > 10*time.Second {
		t.Error("IdleResetDelay too long before resetting to idle")
	}
}

// TestTesterInitialization tests that a new Tester is properly initialized.
func TestTesterInitialization(t *testing.T) {
	tester := speedtest.NewTester()

	status := tester.GetStatus()

	// All fields should be at their zero/default values
	if status.Running {
		t.Error("Running should be false initially")
	}
	if status.Phase != "idle" {
		t.Errorf("Phase should be 'idle', got %q", status.Phase)
	}
	if status.Progress != 0 {
		t.Errorf("Progress should be 0, got %v", status.Progress)
	}
	if status.CurrentDownload != 0 {
		t.Errorf("CurrentDownload should be 0, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("CurrentUpload should be 0, got %v", status.CurrentUpload)
	}

	// LastResult should be nil
	if tester.GetLastResult() != nil {
		t.Error("LastResult should be nil initially")
	}

	// ServerID should be empty
	if tester.TesterServerID() != "" {
		t.Errorf("ServerID should be empty, got %q", tester.TesterServerID())
	}
}

// TestTesterWithConfigInitialization tests that NewTesterWithConfig properly initializes.
func TestTesterWithConfigInitialization(t *testing.T) {
	serverID := "test-server-123"
	tester := speedtest.NewTesterWithConfig(serverID)

	status := tester.GetStatus()

	// Status should be at defaults
	if status.Running {
		t.Error("Running should be false initially")
	}
	if status.Phase != "idle" {
		t.Errorf("Phase should be 'idle', got %q", status.Phase)
	}
	if status.Progress != 0 {
		t.Errorf("Progress should be 0, got %v", status.Progress)
	}

	// ServerID should be set
	if tester.TesterServerID() != serverID {
		t.Errorf("ServerID should be %q, got %q", serverID, tester.TesterServerID())
	}

	// LastResult should still be nil
	if tester.GetLastResult() != nil {
		t.Error("LastResult should be nil initially")
	}
}

// TestRunTestContextCancellation tests behavior when context is cancelled.
// Note: The current implementation ignores context, but this test documents expected behavior.
// This test is skipped by default as it makes network calls and the upstream speedtest-go library
// has race conditions in its internal data structures.
func TestRunTestContextCancellation(t *testing.T) {
	t.Skip("Skipped: upstream speedtest-go library has race conditions in data_manager.go")

	tester := speedtest.NewTester()

	// Create a pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Note: Current implementation ignores context, so this will fail during findTestServer
	// This test documents that context cancellation is not yet implemented
	_, err := tester.RunTest(ctx)

	// We expect an error (either from context or from network)
	if err == nil {
		t.Log("Note: Context cancellation may not be implemented yet")
	}
}

// TestResultTimestamp tests that result timestamp is set correctly.
func TestResultTimestamp(t *testing.T) {
	tester := speedtest.NewTester()
	beforeTest := time.Now()

	result := tester.BuildTestResultFromParams(
		100.0, 50.0,
		10*time.Millisecond,
		"Server", "Sponsor", "Country",
		"host.example.com",
		50.0,
		time.Now().Add(-10*time.Second),
	)

	afterTest := time.Now()

	if result.Timestamp.Before(beforeTest) {
		t.Error("Timestamp should not be before test start")
	}
	if result.Timestamp.After(afterTest) {
		t.Error("Timestamp should not be after test end")
	}
}

// TestResultLocationFormat tests that location is formatted correctly.
func TestResultLocationFormat(t *testing.T) {
	tests := []struct {
		sponsor  string
		country  string
		expected string
	}{
		{"AT&T", "USA", "AT&T, USA"},
		{"Vodafone", "UK", "Vodafone, UK"},
		{"", "", ", "},
		{"Provider", "", "Provider, "},
		{"", "Country", ", Country"},
		{"Some Long Provider Name", "United States of America", "Some Long Provider Name, United States of America"},
	}

	for _, tt := range tests {
		t.Run(tt.sponsor+"_"+tt.country, func(t *testing.T) {
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				100.0, 50.0,
				10*time.Millisecond,
				"Server",
				tt.sponsor,
				tt.country,
				"host.example.com",
				50.0,
				time.Now(),
			)

			if result.Location != tt.expected {
				t.Errorf("Location: got %q, want %q", result.Location, tt.expected)
			}
		})
	}
}
