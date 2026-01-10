package shell_test

import (
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/shell"
)

// ========== DeviceType Constants Tests ==========

func TestDeviceTypeConstantsTypesFile(t *testing.T) {
	tests := []struct {
		name     string
		constant shell.DeviceType
		expected string
	}{
		{"DeviceTypeRouter", shell.DeviceTypeRouter, "router"},
		{"DeviceTypeSwitch", shell.DeviceTypeSwitch, "switch"},
		{"DeviceTypeAP", shell.DeviceTypeAP, "access_point"},
		{"DeviceTypeServer", shell.DeviceTypeServer, "server"},
		{"DeviceTypeWorkstation", shell.DeviceTypeWorkstation, "workstation"},
		{"DeviceTypeMobile", shell.DeviceTypeMobile, "mobile"},
		{"DeviceTypeIoT", shell.DeviceTypeIoT, "iot"},
		{"DeviceTypePrinter", shell.DeviceTypePrinter, "printer"},
		{"DeviceTypeCamera", shell.DeviceTypeCamera, "camera"},
		{"DeviceTypeUnknown", shell.DeviceTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.constant))
			}
		})
	}
}

// ========== VulnSeverity Constants Tests ==========

func TestVulnSeverityConstantsTypesFile(t *testing.T) {
	tests := []struct {
		name     string
		constant shell.VulnSeverity
		expected string
	}{
		{"SeverityCritical", shell.SeverityCritical, "critical"},
		{"SeverityHigh", shell.SeverityHigh, "high"},
		{"SeverityMedium", shell.SeverityMedium, "medium"},
		{"SeverityLow", shell.SeverityLow, "low"},
		{"SeverityInfo", shell.SeverityInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.constant))
			}
		})
	}
}

// ========== VulnStatus Constants Tests ==========

func TestVulnStatusConstantsTypesFile(t *testing.T) {
	tests := []struct {
		name     string
		constant shell.VulnStatus
		expected string
	}{
		{"VulnStatusNew", shell.VulnStatusNew, "new"},
		{"VulnStatusAcknowledged", shell.VulnStatusAcknowledged, "acknowledged"},
		{"VulnStatusInProgress", shell.VulnStatusInProgress, "in_progress"},
		{"VulnStatusResolved", shell.VulnStatusResolved, "resolved"},
		{"VulnStatusFalsePositive", shell.VulnStatusFalsePositive, "false_positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.constant))
			}
		})
	}
}

// ========== Device Structure Tests ==========

func TestDeviceStructure(t *testing.T) {
	now := time.Now()
	ip := net.ParseIP("192.168.1.100")

	device := shell.Device{
		ID:         "test-device-001",
		IPAddress:  ip,
		MACAddress: "00:11:22:33:44:55",
		Hostname:   "test-host",
		Vendor:     "Test Vendor",
		DeviceType: shell.DeviceTypeServer,
		OS:         "Linux",
		Services: []shell.Service{
			{Port: 22, Protocol: "tcp", Name: "ssh", State: "open"},
			{Port: 80, Protocol: "tcp", Name: "http", State: "open"},
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
		FirstSeen: now.Add(-time.Hour),
		LastSeen:  now,
		IsOnline:  true,
		IsGateway: false,
		Metadata: map[string]string{
			"location": "datacenter",
		},
	}

	// Verify ID
	if device.ID != "test-device-001" {
		t.Errorf("expected ID 'test-device-001', got %q", device.ID)
	}

	// Verify IP
	if !device.IPAddress.Equal(ip) {
		t.Errorf("expected IP %v, got %v", ip, device.IPAddress)
	}

	// Verify MAC
	if device.MACAddress != "00:11:22:33:44:55" {
		t.Errorf("expected MAC '00:11:22:33:44:55', got %q", device.MACAddress)
	}

	// Verify Hostname
	if device.Hostname != "test-host" {
		t.Errorf("expected hostname 'test-host', got %q", device.Hostname)
	}
	if device.Vendor != "Test Vendor" {
		t.Errorf("expected vendor 'Test Vendor', got %q", device.Vendor)
	}
	if device.OS != "Linux" {
		t.Errorf("expected OS 'Linux', got %q", device.OS)
	}

	// Verify DeviceType
	if device.DeviceType != shell.DeviceTypeServer {
		t.Errorf("expected DeviceType server, got %v", device.DeviceType)
	}

	// Verify Services count
	if len(device.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(device.Services))
	}

	// Verify Interfaces count
	if len(device.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(device.Interfaces))
	}

	// Verify IsOnline
	if !device.IsOnline {
		t.Error("expected IsOnline to be true")
	}
	if device.IsGateway {
		t.Error("expected IsGateway to be false")
	}
	if !device.FirstSeen.Equal(now.Add(-time.Hour)) {
		t.Errorf("expected FirstSeen %v, got %v", now.Add(-time.Hour), device.FirstSeen)
	}
	if !device.LastSeen.Equal(now) {
		t.Errorf("expected LastSeen %v, got %v", now, device.LastSeen)
	}

	// Verify Metadata
	if device.Metadata["location"] != "datacenter" {
		t.Errorf("expected metadata location 'datacenter', got %q", device.Metadata["location"])
	}
}

