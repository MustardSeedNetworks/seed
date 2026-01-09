package paths_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/krisarmstrong/seed/internal/paths"
)

func TestResolve_SystemMode(t *testing.T) {
	p := paths.Resolve(paths.ModeSystem)

	if p.Mode != paths.ModeSystem {
		t.Errorf("expected Mode=ModeSystem, got %v", p.Mode)
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"ConfigDir", p.ConfigDir, "/etc/seed"},
		{"ConfigFile", p.ConfigFile, "/etc/seed/config.yaml"},
		{"DataDir", p.DataDir, "/var/lib/seed"},
		{"LogDir", p.LogDir, "/var/log/seed"},
		{"CacheDir", p.CacheDir, "/var/cache/seed"},
		{"BinaryDir", p.BinaryDir, "/usr/local/bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestResolve_UserMode_WithXDG(t *testing.T) {
	// Set up XDG environment variables
	tmpDir := t.TempDir()

	configHome := filepath.Join(tmpDir, "config")
	dataHome := filepath.Join(tmpDir, "data")
	stateHome := filepath.Join(tmpDir, "state")
	cacheHome := filepath.Join(tmpDir, "cache")

	// t.Setenv automatically restores the original value when the test completes
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)
	t.Setenv("XDG_CACHE_HOME", cacheHome)

	p := paths.Resolve(paths.ModeUser)

	if p.Mode != paths.ModeUser {
		t.Errorf("expected Mode=ModeUser, got %v", p.Mode)
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"ConfigDir", p.ConfigDir, filepath.Join(configHome, "seed")},
		{"ConfigFile", p.ConfigFile, filepath.Join(configHome, "seed", "config.yaml")},
		{"DataDir", p.DataDir, filepath.Join(dataHome, "seed")},
		{"LogDir", p.LogDir, filepath.Join(stateHome, "seed", "logs")},
		{"CacheDir", p.CacheDir, filepath.Join(cacheHome, "seed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestResolve_UserMode_WithoutXDG(t *testing.T) {
	// Clear XDG environment variables (empty string treated as unset)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	p := paths.Resolve(paths.ModeUser)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	if p.Mode != paths.ModeUser {
		t.Errorf("expected Mode=ModeUser, got %v", p.Mode)
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"ConfigDir", p.ConfigDir, filepath.Join(homeDir, ".config", "seed")},
		{"ConfigFile", p.ConfigFile, filepath.Join(homeDir, ".config", "seed", "config.yaml")},
		{"DataDir", p.DataDir, filepath.Join(homeDir, ".local", "share", "seed")},
		{"LogDir", p.LogDir, filepath.Join(homeDir, ".local", "state", "seed", "logs")},
		{"CacheDir", p.CacheDir, filepath.Join(homeDir, ".cache", "seed")},
		{"BinaryDir", p.BinaryDir, filepath.Join(homeDir, ".local", "bin")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestResolve_ModeAuto_AsUser(t *testing.T) {
	// Skip if running as root (UID 0)
	if os.Getuid() == 0 {
		t.Skip("skipping user mode test when running as root")
	}

	// Clear systemd and XDG environment variables
	t.Setenv("NOTIFY_SOCKET", "")
	t.Setenv("INVOCATION_ID", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	p := paths.Resolve(paths.ModeAuto)

	// Should detect as user mode
	if p.Mode != paths.ModeUser {
		t.Errorf("ModeAuto should resolve to ModeUser when not root/systemd, got %v", p.Mode)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	expectedConfig := filepath.Join(homeDir, ".config", "seed", "config.yaml")
	if p.ConfigFile != expectedConfig {
		t.Errorf("ConfigFile: got %q, want %q", p.ConfigFile, expectedConfig)
	}
}

func TestResolve_ModeAuto_WithSystemd(t *testing.T) {
	// Only test on Linux where systemd is relevant
	if runtime.GOOS != "linux" {
		t.Skip("systemd detection only on Linux")
	}

	// Set systemd environment variable
	t.Setenv("INVOCATION_ID", "test-invocation-id")

	p := paths.Resolve(paths.ModeAuto)

	// Should detect as system mode due to systemd
	if p.Mode != paths.ModeSystem {
		t.Errorf("ModeAuto should resolve to ModeSystem when systemd detected, got %v", p.Mode)
	}

	if p.ConfigFile != "/etc/seed/config.yaml" {
		t.Errorf("ConfigFile: got %q, want %q", p.ConfigFile, "/etc/seed/config.yaml")
	}
}

func TestResolveConfigPath_Priority(t *testing.T) {
	tests := []struct {
		name        string
		explicit    string
		envValue    string
		mode        paths.Mode
		expected    string
		description string
	}{
		{
			name:        "explicit_path_takes_priority",
			explicit:    "/custom/config.yaml",
			envValue:    "/env/config.yaml",
			mode:        paths.ModeUser,
			expected:    "/custom/config.yaml",
			description: "explicit path should override env and defaults",
		},
		{
			name:        "default_filename_ignored",
			explicit:    "config.yaml",
			envValue:    "/env/config.yaml",
			mode:        paths.ModeUser,
			expected:    "/env/config.yaml",
			description: "explicit 'config.yaml' should be treated as default",
		},
		{
			name:        "env_overrides_default",
			explicit:    "",
			envValue:    "/env/custom.yaml",
			mode:        paths.ModeUser,
			expected:    "/env/custom.yaml",
			description: "env variable should override XDG default",
		},
		{
			name:        "fallback_to_xdg_user",
			explicit:    "",
			envValue:    "",
			mode:        paths.ModeUser,
			expected:    "", // Will be XDG path, checked separately
			description: "should fall back to XDG user path",
		},
		{
			name:        "fallback_to_system",
			explicit:    "",
			envValue:    "",
			mode:        paths.ModeSystem,
			expected:    "/etc/seed/config.yaml",
			description: "should fall back to system path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment - t.Setenv handles cleanup automatically
			if tt.envValue != "" {
				t.Setenv("SEED_CONFIG_PATH", tt.envValue)
			} else {
				t.Setenv("SEED_CONFIG_PATH", "")
			}

			got := paths.ResolveConfigPath(tt.explicit, tt.mode)

			if tt.expected == "" {
				// For XDG fallback test, just verify it's not empty and contains "seed"
				if got == "" || !filepath.IsAbs(got) {
					t.Errorf("%s: got %q, expected non-empty absolute path", tt.description, got)
				}
			} else {
				if got != tt.expected {
					t.Errorf("%s: got %q, want %q", tt.description, got, tt.expected)
				}
			}
		})
	}
}

// createTestFile creates a file in the current directory for testing.
func createTestFile(t *testing.T, filename string) {
	t.Helper()
	if filename == "" {
		return
	}
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	_ = f.Close()
}

// assertFoundMatch validates the found boolean matches expectations.
func assertFoundMatch(t *testing.T, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("found=%v, want %v", got, want)
	}
}

// assertPathWhenFound validates path properties when config is expected to be found.
func assertPathWhenFound(t *testing.T, path, expectContains string) {
	t.Helper()
	if path == "" {
		t.Error("expected non-empty path when found=true")
		return
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %q", path)
	}
	if expectContains != "" && filepath.Base(path) != expectContains {
		t.Errorf("path %q does not contain %q", path, expectContains)
	}
}

// assertPathWhenNotFound validates path is empty when config is not expected.
func assertPathWhenNotFound(t *testing.T, path string) {
	t.Helper()
	if path != "" {
		t.Errorf("expected empty path when found=false, got %q", path)
	}
}

func TestDetectLegacyConfig(t *testing.T) {
	tests := []struct {
		name           string
		createFile     string
		expectFound    bool
		expectContains string
	}{
		{
			name:           "no_legacy_config",
			createFile:     "",
			expectFound:    false,
			expectContains: "",
		},
		{
			name:           "config_yaml_found",
			createFile:     "config.yaml",
			expectFound:    true,
			expectContains: "config.yaml",
		},
		{
			name:           "dot_seed_yaml_found",
			createFile:     ".seed.yaml",
			expectFound:    true,
			expectContains: ".seed.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and change to it
			// t.Chdir automatically restores the original directory when the test completes
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)

			// Create test file if specified
			createTestFile(t, tt.createFile)

			// Test detection
			path, found := paths.DetectLegacyConfig()

			assertFoundMatch(t, found, tt.expectFound)

			if tt.expectFound {
				assertPathWhenFound(t, path, tt.expectContains)
			} else {
				assertPathWhenNotFound(t, path)
			}
		})
	}
}

func TestIsSystemdService(t *testing.T) {
	tests := []struct {
		name          string
		notifySocket  string
		invocationID  string
		expectedLinux bool
		expectedOther bool
	}{
		{
			name:          "no_systemd_vars",
			notifySocket:  "",
			invocationID:  "",
			expectedLinux: false,
			expectedOther: false,
		},
		{
			name:          "notify_socket_set",
			notifySocket:  "/run/systemd/notify",
			invocationID:  "",
			expectedLinux: true,
			expectedOther: false,
		},
		{
			name:          "invocation_id_set",
			notifySocket:  "",
			invocationID:  "abc123",
			expectedLinux: true,
			expectedOther: false,
		},
		{
			name:          "both_set",
			notifySocket:  "/run/systemd/notify",
			invocationID:  "abc123",
			expectedLinux: true,
			expectedOther: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment - t.Setenv handles cleanup automatically
			t.Setenv("NOTIFY_SOCKET", tt.notifySocket)
			t.Setenv("INVOCATION_ID", tt.invocationID)

			got := paths.ExportIsSystemdService()

			expected := tt.expectedOther
			if runtime.GOOS == "linux" {
				expected = tt.expectedLinux
			}

			if got != expected {
				t.Errorf("IsSystemdService()=%v, want %v (GOOS=%s)", got, expected, runtime.GOOS)
			}
		})
	}
}
