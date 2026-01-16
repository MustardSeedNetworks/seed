package speedtest_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// TestRunTestGuardBehavior documents the behavior of the RunTest guard that prevents concurrent tests.
func TestRunTestGuardBehavior(t *testing.T) {
	t.Run("guard prevents concurrent execution", func(t *testing.T) {
		tester := speedtest.NewTester()
		tester.SetRunning(true)

		// Multiple calls should all fail
		for i := range 100 {
			_, err := tester.RunTest(context.Background())
			if err == nil {
				t.Fatalf("iteration %d: expected error", i)
			}
			if err.Error() != "test already in progress" {
				t.Fatalf("iteration %d: unexpected error: %v", i, err)
			}
		}
	})

	t.Run("guard allows execution after running cleared", func(t *testing.T) {
		tester := speedtest.NewTester()

		// Initially should check running state
		status := tester.GetStatus()
		if status.Running {
			t.Fatal("should not be running initially")
		}

		// Set running
		tester.SetRunning(true)
		_, err := tester.RunTest(context.Background())
		if err == nil || err.Error() != "test already in progress" {
			t.Fatalf("expected 'test already in progress' error, got %v", err)
		}

		// Clear running - now check would pass (but network call would fail)
		tester.SetRunning(false)
		status = tester.GetStatus()
		if status.Running {
			t.Error("should not be running after SetRunning(false)")
		}
	})
}

// TestStatusPhaseValues documents the valid phase values.
func TestStatusPhaseValues(t *testing.T) {
	validPhases := []struct {
		phase    string
		progress int
	}{
		{"idle", 0},
		{"finding_server", speedtest.ProgressFindingServer},
		{"testing_latency", speedtest.ProgressTestingLatency},
		{"testing_download", speedtest.ProgressTestingDownload},
		{"testing_upload", speedtest.ProgressTestingUpload},
		{"complete", speedtest.ProgressComplete},
	}

	for _, vp := range validPhases {
		t.Run(vp.phase, func(t *testing.T) {
			tester := speedtest.NewTester()
			tester.SetStatus(vp.phase, float64(vp.progress))

			status := tester.GetStatus()
			if status.Phase != vp.phase {
				t.Errorf("phase: got %q, want %q", status.Phase, vp.phase)
			}
			if status.Progress != float64(vp.progress) {
				t.Errorf("progress: got %v, want %v", status.Progress, float64(vp.progress))
			}
		})
	}
}

// TestResultFieldBehavior documents the behavior of Result fields.
func TestResultFieldBehavior(t *testing.T) {
	tests := []struct {
		name   string
		result speedtest.Result
	}{
		{
			name: "complete result",
			result: speedtest.Result{
				Download:     945.67,
				Upload:       923.45,
				Latency:      3.5,
				Server:       "Fast Server",
				Location:     "Provider, Country",
				Host:         "fast.example.com:8080",
				Distance:     25.5,
				Timestamp:    time.Now(),
				TestDuration: 20.5,
			},
		},
		{
			name: "zero values result",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			tester.SetLastResult(&tt.result)

			got := tester.GetLastResult()
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			assertSpeedtestResultFields(t, got, tt.result)
		})
	}
}

func assertSpeedtestResultFields(t *testing.T, got *speedtest.Result, want speedtest.Result) {
	t.Helper()

	if got.Download != want.Download {
		t.Errorf("Download: got %v, want %v", got.Download, want.Download)
	}
	if got.Upload != want.Upload {
		t.Errorf("Upload: got %v, want %v", got.Upload, want.Upload)
	}
	if got.Latency != want.Latency {
		t.Errorf("Latency: got %v, want %v", got.Latency, want.Latency)
	}
	if got.Server != want.Server {
		t.Errorf("Server: got %q, want %q", got.Server, want.Server)
	}
	if got.Location != want.Location {
		t.Errorf("Location: got %q, want %q", got.Location, want.Location)
	}
	if got.Host != want.Host {
		t.Errorf("Host: got %q, want %q", got.Host, want.Host)
	}
	if got.Distance != want.Distance {
		t.Errorf("Distance: got %v, want %v", got.Distance, want.Distance)
	}
	if got.TestDuration != want.TestDuration {
		t.Errorf("TestDuration: got %v, want %v", got.TestDuration, want.TestDuration)
	}
}

