package speedtest

import (
	"testing"
	"time"
)

func TestNewTester(t *testing.T) {
	tester := NewTester()

	if tester == nil {
		t.Fatal("expected non-nil tester")
	}

	status := tester.GetStatus()
	if status.Phase != "idle" {
		t.Errorf("expected initial phase 'idle', got %q", status.Phase)
	}
	if status.Running {
		t.Error("expected Running to be false initially")
	}
	if status.Progress != 0 {
		t.Errorf("expected Progress 0, got %v", status.Progress)
	}
}

func TestNewTesterWithConfig(t *testing.T) {
	tester := NewTesterWithConfig("12345")

	if tester == nil {
		t.Fatal("expected non-nil tester")
	}

	if tester.serverID != "12345" {
		t.Errorf("expected serverID '12345', got %q", tester.serverID)
	}

	status := tester.GetStatus()
	if status.Phase != "idle" {
		t.Errorf("expected initial phase 'idle', got %q", status.Phase)
	}
}

func TestTesterGetStatus(t *testing.T) {
	tester := NewTester()

	status := tester.GetStatus()
	if status.Running {
		t.Error("expected Running to be false initially")
	}
	if status.Phase != "idle" {
		t.Errorf("expected Phase 'idle', got %q", status.Phase)
	}
	if status.Progress != 0 {
		t.Errorf("expected Progress 0, got %v", status.Progress)
	}
}

func TestTesterGetLastResult(t *testing.T) {
	tester := NewTester()

	result := tester.GetLastResult()
	if result != nil {
		t.Error("expected nil result initially")
	}
}

func TestTesterSetServerID(t *testing.T) {
	tester := NewTester()

	if tester.serverID != "" {
		t.Errorf("expected empty serverID initially, got %q", tester.serverID)
	}

	tester.SetServerID("67890")
	if tester.serverID != "67890" {
		t.Errorf("expected serverID '67890', got %q", tester.serverID)
	}
}

func TestTesterSetStatus(t *testing.T) {
	tester := NewTester()

	tester.setStatus("testing_download", 50.0)

	status := tester.GetStatus()
	if status.Phase != "testing_download" {
		t.Errorf("expected Phase 'testing_download', got %q", status.Phase)
	}
	if status.Progress != 50.0 {
		t.Errorf("expected Progress 50.0, got %v", status.Progress)
	}
}

func TestTesterSetRunning(t *testing.T) {
	tester := NewTester()

	tester.setRunning(true)
	status := tester.GetStatus()
	if !status.Running {
		t.Error("expected Running to be true after setRunning(true)")
	}

	tester.setRunning(false)
	status = tester.GetStatus()
	if status.Running {
		t.Error("expected Running to be false after setRunning(false)")
	}
}

func TestResultFields(t *testing.T) {
	now := time.Now()
	result := Result{
		Download:     100.5,
		Upload:       50.2,
		Latency:      15.3,
		Server:       "Test Server",
		Location:     "New York, US",
		Host:         "speedtest.example.com",
		Distance:     50.0,
		Timestamp:    now,
		TestDuration: 30.5,
	}

	if result.Download != 100.5 {
		t.Errorf("expected Download 100.5, got %v", result.Download)
	}
	if result.Upload != 50.2 {
		t.Errorf("expected Upload 50.2, got %v", result.Upload)
	}
	if result.Latency != 15.3 {
		t.Errorf("expected Latency 15.3, got %v", result.Latency)
	}
	if result.Server != "Test Server" {
		t.Errorf("expected Server 'Test Server', got %q", result.Server)
	}
	if result.Location != "New York, US" {
		t.Errorf("expected Location 'New York, US', got %q", result.Location)
	}
	if result.Host != "speedtest.example.com" {
		t.Errorf("expected Host 'speedtest.example.com', got %q", result.Host)
	}
	if result.Distance != 50.0 {
		t.Errorf("expected Distance 50.0, got %v", result.Distance)
	}
	if result.TestDuration != 30.5 {
		t.Errorf("expected TestDuration 30.5, got %v", result.TestDuration)
	}
}

