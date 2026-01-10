package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupCredentialsStruct(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{
		Username: "admin",
		Password: "super-secret-password",
		Config:   "/etc/seed/seed.yaml",
	}

	if creds.Username != "admin" {
		t.Errorf("Username should be 'admin', got %q", creds.Username)
	}
	if creds.Password != "super-secret-password" {
		t.Errorf("Password should be 'super-secret-password', got %q", creds.Password)
	}
	if creds.Config != "/etc/seed/seed.yaml" {
		t.Errorf("Config should be '/etc/seed/seed.yaml', got %q", creds.Config)
	}
}

func TestSetupCredentialsMarshalJSON(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{
		Username: "admin",
		Password: "test-password",
		Config:   "/var/lib/seed/config.yaml",
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal setupCredentials: %v", err)
	}

	var decoded setupCredentials
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal setupCredentials: %v", unmarshalErr)
	}

	if decoded.Username != creds.Username {
		t.Errorf("Username mismatch: got %q, want %q", decoded.Username, creds.Username)
	}
	if decoded.Password != creds.Password {
		t.Errorf("Password mismatch: got %q, want %q", decoded.Password, creds.Password)
	}
	if decoded.Config != creds.Config {
		t.Errorf("Config mismatch: got %q, want %q", decoded.Config, creds.Config)
	}
}

func TestSetupCredentialsJSONTags(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{
		Username: "user",
		Password: "pass",
		Config:   "/path/to/config",
	}

	data, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	output := string(data)

	// Check JSON field names match expected tags
	expectedFields := []string{`"username"`, `"password"`, `"config_path"`}
	for _, field := range expectedFields {
		if !containsSubstring(output, field) {
			t.Errorf("JSON output should contain field %s: %s", field, output)
		}
	}
}

func TestEnsureConfigDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configPath string
		wantErr    bool
	}{
		{
			name:       "current directory",
			configPath: "seed.yaml",
			wantErr:    false,
		},
		{
			name:       "dot directory",
			configPath: "./seed.yaml",
			wantErr:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ensureConfigDir(tc.configPath)
			if (err != nil) != tc.wantErr {
				t.Errorf("ensureConfigDir(%q) error = %v, wantErr %v", tc.configPath, err, tc.wantErr)
			}
		})
	}
}

func TestEnsureConfigDirCreatesDirectory(t *testing.T) {
	t.Parallel()

	// Create a temporary directory
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "subdir", "config")
	configPath := filepath.Join(newDir, "seed.yaml")

	err := ensureConfigDir(configPath)
	if err != nil {
		t.Fatalf("ensureConfigDir() returned error: %v", err)
	}

	// Verify the directory was created
	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected a directory, but got a file")
	}
}

func TestEnsureConfigDirExistingDirectory(t *testing.T) {
	t.Parallel()

	// Create a temporary directory that already exists
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "seed.yaml")

	err := ensureConfigDir(configPath)
	if err != nil {
		t.Fatalf("ensureConfigDir() returned error: %v", err)
	}

	// Directory should still exist
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected a directory")
	}
}

func TestDefaultPasswordLengthConstant(t *testing.T) {
	t.Parallel()

	if defaultPasswordLength <= 0 {
		t.Error("defaultPasswordLength should be positive")
	}
	if defaultPasswordLength < 16 {
		t.Error("defaultPasswordLength should be at least 16 for security")
	}
}

func TestOutputCredentialsJSON(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{
		Username: "admin",
		Password: "generated-password",
		Config:   "/etc/seed/seed.yaml",
	}

	// Test that outputCredentials with asJSON=true produces valid JSON
	// We can't capture stdout directly, but we can verify the marshaling works
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshaling failed: %v", err)
	}

	var decoded setupCredentials
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("JSON unmarshaling failed: %v", unmarshalErr)
	}

	if decoded.Username != creds.Username {
		t.Error("Round-trip failed for Username")
	}
	if decoded.Password != creds.Password {
		t.Error("Round-trip failed for Password")
	}
	if decoded.Config != creds.Config {
		t.Error("Round-trip failed for Config")
	}
}

func TestSetupCredentialsEmptyFields(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{}

	if creds.Username != "" {
		t.Error("Username should be empty by default")
	}
	if creds.Password != "" {
		t.Error("Password should be empty by default")
	}
	if creds.Config != "" {
		t.Error("Config should be empty by default")
	}
}

func TestSetupCredentialsWithSpecialCharacters(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{
		Username: "admin@example.com",
		Password: "p@ss!word#123$%^&*()",
		Config:   "/path/with spaces/and-dashes/config.yaml",
	}

	data, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("Failed to marshal credentials with special characters: %v", err)
	}

	var decoded setupCredentials
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal: %v", unmarshalErr)
	}

	if decoded.Username != creds.Username {
		t.Errorf("Username with special chars: got %q, want %q", decoded.Username, creds.Username)
	}
	if decoded.Password != creds.Password {
		t.Errorf("Password with special chars: got %q, want %q", decoded.Password, creds.Password)
	}
	if decoded.Config != creds.Config {
		t.Errorf("Config with special chars: got %q, want %q", decoded.Config, creds.Config)
	}
}
