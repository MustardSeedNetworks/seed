// Package speedtest_test provides network bandwidth testing using speedtest.net infrastructure.
// Test suite validates speedtest phases, progress tracking, and throughput measurement.
package speedtest_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

func TestNewTester(t *testing.T) {
	tester := speedtest.NewTester()

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
	tester := speedtest.NewTesterWithConfig("12345")
	if tester == nil {
		t.Fatal("expected non-nil tester")
	}

	if tester.TesterServerID() != "12345" {
		t.Errorf("expected serverID '12345', got %q", tester.TesterServerID())
	}

	status := tester.GetStatus()
	if status.Phase != "idle" {
		t.Errorf("expected initial phase 'idle', got %q", status.Phase)
	}
}

func TestTesterGetStatus(t *testing.T) {
	tester := speedtest.NewTester()

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
	tester := speedtest.NewTester()

	result := tester.GetLastResult()
	if result != nil {
		t.Error("expected nil result initially")
	}
}

func TestTesterSetServerID(t *testing.T) {
	tester := speedtest.NewTester()

	if tester.TesterServerID() != "" {
		t.Errorf("expected empty serverID initially, got %q", tester.TesterServerID())
	}

	tester.SetServerID("67890")
	if tester.TesterServerID() != "67890" {
		t.Errorf("expected serverID '67890', got %q", tester.TesterServerID())
	}
}

func TestTesterSetStatus(t *testing.T) {
	tester := speedtest.NewTester()

	tester.SetStatus("testing_download", 50.0)

	status := tester.GetStatus()
	if status.Phase != "testing_download" {
		t.Errorf("expected Phase 'testing_download', got %q", status.Phase)
	}
	if status.Progress != 50.0 {
		t.Errorf("expected Progress 50.0, got %v", status.Progress)
	}
}

func TestTesterSetRunning(t *testing.T) {
	tester := speedtest.NewTester()

	tester.SetRunning(true)
	status := tester.GetStatus()
	if !status.Running {
		t.Error("expected Running to be true after SetRunning(true)")
	}

	tester.SetRunning(false)
	status = tester.GetStatus()
	if status.Running {
		t.Error("expected Running to be false after SetRunning(false)")
	}
}

