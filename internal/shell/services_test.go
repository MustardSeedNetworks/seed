package shell_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// ========== DiscoveryService Tests ==========

// TestDiscoveryServiceCreation tests DiscoveryService initialization.
func TestDiscoveryServiceCreation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupCfg func() *testutil.ConfigBuilder
	}{
		{
			name: "with_loopback_interface",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo")
			},
		},
		{
			name: "with_eth0_interface",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("eth0")
			},
		},
		{
			name: "with_discovery_methods_enabled",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo").
					WithDiscoveryMethods(true, true, true)
			},
		},
		{
			name: "with_discovery_methods_disabled",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo").
					WithDiscoveryMethods(false, false, false)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.setupCfg().Build()
			module := shell.New(cfg, nil)
			service := module.Discovery()

			if service == nil {
				t.Fatal("DiscoveryService should not be nil")
			}
		})
	}
}

// TestDiscoveryServiceStart tests DiscoveryService Start method.
func TestDiscoveryServiceStart(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start may fail on systems without proper network setup
	// We just verify it doesn't panic
	err := service.Start(ctx)
	if err != nil {
		t.Logf("Start returned error (may be expected): %v", err)
	}

	// Stop should always succeed
	service.Stop()
}

// TestDiscoveryServiceStop tests DiscoveryService Stop method.
func TestDiscoveryServiceStop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		startFirst bool
	}{
		{
			name:       "stop_without_start",
			startFirst: false,
		},
		{
			name:       "stop_after_start",
			startFirst: true,
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

			if tt.startFirst {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = service.Start(ctx)
				cancel()
			}

			// Stop should not panic
			service.Stop()
		})
	}
}

// TestDiscoveryServiceDiscover tests the Discover method.
func TestDiscoveryServiceDiscover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    *shell.DiscoveryOptions
		timeout time.Duration
	}{
		{
			name:    "with_nil_options",
			opts:    nil,
			timeout: 2 * time.Second,
		},
		{
			name: "with_minimal_options",
			opts: &shell.DiscoveryOptions{
				Interface:   "lo",
				Timeout:     500 * time.Millisecond,
				Concurrency: 1,
			},
			timeout: 2 * time.Second,
		},
		{
			name: "with_arp_enabled",
			opts: &shell.DiscoveryOptions{
				Interface:   "lo",
				EnableARP:   true,
				Timeout:     500 * time.Millisecond,
				Concurrency: 5,
			},
			timeout: 2 * time.Second,
		},
		{
			name: "with_icmp_enabled",
			opts: &shell.DiscoveryOptions{
				Interface:   "lo",
				EnableICMP:  true,
				Timeout:     500 * time.Millisecond,
				Concurrency: 5,
			},
			timeout: 2 * time.Second,
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

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			result, err := service.Discover(ctx, tt.opts)
			if err != nil {
				t.Logf("Discover returned error (may be expected): %v", err)
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result when no error")
			}

			// Verify result structure
			if result.StartedAt.IsZero() {
				t.Error("StartedAt should be set")
			}
			if result.CompletedAt.IsZero() {
				t.Error("CompletedAt should be set")
			}
			if result.ScanDuration < 0 {
				t.Error("ScanDuration should not be negative")
			}
		})
	}
}

// TestDiscoveryServiceGetDevices tests the GetDevices method.
func TestDiscoveryServiceGetDevices(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	ctx := context.Background()

	devices, err := service.GetDevices(ctx)
	if err != nil {
		// May return ErrNotInitialized
		if !errors.Is(err, shell.ErrNotInitialized) {
			t.Errorf("unexpected error: %v", err)
		}
		return
	}

	// If no error, devices should be a valid slice (possibly empty)
	if devices == nil {
		t.Error("devices should not be nil when no error")
	}
}

