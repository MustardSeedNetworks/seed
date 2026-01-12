package main

import (
	"encoding/json"
	"testing"
)

func TestOutputCredentialsJSONMarshaling(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{
		Username: "admin",
		Password: "testpassword123",
		Config:   "/etc/seed/seed.yaml",
	}

	// Test that credentials can be marshaled to JSON
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal credentials: %v", err)
	}

	// Verify it can be unmarshaled back
	var decoded setupCredentials
	if jsonErr := json.Unmarshal(data, &decoded); jsonErr != nil {
		t.Errorf("Failed to unmarshal: %v", jsonErr)
	}

	// Verify content matches
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

func TestOutputCredentialsJSONFieldNames(t *testing.T) {
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

func TestOutputResultJSONMarshaling(t *testing.T) {
	t.Parallel()

	result := ValidationResult{
		Valid:    true,
		Path:     "/etc/seed/seed.yaml",
		Errors:   nil,
		Warnings: []string{"test warning"},
	}

	// Test that result can be marshaled to JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	// Verify it can be unmarshaled back
	var decoded ValidationResult
	if jsonErr := json.Unmarshal(data, &decoded); jsonErr != nil {
		t.Errorf("Failed to unmarshal: %v", jsonErr)
	}

	// Verify content matches
	if decoded.Valid != result.Valid {
		t.Errorf("Valid mismatch: got %v, want %v", decoded.Valid, result.Valid)
	}
	if decoded.Path != result.Path {
		t.Errorf("Path mismatch: got %q, want %q", decoded.Path, result.Path)
	}
}

func TestOutputResultJSONFieldNames(t *testing.T) {
	t.Parallel()

	result := ValidationResult{
		Valid:    false,
		Path:     "/test/config.yaml",
		Errors:   []string{"error1"},
		Warnings: []string{"warning1"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	output := string(data)

	// Check JSON field names
	expectedFields := []string{`"valid"`, `"path"`, `"errors"`, `"warnings"`}
	for _, field := range expectedFields {
		if !containsSubstring(output, field) {
			t.Errorf("JSON output should contain field %s: %s", field, output)
		}
	}
}

func TestValidationResultJSONOmitEmpty(t *testing.T) {
	t.Parallel()

	result := ValidationResult{
		Valid:    true,
		Path:     "/test/config.yaml",
		Errors:   nil,
		Warnings: nil,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	output := string(data)

	// Empty slices should be omitted due to omitempty tag
	if containsSubstring(output, `"errors"`) {
		t.Error("Empty errors should be omitted from JSON output")
	}
	if containsSubstring(output, `"warnings"`) {
		t.Error("Empty warnings should be omitted from JSON output")
	}
}

func TestValidationResultStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   ValidationResult
		hasError bool
		hasWarn  bool
	}{
		{
			name: "valid no issues",
			result: ValidationResult{
				Valid: true,
				Path:  "/test/path",
			},
			hasError: false,
			hasWarn:  false,
		},
		{
			name: "invalid with errors",
			result: ValidationResult{
				Valid:  false,
				Path:   "/test/path",
				Errors: []string{"error1", "error2"},
			},
			hasError: true,
			hasWarn:  false,
		},
		{
			name: "valid with warnings",
			result: ValidationResult{
				Valid:    true,
				Path:     "/test/path",
				Warnings: []string{"warning1"},
			},
			hasError: false,
			hasWarn:  true,
		},
		{
			name: "invalid with both",
			result: ValidationResult{
				Valid:    false,
				Path:     "/test/path",
				Errors:   []string{"error1"},
				Warnings: []string{"warning1"},
			},
			hasError: true,
			hasWarn:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.hasError && len(tc.result.Errors) == 0 {
				t.Error("Expected errors but none found")
			}
			if !tc.hasError && len(tc.result.Errors) > 0 {
				t.Error("Expected no errors but found some")
			}
			if tc.hasWarn && len(tc.result.Warnings) == 0 {
				t.Error("Expected warnings but none found")
			}
			if !tc.hasWarn && len(tc.result.Warnings) > 0 {
				t.Error("Expected no warnings but found some")
			}
		})
	}
}

func TestSetupCredentialsStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		password string
		config   string
	}{
		{
			name:     "standard values",
			username: "admin",
			password: "securepassword123",
			config:   "/etc/seed/seed.yaml",
		},
		{
			name:     "empty values",
			username: "",
			password: "",
			config:   "",
		},
		{
			name:     "special characters",
			username: "admin@example.com",
			password: "p@ss!word#123",
			config:   "/path/with spaces/config.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			creds := setupCredentials{
				Username: tc.username,
				Password: tc.password,
				Config:   tc.config,
			}

			if creds.Username != tc.username {
				t.Errorf("Username: got %q, want %q", creds.Username, tc.username)
			}
			if creds.Password != tc.password {
				t.Errorf("Password: got %q, want %q", creds.Password, tc.password)
			}
			if creds.Config != tc.config {
				t.Errorf("Config: got %q, want %q", creds.Config, tc.config)
			}
		})
	}
}

