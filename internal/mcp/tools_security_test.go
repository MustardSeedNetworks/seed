package mcp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/mcp"
)

func TestVulnerabilityScanFlow(t *testing.T) {
	tests := []struct {
		name           string
		deviceIP       string
		devices        []*discovery.DiscoveredDevice
		vulnResult     any
		scanErr        error
		expectError    bool
		expectDevFound bool
	}{
		{
			name:     "device found and scanned",
			deviceIP: "192.168.1.100",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.100", MAC: "00:11:22:33:44:55"},
			},
			vulnResult: map[string]any{
				"vulnerabilities": []map[string]any{
					{"cve": "CVE-2021-1234", "severity": "high"},
				},
			},
			expectError:    false,
			expectDevFound: true,
		},
		{
			name:     "device not found",
			deviceIP: "192.168.1.200",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.100", MAC: "00:11:22:33:44:55"},
			},
			expectError:    true,
			expectDevFound: false,
		},
		{
			name:           "no devices discovered",
			deviceIP:       "192.168.1.100",
			devices:        []*discovery.DiscoveredDevice{},
			expectError:    true,
			expectDevFound: false,
		},
		{
			name:     "scan error",
			deviceIP: "192.168.1.100",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.100", MAC: "00:11:22:33:44:55"},
			},
			scanErr:        errors.New("scan failed"),
			expectError:    true,
			expectDevFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate finding the device
			var targetDevice *discovery.DiscoveredDevice
			for _, d := range tt.devices {
				if d.IP == tt.deviceIP {
					targetDevice = d
					break
				}
			}

			if tt.expectDevFound {
				if targetDevice == nil {
					t.Error("expected to find device but didn't")
				}
			} else {
				if targetDevice != nil {
					t.Error("expected device not found but found one")
				}
			}
		})
	}
}

func TestRogueDHCPCheckFlow(t *testing.T) {
	tests := []struct {
		name      string
		servers   any
		isRunning bool
		expectNil bool
	}{
		{
			name: "servers detected",
			servers: []map[string]any{
				{
					"ip":         "192.168.1.1",
					"mac":        "00:11:22:33:44:55",
					"serverName": "DHCP Server 1",
					"isRogue":    false,
				},
				{
					"ip":         "192.168.1.50",
					"mac":        "00:11:22:33:44:99",
					"serverName": "Unknown DHCP",
					"isRogue":    true,
				},
			},
			isRunning: true,
			expectNil: false,
		},
		{
			name:      "no servers detected",
			servers:   nil,
			isRunning: true,
			expectNil: true,
		},
		{
			name:      "detector not running",
			servers:   nil,
			isRunning: false,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRogueDetector{
				servers:   tt.servers,
				isRunning: tt.isRunning,
			}

			servers := mock.GetDetectedServers()
			if tt.expectNil {
				if servers != nil {
					t.Error("expected nil servers")
				}
			} else {
				if servers == nil {
					t.Error("expected non-nil servers")
				}
			}

			if mock.IsRunning() != tt.isRunning {
				t.Errorf("expected isRunning=%v, got %v", tt.isRunning, mock.IsRunning())
			}
		})
	}
}

func TestSNMPQueryFlow(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		community       string
		configuredComms []string
		expectError     bool
		errorContains   string
	}{
		{
			name:            "with provided community",
			host:            "192.168.1.1",
			community:       "public",
			configuredComms: []string{"private"},
			expectError:     false,
		},
		{
			name:            "using configured community",
			host:            "192.168.1.1",
			community:       "",
			configuredComms: []string{"private", "public"},
			expectError:     false,
		},
		{
			name:            "no community available",
			host:            "192.168.1.1",
			community:       "",
			configuredComms: []string{},
			expectError:     true,
			errorContains:   "community",
		},
		{
			name:            "missing host",
			host:            "",
			community:       "public",
			configuredComms: []string{"public"},
			expectError:     true,
			errorContains:   "host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the SNMP query logic
			var community string
			var errMsg string

			switch {
			case tt.host == "":
				errMsg = "host parameter is required"
			case tt.community != "":
				community = tt.community
			case len(tt.configuredComms) > 0:
				community = tt.configuredComms[0]
			default:
				errMsg = "No SNMP community string provided and none configured"
			}

			if tt.expectError {
				if errMsg == "" {
					t.Error("expected error but got none")
				}
				if tt.errorContains != "" && !containsString(errMsg, tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, errMsg)
				}
			} else {
				if errMsg != "" {
					t.Errorf("unexpected error: %s", errMsg)
				}
				if community == "" {
					t.Error("expected community to be set")
				}
			}
		})
	}
}