// ========== Service Structure Tests ==========

func TestServiceStructure(t *testing.T) {
	tests := []struct {
		name      string
		service   shell.Service
		wantPort  int
		wantProto string
		wantName  string
		wantState string
	}{
		{
			name: "SSH service",
			service: shell.Service{
				Port:     22,
				Protocol: "tcp",
				Name:     "ssh",
				Version:  "OpenSSH 8.9",
				Banner:   "SSH-2.0-OpenSSH_8.9",
				State:    "open",
			},
			wantPort:  22,
			wantProto: "tcp",
			wantName:  "ssh",
			wantState: "open",
		},
		{
			name: "HTTP service",
			service: shell.Service{
				Port:     80,
				Protocol: "tcp",
				Name:     "http",
				State:    "open",
			},
			wantPort:  80,
			wantProto: "tcp",
			wantName:  "http",
			wantState: "open",
		},
		{
			name: "DNS service",
			service: shell.Service{
				Port:     53,
				Protocol: "udp",
				Name:     "dns",
				State:    "open",
			},
			wantPort:  53,
			wantProto: "udp",
			wantName:  "dns",
			wantState: "open",
		},
		{
			name: "filtered port",
			service: shell.Service{
				Port:     443,
				Protocol: "tcp",
				Name:     "https",
				State:    "filtered",
			},
			wantPort:  443,
			wantProto: "tcp",
			wantName:  "https",
			wantState: "filtered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.service.Port != tt.wantPort {
				t.Errorf("expected port %d, got %d", tt.wantPort, tt.service.Port)
			}
			if tt.service.Protocol != tt.wantProto {
				t.Errorf("expected protocol %q, got %q", tt.wantProto, tt.service.Protocol)
			}
			if tt.service.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, tt.service.Name)
			}
			if tt.service.State != tt.wantState {
				t.Errorf("expected state %q, got %q", tt.wantState, tt.service.State)
			}
		})
	}
}

// ========== DeviceInterface Structure Tests ==========

func TestDeviceInterfaceStructure(t *testing.T) {
	tests := []struct {
		name     string
		iface    shell.DeviceInterface
		wantType string
	}{
		{
			name: "ethernet interface",
			iface: shell.DeviceInterface{
				Name:        "eth0",
				MACAddress:  "00:11:22:33:44:55",
				IPAddresses: []string{"192.168.1.100", "192.168.1.101"},
				Type:        "ethernet",
				Speed:       "1000Mbps",
				Status:      "up",
			},
			wantType: "ethernet",
		},
		{
			name: "wifi interface",
			iface: shell.DeviceInterface{
				Name:        "wlan0",
				MACAddress:  "AA:BB:CC:DD:EE:FF",
				IPAddresses: []string{"192.168.1.50"},
				Type:        "wifi",
				Speed:       "866Mbps",
				Status:      "up",
			},
			wantType: "wifi",
		},
		{
			name: "loopback interface",
			iface: shell.DeviceInterface{
				Name:        "lo0",
				MACAddress:  "",
				IPAddresses: []string{"127.0.0.1"},
				Type:        "loopback",
				Status:      "up",
			},
			wantType: "loopback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.iface.Type != tt.wantType {
				t.Errorf("expected type %q, got %q", tt.wantType, tt.iface.Type)
			}
			if len(tt.iface.IPAddresses) == 0 {
				t.Error("expected at least one IP address")
			}
		})
	}
}

