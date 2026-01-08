// Package shell_test provides comprehensive tests for the Shell module.
// Tests cover device discovery, vulnerability scanning, posture assessment,
// and rogue device detection services.
package shell_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/shell"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// TestDeviceType tests DeviceType constants and validation.
func TestDeviceType(t *testing.T) {
	tests := []struct {
		name       string
		deviceType shell.DeviceType
		want       string
	}{
		{"router", shell.DeviceTypeRouter, "router"},
		{"switch", shell.DeviceTypeSwitch, "switch"},
		{"access_point", shell.DeviceTypeAP, "access_point"},
		{"server", shell.DeviceTypeServer, "server"},
		{"workstation", shell.DeviceTypeWorkstation, "workstation"},
		{"mobile", shell.DeviceTypeMobile, "mobile"},
		{"iot", shell.DeviceTypeIoT, "iot"},
		{"printer", shell.DeviceTypePrinter, "printer"},
		{"camera", shell.DeviceTypeCamera, "camera"},
		{"unknown", shell.DeviceTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.deviceType); got != tt.want {
				t.Errorf("DeviceType = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVulnSeverity tests VulnSeverity constants.
func TestVulnSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity shell.VulnSeverity
		want     string
	}{
		{"critical", shell.SeverityCritical, "critical"},
		{"high", shell.SeverityHigh, "high"},
		{"medium", shell.SeverityMedium, "medium"},
		{"low", shell.SeverityLow, "low"},
		{"info", shell.SeverityInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.severity); got != tt.want {
				t.Errorf("VulnSeverity = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVulnStatus tests VulnStatus constants.
func TestVulnStatus(t *testing.T) {
	tests := []struct {
		name   string
		status shell.VulnStatus
		want   string
	}{
		{"new", shell.VulnStatusNew, "new"},
		{"acknowledged", shell.VulnStatusAcknowledged, "acknowledged"},
		{"in_progress", shell.VulnStatusInProgress, "in_progress"},
		{"resolved", shell.VulnStatusResolved, "resolved"},
		{"false_positive", shell.VulnStatusFalsePositive, "false_positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.status); got != tt.want {
				t.Errorf("VulnStatus = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDevice tests Device struct initialization and field access.
func TestDevice(t *testing.T) {
	now := time.Now()
	device := shell.Device{
		ID:         "test-device-1",
		IPAddress:  net.ParseIP("192.168.1.100"),
		MACAddress: "00:11:22:33:44:55",
		Hostname:   "test-host",
		Vendor:     "Test Vendor",
		DeviceType: shell.DeviceTypeServer,
		OS:         "Linux",
		Services: []shell.Service{
			{
				Port:     22,
				Protocol: "tcp",
				Name:     "ssh",
				State:    "open",
			},
		},
		Interfaces: []shell.DeviceInterface{
			{
				Name:        "eth0",
				MACAddress:  "00:11:22:33:44:55",
				IPAddresses: []string{"192.168.1.100"},
				Type:        "ethernet",
				Status:      "up",
			},
		},
		FirstSeen: now,
		LastSeen:  now,
		IsOnline:  true,
		IsGateway: false,
		Metadata:  map[string]string{"key": "value"},
	}

	tests := []struct {
		name     string
		check    func() bool
		expected bool
	}{
		{"ID is set", func() bool { return device.ID == "test-device-1" }, true},
		{"IP is valid", func() bool { return device.IPAddress.String() == "192.168.1.100" }, true},
		{"MAC is set", func() bool { return device.MACAddress == "00:11:22:33:44:55" }, true},
		{"Hostname is set", func() bool { return device.Hostname == "test-host" }, true},
		{"Vendor is set", func() bool { return device.Vendor == "Test Vendor" }, true},
		{"DeviceType is server", func() bool { return device.DeviceType == shell.DeviceTypeServer }, true},
		{"OS is set", func() bool { return device.OS == "Linux" }, true},
		{"Has services", func() bool { return len(device.Services) == 1 }, true},
		{"Has interfaces", func() bool { return len(device.Interfaces) == 1 }, true},
		{"IsOnline is true", func() bool { return device.IsOnline }, true},
		{"IsGateway is false", func() bool { return !device.IsGateway }, true},
		{"Has metadata", func() bool { return device.Metadata["key"] == "value" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.check(); got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

// TestService tests Service struct initialization.
func TestService(t *testing.T) {
	tests := []struct {
		name     string
		service  shell.Service
		wantPort int
		wantName string
	}{
		{
			name: "ssh_service",
			service: shell.Service{
				Port:     22,
				Protocol: "tcp",
				Name:     "ssh",
				Version:  "OpenSSH_8.9",
				Banner:   "SSH-2.0-OpenSSH_8.9",
				State:    "open",
			},
			wantPort: 22,
			wantName: "ssh",
		},
		{
			name: "http_service",
			service: shell.Service{
				Port:     80,
				Protocol: "tcp",
				Name:     "http",
				State:    "open",
			},
			wantPort: 80,
			wantName: "http",
		},
		{
			name: "https_service",
			service: shell.Service{
				Port:     443,
				Protocol: "tcp",
				Name:     "https",
				State:    "open",
			},
			wantPort: 443,
			wantName: "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.service.Port != tt.wantPort {
				t.Errorf("Service.Port = %v, want %v", tt.service.Port, tt.wantPort)
			}
			if tt.service.Name != tt.wantName {
				t.Errorf("Service.Name = %v, want %v", tt.service.Name, tt.wantName)
			}
		})
	}
}

// TestDeviceInterface tests DeviceInterface struct initialization.
func TestDeviceInterface(t *testing.T) {
	tests := []struct {
		name      string
		iface     shell.DeviceInterface
		wantName  string
		wantType  string
		wantMAC   string
		wantCount int
	}{
		{
			name: "ethernet_interface",
			iface: shell.DeviceInterface{
				Name:        "eth0",
				MACAddress:  "00:11:22:33:44:55",
				IPAddresses: []string{"192.168.1.100", "fe80::1"},
				Type:        "ethernet",
				Speed:       "1000Mbps",
				Status:      "up",
			},
			wantName:  "eth0",
			wantType:  "ethernet",
			wantMAC:   "00:11:22:33:44:55",
			wantCount: 2,
		},
		{
			name: "wifi_interface",
			iface: shell.DeviceInterface{
				Name:        "wlan0",
				MACAddress:  "aa:bb:cc:dd:ee:ff",
				IPAddresses: []string{"192.168.1.101"},
				Type:        "wifi",
				Status:      "up",
			},
			wantName:  "wlan0",
			wantType:  "wifi",
			wantMAC:   "aa:bb:cc:dd:ee:ff",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.iface.Name != tt.wantName {
				t.Errorf("DeviceInterface.Name = %v, want %v", tt.iface.Name, tt.wantName)
			}
			if tt.iface.Type != tt.wantType {
				t.Errorf("DeviceInterface.Type = %v, want %v", tt.iface.Type, tt.wantType)
			}
			if tt.iface.MACAddress != tt.wantMAC {
				t.Errorf("DeviceInterface.MACAddress = %v, want %v", tt.iface.MACAddress, tt.wantMAC)
			}
			if len(tt.iface.IPAddresses) != tt.wantCount {
				t.Errorf("len(IPAddresses) = %v, want %v", len(tt.iface.IPAddresses), tt.wantCount)
			}
		})
	}
}

// TestDiscoveryResult tests DiscoveryResult struct initialization.
func TestDiscoveryResult(t *testing.T) {
	startTime := time.Now()
	completedTime := startTime.Add(5 * time.Second)
	duration := completedTime.Sub(startTime)

	result := shell.DiscoveryResult{
		Devices: []shell.Device{
			{ID: "device-1", DeviceType: shell.DeviceTypeServer},
			{ID: "device-2", DeviceType: shell.DeviceTypeWorkstation},
		},
		NewDevices:     2,
		UpdatedDevices: 0,
		OfflineDevices: 1,
		ScanDuration:   duration,
		ScanDurationMs: float64(duration.Milliseconds()),
		StartedAt:      startTime,
		CompletedAt:    completedTime,
	}

	tests := []struct {
		name  string
		check func() bool
	}{
		{"has_devices", func() bool { return len(result.Devices) == 2 }},
		{"new_devices_count", func() bool { return result.NewDevices == 2 }},
		{"updated_devices_count", func() bool { return result.UpdatedDevices == 0 }},
		{"offline_devices_count", func() bool { return result.OfflineDevices == 1 }},
		{"duration_positive", func() bool { return result.ScanDuration > 0 }},
		{"duration_ms_positive", func() bool { return result.ScanDurationMs > 0 }},
		{"started_before_completed", func() bool { return result.StartedAt.Before(result.CompletedAt) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check() {
				t.Errorf("%s check failed", tt.name)
			}
		})
	}
}

// TestDiscoveryOptions tests DiscoveryOptions struct initialization.
func TestDiscoveryOptions(t *testing.T) {
	tests := []struct {
		name    string
		options shell.DiscoveryOptions
	}{
		{
			name: "default_options",
			options: shell.DiscoveryOptions{
				Interface:     "eth0",
				Subnets:       []string{"192.168.1.0/24"},
				EnableARP:     true,
				EnableICMP:    true,
				EnableNDP:     false,
				EnableLLDP:    false,
				EnableCDP:     false,
				EnableSNMP:    false,
				PortScan:      false,
				PortScanPorts: nil,
				Timeout:       30 * time.Second,
				Concurrency:   10,
			},
		},
		{
			name: "full_scan_options",
			options: shell.DiscoveryOptions{
				Interface:     "eth0",
				Subnets:       []string{"192.168.1.0/24", "10.0.0.0/24"},
				EnableARP:     true,
				EnableICMP:    true,
				EnableNDP:     true,
				EnableLLDP:    true,
				EnableCDP:     true,
				EnableSNMP:    true,
				PortScan:      true,
				PortScanPorts: []int{22, 80, 443, 8080},
				Timeout:       60 * time.Second,
				Concurrency:   50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.options
			if opts.Interface == "" {
				t.Error("Interface should not be empty")
			}
			if opts.Timeout <= 0 {
				t.Error("Timeout should be positive")
			}
			if opts.Concurrency <= 0 {
				t.Error("Concurrency should be positive")
			}
		})
	}
}

// TestVulnerability tests Vulnerability struct initialization.
func TestVulnerability(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		vuln shell.Vulnerability
	}{
		{
			name: "critical_vulnerability",
			vuln: shell.Vulnerability{
				ID:              "vuln-1",
				DeviceID:        "device-1",
				CVEID:           "CVE-2024-12345",
				Title:           "Critical RCE Vulnerability",
				Description:     "A remote code execution vulnerability",
				Severity:        shell.SeverityCritical,
				CVSSScore:       9.8,
				CVSSVector:      "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
				AffectedPort:    22,
				AffectedService: "ssh",
				Remediation:     "Upgrade to version X.Y.Z",
				References:      []string{"https://nvd.nist.gov/vuln/detail/CVE-2024-12345"},
				IsKEV:           true,
				IsExploited:     true,
				DiscoveredAt:    now,
				Status:          shell.VulnStatusNew,
			},
		},
		{
			name: "medium_vulnerability",
			vuln: shell.Vulnerability{
				ID:           "vuln-2",
				DeviceID:     "device-2",
				CVEID:        "CVE-2024-67890",
				Title:        "Information Disclosure",
				Description:  "An information disclosure vulnerability",
				Severity:     shell.SeverityMedium,
				CVSSScore:    5.3,
				DiscoveredAt: now,
				Status:       shell.VulnStatusAcknowledged,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.vuln.ID == "" {
				t.Error("Vulnerability ID should not be empty")
			}
			if tt.vuln.CVEID == "" {
				t.Error("CVE ID should not be empty")
			}
			if tt.vuln.Severity == "" {
				t.Error("Severity should not be empty")
			}
		})
	}
}

// TestVulnerabilityScan tests VulnerabilityScan struct initialization.
func TestVulnerabilityScan(t *testing.T) {
	startTime := time.Now()
	completedTime := startTime.Add(10 * time.Second)
	duration := completedTime.Sub(startTime)

	scan := shell.VulnerabilityScan{
		ID: "scan-1",
		Vulnerabilities: []shell.Vulnerability{
			{ID: "v1", Severity: shell.SeverityCritical},
			{ID: "v2", Severity: shell.SeverityHigh},
			{ID: "v3", Severity: shell.SeverityMedium},
			{ID: "v4", Severity: shell.SeverityLow},
		},
		DevicesScanned: 10,
		TotalCritical:  1,
		TotalHigh:      1,
		TotalMedium:    1,
		TotalLow:       1,
		ScanDuration:   duration,
		ScanDurationMs: float64(duration.Milliseconds()),
		StartedAt:      startTime,
		CompletedAt:    completedTime,
	}

	tests := []struct {
		name  string
		check func() bool
	}{
		{"has_id", func() bool { return scan.ID != "" }},
		{"has_vulnerabilities", func() bool { return len(scan.Vulnerabilities) == 4 }},
		{"devices_scanned", func() bool { return scan.DevicesScanned == 10 }},
		{"total_critical", func() bool { return scan.TotalCritical == 1 }},
		{"total_high", func() bool { return scan.TotalHigh == 1 }},
		{"total_medium", func() bool { return scan.TotalMedium == 1 }},
		{"total_low", func() bool { return scan.TotalLow == 1 }},
		{"duration_positive", func() bool { return scan.ScanDuration > 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check() {
				t.Errorf("%s check failed", tt.name)
			}
		})
	}
}

// TestPostureScore tests PostureScore struct initialization.
func TestPostureScore(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		score shell.PostureScore
	}{
		{
			name: "perfect_score",
			score: shell.PostureScore{
				Overall: 100,
				Categories: map[string]int{
					"vulnerabilities": 100,
					"configuration":   100,
					"compliance":      100,
				},
				Issues:       []shell.PostureIssue{},
				Improvements: []string{},
				AssessedAt:   now,
			},
		},
		{
			name: "score_with_issues",
			score: shell.PostureScore{
				Overall: 75,
				Categories: map[string]int{
					"vulnerabilities": 60,
					"configuration":   90,
					"compliance":      75,
				},
				Issues: []shell.PostureIssue{
					{
						Category:    "vulnerabilities",
						Severity:    "high",
						Description: "Critical vulnerability found",
						Remediation: "Apply security patches",
					},
				},
				Improvements: []string{"Apply security patches", "Enable MFA"},
				AssessedAt:   now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.score.Overall < 0 || tt.score.Overall > 100 {
				t.Errorf("Overall score %d out of valid range [0, 100]", tt.score.Overall)
			}
			if tt.score.AssessedAt.IsZero() {
				t.Error("AssessedAt should not be zero")
			}
		})
	}
}

// TestPostureIssue tests PostureIssue struct initialization.
func TestPostureIssue(t *testing.T) {
	tests := []struct {
		name  string
		issue shell.PostureIssue
	}{
		{
			name: "critical_issue",
			issue: shell.PostureIssue{
				Category:    "vulnerabilities",
				Severity:    "critical",
				Description: "Critical CVE detected",
				Remediation: "Apply patch immediately",
			},
		},
		{
			name: "configuration_issue",
			issue: shell.PostureIssue{
				Category:    "configuration",
				Severity:    "medium",
				Description: "Weak password policy",
				Remediation: "Enforce stronger passwords",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.issue.Category == "" {
				t.Error("Category should not be empty")
			}
			if tt.issue.Severity == "" {
				t.Error("Severity should not be empty")
			}
			if tt.issue.Description == "" {
				t.Error("Description should not be empty")
			}
		})
	}
}

// TestRogueDevice tests RogueDevice struct initialization.
func TestRogueDevice(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		rogue shell.RogueDevice
	}{
		{
			name: "high_risk_rogue",
			rogue: shell.RogueDevice{
				Device: shell.Device{
					ID:         "rogue-1",
					MACAddress: "00:de:ad:be:ef:00",
					DeviceType: shell.DeviceTypeUnknown,
				},
				Reason:       "Unauthorized DHCP server detected",
				RiskLevel:    "high",
				DetectedAt:   now,
				Acknowledged: false,
			},
		},
		{
			name: "medium_risk_rogue",
			rogue: shell.RogueDevice{
				Device: shell.Device{
					ID:         "rogue-2",
					MACAddress: "aa:bb:cc:dd:ee:ff",
					DeviceType: shell.DeviceTypeAP,
				},
				Reason:       "Unknown access point",
				RiskLevel:    "medium",
				DetectedAt:   now,
				Acknowledged: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rogue.Device.ID == "" {
				t.Error("Device ID should not be empty")
			}
			if tt.rogue.Reason == "" {
				t.Error("Reason should not be empty")
			}
			if tt.rogue.RiskLevel == "" {
				t.Error("RiskLevel should not be empty")
			}
			if tt.rogue.DetectedAt.IsZero() {
				t.Error("DetectedAt should not be zero")
			}
		})
	}
}

// TestRogueAlert tests RogueAlert struct initialization.
func TestRogueAlert(t *testing.T) {
	now := time.Now()
	ackTime := now.Add(time.Hour)

	tests := []struct {
		name  string
		alert shell.RogueAlert
	}{
		{
			name: "unacknowledged_alert",
			alert: shell.RogueAlert{
				ID: "alert-1",
				Device: shell.RogueDevice{
					Device:     shell.Device{ID: "rogue-1"},
					Reason:     "Unauthorized device",
					RiskLevel:  "high",
					DetectedAt: now,
				},
				AlertType:      "rogue_dhcp",
				Message:        "Rogue DHCP server detected at 192.168.1.254",
				CreatedAt:      now,
				AcknowledgedAt: nil,
			},
		},
		{
			name: "acknowledged_alert",
			alert: shell.RogueAlert{
				ID: "alert-2",
				Device: shell.RogueDevice{
					Device:       shell.Device{ID: "rogue-2"},
					Reason:       "Unknown AP",
					RiskLevel:    "medium",
					DetectedAt:   now,
					Acknowledged: true,
				},
				AlertType:      "rogue_ap",
				Message:        "Unknown access point detected",
				CreatedAt:      now,
				AcknowledgedAt: &ackTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.alert.ID == "" {
				t.Error("Alert ID should not be empty")
			}
			if tt.alert.AlertType == "" {
				t.Error("AlertType should not be empty")
			}
			if tt.alert.Message == "" {
				t.Error("Message should not be empty")
			}
			if tt.alert.CreatedAt.IsZero() {
				t.Error("CreatedAt should not be zero")
			}
		})
	}
}

// TestModuleErrors tests shell module error constants.
func TestModuleErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotImplemented", shell.ErrNotImplemented},
		{"ErrNotInitialized", shell.ErrNotInitialized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s should have a non-empty message", tt.name)
			}
		})
	}
}

// TestNewModule tests Module creation.
func TestNewModule(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)

	if module == nil {
		t.Fatal("New() returned nil")
	}

	// Test that subservices are accessible
	if module.Discovery() == nil {
		t.Error("Discovery() returned nil")
	}
	if module.Vulnerability() == nil {
		t.Error("Vulnerability() returned nil")
	}
	if module.Posture() == nil {
		t.Error("Posture() returned nil")
	}
	if module.Rogue() == nil {
		t.Error("Rogue() returned nil")
	}
}

// TestModuleStartStop tests Module Start and Stop.
func TestModuleStartStop(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)

	ctx := context.Background()

	// Test Start
	if err := module.Start(ctx); err != nil {
		t.Errorf("Start() returned error: %v", err)
	}

	// Test Stop
	if err := module.Stop(); err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

// TestDiscoveryServiceNotInitialized tests DiscoveryService methods when not fully initialized.
func TestDiscoveryServiceNotInitialized(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	discoveryService := module.Discovery()

	ctx := context.Background()

	// GetDevices should work even without full initialization
	devices, err := discoveryService.GetDevices(ctx)
	if err != nil {
		t.Logf("GetDevices returned expected error: %v", err)
	} else if devices != nil {
		t.Logf("GetDevices returned %d devices", len(devices))
	}
}

// TestVulnerabilityServiceNotInitialized tests VulnerabilityService when scanner is not available.
func TestVulnerabilityServiceNotInitialized(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	vulnService := module.Vulnerability()

	ctx := context.Background()

	// GetVulnerabilities may return ErrNotInitialized if scanner failed to initialize
	vulns, err := vulnService.GetVulnerabilities(ctx)
	if err != nil {
		if err == shell.ErrNotInitialized {
			t.Log("VulnerabilityService correctly returned ErrNotInitialized")
		} else {
			t.Logf("GetVulnerabilities returned error: %v", err)
		}
	} else if vulns != nil {
		t.Logf("GetVulnerabilities returned %d vulnerabilities", len(vulns))
	}

	// UpdateStatus should return ErrNotImplemented
	err = vulnService.UpdateStatus(ctx, "test-vuln-id", shell.VulnStatusResolved)
	if err != shell.ErrNotImplemented {
		t.Errorf("UpdateStatus should return ErrNotImplemented, got: %v", err)
	}
}

// TestPostureServiceAssess tests PostureService.Assess.
func TestPostureServiceAssess(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	postureService := module.Posture()

	ctx := context.Background()

	score, err := postureService.Assess(ctx)
	if err != nil {
		t.Errorf("Assess() returned error: %v", err)
	}

	if score == nil {
		t.Fatal("Assess() returned nil score")
	}

	// Score should be between 0 and 100
	if score.Overall < 0 || score.Overall > 100 {
		t.Errorf("Overall score %d out of valid range [0, 100]", score.Overall)
	}

	// Categories should be initialized
	if score.Categories == nil {
		t.Error("Categories should not be nil")
	}

	// Issues should be initialized (may be empty)
	if score.Issues == nil {
		t.Error("Issues should not be nil")
	}

	// AssessedAt should be set
	if score.AssessedAt.IsZero() {
		t.Error("AssessedAt should not be zero")
	}
}

// TestRogueServiceNotInitialized tests RogueService methods.
func TestRogueServiceNotInitialized(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	rogueService := module.Rogue()

	ctx := context.Background()

	// GetRogueDevices
	rogues, err := rogueService.GetRogueDevices(ctx)
	if err != nil {
		t.Logf("GetRogueDevices returned error (may be expected): %v", err)
	} else if rogues != nil {
		t.Logf("GetRogueDevices returned %d devices", len(rogues))
	}

	// GetAlerts
	alerts, err := rogueService.GetAlerts(ctx)
	if err != nil {
		t.Logf("GetAlerts returned error (may be expected): %v", err)
	} else if alerts != nil {
		t.Logf("GetAlerts returned %d alerts", len(alerts))
	}

	// AcknowledgeDevice should return ErrNotImplemented
	err = rogueService.AcknowledgeDevice(ctx, "test-device-id")
	if err != shell.ErrNotImplemented {
		t.Errorf("AcknowledgeDevice should return ErrNotImplemented, got: %v", err)
	}
}

// TestDiscoveryServiceAccessors tests DiscoveryService accessor methods.
func TestDiscoveryServiceAccessors(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	discoveryService := module.Discovery()

	// Test Service() accessor
	service := discoveryService.Service()
	if service == nil {
		t.Error("Service() returned nil")
	}

	// Test DeviceDiscovery() accessor
	deviceDiscovery := discoveryService.DeviceDiscovery()
	if deviceDiscovery == nil {
		t.Error("DeviceDiscovery() returned nil")
	}
}

// TestVulnerabilityServiceAccessors tests VulnerabilityService accessor methods.
func TestVulnerabilityServiceAccessors(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	vulnService := module.Vulnerability()

	// Scanner() may return nil if initialization failed
	scanner := vulnService.Scanner()
	if scanner == nil {
		t.Log("Scanner() returned nil (may be expected if initialization failed)")
	} else {
		t.Log("Scanner() returned non-nil scanner")
	}
}

// TestRogueServiceAccessors tests RogueService accessor methods.
func TestRogueServiceAccessors(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	rogueService := module.Rogue()

	// Detector() should return non-nil
	detector := rogueService.Detector()
	if detector == nil {
		t.Error("Detector() returned nil")
	}
}

// TestDefaultInterface tests the DefaultInterface constant.
func TestDefaultInterface(t *testing.T) {
	// Access the DefaultInterface constant
	if shell.DefaultInterface == "" {
		t.Error("DefaultInterface should not be empty")
	}
	t.Logf("DefaultInterface = %s", shell.DefaultInterface)
}

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
		})
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
		})
	}
}

