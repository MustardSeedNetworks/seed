package shell_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// ========== Exported Constants Tests ==========

// TestExportedConstants tests exported constants from export_test.go.
func TestExportedConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		minValue int
		maxValue int
	}{
		{"PerfectSecurityScore", shell.ExportPerfectSecurityScore, 100, 100},
		{"VulnerabilityPenaltyMultiplier", shell.ExportVulnerabilityPenaltyMultiplier, 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value < tt.minValue || tt.value > tt.maxValue {
				t.Errorf("%s = %d, want between %d and %d", tt.name, tt.value, tt.minValue, tt.maxValue)
			}
		})
	}
}

// ========== Conversion Function Tests ==========

// TestConvertDiscoveredDevice tests the convertDiscoveredDevice helper function.
func TestConvertDiscoveredDevice(t *testing.T) {
	tests := []struct {
		name           string
		input          *discovery.DiscoveredDevice
		wantMAC        string
		wantHostname   string
		wantVendor     string
		wantIsGateway  bool
		wantDeviceType shell.DeviceType
	}{
		{
			name: "basic_device",
			input: &discovery.DiscoveredDevice{
				IP:       "192.168.1.100",
				MAC:      "00:11:22:33:44:55",
				Hostname: "test-host",
				Vendor:   "Test Vendor",
				IsRouter: false,
				LastSeen: time.Now(),
			},
			wantMAC:        "00:11:22:33:44:55",
			wantHostname:   "test-host",
			wantVendor:     "Test Vendor",
			wantIsGateway:  false,
			wantDeviceType: shell.DeviceTypeUnknown,
		},
		{
			name: "router_device",
			input: &discovery.DiscoveredDevice{
				IP:       "192.168.1.1",
				MAC:      "aa:bb:cc:dd:ee:ff",
				Hostname: "router",
				Vendor:   "Cisco",
				IsRouter: true,
				LastSeen: time.Now(),
			},
			wantMAC:        "aa:bb:cc:dd:ee:ff",
			wantHostname:   "router",
			wantVendor:     "Cisco",
			wantIsGateway:  true,
			wantDeviceType: shell.DeviceTypeRouter,
		},
		{
			name: "device_with_profile",
			input: &discovery.DiscoveredDevice{
				IP:       "192.168.1.50",
				MAC:      "11:22:33:44:55:66",
				Hostname: "server",
				Vendor:   "Dell",
				IsRouter: false,
				LastSeen: time.Now(),
				Profile: &discovery.DeviceProfile{
					DeviceType: "server",
					OpenPorts: []discovery.OpenPort{
						{Port: 22, Protocol: "tcp", Service: "ssh", Banner: "SSH-2.0", IsOpen: true},
						{Port: 80, Protocol: "tcp", Service: "http", IsOpen: true},
						{Port: 443, Protocol: "tcp", Service: "https", IsOpen: true},
					},
				},
			},
			wantMAC:        "11:22:33:44:55:66",
			wantHostname:   "server",
			wantVendor:     "Dell",
			wantIsGateway:  false,
			wantDeviceType: shell.DeviceType("server"),
		},
		{
			name: "device_with_many_ports_inferred_as_server",
			input: &discovery.DiscoveredDevice{
				IP:       "192.168.1.60",
				MAC:      "22:33:44:55:66:77",
				Hostname: "multi-service",
				Vendor:   "HP",
				IsRouter: false,
				LastSeen: time.Now(),
				Profile: &discovery.DeviceProfile{
					OpenPorts: []discovery.OpenPort{
						{Port: 22, Protocol: "tcp", Service: "ssh", IsOpen: true},
						{Port: 80, Protocol: "tcp", Service: "http", IsOpen: true},
						{Port: 443, Protocol: "tcp", Service: "https", IsOpen: true},
						{Port: 3306, Protocol: "tcp", Service: "mysql", IsOpen: true},
						{Port: 5432, Protocol: "tcp", Service: "postgresql", IsOpen: true},
						{Port: 6379, Protocol: "tcp", Service: "redis", IsOpen: true},
					},
				},
			},
			wantMAC:        "22:33:44:55:66:77",
			wantHostname:   "multi-service",
			wantVendor:     "HP",
			wantIsGateway:  false,
			wantDeviceType: shell.DeviceTypeServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shell.ExportConvertDiscoveredDevice(tt.input)

			if result.MACAddress != tt.wantMAC {
				t.Errorf("MACAddress = %v, want %v", result.MACAddress, tt.wantMAC)
			}
			if result.Hostname != tt.wantHostname {
				t.Errorf("Hostname = %v, want %v", result.Hostname, tt.wantHostname)
			}
			if result.Vendor != tt.wantVendor {
				t.Errorf("Vendor = %v, want %v", result.Vendor, tt.wantVendor)
			}
			if result.IsGateway != tt.wantIsGateway {
				t.Errorf("IsGateway = %v, want %v", result.IsGateway, tt.wantIsGateway)
			}
			if result.DeviceType != tt.wantDeviceType {
				t.Errorf("DeviceType = %v, want %v", result.DeviceType, tt.wantDeviceType)
			}
			// All converted devices should be online
			if !result.IsOnline {
				t.Error("IsOnline should be true for converted devices")
			}
			// ID should be set to MAC
			if result.ID != tt.wantMAC {
				t.Errorf("ID = %v, want %v (should match MAC)", result.ID, tt.wantMAC)
			}
			// Metadata should be initialized
			if result.Metadata == nil {
				t.Error("Metadata should be initialized")
			}
		})
	}
}

