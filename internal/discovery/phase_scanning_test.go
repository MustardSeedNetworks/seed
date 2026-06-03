package discovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestNewScanningPhase(t *testing.T) {
	pipelineCfg := discovery.DefaultPipelineConfig()
	snmpCfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Timeout:     1 * time.Second,
	}

	phase := discovery.NewScanningPhase(&pipelineCfg, snmpCfg, nil)
	if phase == nil {
		t.Fatal("NewScanningPhase returned nil")
	}
	if phase.Name() != "scanning" {
		t.Errorf("Expected Name='scanning', got %s", phase.Name())
	}
}

func TestScanningPhase_Name(t *testing.T) {
	pipelineCfg := discovery.DefaultPipelineConfig()
	phase := discovery.NewScanningPhase(&pipelineCfg, nil, nil)

	if phase.Name() != "scanning" {
		t.Errorf("Expected Name='scanning', got %s", phase.Name())
	}
}

func TestScanningPhase_RunEmptyDevices(t *testing.T) {
	pipelineCfg := discovery.DefaultPipelineConfig()
	phase := discovery.NewScanningPhase(&pipelineCfg, nil, nil)
	ctx := context.Background()

	// Run with empty devices list
	devices, err := phase.Run(ctx, nil, nil)
	if err != nil {
		t.Errorf("Run returned error for empty devices: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("Expected empty devices, got %d", len(devices))
	}

	// Run with empty slice
	devices, err = phase.Run(ctx, []*discovery.DiscoveredDevice{}, nil)
	if err != nil {
		t.Errorf("Run returned error for empty slice: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("Expected empty devices, got %d", len(devices))
	}
}

func TestScanningPhase_RunBothDisabled(t *testing.T) {
	pipelineCfg := discovery.DefaultPipelineConfig()
	pipelineCfg.PortScan.Intensity = discovery.PortScanOff
	pipelineCfg.SNMPCollection.Enabled = false

	phase := discovery.NewScanningPhase(&pipelineCfg, nil, nil)
	ctx := context.Background()

	devices := []*discovery.DiscoveredDevice{
		{IP: "192.0.2.1", MAC: "00:11:22:33:44:55"},
	}

	result, err := phase.Run(ctx, devices, nil)
	if err != nil {
		t.Errorf("Run returned error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 device returned, got %d", len(result))
	}
}

func TestScanningProgress(t *testing.T) {
	var progress discovery.ScanningProgress

	// Test Start
	progress.Start(10)

	// Test initial state
	if progress.Scanned() != 0 {
		t.Errorf("Expected Scanned=0 initially, got %d", progress.Scanned())
	}
	if progress.PortsFound() != 0 {
		t.Errorf("Expected PortsFound=0 initially, got %d", progress.PortsFound())
	}
	if progress.SNMPSuccess() != 0 {
		t.Errorf("Expected SNMPSuccess=0 initially, got %d", progress.SNMPSuccess())
	}
	if progress.CurrentTarget() != "" {
		t.Errorf("Expected empty CurrentTarget initially, got %s", progress.CurrentTarget())
	}
}

func TestScanningProgress_IncrementScanned(t *testing.T) {
	var progress discovery.ScanningProgress
	progress.Start(10)

	progress.IncrementScanned()
	progress.IncrementScanned()
	progress.IncrementScanned()

	if progress.Scanned() != 3 {
		t.Errorf("Expected Scanned=3, got %d", progress.Scanned())
	}
}

func TestScanningProgress_AddPortsFound(t *testing.T) {
	var progress discovery.ScanningProgress
	progress.Start(10)

	progress.AddPortsFound(5)
	progress.AddPortsFound(3)
	progress.AddPortsFound(2)

	if progress.PortsFound() != 10 {
		t.Errorf("Expected PortsFound=10, got %d", progress.PortsFound())
	}
}

func TestScanningProgress_IncrementSNMPSuccess(t *testing.T) {
	var progress discovery.ScanningProgress
	progress.Start(10)

	progress.IncrementSNMPSuccess()
	progress.IncrementSNMPSuccess()

	if progress.SNMPSuccess() != 2 {
		t.Errorf("Expected SNMPSuccess=2, got %d", progress.SNMPSuccess())
	}
}

func TestScanningProgress_SetCurrentTarget(t *testing.T) {
	var progress discovery.ScanningProgress
	progress.Start(10)

	progress.SetCurrentTarget("192.168.1.1")
	if progress.CurrentTarget() != "192.168.1.1" {
		t.Errorf("Expected CurrentTarget='192.168.1.1', got %s", progress.CurrentTarget())
	}

	progress.SetCurrentTarget("10.0.0.1")
	if progress.CurrentTarget() != "10.0.0.1" {
		t.Errorf("Expected CurrentTarget='10.0.0.1', got %s", progress.CurrentTarget())
	}
}

func TestScanningProgress_PercentComplete(t *testing.T) {
	tests := []struct {
		name         string
		total        int
		scanned      int64
		expectedPct  float64
		tolerancePct float64
	}{
		{"no_devices", 0, 0, 100.0, 0.1}, // 0 devices = 100% complete
		{"none_scanned", 10, 0, 0.0, 0.1},
		{"half_scanned", 10, 5, 50.0, 0.1},
		{"all_scanned", 10, 10, 100.0, 0.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var progress discovery.ScanningProgress
			progress.Start(tt.total)

			for range tt.scanned {
				progress.IncrementScanned()
			}

			pct := progress.PercentComplete()
			diff := pct - tt.expectedPct
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerancePct {
				t.Errorf("Expected PercentComplete~=%.1f%%, got %.1f%%", tt.expectedPct, pct)
			}
		})
	}
}

func TestScanningProgress_GetStats(t *testing.T) {
	var progress discovery.ScanningProgress
	progress.Start(100)

	// Simulate some progress
	for range 50 {
		progress.IncrementScanned()
	}
	progress.AddPortsFound(150)
	progress.IncrementSNMPSuccess()
	progress.IncrementSNMPSuccess()
	progress.IncrementSNMPSuccess()

	start := time.Now().Add(-1 * time.Second)
	stats := progress.GetStats(start)

	if stats.TotalDevices != 100 {
		t.Errorf("Expected TotalDevices=100, got %d", stats.TotalDevices)
	}
	if stats.ScannedDevices != 50 {
		t.Errorf("Expected ScannedDevices=50, got %d", stats.ScannedDevices)
	}
	if stats.OpenPortsFound != 150 {
		t.Errorf("Expected OpenPortsFound=150, got %d", stats.OpenPortsFound)
	}
	if stats.SNMPSuccessful != 3 {
		t.Errorf("Expected SNMPSuccessful=3, got %d", stats.SNMPSuccessful)
	}
	if stats.PercentComplete != 50.0 {
		t.Errorf("Expected PercentComplete=50.0, got %f", stats.PercentComplete)
	}
	if stats.ElapsedMs < 1000 {
		t.Errorf("Expected ElapsedMs >= 1000, got %d", stats.ElapsedMs)
	}
}

func TestScanningProgress_Concurrency(t *testing.T) {
	var progress discovery.ScanningProgress
	progress.Start(1000)

	// Test concurrent operations
	done := make(chan struct{})
	for range 10 {
		go func() {
			for range 100 {
				progress.IncrementScanned()
				progress.AddPortsFound(1)
				progress.IncrementSNMPSuccess()
			}
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	if progress.Scanned() != 1000 {
		t.Errorf("Expected Scanned=1000, got %d", progress.Scanned())
	}
	if progress.PortsFound() != 1000 {
		t.Errorf("Expected PortsFound=1000, got %d", progress.PortsFound())
	}
	if progress.SNMPSuccess() != 1000 {
		t.Errorf("Expected SNMPSuccess=1000, got %d", progress.SNMPSuccess())
	}
}

func TestScanningStatsPayload_Fields(t *testing.T) {
	stats := discovery.ScanningStatsPayload{
		TotalDevices:    100,
		ScannedDevices:  75,
		OpenPortsFound:  250,
		SNMPSuccessful:  50,
		PercentComplete: 75.0,
		ElapsedMs:       5000,
	}

	if stats.TotalDevices != 100 {
		t.Errorf("Expected TotalDevices=100, got %d", stats.TotalDevices)
	}
	if stats.ScannedDevices != 75 {
		t.Errorf("Expected ScannedDevices=75, got %d", stats.ScannedDevices)
	}
	if stats.OpenPortsFound != 250 {
		t.Errorf("Expected OpenPortsFound=250, got %d", stats.OpenPortsFound)
	}
	if stats.SNMPSuccessful != 50 {
		t.Errorf("Expected SNMPSuccessful=50, got %d", stats.SNMPSuccessful)
	}
	if stats.PercentComplete != 75.0 {
		t.Errorf("Expected PercentComplete=75.0, got %f", stats.PercentComplete)
	}
	if stats.ElapsedMs != 5000 {
		t.Errorf("Expected ElapsedMs=5000, got %d", stats.ElapsedMs)
	}
}

func TestDeviceUpdatedPayload_Fields(t *testing.T) {
	device := &discovery.DiscoveredDevice{
		IP:  "192.168.1.1",
		MAC: "00:11:22:33:44:55",
	}
	payload := discovery.DeviceUpdatedPayload{
		Device: device,
		Phase:  "scanning",
	}

	if payload.Device != device {
		t.Error("Expected Device to be the same reference")
	}
	if payload.Phase != "scanning" {
		t.Errorf("Expected Phase='scanning', got %s", payload.Phase)
	}
}

func TestScanningPhase_RunWithDevices(t *testing.T) {
	pipelineCfg := discovery.DefaultPipelineConfig()
	pipelineCfg.PortScan.Intensity = discovery.PortScanQuick
	pipelineCfg.PortScan.ConnectTimeout = 50 * time.Millisecond
	pipelineCfg.Timing.PhaseTimeout = 500 * time.Millisecond
	pipelineCfg.Timing.MaxConcurrentHosts = 5
	pipelineCfg.SNMPCollection.Enabled = false

	phase := discovery.NewScanningPhase(&pipelineCfg, nil, nil)
	ctx := context.Background()

	// Create test devices with TEST-NET addresses
	devices := []*discovery.DiscoveredDevice{
		{IP: "192.0.2.1", MAC: "00:11:22:33:44:55"},
		{IP: "192.0.2.2", MAC: "AA:BB:CC:DD:EE:FF"},
	}

	result, err := phase.Run(ctx, devices, nil)
	if err != nil {
		t.Errorf("Run returned error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 devices returned, got %d", len(result))
	}
}

func TestScanningPhase_RunWithProgressChannel(t *testing.T) {
	pipelineCfg := discovery.DefaultPipelineConfig()
	pipelineCfg.PortScan.Intensity = discovery.PortScanQuick
	pipelineCfg.PortScan.ConnectTimeout = 50 * time.Millisecond
	pipelineCfg.Timing.PhaseTimeout = 1 * time.Second
	pipelineCfg.Timing.MaxConcurrentHosts = 5
	pipelineCfg.SNMPCollection.Enabled = false

	phase := discovery.NewScanningPhase(&pipelineCfg, nil, nil)
	ctx := context.Background()

	devices := []*discovery.DiscoveredDevice{
		{IP: "192.0.2.1", MAC: "00:11:22:33:44:55"},
	}

	progressCh := make(chan discovery.PhaseProgressPayload, 100)

	result, err := phase.Run(ctx, devices, progressCh)
	if err != nil {
		t.Errorf("Run returned error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 device returned, got %d", len(result))
	}

	// Drain progress channel
	close(progressCh)
	progressUpdates := 0
	for range progressCh {
		progressUpdates++
	}
	t.Logf("Received %d progress updates", progressUpdates)
}

func TestScanningPhase_ContextCancellation(t *testing.T) {
	pipelineCfg := discovery.DefaultPipelineConfig()
	pipelineCfg.PortScan.Intensity = discovery.PortScanComprehensive
	pipelineCfg.PortScan.ConnectTimeout = 5 * time.Second
	pipelineCfg.Timing.MaxConcurrentHosts = 1
	pipelineCfg.SNMPCollection.Enabled = false

	phase := discovery.NewScanningPhase(&pipelineCfg, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())

	devices := []*discovery.DiscoveredDevice{
		{IP: "192.0.2.1", MAC: "00:11:22:33:44:55"},
		{IP: "192.0.2.2", MAC: "AA:BB:CC:DD:EE:FF"},
		{IP: "192.0.2.3", MAC: "11:22:33:44:55:66"},
	}

	// Cancel after a brief delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := phase.Run(ctx, devices, nil)
	// Should return without error (context cancellation is handled gracefully)
	if err != nil {
		t.Errorf("Run returned error: %v", err)
	}
	t.Logf("Scanned %d devices before cancellation", len(result))
}

func TestScanningPhase_WithSNMP(t *testing.T) {
	pipelineCfg := discovery.DefaultPipelineConfig()
	pipelineCfg.PortScan.Intensity = discovery.PortScanOff
	pipelineCfg.SNMPCollection.Enabled = true
	pipelineCfg.Timing.PhaseTimeout = 500 * time.Millisecond

	snmpCfg := &config.SNMPConfig{
		Communities: []string{"public"},
		Timeout:     100 * time.Millisecond,
	}

	phase := discovery.NewScanningPhase(&pipelineCfg, snmpCfg, nil)
	ctx := context.Background()

	devices := []*discovery.DiscoveredDevice{
		{IP: "192.0.2.1", MAC: "00:11:22:33:44:55"},
	}

	result, err := phase.Run(ctx, devices, nil)
	if err != nil {
		t.Errorf("Run returned error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 device returned, got %d", len(result))
	}
}

func TestScanningPhase_EmptyIPDevice(t *testing.T) {
	pipelineCfg := discovery.DefaultPipelineConfig()
	pipelineCfg.PortScan.Intensity = discovery.PortScanQuick
	pipelineCfg.PortScan.ConnectTimeout = 50 * time.Millisecond
	pipelineCfg.Timing.PhaseTimeout = 500 * time.Millisecond
	pipelineCfg.SNMPCollection.Enabled = false

	phase := discovery.NewScanningPhase(&pipelineCfg, nil, nil)
	ctx := context.Background()

	// Device with empty IP should be skipped
	devices := []*discovery.DiscoveredDevice{
		{IP: "", MAC: "00:11:22:33:44:55"},
		{IP: "192.0.2.1", MAC: "AA:BB:CC:DD:EE:FF"},
	}

	result, err := phase.Run(ctx, devices, nil)
	if err != nil {
		t.Errorf("Run returned error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 devices returned, got %d", len(result))
	}
}
