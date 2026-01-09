package shell_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// ========== Discovery Start/Stop Tests ==========

// TestDiscoveryServiceStartStop tests Start and Stop methods.
func TestDiscoveryServiceStartStop(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start may fail without privileges
	err := service.Start(ctx)
	if err != nil {
		t.Logf("Start returned error (may be expected): %v", err)
	}

	// Stop should always be safe
	service.Stop()
}

// TestDiscoveryServiceStopIdempotent tests that Stop is idempotent.
func TestDiscoveryServiceStopIdempotent(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	// Multiple stops should be safe
	service.Stop()
	service.Stop()
	service.Stop()
}

// ========== Discovery Scan Tests ==========

// TestDiscoveryServiceDiscoverWithOptions tests Discover with various options.
func TestDiscoveryServiceDiscoverWithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts *shell.DiscoveryOptions
	}{
		{
			name: "nil_options",
			opts: nil,
		},
		{
			name: "arp_only",
			opts: &shell.DiscoveryOptions{
				Interface:   "lo",
				EnableARP:   true,
				EnableICMP:  false,
				Timeout:     500 * time.Millisecond,
				Concurrency: 1,
			},
		},
		{
			name: "icmp_only",
			opts: &shell.DiscoveryOptions{
				Interface:   "lo",
				EnableARP:   false,
				EnableICMP:  true,
				Timeout:     500 * time.Millisecond,
				Concurrency: 1,
			},
		},
		{
			name: "with_subnets",
			opts: &shell.DiscoveryOptions{
				Interface:   "lo",
				Subnets:     []string{"192.168.1.0/24"},
				EnableARP:   true,
				Timeout:     500 * time.Millisecond,
				Concurrency: 1,
			},
		},
		{
			name: "with_port_scan",
			opts: &shell.DiscoveryOptions{
				Interface:     "lo",
				PortScan:      true,
				PortScanPorts: []int{22, 80, 443},
				Timeout:       500 * time.Millisecond,
				Concurrency:   1,
			},
		},
		{
			name: "all_protocols_enabled",
			opts: &shell.DiscoveryOptions{
				Interface:   "lo",
				EnableARP:   true,
				EnableICMP:  true,
				EnableNDP:   true,
				EnableLLDP:  true,
				EnableCDP:   true,
				EnableSNMP:  true,
				Timeout:     500 * time.Millisecond,
				Concurrency: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				WithDiscoveryMethods(false, false, false).
				Build()

			module := shell.New(cfg, nil)
			service := module.Discovery()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := service.Discover(ctx, tt.opts)
			if err != nil {
				t.Logf("Discover returned error (may be expected): %v", err)
				return
			}

			if result == nil {
				t.Fatal("result should not be nil when no error")
			}

			// Verify result has valid timestamps
			if result.StartedAt.IsZero() {
				t.Error("StartedAt should be set")
			}
			if result.CompletedAt.IsZero() {
				t.Error("CompletedAt should be set")
			}
		})
	}
}

// TestDiscoveryServiceDiscoverResultFields tests all fields of DiscoveryResult.
func TestDiscoveryServiceDiscoverResultFields(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		WithDiscoveryMethods(false, false, false).
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := service.Discover(ctx, nil)
	if err != nil {
		t.Logf("Discover returned error (may be expected): %v", err)
		return
	}

	// Check all fields
	if result.Devices == nil {
		t.Error("Devices should be initialized (may be empty)")
	}
	if result.NewDevices < 0 {
		t.Error("NewDevices should not be negative")
	}
	if result.UpdatedDevices < 0 {
		t.Error("UpdatedDevices should not be negative")
	}
	if result.OfflineDevices < 0 {
		t.Error("OfflineDevices should not be negative")
	}
	if result.ScanDuration < 0 {
		t.Error("ScanDuration should not be negative")
	}
}

// ========== GetDevices Tests ==========

