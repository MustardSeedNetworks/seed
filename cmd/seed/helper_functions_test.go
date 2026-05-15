package main

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/paths"
)

func TestGeneratePasswordAndHashValid(t *testing.T) {

	password, hash, err := generatePasswordAndHash()
	if err != nil {
		t.Fatalf("generatePasswordAndHash failed: %v", err)
	}

	if password == "" {
		t.Error("Generated password should not be empty")
	}

	if hash == "" {
		t.Error("Generated hash should not be empty")
	}

	// Password should be at least defaultPasswordLength
	if len(password) < GetDefaultPasswordLength() {
		t.Errorf(
			"Password length should be at least %d, got %d",
			GetDefaultPasswordLength(),
			len(password),
		)
	}

	// Hash should be different from password
	if password == hash {
		t.Error("Hash should be different from password")
	}
}

func TestGeneratePasswordAndHashVerifiable(t *testing.T) {

	password, hash, err := generatePasswordAndHash()
	if err != nil {
		t.Fatalf("generatePasswordAndHash failed: %v", err)
	}

	// Verify the password matches the hash using bcrypt
	if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); compareErr != nil {
		t.Error("Password should verify against its hash")
	}

	// Wrong password should not match
	if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrongpassword")); compareErr == nil {
		t.Error("Wrong password should not verify against hash")
	}
}

func TestEnsureConfigDirCreatesNestedDirectories(t *testing.T) {

	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "a", "b", "c", "config.json")

	err := ensureConfigDir(nestedPath)
	if err != nil {
		t.Fatalf("ensureConfigDir failed: %v", err)
	}

	// Verify parent directory was created
	parentDir := filepath.Dir(nestedPath)
	if _, statErr := os.Stat(parentDir); statErr != nil {
		t.Errorf("Parent directory should be created: %v", statErr)
	}
}

func TestEnsureConfigDirWithExistingParent(t *testing.T) {

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Should not error for existing directory
	err := ensureConfigDir(configPath)
	if err != nil {
		t.Errorf("ensureConfigDir should not error for existing directory: %v", err)
	}
}

func TestEnsureConfigDirCurrentDirectory(t *testing.T) {

	// Test with just a filename (current directory)
	err := ensureConfigDir("config.json")
	if err != nil {
		t.Errorf("ensureConfigDir should not error for current directory: %v", err)
	}
}

func TestEnsureConfigDirDotPath(t *testing.T) {

	err := ensureConfigDir("./config.json")
	if err != nil {
		t.Errorf("ensureConfigDir should not error for dot path: %v", err)
	}
}

func TestPreserveExistingCredentialsWithAllFlags(t *testing.T) {

	newCfg := &config.Config{
		Auth: config.AuthConfig{
			DefaultUsername:     "new-user",
			DefaultPasswordHash: "new-hash",
			JWTSecret:           "new-jwt",
		},
	}

	existingCfg := &config.Config{
		Auth: config.AuthConfig{
			DefaultUsername:     "old-user",
			DefaultPasswordHash: "old-hash",
			JWTSecret:           "old-jwt",
		},
	}

	flags := resetFlags{
		preserveAuth: true,
		preserveJWT:  true,
	}

	preserveExistingCredentials(newCfg, existingCfg, flags)

	// Both should be preserved
	if newCfg.Auth.DefaultUsername != "old-user" {
		t.Errorf("Username should be preserved: got %q, want %q", newCfg.Auth.DefaultUsername, "old-user")
	}
	if newCfg.Auth.DefaultPasswordHash != "old-hash" {
		t.Errorf("Hash should be preserved: got %q, want %q", newCfg.Auth.DefaultPasswordHash, "old-hash")
	}
	if newCfg.Auth.JWTSecret != "old-jwt" {
		t.Errorf("JWT should be preserved: got %q, want %q", newCfg.Auth.JWTSecret, "old-jwt")
	}
}

func TestPreserveExistingCredentialsNoPreservation(t *testing.T) {

	newCfg := &config.Config{
		Auth: config.AuthConfig{
			DefaultUsername:     "new-user",
			DefaultPasswordHash: "new-hash",
			JWTSecret:           "new-jwt",
		},
	}

	existingCfg := &config.Config{
		Auth: config.AuthConfig{
			DefaultUsername:     "old-user",
			DefaultPasswordHash: "old-hash",
			JWTSecret:           "old-jwt",
		},
	}

	flags := resetFlags{
		preserveAuth: false,
		preserveJWT:  false,
	}

	preserveExistingCredentials(newCfg, existingCfg, flags)

	// Nothing should be preserved
	if newCfg.Auth.DefaultUsername != "new-user" {
		t.Errorf("Username should not be preserved: got %q, want %q", newCfg.Auth.DefaultUsername, "new-user")
	}
	if newCfg.Auth.DefaultPasswordHash != "new-hash" {
		t.Errorf("Hash should not be preserved: got %q, want %q", newCfg.Auth.DefaultPasswordHash, "new-hash")
	}
	if newCfg.Auth.JWTSecret != "new-jwt" {
		t.Errorf("JWT should not be preserved: got %q, want %q", newCfg.Auth.JWTSecret, "new-jwt")
	}
}

