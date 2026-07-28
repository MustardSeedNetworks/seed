package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

func TestLoadRejectsRemovedTLSServerSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setting string
	}{
		{name: "HTTPS toggle", setting: `"https": false`},
		{name: "HTTP redirect port", setting: `"http_redirect_port": 8080`},
		{name: "ACME", setting: `"acme": {"enabled": true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "seed.json")
			contents := []byte(`{"server": {` + tt.setting + `}}`)
			if err := os.WriteFile(path, contents, 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}

			_, err := config.Load(path)
			if err == nil {
				t.Fatal("Load() accepted a removed TLS server setting")
			}
			if !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("Load() error = %q, want an unknown-field error", err)
			}
		})
	}
}

func TestLoadRejectsRemovedTLSEnvironment(t *testing.T) {
	tests := []string{
		"SEED_HTTP_PORT",
		"SEED_HTTPS_ENABLED",
		"SEED_HTTP_REDIRECT_PORT",
		"SEED_ACME_ENABLED",
		"SEED_ACME_DOMAIN",
		"SEED_ACME_EMAIL",
		"SEED_ACME_CACHE_DIR",
		"SEED_ACME_STAGING",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, "removed")
			_, err := config.Load(filepath.Join(t.TempDir(), "missing.json"))
			if err == nil {
				t.Fatalf("Load() accepted removed environment variable %s", name)
			}
			if !strings.Contains(err.Error(), name+" is no longer supported") {
				t.Fatalf("Load() error = %q, want removed-variable guidance", err)
			}
		})
	}
}

func TestLoadSystemConfigUsesHTTPSPortEnvironment(t *testing.T) {
	t.Setenv("SEED_HTTPS_PORT", "9443")
	cfg, err := config.LoadSystemConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9443 {
		t.Fatalf("HTTPS port = %d, want 9443", cfg.Server.Port)
	}
}

func TestLoadSystemConfigRejectsRemovedTLSServerSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "system.json")
	if err := os.WriteFile(path, []byte(`{"server":{"https":false}}`), 0o600); err != nil {
		t.Fatalf("write system config: %v", err)
	}

	_, err := config.LoadSystemConfig(path)
	if err == nil {
		t.Fatal("LoadSystemConfig() accepted the removed HTTPS toggle")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("LoadSystemConfig() error = %q, want an unknown-field error", err)
	}
}

func TestHTTPSServerEnvironmentOverrides(t *testing.T) {
	t.Setenv("SEED_HTTPS_PORT", "9443")
	t.Setenv("SEED_PUBLIC_ORIGIN", "https://seed.example.com:8443")
	t.Setenv("SEED_TLS_CERT_FILE", "server.crt")
	t.Setenv("SEED_TLS_KEY_FILE", "server.key")

	legacy, err := config.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Server.Port != 9443 || legacy.Server.PublicOrigin != "https://seed.example.com:8443" ||
		legacy.Server.CertFile != "server.crt" || legacy.Server.KeyFile != "server.key" {
		t.Fatalf("legacy server environment was not applied: %+v", legacy.Server)
	}

	cfg, err := config.LoadSystemConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if validateErr := cfg.Validate(); validateErr != nil {
		t.Fatal(validateErr)
	}
	if got := cfg.ToLegacyConfig().Server.PublicOrigin; got != "https://seed.example.com:8443" {
		t.Fatalf("legacy public origin = %q", got)
	}
}

func TestLoadRejectsInvalidHTTPSPortEnvironment(t *testing.T) {
	t.Setenv("SEED_HTTPS_PORT", "not-a-port")
	if _, err := config.Load(filepath.Join(t.TempDir(), "missing.json")); err == nil ||
		!strings.Contains(err.Error(), "SEED_HTTPS_PORT must be an integer") {
		t.Fatalf("Load() error = %v", err)
	}
	if _, err := config.LoadSystemConfig(filepath.Join(t.TempDir(), "missing.json")); err == nil ||
		!strings.Contains(err.Error(), "SEED_HTTPS_PORT must be an integer") {
		t.Fatalf("LoadSystemConfig() error = %v", err)
	}
}

func TestValidateRequiresCompleteOperatorCertificatePair(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		certFile string
		keyFile  string
		wantErr  bool
	}{
		{name: "self-signed fallback"},
		{name: "operator certificate", certFile: "server.crt", keyFile: "server.key"},
		{name: "certificate without key", certFile: "server.crt", wantErr: true},
		{name: "key without certificate", keyFile: "server.key", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "test-hash"
			cfg.Server.CertFile = tt.certFile
			cfg.Server.KeyFile = tt.keyFile

			err := cfg.Validate()
			if tt.wantErr && (err == nil || !strings.Contains(err.Error(), "must be configured together")) {
				t.Fatalf("Validate() error = %v, want incomplete certificate-pair error", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestValidatePublicOrigin(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		origin  string
		wantErr bool
	}{
		{name: "local fallback"},
		{name: "DNS origin", origin: "https://seed.example.com"},
		{name: "origin with port", origin: "https://seed.example.com:8443"},
		{name: "plaintext", origin: "http://seed.example.com", wantErr: true},
		{name: "path", origin: "https://seed.example.com/admin", wantErr: true},
		{name: "credentials", origin: "https://operator@seed.example.com", wantErr: true},
		{name: "query", origin: "https://seed.example.com?mode=admin", wantErr: true},
		{name: "empty query", origin: "https://seed.example.com?", wantErr: true},
		{name: "empty fragment", origin: "https://seed.example.com#", wantErr: true},
		{name: "missing host", origin: "https://", wantErr: true},
		{name: "empty port", origin: "https://seed.example.com:", wantErr: true},
		{name: "zero port", origin: "https://seed.example.com:0", wantErr: true},
		{name: "port above range", origin: "https://seed.example.com:70000", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "test-hash"
			cfg.Server.PublicOrigin = tt.origin
			err := cfg.Validate()
			if tt.wantErr && (err == nil || !strings.Contains(err.Error(), "server.public_origin")) {
				t.Fatalf("Validate() error = %v, want public-origin error", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}