// TestCurrentSpeedsBehavior documents the behavior of live speed updates.
func TestCurrentSpeedsBehavior(t *testing.T) {
	t.Run("speeds update independently", func(t *testing.T) {
		tester := speedtest.NewTester()

		// Set only download
		tester.SetCurrentSpeeds(100, 0)
		status := tester.GetStatus()
		if status.CurrentDownload != 100 || status.CurrentUpload != 0 {
			t.Error("download-only update failed")
		}

		// Set only upload (download resets if both are passed)
		tester.SetCurrentSpeeds(0, 50)
		status = tester.GetStatus()
		if status.CurrentDownload != 0 || status.CurrentUpload != 50 {
			t.Error("upload-only update failed")
		}

		// Set both
		tester.SetCurrentSpeeds(200, 100)
		status = tester.GetStatus()
		if status.CurrentDownload != 200 || status.CurrentUpload != 100 {
			t.Error("both speeds update failed")
		}
	})

	t.Run("speeds persist across status changes", func(t *testing.T) {
		tester := speedtest.NewTester()
		tester.SetCurrentSpeeds(500, 250)

		// Change phase
		tester.SetStatus("testing_upload", 70)
		status := tester.GetStatus()
		if status.CurrentDownload != 500 || status.CurrentUpload != 250 {
			t.Error("speeds should persist across status changes")
		}
	})
}

// TestServerIDBehavior documents the behavior of server ID configuration.
func TestServerIDBehavior(t *testing.T) {
	t.Run("NewTester has empty server ID", func(t *testing.T) {
		tester := speedtest.NewTester()
		if tester.TesterServerID() != "" {
			t.Errorf("expected empty, got %q", tester.TesterServerID())
		}
	})

	t.Run("NewTesterWithConfig sets server ID", func(t *testing.T) {
		tester := speedtest.NewTesterWithConfig("my-server")
		if tester.TesterServerID() != "my-server" {
			t.Errorf("expected 'my-server', got %q", tester.TesterServerID())
		}
	})

	t.Run("SetServerID replaces existing", func(t *testing.T) {
		tester := speedtest.NewTesterWithConfig("initial")
		tester.SetServerID("replaced")
		if tester.TesterServerID() != "replaced" {
			t.Errorf("expected 'replaced', got %q", tester.TesterServerID())
		}
	})

	t.Run("SetServerID can clear", func(t *testing.T) {
		tester := speedtest.NewTesterWithConfig("set")
		tester.SetServerID("")
		if tester.TesterServerID() != "" {
			t.Errorf("expected empty, got %q", tester.TesterServerID())
		}
	})
}

// TestConstantsBehavior documents the behavior of package constants.
func TestConstantsBehavior(t *testing.T) {
	t.Run("progress constants are in order", func(t *testing.T) {
		progressOrder := []int{
			speedtest.ProgressFindingServer,
			speedtest.ProgressTestingLatency,
			speedtest.ProgressTestingDownload,
			speedtest.ProgressTestingUpload,
			speedtest.ProgressComplete,
		}

		for i := 1; i < len(progressOrder); i++ {
			if progressOrder[i] <= progressOrder[i-1] {
				t.Errorf("progress %d should be > %d", progressOrder[i], progressOrder[i-1])
			}
		}
	})

	t.Run("BytesToMbps conversion is accurate", func(t *testing.T) {
		// 1 Mbps = 1,000,000 bits/sec = 125,000 bytes/sec
		if speedtest.BytesToMbps != 125000.0 {
			t.Errorf("BytesToMbps: got %v, want 125000.0", speedtest.BytesToMbps)
		}
	})

	t.Run("SpeedPollInterval is reasonable", func(t *testing.T) {
		// Should be between 50ms and 1s for responsive UI
		if speedtest.SpeedPollInterval < 50*time.Millisecond ||
			speedtest.SpeedPollInterval > time.Second {
			t.Errorf("SpeedPollInterval %v is outside reasonable range", speedtest.SpeedPollInterval)
		}
	})

	t.Run("IdleResetDelay is reasonable", func(t *testing.T) {
		// Should be between 1s and 10s
		if speedtest.IdleResetDelay < time.Second || speedtest.IdleResetDelay > 10*time.Second {
			t.Errorf("IdleResetDelay %v is outside reasonable range", speedtest.IdleResetDelay)
		}
	})
}

// TestConcurrencyBehavior documents thread-safety behavior.
func TestConcurrencyBehavior(t *testing.T) {
	t.Parallel()
	t.Run("concurrent status reads are safe", func(t *testing.T) {
		t.Parallel()

		tester := speedtest.NewTester()
		tester.SetStatus("testing", 50)

		var wg sync.WaitGroup
		for range 100 {
			wg.Go(func() {
				for range 100 {
					status := tester.GetStatus()
					_ = status.Phase
					_ = status.Progress
				}
			})
		}
		wg.Wait()
	})

	t.Run("concurrent writes and reads are safe", func(t *testing.T) {
		t.Parallel()

		tester := speedtest.NewTester()

		var wg sync.WaitGroup
		for range 50 {
			wg.Go(func() {
				for range 100 {
					tester.SetStatus("testing", 50)
					tester.SetCurrentSpeeds(100, 50)
				}
			})
			wg.Go(func() {
				for range 100 {
					_ = tester.GetStatus()
					_ = tester.GetLastResult()
				}
			})
		}
		wg.Wait()
	})
}

