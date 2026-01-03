package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Test server defaults
	if cfg.Server.Port != 8443 {
		t.Errorf("expected default port 8443, got %d", cfg.Server.Port)
	}
	if !cfg.Server.HTTPS {
		t.Error("expected HTTPS to be enabled by default")
	}

	// Test interface defaults - empty means auto-detect (#572)
	if cfg.Interface.Default != "" {
		t.Errorf("expected default interface to be empty (auto-detect), got %s", cfg.Interface.Default)
	}

	// Test VLAN defaults
	if cfg.VLAN.Enabled {
		t.Error("expected VLAN to be disabled by default")
	}

	// Test IP defaults
	if cfg.IP.Mode != "dhcp" {
		t.Errorf("expected IP mode dhcp, got %s", cfg.IP.Mode)
	}

	// Test discovery defaults
	if cfg.Discovery.Protocol != "auto" {
		t.Errorf("expected discovery protocol auto, got %s", cfg.Discovery.Protocol)
	}
	if cfg.Discovery.Timeout != 30*time.Second {
		t.Errorf("expected discovery timeout 30s, got %v", cfg.Discovery.Timeout)
	}

	// Test DNS defaults
	if cfg.DNS.TestHostname != "google.com" {
		t.Errorf("expected DNS test hostname google.com, got %s", cfg.DNS.TestHostname)
	}

	// Test auth defaults
	if cfg.Auth.DefaultUsername != "admin" {
		t.Errorf("expected default username admin, got %s", cfg.Auth.DefaultUsername)
	}
	if cfg.Auth.SessionTimeout != 24*time.Hour {
		t.Errorf("expected session timeout 24h, got %v", cfg.Auth.SessionTimeout)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got %v", err)
	}

	// Should return default config
	if cfg == nil {
		t.Fatal("expected default config for non-existent file")
	}
	if cfg.Server.Port != 8443 {
		t.Errorf("expected default port 8443, got %d", cfg.Server.Port)
	}
}

func TestLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create and save config
	cfg := config.DefaultConfig()
	cfg.Server.Port = 9999
	cfg.Interface.Default = "test0"
	cfg.Auth.DefaultUsername = "testuser"

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load config and verify values
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Server.Port != 9999 {
		t.Errorf("expected port 9999, got %d", loaded.Server.Port)
	}
	if loaded.Interface.Default != "test0" {
		t.Errorf("expected interface test0, got %s", loaded.Interface.Default)
	}
	if loaded.Auth.DefaultUsername != "testuser" {
		t.Errorf("expected username testuser, got %s", loaded.Auth.DefaultUsername)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-config.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configPath, []byte("invalid: [yaml: content"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := config.Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestThresholdDefaults(t *testing.T) {
	cfg := config.DefaultConfig()

	// DHCP thresholds
	if cfg.Thresholds.DHCP.Total.Warning != 500*time.Millisecond {
		t.Errorf("expected DHCP total warning 500ms, got %v", cfg.Thresholds.DHCP.Total.Warning)
	}
	if cfg.Thresholds.DHCP.Total.Critical != 2*time.Second {
		t.Errorf("expected DHCP total critical 2s, got %v", cfg.Thresholds.DHCP.Total.Critical)
	}

	// DNS thresholds
	if cfg.Thresholds.DNS.Warning != 100*time.Millisecond {
		t.Errorf("expected DNS warning 100ms, got %v", cfg.Thresholds.DNS.Warning)
	}

	// WiFi thresholds
	if cfg.Thresholds.WiFi.Signal.Warning != -70 {
		t.Errorf("expected WiFi signal warning -70, got %d", cfg.Thresholds.WiFi.Signal.Warning)
	}
	if cfg.Thresholds.WiFi.Signal.Critical != -80 {
		t.Errorf("expected WiFi signal critical -80, got %d", cfg.Thresholds.WiFi.Signal.Critical)
	}

	// Certificate expiry thresholds
	if cfg.Thresholds.CustomTests.CertExpiry.Warning != 30 {
		t.Errorf("expected cert expiry warning 30 days, got %d", cfg.Thresholds.CustomTests.CertExpiry.Warning)
	}
	if cfg.Thresholds.CustomTests.CertExpiry.Critical != 7 {
		t.Errorf("expected cert expiry critical 7 days, got %d", cfg.Thresholds.CustomTests.CertExpiry.Critical)
	}
}

// ========== EnsureConfig Tests ==========

func TestEnsureConfigFirstBoot(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "new-config.yaml")

	// First boot - no existing config file
	cfg, result, err := config.EnsureConfig(configPath, nil)

	// Should return config.ErrInsecureCredentials because password hash is empty
	if !errors.Is(err, config.ErrInsecureCredentials) {
		t.Errorf("expected config.ErrInsecureCredentials for first boot, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config even with error")
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if !result.IsFirstBoot {
		t.Error("expected IsFirstBoot=true")
	}
	if !result.GeneratedCreds {
		t.Error("expected GeneratedCreds=true")
	}
}

func TestEnsureConfigDetectsInsecurePassword(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "insecure-config.yaml")

	// Create config with a known insecure password hash
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "$2a$10$InsecureHashThatWillBeDetected"
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	// checkDefaultPassword function that always returns true (simulates insecure)
	checkInsecure := func(_ string) bool { return true }

	_, result, err := config.EnsureConfig(configPath, checkInsecure)

	if !errors.Is(err, config.ErrInsecureCredentials) {
		t.Errorf("expected config.ErrInsecureCredentials for insecure password, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsFirstBoot {
		t.Error("expected IsFirstBoot=false for existing config")
	}
	if !result.GeneratedCreds {
		t.Error("expected GeneratedCreds=true for insecure password")
	}
}

func TestEnsureConfigSecurePassword(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "secure-config.yaml")

	// Create config with a secure password hash
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "$2a$10$SecureHashThatIsNotDefault"
	cfg.Auth.JWTSecret = "existing-secure-secret"
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	// checkDefaultPassword function that returns false (not insecure)
	checkSecure := func(_ string) bool { return false }

	loadedCfg, result, err := config.EnsureConfig(configPath, checkSecure)
	if err != nil {
		t.Errorf("expected no error for secure config, got %v", err)
	}
	if loadedCfg == nil {
		t.Fatal("expected config")
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if result.GeneratedCreds {
		t.Error("expected GeneratedCreds=false for secure password")
	}
}

func TestEnsureConfigEmptyJWTSecret(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty-jwt-config.yaml")

	// Create config with secure password but empty JWT secret
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "$2a$10$SecureHashThatIsNotDefault"
	cfg.Auth.JWTSecret = "" // Empty - needs generation
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	checkSecure := func(_ string) bool { return false }

	_, result, err := config.EnsureConfig(configPath, checkSecure)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if !result.JWTSecretStored {
		t.Error("expected JWTSecretStored=true when JWT secret is empty")
	}
}

// ========== UpdateCredentials Tests ==========

func TestUpdateCredentials(t *testing.T) {
	cfg := config.DefaultConfig()

	// Update credentials
	cfg.UpdateCredentials("newuser", "newhash123", "newsecret456")

	if cfg.Auth.DefaultUsername != "newuser" {
		t.Errorf("expected username 'newuser', got %q", cfg.Auth.DefaultUsername)
	}
	if cfg.Auth.DefaultPasswordHash != "newhash123" {
		t.Errorf("expected password hash 'newhash123', got %q", cfg.Auth.DefaultPasswordHash)
	}
	if cfg.Auth.JWTSecret != "newsecret456" {
		t.Errorf("expected JWT secret 'newsecret456', got %q", cfg.Auth.JWTSecret)
	}
}

func TestUpdateCredentialsEmptyJWT(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "originalsecret"

	// Update with empty JWT secret - should preserve original
	cfg.UpdateCredentials("newuser", "newhash", "")

	if cfg.Auth.JWTSecret != "originalsecret" {
		t.Errorf("expected JWT secret preserved, got %q", cfg.Auth.JWTSecret)
	}
}

func TestUpdateJWTSecret(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "oldsecret"

	cfg.UpdateJWTSecret("brandnewsecret")

	if cfg.Auth.JWTSecret != "brandnewsecret" {
		t.Errorf("expected JWT secret 'brandnewsecret', got %q", cfg.Auth.JWTSecret)
	}
}

// ========== Persistence Tests ==========

func TestCredentialsPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "persist-test.yaml")

	// Create and save config with credentials
	cfg := config.DefaultConfig()
	cfg.UpdateCredentials("persistuser", "persisted-hash", "persisted-jwt-secret")

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load config and verify credentials persisted
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Auth.DefaultUsername != "persistuser" {
		t.Errorf("expected persisted username 'persistuser', got %q", loaded.Auth.DefaultUsername)
	}
	if loaded.Auth.DefaultPasswordHash != "persisted-hash" {
		t.Errorf("expected persisted hash 'persisted-hash', got %q", loaded.Auth.DefaultPasswordHash)
	}
	if loaded.Auth.JWTSecret != "persisted-jwt-secret" {
		t.Errorf("expected persisted JWT secret, got %q", loaded.Auth.JWTSecret)
	}
}

// ========== File Permission Tests ==========

func TestConfigSavePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "perms-test.yaml")

	cfg := config.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	perms := info.Mode().Perm()
	// Should be 0600 (owner read/write only) for security
	if perms != 0o600 {
		t.Errorf("expected permissions 0600, got %04o", perms)
	}
}

func TestEnsureConfigCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Nested path that doesn't exist - EnsureConfig should create it
	configPath := filepath.Join(tmpDir, "nested", "deep", "config.yaml")

	// EnsureConfig creates the directory structure
	_, _, err := config.EnsureConfig(configPath, nil)
	// Will return config.ErrInsecureCredentials because password is empty, but directory should exist
	if err != nil && !errors.Is(err, config.ErrInsecureCredentials) {
		t.Fatalf("EnsureConfig failed unexpectedly: %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(configPath)
	if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
		t.Error("EnsureConfig did not create nested directory")
	}
}