func TestStatusFields(t *testing.T) {
	status := Status{
		Running:  true,
		Phase:    "testing_upload",
		Progress: 75.0,
	}

	if !status.Running {
		t.Error("expected Running true")
	}
	if status.Phase != "testing_upload" {
		t.Errorf("expected Phase 'testing_upload', got %q", status.Phase)
	}
	if status.Progress != 75.0 {
		t.Errorf("expected Progress 75.0, got %v", status.Progress)
	}
}

func TestStatusPhases(t *testing.T) {
	validPhases := []string{
		"idle",
		"finding_server",
		"testing_latency",
		"testing_download",
		"testing_upload",
		"complete",
	}

	tester := NewTester()

	for _, phase := range validPhases {
		tester.setStatus(phase, 0)
		status := tester.GetStatus()
		if status.Phase != phase {
			t.Errorf("expected Phase %q, got %q", phase, status.Phase)
		}
	}
}

func TestConcurrentStatusAccess(t *testing.T) {
	tester := NewTester()

	// Test concurrent reads don't cause race conditions
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = tester.GetStatus()
				_ = tester.GetLastResult()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestProgressRange(t *testing.T) {
	tester := NewTester()

	// Test various progress values
	progressValues := []float64{0, 10, 25, 50, 75, 100}
	for _, progress := range progressValues {
		tester.setStatus("testing", progress)
		status := tester.GetStatus()
		if status.Progress != progress {
			t.Errorf("expected Progress %v, got %v", progress, status.Progress)
		}
	}
}

func TestTesterMuLocking(t *testing.T) {
	tester := NewTester()

	// Test concurrent writes don't cause race conditions
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				tester.setStatus("phase"+string(rune('0'+id)), float64(j))
				tester.setRunning(j%2 == 0)
				tester.SetServerID("server" + string(rune('0'+id)))
			}
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestResultZeroValues(t *testing.T) {
	result := Result{}

	if result.Download != 0 {
		t.Error("expected zero Download")
	}
	if result.Upload != 0 {
		t.Error("expected zero Upload")
	}
	if result.Latency != 0 {
		t.Error("expected zero Latency")
	}
	if result.Server != "" {
		t.Error("expected empty Server")
	}
	if result.Location != "" {
		t.Error("expected empty Location")
	}
	if result.Host != "" {
		t.Error("expected empty Host")
	}
}

func TestStatusZeroValues(t *testing.T) {
	status := Status{}

	if status.Running {
		t.Error("expected Running false by default")
	}
	if status.Phase != "" {
		t.Error("expected empty Phase by default")
	}
	if status.Progress != 0 {
		t.Error("expected zero Progress by default")
	}
}

func TestTesterSetServerIDMultiple(t *testing.T) {
	tester := NewTester()

	serverIDs := []string{"server1", "server2", "server3", ""}
	for _, id := range serverIDs {
		tester.SetServerID(id)
		if tester.serverID != id {
			t.Errorf("expected serverID %q, got %q", id, tester.serverID)
		}
	}
}

func TestTesterStatusTransitions(t *testing.T) {
	tester := NewTester()

	// Simulate a test workflow
	transitions := []struct {
		phase    string
		progress float64
		running  bool
	}{
		{"idle", 0, false},
		{"finding_server", 10, true},
		{"testing_latency", 20, true},
		{"testing_download", 40, true},
		{"testing_upload", 70, true},
		{"complete", 100, true},
		{"idle", 0, false},
	}

	for _, tr := range transitions {
		tester.setStatus(tr.phase, tr.progress)
		tester.setRunning(tr.running)

		status := tester.GetStatus()
		if status.Phase != tr.phase {
			t.Errorf("expected Phase %q, got %q", tr.phase, status.Phase)
		}
		if status.Progress != tr.progress {
			t.Errorf("expected Progress %v, got %v", tr.progress, status.Progress)
		}
		if status.Running != tr.running {
			t.Errorf("expected Running %v, got %v", tr.running, status.Running)
		}
	}
}

func TestTesterGetLastResultNil(t *testing.T) {
	tester := NewTester()

	// Should be nil on new tester
	result := tester.GetLastResult()
	if result != nil {
		t.Error("expected nil result on new tester")
	}
}

func TestTesterMultipleGetStatus(t *testing.T) {
	tester := NewTester()

	// Get status multiple times
	for i := 0; i < 100; i++ {
		status := tester.GetStatus()
		if status.Phase != "idle" {
			t.Errorf("expected Phase 'idle', got %q", status.Phase)
		}
	}
}
