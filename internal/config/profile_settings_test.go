package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewProfileSettings(t *testing.T) {
	ps := NewProfileSettings()
	if ps == nil {
		t.Fatal("NewProfileSettings returned nil")
	}
	if ps.Version != ProfileSettingsVersion {
		t.Errorf("expected version %d, got %d", ProfileSettingsVersion, ps.Version)
	}
}

//nolint:gocyclo // Test function requires extensive validation.
func TestProfileSettingsFromConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Set some non-default values to verify extraction
	cfg.Thresholds.DNS.Warning = 50 * time.Millisecond
	cfg.Thresholds.DNS.Critical = 100 * time.Millisecond
	cfg.Thresholds.Ping.Warning = 25 * time.Millisecond
	cfg.Thresholds.Ping.Critical = 75 * time.Millisecond
	cfg.Thresholds.WiFi.Signal.Warning = -70
	cfg.Thresholds.WiFi.Signal.Critical = -80

	cfg.HealthChecks.RunPerformance = true
	cfg.HealthChecks.RunSpeedtest = false
	cfg.HealthChecks.RunIperf = true
	cfg.HealthChecks.PingTargets = []PingTarget{
		{Name: "Google", Host: "8.8.8.8", Enabled: true},
		{Name: "Cloudflare", Host: "1.1.1.1", Enabled: false},
	}
	cfg.HealthChecks.TCPPorts = []TCPPortTest{
		{Name: "HTTP", Host: "example.com", Port: 80, Enabled: true},
	}
	cfg.HealthChecks.HTTPEndpoints = []HTTPEndpoint{
		{Name: "API", URL: "https://api.example.com", ExpectedStatus: 200, Enabled: true},
	}

	cfg.DNS.TestHostname = "example.org"
	cfg.DNS.Timeout = 5 * time.Second
	cfg.DNS.Servers = []DNSServer{
		{Address: "8.8.8.8", Enabled: true},
		{Address: "1.1.1.1", Enabled: false},
	}

	cfg.SNMP.Communities = []string{"public", "private"}
	cfg.SNMP.Timeout = 10 * time.Second
	cfg.SNMP.Port = 161

	cfg.NetworkDiscovery.Enabled = true
	cfg.NetworkDiscovery.AutoScan = true
	cfg.NetworkDiscovery.AdditionalSubnets = []SubnetConfig{
		{CIDR: "10.0.0.0/24", Name: "LAN", Enabled: true},
	}

	cfg.FABOptions.RunLink = true
	cfg.FABOptions.RunSpeedtest = false
	cfg.FABOptions.AutoScanOnLink = true

	cfg.DisplayOptions.ShowPublicIP = true
	cfg.DisplayOptions.UnitSystem = "metric"

	// Extract settings from config
	ps := NewProfileSettings()
	ps.FromConfig(cfg)

	// Verify thresholds
	if ps.Thresholds.DNS.Warning != 50 {
		t.Errorf("DNS warning threshold: expected 50, got %d", ps.Thresholds.DNS.Warning)
	}
	if ps.Thresholds.DNS.Critical != 100 {
		t.Errorf("DNS critical threshold: expected 100, got %d", ps.Thresholds.DNS.Critical)
	}
	if ps.Thresholds.Gateway.Warning != 25 {
		t.Errorf("Gateway warning threshold: expected 25, got %d", ps.Thresholds.Gateway.Warning)
	}
	if ps.Thresholds.Gateway.Critical != 75 {
		t.Errorf("Gateway critical threshold: expected 75, got %d", ps.Thresholds.Gateway.Critical)
	}
	if ps.Thresholds.WiFi.Warning != -70 {
		t.Errorf("WiFi warning threshold: expected -70, got %d", ps.Thresholds.WiFi.Warning)
	}
	if ps.Thresholds.WiFi.Critical != -80 {
		t.Errorf("WiFi critical threshold: expected -80, got %d", ps.Thresholds.WiFi.Critical)
	}

	// Verify health checks
	if !ps.HealthChecks.RunPerformance {
		t.Error("expected RunPerformance to be true")
	}
	if ps.HealthChecks.RunSpeedtest {
		t.Error("expected RunSpeedtest to be false")
	}
	if !ps.HealthChecks.RunIperf {
		t.Error("expected RunIperf to be true")
	}
	if len(ps.HealthChecks.PingTargets) != 2 {
		t.Errorf("expected 2 ping targets, got %d", len(ps.HealthChecks.PingTargets))
	}
	if ps.HealthChecks.PingTargets[0].Name != "Google" {
		t.Errorf("expected first ping target name 'Google', got %s", ps.HealthChecks.PingTargets[0].Name)
	}
	if len(ps.HealthChecks.TCPPorts) != 1 {
		t.Errorf("expected 1 TCP port, got %d", len(ps.HealthChecks.TCPPorts))
	}
	if len(ps.HealthChecks.HTTPEndpoints) != 1 {
		t.Errorf("expected 1 HTTP endpoint, got %d", len(ps.HealthChecks.HTTPEndpoints))
	}

	// Verify DNS
	if ps.DNS.TestHostname != "example.org" {
		t.Errorf("expected DNS hostname 'example.org', got %s", ps.DNS.TestHostname)
	}
	if ps.DNS.Timeout != 5000 {
		t.Errorf("expected DNS timeout 5000ms, got %d", ps.DNS.Timeout)
	}
	if len(ps.DNS.Servers) != 2 {
		t.Errorf("expected 2 DNS servers, got %d", len(ps.DNS.Servers))
	}

	// Verify SNMP
	if len(ps.SNMP.Communities) != 2 {
		t.Errorf("expected 2 SNMP communities, got %d", len(ps.SNMP.Communities))
	}
	if ps.SNMP.Timeout != 10000 {
		t.Errorf("expected SNMP timeout 10000ms, got %d", ps.SNMP.Timeout)
	}
	if ps.SNMP.Port != 161 {
		t.Errorf("expected SNMP port 161, got %d", ps.SNMP.Port)
	}

	// Verify network discovery
	if !ps.NetworkDiscovery.Enabled {
		t.Error("expected NetworkDiscovery.Enabled to be true")
	}
	if !ps.NetworkDiscovery.AutoScan {
		t.Error("expected NetworkDiscovery.AutoScan to be true")
	}
	if len(ps.NetworkDiscovery.AdditionalSubnets) != 1 {
		t.Errorf("expected 1 additional subnet, got %d", len(ps.NetworkDiscovery.AdditionalSubnets))
	}

	// Verify FAB options
	if !ps.FABOptions.RunLink {
		t.Error("expected FABOptions.RunLink to be true")
	}
	if ps.FABOptions.RunSpeedtest {
		t.Error("expected FABOptions.RunSpeedtest to be false")
	}
	if !ps.FABOptions.AutoScanOnLink {
		t.Error("expected FABOptions.AutoScanOnLink to be true")
	}

	// Verify display options
	if !ps.DisplayOptions.ShowPublicIP {
		t.Error("expected DisplayOptions.ShowPublicIP to be true")
	}
	if ps.DisplayOptions.UnitSystem != "metric" {
		t.Errorf("expected unit system 'metric', got %s", ps.DisplayOptions.UnitSystem)
	}
}