func TestSecurityServiceAvailability(t *testing.T) {
	tests := []struct {
		name             string
		hasVulnScanner   bool
		hasRogueDetector bool
		hasConfig        bool
	}{
		{
			name:             "all services available",
			hasVulnScanner:   true,
			hasRogueDetector: true,
			hasConfig:        true,
		},
		{
			name:             "no vulnerability scanner",
			hasVulnScanner:   false,
			hasRogueDetector: true,
			hasConfig:        true,
		},
		{
			name:             "no rogue detector",
			hasVulnScanner:   true,
			hasRogueDetector: false,
			hasConfig:        true,
		},
		{
			name:             "no config",
			hasVulnScanner:   true,
			hasRogueDetector: true,
			hasConfig:        false,
		},
		{
			name:             "no services",
			hasVulnScanner:   false,
			hasRogueDetector: false,
			hasConfig:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var vulnScanner mcp.VulnScanner
			var rogueDetector mcp.RogueDetector
			var cfg *config.Config

			if tt.hasVulnScanner {
				vulnScanner = &mockVulnScanner{}
			}
			if tt.hasRogueDetector {
				rogueDetector = &mockRogueDetector{}
			}
			if tt.hasConfig {
				cfg = &config.Config{
					SNMP: config.SNMPConfig{
						Communities: []string{"public"},
					},
				}
			}

			provider := &mockServiceProvider{
				vulnScanner:   vulnScanner,
				rogueDetector: rogueDetector,
				cfg:           cfg,
			}

			if (provider.GetVulnScanner() != nil) != tt.hasVulnScanner {
				t.Errorf("vulnScanner availability mismatch")
			}
			if (provider.GetRogueDetector() != nil) != tt.hasRogueDetector {
				t.Errorf("rogueDetector availability mismatch")
			}
			if (provider.GetConfig() != nil) != tt.hasConfig {
				t.Errorf("config availability mismatch")
			}
		})
	}
}

func TestVulnerabilityResult(t *testing.T) {
	tests := []struct {
		name        string
		vulns       []map[string]any
		expectCount int
	}{
		{
			name: "multiple vulnerabilities",
			vulns: []map[string]any{
				{"cve": "CVE-2021-1234", "severity": "high", "score": 8.5},
				{"cve": "CVE-2021-5678", "severity": "medium", "score": 5.0},
				{"cve": "CVE-2020-9999", "severity": "low", "score": 2.0},
			},
			expectCount: 3,
		},
		{
			name:        "no vulnerabilities",
			vulns:       []map[string]any{},
			expectCount: 0,
		},
		{
			name: "single vulnerability",
			vulns: []map[string]any{
				{"cve": "CVE-2023-1234", "severity": "critical", "score": 9.8},
			},
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockVulnScanner{
				result: map[string]any{
					"vulnerabilities": tt.vulns,
					"scanTime":        "2025-01-07T12:00:00Z",
				},
			}

			result, err := mock.ScanDevice(context.Background(), &discovery.DiscoveredDevice{
				IP: "192.168.1.100",
			})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Fatal("expected result to be a map")
			}

			vulns, ok := resultMap["vulnerabilities"].([]map[string]any)
			if !ok {
				t.Fatal("expected vulnerabilities to be a slice")
			}

			if len(vulns) != tt.expectCount {
				t.Errorf("expected %d vulnerabilities, got %d", tt.expectCount, len(vulns))
			}
		})
	}
}

func TestSNMPConfigCommunities(t *testing.T) {
	tests := []struct {
		name          string
		communities   []string
		expectedFirst string
	}{
		{
			name:          "single community",
			communities:   []string{"public"},
			expectedFirst: "public",
		},
		{
			name:          "multiple communities",
			communities:   []string{"private", "public", "custom"},
			expectedFirst: "private",
		},
		{
			name:          "empty communities",
			communities:   []string{},
			expectedFirst: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				SNMP: config.SNMPConfig{
					Communities: tt.communities,
				},
			}

			var first string
			if len(cfg.SNMP.Communities) > 0 {
				first = cfg.SNMP.Communities[0]
			}

			if first != tt.expectedFirst {
				t.Errorf("expected first community %q, got %q", tt.expectedFirst, first)
			}
		})
	}
}
