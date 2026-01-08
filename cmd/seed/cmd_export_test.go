package main

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
)

func TestRedactSecrets(t *testing.T) {
	t.Parallel()

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
			name: "redacts SNMP v3 credentials",
			cfg: &config.Config{
				SNMP: config.SNMPConfig{
					V3Credentials: []config.SNMPv3Credential{
						{
							Username:     "snmpuser1",
							AuthPassword: "auth-pass-1",
							PrivPassword: "priv-pass-1",
						},
						{
							Username:     "snmpuser2",
							AuthPassword: "auth-pass-2",
							PrivPassword: "priv-pass-2",
						},
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
					Port:  8443,
					HTTPS: true,
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
			t.Parallel()

			redacted := redactSecrets(tc.cfg)

			// Verify auth secrets are redacted
			if tc.cfg.Auth.DefaultPasswordHash != "" && redacted.Auth.DefaultPasswordHash != redactedValue {
				t.Errorf(
					"DefaultPasswordHash should be redacted, got %q",
					redacted.Auth.DefaultPasswordHash,
				)
			}
			if tc.cfg.Auth.JWTSecret != "" && redacted.Auth.JWTSecret != redactedValue {
				t.Errorf("JWTSecret should be redacted, got %q", redacted.Auth.JWTSecret)
			}

			// Verify NVD API key is redacted
			if tc.cfg.Security.VulnerabilityScanning.NVDAPIKey != "" &&
				redacted.Security.VulnerabilityScanning.NVDAPIKey != redactedValue {
				t.Errorf(
					"NVDAPIKey should be redacted, got %q",
					redacted.Security.VulnerabilityScanning.NVDAPIKey,
				)
			}

			// Verify SNMP credentials are redacted
			for i, cred := range tc.cfg.SNMP.V3Credentials {
				if cred.AuthPassword != "" && redacted.SNMP.V3Credentials[i].AuthPassword != redactedValue {
					t.Errorf(
						"SNMP AuthPassword[%d] should be redacted, got %q",
						i,
						redacted.SNMP.V3Credentials[i].AuthPassword,
					)
				}
				if cred.PrivPassword != "" && redacted.SNMP.V3Credentials[i].PrivPassword != redactedValue {
					t.Errorf(
						"SNMP PrivPassword[%d] should be redacted, got %q",
						i,
						redacted.SNMP.V3Credentials[i].PrivPassword,
					)
				}
			}
		})
	}
}