//nolint:gocyclo // Test function requires extensive validation.
func TestProfileSettingsApplyTo(t *testing.T) {
	// Create profile settings with specific values
	ps := &ProfileSettings{
		Version: ProfileSettingsVersion,
		Thresholds: ProfileThresholds{
			DNS:        ThresholdPair{Warning: 100, Critical: 200},
			Gateway:    ThresholdPair{Warning: 50, Critical: 150},
			WiFi:       WiFiThresholdPair{Warning: -65, Critical: -75},
			CustomPing: ThresholdPair{Warning: 30, Critical: 60},
		},
		HealthChecks: ProfileHealthChecks{
			RunPerformance: true,
			RunSpeedtest:   true,
			RunIperf:       false,
			RunDiscovery:   true,
			PingTargets: []ProfilePingTarget{
				{Name: "Router", Host: "192.168.1.1", Enabled: true},
			},
			TCPPorts: []ProfileTCPPort{
				{Name: "SSH", Host: "server.local", Port: 22, Enabled: true},
			},
		},
		DNS: ProfileDNS{
			TestHostname: "test.example.com",
			Timeout:      3000,
			Servers: []ProfileDNSServer{
				{Address: "10.0.0.1", Enabled: true},
			},
		},
		SNMP: ProfileSNMP{
			Communities: []string{"community1"},
			Timeout:     5000,
			Port:        1161,
			Retries:     3,
		},
		NetworkDiscovery: ProfileNetworkDiscovery{
			Enabled:  true,
			AutoScan: false,
			AdditionalSubnets: []ProfileSubnet{
				{CIDR: "172.16.0.0/16", Name: "Corp", Enabled: true},
			},
		},
		FABOptions: ProfileFABOptions{
			RunLink:        false,
			RunSpeedtest:   true,
			AutoScanOnLink: false,
		},
		DisplayOptions: ProfileDisplayOptions{
			ShowPublicIP: false,
			UnitSystem:   "imperial",
		},
	}

	// Apply to a default config
	cfg := DefaultConfig()
	ps.ApplyTo(cfg)

	// Verify thresholds were applied
	if cfg.Thresholds.DNS.Warning != 100*time.Millisecond {
		t.Errorf("DNS warning: expected 100ms, got %v", cfg.Thresholds.DNS.Warning)
	}
	if cfg.Thresholds.DNS.Critical != 200*time.Millisecond {
		t.Errorf("DNS critical: expected 200ms, got %v", cfg.Thresholds.DNS.Critical)
	}
	if cfg.Thresholds.Ping.Warning != 50*time.Millisecond {
		t.Errorf("Ping warning: expected 50ms, got %v", cfg.Thresholds.Ping.Warning)
	}
	if cfg.Thresholds.WiFi.Signal.Warning != -65 {
		t.Errorf("WiFi warning: expected -65, got %d", cfg.Thresholds.WiFi.Signal.Warning)
	}

	// Verify health checks were applied
	if !cfg.HealthChecks.RunPerformance {
		t.Error("expected RunPerformance to be true")
	}
	if !cfg.HealthChecks.RunSpeedtest {
		t.Error("expected RunSpeedtest to be true")
	}
	if cfg.HealthChecks.RunIperf {
		t.Error("expected RunIperf to be false")
	}
	if len(cfg.HealthChecks.PingTargets) != 1 {
		t.Errorf("expected 1 ping target, got %d", len(cfg.HealthChecks.PingTargets))
	}
	if cfg.HealthChecks.PingTargets[0].Name != "Router" {
		t.Errorf("expected ping target name 'Router', got %s", cfg.HealthChecks.PingTargets[0].Name)
	}

	// Verify DNS was applied
	if cfg.DNS.TestHostname != "test.example.com" {
		t.Errorf("expected DNS hostname 'test.example.com', got %s", cfg.DNS.TestHostname)
	}
	if cfg.DNS.Timeout != 3*time.Second {
		t.Errorf("expected DNS timeout 3s, got %v", cfg.DNS.Timeout)
	}
	if len(cfg.DNS.Servers) != 1 {
		t.Errorf("expected 1 DNS server, got %d", len(cfg.DNS.Servers))
	}

	// Verify SNMP was applied
	if len(cfg.SNMP.Communities) != 1 || cfg.SNMP.Communities[0] != "community1" {
		t.Errorf("expected SNMP community 'community1', got %v", cfg.SNMP.Communities)
	}
	if cfg.SNMP.Port != 1161 {
		t.Errorf("expected SNMP port 1161, got %d", cfg.SNMP.Port)
	}

	// Verify network discovery was applied
	if !cfg.NetworkDiscovery.Enabled {
		t.Error("expected discovery enabled")
	}
	if cfg.NetworkDiscovery.AutoScan {
		t.Error("expected auto scan to be false")
	}

	// Verify FAB options were applied
	if cfg.FABOptions.RunLink {
		t.Error("expected FABOptions.RunLink to be false")
	}
	if !cfg.FABOptions.RunSpeedtest {
		t.Error("expected FABOptions.RunSpeedtest to be true")
	}

	// Verify display options were applied
	if cfg.DisplayOptions.ShowPublicIP {
		t.Error("expected ShowPublicIP to be false")
	}
	if cfg.DisplayOptions.UnitSystem != "imperial" {
		t.Errorf("expected unit system 'imperial', got %s", cfg.DisplayOptions.UnitSystem)
	}
}