// TestDiscoveryServiceGetDevice tests the GetDevice method.
func TestDiscoveryServiceGetDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		deviceID string
	}{
		{
			name:     "get_by_mac",
			deviceID: "00:11:22:33:44:55",
		},
		{
			name:     "get_by_ip",
			deviceID: "192.168.1.100",
		},
		{
			name:     "get_nonexistent",
			deviceID: "nonexistent-device",
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

			ctx := context.Background()

			device, err := service.GetDevice(ctx, tt.deviceID)
			if err != nil {
				// Expected for nonexistent devices or uninitialized service
				t.Logf("GetDevice returned error (may be expected): %v", err)
				return
			}

			if device == nil {
				t.Error("device should not be nil when no error")
			}
		})
	}
}

// TestDiscoveryServiceAccessors tests the underlying service accessors.
func TestDiscoveryServiceAccessors(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()

	// Test Service() accessor
	underlyingService := service.Service()
	if underlyingService == nil {
		t.Error("Service() should return the underlying discovery.Service")
	}

	// Test DeviceDiscovery() accessor
	deviceDiscovery := service.DeviceDiscovery()
	if deviceDiscovery == nil {
		t.Error("DeviceDiscovery() should return the underlying DeviceDiscovery")
	}
}

// ========== VulnerabilityService Tests ==========

// TestVulnerabilityServiceCreation tests VulnerabilityService initialization.
func TestVulnerabilityServiceCreation(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Vulnerability()

	if service == nil {
		t.Fatal("VulnerabilityService should not be nil")
	}
}

// TestVulnerabilityServiceStop tests VulnerabilityService Stop method.
func TestVulnerabilityServiceStop(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Vulnerability()

	// Stop should not panic even without starting
	service.Stop()

	// Stop multiple times should be safe
	service.Stop()
}

// TestVulnerabilityServiceScan tests the Scan method.
func TestVulnerabilityServiceScan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		targets []string
	}{
		{
			name:    "empty_targets",
			targets: []string{},
		},
		{
			name:    "single_target",
			targets: []string{"192.168.1.1"},
		},
		{
			name:    "multiple_targets",
			targets: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
		},
		{
			name:    "localhost_target",
			targets: []string{"127.0.0.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				Build()

			module := shell.New(cfg, nil)
			service := module.Vulnerability()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := service.Scan(ctx, tt.targets)
			if err != nil {
				// May return ErrNotInitialized if scanner failed to initialize
				if !errors.Is(err, shell.ErrNotInitialized) {
					t.Logf("Scan returned error (may be expected): %v", err)
				}
				return
			}

			if result == nil {
				t.Fatal("result should not be nil when no error")
			}

			// Verify result structure
			if result.ID == "" {
				t.Error("ID should be set")
			}
			if result.DevicesScanned != len(tt.targets) {
				t.Errorf("DevicesScanned = %d, want %d", result.DevicesScanned, len(tt.targets))
			}
			if result.StartedAt.IsZero() {
				t.Error("StartedAt should be set")
			}
			if result.CompletedAt.IsZero() {
				t.Error("CompletedAt should be set")
			}
		})
	}
}

// TestVulnerabilityServiceGetVulnerabilities tests the GetVulnerabilities method.
func TestVulnerabilityServiceGetVulnerabilities(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Vulnerability()

	ctx := context.Background()

	vulns, err := service.GetVulnerabilities(ctx)
	if err != nil {
		if !errors.Is(err, shell.ErrNotInitialized) {
			t.Errorf("unexpected error: %v", err)
		}
		return
	}

	// If no error, vulns should be a valid slice (possibly empty)
	if vulns == nil {
		t.Logf("vulnerabilities is nil (empty is acceptable)")
	}
}

// TestVulnerabilityServiceGetDeviceVulnerabilities tests the GetDeviceVulnerabilities method.
func TestVulnerabilityServiceGetDeviceVulnerabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		deviceIP string
	}{
		{
			name:     "localhost",
			deviceIP: "127.0.0.1",
		},
		{
			name:     "private_ip",
			deviceIP: "192.168.1.100",
		},
		{
			name:     "nonexistent_device",
			deviceIP: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				Build()

			module := shell.New(cfg, nil)
			service := module.Vulnerability()

			ctx := context.Background()

			vulns, err := service.GetDeviceVulnerabilities(ctx, tt.deviceIP)
			if err != nil {
				if !errors.Is(err, shell.ErrNotInitialized) {
					t.Logf("GetDeviceVulnerabilities returned error (may be expected): %v", err)
				}
				return
			}

			// vulns may be nil or empty, which is acceptable
			_ = vulns
		})
	}
}