func TestCheckConfigWarningsAllMissing(t *testing.T) {

	cfg := &config.Config{
		Interface: config.InterfaceConfig{
			Default: "",
		},
		Auth: config.AuthConfig{
			JWTSecret: "",
		},
		SNMP: config.SNMPConfig{
			Communities: nil,
		},
	}

	warnings := checkConfigWarnings(cfg)

	// Should have 3 warnings
	if len(warnings) != 3 {
		t.Errorf("Expected 3 warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestCheckConfigWarningsNoneMissing(t *testing.T) {

	cfg := &config.Config{
		Interface: config.InterfaceConfig{
			Default: "eth0",
		},
		Auth: config.AuthConfig{
			JWTSecret: "secret",
		},
		SNMP: config.SNMPConfig{
			Communities: []string{"public"},
		},
	}

	warnings := checkConfigWarnings(cfg)

	// Should have 0 warnings
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestRedactSecretsComprehensive(t *testing.T) {

	cfg := &config.Config{
		Version: 1,
		Server: config.ServerConfig{
			Port:  8443,
			HTTPS: true,
		},
		Auth: config.AuthConfig{
			DefaultUsername:     "admin",
			DefaultPasswordHash: "secret-hash",
			JWTSecret:           "secret-jwt",
		},
		Security: config.SecurityConfig{
			VulnerabilityScanning: config.VulnerabilityScanConfig{
				NVDAPIKey: "api-key-12345",
			},
		},
		SNMP: config.SNMPConfig{
			V3Credentials: []config.SNMPv3Credential{
				{
					Username:     "snmpuser",
					AuthPassword: "auth-pass",
					PrivPassword: "priv-pass",
				},
			},
		},
	}

	redacted := redactSecrets(cfg)

	// Check auth secrets are redacted
	if redacted.Auth.DefaultPasswordHash != GetRedactedValue() {
		t.Errorf("Password hash should be redacted: got %q", redacted.Auth.DefaultPasswordHash)
	}
	if redacted.Auth.JWTSecret != GetRedactedValue() {
		t.Errorf("JWT secret should be redacted: got %q", redacted.Auth.JWTSecret)
	}

	// Check security secrets are redacted
	if redacted.Security.VulnerabilityScanning.NVDAPIKey != GetRedactedValue() {
		t.Errorf("NVD API key should be redacted: got %q", redacted.Security.VulnerabilityScanning.NVDAPIKey)
	}

	// Check SNMP secrets are redacted
	if len(redacted.SNMP.V3Credentials) > 0 {
		if redacted.SNMP.V3Credentials[0].AuthPassword != GetRedactedValue() {
			t.Errorf("SNMP auth password should be redacted: got %q", redacted.SNMP.V3Credentials[0].AuthPassword)
		}
		if redacted.SNMP.V3Credentials[0].PrivPassword != GetRedactedValue() {
			t.Errorf("SNMP priv password should be redacted: got %q", redacted.SNMP.V3Credentials[0].PrivPassword)
		}
	}

	// Check non-sensitive fields are preserved
	if redacted.Version != 1 {
		t.Errorf("Version should be preserved: got %d", redacted.Version)
	}
	if redacted.Server.Port != 8443 {
		t.Errorf("Port should be preserved: got %d", redacted.Server.Port)
	}
	if redacted.Auth.DefaultUsername != "admin" {
		t.Errorf("Username should be preserved: got %q", redacted.Auth.DefaultUsername)
	}

	// Check original is not modified
	if cfg.Auth.DefaultPasswordHash == GetRedactedValue() {
		t.Error("Original config should not be modified")
	}
}

func TestDistroParsingEdgeCases(t *testing.T) {

	tests := []struct {
		name       string
		content    string
		wantID     string
		wantFamily string
	}{
		{
			name:       "empty content",
			content:    "",
			wantID:     "",
			wantFamily: "",
		},
		{
			name:       "only whitespace",
			content:    "   \n\t\n  ",
			wantID:     "",
			wantFamily: "",
		},
		{
			name:       "no ID_LIKE uses ID for family",
			content:    "ID=customdistro\nNAME=\"Custom Distro\"",
			wantID:     "customdistro",
			wantFamily: "customdistro",
		},
		{
			name:       "has ID_LIKE",
			content:    "ID=mint\nID_LIKE=ubuntu\nNAME=\"Linux Mint\"",
			wantID:     "mint",
			wantFamily: "ubuntu",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			distro := parseOSRelease(tc.content)

			if distro.ID != tc.wantID {
				t.Errorf("ID: got %q, want %q", distro.ID, tc.wantID)
			}
			if distro.Family != tc.wantFamily {
				t.Errorf("Family: got %q, want %q", distro.Family, tc.wantFamily)
			}
		})
	}
}

func TestModeStringAllValues(t *testing.T) {

	// paths.Mode constant values:
	// ModeAuto = 0, ModeUser = 1, ModeSystem = 2
	tests := []struct {
		name     string
		input    paths.Mode
		expected string
	}{
		{"auto", paths.ModeAuto, "auto"},
		{"user", paths.ModeUser, "user"},
		{"system", paths.ModeSystem, "system"},
		{"unknown", paths.Mode(999), "unknown"},
		{"negative", paths.Mode(-1), "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			result := modeString(tc.input)

			if result != tc.expected {
				t.Errorf("modeString(%d): got %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