func TestProfileSettingsJSONRoundTrip(t *testing.T) {
	// Create profile settings with various values
	original := &ProfileSettings{
		Version: ProfileSettingsVersion,
		Thresholds: ProfileThresholds{
			DNS:     ThresholdPair{Warning: 50, Critical: 100},
			Gateway: ThresholdPair{Warning: 25, Critical: 75},
			WiFi:    WiFiThresholdPair{Warning: -70, Critical: -80},
		},
		HealthChecks: ProfileHealthChecks{
			RunPerformance: true,
			PingTargets: []ProfilePingTarget{
				{Name: "Test", Host: "1.2.3.4", Enabled: true},
			},
		},
		DNS: ProfileDNS{
			TestHostname: "roundtrip.test",
			Timeout:      2000,
		},
		Notes: "Test profile for unit testing",
	}

	// Serialize to JSON
	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify JSON is valid
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Deserialize back
	restored := NewProfileSettings()
	if err := restored.FromJSON(jsonStr); err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// Verify key fields match
	if restored.Version != original.Version {
		t.Errorf("Version mismatch: expected %d, got %d", original.Version, restored.Version)
	}
	if restored.Thresholds.DNS.Warning != original.Thresholds.DNS.Warning {
		t.Errorf("DNS warning mismatch: expected %d, got %d",
			original.Thresholds.DNS.Warning, restored.Thresholds.DNS.Warning)
	}
	if restored.DNS.TestHostname != original.DNS.TestHostname {
		t.Errorf("DNS hostname mismatch: expected %s, got %s",
			original.DNS.TestHostname, restored.DNS.TestHostname)
	}
	if restored.Notes != original.Notes {
		t.Errorf("Notes mismatch: expected %s, got %s", original.Notes, restored.Notes)
	}
	if len(restored.HealthChecks.PingTargets) != len(original.HealthChecks.PingTargets) {
		t.Errorf("Ping targets count mismatch: expected %d, got %d",
			len(original.HealthChecks.PingTargets), len(restored.HealthChecks.PingTargets))
	}
}