// TestVulnerabilityServiceUpdateStatus tests the UpdateStatus method.
func TestVulnerabilityServiceUpdateStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		vulnID  string
		status  shell.VulnStatus
		wantErr error
	}{
		{
			name:    "update_to_acknowledged",
			vulnID:  "vuln-001",
			status:  shell.VulnStatusAcknowledged,
			wantErr: shell.ErrNotImplemented,
		},
		{
			name:    "update_to_in_progress",
			vulnID:  "vuln-002",
			status:  shell.VulnStatusInProgress,
			wantErr: shell.ErrNotImplemented,
		},
		{
			name:    "update_to_resolved",
			vulnID:  "vuln-003",
			status:  shell.VulnStatusResolved,
			wantErr: shell.ErrNotImplemented,
		},
		{
			name:    "update_to_false_positive",
			vulnID:  "vuln-004",
			status:  shell.VulnStatusFalsePositive,
			wantErr: shell.ErrNotImplemented,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				Build()

			module := shell.New(cfg, nil)
			service := module.Vulnerability()

			ctx := context.Background()

			err := service.UpdateStatus(ctx, tt.vulnID, tt.status)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("UpdateStatus() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestVulnerabilityServiceScanner tests the Scanner accessor.
func TestVulnerabilityServiceScanner(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Vulnerability()

	// Scanner may be nil if initialization failed
	// Just verify the accessor doesn't panic
	_ = service.Scanner()
}

// ========== PostureService Tests ==========

// TestPostureServiceCreation tests PostureService initialization.
func TestPostureServiceCreation(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	if service == nil {
		t.Fatal("PostureService should not be nil")
	}
}

// TestPostureServiceAssess tests the Assess method.
func TestPostureServiceAssess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "assess_with_minimal_config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testutil.NewConfigBuilder().
				WithInterface("lo").
				Build()

			module := shell.New(cfg, nil)
			service := module.Posture()

			ctx := context.Background()

			score, err := service.Assess(ctx)
			if err != nil {
				t.Errorf("Assess() returned error: %v", err)
				return
			}

			if score == nil {
				t.Fatal("score should not be nil")
			}

			// Verify score structure
			if score.Overall < 0 || score.Overall > 100 {
				t.Errorf("Overall score %d out of range [0, 100]", score.Overall)
			}
			if score.Categories == nil {
				t.Error("Categories should be initialized")
			}
			if score.Issues == nil {
				t.Error("Issues should be initialized")
			}
			if score.AssessedAt.IsZero() {
				t.Error("AssessedAt should be set")
			}
		})
	}
}

// TestPostureServiceAssessScore tests that Assess returns reasonable scores.
func TestPostureServiceAssessScore(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()

	ctx := context.Background()

	score, err := service.Assess(ctx)
	if err != nil {
		t.Fatalf("Assess() returned error: %v", err)
	}

	// Verify the perfect score baseline is used correctly
	if score.Overall > shell.ExportPerfectSecurityScore {
		t.Errorf(
			"Overall score %d exceeds perfect score %d",
			score.Overall,
			shell.ExportPerfectSecurityScore,
		)
	}
}

// TestPostureServiceTestAccessor tests the PostureServiceTestAccessor.
func TestPostureServiceTestAccessor(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Posture()
	accessor := shell.PostureServiceTestAccessor{Service: service}

	if accessor.GetCfg() == nil {
		t.Error("GetCfg() should not return nil")
	}

	if accessor.GetDiscovery() == nil {
		t.Error("GetDiscovery() should not return nil")
	}

	if accessor.GetVulnerability() == nil {
		t.Error("GetVulnerability() should not return nil")
	}
}