// TestBuildTestResultFromParamsBehavior documents BuildTestResultFromParams behavior.
func TestBuildTestResultFromParamsBehavior(t *testing.T) {
	t.Run("location format is sponsor comma country", func(t *testing.T) {
		tester := speedtest.NewTester()
		result := tester.BuildTestResultFromParams(
			100, 50,
			10*time.Millisecond,
			"Server",
			"Sponsor Name",
			"Country Name",
			"host.example.com",
			50,
			time.Now(),
		)

		expected := "Sponsor Name, Country Name"
		if result.Location != expected {
			t.Errorf("Location: got %q, want %q", result.Location, expected)
		}
	})

	t.Run("latency is converted from Duration to milliseconds", func(t *testing.T) {
		tester := speedtest.NewTester()

		tests := []struct {
			duration    time.Duration
			expectedMs  float64
			description string
		}{
			{5 * time.Millisecond, 5, "simple milliseconds"},
			{1 * time.Second, 1000, "one second"},
			{500 * time.Microsecond, 0, "sub-millisecond rounds to 0"},
			{1500 * time.Microsecond, 1, "1.5ms truncates to 1"},
		}

		for _, tt := range tests {
			result := tester.BuildTestResultFromParams(
				100, 50,
				tt.duration,
				"Server", "Sponsor", "Country",
				"host.example.com",
				50,
				time.Now(),
			)

			if result.Latency != tt.expectedMs {
				t.Errorf("%s: got %v, want %v", tt.description, result.Latency, tt.expectedMs)
			}
		}
	})

	t.Run("test duration is calculated from start time", func(t *testing.T) {
		tester := speedtest.NewTester()
		startTime := time.Now().Add(-15 * time.Second)

		result := tester.BuildTestResultFromParams(
			100, 50,
			10*time.Millisecond,
			"Server", "Sponsor", "Country",
			"host.example.com",
			50,
			startTime,
		)

		// Allow 0.5s tolerance
		if result.TestDuration < 14.5 || result.TestDuration > 15.5 {
			t.Errorf("TestDuration: got %v, want ~15", result.TestDuration)
		}
	})

	t.Run("timestamp is set to current time", func(t *testing.T) {
		tester := speedtest.NewTester()
		before := time.Now()

		result := tester.BuildTestResultFromParams(
			100, 50,
			10*time.Millisecond,
			"Server", "Sponsor", "Country",
			"host.example.com",
			50,
			time.Now().Add(-10*time.Second),
		)

		after := time.Now()

		if result.Timestamp.Before(before) || result.Timestamp.After(after) {
			t.Errorf("Timestamp %v not between %v and %v", result.Timestamp, before, after)
		}
	})
}

// TestRunTestConcurrentAttemptsBehavior documents concurrent RunTest behavior.
func TestRunTestConcurrentAttemptsBehavior(t *testing.T) {
	t.Run("only one test can run at a time", func(t *testing.T) {
		tester := speedtest.NewTester()
		tester.SetRunning(true)

		var successCount int32
		var errorCount int32
		var wg sync.WaitGroup

		for range 100 {
			wg.Go(func() {
				_, err := tester.RunTest(context.Background())
				if err == nil {
					atomic.AddInt32(&successCount, 1)
				} else {
					atomic.AddInt32(&errorCount, 1)
				}
			})
		}

		wg.Wait()

		if successCount != 0 {
			t.Errorf("expected 0 successes, got %d", successCount)
		}
		if errorCount != 100 {
			t.Errorf("expected 100 errors, got %d", errorCount)
		}
	})
}

// TestTesterIndependence documents that multiple Tester instances are independent.
func TestTesterIndependence(t *testing.T) {
	t.Run("status changes are isolated", func(t *testing.T) {
		tester1 := speedtest.NewTester()
		tester2 := speedtest.NewTester()

		tester1.SetStatus("testing_download", 50)
		tester2.SetStatus("testing_upload", 75)

		if tester1.GetStatus().Phase != "testing_download" {
			t.Error("tester1 status leaked to tester2")
		}
		if tester2.GetStatus().Phase != "testing_upload" {
			t.Error("tester2 status leaked to tester1")
		}
	})

	t.Run("results are isolated", func(t *testing.T) {
		tester1 := speedtest.NewTester()
		tester2 := speedtest.NewTester()

		tester1.SetLastResult(&speedtest.Result{Download: 100})
		tester2.SetLastResult(&speedtest.Result{Download: 200})

		if tester1.GetLastResult().Download != 100 {
			t.Error("tester1 result leaked")
		}
		if tester2.GetLastResult().Download != 200 {
			t.Error("tester2 result leaked")
		}
	})

	t.Run("server IDs are isolated", func(t *testing.T) {
		tester1 := speedtest.NewTesterWithConfig("server1")
		tester2 := speedtest.NewTesterWithConfig("server2")

		if tester1.TesterServerID() != "server1" {
			t.Error("tester1 serverID leaked")
		}
		if tester2.TesterServerID() != "server2" {
			t.Error("tester2 serverID leaked")
		}
	})
}

