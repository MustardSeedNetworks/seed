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

	// Test interface defaults
	if cfg.Interface.Default != "eth0" {
		t.Errorf("expected default interface eth0, got %s", cfg.Interface.Default)
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