// ========== RogueService Tests ==========

// TestRogueServiceCreation tests RogueService initialization.
func TestRogueServiceCreation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupCfg func() *testutil.ConfigBuilder
	}{
		{
			name: "with_loopback_interface",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("lo")
			},
		},
		{
			name: "with_eth0_interface",
			setupCfg: func() *testutil.ConfigBuilder {
				return testutil.NewConfigBuilder().
					WithInterface("eth0")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.setupCfg().Build()
			module := shell.New(cfg, nil)
			service := module.Rogue()

			if service == nil {
				t.Fatal("RogueService should not be nil")
			}
		})
	}
}

// TestRogueServiceStart tests RogueService Start method.
func TestRogueServiceStart(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start may fail on systems without CAP_NET_RAW or root
	err := service.Start(ctx)
	if err != nil {
		t.Logf("Start returned error (may be expected without privileges): %v", err)
	}

	// Stop should always succeed
	service.Stop()
}

// TestRogueServiceStop tests RogueService Stop method.
func TestRogueServiceStop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		startFirst bool
	}{
		{
			name:       "stop_without_start",
			startFirst: false,
		},
		{
			name:       "stop_after_start_attempt",
			startFirst: true,
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

			if tt.startFirst {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				_ = service.Start(ctx)
				cancel()
			}

			// Stop should not panic
			service.Stop()
		})
	}
}

// TestRogueServiceGetRogueDevices tests the GetRogueDevices method.
func TestRogueServiceGetRogueDevices(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	ctx := context.Background()

	devices, err := service.GetRogueDevices(ctx)
	if err != nil {
		if !errors.Is(err, shell.ErrNotInitialized) {
			t.Errorf("unexpected error: %v", err)
		}
		return
	}

	// If no error, devices should be a valid slice (possibly empty)
	if devices == nil {
		t.Error("devices should not be nil when no error")
	}
}

// TestRogueServiceGetAlerts tests the GetAlerts method.
func TestRogueServiceGetAlerts(t *testing.T) {
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

	// If no error, alerts should be a valid slice (possibly empty)
	if alerts == nil {
		t.Error("alerts should not be nil when no error")
	}
}

// TestRogueServiceAcknowledgeDevice tests the AcknowledgeDevice method.
func TestRogueServiceAcknowledgeDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		deviceID string
		wantErr  error
	}{
		{
			name:     "acknowledge_valid_device",
			deviceID: "192.168.1.100",
			wantErr:  shell.ErrNotImplemented,
		},
		{
			name:     "acknowledge_by_mac",
			deviceID: "00:11:22:33:44:55",
			wantErr:  shell.ErrNotImplemented,
		},
		{
			name:     "acknowledge_nonexistent",
			deviceID: "nonexistent",
			wantErr:  shell.ErrNotImplemented,
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
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AcknowledgeDevice() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestRogueServiceDetector tests the Detector accessor.
func TestRogueServiceDetector(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()

	detector := service.Detector()
	if detector == nil {
		t.Error("Detector() should return the underlying dhcp.RogueDetector")
	}
}

// TestRogueServiceTestAccessor tests the RogueServiceTestAccessor.
func TestRogueServiceTestAccessor(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Rogue()
	accessor := shell.RogueServiceTestAccessor{Service: service}

	if accessor.GetCfg() == nil {
		t.Error("GetCfg() should not return nil")
	}
}

// TestDiscoveryServiceTestAccessor tests the DiscoveryServiceTestAccessor.
func TestDiscoveryServiceTestAccessor(t *testing.T) {
	t.Parallel()

	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	service := module.Discovery()
	accessor := shell.DiscoveryServiceTestAccessor{Service: service}

	if accessor.GetCfg() == nil {
		t.Error("GetCfg() should not return nil")
	}

	// DB may be nil in test configuration
	_ = accessor.GetDB()
}