func TestRedactSecretsPreservesNonSensitiveData(t *testing.T) {
	t.Parallel()

	original := &config.Config{
		Version: 2,
		Server: config.ServerConfig{
			Port:  8443,
			HTTPS: true,
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
		SNMP: config.SNMPConfig{
			Communities: []string{"public", "private"},
			V3Credentials: []config.SNMPv3Credential{
				{
					Username:     "snmpuser",
					AuthPassword: "authpass",
					PrivPassword: "privpass",
				},
			},
		},
	}

	redacted := redactSecrets(original)

	// Non-sensitive fields should be preserved
	if redacted.Version != original.Version {
		t.Errorf("Version should be preserved: got %d, want %d", redacted.Version, original.Version)
	}
	if redacted.Server.Port != original.Server.Port {
		t.Errorf("Server.Port should be preserved: got %d, want %d", redacted.Server.Port, original.Server.Port)
	}
	if redacted.Server.HTTPS != original.Server.HTTPS {
		t.Errorf("Server.HTTPS should be preserved: got %v, want %v", redacted.Server.HTTPS, original.Server.HTTPS)
	}
	if redacted.Interface.Default != original.Interface.Default {
		t.Errorf(
			"Interface.Default should be preserved: got %q, want %q",
			redacted.Interface.Default,
			original.Interface.Default,
		)
	}

	// Username should be preserved (not a secret)
	if redacted.Auth.DefaultUsername != original.Auth.DefaultUsername {
		t.Errorf(
			"Auth.DefaultUsername should be preserved: got %q, want %q",
			redacted.Auth.DefaultUsername,
			original.Auth.DefaultUsername,
		)
	}

	// SNMP username should be preserved
	if len(redacted.SNMP.V3Credentials) > 0 {
		if redacted.SNMP.V3Credentials[0].Username != original.SNMP.V3Credentials[0].Username {
			t.Errorf(
				"SNMP Username should be preserved: got %q, want %q",
				redacted.SNMP.V3Credentials[0].Username,
				original.SNMP.V3Credentials[0].Username,
			)
		}
	}

	// SNMP communities should be preserved
	if len(redacted.SNMP.Communities) != len(original.SNMP.Communities) {
		t.Errorf(
			"SNMP Communities should be preserved: got %d, want %d",
			len(redacted.SNMP.Communities),
			len(original.SNMP.Communities),
		)
	}
}

func TestRedactSecretsDoesNotModifyOriginal(t *testing.T) {
	t.Parallel()

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
		SNMP: config.SNMPConfig{
			V3Credentials: []config.SNMPv3Credential{
				{
					AuthPassword: "original-auth",
					PrivPassword: "original-priv",
				},
			},
		},
	}

	// Store original values
	originalHash := original.Auth.DefaultPasswordHash
	originalJWT := original.Auth.JWTSecret
	originalAPIKey := original.Security.VulnerabilityScanning.NVDAPIKey
	originalAuthPass := original.SNMP.V3Credentials[0].AuthPassword
	originalPrivPass := original.SNMP.V3Credentials[0].PrivPassword

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
	if original.SNMP.V3Credentials[0].AuthPassword != originalAuthPass {
		t.Errorf(
			"Original SNMP AuthPassword was modified: got %q, want %q",
			original.SNMP.V3Credentials[0].AuthPassword,
			originalAuthPass,
		)
	}
	if original.SNMP.V3Credentials[0].PrivPassword != originalPrivPass {
		t.Errorf(
			"Original SNMP PrivPassword was modified: got %q, want %q",
			original.SNMP.V3Credentials[0].PrivPassword,
			originalPrivPass,
		)
	}
}

func TestRedactedValueConstant(t *testing.T) {
	t.Parallel()

	if redactedValue != "[REDACTED]" {
		t.Errorf("redactedValue should be '[REDACTED]', got %q", redactedValue)
	}
}

func TestRedactSecretsWithEmptySNMPCredentials(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Auth: config.AuthConfig{
			DefaultPasswordHash: "hash",
			JWTSecret:           "jwt",
		},
		SNMP: config.SNMPConfig{
			V3Credentials: []config.SNMPv3Credential{},
		},
	}

	redacted := redactSecrets(cfg)

	// Should handle empty credentials slice without panic
	if len(redacted.SNMP.V3Credentials) != 0 {
		t.Errorf("Expected empty V3Credentials slice, got %d items", len(redacted.SNMP.V3Credentials))
	}
}

func TestRedactSecretsWithMultipleSNMPCredentials(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		SNMP: config.SNMPConfig{
			V3Credentials: []config.SNMPv3Credential{
				{Username: "user1", AuthPassword: "auth1", PrivPassword: "priv1"},
				{Username: "user2", AuthPassword: "auth2", PrivPassword: "priv2"},
				{Username: "user3", AuthPassword: "auth3", PrivPassword: "priv3"},
			},
		},
	}

	redacted := redactSecrets(cfg)

	if len(redacted.SNMP.V3Credentials) != 3 {
		t.Fatalf("Expected 3 credentials, got %d", len(redacted.SNMP.V3Credentials))
	}

	for i, cred := range redacted.SNMP.V3Credentials {
		expectedUsername := cfg.SNMP.V3Credentials[i].Username
		if cred.Username != expectedUsername {
			t.Errorf("Credential %d: username should be preserved: got %q, want %q", i, cred.Username, expectedUsername)
		}
		if cred.AuthPassword != redactedValue {
			t.Errorf("Credential %d: AuthPassword should be redacted: got %q", i, cred.AuthPassword)
		}
		if cred.PrivPassword != redactedValue {
			t.Errorf("Credential %d: PrivPassword should be redacted: got %q", i, cred.PrivPassword)
		}
	}
}