// ========== DiscoveryResult Structure Tests ==========

func TestDiscoveryResultStructure(t *testing.T) {
	now := time.Now()
	duration := 5 * time.Second

	result := shell.DiscoveryResult{
		Devices: []shell.Device{
			{ID: "device-1", DeviceType: shell.DeviceTypeServer},
			{ID: "device-2", DeviceType: shell.DeviceTypeRouter},
		},
		NewDevices:     2,
		UpdatedDevices: 5,
		OfflineDevices: 1,
		ScanDuration:   duration,
		ScanDurationMs: float64(duration.Milliseconds()),
		StartedAt:      now.Add(-duration),
		CompletedAt:    now,
	}

	if len(result.Devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(result.Devices))
	}
	if result.NewDevices != 2 {
		t.Errorf("expected 2 new devices, got %d", result.NewDevices)
	}
	if result.UpdatedDevices != 5 {
		t.Errorf("expected 5 updated devices, got %d", result.UpdatedDevices)
	}
	if result.OfflineDevices != 1 {
		t.Errorf("expected 1 offline device, got %d", result.OfflineDevices)
	}
	if result.ScanDuration != duration {
		t.Errorf("expected scan duration %v, got %v", duration, result.ScanDuration)
	}
	if result.ScanDurationMs != float64(duration.Milliseconds()) {
		t.Errorf("expected scan duration ms %v, got %v",
			float64(duration.Milliseconds()), result.ScanDurationMs)
	}
	if !result.StartedAt.Equal(now.Add(-duration)) {
		t.Errorf("expected StartedAt %v, got %v", now.Add(-duration), result.StartedAt)
	}
	if !result.CompletedAt.Equal(now) {
		t.Errorf("expected CompletedAt %v, got %v", now, result.CompletedAt)
	}
}

// ========== DiscoveryOptions Structure Tests ==========

func TestDiscoveryOptionsStructure(t *testing.T) {
	opts := shell.DiscoveryOptions{
		Interface:     "eth0",
		Subnets:       []string{"192.168.1.0/24", "10.0.0.0/8"},
		EnableARP:     true,
		EnableICMP:    true,
		EnableNDP:     true,
		EnableLLDP:    true,
		EnableCDP:     false,
		EnableSNMP:    true,
		PortScan:      true,
		PortScanPorts: []int{22, 80, 443, 8080},
		Timeout:       30 * time.Second,
		Concurrency:   10,
	}

	if opts.Interface != "eth0" {
		t.Errorf("expected interface 'eth0', got %q", opts.Interface)
	}
	if len(opts.Subnets) != 2 {
		t.Errorf("expected 2 subnets, got %d", len(opts.Subnets))
	}
	if !opts.EnableARP {
		t.Error("expected EnableARP to be true")
	}
	if !opts.EnableICMP {
		t.Error("expected EnableICMP to be true")
	}
	if !opts.EnableNDP {
		t.Error("expected EnableNDP to be true")
	}
	if !opts.EnableLLDP {
		t.Error("expected EnableLLDP to be true")
	}
	if opts.EnableCDP {
		t.Error("expected EnableCDP to be false")
	}
	if !opts.EnableSNMP {
		t.Error("expected EnableSNMP to be true")
	}
	if len(opts.PortScanPorts) != 4 {
		t.Errorf("expected 4 port scan ports, got %d", len(opts.PortScanPorts))
	}
	if !opts.PortScan {
		t.Error("expected PortScan to be true")
	}
	if opts.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", opts.Timeout)
	}
	if opts.Concurrency != 10 {
		t.Errorf("expected Concurrency 10, got %d", opts.Concurrency)
	}
}