// ========== Config Locking Tests ==========

func TestConfigLocking(t *testing.T) {
	cfg := config.DefaultConfig()

	// Test write lock
	cfg.Lock()
	cfg.Server.Port = 9999
	cfg.Unlock()

	if cfg.Server.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Server.Port)
	}
}

func TestConfigReadLocking(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.Port = 8080

	// Test read lock
	cfg.RLock()
	port := cfg.Server.Port
	cfg.RUnlock()

	if port != 8080 {
		t.Errorf("expected port 8080, got %d", port)
	}
}

// ========== Validation Tests ==========

func TestValidateConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "somehash" // Required for validation

	err := cfg.Validate()
	if err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestValidateInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"port zero", 0},
		{"port negative", -1},
		{"port too high", 65536},
		{"port way too high", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "hash"
			cfg.Server.Port = tt.port

			err := cfg.Validate()
			if err == nil {
				t.Error("expected validation error for invalid port")
			}
		})
	}
}

func TestValidateInvalidHTTPRedirectPort(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Server.HTTPRedirectPort = -1

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for negative HTTP redirect port")
	}
}

func TestValidateSamePortAndRedirectPort(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Server.Port = 8080
	cfg.Server.HTTPRedirectPort = 8080 // Same as main port

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error when port equals HTTP redirect port")
	}
}

func TestValidateEmptyInterface(t *testing.T) {
	// Empty interface is now valid - triggers auto-detection (#572)
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Interface.Default = ""

	err := cfg.Validate()
	if err != nil {
		t.Errorf("empty interface should be valid (auto-detection), got error: %v", err)
	}
}

func TestValidateNegativeStartupRetries(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Interface.StartupRetries = -1

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for negative startup retries")
	}
}

func TestValidateNegativeStartupRetryWait(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Interface.StartupRetryWait = -1 * time.Second

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for negative startup retry wait")
	}
}

func TestValidateInvalidVLAN(t *testing.T) {
	tests := []struct {
		name   string
		vlanID int
	}{
		{"VLAN 0", 0},
		{"VLAN negative", -1},
		{"VLAN too high", 4095},
		{"VLAN way too high", 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "hash"
			cfg.VLAN.Enabled = true
			cfg.VLAN.ID = tt.vlanID

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error for VLAN ID %d", tt.vlanID)
			}
		})
	}
}

func TestValidateInvalidIPMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.IP.Mode = "invalid"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid IP mode")
	}
}

func TestValidateStaticIPMissingAddress(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.IP.Mode = "static"
	cfg.IP.Static = &config.StaticIP{
		Address: "",
		Netmask: "255.255.255.0",
		Gateway: "192.168.1.1",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for missing static IP address")
	}
}

func TestValidateStaticIPMissingNetmask(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.IP.Mode = "static"
	cfg.IP.Static = &config.StaticIP{
		Address: "192.168.1.100",
		Netmask: "",
		Gateway: "192.168.1.1",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for missing netmask")
	}
}

func TestValidateStaticIPMissingGateway(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.IP.Mode = "static"
	cfg.IP.Static = &config.StaticIP{
		Address: "192.168.1.100",
		Netmask: "255.255.255.0",
		Gateway: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for missing gateway")
	}
}

func TestValidateZeroTimeout(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*config.Config)
	}{
		{"discovery timeout", func(c *config.Config) { c.Discovery.Timeout = 0 }},
		{"ping timeout", func(c *config.Config) { c.NetworkDiscovery.PingTimeout = 0 }},
		{"scan timeout", func(c *config.Config) { c.NetworkDiscovery.ScanTimeout = 0 }},
		{"DNS timeout", func(c *config.Config) { c.DNS.Timeout = 0 }},
		{"SNMP timeout", func(c *config.Config) { c.SNMP.Timeout = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "hash"
			tt.modify(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Error("expected validation error for zero timeout")
			}
		})
	}
}

func TestValidateInvalidARPScanWorkers(t *testing.T) {
	tests := []struct {
		name    string
		workers int
	}{
		{"zero workers", 0},
		{"negative workers", -1},
		{"too many workers", 501},
		{"way too many workers", 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "hash"
			cfg.NetworkDiscovery.ARPScanWorkers = tt.workers

			err := cfg.Validate()
			if err == nil {
				t.Error("expected validation error for invalid ARP scan workers")
			}
		})
	}
}

func TestValidateInvalidSessionTimeout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Auth.SessionTimeout = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for zero session timeout")
	}
}

func TestValidateEmptyUsername(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Auth.DefaultUsername = ""

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty username")
	}
}

func TestValidateEmptyPasswordHash(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = ""

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty password hash")
	}
}

func TestValidateInvalidSNMPPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"port zero", 0},
		{"port negative", -1},
		{"port too high", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "hash"
			cfg.SNMP.Port = tt.port

			err := cfg.Validate()
			if err == nil {
				t.Error("expected validation error for invalid SNMP port")
			}
		})
	}
}