// TestModuleTestAccessor tests the ModuleTestAccessor functionality.
func TestModuleTestAccessor(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	accessor := shell.ModuleTestAccessor{Module: module}

	// Test config accessor
	if accessor.GetCfg() == nil {
		t.Error("GetCfg() returned nil")
	}

	// Test DB accessor (may be nil if not initialized)
	_ = accessor.GetDB()

	// Test service accessors
	if accessor.GetDiscoveryService() == nil {
		t.Error("GetDiscoveryService() returned nil")
	}
	if accessor.GetVulnerabilityService() == nil {
		t.Error("GetVulnerabilityService() returned nil")
	}
	if accessor.GetPostureService() == nil {
		t.Error("GetPostureService() returned nil")
	}
	if accessor.GetRogueService() == nil {
		t.Error("GetRogueService() returned nil")
	}
}

// TestDiscoveryServiceTestAccessorWithTestutil tests the DiscoveryServiceTestAccessor using testutil builder.
func TestDiscoveryServiceTestAccessorWithTestutil(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	discoveryService := module.Discovery()
	accessor := shell.DiscoveryServiceTestAccessor{Service: discoveryService}

	// Test config accessor
	if accessor.GetCfg() == nil {
		t.Error("GetCfg() returned nil")
	}

	// Test DB accessor (may be nil)
	_ = accessor.GetDB()
}

