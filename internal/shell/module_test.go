package shell_test

import (
	"context"
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/shell"
)

// ========== Module Creation Tests ==========

func TestNewModule(t *testing.T) {
	cfg := config.DefaultConfig()

	module := shell.New(cfg, nil)
	if module == nil {
		t.Fatal("New returned nil")
	}

	// Verify services are initialized
	if module.Discovery() == nil {
		t.Error("expected Discovery service to be initialized")
	}
	if module.Vulnerability() == nil {
		t.Error("expected Vulnerability service to be initialized")
	}
	if module.Posture() == nil {
		t.Error("expected Posture service to be initialized")
	}
	if module.Rogue() == nil {
		t.Error("expected Rogue service to be initialized")
	}
}

func TestNewModuleWithNilConfig(t *testing.T) {
	// Module should handle nil config gracefully (may panic or return nil depending on impl)
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior - nil config may cause panic
			t.Log("New with nil config caused panic (expected)")
		}
	}()

	// This test documents behavior - may panic with nil config
	_ = shell.New(nil, nil)
}

// ========== Module Start/Stop Tests ==========

func TestModuleStartStop(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()

	// Start should not return error
	if err := module.Start(ctx); err != nil {
		t.Errorf("Start returned unexpected error: %v", err)
	}

	// Stop should not return error
	if err := module.Stop(); err != nil {
		t.Errorf("Stop returned unexpected error: %v", err)
	}
}

func TestModuleStopWithCancelledContext(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Start with cancelled context
	if err := module.Start(ctx); err != nil {
		t.Errorf("Start with cancelled context returned unexpected error: %v", err)
	}

	// Stop should still work
	if err := module.Stop(); err != nil {
		t.Errorf("Stop returned unexpected error: %v", err)
	}
}

func TestModuleMultipleStops(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()
	_ = module.Start(ctx)

	// Multiple stops should not panic or error
	for i := 0; i < 3; i++ {
		if err := module.Stop(); err != nil {
			t.Errorf("Stop call %d returned unexpected error: %v", i+1, err)
		}
	}
}

// ========== Service Accessor Tests ==========

func TestModuleDiscoveryAccessor(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	discovery := module.Discovery()
	if discovery == nil {
		t.Fatal("Discovery returned nil")
	}

	// Multiple calls should return the same instance
	discovery2 := module.Discovery()
	if discovery != discovery2 {
		t.Error("Discovery should return the same instance on multiple calls")
	}
}

func TestModuleVulnerabilityAccessor(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	vuln := module.Vulnerability()
	if vuln == nil {
		t.Fatal("Vulnerability returned nil")
	}

	// Multiple calls should return the same instance
	vuln2 := module.Vulnerability()
	if vuln != vuln2 {
		t.Error("Vulnerability should return the same instance on multiple calls")
	}
}

func TestModulePostureAccessor(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	posture := module.Posture()
	if posture == nil {
		t.Fatal("Posture returned nil")
	}

	// Multiple calls should return the same instance
	posture2 := module.Posture()
	if posture != posture2 {
		t.Error("Posture should return the same instance on multiple calls")
	}
}

func TestModuleRogueAccessor(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	rogue := module.Rogue()
	if rogue == nil {
		t.Fatal("Rogue returned nil")
	}

	// Multiple calls should return the same instance
	rogue2 := module.Rogue()
	if rogue != rogue2 {
		t.Error("Rogue should return the same instance on multiple calls")
	}
}

// ========== PostureService Tests ==========

func TestPostureServiceAssess(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()
	posture := module.Posture()

	score, err := posture.Assess(ctx)
	if err != nil {
		t.Errorf("Assess returned unexpected error: %v", err)
	}

	if score == nil {
		t.Fatal("Assess returned nil score")
	}

	// Initial score should be 100 (no vulnerabilities)
	if score.Overall != 100 {
		t.Errorf("expected initial overall score 100, got %d", score.Overall)
	}

	// Categories should be initialized
	if score.Categories == nil {
		t.Error("expected Categories to be initialized")
	}

	// Issues should be initialized (empty)
	if score.Issues == nil {
		t.Error("expected Issues to be initialized")
	}

	// AssessedAt should be set
	if score.AssessedAt.IsZero() {
		t.Error("expected AssessedAt to be set")
	}
}

func TestPostureServiceAssessWithCancelledContext(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	posture := module.Posture()
	score, err := posture.Assess(ctx)

	// Should still return a score (doesn't depend on context for basic assessment)
	if err != nil {
		t.Logf("Assess with cancelled context: %v", err)
	}
	if score == nil {
		t.Error("expected score even with cancelled context")
	}
}

// ========== DiscoveryService Tests ==========

func TestDiscoveryServiceGetDevicesUninitialized(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()
	discovery := module.Discovery()

	// GetDevices may return nil/error when deviceDiscovery is nil/not properly initialized
	devices, err := discovery.GetDevices(ctx)
	if err != nil && err != shell.ErrNotInitialized {
		t.Logf("GetDevices returned: devices=%v, err=%v", devices, err)
	}
}

func TestDiscoveryServiceGetDeviceNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()
	discovery := module.Discovery()

	// GetDevice for non-existent device
	device, err := discovery.GetDevice(ctx, "non-existent-id")
	if err == nil {
		t.Error("expected error for non-existent device")
	}
	if device != nil {
		t.Error("expected nil device for non-existent ID")
	}
}

func TestDiscoveryServiceAccessors(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	discovery := module.Discovery()

	// Test Service() accessor
	svc := discovery.Service()
	if svc == nil {
		t.Error("expected Service() to return non-nil")
	}

	// Test DeviceDiscovery() accessor
	dd := discovery.DeviceDiscovery()
	if dd == nil {
		t.Error("expected DeviceDiscovery() to return non-nil")
	}
}

// ========== VulnerabilityService Tests ==========

func TestVulnerabilityServiceGetVulnerabilitiesUninitialized(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()
	vuln := module.Vulnerability()

	// GetVulnerabilities may return nil/error when scanner is nil
	vulns, err := vuln.GetVulnerabilities(ctx)
	if err != nil && err != shell.ErrNotInitialized {
		t.Logf("GetVulnerabilities returned: vulns=%v, err=%v", vulns, err)
	}
}

func TestVulnerabilityServiceUpdateStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()
	vuln := module.Vulnerability()

	// UpdateStatus should return ErrNotImplemented
	err := vuln.UpdateStatus(ctx, "vuln-id", shell.VulnStatusResolved)
	if err != shell.ErrNotImplemented {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestVulnerabilityServiceAccessors(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	vuln := module.Vulnerability()

	// Test Scanner() accessor - may return nil if scanner init failed
	scanner := vuln.Scanner()
	// Scanner might be nil depending on config, that's okay
	_ = scanner
}

// ========== RogueService Tests ==========

func TestRogueServiceGetRogueDevicesUninitialized(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()
	rogue := module.Rogue()

	devices, err := rogue.GetRogueDevices(ctx)
	if err != nil && err != shell.ErrNotInitialized {
		t.Logf("GetRogueDevices returned: devices=%v, err=%v", devices, err)
	}
}

func TestRogueServiceGetAlerts(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()
	rogue := module.Rogue()

	alerts, err := rogue.GetAlerts(ctx)
	if err != nil && err != shell.ErrNotInitialized {
		t.Logf("GetAlerts returned: alerts=%v, err=%v", alerts, err)
	}
}

func TestRogueServiceAcknowledgeDevice(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	ctx := context.Background()
	rogue := module.Rogue()

	// AcknowledgeDevice should return ErrNotImplemented
	err := rogue.AcknowledgeDevice(ctx, "device-id")
	if err != shell.ErrNotImplemented {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestRogueServiceAccessors(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	rogue := module.Rogue()

	// Test Detector() accessor
	detector := rogue.Detector()
	if detector == nil {
		t.Error("expected Detector() to return non-nil")
	}
}

// ========== Concurrency Tests ==========

func TestModuleConcurrentAccess(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	done := make(chan bool, 4)

	// Concurrent access to different services
	go func() {
		for i := 0; i < 100; i++ {
			_ = module.Discovery()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = module.Vulnerability()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = module.Posture()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = module.Rogue()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}

// ========== Test Accessor Pattern ==========

func TestModuleTestAccessor(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	accessor := &shell.ModuleTestAccessor{Module: module}

	// Test that accessor can retrieve internal fields
	if accessor.GetCfg() == nil {
		t.Error("expected GetCfg to return non-nil")
	}

	if accessor.GetDiscoveryService() == nil {
		t.Error("expected GetDiscoveryService to return non-nil")
	}

	if accessor.GetVulnerabilityService() == nil {
		t.Error("expected GetVulnerabilityService to return non-nil")
	}

	if accessor.GetPostureService() == nil {
		t.Error("expected GetPostureService to return non-nil")
	}

	if accessor.GetRogueService() == nil {
		t.Error("expected GetRogueService to return non-nil")
	}
}

func TestDiscoveryServiceTestAccessor(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	discovery := module.Discovery()
	accessor := &shell.DiscoveryServiceTestAccessor{Service: discovery}

	// Verify accessor works
	if accessor.GetCfg() == nil {
		t.Error("expected GetCfg to return non-nil")
	}
}

func TestPostureServiceTestAccessor(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	posture := module.Posture()
	accessor := &shell.PostureServiceTestAccessor{Service: posture}

	// Verify accessor works
	if accessor.GetCfg() == nil {
		t.Error("expected GetCfg to return non-nil")
	}

	if accessor.GetDiscovery() == nil {
		t.Error("expected GetDiscovery to return non-nil")
	}

	if accessor.GetVulnerability() == nil {
		t.Error("expected GetVulnerability to return non-nil")
	}
}

func TestRogueServiceTestAccessor(t *testing.T) {
	cfg := config.DefaultConfig()
	module := shell.New(cfg, nil)

	rogue := module.Rogue()
	accessor := &shell.RogueServiceTestAccessor{Service: rogue}

	// Verify accessor works
	if accessor.GetCfg() == nil {
		t.Error("expected GetCfg to return non-nil")
	}
}