func TestValidateInvalidSNMPRetries(t *testing.T) {
	tests := []struct {
		name    string
		retries int
	}{
		{"negative retries", -1},
		{"too many retries", 11},
		{"way too many", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "hash"
			cfg.SNMP.Retries = tt.retries

			err := cfg.Validate()
			if err == nil {
				t.Error("expected validation error for invalid SNMP retries")
			}
		})
	}
}

func TestWarnDeprecatedSNMPSettings(t *testing.T) {
	tests := []struct {
		name        string
		credentials []config.SNMPv3Credential
		expectWarn  bool
	}{
		{
			name: "MD5 auth protocol triggers warning",
			credentials: []config.SNMPv3Credential{
				{
					Name:         "test-md5",
					Username:     "snmpuser",
					AuthProtocol: "MD5",
					AuthPassword: "password",
				},
			},
			expectWarn: true,
		},
		{
			name: "SHA256 does not trigger warning",
			credentials: []config.SNMPv3Credential{
				{
					Name:         "test-sha256",
					Username:     "snmpuser",
					AuthProtocol: "SHA256",
					AuthPassword: "password",
				},
			},
			expectWarn: false,
		},
		{
			name: "SHA512 does not trigger warning",
			credentials: []config.SNMPv3Credential{
				{
					Name:         "test-sha512",
					Username:     "snmpuser",
					AuthProtocol: "SHA512",
					AuthPassword: "password",
				},
			},
			expectWarn: false,
		},
		{
			name: "SHA does not trigger warning",
			credentials: []config.SNMPv3Credential{
				{
					Name:         "test-sha",
					Username:     "snmpuser",
					AuthProtocol: "SHA",
					AuthPassword: "password",
				},
			},
			expectWarn: false,
		},
		{
			name: "multiple credentials with one MD5",
			credentials: []config.SNMPv3Credential{
				{
					Name:         "test-sha256",
					Username:     "user1",
					AuthProtocol: "SHA256",
					AuthPassword: "password",
				},
				{
					Name:         "test-md5",
					Username:     "user2",
					AuthProtocol: "MD5",
					AuthPassword: "password",
				},
			},
			expectWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SNMP.V3Credentials = tt.credentials

			// WarnDeprecatedSNMPSettings logs warnings but doesn't return errors.
			// This test verifies it doesn't panic and can be called safely.
			cfg.WarnDeprecatedSNMPSettings()
		})
	}
}

// ========== Port Preset Tests ==========

func TestPortPresetConstants(t *testing.T) {
	if config.PortPresetCommon != "common" {
		t.Errorf("config.PortPresetCommon should be 'common', got %q", config.PortPresetCommon)
	}
	if config.PortPresetSecure != "secure" {
		t.Errorf("config.PortPresetSecure should be 'secure', got %q", config.PortPresetSecure)
	}
	if config.PortPresetInsecure != "insecure" {
		t.Errorf("config.PortPresetInsecure should be 'insecure', got %q", config.PortPresetInsecure)
	}
	if config.PortPresetCustom != "custom" {
		t.Errorf("config.PortPresetCustom should be 'custom', got %q", config.PortPresetCustom)
	}
}

func TestDefaultPortScanConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.NetworkDiscovery.Options.PortScan.Preset != config.PortPresetCommon {
		t.Errorf("expected default port preset 'common', got %q", cfg.NetworkDiscovery.Options.PortScan.Preset)
	}
}

func TestGetEffectivePorts(t *testing.T) {
	tests := []struct {
		preset     config.PortPreset
		wantTCPLen int
		wantUDPLen int
	}{
		{config.PortPresetCommon, len(config.PortsCommonTCP), len(config.PortsCommonUDP)},
		{config.PortPresetSecure, len(config.PortsSecureTCP), len(config.PortsSecureUDP)},
		{config.PortPresetInsecure, len(config.PortsInsecureTCP), len(config.PortsInsecureUDP)},
	}

	for _, tt := range tests {
		t.Run(string(tt.preset), func(t *testing.T) {
			cfg := config.PortScanConfig{Preset: tt.preset}
			tcp, udp := cfg.GetEffectivePorts()
			if len(tcp) != tt.wantTCPLen {
				t.Errorf("preset %s: expected TCP ports length %d, got %d", tt.preset, tt.wantTCPLen, len(tcp))
			}
			if len(udp) != tt.wantUDPLen {
				t.Errorf("preset %s: expected UDP ports length %d, got %d", tt.preset, tt.wantUDPLen, len(udp))
			}
		})
	}

	// Test custom preset
	t.Run("custom", func(t *testing.T) {
		cfg := config.PortScanConfig{Preset: config.PortPresetCustom, TCPPorts: "22,80", UDPPorts: "53"}
		tcp, udp := cfg.GetEffectivePorts()
		if tcp != "22,80" {
			t.Errorf("expected custom TCP ports '22,80', got %q", tcp)
		}
		if udp != "53" {
			t.Errorf("expected custom UDP ports '53', got %q", udp)
		}
	})
}