// ========== Vulnerability Structure Tests ==========

func TestVulnerabilityStructure(t *testing.T) {
	now := time.Now()
	wantDescription := "A critical vulnerability allowing remote code execution"
	wantCVSSVector := "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
	wantReferences := []string{"https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2024-12345"}

	vuln := shell.Vulnerability{
		ID:              "vuln-001",
		DeviceID:        "device-001",
		CVEID:           "CVE-2024-12345",
		Title:           "Critical Remote Code Execution",
		Description:     wantDescription,
		Severity:        shell.SeverityCritical,
		CVSSScore:       9.8,
		CVSSVector:      wantCVSSVector,
		AffectedPort:    443,
		AffectedService: "https",
		Remediation:     "Update to version 2.0.0 or later",
		References:      wantReferences,
		IsKEV:           true,
		IsExploited:     true,
		DiscoveredAt:    now,
		Status:          shell.VulnStatusNew,
	}

	if vuln.ID != "vuln-001" {
		t.Errorf("expected ID 'vuln-001', got %q", vuln.ID)
	}
	if vuln.DeviceID != "device-001" {
		t.Errorf("expected DeviceID 'device-001', got %q", vuln.DeviceID)
	}
	if vuln.CVEID != "CVE-2024-12345" {
		t.Errorf("expected CVEID 'CVE-2024-12345', got %q", vuln.CVEID)
	}
	if vuln.Title != "Critical Remote Code Execution" {
		t.Errorf("expected Title 'Critical Remote Code Execution', got %q", vuln.Title)
	}
	if vuln.Description != wantDescription {
		t.Errorf("expected Description %q, got %q", wantDescription, vuln.Description)
	}
	if vuln.Severity != shell.SeverityCritical {
		t.Errorf("expected severity critical, got %v", vuln.Severity)
	}
	if vuln.CVSSScore != 9.8 {
		t.Errorf("expected CVSS score 9.8, got %f", vuln.CVSSScore)
	}
	if vuln.CVSSVector != wantCVSSVector {
		t.Errorf("expected CVSSVector %q, got %q", wantCVSSVector, vuln.CVSSVector)
	}
	if vuln.AffectedPort != 443 {
		t.Errorf("expected AffectedPort 443, got %d", vuln.AffectedPort)
	}
	if vuln.AffectedService != "https" {
		t.Errorf("expected AffectedService 'https', got %q", vuln.AffectedService)
	}
	if vuln.Remediation != "Update to version 2.0.0 or later" {
		t.Errorf(
			"expected Remediation 'Update to version 2.0.0 or later', got %q",
			vuln.Remediation,
		)
	}
	if len(vuln.References) != len(wantReferences) {
		t.Fatalf("expected %d references, got %d", len(wantReferences), len(vuln.References))
	}
	if vuln.References[0] != wantReferences[0] {
		t.Errorf("expected References[0] %q, got %q", wantReferences[0], vuln.References[0])
	}
	if !vuln.DiscoveredAt.Equal(now) {
		t.Errorf("expected DiscoveredAt %v, got %v", now, vuln.DiscoveredAt)
	}
	if !vuln.IsKEV {
		t.Error("expected IsKEV to be true")
	}
	if !vuln.IsExploited {
		t.Error("expected IsExploited to be true")
	}
	if vuln.Status != shell.VulnStatusNew {
		t.Errorf("expected status new, got %v", vuln.Status)
	}
}

// ========== VulnerabilityScan Structure Tests ==========

