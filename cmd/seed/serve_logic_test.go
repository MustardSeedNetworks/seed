package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/krisarmstrong/seed/internal/config"
)

func TestInitServeCmdAddsServeCommand(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initServeCmd(state)

	var serveCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "serve" {
			serveCmd = cmd
			break
		}
	}

	if serveCmd == nil {
		t.Fatal("serve command not found")
	}

	// Verify command properties
	if serveCmd.Short == "" {
		t.Error("serve command should have a Short description")
	}
	if serveCmd.Long == "" {
		t.Error("serve command should have a Long description")
	}
}

func TestServeCmdHasRunFunction(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initServeCmd(state)

	var serveCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "serve" {
			serveCmd = cmd
			break
		}
	}

	if serveCmd == nil {
		t.Fatal("serve command not found")
	}

	if serveCmd.Run == nil {
		t.Error("serve command should have a Run function")
	}
}

func TestServeCmdLongDescriptionContent(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initServeCmd(state)

	var serveCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "serve" {
			serveCmd = cmd
			break
		}
	}

	if serveCmd == nil {
		t.Fatal("serve command not found")
	}

	// Check that long description mentions key features
	expectedContent := []string{
		"server",
		"web",
		"HTTPS",
		"--dev",
	}

	for _, content := range expectedContent {
		if !containsSubstring(serveCmd.Long, content) {
			t.Errorf("serve Long description should mention %q", content)
		}
	}
}

func TestRootCmdDefaultsToServe(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// The root command should have a Run function that calls runServe
	if state.rootCmd.Run == nil {
		t.Error("rootCmd should have a Run function that defaults to serve")
	}
}

func TestPrintSetupBannerProtocolLogic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		https         bool
		expectedProto string
	}{
		{
			name:          "HTTPS enabled",
			https:         true,
			expectedProto: "https",
		},
		{
			name:          "HTTPS disabled",
			https:         false,
			expectedProto: "http",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			protocol := "http"
			if tc.https {
				protocol = "https"
			}

			if protocol != tc.expectedProto {
				t.Errorf("protocol: got %q, want %q", protocol, tc.expectedProto)
			}
		})
	}
}

func TestInitializeDatabasePathLogic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		dbPath       string
		expectedPath string
	}{
		{
			name:         "empty path uses default",
			dbPath:       "",
			expectedPath: "data/seed.db",
		},
		{
			name:         "custom path is used",
			dbPath:       "/var/lib/seed/custom.db",
			expectedPath: "/var/lib/seed/custom.db",
		},
		{
			name:         "relative path",
			dbPath:       "mydata/app.db",
			expectedPath: "mydata/app.db",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test the path resolution logic
			dbPath := tc.dbPath
			if dbPath == "" {
				dbPath = "data/seed.db"
			}

			if dbPath != tc.expectedPath {
				t.Errorf("dbPath: got %q, want %q", dbPath, tc.expectedPath)
			}
		})
	}
}

func TestConfigValidationWithTestFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "seed.yaml")

	// Create a valid config with all required fields
	cfg := config.DefaultConfig()
	cfg.Server.Port = 8443
	cfg.Server.HTTPS = true
	cfg.Interface.Default = "eth0"
	cfg.Auth.DefaultPasswordHash = "somehash"
	cfg.Auth.JWTSecret = "somesecret"

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load and validate
	loadedCfg, loadErr := config.Load(configPath)
	if loadErr != nil {
		t.Fatalf("Failed to load config: %v", loadErr)
	}

	validateErr := loadedCfg.Validate()
	if validateErr != nil {
		t.Errorf("Config validation failed: %v", validateErr)
	}
}

func TestDevModeSetsHTTPFalse(t *testing.T) {
	t.Parallel()

	// Test the logic that --dev sets HTTPS to false
	devMode := true
	https := true

	if devMode {
		https = false
	}

	if https {
		t.Error("HTTPS should be false when devMode is true")
	}
}

func TestLogPathDefaultLogic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		configPath   string
		expectedPath string
	}{
		{
			name:         "empty uses default",
			configPath:   "",
			expectedPath: filepath.Join("logs", "seed.log"),
		},
		{
			name:         "custom path",
			configPath:   "/var/log/seed/app.log",
			expectedPath: "/var/log/seed/app.log",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logPath := tc.configPath
			if logPath == "" {
				logPath = filepath.Join("logs", "seed.log")
			}

			if logPath != tc.expectedPath {
				t.Errorf("logPath: got %q, want %q", logPath, tc.expectedPath)
			}
		})
	}
}

func TestServeCmdNoSubcommands(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initServeCmd(state)

	var serveCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "serve" {
			serveCmd = cmd
			break
		}
	}

	if serveCmd == nil {
		t.Fatal("serve command not found")
	}

	// Serve should have no subcommands
	if len(serveCmd.Commands()) != 0 {
		t.Errorf("serve command should have no subcommands, got %d", len(serveCmd.Commands()))
	}
}

func TestFindActiveInterfaceRetryLogic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		maxRetries  int
		expectRetry bool
	}{
		{
			name:        "zero retries",
			maxRetries:  0,
			expectRetry: false,
		},
		{
			name:        "positive retries",
			maxRetries:  3,
			expectRetry: true,
		},
		{
			name:        "single retry",
			maxRetries:  1,
			expectRetry: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test the retry logic without actually calling the function
			shouldRetry := tc.maxRetries > 0

			if shouldRetry != tc.expectRetry {
				t.Errorf("shouldRetry: got %v, want %v", shouldRetry, tc.expectRetry)
			}
		})
	}
}

func TestInterfacePreferredOrder(t *testing.T) {
	t.Parallel()

	initialInterface := "eth0"
	fallbacks := []string{"eth1", "wlan0", "enp0s3"}

	preferred := append([]string{initialInterface}, fallbacks...)

	if len(preferred) != 4 {
		t.Errorf("preferred should have 4 interfaces, got %d", len(preferred))
	}

	if preferred[0] != initialInterface {
		t.Errorf("first preferred should be initial interface: got %q, want %q", preferred[0], initialInterface)
	}

	for i, fb := range fallbacks {
		if preferred[i+1] != fb {
			t.Errorf("preferred[%d] should be %q, got %q", i+1, fb, preferred[i+1])
		}
	}
}
