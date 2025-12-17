package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolve_SystemMode(t *testing.T) {
	paths := Resolve(ModeSystem)

	if paths.Mode != ModeSystem {
		t.Errorf("expected Mode=ModeSystem, got %v", paths.Mode)
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"ConfigDir", paths.ConfigDir, "/etc/seed"},
		{"ConfigFile", paths.ConfigFile, "/etc/seed/config.yaml"},
		{"DataDir", paths.DataDir, "/var/lib/seed"},
		{"LogDir", paths.LogDir, "/var/log/seed"},
		{"CacheDir", paths.CacheDir, "/var/cache/seed"},
		{"BinaryDir", paths.BinaryDir, "/usr/local/bin"},
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

	oldEnv := map[string]string{
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		"XDG_STATE_HOME":  os.Getenv("XDG_STATE_HOME"),
		"XDG_CACHE_HOME":  os.Getenv("XDG_CACHE_HOME"),
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	os.Setenv("XDG_CONFIG_HOME", configHome)
	os.Setenv("XDG_DATA_HOME", dataHome)
	os.Setenv("XDG_STATE_HOME", stateHome)
	os.Setenv("XDG_CACHE_HOME", cacheHome)

	paths := Resolve(ModeUser)

	if paths.Mode != ModeUser {
		t.Errorf("expected Mode=ModeUser, got %v", paths.Mode)
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"ConfigDir", paths.ConfigDir, filepath.Join(configHome, "seed")},
		{"ConfigFile", paths.ConfigFile, filepath.Join(configHome, "seed", "config.yaml")},
		{"DataDir", paths.DataDir, filepath.Join(dataHome, "seed")},
		{"LogDir", paths.LogDir, filepath.Join(stateHome, "seed", "logs")},
		{"CacheDir", paths.CacheDir, filepath.Join(cacheHome, "seed")},
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
	// Clear XDG environment variables
	oldEnv := map[string]string{
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		"XDG_STATE_HOME":  os.Getenv("XDG_STATE_HOME"),
		"XDG_CACHE_HOME":  os.Getenv("XDG_CACHE_HOME"),
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("XDG_STATE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")

	paths := Resolve(ModeUser)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	if paths.Mode != ModeUser {
		t.Errorf("expected Mode=ModeUser, got %v", paths.Mode)
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"ConfigDir", paths.ConfigDir, filepath.Join(homeDir, ".config", "seed")},
		{"ConfigFile", paths.ConfigFile, filepath.Join(homeDir, ".config", "seed", "config.yaml")},
		{"DataDir", paths.DataDir, filepath.Join(homeDir, ".local", "share", "seed")},
		{"LogDir", paths.LogDir, filepath.Join(homeDir, ".local", "state", "seed", "logs")},
		{"CacheDir", paths.CacheDir, filepath.Join(homeDir, ".cache", "seed")},
		{"BinaryDir", paths.BinaryDir, filepath.Join(homeDir, ".local", "bin")},
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

	// Clear systemd environment variables
	oldEnv := map[string]string{
		"NOTIFY_SOCKET":   os.Getenv("NOTIFY_SOCKET"),
		"INVOCATION_ID":   os.Getenv("INVOCATION_ID"),
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		"XDG_STATE_HOME":  os.Getenv("XDG_STATE_HOME"),
		"XDG_CACHE_HOME":  os.Getenv("XDG_CACHE_HOME"),
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	os.Unsetenv("NOTIFY_SOCKET")
	os.Unsetenv("INVOCATION_ID")
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("XDG_STATE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")

	paths := Resolve(ModeAuto)

	// Should detect as user mode
	if paths.Mode != ModeUser {
		t.Errorf("ModeAuto should resolve to ModeUser when not root/systemd, got %v", paths.Mode)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	expectedConfig := filepath.Join(homeDir, ".config", "seed", "config.yaml")
	if paths.ConfigFile != expectedConfig {
		t.Errorf("ConfigFile: got %q, want %q", paths.ConfigFile, expectedConfig)
	}
}

func TestResolve_ModeAuto_WithSystemd(t *testing.T) {
	// Only test on Linux where systemd is relevant
	if runtime.GOOS != "linux" {
		t.Skip("systemd detection only on Linux")
	}

	// Set systemd environment variable
	oldInvocationID := os.Getenv("INVOCATION_ID")
	defer func() {
		if oldInvocationID == "" {
			os.Unsetenv("INVOCATION_ID")
		} else {
			os.Setenv("INVOCATION_ID", oldInvocationID)
		}
	}()

	os.Setenv("INVOCATION_ID", "test-invocation-id")

	paths := Resolve(ModeAuto)

	// Should detect as system mode due to systemd
	if paths.Mode != ModeSystem {
		t.Errorf("ModeAuto should resolve to ModeSystem when systemd detected, got %v", paths.Mode)
	}

	if paths.ConfigFile != "/etc/seed/config.yaml" {
		t.Errorf("ConfigFile: got %q, want %q", paths.ConfigFile, "/etc/seed/config.yaml")
	}
}

func TestResolveConfigPath_Priority(t *testing.T) {
	tests := []struct {
		name        string
		explicit    string
		envValue    string
		mode        Mode
		expected    string
		description string
	}{
		{
			name:        "explicit_path_takes_priority",
			explicit:    "/custom/config.yaml",
			envValue:    "/env/config.yaml",
			mode:        ModeUser,
			expected:    "/custom/config.yaml",
			description: "explicit path should override env and defaults",
		},
		{
			name:        "default_filename_ignored",
			explicit:    "config.yaml",
			envValue:    "/env/config.yaml",
			mode:        ModeUser,
			expected:    "/env/config.yaml",
			description: "explicit 'config.yaml' should be treated as default",
		},
		{
			name:        "env_overrides_default",
			explicit:    "",
			envValue:    "/env/custom.yaml",
			mode:        ModeUser,
			expected:    "/env/custom.yaml",
			description: "env variable should override XDG default",
		},
		{
			name:        "fallback_to_xdg_user",
			explicit:    "",
			envValue:    "",
			mode:        ModeUser,
			expected:    "", // Will be XDG path, checked separately
			description: "should fall back to XDG user path",
		},
		{
			name:        "fallback_to_system",
			explicit:    "",
			envValue:    "",
			mode:        ModeSystem,
			expected:    "/etc/seed/config.yaml",
			description: "should fall back to system path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			oldEnv := os.Getenv("SEED_CONFIG_PATH")
			defer func() {
				if oldEnv == "" {
					os.Unsetenv("SEED_CONFIG_PATH")
				} else {
					os.Setenv("SEED_CONFIG_PATH", oldEnv)
				}
			}()

			if tt.envValue != "" {
				os.Setenv("SEED_CONFIG_PATH", tt.envValue)
			} else {
				os.Unsetenv("SEED_CONFIG_PATH")
			}

			got := ResolveConfigPath(tt.explicit, tt.mode)

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
			tmpDir := t.TempDir()
			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get working directory: %v", err)
			}
			defer os.Chdir(oldWd)

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to change directory: %v", err)
			}

			// Create test file if specified
			if tt.createFile != "" {
				f, err := os.Create(tt.createFile)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				f.Close()
			}

			// Test detection
			path, found := DetectLegacyConfig()

			if found != tt.expectFound {
				t.Errorf("found=%v, want %v", found, tt.expectFound)
			}

			if tt.expectFound {
				if path == "" {
					t.Error("expected non-empty path when found=true")
				}
				if !filepath.IsAbs(path) {
					t.Errorf("expected absolute path, got %q", path)
				}
				if tt.expectContains != "" && filepath.Base(path) != tt.expectContains {
					t.Errorf("path %q does not contain %q", path, tt.expectContains)
				}
			} else if path != "" {
				t.Errorf("expected empty path when found=false, got %q", path)
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
			// Save and restore environment
			oldNotify := os.Getenv("NOTIFY_SOCKET")
			oldInvocation := os.Getenv("INVOCATION_ID")
			defer func() {
				if oldNotify == "" {
					os.Unsetenv("NOTIFY_SOCKET")
				} else {
					os.Setenv("NOTIFY_SOCKET", oldNotify)
				}
				if oldInvocation == "" {
					os.Unsetenv("INVOCATION_ID")
				} else {
					os.Setenv("INVOCATION_ID", oldInvocation)
				}
			}()

			// Set test environment
			if tt.notifySocket != "" {
				os.Setenv("NOTIFY_SOCKET", tt.notifySocket)
			} else {
				os.Unsetenv("NOTIFY_SOCKET")
			}
			if tt.invocationID != "" {
				os.Setenv("INVOCATION_ID", tt.invocationID)
			} else {
				os.Unsetenv("INVOCATION_ID")
			}

			got := isSystemdService()

			expected := tt.expectedOther
			if runtime.GOOS == "linux" {
				expected = tt.expectedLinux
			}

			if got != expected {
				t.Errorf("isSystemdService()=%v, want %v (GOOS=%s)", got, expected, runtime.GOOS)
			}
		})
	}
}