func TestVulnerabilityScanStructure(t *testing.T) {
	now := time.Now()
	duration := 10 * time.Minute

	scan := shell.VulnerabilityScan{
		ID: "scan-001",
		Vulnerabilities: []shell.Vulnerability{
			{ID: "v1", Severity: shell.SeverityCritical},
			{ID: "v2", Severity: shell.SeverityHigh},
			{ID: "v3", Severity: shell.SeverityMedium},
		},
		DevicesScanned: 50,
		TotalCritical:  1,
		TotalHigh:      1,
		TotalMedium:    1,
		TotalLow:       0,
		ScanDuration:   duration,
		ScanDurationMs: float64(duration.Milliseconds()),
		StartedAt:      now.Add(-duration),
		CompletedAt:    now,
	}

	if scan.ID != "scan-001" {
		t.Errorf("expected ID 'scan-001', got %q", scan.ID)
	}
	if len(scan.Vulnerabilities) != 3 {
		t.Errorf("expected 3 vulnerabilities, got %d", len(scan.Vulnerabilities))
	}
	if scan.DevicesScanned != 50 {
		t.Errorf("expected 50 devices scanned, got %d", scan.DevicesScanned)
	}
	if scan.TotalCritical != 1 {
		t.Errorf("expected 1 critical, got %d", scan.TotalCritical)
	}
	if scan.TotalHigh != 1 {
		t.Errorf("expected 1 high, got %d", scan.TotalHigh)
	}
	if scan.TotalMedium != 1 {
		t.Errorf("expected 1 medium, got %d", scan.TotalMedium)
	}
	if scan.TotalLow != 0 {
		t.Errorf("expected 0 low, got %d", scan.TotalLow)
	}
	if scan.ScanDuration != duration {
		t.Errorf("expected ScanDuration %v, got %v", duration, scan.ScanDuration)
	}
	if scan.ScanDurationMs != float64(duration.Milliseconds()) {
		t.Errorf("expected ScanDurationMs %v, got %v",
			float64(duration.Milliseconds()), scan.ScanDurationMs)
	}
	if !scan.StartedAt.Equal(now.Add(-duration)) {
		t.Errorf("expected StartedAt %v, got %v", now.Add(-duration), scan.StartedAt)
	}
	if !scan.CompletedAt.Equal(now) {
		t.Errorf("expected CompletedAt %v, got %v", now, scan.CompletedAt)
	}
}

// ========== PostureScore Structure Tests ==========

func TestPostureScoreStructure(t *testing.T) {
	tests := []struct {
		name        string
		score       shell.PostureScore
		wantOverall int
		wantIssues  int
	}{
		{
			name: "perfect score",
			score: shell.PostureScore{
				Overall:    100,
				Categories: map[string]int{"vulnerabilities": 100, "configuration": 100},
				Issues:     []shell.PostureIssue{},
				AssessedAt: time.Now(),
			},
			wantOverall: 100,
			wantIssues:  0,
		},
		{
			name: "score with issues",
			score: shell.PostureScore{
				Overall:    75,
				Categories: map[string]int{"vulnerabilities": 60, "configuration": 90},
				Issues: []shell.PostureIssue{
					{
						Category:    "vulnerabilities",
						Severity:    "high",
						Description: "Critical vuln found",
					},
					{
						Category:    "configuration",
						Severity:    "medium",
						Description: "Weak password policy",
					},
				},
				Improvements: []string{"Patch critical vulnerabilities", "Enable MFA"},
				AssessedAt:   time.Now(),
			},
			wantOverall: 75,
			wantIssues:  2,
		},
		{
			name: "zero score",
			score: shell.PostureScore{
				Overall:    0,
				Categories: map[string]int{"vulnerabilities": 0, "configuration": 0},
				Issues: []shell.PostureIssue{
					{
						Category:    "critical",
						Severity:    "critical",
						Description: "Major breach detected",
					},
				},
				AssessedAt: time.Now(),
			},
			wantOverall: 0,
			wantIssues:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.score.Overall != tt.wantOverall {
				t.Errorf("expected overall %d, got %d", tt.wantOverall, tt.score.Overall)
			}
			if len(tt.score.Issues) != tt.wantIssues {
				t.Errorf("expected %d issues, got %d", tt.wantIssues, len(tt.score.Issues))
			}
		})
	}
}

// ========== PostureIssue Structure Tests ==========

