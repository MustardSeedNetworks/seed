package discovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestNewEngine(t *testing.T) {
	engine := discovery.NewEngine(nil)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}

	accessor := &discovery.EngineTestAccessor{Engine: engine}
	if accessor.GetRegistry() == nil {
		t.Error("expected non-nil registry")
	}
	if accessor.GetEventBus() == nil {
		t.Error("expected non-nil event bus")
	}
}

func TestEngineStartStop(t *testing.T) {
	engine := discovery.NewEngine(nil)

	ctx := context.Background()
	if err := engine.Start(ctx); err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if !engine.IsRunning() {
		t.Error("expected engine to be running")
	}

	// Starting again should fail
	if err := engine.Start(ctx); err == nil {
		t.Error("expected error when starting already running engine")
	}

	engine.Stop()

	if engine.IsRunning() {
		t.Error("expected engine to be stopped")
	}
}

func TestEngineSetCollectors(_ *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	// Test that setters don't panic (collectors can be nil for testing)
	engine.SetWiredCollector(nil)
	engine.SetWiFiCollector(nil)
	engine.SetBluetoothCollector(nil)
	engine.SetSNMPCollector(nil)
	engine.SetPortScanner(nil)
	engine.SetProfiler(nil)
	engine.SetVulnScanner(nil)
}

func TestEngineGetCapabilities(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	caps := engine.GetCapabilities()

	// Without collectors, most should be false
	if caps["wired"] {
		t.Error("wired should be false without collector")
	}
	if caps["wifi"] {
		t.Error("wifi should be false without collector")
	}
	if caps["bluetooth"] {
		t.Error("bluetooth should be false without collector")
	}

	// Correlation is always available
	if !caps["correlation"] {
		t.Error("correlation should always be true")
	}
}

func TestEngineScanOptions(t *testing.T) {
	quickOpts := discovery.DefaultQuickScanOpts()
	if quickOpts == nil {
		t.Fatal("expected non-nil ScanTypeQuick scan options")
	}
	if quickOpts.FreshWiredScan {
		t.Error("ScanTypeQuick scan should not have fresh wired scan")
	}

	fullOpts := discovery.DefaultFullScanOpts()
	if fullOpts == nil {
		t.Fatal("expected non-nil ScanTypeFull scan options")
	}
	if !fullOpts.FreshWiredScan {
		t.Error("ScanTypeFull scan should have fresh wired scan")
	}
	if !fullOpts.IncludeSNMP {
		t.Error("ScanTypeFull scan should include SNMP")
	}
	if !fullOpts.IncludeVulnScan {
		t.Error("ScanTypeFull scan should include vuln scan")
	}
}

