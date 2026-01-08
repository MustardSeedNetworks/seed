// Package shell_test provides comprehensive tests for rogue device detection.
// These tests cover RogueService, RogueDevice, and RogueAlert types and behavior.
package shell_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// ========== Rogue Detection Lifecycle Tests ==========

// TestRogueServiceLifecycle tests the full lifecycle of RogueService.
func TestRogueServiceLifecycle(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	// Test initial state
	ctx := context.Background()
	devices, err := service.GetRogueDevices(ctx)
	if err != nil {
		if !errors.Is(err, shell.ErrNotInitialized) {
			t.Errorf("initial GetRogueDevices unexpected error: %v", err)
		}
	} else if len(devices) != 0 {
		t.Logf("initial devices count: %d (expected 0 for fresh service)", len(devices))
	}

	// Try to start (may fail without privileges)
	err = service.Start(ctx)
	if err != nil {
		t.Logf("Start returned error (expected without privileges): %v", err)
	}

	// Stop should always be safe
	service.Stop()
}

// TestRogueServiceMultipleStarts tests multiple start attempts.
func TestRogueServiceMultipleStarts(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	ctx := context.Background()

	// First start
	err1 := service.Start(ctx)
	if err1 == nil {
		// If first start succeeded, second should fail or be ignored
		err2 := service.Start(ctx)
		if err2 == nil {
			t.Log("Multiple starts succeeded (detector may handle this internally)")
		}
	}

	// Cleanup
	service.Stop()
}

// TestRogueServiceMultipleStops tests multiple stop calls.
func TestRogueServiceMultipleStops(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	// Stop should be safe to call multiple times
	service.Stop()
	service.Stop()
	service.Stop()
}

// ========== Rogue Device Retrieval Tests ==========

// TestRogueServiceGetRogueDevicesEmpty tests GetRogueDevices on fresh service.
func TestRogueServiceGetRogueDevicesEmpty(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	ctx := context.Background()

	devices, err := service.GetRogueDevices(ctx)
	if err != nil {
		// ErrNotInitialized is acceptable if detector isn't running
		if !errors.Is(err, shell.ErrNotInitialized) {
			t.Errorf("unexpected error: %v", err)
		}
		return
	}

	// Empty slice is expected for fresh service
	if devices == nil {
		t.Error("devices should be a non-nil slice")
	}
}

// TestRogueServiceGetRogueDevicesWithContext tests GetRogueDevices with various contexts.
func TestRogueServiceGetRogueDevicesWithContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ctxFunc func() (context.Context, context.CancelFunc)
	}{
		{
			name: "with_background_context",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
		{
			name: "with_timeout_context",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
		},
		{
			name: "with_cancelled_context",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
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

			devices, err := service.GetRogueDevices(ctx)
			if err != nil {
				if !errors.Is(err, shell.ErrNotInitialized) {
					t.Logf("GetRogueDevices returned error (may be expected): %v", err)
				}
				return
			}

			if devices == nil {
				t.Error("devices should be a non-nil slice")
			}
		})
	}
}

// ========== Rogue Alert Tests ==========

// TestRogueServiceGetAlertsEmpty tests GetAlerts on fresh service.
func TestRogueServiceGetAlertsEmpty(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	ctx := context.Background()

	alerts, err := service.GetAlerts(ctx)
	if err != nil {
		if !errors.Is(err, shell.ErrNotInitialized) {
			t.Errorf("unexpected error: %v", err)
		}
		return
	}

	// Empty slice is expected for fresh service
	if alerts == nil {
		t.Error("alerts should be a non-nil slice")
	}
}

// TestRogueServiceGetAlertsWithContext tests GetAlerts with various contexts.
func TestRogueServiceGetAlertsWithContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ctxFunc func() (context.Context, context.CancelFunc)
	}{
		{
			name: "with_background_context",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
		{
			name: "with_short_timeout",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 100*time.Millisecond)
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

			alerts, err := service.GetAlerts(ctx)
			if err != nil {
				if !errors.Is(err, shell.ErrNotInitialized) {
					t.Logf("GetAlerts returned error (may be expected): %v", err)
				}
				return
			}

			if alerts == nil {
				t.Error("alerts should be a non-nil slice")
			}
		})
	}
}

// ========== Acknowledge Device Tests ==========

// TestRogueServiceAcknowledgeDeviceVariousIDs tests AcknowledgeDevice with various ID formats.
func TestRogueServiceAcknowledgeDeviceVariousIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		deviceID string
	}{
		{
			name:     "ipv4_address",
			deviceID: "192.168.1.100",
		},
		{
			name:     "ipv6_address",
			deviceID: "fe80::1",
		},
		{
			name:     "mac_address_colon",
			deviceID: "00:11:22:33:44:55",
		},
		{
			name:     "mac_address_dash",
			deviceID: "00-11-22-33-44-55",
		},
		{
			name:     "empty_string",
			deviceID: "",
		},
		{
			name:     "uuid_format",
			deviceID: "550e8400-e29b-41d4-a716-446655440000",
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

			ctx := context.Background()

			err := service.AcknowledgeDevice(ctx, tt.deviceID)
			// Currently returns ErrNotImplemented for all inputs
			if !errors.Is(err, shell.ErrNotImplemented) {
				t.Errorf("expected ErrNotImplemented, got: %v", err)
			}
		})
	}
}

// ========== Rogue Detector Accessor Tests ==========

// TestRogueServiceDetectorNotNil tests that Detector() returns non-nil.
func TestRogueServiceDetectorNotNil(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	detector := service.Detector()
	if detector == nil {
		t.Error("Detector() should return non-nil RogueDetector")
	}
}

// TestRogueServiceDetectorConfiguration tests that detector has correct config.
func TestRogueServiceDetectorConfiguration(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	detector := service.Detector()
	if detector == nil {
		t.Fatal("Detector() should return non-nil")
	}

	// Get detector config
	detectorCfg := detector.GetConfig()
	if detectorCfg == nil {
		t.Fatal("Detector config should not be nil")
	}

	// Interface should be set (either "lo" or DefaultInterface)
	if detectorCfg.Interface == "" {
		t.Error("Detector interface should be set")
	}
}

// ========== Concurrent Rogue Operations Tests ==========

// TestRogueServiceConcurrentGetRogueDevices tests concurrent GetRogueDevices calls.
func TestRogueServiceConcurrentGetRogueDevices(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	ctx := context.Background()

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, _ = service.GetRogueDevices(ctx)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for GetRogueDevices")
		}
	}
}

// TestRogueServiceConcurrentGetAlerts tests concurrent GetAlerts calls.
func TestRogueServiceConcurrentGetAlerts(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	ctx := context.Background()

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, _ = service.GetAlerts(ctx)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for GetAlerts")
		}
	}
}

// ========== Rogue Service with Different Configurations ==========

// TestRogueServiceWithDifferentInterfaces tests RogueService with various interfaces.
func TestRogueServiceWithDifferentInterfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		interface_ string
	}{
		{
			name:      "loopback",
			interface_: "lo",
		},
		{
			name:      "eth0",
			interface_: "eth0",
		},
		{
			name:      "en0",
			interface_: "en0",
		},
		{
			name:      "wlan0",
			interface_: "wlan0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface(tt.interface_).
				Build()

			module := shell.New(cfg, nil)
			service := module.Rogue()

			if service == nil {
				t.Fatal("RogueService should not be nil")
			}

			detector := service.Detector()
			if detector == nil {
				t.Fatal("Detector should not be nil")
			}
		})
	}
}