func TestOutputCredentialsFunctionExists(t *testing.T) {
	t.Parallel()

	// Just verify the function exists and can be called with valid arguments
	// We don't capture output to avoid race conditions
	creds := setupCredentials{
		Username: "test",
		Password: "test",
		Config:   "/test/path",
	}

	// The function signature should be:
	// func outputCredentials(creds setupCredentials, asJSON bool) error
	_ = outputCredentials

	// Verify JSON marshaling works for the type
	_, err := json.Marshal(creds)
	if err != nil {
		t.Errorf("Should be able to marshal credentials: %v", err)
	}
}

func TestOutputResultFunctionExists(t *testing.T) {
	t.Parallel()

	// Just verify the function exists with correct signature
	result := ValidationResult{
		Valid: true,
		Path:  "/test/path",
	}

	// The function signature should be:
	// func outputResult(result ValidationResult, asJSON bool)
	_ = outputResult

	// Verify JSON marshaling works for the type
	_, err := json.Marshal(result)
	if err != nil {
		t.Errorf("Should be able to marshal result: %v", err)
	}
}

func TestValidationResultRoundTrip(t *testing.T) {
	t.Parallel()

	original := ValidationResult{
		Valid:    false,
		Path:     "/var/lib/seed/config.yaml",
		Errors:   []string{"error 1", "error 2"},
		Warnings: []string{"warning 1", "warning 2", "warning 3"},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded ValidationResult
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal: %v", unmarshalErr)
	}

	// Verify all fields match
	if decoded.Valid != original.Valid {
		t.Errorf("Valid mismatch: got %v, want %v", decoded.Valid, original.Valid)
	}
	if decoded.Path != original.Path {
		t.Errorf("Path mismatch: got %v, want %v", decoded.Path, original.Path)
	}
	if len(decoded.Errors) != len(original.Errors) {
		t.Errorf("Errors length mismatch: got %d, want %d", len(decoded.Errors), len(original.Errors))
	}
	if len(decoded.Warnings) != len(original.Warnings) {
		t.Errorf("Warnings length mismatch: got %d, want %d", len(decoded.Warnings), len(original.Warnings))
	}
}

func TestSetupCredentialsRoundTrip(t *testing.T) {
	t.Parallel()

	original := setupCredentials{
		Username: "admin",
		Password: "supersecret",
		Config:   "/etc/seed/seed.yaml",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded setupCredentials
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal: %v", unmarshalErr)
	}

	// Verify all fields match
	if decoded.Username != original.Username {
		t.Errorf("Username mismatch: got %q, want %q", decoded.Username, original.Username)
	}
	if decoded.Password != original.Password {
		t.Errorf("Password mismatch: got %q, want %q", decoded.Password, original.Password)
	}
	if decoded.Config != original.Config {
		t.Errorf("Config mismatch: got %q, want %q", decoded.Config, original.Config)
	}
}