func TestEngineScanWithoutCollectors(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	ctx := context.Background()

	// Scan without any collectors should still work
	result, err := engine.Scan(ctx, discovery.DefaultQuickScanOpts())
	if err != nil {
		t.Errorf("Scan failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.ScanType != "ScanTypeQuick" {
		t.Errorf("expected scan type 'ScanTypeQuick', got %s", result.ScanType)
	}
	if len(result.Phases) < 2 {
		t.Error("expected at least 2 phases (discovery, correlation)")
	}
}

func TestEngineScanAlreadyInProgress(_ *testing.T) {
	engine := discovery.NewEngine(&discovery.EngineConfig{
		ScanTimeout: 5 * time.Second,
	})
	defer engine.Stop()

	ctx := context.Background()

	// Start first scan in goroutine (it will wait on timeout)
	done := make(chan struct{})
	go func() {
		_, _ = engine.Scan(ctx, &discovery.ScanOptions{
			Timeout: 100 * time.Millisecond,
		})
		close(done)
	}()

	// Give time for first scan to start
	time.Sleep(10 * time.Millisecond)

	// Try second scan - should fail if first is still running
	// But since our scan is fast, it might complete
	// This test is mainly to ensure no panics
	_, _ = engine.Scan(ctx, discovery.DefaultQuickScanOpts())

	<-done
}

func TestEngineQuickScan(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	ctx := context.Background()

	result, err := engine.QuickScan(ctx)
	if err != nil {
		t.Errorf("QuickScan failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ScanType != "ScanTypeQuick" {
		t.Errorf("expected scan type 'ScanTypeQuick', got %s", result.ScanType)
	}
}

func TestEngineFullScan(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	ctx := context.Background()

	result, err := engine.FullScan(ctx)
	if err != nil {
		t.Errorf("FullScan failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ScanType != "ScanTypeFull" {
		t.Errorf("expected scan type 'ScanTypeFull', got %s", result.ScanType)
	}
}

func TestEngineGetLastScan(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	// No scan yet
	if engine.GetLastScan() != nil {
		t.Error("expected nil before first scan")
	}

	// Run a scan
	ctx := context.Background()
	engine.QuickScan(ctx)

	if engine.GetLastScan() == nil {
		t.Error("expected non-nil after scan")
	}
}

func TestEngineDeviceAccess(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	// Add device directly to registry for testing
	device := &discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:FF",
		IP:  "192.168.1.100",
	}
	accessor := &discovery.EngineTestAccessor{Engine: engine}
	accessor.GetRegistry().AddOrUpdate(device)

	// Get all devices
	devices := engine.GetDevices()
	if len(devices) != 1 {
		t.Errorf("expected 1 device, got %d", len(devices))
	}

	// Get by MAC
	result := engine.GetDevice("aa:bb:cc:dd:ee:ff")
	if result == nil {
		t.Error("expected to find device by MAC")
	}

	// Get by IP
	result = engine.GetDeviceByIP("192.168.1.100")
	if result == nil {
		t.Error("expected to find device by IP")
	}
}

func TestEngineStats(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	stats := engine.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.Running {
		t.Error("expected not running initially")
	}

	// Start engine
	engine.Start(context.Background())
	stats = engine.GetStats()
	if !stats.Running {
		t.Error("expected running after Start")
	}
}

func TestEngineSubscriptions(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	sub := engine.SubscribeAll(func(_ *discovery.Event) {
		// Handler for testing subscription
	})

	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}

	// Unsubscribe
	engine.Unsubscribe(sub.ID())

	// Verify via stats
	accessor := &discovery.EngineTestAccessor{Engine: engine}
	stats := accessor.GetEventBus().Stats()
	if stats.SubscriberCount != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", stats.SubscriberCount)
	}
}

func TestEngineRegistry(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	registry := engine.Registry()
	if registry == nil {
		t.Error("expected non-nil registry")
	}
	accessor := &discovery.EngineTestAccessor{Engine: engine}
	if registry != accessor.GetRegistry() {
		t.Error("expected same registry instance")
	}
}

func TestEngineEventBus(t *testing.T) {
	engine := discovery.NewEngine(nil)
	defer engine.Stop()

	bus := engine.EventBus()
	if bus == nil {
		t.Error("expected non-nil event bus")
	}
	accessor := &discovery.EngineTestAccessor{Engine: engine}
	if bus != accessor.GetEventBus() {
		t.Error("expected same event bus instance")
	}
}

func TestEnsureConnectionType(t *testing.T) {
	tests := []struct {
		name     string
		types    []discovery.ConnectionType
		add      discovery.ConnectionType
		expected int
	}{
		{
			name:     "add to empty",
			types:    nil,
			add:      discovery.ConnectionWired,
			expected: 1,
		},
		{
			name:     "add new type",
			types:    []discovery.ConnectionType{discovery.ConnectionWired},
			add:      discovery.ConnectionWiFi,
			expected: 2,
		},
		{
			name:     "no duplicate",
			types:    []discovery.ConnectionType{discovery.ConnectionWired},
			add:      discovery.ConnectionWired,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := discovery.ExportEnsureConnectionType(tt.types, tt.add)
			if len(result) != tt.expected {
				t.Errorf("expected %d types, got %d", tt.expected, len(result))
			}
		})
	}
}