func TestPostureIssueStructure(t *testing.T) {
	issue := shell.PostureIssue{
		Category:    "vulnerabilities",
		Severity:    "critical",
		Description: "Critical vulnerability CVE-2024-12345 detected",
		Remediation: "Apply security patch immediately",
	}

	if issue.Category != "vulnerabilities" {
		t.Errorf("expected category 'vulnerabilities', got %q", issue.Category)
	}
	if issue.Severity != "critical" {
		t.Errorf("expected severity 'critical', got %q", issue.Severity)
	}
	if issue.Description != "Critical vulnerability CVE-2024-12345 detected" {
		t.Errorf("expected description to match, got %q", issue.Description)
	}
	if issue.Remediation == "" {
		t.Error("expected remediation to be set")
	}
}

// ========== RogueDevice Structure Tests ==========

func TestRogueDeviceStructure(t *testing.T) {
	now := time.Now()

	rogue := shell.RogueDevice{
		Device: shell.Device{
			ID:         "rogue-001",
			MACAddress: "DE:AD:BE:EF:CA:FE",
			DeviceType: shell.DeviceTypeUnknown,
		},
		Reason:       "Unauthorized MAC address not in allowlist",
		RiskLevel:    "high",
		DetectedAt:   now,
		Acknowledged: false,
	}

	if rogue.Device.ID != "rogue-001" {
		t.Errorf("expected device ID 'rogue-001', got %q", rogue.Device.ID)
	}
	if rogue.RiskLevel != "high" {
		t.Errorf("expected risk level 'high', got %q", rogue.RiskLevel)
	}
	if rogue.Acknowledged {
		t.Error("expected Acknowledged to be false")
	}
	if rogue.Reason == "" {
		t.Error("expected reason to be set")
	}
	if !rogue.DetectedAt.Equal(now) {
		t.Errorf("expected DetectedAt %v, got %v", now, rogue.DetectedAt)
	}
}

// ========== RogueAlert Structure Tests ==========

func TestRogueAlertStructure(t *testing.T) {
	now := time.Now()
	ackTime := now.Add(5 * time.Minute)

	tests := []struct {
		name       string
		alert      shell.RogueAlert
		wantAckNil bool
	}{
		{
			name: "unacknowledged alert",
			alert: shell.RogueAlert{
				ID: "alert-001",
				Device: shell.RogueDevice{
					Device:     shell.Device{ID: "device-001"},
					RiskLevel:  "high",
					DetectedAt: now,
				},
				AlertType:      "rogue_dhcp",
				Message:        "Rogue DHCP server detected",
				CreatedAt:      now,
				AcknowledgedAt: nil,
			},
			wantAckNil: true,
		},
		{
			name: "acknowledged alert",
			alert: shell.RogueAlert{
				ID: "alert-002",
				Device: shell.RogueDevice{
					Device:       shell.Device{ID: "device-002"},
					RiskLevel:    "medium",
					DetectedAt:   now,
					Acknowledged: true,
				},
				AlertType:      "rogue_ap",
				Message:        "Rogue access point detected",
				CreatedAt:      now,
				AcknowledgedAt: &ackTime,
			},
			wantAckNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.alert.ID == "" {
				t.Error("expected alert ID to be set")
			}
			if tt.alert.AlertType == "" {
				t.Error("expected alert type to be set")
			}
			if tt.alert.Device.DetectedAt.IsZero() {
				t.Error("expected device DetectedAt to be set")
			}
			isAckNil := tt.alert.AcknowledgedAt == nil
			if isAckNil != tt.wantAckNil {
				t.Errorf(
					"expected AcknowledgedAt nil=%v, got nil=%v",
					tt.wantAckNil,
					isAckNil,
				)
			}
		})
	}
}

// ========== DefaultInterface Constant Test ==========

func TestDefaultInterfaceConstant(t *testing.T) {
	if shell.DefaultInterface != "eth0" {
		t.Errorf("expected DefaultInterface 'eth0', got %q", shell.DefaultInterface)
	}
}