// TestConvertDiscoveredDeviceServices tests that services are properly extracted from profile.
func TestConvertDiscoveredDeviceServices(t *testing.T) {
	input := &discovery.DiscoveredDevice{
		IP:       "192.168.1.100",
		MAC:      "00:11:22:33:44:55",
		LastSeen: time.Now(),
		Profile: &discovery.DeviceProfile{
			OpenPorts: []discovery.OpenPort{
				{Port: 22, Protocol: "tcp", Service: "ssh", Banner: "SSH-2.0-OpenSSH", IsOpen: true},
				{Port: 80, Protocol: "tcp", Service: "http", IsOpen: true},
			},
		},
	}

	result := shell.ExportConvertDiscoveredDevice(input)

	if len(result.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(result.Services))
		return
	}

	// Check first service (SSH)
	ssh := result.Services[0]
	if ssh.Port != 22 {
		t.Errorf("SSH port = %d, want 22", ssh.Port)
	}
	if ssh.Protocol != "tcp" {
		t.Errorf("SSH protocol = %s, want tcp", ssh.Protocol)
	}
	if ssh.Name != "ssh" {
		t.Errorf("SSH name = %s, want ssh", ssh.Name)
	}
	if ssh.Banner != "SSH-2.0-OpenSSH" {
		t.Errorf("SSH banner = %s, want SSH-2.0-OpenSSH", ssh.Banner)
	}
	if ssh.State != "open" {
		t.Errorf("SSH state = %s, want open", ssh.State)
	}
}