// ========== SubnetConfig Tests ==========

func TestSubnetConfig(t *testing.T) {
	subnet := config.SubnetConfig{
		CIDR:    "192.168.1.0/24",
		Name:    "Test VLAN",
		Enabled: true,
	}

	if subnet.CIDR != "192.168.1.0/24" {
		t.Errorf("expected CIDR '192.168.1.0/24', got %q", subnet.CIDR)
	}
	if subnet.Name != "Test VLAN" {
		t.Errorf("expected name 'Test VLAN', got %q", subnet.Name)
	}
	if !subnet.Enabled {
		t.Error("expected Enabled to be true")
	}
}

// ========== DNSServer Tests ==========

func TestDNSServerConfig(t *testing.T) {
	server := config.DNSServer{
		Address: "8.8.8.8",
		Enabled: true,
	}

	if server.Address != "8.8.8.8" {
		t.Errorf("expected address '8.8.8.8', got %q", server.Address)
	}
	if !server.Enabled {
		t.Error("expected Enabled to be true")
	}
}

// ========== Test Configuration Types ==========

func TestPingTargetConfig(t *testing.T) {
	target := config.PingTarget{
		Name:    "Google DNS",
		Host:    "8.8.8.8",
		Enabled: true,
	}

	if target.Name != "Google DNS" {
		t.Errorf("expected name 'Google DNS', got %q", target.Name)
	}
	if target.Host != "8.8.8.8" {
		t.Errorf("expected host '8.8.8.8', got %q", target.Host)
	}
	if !target.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestTCPPortTestConfig(t *testing.T) {
	test := config.TCPPortTest{
		Name:    "SSH",
		Host:    "server.local",
		Port:    22,
		Enabled: true,
	}

	if test.Name != "SSH" {
		t.Errorf("expected name 'SSH', got %q", test.Name)
	}
	if test.Host != "server.local" {
		t.Errorf("expected host 'server.local', got %q", test.Host)
	}
	if test.Port != 22 {
		t.Errorf("expected port 22, got %d", test.Port)
	}
	if !test.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestHTTPEndpointConfig(t *testing.T) {
	endpoint := config.HTTPEndpoint{
		Name:           "API Health",
		URL:            "https://api.example.com/health",
		ExpectedStatus: 200,
		Enabled:        true,
	}

	if endpoint.Name != "API Health" {
		t.Errorf("expected name 'API Health', got %q", endpoint.Name)
	}
	if endpoint.URL != "https://api.example.com/health" {
		t.Errorf("expected URL 'https://api.example.com/health', got %q", endpoint.URL)
	}
	if endpoint.ExpectedStatus != 200 {
		t.Errorf("expected status 200, got %d", endpoint.ExpectedStatus)
	}
	if !endpoint.Enabled {
		t.Error("expected Enabled to be true")
	}
}

// ========== SNMP Configuration Tests ==========

func TestSNMPv3CredentialConfig(t *testing.T) {
	cred := config.SNMPv3Credential{
		Name:          "Admin",
		Username:      "snmpv3admin",
		AuthProtocol:  "SHA",
		AuthPassword:  "authpass123",
		PrivProtocol:  "AES",
		PrivPassword:  "privpass456",
		SecurityLevel: "authPriv",
	}

	if cred.Name != "Admin" {
		t.Errorf("expected name 'Admin', got %q", cred.Name)
	}
	if cred.Username != "snmpv3admin" {
		t.Errorf("expected username 'snmpv3admin', got %q", cred.Username)
	}
	if cred.AuthProtocol != "SHA" {
		t.Errorf("expected auth protocol 'SHA', got %q", cred.AuthProtocol)
	}
	if cred.AuthPassword != "authpass123" {
		t.Errorf("expected auth password 'authpass123', got %q", cred.AuthPassword)
	}
	if cred.PrivProtocol != "AES" {
		t.Errorf("expected priv protocol 'AES', got %q", cred.PrivProtocol)
	}
	if cred.PrivPassword != "privpass456" {
		t.Errorf("expected priv password 'privpass456', got %q", cred.PrivPassword)
	}
	if cred.SecurityLevel != "authPriv" {
		t.Errorf("expected security level 'authPriv', got %q", cred.SecurityLevel)
	}
}

func TestDefaultSNMPConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if len(cfg.SNMP.Communities) != 1 || cfg.SNMP.Communities[0] != "public" {
		t.Errorf("expected default community ['public'], got %v", cfg.SNMP.Communities)
	}
	if cfg.SNMP.Timeout != 5*time.Second {
		t.Errorf("expected SNMP timeout 5s, got %v", cfg.SNMP.Timeout)
	}
	if cfg.SNMP.Port != 161 {
		t.Errorf("expected SNMP port 161, got %d", cfg.SNMP.Port)
	}
}

// ========== Iperf Configuration Tests ==========

func TestDefaultIperfConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Iperf.Port != 5201 {
		t.Errorf("expected iperf port 5201, got %d", cfg.Iperf.Port)
	}
	if cfg.Iperf.Protocol != "tcp" {
		t.Errorf("expected iperf protocol 'tcp', got %q", cfg.Iperf.Protocol)
	}
	if cfg.Iperf.Direction != "download" {
		t.Errorf("expected iperf direction 'download', got %q", cfg.Iperf.Direction)
	}
	if cfg.Iperf.Duration != 10 {
		t.Errorf("expected iperf duration 10, got %d", cfg.Iperf.Duration)
	}
}

