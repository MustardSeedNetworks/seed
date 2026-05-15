package main

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/paths"
)

func TestModeString(t *testing.T) {

	tests := []struct {
		name     string
		mode     paths.Mode
		expected string
	}{
		{
			name:     "system mode",
			mode:     paths.ModeSystem,
			expected: "system",
		},
		{
			name:     "user mode",
			mode:     paths.ModeUser,
			expected: "user",
		},
		{
			name:     "auto mode",
			mode:     paths.ModeAuto,
			expected: "auto",
		},
		{
			name:     "unknown mode",
			mode:     paths.Mode(999),
			expected: "unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := modeString(tc.mode)
			if result != tc.expected {
				t.Errorf("modeString(%d) = %q, want %q", tc.mode, result, tc.expected)
			}
		})
	}
}

func TestInstallFlagsStruct(t *testing.T) {

	// Test default values
	flags := installFlags{}

	if flags.systemMode {
		t.Error("systemMode should default to false")
	}
	if flags.userMode {
		t.Error("userMode should default to false")
	}
	if flags.noService {
		t.Error("noService should default to false")
	}
	if flags.force {
		t.Error("force should default to false")
	}
}

func TestInstallFlagsAllTrue(t *testing.T) {

	flags := installFlags{
		systemMode: true,
		userMode:   true,
		noService:  true,
		force:      true,
	}

	if !flags.systemMode {
		t.Error("systemMode should be true")
	}
	if !flags.userMode {
		t.Error("userMode should be true")
	}
	if !flags.noService {
		t.Error("noService should be true")
	}
	if !flags.force {
		t.Error("force should be true")
	}
}

func TestServiceConfigStruct(t *testing.T) {

	cfg := serviceConfig{
		User:       "seed",
		Group:      "seed",
		BinaryPath: "/usr/local/bin/seed",
		ConfigDir:  "/etc/seed",
		DataDir:    "/var/lib/seed",
		LogDir:     "/var/log/seed",
		CacheDir:   "/var/cache/seed",
	}

	if cfg.User != "seed" {
		t.Errorf("User should be 'seed', got %q", cfg.User)
	}
	if cfg.Group != "seed" {
		t.Errorf("Group should be 'seed', got %q", cfg.Group)
	}
	if cfg.BinaryPath != "/usr/local/bin/seed" {
		t.Errorf("BinaryPath mismatch: got %q", cfg.BinaryPath)
	}
	if cfg.ConfigDir != "/etc/seed" {
		t.Errorf("ConfigDir mismatch: got %q", cfg.ConfigDir)
	}
	if cfg.DataDir != "/var/lib/seed" {
		t.Errorf("DataDir mismatch: got %q", cfg.DataDir)
	}
	if cfg.LogDir != "/var/log/seed" {
		t.Errorf("LogDir mismatch: got %q", cfg.LogDir)
	}
	if cfg.CacheDir != "/var/cache/seed" {
		t.Errorf("CacheDir mismatch: got %q", cfg.CacheDir)
	}
}

func TestSystemdServiceTemplateContainsRequiredFields(t *testing.T) {

	requiredFields := []string{
		"[Unit]",
		"[Service]",
		"[Install]",
		"Description=",
		"Type=simple",
		"ExecStart=",
		"Restart=on-failure",
		"{{.BinaryPath}}",
		"{{.User}}",
		"{{.Group}}",
		"{{.DataDir}}",
		"{{.ConfigDir}}",
		"{{.LogDir}}",
		"{{.CacheDir}}",
	}

	for _, field := range requiredFields {
		if !containsSubstring(systemdServiceTemplate, field) {
			t.Errorf("systemdServiceTemplate should contain %q", field)
		}
	}
}

func TestUserServiceTemplateContainsRequiredFields(t *testing.T) {

	requiredFields := []string{
		"[Unit]",
		"[Service]",
		"[Install]",
		"Description=",
		"Type=simple",
		"ExecStart=",
		"Restart=on-failure",
		"{{.BinaryPath}}",
		"WantedBy=default.target",
	}

	for _, field := range requiredFields {
		if !containsSubstring(userServiceTemplate, field) {
			t.Errorf("userServiceTemplate should contain %q", field)
		}
	}
}

func TestUserServiceTemplateDoesNotContainSystemFields(t *testing.T) {

	// User service template should not require system-level fields
	systemFields := []string{
		"{{.User}}",
		"{{.Group}}",
		"{{.DataDir}}",
		"{{.ConfigDir}}",
		"{{.LogDir}}",
		"{{.CacheDir}}",
	}

	for _, field := range systemFields {
		if containsSubstring(userServiceTemplate, field) {
			t.Errorf("userServiceTemplate should not contain %q", field)
		}
	}
}

func TestTimeoutConstants(t *testing.T) {

	if userCheckTimeoutSeconds <= 0 {
		t.Error("userCheckTimeoutSeconds should be positive")
	}
	if commandTimeoutSeconds <= 0 {
		t.Error("commandTimeoutSeconds should be positive")
	}
	if userCheckTimeoutSeconds > commandTimeoutSeconds {
		t.Error("userCheckTimeoutSeconds should not exceed commandTimeoutSeconds")
	}
}

// containsSubstring is a helper function to check if a string contains a substring.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