// TestConvertVulnerability tests the convertVulnerability helper function.
func TestConvertVulnerability(t *testing.T) {
	tests := []struct {
		name         string
		input        *discovery.Vulnerability
		wantCVEID    string
		wantSeverity shell.VulnSeverity
		wantIsKEV    bool
		wantStatus   shell.VulnStatus
	}{
		{
			name: "critical_vulnerability",
			input: &discovery.Vulnerability{
				CVEID:             "CVE-2024-12345",
				Description:       "Critical vulnerability",
				Severity:          "CRITICAL",
				Score:             9.8,
				ActivelyExploited: true,
				References:        []string{"https://example.com"},
			},
			wantCVEID:    "CVE-2024-12345",
			wantSeverity: shell.SeverityCritical,
			wantIsKEV:    true,
			wantStatus:   shell.VulnStatusNew,
		},
		{
			name: "high_vulnerability",
			input: &discovery.Vulnerability{
				CVEID:             "CVE-2024-67890",
				Description:       "High severity issue",
				Severity:          "HIGH",
				Score:             7.5,
				ActivelyExploited: false,
			},
			wantCVEID:    "CVE-2024-67890",
			wantSeverity: shell.SeverityHigh,
			wantIsKEV:    false,
			wantStatus:   shell.VulnStatusNew,
		},
		{
			name: "medium_vulnerability",
			input: &discovery.Vulnerability{
				CVEID:       "CVE-2024-11111",
				Description: "Medium severity issue",
				Severity:    "MEDIUM",
				Score:       5.0,
			},
			wantCVEID:    "CVE-2024-11111",
			wantSeverity: shell.SeverityMedium,
			wantIsKEV:    false,
			wantStatus:   shell.VulnStatusNew,
		},
		{
			name: "low_vulnerability",
			input: &discovery.Vulnerability{
				CVEID:       "CVE-2024-22222",
				Description: "Low severity issue",
				Severity:    "LOW",
				Score:       2.0,
			},
			wantCVEID:    "CVE-2024-22222",
			wantSeverity: shell.SeverityLow,
			wantIsKEV:    false,
			wantStatus:   shell.VulnStatusNew,
		},
		{
			name: "unknown_severity_defaults_to_info",
			input: &discovery.Vulnerability{
				CVEID:       "CVE-2024-33333",
				Description: "Unknown severity",
				Severity:    "UNKNOWN",
				Score:       0.0,
			},
			wantCVEID:    "CVE-2024-33333",
			wantSeverity: shell.SeverityInfo,
			wantIsKEV:    false,
			wantStatus:   shell.VulnStatusNew,
		},
		{
			name: "empty_severity_defaults_to_info",
			input: &discovery.Vulnerability{
				CVEID:       "CVE-2024-44444",
				Description: "Empty severity",
				Severity:    "",
				Score:       0.0,
			},
			wantCVEID:    "CVE-2024-44444",
			wantSeverity: shell.SeverityInfo,
			wantIsKEV:    false,
			wantStatus:   shell.VulnStatusNew,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shell.ExportConvertVulnerability(tt.input)

			if result.CVEID != tt.wantCVEID {
				t.Errorf("CVEID = %v, want %v", result.CVEID, tt.wantCVEID)
			}
			if result.Severity != tt.wantSeverity {
				t.Errorf("Severity = %v, want %v", result.Severity, tt.wantSeverity)
			}
			if result.IsKEV != tt.wantIsKEV {
				t.Errorf("IsKEV = %v, want %v", result.IsKEV, tt.wantIsKEV)
			}
			if result.IsExploited != tt.wantIsKEV {
				t.Errorf("IsExploited = %v, want %v", result.IsExploited, tt.wantIsKEV)
			}
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}
			// ID should match CVEID
			if result.ID != tt.wantCVEID {
				t.Errorf("ID = %v, want %v (should match CVEID)", result.ID, tt.wantCVEID)
			}
			// Score should be preserved
			if result.CVSSScore != tt.input.Score {
				t.Errorf("CVSSScore = %v, want %v", result.CVSSScore, tt.input.Score)
			}
			// Description should be used for both Title and Description
			if result.Title != tt.input.Description {
				t.Errorf("Title = %v, want %v", result.Title, tt.input.Description)
			}
			if result.Description != tt.input.Description {
				t.Errorf("Description = %v, want %v", result.Description, tt.input.Description)
			}
		})
	}
}

// TestConvertVulnerabilityReferences tests that references are properly copied.
func TestConvertVulnerabilityReferences(t *testing.T) {
	refs := []string{"https://nvd.nist.gov/vuln/1", "https://example.com/advisory"}
	input := &discovery.Vulnerability{
		CVEID:      "CVE-2024-99999",
		Severity:   "HIGH",
		References: refs,
	}

	result := shell.ExportConvertVulnerability(input)

	if len(result.References) != len(refs) {
		t.Errorf("expected %d references, got %d", len(refs), len(result.References))
		return
	}

	for i, ref := range refs {
		if result.References[i] != ref {
			t.Errorf("Reference[%d] = %s, want %s", i, result.References[i], ref)
		}
	}
}

