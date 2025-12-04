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
	if err := os.WriteFile(configPath, []byte("invalid: [yaml: content"), 0600); err != nil {
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