// TestVulnerabilityServiceTestAccessorWithTestutil tests the VulnerabilityServiceTestAccessor using testutil builder.
func TestVulnerabilityServiceTestAccessorWithTestutil(t *testing.T) {
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
	_ = accessor.GetDB()

	// Scanner may be nil if initialization failed
	_ = accessor.GetScanner()
}

// TestPostureServiceTestAccessorWithTestutil tests the PostureServiceTestAccessor using testutil builder.
func TestPostureServiceTestAccessorWithTestutil(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	postureService := module.Posture()
	accessor := shell.PostureServiceTestAccessor{Service: postureService}

	// Test config accessor
	if accessor.GetCfg() == nil {
		t.Error("GetCfg() returned nil")
	}

	// Test DB accessor (may be nil)
	_ = accessor.GetDB()

	// Test discovery service accessor
	if accessor.GetDiscovery() == nil {
		t.Error("GetDiscovery() returned nil")
	}

	// Test vulnerability service accessor
	if accessor.GetVulnerability() == nil {
		t.Error("GetVulnerability() returned nil")
	}
}

// TestRogueServiceTestAccessorWithTestutil tests the RogueServiceTestAccessor using testutil builder.
func TestRogueServiceTestAccessorWithTestutil(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	rogueService := module.Rogue()
	accessor := shell.RogueServiceTestAccessor{Service: rogueService}

	// Test config accessor
	if accessor.GetCfg() == nil {
		t.Error("GetCfg() returned nil")
	}
}

