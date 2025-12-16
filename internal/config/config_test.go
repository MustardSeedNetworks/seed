package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
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
	cfg, err := Load("/nonexistent/path/config.yaml")
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
	cfg := DefaultConfig()
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
	loaded, err := Load(configPath)
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

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestThresholdDefaults(t *testing.T) {
	cfg := DefaultConfig()

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
	cfg, result, err := EnsureConfig(configPath, nil)

	// Should return ErrInsecureCredentials because password hash is empty
	if err != ErrInsecureCredentials {
		t.Errorf("expected ErrInsecureCredentials for first boot, got %v", err)
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
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "$2a$10$InsecureHashThatWillBeDetected"
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	// checkDefaultPassword function that always returns true (simulates insecure)
	checkInsecure := func(_ string) bool { return true }

	_, result, err := EnsureConfig(configPath, checkInsecure)

	if err != ErrInsecureCredentials {
		t.Errorf("expected ErrInsecureCredentials for insecure password, got %v", err)
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
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "$2a$10$SecureHashThatIsNotDefault"
	cfg.Auth.JWTSecret = "existing-secure-secret"
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	// checkDefaultPassword function that returns false (not insecure)
	checkSecure := func(_ string) bool { return false }

	loadedCfg, result, err := EnsureConfig(configPath, checkSecure)

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
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "$2a$10$SecureHashThatIsNotDefault"
	cfg.Auth.JWTSecret = "" // Empty - needs generation
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	checkSecure := func(_ string) bool { return false }

	_, result, err := EnsureConfig(configPath, checkSecure)

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
	cfg := DefaultConfig()

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
	cfg := DefaultConfig()
	cfg.Auth.JWTSecret = "originalsecret"

	// Update with empty JWT secret - should preserve original
	cfg.UpdateCredentials("newuser", "newhash", "")

	if cfg.Auth.JWTSecret != "originalsecret" {
		t.Errorf("expected JWT secret preserved, got %q", cfg.Auth.JWTSecret)
	}
}

func TestUpdateJWTSecret(t *testing.T) {
	cfg := DefaultConfig()
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
	cfg := DefaultConfig()
	cfg.UpdateCredentials("persistuser", "persisted-hash", "persisted-jwt-secret")

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load config and verify credentials persisted
	loaded, err := Load(configPath)
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

	cfg := DefaultConfig()
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
	_, _, err := EnsureConfig(configPath, nil)
	// Will return ErrInsecureCredentials because password is empty, but directory should exist
	if err != nil && err != ErrInsecureCredentials {
		t.Fatalf("EnsureConfig failed unexpectedly: %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("EnsureConfig did not create nested directory")
	}
}

// ========== Config Locking Tests ==========

func TestConfigLocking(t *testing.T) {
	cfg := DefaultConfig()

	// Test write lock
	cfg.Lock()
	cfg.Server.Port = 9999
	cfg.Unlock()

	if cfg.Server.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Server.Port)
	}
}

func TestConfigReadLocking(t *testing.T) {
	cfg := DefaultConfig()
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
	cfg := DefaultConfig()
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
			cfg := DefaultConfig()
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
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Server.HTTPRedirectPort = -1

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for negative HTTP redirect port")
	}
}

func TestValidateSamePortAndRedirectPort(t *testing.T) {
	cfg := DefaultConfig()
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
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Interface.Default = ""

	err := cfg.Validate()
	if err != nil {
		t.Errorf("empty interface should be valid (auto-detection), got error: %v", err)
	}
}

func TestValidateNegativeStartupRetries(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Interface.StartupRetries = -1

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for negative startup retries")
	}
}

func TestValidateNegativeStartupRetryWait(t *testing.T) {
	cfg := DefaultConfig()
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
			cfg := DefaultConfig()
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
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.IP.Mode = "invalid"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid IP mode")
	}
}