// ========== VulnerabilityService Test Accessor Tests ==========

// TestVulnerabilityServiceTestAccessorFields tests the VulnerabilityServiceTestAccessor.
func TestVulnerabilityServiceTestAccessorFields(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	vulnService := module.Vulnerability()
	accessor := shell.VulnerabilityServiceTestAccessor{Service: vulnService}

	// Test config accessor
	if accessor.GetCfg() == nil {
		t.Error("GetCfg() returned nil")
	}

	// Test DB accessor (may be nil)
	// No assertion - DB may be nil when not initialized
	_ = accessor.GetDB()

	// Scanner may be nil if initialization failed
	// No assertion - just verifying it doesn't panic
	_ = accessor.GetScanner()
}

// ========== Discovery Service Additional Tests ==========

// TestDiscoveryServiceWithTestutil tests DiscoveryService with testutil config builder.
func TestDiscoveryServiceWithTestutil(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		WithDiscoveryMethods(false, false, false). // Disable all for faster test
		Build()

	module := shell.New(cfg, nil)
	discoveryService := module.Discovery()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Discover with minimal options
	opts := &shell.DiscoveryOptions{
		Interface:   "lo",
		EnableARP:   false,
		EnableICMP:  false,
		Timeout:     500 * time.Millisecond,
		Concurrency: 1,
	}

	result, err := discoveryService.Discover(ctx, opts)
	if err != nil {
		t.Logf("Discover returned error (may be expected): %v", err)
		return
	}

	if result == nil {
		t.Fatal("Discover returned nil result")
	}

	// Verify timing fields
	if result.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
	if result.CompletedAt.IsZero() {
		t.Error("CompletedAt should not be zero")
	}
	if result.ScanDuration <= 0 {
		t.Error("ScanDuration should be positive")
	}

	t.Logf("Discovery completed in %v, found %d devices", result.ScanDuration, len(result.Devices))
}

// ========== Rogue Service Additional Tests ==========

// TestRogueServiceWithTestutil tests RogueService with testutil config builder.
func TestRogueServiceWithTestutil(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	rogueService := module.Rogue()

	ctx := context.Background()

	// Start may fail on test system without proper network interface
	err := rogueService.Start(ctx)
	if err != nil {
		t.Logf("Start returned error (may be expected on test system): %v", err)
	}

	// Stop should always succeed
	rogueService.Stop()
}

// ========== Multiple Module Instance Tests ==========

// TestMultipleModuleInstancesIndependence tests that multiple modules are independent.
func TestMultipleModuleInstancesIndependence(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	// Create multiple instances
	module1 := shell.New(cfg, nil)
	module2 := shell.New(cfg, nil)

	if module1 == nil || module2 == nil {
		t.Fatal("Failed to create module instances")
	}

	// Get services from both modules
	disc1 := module1.Discovery()
	disc2 := module2.Discovery()

	// Services should be different instances
	if disc1 == disc2 {
		t.Error("Discovery services should be different instances")
	}

	vuln1 := module1.Vulnerability()
	vuln2 := module2.Vulnerability()

	if vuln1 == vuln2 {
		t.Error("Vulnerability services should be different instances")
	}

	posture1 := module1.Posture()
	posture2 := module2.Posture()

	if posture1 == posture2 {
		t.Error("Posture services should be different instances")
	}

	rogue1 := module1.Rogue()
	rogue2 := module2.Rogue()

	if rogue1 == rogue2 {
		t.Error("Rogue services should be different instances")
	}
}

// ========== Error Constants Tests ==========

// TestErrorConstants tests that error constants are properly defined.
func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantMessage string
	}{
		{
			name:        "ErrNotImplemented",
			err:         shell.ErrNotImplemented,
			wantMessage: "not implemented: pending migration",
		},
		{
			name:        "ErrNotInitialized",
			err:         shell.ErrNotInitialized,
			wantMessage: "service not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
				return
			}
			if tt.err.Error() != tt.wantMessage {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.wantMessage)
			}
		})
	}
}