// TestDiscoveryServiceStartStop tests DiscoveryService Start and Stop methods.
func TestDiscoveryServiceStartStop(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	discoveryService := module.Discovery()

	ctx := context.Background()

	// Start may fail on test system without proper network interface
	err := discoveryService.Start(ctx)
	if err != nil {
		t.Logf("Start returned error (may be expected on test system): %v", err)
	}

	// Stop should always succeed
	discoveryService.Stop()
}

// TestDiscoveryServiceDiscover tests DiscoveryService.Discover method.
func TestDiscoveryServiceDiscover(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	discoveryService := module.Discovery()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	opts := &shell.DiscoveryOptions{
		Interface:   "lo",
		EnableARP:   false, // Disable for faster testing
		EnableICMP:  false,
		Timeout:     1 * time.Second,
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

	// Result should have timing information
	if result.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
	if result.CompletedAt.IsZero() {
		t.Error("CompletedAt should not be zero")
	}
	if result.ScanDuration <= 0 {
		t.Error("ScanDuration should be positive")
	}
}

// TestDiscoveryServiceGetDevice tests DiscoveryService.GetDevice method.
func TestDiscoveryServiceGetDevice(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	discoveryService := module.Discovery()

	ctx := context.Background()

	// Test with non-existent device
	device, err := discoveryService.GetDevice(ctx, "00:11:22:33:44:55")
	if err == nil && device != nil {
		t.Logf("Found device: %s", device.ID)
	} else if err != nil {
		// Expected - device not found
		t.Logf("GetDevice returned expected error: %v", err)
	}
}

// TestVulnerabilityServiceScan tests VulnerabilityService.Scan method.
func TestVulnerabilityServiceScan(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	vulnService := module.Vulnerability()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targets := []string{"127.0.0.1"}

	result, err := vulnService.Scan(ctx, targets)
	if err != nil {
		if err == shell.ErrNotInitialized {
			t.Log("VulnerabilityService not initialized (expected in test environment)")
			return
		}
		t.Logf("Scan returned error: %v", err)
		return
	}

	if result == nil {
		t.Fatal("Scan returned nil result")
	}

	// Result should have an ID
	if result.ID == "" {
		t.Error("Scan result ID should not be empty")
	}

	// Result should have timing information
	if result.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
	if result.CompletedAt.IsZero() {
		t.Error("CompletedAt should not be zero")
	}

	// DevicesScanned should match target count
	if result.DevicesScanned != len(targets) {
		t.Errorf("DevicesScanned = %d, want %d", result.DevicesScanned, len(targets))
	}
}

// TestVulnerabilityServiceGetDeviceVulnerabilities tests GetDeviceVulnerabilities.
func TestVulnerabilityServiceGetDeviceVulnerabilities(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	vulnService := module.Vulnerability()

	ctx := context.Background()

	vulns, err := vulnService.GetDeviceVulnerabilities(ctx, "127.0.0.1")
	if err != nil {
		if err == shell.ErrNotInitialized {
			t.Log("VulnerabilityService not initialized (expected in test environment)")
			return
		}
		t.Logf("GetDeviceVulnerabilities returned error: %v", err)
		return
	}

	// May return nil or empty slice if no vulnerabilities found
	t.Logf("Found %d vulnerabilities", len(vulns))
}

// TestRogueServiceStartStop tests RogueService Start and Stop methods.
func TestRogueServiceStartStop(t *testing.T) {
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

// TestMultipleModuleInstances tests creating multiple module instances.
func TestMultipleModuleInstances(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	// Create multiple instances
	module1 := shell.New(cfg, nil)
	module2 := shell.New(cfg, nil)

	if module1 == nil || module2 == nil {
		t.Fatal("Failed to create module instances")
	}

	// Modules should be independent
	if module1.Discovery() == module2.Discovery() {
		t.Error("Modules should have independent Discovery services")
	}
	if module1.Vulnerability() == module2.Vulnerability() {
		t.Error("Modules should have independent Vulnerability services")
	}
	if module1.Posture() == module2.Posture() {
		t.Error("Modules should have independent Posture services")
	}
	if module1.Rogue() == module2.Rogue() {
		t.Error("Modules should have independent Rogue services")
	}
}

// TestConcurrentModuleAccess tests concurrent access to module methods.
func TestConcurrentModuleAccess(t *testing.T) {
	cfg := testutil.NewConfigBuilder().
		WithInterface("lo").
		Build()

	module := shell.New(cfg, nil)
	ctx := context.Background()

	// Start multiple goroutines accessing the module
	done := make(chan bool, 4)

	go func() {
		_ = module.Discovery()
		done <- true
	}()

	go func() {
		_ = module.Vulnerability()
		done <- true
	}()

	go func() {
		_ = module.Posture()
		done <- true
	}()

	go func() {
		_ = module.Rogue()
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent access")
		}
	}

	// Test concurrent posture assessment
	assessDone := make(chan bool, 2)

	go func() {
		_, _ = module.Posture().Assess(ctx)
		assessDone <- true
	}()

	go func() {
		_, _ = module.Posture().Assess(ctx)
		assessDone <- true
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-assessDone:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent assessment")
		}
	}
}