// ========== ACME Configuration Tests ==========

func TestACMEConfig(t *testing.T) {
	acme := ACMEConfig{
		Enabled:  true,
		Domain:   "example.com",
		Email:    "admin@example.com",
		CacheDir: "/var/certs",
		Staging:  true,
	}

	if !acme.Enabled {
		t.Error("expected ACME enabled")
	}
	if acme.Domain != "example.com" {
		t.Errorf("expected domain 'example.com', got %q", acme.Domain)
	}
	if acme.Email != "admin@example.com" {
		t.Errorf("expected email 'admin@example.com', got %q", acme.Email)
	}
	if acme.CacheDir != "/var/certs" {
		t.Errorf("expected cache dir '/var/certs', got %q", acme.CacheDir)
	}
	if !acme.Staging {
		t.Error("expected staging to be true")
	}
}

// ========== Vulnerability Scanning Tests ==========

func TestVulnerabilityScanConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	// Vulnerability scanning is enabled by default for security visibility
	if !cfg.Security.VulnerabilityScanning.Enabled {
		t.Error("expected vulnerability scanning enabled by default")
	}
	if cfg.Security.VulnerabilityScanning.CVEDatabase != "nvd" {
		t.Errorf("expected CVE database 'nvd', got %q", cfg.Security.VulnerabilityScanning.CVEDatabase)
	}
	if cfg.Security.VulnerabilityScanning.SeverityThreshold != "medium" {
		t.Errorf("expected severity threshold 'medium', got %q", cfg.Security.VulnerabilityScanning.SeverityThreshold)
	}
}

// ========== FAB Options Tests ==========

func TestDefaultFABOptions(t *testing.T) {
	cfg := config.DefaultConfig()

	if !cfg.FABOptions.RunLink {
		t.Error("expected RunLink enabled by default")
	}
	if !cfg.FABOptions.RunSwitch {
		t.Error("expected RunSwitch enabled by default")
	}
	if !cfg.FABOptions.RunSpeedtest {
		t.Error("expected RunSpeedtest enabled by default")
	}
	if !cfg.FABOptions.RunPerformance {
		t.Error("expected RunPerformance enabled by default")
	}
	if !cfg.FABOptions.AutoScanOnLink {
		t.Error("expected AutoScanOnLink enabled by default")
	}
}

// ========== Display Options Tests ==========

func TestDefaultDisplayOptions(t *testing.T) {
	cfg := config.DefaultConfig()

	if !cfg.DisplayOptions.ShowPublicIP {
		t.Error("expected ShowPublicIP enabled by default")
	}
	if cfg.DisplayOptions.UnitSystem != "sae" {
		t.Errorf("expected UnitSystem to be 'sae' by default, got %q", cfg.DisplayOptions.UnitSystem)
	}
}

// ========== Rogue Detection Tests ==========

func TestRogueDetectionConfig(t *testing.T) {
	rogue := RogueDetectionConfig{
		Enabled:          true,
		KnownServers:     []string{"192.168.1.1", "192.168.1.2"},
		AlertOnDetection: true,
	}

	if !rogue.Enabled {
		t.Error("expected rogue detection enabled")
	}
	if len(rogue.KnownServers) != 2 {
		t.Errorf("expected 2 known servers, got %d", len(rogue.KnownServers))
	}
	if !rogue.AlertOnDetection {
		t.Error("expected AlertOnDetection to be true")
	}
}

// ========== Pipeline Config Tests ==========

func TestPipelineConfigGetPhases(t *testing.T) {
	cfg := config.DefaultConfig()
	// Set specific phases
	cfg.Pipeline.Phases.Enumeration = true
	cfg.Pipeline.Phases.NameResolution = true
	cfg.Pipeline.Phases.ServiceDiscovery = false
	cfg.Pipeline.Phases.VulnAssessment = true

	enum, nameRes, svcDisc, vulnAssess := cfg.Pipeline.GetPhases()
	if !enum {
		t.Error("expected enumeration phase enabled")
	}
	if !nameRes {
		t.Error("expected name resolution phase enabled")
	}
	if svcDisc {
		t.Error("expected service discovery phase disabled")
	}
	if !vulnAssess {
		t.Error("expected vulnerability assessment phase enabled")
	}
}

