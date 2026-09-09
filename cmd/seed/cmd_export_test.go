package main

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "redacts auth secrets",
			cfg: &config.Config{
				Auth: config.AuthConfig{
					DefaultUsername:     "admin",
					DefaultPasswordHash: "secret-hash-value",
					JWTSecret:           "super-secret-jwt",
				},
			},
		},
		{
			name: "redacts vulnerability scanning API key",
			cfg: &config.Config{
				Security: config.SecurityConfig{
					VulnerabilityScanning: config.VulnerabilityScanConfig{
						NVDAPIKey: "nvd-api-key-12345",
					},
				},
			},
		},
		{
			name: "handles empty config",
			cfg:  &config.Config{},
		},
		{
			name: "preserves non-sensitive fields",
			cfg: &config.Config{
				Version: 1,
				Server: config.ServerConfig{
					Port: 8443,
				},
				Interface: config.InterfaceConfig{
					Default: "eth0",
				},
				Auth: config.AuthConfig{
					DefaultUsername:     "admin",
					DefaultPasswordHash: "hash",
					JWTSecret:           "secret",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			redacted := redactSecrets(tc.cfg)
			assertRedactedSecrets(t, tc.cfg, redacted)
		})
	}
}

func TestRedactSecretsPreservesNonSensitiveData(t *testing.T) {
	original := &config.Config{
		Version: 2,
		Server: config.ServerConfig{
			Port: 8443,
		},
		Interface: config.InterfaceConfig{
			Default:   "eth0",
			Fallbacks: []string{"eth1", "wlan0"},
		},
		Auth: config.AuthConfig{
			DefaultUsername:     "admin",
			DefaultPasswordHash: "secret-hash",
			JWTSecret:           "secret-jwt",
		},
	}

	redacted := redactSecrets(original)

	assertPreservedNonSensitiveData(t, original, redacted)
}

func assertRedactedSecrets(t *testing.T, original, redacted *config.Config) {
	t.Helper()
	assertAuthRedacted(t, original, redacted)
	assertSecurityRedacted(t, original, redacted)
}

func assertAuthRedacted(t *testing.T, original, redacted *config.Config) {
	t.Helper()
	if original.Auth.DefaultPasswordHash != "" && redacted.Auth.DefaultPasswordHash != redactedValue {
		t.Errorf("DefaultPasswordHash should be redacted, got %q", redacted.Auth.DefaultPasswordHash)
	}
	if original.Auth.JWTSecret != "" && redacted.Auth.JWTSecret != redactedValue {
		t.Errorf("JWTSecret should be redacted, got %q", redacted.Auth.JWTSecret)
	}
}

func assertSecurityRedacted(t *testing.T, original, redacted *config.Config) {
	t.Helper()
	if original.Security.VulnerabilityScanning.NVDAPIKey != "" &&
		redacted.Security.VulnerabilityScanning.NVDAPIKey != redactedValue {
		t.Errorf(
			"NVDAPIKey should be redacted, got %q",
			redacted.Security.VulnerabilityScanning.NVDAPIKey,
		)
	}
}

func assertPreservedNonSensitiveData(t *testing.T, original, redacted *config.Config) {
	t.Helper()
	if redacted.Version != original.Version {
		t.Errorf("Version should be preserved: got %d, want %d", redacted.Version, original.Version)
	}
	if redacted.Server.Port != original.Server.Port {
		t.Errorf("Server.Port should be preserved: got %d, want %d", redacted.Server.Port, original.Server.Port)
	}
	if redacted.Interface.Default != original.Interface.Default {
		t.Errorf(
			"Interface.Default should be preserved: got %q, want %q",
			redacted.Interface.Default,
			original.Interface.Default,
		)
	}
	if redacted.Auth.DefaultUsername != original.Auth.DefaultUsername {
		t.Errorf(
			"Auth.DefaultUsername should be preserved: got %q, want %q",
			redacted.Auth.DefaultUsername,
			original.Auth.DefaultUsername,
		)
	}
}

func TestRedactSecretsDoesNotModifyOriginal(t *testing.T) {
	original := &config.Config{
		Auth: config.AuthConfig{
			DefaultPasswordHash: "original-hash",
			JWTSecret:           "original-jwt",
		},
		Security: config.SecurityConfig{
			VulnerabilityScanning: config.VulnerabilityScanConfig{
				NVDAPIKey: "original-api-key",
			},
		},
	}

	// Store original values
	originalHash := original.Auth.DefaultPasswordHash
	originalJWT := original.Auth.JWTSecret
	originalAPIKey := original.Security.VulnerabilityScanning.NVDAPIKey

	// Call redactSecrets
	_ = redactSecrets(original)

	// Verify original is not modified
	if original.Auth.DefaultPasswordHash != originalHash {
		t.Errorf(
			"Original DefaultPasswordHash was modified: got %q, want %q",
			original.Auth.DefaultPasswordHash,
			originalHash,
		)
	}
	if original.Auth.JWTSecret != originalJWT {
		t.Errorf("Original JWTSecret was modified: got %q, want %q", original.Auth.JWTSecret, originalJWT)
	}
	if original.Security.VulnerabilityScanning.NVDAPIKey != originalAPIKey {
		t.Errorf(
			"Original NVDAPIKey was modified: got %q, want %q",
			original.Security.VulnerabilityScanning.NVDAPIKey,
			originalAPIKey,
		)
	}
}

func TestRedactedValueConstant(t *testing.T) {
	if redactedValue != "[REDACTED]" {
		t.Errorf("redactedValue should be '[REDACTED]', got %q", redactedValue)
	}
}