func TestValidateStaticIPMissingAddress(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.IP.Mode = "static"
	cfg.IP.Static = &StaticIP{
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
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.IP.Mode = "static"
	cfg.IP.Static = &StaticIP{
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
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.IP.Mode = "static"
	cfg.IP.Static = &StaticIP{
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
		modify func(*Config)
	}{
		{"discovery timeout", func(c *Config) { c.Discovery.Timeout = 0 }},
		{"ping timeout", func(c *Config) { c.NetworkDiscovery.PingTimeout = 0 }},
		{"scan timeout", func(c *Config) { c.NetworkDiscovery.ScanTimeout = 0 }},
		{"DNS timeout", func(c *Config) { c.DNS.Timeout = 0 }},
		{"SNMP timeout", func(c *Config) { c.SNMP.Timeout = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
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
			cfg := DefaultConfig()
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
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Auth.SessionTimeout = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for zero session timeout")
	}
}

func TestValidateEmptyUsername(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.DefaultPasswordHash = "hash"
	cfg.Auth.DefaultUsername = ""

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty username")
	}
}

func TestValidateEmptyPasswordHash(t *testing.T) {
	cfg := DefaultConfig()
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
			cfg := DefaultConfig()
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
			cfg := DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "hash"
			cfg.SNMP.Retries = tt.retries

			err := cfg.Validate()
			if err == nil {
				t.Error("expected validation error for invalid SNMP retries")
			}
		})
	}
}

// ========== Discovery Profile Tests ==========

func TestDiscoveryProfileConstants(t *testing.T) {
	if ProfileStealth != "stealth" {
		t.Errorf("ProfileStealth should be 'stealth', got %q", ProfileStealth)
	}
	if ProfileStandard != "standard" {
		t.Errorf("ProfileStandard should be 'standard', got %q", ProfileStandard)
	}
	if ProfileFullScan != "full_scan" {
		t.Errorf("ProfileFullScan should be 'full_scan', got %q", ProfileFullScan)
	}
	if ProfileCustom != "custom" {
		t.Errorf("ProfileCustom should be 'custom', got %q", ProfileCustom)
	}
}

func TestDefaultNetworkDiscoveryProfile(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.NetworkDiscovery.Profile != ProfileStandard {
		t.Errorf("expected default profile 'standard', got %q", cfg.NetworkDiscovery.Profile)
	}
}

// ========== SubnetConfig Tests ==========

func TestSubnetConfig(t *testing.T) {
	subnet := SubnetConfig{
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
	server := DNSServer{
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
	target := PingTarget{
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
	test := TCPPortTest{
		Name:    "SSH",
		Host:    "server.local",
		Port:    22,
		Enabled: true,
	}

	if test.Name != "SSH" {
		t.Errorf("expected name 'SSH', got %q", test.Name)
	}
	if test.Port != 22 {
		t.Errorf("expected port 22, got %d", test.Port)
	}
}

func TestHTTPEndpointConfig(t *testing.T) {
	endpoint := HTTPEndpoint{
		Name:           "API Health",
		URL:            "https://api.example.com/health",
		ExpectedStatus: 200,
		Enabled:        true,
	}

	if endpoint.Name != "API Health" {
		t.Errorf("expected name 'API Health', got %q", endpoint.Name)
	}
	if endpoint.ExpectedStatus != 200 {
		t.Errorf("expected status 200, got %d", endpoint.ExpectedStatus)
	}
}

// ========== SNMP Configuration Tests ==========

func TestSNMPv3CredentialConfig(t *testing.T) {
	cred := SNMPv3Credential{
		Name:          "Admin",
		Username:      "snmpv3admin",
		AuthProtocol:  "SHA",
		AuthPassword:  "authpass123",
		PrivProtocol:  "AES",
		PrivPassword:  "privpass456",
		SecurityLevel: "authPriv",
	}

	if cred.Username != "snmpv3admin" {
		t.Errorf("expected username 'snmpv3admin', got %q", cred.Username)
	}
	if cred.SecurityLevel != "authPriv" {
		t.Errorf("expected security level 'authPriv', got %q", cred.SecurityLevel)
	}
}

func TestDefaultSNMPConfig(t *testing.T) {
	cfg := DefaultConfig()

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
	cfg := DefaultConfig()

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
}

// ========== Vulnerability Scanning Tests ==========

func TestVulnerabilityScanConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Security.VulnerabilityScanning.Enabled {
		t.Error("expected vulnerability scanning disabled by default")
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
	cfg := DefaultConfig()

	if !cfg.FABOptions.RunLink {
		t.Error("expected RunLink enabled by default")
	}
	if !cfg.FABOptions.RunSwitch {
		t.Error("expected RunSwitch enabled by default")
	}
	if cfg.FABOptions.RunSpeedtest {
		t.Error("expected RunSpeedtest disabled by default")
	}
	if !cfg.FABOptions.AutoScanOnLink {
		t.Error("expected AutoScanOnLink enabled by default")
	}
}

// ========== Display Options Tests ==========

func TestDefaultDisplayOptions(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.DisplayOptions.ShowPublicIP {
		t.Error("expected ShowPublicIP enabled by default")
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
}