func TestResultFields(t *testing.T) {
	now := time.Now()
	result := speedtest.Result{
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
	if result.Timestamp != now {
		t.Errorf("expected Timestamp %v, got %v", now, result.Timestamp)
	}
	if result.TestDuration != 30.5 {
		t.Errorf("expected TestDuration 30.5, got %v", result.TestDuration)
	}
}

func TestStatusFields(t *testing.T) {
	status := speedtest.Status{
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

	tester := speedtest.NewTester()

	for _, phase := range validPhases {
		tester.SetStatus(phase, 0)
		status := tester.GetStatus()
		if status.Phase != phase {
			t.Errorf("expected Phase %q, got %q", phase, status.Phase)
		}
	}
}

func TestConcurrentStatusAccess(_ *testing.T) {
	tester := speedtest.NewTester()

	// Test concurrent reads don't cause race conditions.
	done := make(chan bool)
	for range 10 {
		go func() {
			for range 100 {
				_ = tester.GetStatus()
				_ = tester.GetLastResult()
			}
			done <- true
		}()
	}

	// Wait for all goroutines.
	for range 10 {
		<-done
	}
}

func TestProgressRange(t *testing.T) {
	tester := speedtest.NewTester()

	// Test various progress values.
	progressValues := []float64{0, 10, 25, 50, 75, 100}
	for _, progress := range progressValues {
		tester.SetStatus("testing", progress)
		status := tester.GetStatus()
		if status.Progress != progress {
			t.Errorf("expected Progress %v, got %v", progress, status.Progress)
		}
	}
}

func TestTesterMuLocking(_ *testing.T) {
	tester := speedtest.NewTester()

	// Test concurrent writes don't cause race conditions.
	done := make(chan bool)
	for i := range 5 {
		go func(id int) {
			for j := range 50 {
				tester.SetStatus("phase"+string(rune('0'+id)), float64(j))
				tester.SetRunning(j%2 == 0)
				tester.SetServerID("server" + string(rune('0'+id)))
			}
			done <- true
		}(i)
	}

	for range 5 {
		<-done
	}
}

func TestResultZeroValues(t *testing.T) {
	result := speedtest.Result{}

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
	status := speedtest.Status{}

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
	tester := speedtest.NewTester()

	serverIDs := []string{"server1", "server2", "server3", ""}
	for _, id := range serverIDs {
		tester.SetServerID(id)
		if tester.TesterServerID() != id {
			t.Errorf("expected serverID %q, got %q", id, tester.TesterServerID())
		}
	}
}

func TestTesterStatusTransitions(t *testing.T) {
	tester := speedtest.NewTester()

	// Simulate a test workflow.
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
		tester.SetStatus(tr.phase, tr.progress)
		tester.SetRunning(tr.running)

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
	tester := speedtest.NewTester()

	// Should be nil on new tester.
	result := tester.GetLastResult()
	if result != nil {
		t.Error("expected nil result on new tester")
	}
}

func TestTesterMultipleGetStatus(t *testing.T) {
	tester := speedtest.NewTester()

	// Get status multiple times.
	for range 100 {
		status := tester.GetStatus()
		if status.Phase != "idle" {
			t.Errorf("expected Phase 'idle', got %q", status.Phase)
		}
	}
}

func TestTesterSetCurrentSpeeds(t *testing.T) {
	tester := speedtest.NewTester()

	// Initially zero.
	status := tester.GetStatus()
	if status.CurrentDownload != 0 {
		t.Errorf("expected CurrentDownload 0, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("expected CurrentUpload 0, got %v", status.CurrentUpload)
	}

	// Set download speed only.
	tester.SetCurrentSpeeds(100.5, 0)
	status = tester.GetStatus()
	if status.CurrentDownload != 100.5 {
		t.Errorf("expected CurrentDownload 100.5, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("expected CurrentUpload 0, got %v", status.CurrentUpload)
	}

	// Set both download and upload speeds.
	tester.SetCurrentSpeeds(250.75, 50.25)
	status = tester.GetStatus()
	if status.CurrentDownload != 250.75 {
		t.Errorf("expected CurrentDownload 250.75, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 50.25 {
		t.Errorf("expected CurrentUpload 50.25, got %v", status.CurrentUpload)
	}

	// Reset speeds to zero.
	tester.SetCurrentSpeeds(0, 0)
	status = tester.GetStatus()
	if status.CurrentDownload != 0 {
		t.Errorf("expected CurrentDownload 0, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("expected CurrentUpload 0, got %v", status.CurrentUpload)
	}
}

func TestTesterSetCurrentSpeedsTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		download float64
		upload   float64
	}{
		{"zero speeds", 0, 0},
		{"download only", 500.0, 0},
		{"upload only", 0, 100.0},
		{"both speeds", 1000.0, 500.0},
		{"fractional speeds", 123.456, 78.901},
		{"very small speeds", 0.001, 0.002},
		{"very large speeds", 10000.0, 5000.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := speedtest.NewTester()
			tester.SetCurrentSpeeds(tt.download, tt.upload)
			status := tester.GetStatus()

			if status.CurrentDownload != tt.download {
				t.Errorf("expected CurrentDownload %v, got %v", tt.download, status.CurrentDownload)
			}
			if status.CurrentUpload != tt.upload {
				t.Errorf("expected CurrentUpload %v, got %v", tt.upload, status.CurrentUpload)
			}
		})
	}
}

func TestStatusCurrentSpeedFields(t *testing.T) {
	status := speedtest.Status{
		Running:         true,
		Phase:           "testing_download",
		Progress:        50.0,
		CurrentDownload: 250.5,
		CurrentUpload:   0,
	}

	if !status.Running {
		t.Error("expected Running true")
	}
	if status.Phase != "testing_download" {
		t.Errorf("expected Phase 'testing_download', got %q", status.Phase)
	}
	if status.Progress != 50.0 {
		t.Errorf("expected Progress 50.0, got %v", status.Progress)
	}
	if status.CurrentDownload != 250.5 {
		t.Errorf("expected CurrentDownload 250.5, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("expected CurrentUpload 0, got %v", status.CurrentUpload)
	}
}

func TestStatusUploadPhase(t *testing.T) {
	status := speedtest.Status{
		Running:         true,
		Phase:           "testing_upload",
		Progress:        75.0,
		CurrentDownload: 500.0,
		CurrentUpload:   125.5,
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
	if status.CurrentDownload != 500.0 {
		t.Errorf("expected CurrentDownload 500.0, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 125.5 {
		t.Errorf("expected CurrentUpload 125.5, got %v", status.CurrentUpload)
	}
}

func TestTesterSetLastResult(t *testing.T) {
	tester := speedtest.NewTester()

	// Initially nil.
	if tester.GetLastResult() != nil {
		t.Error("expected nil result initially")
	}

	// Set a result.
	now := time.Now()
	result := &speedtest.Result{
		Download:     500.0,
		Upload:       100.0,
		Latency:      10.5,
		Server:       "Test Server",
		Location:     "Test Location",
		Host:         "test.example.com",
		Distance:     25.0,
		Timestamp:    now,
		TestDuration: 15.0,
	}
	tester.SetLastResult(result)

	got := tester.GetLastResult()
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.Download != 500.0 {
		t.Errorf("expected Download 500.0, got %v", got.Download)
	}
	if got.Upload != 100.0 {
		t.Errorf("expected Upload 100.0, got %v", got.Upload)
	}
	if got.Latency != 10.5 {
		t.Errorf("expected Latency 10.5, got %v", got.Latency)
	}
	if got.Server != "Test Server" {
		t.Errorf("expected Server 'Test Server', got %q", got.Server)
	}
	if got.Location != "Test Location" {
		t.Errorf("expected Location 'Test Location', got %q", got.Location)
	}
	if got.Host != "test.example.com" {
		t.Errorf("expected Host 'test.example.com', got %q", got.Host)
	}
	if got.Distance != 25.0 {
		t.Errorf("expected Distance 25.0, got %v", got.Distance)
	}
	if got.TestDuration != 15.0 {
		t.Errorf("expected TestDuration 15.0, got %v", got.TestDuration)
	}
}

func TestTesterOverwriteLastResult(t *testing.T) {
	tester := speedtest.NewTester()

	// Set first result.
	result1 := &speedtest.Result{
		Download: 100.0,
		Upload:   50.0,
	}
	tester.SetLastResult(result1)

	// Verify first result.
	got := tester.GetLastResult()
	if got.Download != 100.0 {
		t.Errorf("expected Download 100.0, got %v", got.Download)
	}

	// Overwrite with second result.
	result2 := &speedtest.Result{
		Download: 200.0,
		Upload:   100.0,
	}
	tester.SetLastResult(result2)

	// Verify second result.
	got = tester.GetLastResult()
	if got.Download != 200.0 {
		t.Errorf("expected Download 200.0, got %v", got.Download)
	}
	if got.Upload != 100.0 {
		t.Errorf("expected Upload 100.0, got %v", got.Upload)
	}
}

func TestTesterClearLastResult(t *testing.T) {
	tester := speedtest.NewTester()

	// Set a result.
	result := &speedtest.Result{Download: 100.0}
	tester.SetLastResult(result)

	if tester.GetLastResult() == nil {
		t.Fatal("expected non-nil result")
	}

	// Clear by setting nil.
	tester.SetLastResult(nil)
	if tester.GetLastResult() != nil {
		t.Error("expected nil result after clearing")
	}
}

func TestProgressConstants(t *testing.T) {
	// Verify progress constants are in expected range.
	if speedtest.ProgressFindingServer != 10 {
		t.Errorf("expected ProgressFindingServer 10, got %d", speedtest.ProgressFindingServer)
	}
	if speedtest.ProgressTestingLatency != 20 {
		t.Errorf("expected ProgressTestingLatency 20, got %d", speedtest.ProgressTestingLatency)
	}
	if speedtest.ProgressTestingDownload != 40 {
		t.Errorf("expected ProgressTestingDownload 40, got %d", speedtest.ProgressTestingDownload)
	}
	if speedtest.ProgressTestingUpload != 70 {
		t.Errorf("expected ProgressTestingUpload 70, got %d", speedtest.ProgressTestingUpload)
	}
	if speedtest.ProgressComplete != 100 {
		t.Errorf("expected ProgressComplete 100, got %d", speedtest.ProgressComplete)
	}

	// Verify progress values are in ascending order.
	if speedtest.ProgressFindingServer >= speedtest.ProgressTestingLatency {
		t.Error("ProgressFindingServer should be less than ProgressTestingLatency")
	}
	if speedtest.ProgressTestingLatency >= speedtest.ProgressTestingDownload {
		t.Error("ProgressTestingLatency should be less than ProgressTestingDownload")
	}
	if speedtest.ProgressTestingDownload >= speedtest.ProgressTestingUpload {
		t.Error("ProgressTestingDownload should be less than ProgressTestingUpload")
	}
	if speedtest.ProgressTestingUpload >= speedtest.ProgressComplete {
		t.Error("ProgressTestingUpload should be less than ProgressComplete")
	}
}

func TestBytesToMbpsConstant(t *testing.T) {
	// 1 Mbps = 125000 bytes/sec (1,000,000 bits / 8 bits per byte).
	if speedtest.BytesToMbps != 125000.0 {
		t.Errorf("expected BytesToMbps 125000.0, got %v", speedtest.BytesToMbps)
	}

	// Verify conversion: 1,000,000 bytes/sec = 8 Mbps.
	bytesPerSec := 1_000_000.0
	mbps := bytesPerSec / speedtest.BytesToMbps
	if mbps != 8.0 {
		t.Errorf("1,000,000 bytes/sec should be 8 Mbps, got %v", mbps)
	}

	// Verify conversion: 125,000 bytes/sec = 1 Mbps.
	bytesPerSec = 125_000.0
	mbps = bytesPerSec / speedtest.BytesToMbps
	if mbps != 1.0 {
		t.Errorf("125,000 bytes/sec should be 1 Mbps, got %v", mbps)
	}
}

func TestSpeedPollIntervalConstant(t *testing.T) {
	if speedtest.SpeedPollInterval != 100*time.Millisecond {
		t.Errorf("expected SpeedPollInterval 100ms, got %v", speedtest.SpeedPollInterval)
	}
}

func TestIdleResetDelayConstant(t *testing.T) {
	if speedtest.IdleResetDelay != 2*time.Second {
		t.Errorf("expected IdleResetDelay 2s, got %v", speedtest.IdleResetDelay)
	}
}

func TestResultTypicalValues(t *testing.T) {
	tests := []struct {
		name     string
		download float64
		upload   float64
		latency  float64
	}{
		{"DSL connection", 20.0, 5.0, 25.0},
		{"Cable connection", 200.0, 20.0, 15.0},
		{"Fiber connection", 950.0, 950.0, 5.0},
		{"Mobile 4G", 50.0, 25.0, 35.0},
		{"Mobile 5G", 500.0, 100.0, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := speedtest.Result{
				Download: tt.download,
				Upload:   tt.upload,
				Latency:  tt.latency,
			}

			if result.Download != tt.download {
				t.Errorf("expected Download %v, got %v", tt.download, result.Download)
			}
			if result.Upload != tt.upload {
				t.Errorf("expected Upload %v, got %v", tt.upload, result.Upload)
			}
			if result.Latency != tt.latency {
				t.Errorf("expected Latency %v, got %v", tt.latency, result.Latency)
			}
		})
	}
}

func TestResultEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		download float64
		upload   float64
		latency  float64
	}{
		{"zero values", 0, 0, 0},
		{"very slow connection", 0.1, 0.01, 1000.0},
		{"asymmetric", 1000.0, 1.0, 5.0},
		{"high latency", 100.0, 50.0, 500.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := speedtest.Result{
				Download: tt.download,
				Upload:   tt.upload,
				Latency:  tt.latency,
			}

			if result.Download != tt.download {
				t.Errorf("expected Download %v, got %v", tt.download, result.Download)
			}
			if result.Upload != tt.upload {
				t.Errorf("expected Upload %v, got %v", tt.upload, result.Upload)
			}
			if result.Latency != tt.latency {
				t.Errorf("expected Latency %v, got %v", tt.latency, result.Latency)
			}
		})
	}
}

func TestResultServerInfo(t *testing.T) {
	tests := []struct {
		name     string
		server   string
		location string
		host     string
		distance float64
	}{
		{
			name:     "typical server",
			server:   "Comcast Speed Test",
			location: "Comcast, United States",
			host:     "speedtest.comcast.net:8080",
			distance: 50.5,
		},
		{
			name:     "international server",
			server:   "Vodafone UK",
			location: "Vodafone, United Kingdom",
			host:     "speedtest.vodafone.co.uk:8080",
			distance: 5000.0,
		},
		{
			name:     "empty location",
			server:   "Test Server",
			location: "",
			host:     "test.example.com",
			distance: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := speedtest.Result{
				Server:   tt.server,
				Location: tt.location,
				Host:     tt.host,
				Distance: tt.distance,
			}

			if result.Server != tt.server {
				t.Errorf("expected Server %q, got %q", tt.server, result.Server)
			}
			if result.Location != tt.location {
				t.Errorf("expected Location %q, got %q", tt.location, result.Location)
			}
			if result.Host != tt.host {
				t.Errorf("expected Host %q, got %q", tt.host, result.Host)
			}
			if result.Distance != tt.distance {
				t.Errorf("expected Distance %v, got %v", tt.distance, result.Distance)
			}
		})
	}
}

func TestResultTimestampAndDuration(t *testing.T) {
	now := time.Now()
	duration := 25.5

	result := speedtest.Result{
		Timestamp:    now,
		TestDuration: duration,
	}

	if result.Timestamp != now {
		t.Errorf("expected Timestamp %v, got %v", now, result.Timestamp)
	}
	if result.TestDuration != duration {
		t.Errorf("expected TestDuration %v, got %v", duration, result.TestDuration)
	}
}

func TestTesterConcurrentWriteOperations(_ *testing.T) {
	tester := speedtest.NewTester()

	done := make(chan bool)

	// Helper to spawn concurrent writers.
	spawnStatusWriters(tester, done, 5)
	spawnRunningWriters(tester, done, 5)
	spawnSpeedWriters(tester, done, 5)
	spawnResultWriters(tester, done, 5)
	spawnReaders(tester, done, 5)

	// Wait for all goroutines (5 groups x 5 goroutines each).
	for range 25 {
		<-done
	}
}

func spawnStatusWriters(tester *speedtest.Tester, done chan bool, count int) {
	for i := range count {
		go func(id int) {
			for j := range 20 {
				tester.SetStatus("phase"+string(rune('0'+id)), float64(j)*5)
			}
			done <- true
		}(i)
	}
}

func spawnRunningWriters(tester *speedtest.Tester, done chan bool, count int) {
	for range count {
		go func() {
			for j := range 20 {
				tester.SetRunning(j%2 == 0)
			}
			done <- true
		}()
	}
}

func spawnSpeedWriters(tester *speedtest.Tester, done chan bool, count int) {
	for i := range count {
		go func(id int) {
			for j := range 20 {
				tester.SetCurrentSpeeds(float64(id*100+j), float64(j*10))
			}
			done <- true
		}(i)
	}
}

func spawnResultWriters(tester *speedtest.Tester, done chan bool, count int) {
	for i := range count {
		go func(id int) {
			for j := range 20 {
				tester.SetLastResult(&speedtest.Result{
					Download: float64(id * 100),
					Upload:   float64(j * 10),
				})
			}
			done <- true
		}(i)
	}
}

func spawnReaders(tester *speedtest.Tester, done chan bool, count int) {
	for range count {
		go func() {
			for range 20 {
				_ = tester.GetStatus()
				_ = tester.GetLastResult()
			}
			done <- true
		}()
	}
}

func TestTesterMixedReadWriteOperations(_ *testing.T) {
	tester := speedtest.NewTester()

	done := make(chan bool)

	// Writer goroutine.
	go func() {
		for i := range 100 {
			tester.SetStatus("testing", float64(i))
			tester.SetRunning(true)
			tester.SetCurrentSpeeds(float64(i*10), float64(i*5))
			tester.SetServerID("server" + string(rune('0'+i%10)))
		}
		done <- true
	}()

	// Multiple reader goroutines.
	for range 10 {
		go func() {
			for range 100 {
				_ = tester.GetStatus()
				_ = tester.GetLastResult()
				_ = tester.TesterServerID()
			}
			done <- true
		}()
	}

	// Wait for all goroutines.
	for range 11 {
		<-done
	}
}

func TestStatusFullLifecycle(t *testing.T) {
	tester := speedtest.NewTester()

	// Initial state.
	status := tester.GetStatus()
	if status.Running || status.Phase != "idle" || status.Progress != 0 {
		t.Error("unexpected initial state")
	}

	// Simulate test start.
	tester.SetRunning(true)
	status = tester.GetStatus()
	if !status.Running {
		t.Error("expected Running after SetRunning(true)")
	}

	// Finding server phase.
	tester.SetStatus("finding_server", float64(speedtest.ProgressFindingServer))
	status = tester.GetStatus()
	if status.Phase != "finding_server" {
		t.Errorf("expected phase 'finding_server', got %q", status.Phase)
	}
	if status.Progress != float64(speedtest.ProgressFindingServer) {
		t.Errorf("expected progress %d, got %v", speedtest.ProgressFindingServer, status.Progress)
	}

	// Testing latency phase.
	tester.SetStatus("testing_latency", float64(speedtest.ProgressTestingLatency))
	status = tester.GetStatus()
	if status.Phase != "testing_latency" {
		t.Errorf("expected phase 'testing_latency', got %q", status.Phase)
	}

	// Testing download phase with live speeds.
	tester.SetStatus("testing_download", float64(speedtest.ProgressTestingDownload))
	tester.SetCurrentSpeeds(250.5, 0)
	status = tester.GetStatus()
	if status.Phase != "testing_download" {
		t.Errorf("expected phase 'testing_download', got %q", status.Phase)
	}
	if status.CurrentDownload != 250.5 {
		t.Errorf("expected CurrentDownload 250.5, got %v", status.CurrentDownload)
	}

	// Testing upload phase with live speeds.
	tester.SetStatus("testing_upload", float64(speedtest.ProgressTestingUpload))
	tester.SetCurrentSpeeds(500.0, 125.0)
	status = tester.GetStatus()
	if status.Phase != "testing_upload" {
		t.Errorf("expected phase 'testing_upload', got %q", status.Phase)
	}
	if status.CurrentDownload != 500.0 {
		t.Errorf("expected CurrentDownload 500.0, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 125.0 {
		t.Errorf("expected CurrentUpload 125.0, got %v", status.CurrentUpload)
	}

	// Complete phase.
	tester.SetStatus("complete", float64(speedtest.ProgressComplete))
	status = tester.GetStatus()
	if status.Phase != "complete" {
		t.Errorf("expected phase 'complete', got %q", status.Phase)
	}
	if status.Progress != 100 {
		t.Errorf("expected progress 100, got %v", status.Progress)
	}

	// Store result.
	now := time.Now()
	tester.SetLastResult(&speedtest.Result{
		Download:     500.0,
		Upload:       125.0,
		Latency:      15.0,
		Server:       "Test Server",
		Location:     "Test Location",
		Host:         "test.example.com",
		Distance:     50.0,
		Timestamp:    now,
		TestDuration: 20.0,
	})

	result := tester.GetLastResult()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Download != 500.0 {
		t.Errorf("expected Download 500.0, got %v", result.Download)
	}

	// Reset to idle.
	tester.SetRunning(false)
	tester.SetStatus("idle", 0)
	tester.SetCurrentSpeeds(0, 0)
	status = tester.GetStatus()
	if status.Running {
		t.Error("expected not running after reset")
	}
	if status.Phase != "idle" {
		t.Errorf("expected phase 'idle', got %q", status.Phase)
	}
}

func TestNewTesterWithEmptyConfig(t *testing.T) {
	tester := speedtest.NewTesterWithConfig("")
	if tester == nil {
		t.Fatal("expected non-nil tester")
	}

	if tester.TesterServerID() != "" {
		t.Errorf("expected empty serverID, got %q", tester.TesterServerID())
	}

	status := tester.GetStatus()
	if status.Phase != "idle" {
		t.Errorf("expected Phase 'idle', got %q", status.Phase)
	}
}

func TestNewTesterWithLongServerID(t *testing.T) {
	longID := "server-id-12345-abcdef-67890-ghijkl"
	tester := speedtest.NewTesterWithConfig(longID)

	if tester.TesterServerID() != longID {
		t.Errorf("expected serverID %q, got %q", longID, tester.TesterServerID())
	}
}

func TestTesterSetServerIDEmpty(t *testing.T) {
	tester := speedtest.NewTesterWithConfig("initial-server")

	if tester.TesterServerID() != "initial-server" {
		t.Errorf("expected serverID 'initial-server', got %q", tester.TesterServerID())
	}

	// Clear server ID.
	tester.SetServerID("")
	if tester.TesterServerID() != "" {
		t.Errorf("expected empty serverID, got %q", tester.TesterServerID())
	}
}

func TestStatusCompletePhase(t *testing.T) {
	status := speedtest.Status{
		Running:         true,
		Phase:           "complete",
		Progress:        100.0,
		CurrentDownload: 500.0,
		CurrentUpload:   100.0,
	}

	if !status.Running {
		t.Error("expected Running true")
	}
	if status.Phase != "complete" {
		t.Errorf("expected Phase 'complete', got %q", status.Phase)
	}
	if status.Progress != 100.0 {
		t.Errorf("expected Progress 100.0, got %v", status.Progress)
	}
	if status.CurrentDownload != 500.0 {
		t.Errorf("expected CurrentDownload 500.0, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 100.0 {
		t.Errorf("expected CurrentUpload 100.0, got %v", status.CurrentUpload)
	}
}

func TestResultWithAllFields(t *testing.T) {
	now := time.Now()
	result := speedtest.Result{
		Download:     945.67,
		Upload:       923.45,
		Latency:      3.5,
		Server:       "Fiber Provider Speed Test",
		Location:     "Fiber Provider, USA",
		Host:         "speedtest.fiber.example.com:8080",
		Distance:     12.34,
		Timestamp:    now,
		TestDuration: 18.75,
	}

	// Verify all fields.
	if result.Download != 945.67 {
		t.Errorf("expected Download 945.67, got %v", result.Download)
	}
	if result.Upload != 923.45 {
		t.Errorf("expected Upload 923.45, got %v", result.Upload)
	}
	if result.Latency != 3.5 {
		t.Errorf("expected Latency 3.5, got %v", result.Latency)
	}
	if result.Server != "Fiber Provider Speed Test" {
		t.Errorf("expected Server 'Fiber Provider Speed Test', got %q", result.Server)
	}
	if result.Location != "Fiber Provider, USA" {
		t.Errorf("expected Location 'Fiber Provider, USA', got %q", result.Location)
	}
	if result.Host != "speedtest.fiber.example.com:8080" {
		t.Errorf("expected Host 'speedtest.fiber.example.com:8080', got %q", result.Host)
	}
	if result.Distance != 12.34 {
		t.Errorf("expected Distance 12.34, got %v", result.Distance)
	}
	if !result.Timestamp.Equal(now) {
		t.Errorf("expected Timestamp %v, got %v", now, result.Timestamp)
	}
	if result.TestDuration != 18.75 {
		t.Errorf("expected TestDuration 18.75, got %v", result.TestDuration)
	}
}

func TestMultipleTesterInstances(t *testing.T) {
	tester1 := speedtest.NewTester()
	tester2 := speedtest.NewTesterWithConfig("server-1")
	tester3 := speedtest.NewTesterWithConfig("server-2")

	// Verify testers are independent.
	tester1.SetStatus("testing", 50)
	tester2.SetStatus("complete", 100)
	tester3.SetStatus("idle", 0)

	status1 := tester1.GetStatus()
	status2 := tester2.GetStatus()
	status3 := tester3.GetStatus()

	if status1.Phase != "testing" {
		t.Errorf("tester1: expected Phase 'testing', got %q", status1.Phase)
	}
	if status2.Phase != "complete" {
		t.Errorf("tester2: expected Phase 'complete', got %q", status2.Phase)
	}
	if status3.Phase != "idle" {
		t.Errorf("tester3: expected Phase 'idle', got %q", status3.Phase)
	}

	// Verify server IDs are independent.
	if tester1.TesterServerID() != "" {
		t.Errorf("tester1: expected empty serverID, got %q", tester1.TesterServerID())
	}
	if tester2.TesterServerID() != "server-1" {
		t.Errorf("tester2: expected serverID 'server-1', got %q", tester2.TesterServerID())
	}
	if tester3.TesterServerID() != "server-2" {
		t.Errorf("tester3: expected serverID 'server-2', got %q", tester3.TesterServerID())
	}
}

func TestProgressBoundaries(t *testing.T) {
	tester := speedtest.NewTester()

	tests := []struct {
		name     string
		progress float64
	}{
		{"zero", 0},
		{"minimum positive", 0.0001},
		{"ten percent", 10},
		{"quarter", 25},
		{"half", 50},
		{"three quarters", 75},
		{"ninety nine", 99},
		{"full", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester.SetStatus("testing", tt.progress)
			status := tester.GetStatus()
			if status.Progress != tt.progress {
				t.Errorf("expected Progress %v, got %v", tt.progress, status.Progress)
			}
		})
	}
}

func TestStatusIdleDefault(t *testing.T) {
	status := speedtest.Status{}

	// Default (zero) values.
	if status.Running {
		t.Error("expected Running false by default")
	}
	if status.Phase != "" {
		t.Errorf("expected empty Phase by default, got %q", status.Phase)
	}
	if status.Progress != 0 {
		t.Errorf("expected Progress 0 by default, got %v", status.Progress)
	}
	if status.CurrentDownload != 0 {
		t.Errorf("expected CurrentDownload 0 by default, got %v", status.CurrentDownload)
	}
	if status.CurrentUpload != 0 {
		t.Errorf("expected CurrentUpload 0 by default, got %v", status.CurrentUpload)
	}
}

func TestRunTestAlreadyRunning(t *testing.T) {
	tester := speedtest.NewTester()

	// Simulate a test already in progress.
	tester.SetRunning(true)

	ctx := context.Background()
	result, err := tester.RunTest(ctx)

	// Should return error.
	if err == nil {
		t.Error("expected error when test already in progress")
	}
	if err.Error() != "test already in progress" {
		t.Errorf("expected 'test already in progress' error, got %q", err.Error())
	}
	if result != nil {
		t.Error("expected nil result when error occurs")
	}

	// Clean up.
	tester.SetRunning(false)
}

func TestRunTestAlreadyRunningConcurrent(t *testing.T) {
	tester := speedtest.NewTester()

	// Simulate a test in progress.
	tester.SetRunning(true)

	// Try to start multiple tests concurrently.
	const numAttempts = 10
	errors := make(chan error, numAttempts)
	results := make(chan *speedtest.Result, numAttempts)

	for range numAttempts {
		go func() {
			ctx := context.Background()
			result, err := tester.RunTest(ctx)
			errors <- err
			results <- result
		}()
	}

	// All attempts should fail with "already in progress".
	for range numAttempts {
		err := <-errors
		result := <-results

		if err == nil {
			t.Error("expected error for concurrent test attempt")
		}
		if err != nil && err.Error() != "test already in progress" {
			t.Errorf("expected 'test already in progress' error, got %q", err.Error())
		}
		if result != nil {
			t.Error("expected nil result for concurrent test attempt")
		}
	}

	// Clean up.
	tester.SetRunning(false)
}

func TestRunTestNotRunningInitially(t *testing.T) {
	tester := speedtest.NewTester()

	// Verify not running initially.
	status := tester.GetStatus()
	if status.Running {
		t.Error("expected not running initially")
	}
}