func TestParseProfileSettings(t *testing.T) {
	jsonStr := `{
		"version": 1,
		"thresholds": {
			"dns": {"warning": 60, "critical": 120},
			"gateway": {"warning": 30, "critical": 90}
		},
		"dns": {
			"test_hostname": "parsed.test",
			"timeout_ms": 4000
		},
		"notes": "Parsed from JSON string"
	}`

	ps, err := ParseProfileSettings(jsonStr)
	if err != nil {
		t.Fatalf("ParseProfileSettings failed: %v", err)
	}

	// After parsing, v1 profiles are migrated to current version.
	if ps.Version != ProfileSettingsVersion {
		t.Errorf("expected version %d (migrated from v1), got %d", ProfileSettingsVersion, ps.Version)
	}
	if ps.Thresholds.DNS.Warning != 60 {
		t.Errorf("expected DNS warning 60, got %d", ps.Thresholds.DNS.Warning)
	}
	if ps.DNS.TestHostname != "parsed.test" {
		t.Errorf("expected hostname 'parsed.test', got %s", ps.DNS.TestHostname)
	}
	if ps.Notes != "Parsed from JSON string" {
		t.Errorf("expected notes 'Parsed from JSON string', got %s", ps.Notes)
	}
}

func TestParseProfileSettingsEmptyString(t *testing.T) {
	ps, err := ParseProfileSettings("")
	if err != nil {
		t.Fatalf("ParseProfileSettings with empty string should not error: %v", err)
	}
	if ps.Version != ProfileSettingsVersion {
		t.Errorf("expected version %d for empty string, got %d", ProfileSettingsVersion, ps.Version)
	}
}

