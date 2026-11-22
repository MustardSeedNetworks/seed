package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"

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
	path := filepath.Join(privateTempDir(t), "seed.json")
	if err := os.WriteFile(path, []byte(`{"server":{"https":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadAndConfigureConfigForService(path, testDirOptions(t))
	if err == nil || !strings.Contains(err.Error(), `unknown field "https"`) {
		t.Fatalf("expected removed-setting error, got %v", err)
	}
}

// TestLoadAndConfigureConfigForServiceMigratesSatisfiedHTTPSToggle is the other
// half: the service loader must start on a config that only asks for HTTPS.
func TestLoadAndConfigureConfigForServiceMigratesSatisfiedHTTPSToggle(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "seed.json")
	if err := os.WriteFile(path, []byte(`{"server":{"https":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAndConfigureConfigForService(path, testDirOptions(t)); err != nil {
		t.Fatalf("a config asking only for HTTPS must load: %v", err)
	}
}

func TestLoadAndConfigureConfigForServiceReturnsValidatedConfig(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "seed.json")
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = auth.SetupModePlaceholder
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadAndConfigureConfigForService(path, testDirOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Server.Port != cfg.Server.Port {
		t.Fatalf("server port = %d, want %d", loaded.Server.Port, cfg.Server.Port)
	}
}

func TestLoadAndConfigureConfigForServiceReportsPartialCertificatePair(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "seed.json")
	cfg := config.DefaultConfig()
	cfg.Auth.DefaultPasswordHash = auth.SetupModePlaceholder
	cfg.Server.CertFile = "server.crt"
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAndConfigureConfigForService(path, testDirOptions(t)); err == nil ||
		!strings.Contains(err.Error(), "server.cert_file and server.key_file") {
		t.Fatalf("expected certificate-pair validation error, got %v", err)
	}
}

// privateTempDir is a temporary directory in the form securedir accepts: its
// links resolved, since macOS keeps its temp tree behind /var.
func privateTempDir(t *testing.T) string {
	t.Helper()
	dir, err := securedir.Canonical(t.TempDir(), testDirOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	return dir
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
	opts := testDirOptions(t)
	sp := servicePathsUnder(filepath.Join(privateTempDir(t), "Seed"))

	got, err := prepareServiceDirs(sp, opts)
	if err != nil {
		t.Fatalf("prepareServiceDirs: %v", err)
	}
	for _, dir := range []string{got.Root, got.ConfigDir, got.DataDir, got.LogDir, got.CertDir} {
		if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
			t.Errorf("%s was not created as a directory (err %v)", dir, statErr)
		}
	}
	if got.ConfigFile != filepath.Join(got.ConfigDir, "seed.json") {
		t.Errorf("ConfigFile = %s, want it inside %s", got.ConfigFile, got.ConfigDir)
	}
}

// install.ps1 creates the root before the service ever starts; the service
// takes it over rather than refusing it.
func TestPrepareServiceDirsTakesOverTheInstallersRoot(t *testing.T) {
	opts := testDirOptions(t)
	sp := servicePathsUnder(filepath.Join(privateTempDir(t), "Seed"))
	if err := securedir.Ensure(sp.Root, opts); err != nil {
		t.Fatal(err)
	}

	if _, err := prepareServiceDirs(sp, opts); err != nil {
		t.Fatalf("prepareServiceDirs over an existing root: %v", err)
	}
	if info, err := os.Stat(sp.CertDir); err != nil || !info.IsDir() {
		t.Errorf("certificate directory missing under the existing root (err %v)", err)
	}
}

func TestPrepareServiceDirsReturnsCanonicalPaths(t *testing.T) {
	opts := testDirOptions(t)
	sp := servicePathsUnder(filepath.Join(t.TempDir(), "Seed"))
	want, err := securedir.Canonical(sp.Root, opts)
	if err != nil {
		t.Fatal(err)
	}

	got, err := prepareServiceDirs(sp, opts)
	if err != nil {
		t.Fatalf("prepareServiceDirs: %v", err)
	}
	if got.Root != want || got.CertDir != filepath.Join(want, "certs") {
		t.Errorf("Root = %s, CertDir = %s; want %s and its certs", got.Root, got.CertDir, want)
	}
}

func TestPrepareServiceDirsRefusesAFileAsTheRoot(t *testing.T) {
	file := filepath.Join(privateTempDir(t), "Seed")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareServiceDirs(servicePathsUnder(file), testDirOptions(t)); err == nil {
		t.Fatal("a file where the service directory belongs was accepted")
	}
}

func TestVerifyDataDir(t *testing.T) {
	opts := testDirOptions(t)
	dir := privateTempDir(t)
	if err := os.WriteFile(filepath.Join(dir, "seed.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	notDir := filepath.Join(dir, "file")
	if err := os.WriteFile(notDir, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"private directory with a regular file in it", dir, false},
		{"missing directory", filepath.Join(dir, "absent"), true},
		{"a file where the directory should be", notDir, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := verifyDataDir(tt.path, opts); (err != nil) != tt.wantErr {
				t.Errorf("verifyDataDir error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSavePrivateConfigRoundTrips(t *testing.T) {
	path := filepath.Join(privateTempDir(t), "seed.json")
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "persisted-secret"

	if err := savePrivateConfig(cfg, path, testDirOptions(t)); err != nil {
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

func TestCheckSystemTree(t *testing.T) {
	opts := testDirOptions(t)
	root := privateTempDir(t)
	system := servicePathsUnder(root)
	file := filepath.Join(root, "afile")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	missing := servicePathsUnder(filepath.Join(root, "absent"))
	noRoot := &paths.Paths{ConfigFile: system.ConfigFile}

	tests := []struct {
		name       string
		configPath string
		system     *paths.Paths
		wantErr    bool
	}{
		{"system config in a verified tree", system.ConfigFile, system, false},
		{"another config is not checked", filepath.Join(t.TempDir(), "seed.json"), system, false},
		{"a layout with no root has no tree to check", system.ConfigFile, noRoot, false},
		{"system tree missing", missing.ConfigFile, missing, true},
		{"system root is a file", servicePathsUnder(file).ConfigFile, servicePathsUnder(file), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSystemTree(tt.configPath, tt.system, opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkSystemTree error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && exitCodeFor(err) != exitRefused {
				t.Errorf("exit code = %d, want %d", exitCodeFor(err), exitRefused)
			}
		})
	}
}
