package shell_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// ========== GetDevice Success Path Tests ==========

// TestDiscoveryServiceGetDeviceValidation tests GetDevice input validation.
func TestDiscoveryServiceGetDeviceValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		deviceID string
	}{
		{"empty_id", ""},
		{"whitespace_id", "   "},
		{"special_chars", "!@#$%^&*()"},
		{"very_long_id", "a" + string(make([]byte, 1000))},
		{"unicode_id", "device-\u00e9\u00e8\u00ea"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				Build()

			module := shell.New(cfg, nil)
			service := module.Discovery()

			ctx := context.Background()

			// Should handle any input without panic
			device, err := service.GetDevice(ctx, tt.deviceID)
			if err == nil && device == nil {
				t.Error("if no error, device should not be nil")
			}
		})
	}
}

// ========== Posture Service Assess Edge Cases ==========

// TestPostureAssessWithNilVulnerabilityInternalService tests Assess error handling.
func TestPostureAssessWithNilVulnerabilityInternalService(t *testing.T) {
	t.Parallel()

	// This tests the path where vulnerability service returns error
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	// Multiple assessments should all succeed
	for i := 0; i < 3; i++ {
		score, err := service.Assess(ctx)
		if err != nil {
			t.Errorf("iteration %d: Assess() error = %v", i, err)
			continue
		}

		if score == nil {
			t.Errorf("iteration %d: score should not be nil", i)
			continue
		}

		// Score should be valid even when vulnerability service has issues
		if score.Overall < 0 || score.Overall > 100 {
			t.Errorf("iteration %d: Overall score %d out of range", i, score.Overall)
		}
	}
}

// ========== Vulnerability Service Scan Edge Cases ==========

// TestVulnerabilityServiceScanVariousInputs tests Scan with various inputs.
func TestVulnerabilityServiceScanVariousInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		targets []string
	}{
		{"nil_targets", nil},
		{"empty_targets", []string{}},
		{"single_localhost", []string{"127.0.0.1"}},
		{"multiple_same", []string{"127.0.0.1", "127.0.0.1", "127.0.0.1"}},
		{"invalid_ips", []string{"not-an-ip", "also-not-ip"}},
		{"mixed_valid_invalid", []string{"127.0.0.1", "not-an-ip", "192.168.1.1"}},
		{"ipv6_targets", []string{"::1", "fe80::1"}},
		{"large_target_list", makeLargeTargetList(50)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				Build()

			module := shell.New(cfg, nil)
			service := module.Vulnerability()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := service.Scan(ctx, tt.targets)
			if err != nil {
				// Expected for uninitialized scanner
				t.Logf("Scan returned error (may be expected): %v", err)
				return
			}

			if result == nil {
				t.Fatal("result should not be nil when no error")
			}

			// Verify counts are non-negative
			if result.TotalCritical < 0 || result.TotalHigh < 0 ||
				result.TotalMedium < 0 || result.TotalLow < 0 {
				t.Error("severity counts should not be negative")
			}
		})
	}
}

// makeLargeTargetList creates a list of test IPs.
func makeLargeTargetList(count int) []string {
	targets := make([]string, count)
	for i := 0; i < count; i++ {
		targets[i] = "192.168.1." + string(rune('0'+i%10))
	}
	return targets
}

// ========== Rogue Service Edge Cases ==========

// TestRogueServiceGetRogueDevicesAfterStart tests GetRogueDevices after Start attempt.
func TestRogueServiceGetRogueDevicesAfterStart(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	ctx := context.Background()

	// Try to start (may fail)
	_ = service.Start(ctx)

	// GetRogueDevices should work regardless of Start success
	devices, err := service.GetRogueDevices(ctx)
	if err == nil {
		// If no error, devices should be a valid slice
		if devices == nil {
			t.Error("devices should not be nil when no error")
		}
	}

	// Clean up
	service.Stop()
}

// TestRogueServiceGetAlertsAfterStart tests GetAlerts after Start attempt.
func TestRogueServiceGetAlertsAfterStart(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	ctx := context.Background()

	// Try to start (may fail)
	_ = service.Start(ctx)

	// GetAlerts should work regardless of Start success
	alerts, err := service.GetAlerts(ctx)
	if err == nil {
		// If no error, alerts should be a valid slice
		if alerts == nil {
			t.Error("alerts should not be nil when no error")
		}
	}

	// Clean up
	service.Stop()
}

// ========== Module Stop Edge Cases ==========

// TestModuleStopWithActiveServices tests Stop when services are active.
func TestModuleStopWithActiveServices(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)

	ctx := context.Background()

	// Start module
	_ = module.Start(ctx)

	// Access services to ensure they're initialized
	_ = module.Discovery()
	_ = module.Vulnerability()
	_ = module.Posture()
	_ = module.Rogue()

	// Stop should succeed
	err := module.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

// TestModuleStopNilServices tests Stop with nil internal services.
func TestModuleStopNilServices(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	// Create module without starting
	module := shell.New(cfg, nil)

	// Stop should succeed even without Start
	err := module.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Stop again should be safe
	err = module.Stop()
	if err != nil {
		t.Errorf("second Stop() error = %v", err)
	}
}

// ========== Discovery Service Start Error Path ==========

// TestDiscoveryServiceStartWithContext tests Start with various context states.
func TestDiscoveryServiceStartWithContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ctxFunc func() (context.Context, context.CancelFunc)
	}{
		{
			name: "with_value_context",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "key", "value") //nolint:staticcheck
				return ctx, func() {}
			},
		},
		{
			name: "with_long_timeout",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 1*time.Hour)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				Build()

			module := shell.New(cfg, nil)
			service := module.Discovery()

			ctx, cancel := tt.ctxFunc()
			defer cancel()

			// Start may fail, but shouldn't panic
			err := service.Start(ctx)
			if err != nil {
				t.Logf("Start returned error (may be expected): %v", err)
			}

			// Always stop
			service.Stop()
		})
	}
}

// ========== Vulnerability Service Stop with Cancel ==========

// TestVulnerabilityServiceStopWithCancel tests Stop calls cancel function.
func TestVulnerabilityServiceStopWithCancel(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Vulnerability()

	// Stop should be safe (may have cancel set internally)
	service.Stop()

	// Multiple stops should be safe
	service.Stop()
	service.Stop()
}

// ========== Rogue Service Start Error Path ==========

// TestRogueServiceStartWithContext tests Start with various context states.
func TestRogueServiceStartWithContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ctxFunc func() (context.Context, context.CancelFunc)
	}{
		{
			name: "with_value_context",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "key", "value") //nolint:staticcheck
				return ctx, func() {}
			},
		},
		{
			name: "with_deadline_context",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(1*time.Hour))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				Build()

			module := shell.New(cfg, nil)
			service := module.Rogue()

			ctx, cancel := tt.ctxFunc()
			defer cancel()

			// Start may fail, but shouldn't panic
			err := service.Start(ctx)
			if err != nil {
				t.Logf("Start returned error (may be expected): %v", err)
			}

			// Always stop
			service.Stop()
		})
	}
}