func TestParseProfileSettingsInvalidJSON(t *testing.T) {
	_, err := ParseProfileSettings("{invalid json}")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestProfileSettingsMigration(t *testing.T) {
	// V1 profile should be migrated to V2.
	v1JSON := `{"version": 1, "notes": "v1 profile"}`
	ps, err := ParseProfileSettings(v1JSON)
	if err != nil {
		t.Fatalf("ParseProfileSettings failed: %v", err)
	}

	if ps.Version != ProfileSettingsVersion {
		t.Errorf("expected version %d after migration, got %d", ProfileSettingsVersion, ps.Version)
	}

	// Notes should be preserved after migration.
	if ps.Notes != "v1 profile" {
		t.Errorf("expected notes 'v1 profile', got %s", ps.Notes)
	}

	// Interfaces should be empty (will be configured by user).
	if ps.Interfaces.Ethernet != nil {
		t.Error("expected nil Ethernet interface after migration")
	}
	if ps.Interfaces.WiFi != nil {
		t.Error("expected nil WiFi interface after migration")
	}
}

func TestProfileSettingsInterfaceSelection(t *testing.T) {
	ps := NewProfileSettings()

	// Set ethernet interface.
	ps.SetEthernetInterface("eth0", true)
	if ps.GetEthernetInterfaceName() != "eth0" {
		t.Errorf("expected ethernet interface 'eth0', got '%s'", ps.GetEthernetInterfaceName())
	}
	// Verify the interface is in the array and enabled
	ethIface := ps.GetActiveEthernetInterface()
	if ethIface == nil || !ethIface.Enabled {
		t.Error("expected ethernet interface to be enabled")
	}

	// Set WiFi interface.
	ps.SetWiFiInterface("wlan0", true)
	if ps.GetWiFiInterfaceName() != "wlan0" {
		t.Errorf("expected WiFi interface 'wlan0', got '%s'", ps.GetWiFiInterfaceName())
	}
	// Verify the interface is in the array and enabled
	wifiIface := ps.GetActiveWiFiInterface()
	if wifiIface == nil || !wifiIface.Enabled {
		t.Error("expected WiFi interface to be enabled")
	}

	// Test empty interface names.
	ps2 := NewProfileSettings()
	if ps2.GetEthernetInterfaceName() != "" {
		t.Error("expected empty ethernet interface name for new profile")
	}
	if ps2.GetWiFiInterfaceName() != "" {
		t.Error("expected empty WiFi interface name for new profile")
	}
}

func TestProfileSettingsInterfaceJSON(t *testing.T) {
	ps := NewProfileSettings()
	ps.SetEthernetInterface("enp0s1", true)
	ps.SetWiFiInterface("wlp2s0", false)

	// Serialize to JSON.
	jsonStr, err := ps.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Parse back.
	ps2, err := ParseProfileSettings(jsonStr)
	if err != nil {
		t.Fatalf("ParseProfileSettings failed: %v", err)
	}

	if ps2.GetEthernetInterfaceName() != "enp0s1" {
		t.Errorf("expected ethernet 'enp0s1', got '%s'", ps2.GetEthernetInterfaceName())
	}
	// Verify the interface is in the array and enabled
	ethIface := ps2.GetActiveEthernetInterface()
	if ethIface == nil || !ethIface.Enabled {
		t.Error("expected ethernet to be enabled")
	}

	if ps2.GetWiFiInterfaceName() != "wlp2s0" {
		t.Errorf("expected WiFi 'wlp2s0', got '%s'", ps2.GetWiFiInterfaceName())
	}
	// Verify the interface is in the array and disabled
	wifiIface := ps2.GetActiveWiFiInterface()
	if wifiIface == nil || wifiIface.Enabled {
		t.Error("expected WiFi to be disabled")
	}
}

func TestProfileSettingsRoundTripThroughConfig(t *testing.T) {
	// Create a config with specific settings
	originalCfg := DefaultConfig()
	originalCfg.Thresholds.DNS.Warning = 75 * time.Millisecond
	originalCfg.Thresholds.DNS.Critical = 150 * time.Millisecond
	originalCfg.DNS.TestHostname = "roundtrip.example.com"
	originalCfg.DNS.Servers = []DNSServer{
		{Address: "4.4.4.4", Enabled: true},
		{Address: "5.5.5.5", Enabled: false},
	}
	originalCfg.HealthChecks.PingTargets = []PingTarget{
		{Name: "Target1", Host: "10.0.0.1", Enabled: true},
		{Name: "Target2", Host: "10.0.0.2", Enabled: false},
	}
	originalCfg.FABOptions.RunSpeedtest = true
	originalCfg.FABOptions.RunIperf = false

	// Extract settings from config
	ps := NewProfileSettings()
	ps.FromConfig(originalCfg)
	ps.Notes = "Test notes for round trip"

	// Serialize to JSON (simulates database storage)
	jsonStr, err := ps.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Parse from JSON (simulates database retrieval)
	restoredPS, err := ParseProfileSettings(jsonStr)
	if err != nil {
		t.Fatalf("ParseProfileSettings failed: %v", err)
	}

	// Apply to a fresh config
	newCfg := DefaultConfig()
	restoredPS.ApplyTo(newCfg)

	// Verify settings match original config
	if newCfg.Thresholds.DNS.Warning != originalCfg.Thresholds.DNS.Warning {
		t.Errorf("DNS warning mismatch: expected %v, got %v",
			originalCfg.Thresholds.DNS.Warning, newCfg.Thresholds.DNS.Warning)
	}
	if newCfg.DNS.TestHostname != originalCfg.DNS.TestHostname {
		t.Errorf("DNS hostname mismatch: expected %s, got %s",
			originalCfg.DNS.TestHostname, newCfg.DNS.TestHostname)
	}
	if len(newCfg.DNS.Servers) != len(originalCfg.DNS.Servers) {
		t.Errorf("DNS servers count mismatch: expected %d, got %d",
			len(originalCfg.DNS.Servers), len(newCfg.DNS.Servers))
	}
	if len(newCfg.HealthChecks.PingTargets) != len(originalCfg.HealthChecks.PingTargets) {
		t.Errorf("Ping targets count mismatch: expected %d, got %d",
			len(originalCfg.HealthChecks.PingTargets), len(newCfg.HealthChecks.PingTargets))
	}
	if newCfg.FABOptions.RunSpeedtest != originalCfg.FABOptions.RunSpeedtest {
		t.Errorf("FAB RunSpeedtest mismatch: expected %v, got %v",
			originalCfg.FABOptions.RunSpeedtest, newCfg.FABOptions.RunSpeedtest)
	}
	if newCfg.FABOptions.RunIperf != originalCfg.FABOptions.RunIperf {
		t.Errorf("FAB RunIperf mismatch: expected %v, got %v",
			originalCfg.FABOptions.RunIperf, newCfg.FABOptions.RunIperf)
	}
}

func TestMultipleProfilesWithDifferentSettings(t *testing.T) {
	// Simulate two different profiles with different settings
	profile1Settings := &ProfileSettings{
		Version: ProfileSettingsVersion,
		Thresholds: ProfileThresholds{
			DNS:     ThresholdPair{Warning: 50, Critical: 100},
			Gateway: ThresholdPair{Warning: 20, Critical: 50},
		},
		DNS: ProfileDNS{
			TestHostname: "profile1.test",
			Timeout:      2000,
		},
		FABOptions: ProfileFABOptions{
			RunSpeedtest: true,
			RunIperf:     false,
		},
		Notes: "Profile 1 - Fast network",
	}

	profile2Settings := &ProfileSettings{
		Version: ProfileSettingsVersion,
		Thresholds: ProfileThresholds{
			DNS:     ThresholdPair{Warning: 200, Critical: 500},
			Gateway: ThresholdPair{Warning: 100, Critical: 300},
		},
		DNS: ProfileDNS{
			TestHostname: "profile2.test",
			Timeout:      10000,
		},
		FABOptions: ProfileFABOptions{
			RunSpeedtest: false,
			RunIperf:     true,
		},
		Notes: "Profile 2 - Slow network",
	}

	// Serialize both profiles
	json1, _ := profile1Settings.ToJSON()
	json2, _ := profile2Settings.ToJSON()

	// Apply profile 1
	cfg := DefaultConfig()
	ps1, _ := ParseProfileSettings(json1)
	ps1.ApplyTo(cfg)

	if cfg.Thresholds.DNS.Warning != 50*time.Millisecond {
		t.Errorf("Profile 1: expected DNS warning 50ms, got %v", cfg.Thresholds.DNS.Warning)
	}
	if cfg.DNS.TestHostname != "profile1.test" {
		t.Errorf("Profile 1: expected hostname 'profile1.test', got %s", cfg.DNS.TestHostname)
	}
	if !cfg.FABOptions.RunSpeedtest {
		t.Error("Profile 1: expected RunSpeedtest true")
	}

	// Switch to profile 2
	ps2, _ := ParseProfileSettings(json2)
	ps2.ApplyTo(cfg)

	if cfg.Thresholds.DNS.Warning != 200*time.Millisecond {
		t.Errorf("Profile 2: expected DNS warning 200ms, got %v", cfg.Thresholds.DNS.Warning)
	}
	if cfg.DNS.TestHostname != "profile2.test" {
		t.Errorf("Profile 2: expected hostname 'profile2.test', got %s", cfg.DNS.TestHostname)
	}
	if cfg.FABOptions.RunSpeedtest {
		t.Error("Profile 2: expected RunSpeedtest false")
	}
	if !cfg.FABOptions.RunIperf {
		t.Error("Profile 2: expected RunIperf true")
	}

	// Switch back to profile 1 to verify settings restore correctly
	ps1Again, _ := ParseProfileSettings(json1)
	ps1Again.ApplyTo(cfg)

	if cfg.Thresholds.DNS.Warning != 50*time.Millisecond {
		t.Errorf("Profile 1 (restored): expected DNS warning 50ms, got %v", cfg.Thresholds.DNS.Warning)
	}
	if cfg.DNS.TestHostname != "profile1.test" {
		t.Errorf("Profile 1 (restored): expected hostname 'profile1.test', got %s", cfg.DNS.TestHostname)
	}
}

func TestProfileSettingsPreservesNotes(t *testing.T) {
	// Create settings with notes
	ps1 := NewProfileSettings()
	ps1.Notes = "Important client notes"

	// Extract from config (should not have notes)
	cfg := DefaultConfig()
	ps2 := NewProfileSettings()
	ps2.FromConfig(cfg)

	// Verify notes are empty after FromConfig
	if ps2.Notes != "" {
		t.Errorf("expected empty notes after FromConfig, got %s", ps2.Notes)
	}

	// Verify notes are preserved in original
	if ps1.Notes != "Important client notes" {
		t.Errorf("expected notes to be preserved, got %s", ps1.Notes)
	}

	// Serialize and restore with notes
	jsonData, _ := ps1.ToJSON()
	restored, _ := ParseProfileSettings(jsonData)
	if restored.Notes != "Important client notes" {
		t.Errorf("notes not preserved through JSON round trip, got %s", restored.Notes)
	}
}

func TestSNMPv3CredentialsRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SNMP.V3Credentials = []SNMPv3Credential{
		{
			Name:          "Device1",
			Username:      "admin",
			AuthProtocol:  "SHA",
			AuthPassword:  "authpass123",
			PrivProtocol:  "AES",
			PrivPassword:  "privpass456",
			SecurityLevel: "authPriv",
		},
		{
			Name:          "Device2",
			Username:      "readonly",
			AuthProtocol:  "MD5",
			AuthPassword:  "readpass",
			SecurityLevel: "authNoPriv",
		},
	}

	// Extract settings
	ps := NewProfileSettings()
	ps.FromConfig(cfg)

	// Verify extraction
	if len(ps.SNMP.V3Credentials) != 2 {
		t.Fatalf("expected 2 V3 credentials, got %d", len(ps.SNMP.V3Credentials))
	}
	if ps.SNMP.V3Credentials[0].Name != "Device1" {
		t.Errorf("expected first credential name 'Device1', got %s", ps.SNMP.V3Credentials[0].Name)
	}
	if ps.SNMP.V3Credentials[0].AuthProtocol != "SHA" {
		t.Errorf("expected auth protocol 'SHA', got %s", ps.SNMP.V3Credentials[0].AuthProtocol)
	}

	// JSON round trip
	jsonStr, _ := ps.ToJSON()
	restored, _ := ParseProfileSettings(jsonStr)

	// Apply to new config
	newCfg := DefaultConfig()
	restored.ApplyTo(newCfg)

	// Verify restoration
	if len(newCfg.SNMP.V3Credentials) != 2 {
		t.Fatalf("expected 2 V3 credentials after restore, got %d", len(newCfg.SNMP.V3Credentials))
	}
	if newCfg.SNMP.V3Credentials[0].Username != "admin" {
		t.Errorf("expected username 'admin', got %s", newCfg.SNMP.V3Credentials[0].Username)
	}
	if newCfg.SNMP.V3Credentials[1].SecurityLevel != "authNoPriv" {
		t.Errorf("expected security level 'authNoPriv', got %s", newCfg.SNMP.V3Credentials[1].SecurityLevel)
	}
}