// TestDiscoveryServiceGetDevicesReturnsSlice tests that GetDevices returns a valid slice.
func TestDiscoveryServiceGetDevicesReturnsSlice(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	ctx := context.Background()

	devices, err := service.GetDevices(ctx)
	if err != nil {
		if errors.Is(err, shell.ErrNotInitialized) {
			t.Logf("Service not initialized (expected): %v", err)
			return
		}
		t.Errorf("unexpected error: %v", err)
		return
	}

	if devices == nil {
		t.Error("devices should not be nil when no error")
	}

	t.Logf("GetDevices returned %d devices", len(devices))
}

// TestDiscoveryServiceGetDevicesAfterDiscover tests GetDevices after a Discover call.
func TestDiscoveryServiceGetDevicesAfterDiscover(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		WithDiscoveryMethods(false, false, false).
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First discover
	_, err := service.Discover(ctx, nil)
	if err != nil {
		t.Logf("Discover returned error (may be expected): %v", err)
		return
	}

	// Then get devices
	devices, err := service.GetDevices(ctx)
	if err != nil {
		t.Logf("GetDevices returned error: %v", err)
		return
	}

	t.Logf("GetDevices after Discover returned %d devices", len(devices))
}

// ========== GetDevice Tests ==========

// TestDiscoveryServiceGetDeviceByMAC tests GetDevice with MAC address.
func TestDiscoveryServiceGetDeviceByMAC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mac  string
	}{
		{"standard_mac", "00:11:22:33:44:55"},
		{"uppercase_mac", "AA:BB:CC:DD:EE:FF"},
		{"mixed_case_mac", "Aa:Bb:Cc:Dd:Ee:Ff"},
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

			device, err := service.GetDevice(ctx, tt.mac)
			if err != nil {
				// Expected for nonexistent devices
				t.Logf("GetDevice by MAC returned error (expected): %v", err)
				return
			}

			if device == nil {
				t.Error("device should not be nil when no error")
			}
		})
	}
}

// TestDiscoveryServiceGetDeviceByIP tests GetDevice with IP address.
func TestDiscoveryServiceGetDeviceByIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
	}{
		{"localhost", "127.0.0.1"},
		{"private_ip", "192.168.1.100"},
		{"public_ip", "8.8.8.8"},
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

			device, err := service.GetDevice(ctx, tt.ip)
			if err != nil {
				// Expected for nonexistent devices
				t.Logf("GetDevice by IP returned error (expected): %v", err)
				return
			}

			if device == nil {
				t.Error("device should not be nil when no error")
			}
		})
	}
}

// TestDiscoveryServiceGetDeviceNotFound tests GetDevice with nonexistent device.
func TestDiscoveryServiceGetDeviceNotFound(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	ctx := context.Background()

	device, err := service.GetDevice(ctx, "nonexistent-device-id")
	if err == nil {
		t.Error("GetDevice should return error for nonexistent device")
	}
	if device != nil {
		t.Error("device should be nil when error is returned")
	}
}

// ========== Concurrent Discovery Tests ==========

// TestDiscoveryServiceConcurrentGetDevices tests concurrent GetDevices calls.
func TestDiscoveryServiceConcurrentGetDevices(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	ctx := context.Background()

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, _ = service.GetDevices(ctx)
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for GetDevices")
			return
		}
	}
}

// TestDiscoveryServiceConcurrentGetDevice tests concurrent GetDevice calls.
func TestDiscoveryServiceConcurrentGetDevice(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	ctx := context.Background()

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			_, _ = service.GetDevice(ctx, "192.168.1."+string(rune('0'+id)))
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for GetDevice")
			return
		}
	}
}

// ========== Discovery With Timeout Tests ==========

// TestDiscoveryServiceDiscoverWithShortTimeout tests Discover with very short timeout.
func TestDiscoveryServiceDiscoverWithShortTimeout(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		WithDiscoveryMethods(false, false, false).
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	// Very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	opts := &shell.DiscoveryOptions{
		Interface:   "lo",
		Timeout:     5 * time.Millisecond,
		Concurrency: 1,
	}

	result, err := service.Discover(ctx, opts)
	if err != nil {
		// Context deadline exceeded is expected
		t.Logf("Discover with short timeout returned error (expected): %v", err)
		return
	}

	if result != nil {
		t.Logf("Discover completed before timeout, found %d devices", len(result.Devices))
	}
}