// TestStatusStructBehavior documents Status struct behavior.
func TestStatusStructBehavior(t *testing.T) {
	t.Run("zero value is valid idle state", func(t *testing.T) {
		status := speedtest.Status{}

		if status.Running {
			t.Error("zero Running should be false")
		}
		if status.Phase != "" {
			t.Errorf("zero Phase should be empty, got %q", status.Phase)
		}
		if status.Progress != 0 {
			t.Errorf("zero Progress should be 0, got %v", status.Progress)
		}
		if status.CurrentDownload != 0 {
			t.Errorf("zero CurrentDownload should be 0, got %v", status.CurrentDownload)
		}
		if status.CurrentUpload != 0 {
			t.Errorf("zero CurrentUpload should be 0, got %v", status.CurrentUpload)
		}
	})
}

// TestResultStructBehavior documents Result struct behavior.
func TestResultStructBehavior(t *testing.T) {
	t.Run("zero value is valid empty result", func(t *testing.T) {
		result := speedtest.Result{}

		if result.Download != 0 {
			t.Error("zero Download should be 0")
		}
		if result.Upload != 0 {
			t.Error("zero Upload should be 0")
		}
		if result.Latency != 0 {
			t.Error("zero Latency should be 0")
		}
		if result.Server != "" {
			t.Error("zero Server should be empty")
		}
		if result.Location != "" {
			t.Error("zero Location should be empty")
		}
		if result.Host != "" {
			t.Error("zero Host should be empty")
		}
		if result.Distance != 0 {
			t.Error("zero Distance should be 0")
		}
		if result.TestDuration != 0 {
			t.Error("zero TestDuration should be 0")
		}
		if !result.Timestamp.IsZero() {
			t.Error("zero Timestamp should be zero time")
		}
	})
}

// TestNilContextBehavior documents behavior when nil context is passed.
func TestNilContextBehavior(t *testing.T) {
	t.Run("nil context is accepted when already running", func(t *testing.T) {
		tester := speedtest.NewTester()
		tester.SetRunning(true)

		// Should not panic with nil context
		_, err := tester.RunTest(context.TODO())
		if err == nil {
			t.Error("expected error")
		}
		if err.Error() != "test already in progress" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestLastResultNilBehavior documents behavior when no result is set.
func TestLastResultNilBehavior(t *testing.T) {
	t.Run("GetLastResult returns nil initially", func(t *testing.T) {
		tester := speedtest.NewTester()
		if tester.GetLastResult() != nil {
			t.Error("expected nil result initially")
		}
	})

	t.Run("GetLastResult returns nil after SetLastResult(nil)", func(t *testing.T) {
		tester := speedtest.NewTester()
		tester.SetLastResult(&speedtest.Result{Download: 100})
		tester.SetLastResult(nil)
		if tester.GetLastResult() != nil {
			t.Error("expected nil result after SetLastResult(nil)")
		}
	})
}

// TestProgressValuesBehavior documents that any progress value is accepted.
func TestProgressValuesBehavior(t *testing.T) {
	t.Run("progress can be any float64 value", func(t *testing.T) {
		tester := speedtest.NewTester()

		values := []float64{0, 0.5, 1, 50, 99.9, 100, 150, -10}
		for _, v := range values {
			tester.SetStatus("testing", v)
			if tester.GetStatus().Progress != v {
				t.Errorf("progress %v not accepted", v)
			}
		}
	})
}

// TestPhaseValuesBehavior documents that any phase string is accepted.
func TestPhaseValuesBehavior(t *testing.T) {
	t.Run("phase can be any string value", func(t *testing.T) {
		tester := speedtest.NewTester()

		phases := []string{"", "idle", "testing", "custom_phase", "with spaces", "unicode_サーバー"}
		for _, p := range phases {
			tester.SetStatus(p, 0)
			if tester.GetStatus().Phase != p {
				t.Errorf("phase %q not accepted", p)
			}
		}
	})
}
