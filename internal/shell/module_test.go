// These tests cover module initialization, service access, and lifecycle management.
package shell_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// ========== Module Creation Tests ==========

// TestNew tests the Shell module constructor with various configurations.
func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupCfg   func() *testutil.ConfigBuilder
		wantModule bool
	}{
		{
			name: "with_minimal_config",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo")
			},
			wantModule: true,
		},
		{
			name: "with_full_scan_config",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo").
					WithDiscoveryMethods(true, true, true).
					WithDiscoveryConcurrency(50)
			},
			wantModule: true,
		},
		{
			name: "with_passive_only_config",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo").
					WithDiscoveryMethods(false, false, false)
			},
			wantModule: true,
		},
		{
			name: "with_custom_interface",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("eth0")
			},
			wantModule: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.setupCfg().Build()
			module := shell.New(cfg, nil)

			if tt.wantModule && module == nil {
				t.Fatal("expected non-nil module")
			}

			// Verify all services are initialized
			if module.Discovery() == nil {
				t.Error("Discovery service should not be nil")
			}
			if module.Vulnerability() == nil {
				t.Error("Vulnerability service should not be nil")
			}
			if module.Posture() == nil {
				t.Error("Posture service should not be nil")
			}
			if module.Rogue() == nil {
				t.Error("Rogue service should not be nil")
			}
		})
	}
}

// TestNewWithNilDB tests module creation with nil database.
func TestNewWithNilDB(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)

	if module == nil {
		t.Fatal("module should be created even with nil database")
	}

	// Services should still be accessible
	if module.Discovery() == nil {
		t.Error("Discovery should be accessible with nil DB")
	}
	if module.Vulnerability() == nil {
		t.Error("Vulnerability should be accessible with nil DB")
	}
	if module.Posture() == nil {
		t.Error("Posture should be accessible with nil DB")
	}
	if module.Rogue() == nil {
		t.Error("Rogue should be accessible with nil DB")
	}
}

// ========== Module Lifecycle Tests ==========

// TestModuleStart tests the Start method of the Shell module.
func TestModuleStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupCfg  func() *testutil.ConfigBuilder
		ctxFunc   func() (context.Context, context.CancelFunc)
		wantError bool
	}{
		{
			name: "start_with_valid_context",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo")
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
			wantError: false,
		},
		{
			name: "start_with_background_context",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo")
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			wantError: false,
		},
		{
			name: "start_with_cancelled_context",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo")
			},
			ctxFunc: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			wantError: false, // Start should still succeed even with cancelled context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.setupCfg().Build()
			module := shell.New(cfg, nil)

			ctx, cancel := tt.ctxFunc()
			defer cancel()

			err := module.Start(ctx)

			if tt.wantError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Always stop the module after test
			_ = module.Stop()
		})
	}
}

// TestModuleStop tests the Stop method of the Shell module.
func TestModuleStop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		startFirst   bool
		stopMultiple bool
	}{
		{
			name:         "stop_after_start",
			startFirst:   true,
			stopMultiple: false,
		},
		{
			name:         "stop_without_start",
			startFirst:   false,
			stopMultiple: false,
		},
		{
			name:         "stop_multiple_times",
			startFirst:   true,
			stopMultiple: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				Build()

			module := shell.New(cfg, nil)

			if tt.startFirst {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = module.Start(ctx)
			}

			err := module.Stop()
			if err != nil {
				t.Errorf("first Stop() returned error: %v", err)
			}

			if tt.stopMultiple {
				err = module.Stop()
				if err != nil {
					t.Errorf("second Stop() returned error: %v", err)
				}
			}
		})
	}
}

// TestModuleStartStopCycle tests multiple start/stop cycles.
func TestModuleStartStopCycle(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)

	for i := range 3 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		err := module.Start(ctx)
		if err != nil {
			t.Errorf("cycle %d: Start() error: %v", i, err)
		}

		err = module.Stop()
		if err != nil {
			t.Errorf("cycle %d: Stop() error: %v", i, err)
		}

		cancel()
	}
}

// ========== Service Accessor Tests ==========

// TestModuleServiceAccessors tests concurrent access to module services.
func TestModuleServiceAccessors(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)

	// Run concurrent accesses
	done := make(chan bool, 4)

	go func() {
		for range 100 {
			_ = module.Discovery()
		}
		done <- true
	}()

	go func() {
		for range 100 {
			_ = module.Vulnerability()
		}
		done <- true
	}()

	go func() {
		for range 100 {
			_ = module.Posture()
		}
		done <- true
	}()

	go func() {
		for range 100 {
			_ = module.Rogue()
		}
		done <- true
	}()

	// Wait for all goroutines
	for range 4 {
		<-done
	}
}

// TestModuleServiceConsistency tests that service accessors return consistent instances.
func TestModuleServiceConsistency(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)

	// Get services multiple times
	disc1 := module.Discovery()
	disc2 := module.Discovery()
	if disc1 != disc2 {
		t.Error("Discovery() should return same instance")
	}

	vuln1 := module.Vulnerability()
	vuln2 := module.Vulnerability()
	if vuln1 != vuln2 {
		t.Error("Vulnerability() should return same instance")
	}

	pos1 := module.Posture()
	pos2 := module.Posture()
	if pos1 != pos2 {
		t.Error("Posture() should return same instance")
	}

	rogue1 := module.Rogue()
	rogue2 := module.Rogue()
	if rogue1 != rogue2 {
		t.Error("Rogue() should return same instance")
	}
}

// ========== Test Accessor Tests ==========

// TestModuleTestAccessor tests the ModuleTestAccessor helper.
func TestModuleTestAccessor(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	accessor := shell.ModuleTestAccessor{Module: module}

	// Test config accessor
	if accessor.GetCfg() == nil {
		t.Error("GetCfg() should not return nil")
	}

	// Test DB accessor (may be nil)
	// No assertion on DB value - it's nil in this test

	// Test service accessors
	if accessor.GetDiscoveryService() == nil {
		t.Error("GetDiscoveryService() should not return nil")
	}
	if accessor.GetVulnerabilityService() == nil {
		t.Error("GetVulnerabilityService() should not return nil")
	}
	if accessor.GetPostureService() == nil {
		t.Error("GetPostureService() should not return nil")
	}
	if accessor.GetRogueService() == nil {
		t.Error("GetRogueService() should not return nil")
	}
}