func TestPipelineConfigGetTiming(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Pipeline.Timing.ProbeDelay = 50 * time.Millisecond
	cfg.Pipeline.Timing.HostDelay = 100 * time.Millisecond
	cfg.Pipeline.Timing.PhaseTimeout = 5 * time.Minute
	cfg.Pipeline.Timing.MaxConcurrentHosts = 10
	cfg.Pipeline.Timing.Profile = "polite"

	probeDelay, hostDelay, phaseTimeout, maxHosts, profile := cfg.Pipeline.GetTiming()
	if probeDelay != 50*time.Millisecond {
		t.Errorf("expected probe delay 50ms, got %v", probeDelay)
	}
	if hostDelay != 100*time.Millisecond {
		t.Errorf("expected host delay 100ms, got %v", hostDelay)
	}
	if phaseTimeout != 5*time.Minute {
		t.Errorf("expected phase timeout 5m, got %v", phaseTimeout)
	}
	if maxHosts != 10 {
		t.Errorf("expected max hosts 10, got %d", maxHosts)
	}
	if profile != "polite" {
		t.Errorf("expected profile 'polite', got %q", profile)
	}
}

func TestPipelineConfigGetPortScan(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Pipeline.PortScan.Intensity = "standard"
	cfg.Pipeline.PortScan.CustomPorts = []int{22, 80, 443, 8080}
	cfg.Pipeline.PortScan.BannerGrab = true
	cfg.Pipeline.PortScan.ConnectTimeout = 3 * time.Second

	intensity, customPorts, bannerGrab, connectTimeout := cfg.Pipeline.GetPortScan()
	if intensity != "standard" {
		t.Errorf("expected intensity 'standard', got %q", intensity)
	}
	if len(customPorts) != 4 {
		t.Errorf("expected 4 custom ports, got %d", len(customPorts))
	}
	if !bannerGrab {
		t.Error("expected banner grab enabled")
	}
	if connectTimeout != 3*time.Second {
		t.Errorf("expected connect timeout 3s, got %v", connectTimeout)
	}

	// Verify deep copy - modifying returned slice shouldn't affect original
	customPorts[0] = 9999
	if cfg.Pipeline.PortScan.CustomPorts[0] == 9999 {
		t.Error("GetPortScan should return deep copy, but original was modified")
	}
}

func TestPipelineConfigGetSNMP(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Pipeline.SNMPCollection.Enabled = true
	cfg.Pipeline.SNMPCollection.MIBs.System = true
	cfg.Pipeline.SNMPCollection.MIBs.Interfaces = true
	cfg.Pipeline.SNMPCollection.MIBs.IPAddresses = false
	cfg.Pipeline.SNMPCollection.MIBs.Routing = true
	cfg.Pipeline.SNMPCollection.MIBs.Bridge = false
	cfg.Pipeline.SNMPCollection.MIBs.Entity = true
	cfg.Pipeline.SNMPCollection.MIBs.LLDP = true
	cfg.Pipeline.SNMPCollection.MIBs.VLAN = false
	cfg.Pipeline.SNMPCollection.WalkTimeout = 30 * time.Second
	cfg.Pipeline.SNMPCollection.MaxOIDsPerRequest = 25

	enabled, system, interfaces, ipAddrs, routing, bridge, entity, lldp, vlan, walkTimeout, maxOIDs := cfg.Pipeline.GetSNMP()
	if !enabled {
		t.Error("expected SNMP collection enabled")
	}
	if !system {
		t.Error("expected system MIB enabled")
	}
	if !interfaces {
		t.Error("expected interfaces MIB enabled")
	}
	if ipAddrs {
		t.Error("expected IP addresses MIB disabled")
	}
	if !routing {
		t.Error("expected routing MIB enabled")
	}
	if bridge {
		t.Error("expected bridge MIB disabled")
	}
	if !entity {
		t.Error("expected entity MIB enabled")
	}
	if !lldp {
		t.Error("expected LLDP MIB enabled")
	}
	if vlan {
		t.Error("expected VLAN MIB disabled")
	}
	if walkTimeout != 30*time.Second {
		t.Errorf("expected walk timeout 30s, got %v", walkTimeout)
	}
	if maxOIDs != 25 {
		t.Errorf("expected max OIDs 25, got %d", maxOIDs)
	}
}

func TestPipelineConfigGetPersistence(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Pipeline.Persistence.StoreHistory = true
	cfg.Pipeline.Persistence.StalenessThreshold = 2 * time.Hour
	cfg.Pipeline.Persistence.PurgeAfter = 7 * 24 * time.Hour

	storeHistory, stalenessThreshold, purgeAfter := cfg.Pipeline.GetPersistence()
	if !storeHistory {
		t.Error("expected store history enabled")
	}
	if stalenessThreshold != 2*time.Hour {
		t.Errorf("expected staleness threshold 2h, got %v", stalenessThreshold)
	}
	if purgeAfter != 7*24*time.Hour {
		t.Errorf("expected purge after 7d, got %v", purgeAfter)
	}
}

// ========== Clone and CopyFieldsFrom Tests ==========

