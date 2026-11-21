package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/paths"
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

func servicePathsUnder(root string) *paths.Paths {
	return &paths.Paths{
		Root:       root,
		ConfigDir:  filepath.Join(root, "config"),
		ConfigFile: filepath.Join(root, "config", "seed.json"),
		DataDir:    filepath.Join(root, "data"),
		LogDir:     filepath.Join(root, "logs"),
		CertDir:    filepath.Join(root, "certs"),
	}
}

func TestAnchorServicePaths(t *testing.T) {
	sp := servicePathsUnder(t.TempDir())
	elsewhere := filepath.Join(t.TempDir(), "keep.db")

	tests := []struct {
		name string
		set  func(*config.Config)
		got  func(*config.Config) string
		want string
	}{
		{
			name: "stock database path lands in the data directory",
			set:  func(c *config.Config) { c.Database.Path = "data/seed.db" },
			got:  func(c *config.Config) string { return c.Database.Path },
			want: filepath.Join(sp.Root, "data", "seed.db"),
		},
		{
			name: "unset database path takes the service default",
			set:  func(c *config.Config) { c.Database.Path = "" },
			got:  func(c *config.Config) string { return c.Database.Path },
			want: filepath.Join(sp.DataDir, "seed.db"),
		},
		{
			name: "absolute database path is left alone",
			set:  func(c *config.Config) { c.Database.Path = elsewhere },
			got:  func(c *config.Config) string { return c.Database.Path },
			want: elsewhere,
		},
		{
			name: "unset log file takes the service default",
			set:  func(c *config.Config) { c.Logging.File = "" },
			got:  func(c *config.Config) string { return c.Logging.File },
			want: filepath.Join(sp.LogDir, "seed.log"),
		},
		{
			name: "relative log file is anchored",
			set:  func(c *config.Config) { c.Logging.File = "logs/custom.log" },
			got:  func(c *config.Config) string { return c.Logging.File },
			want: filepath.Join(sp.Root, "logs", "custom.log"),
		},
		{
			name: "OUI file is anchored",
			set:  func(c *config.Config) { c.NetworkDiscovery.OUIFilePath = "data/oui.txt" },
			got:  func(c *config.Config) string { return c.NetworkDiscovery.OUIFilePath },
			want: filepath.Join(sp.Root, "data", "oui.txt"),
		},
		{
			name: "unset TLS pair stays unset so the generated pair is used",
			set: func(c *config.Config) {
				c.Server.CertFile, c.Server.KeyFile = "", ""
			},
			got:  func(c *config.Config) string { return c.Server.CertFile + c.Server.KeyFile },
			want: "",
		},
		{
			name: "relative TLS files are anchored",
			set: func(c *config.Config) {
				c.Server.CertFile, c.Server.KeyFile = "certs/own.crt", "certs/own.key"
			},
			got:  func(c *config.Config) string { return c.Server.CertFile + "|" + c.Server.KeyFile },
			want: filepath.Join(sp.Root, "certs", "own.crt") + "|" + filepath.Join(sp.Root, "certs", "own.key"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.set(cfg)
			anchorServicePaths(cfg, sp)
			if got := tt.got(cfg); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrepareServiceDirsCreatesTheTree(t *testing.T) {
	sp := servicePathsUnder(filepath.Join(t.TempDir(), "Seed"))

	if err := prepareServiceDirs(sp); err != nil {
		t.Fatalf("prepareServiceDirs: %v", err)
	}
	for _, dir := range []string{sp.Root, sp.ConfigDir, sp.DataDir, sp.LogDir, sp.CertDir} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("%s was not created as a directory (err %v)", dir, err)
		}
	}
}

func TestVerifyDataDir(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "seed.json")
	if err := os.WriteFile(regular, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	notDir := filepath.Join(dir, "file")
	if err := os.WriteFile(notDir, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0o750); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		files   []string
		wantErr bool
	}{
		{"directory with an existing regular file", dir, []string{regular}, false},
		{"missing files are fine", dir, []string{filepath.Join(dir, "absent.db")}, false},
		{"missing directory", filepath.Join(dir, "absent"), nil, true},
		{"a file where the directory should be", notDir, nil, true},
		{"a directory where a file should be", dir, []string{subdir}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := verifyDataDir(tt.path, tt.files...); (err != nil) != tt.wantErr {
				t.Errorf("verifyDataDir error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSavePrivateConfigRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed.json")
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "persisted-secret"

	if err := savePrivateConfig(cfg, path); err != nil {
		t.Fatalf("savePrivateConfig: %v", err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Auth.JWTSecret != "persisted-secret" {
		t.Errorf("JWT secret = %q, want the saved one", loaded.Auth.JWTSecret)
	}
}
