package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

// TestLoadAndConfigureConfigForServiceReportsRemovedSetting covers the half of
// server.https that still has to be fatal.
//
// `https: false` asks for plaintext, which Seed cannot serve, so accepting the
// config would leave the operator believing something untrue. `https: true` is
// a different case: it asks for what Seed already does, it is what every
// v0.200.0 config carries, and treating it as fatal crash-loops the upgrade
// (#2377). That one is migrated away with a warning instead.
func TestLoadAndConfigureConfigForServiceReportsRemovedSetting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed.json")
	if err := os.WriteFile(path, []byte(`{"server":{"https":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadAndConfigureConfigForService(path)
	if err == nil || !strings.Contains(err.Error(), `unknown field "https"`) {
		t.Fatalf("expected removed-setting error, got %v", err)
	}
}

// TestLoadAndConfigureConfigForServiceMigratesSatisfiedHTTPSToggle is the other
// half: the service loader must start on a config that only asks for HTTPS.
func TestLoadAndConfigureConfigForServiceMigratesSatisfiedHTTPSToggle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed.json")
	if err := os.WriteFile(path, []byte(`{"server":{"https":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAndConfigureConfigForService(path); err != nil {
		t.Fatalf("a config asking only for HTTPS must load: %v", err)
	}
}

func TestLoadAndConfigureConfigForServiceReturnsValidatedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed.json")
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = auth.SetupModePlaceholder
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadAndConfigureConfigForService(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Server.Port != cfg.Server.Port {
		t.Fatalf("server port = %d, want %d", loaded.Server.Port, cfg.Server.Port)
	}
}

func TestLoadAndConfigureConfigForServiceReportsPartialCertificatePair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed.json")
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = auth.SetupModePlaceholder
	cfg.Server.CertFile = "server.crt"
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAndConfigureConfigForService(path); err == nil ||
		!strings.Contains(err.Error(), "server.cert_file and server.key_file") {
		t.Fatalf("expected certificate-pair validation error, got %v", err)
	}
}