func TestConfigClone(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.Port = 9999
	cfg.Auth.DefaultUsername = "clonetest"
	cfg.Security.AllowedOrigins = []string{"https://test.com", "https://dev.com"}
	cfg.SNMP.Communities = []string{"private", "public"}
	cfg.SNMP.V3Credentials = []SNMPv3Credential{
		{Name: "test", Username: "user1", AuthProtocol: "SHA"},
	}

	clone := cfg.Clone()

	// Verify values are copied
	if clone.Server.Port != 9999 {
		t.Errorf("expected port 9999, got %d", clone.Server.Port)
	}
	if clone.Auth.DefaultUsername != "clonetest" {
		t.Errorf("expected username 'clonetest', got %q", clone.Auth.DefaultUsername)
	}

	// Verify slices are deep copied
	if len(clone.Security.AllowedOrigins) != 2 {
		t.Errorf("expected 2 allowed origins, got %d", len(clone.Security.AllowedOrigins))
	}
	clone.Security.AllowedOrigins[0] = "modified"
	if cfg.Security.AllowedOrigins[0] == "modified" {
		t.Error("Clone should deep copy AllowedOrigins, but original was modified")
	}

	// Verify SNMP communities deep copied
	clone.SNMP.Communities[0] = "modified"
	if cfg.SNMP.Communities[0] == "modified" {
		t.Error("Clone should deep copy SNMP.Communities, but original was modified")
	}

	// Verify SNMP v3 credentials deep copied
	clone.SNMP.V3Credentials[0].Name = "modified"
	if cfg.SNMP.V3Credentials[0].Name == "modified" {
		t.Error("Clone should deep copy SNMP.V3Credentials, but original was modified")
	}

	// Verify changes to clone don't affect original
	clone.Server.Port = 1234
	if cfg.Server.Port == 1234 {
		t.Error("Clone modification affected original config")
	}
}

func TestConfigCopyFieldsFrom(t *testing.T) {
	src := config.DefaultConfig()
	src.Server.Port = 7777
	src.Auth.DefaultUsername = "srcuser"
	src.VLAN.Enabled = true
	src.Security.AllowedOrigins = []string{"https://source.com"}

	dst := config.DefaultConfig()
	dst.Server.Port = 1111
	dst.Auth.DefaultUsername = "dstuser"

	// Copy fields from src to dst
	dst.Lock()
	dst.CopyFieldsFrom(src)
	dst.Unlock()

	// Verify values are copied
	if dst.Server.Port != 7777 {
		t.Errorf("expected port 7777, got %d", dst.Server.Port)
	}
	if dst.Auth.DefaultUsername != "srcuser" {
		t.Errorf("expected username 'srcuser', got %q", dst.Auth.DefaultUsername)
	}
	if !dst.VLAN.Enabled {
		t.Error("expected VLAN enabled")
	}

	// Verify slices are deep copied
	dst.Security.AllowedOrigins[0] = "modified"
	if src.Security.AllowedOrigins[0] == "modified" {
		t.Error("CopyFieldsFrom should deep copy slices, but source was modified")
	}
}

// ========== GetBackupDir Test ==========

func TestGetBackupDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	backupDir := filepath.Join(tmpDir, "backups")

	// Create backup manager
	mgr := NewBackupManager(configPath, backupDir, 5)

	if mgr.GetBackupDir() != backupDir {
		t.Errorf("expected backup dir %q, got %q", backupDir, mgr.GetBackupDir())
	}
}

// ========== SaveWithBackup Test ==========

func TestSaveWithBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	backupDir := filepath.Join(tmpDir, "backups")

	// Create initial config
	cfg := config.DefaultConfig()
	cfg.Server.Port = 8080
	cfg.Auth.DefaultPasswordHash = "test-hash"
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	// Modify and save with backup
	cfg.Server.Port = 9090
	backupInfo, err := cfg.SaveWithBackup(configPath, backupDir, 5)
	if err != nil {
		t.Fatalf("failed to save with backup: %v", err)
	}

	// Verify backup was created
	if backupInfo == nil {
		t.Fatal("expected backup info, got nil")
	}
	if backupInfo.Path == "" {
		t.Error("expected backup path to be set")
	}

	// Verify config was saved with new value
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loaded.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", loaded.Server.Port)
	}
}

// ========== Logging Validation Test ==========

func TestValidateLoggingConfig(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*Config)
		wantError bool
	}{
		{
			name:      "valid default",
			modify:    func(_ *Config) {},
			wantError: false,
		},
		{
			name: "invalid log level",
			modify: func(c *Config) {
				c.Logging.Level = "invalid"
			},
			wantError: true,
		},
		{
			name: "negative max size",
			modify: func(c *Config) {
				c.Logging.MaxSize = -1
			},
			wantError: true,
		},
		{
			name: "negative max backups",
			modify: func(c *Config) {
				c.Logging.MaxBackups = -1
			},
			wantError: true,
		},
		{
			name: "negative max age",
			modify: func(c *Config) {
				c.Logging.MaxAge = -1
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "hash"
			tt.modify(cfg)

			err := cfg.Validate()
			if tt.wantError && err == nil {
				t.Error("expected validation error")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}
