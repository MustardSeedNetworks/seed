package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestOutputCredentialsJSONFormat(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{
		Username: "admin",
		Password: "testpassword123",
		Config:   "/etc/seed/seed.yaml",
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputCredentials(creds, true)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputCredentials returned error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify output is valid JSON
	var decoded setupCredentials
	if jsonErr := json.Unmarshal([]byte(output), &decoded); jsonErr != nil {
		t.Errorf("Output should be valid JSON: %v", jsonErr)
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

func TestOutputCredentialsHumanFormat(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{
		Username: "admin",
		Password: "mysecretpassword",
		Config:   "/home/user/.config/seed/seed.yaml",
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputCredentials(creds, false)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputCredentials returned error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify human-readable format contains expected content
	expectedContent := []string{
		"Username:",
		"admin",
		"Password:",
		"mysecretpassword",
		"IMPORTANT",
	}

	for _, content := range expectedContent {
		if !containsSubstring(output, content) {
			t.Errorf("Human-readable output should contain %q", content)
		}
	}
}

func TestOutputResultJSONFormat(t *testing.T) {
	t.Parallel()

	result := ValidationResult{
		Valid:    true,
		Path:     "/etc/seed/seed.yaml",
		Errors:   nil,
		Warnings: []string{"test warning"},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputResult(result, true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify output is valid JSON
	var decoded ValidationResult
	if jsonErr := json.Unmarshal([]byte(output), &decoded); jsonErr != nil {
		t.Errorf("Output should be valid JSON: %v", jsonErr)
	}

	// Verify content matches
	if decoded.Valid != result.Valid {
		t.Errorf("Valid mismatch: got %v, want %v", decoded.Valid, result.Valid)
	}
	if decoded.Path != result.Path {
		t.Errorf("Path mismatch: got %q, want %q", decoded.Path, result.Path)
	}
}

func TestOutputResultHumanFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		result          ValidationResult
		expectedContent []string
	}{
		{
			name: "valid config",
			result: ValidationResult{
				Valid: true,
				Path:  "/etc/seed/config.yaml",
			},
			expectedContent: []string{
				"Config:",
				"/etc/seed/config.yaml",
				"Status: VALID",
			},
		},
		{
			name: "invalid config with errors",
			result: ValidationResult{
				Valid:  false,
				Path:   "/etc/seed/config.yaml",
				Errors: []string{"missing field", "invalid value"},
			},
			expectedContent: []string{
				"Status: INVALID",
				"ERROR:",
			},
		},
		{
			name: "valid with warnings",
			result: ValidationResult{
				Valid:    true,
				Path:     "/home/user/.config/seed/seed.yaml",
				Warnings: []string{"interface not configured"},
			},
			expectedContent: []string{
				"Status: VALID",
				"WARNING:",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			outputResult(tc.result, false)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			for _, content := range tc.expectedContent {
				if !containsSubstring(output, content) {
					t.Errorf("Output should contain %q, got: %s", content, output)
				}
			}
		})
	}
}

func TestOutputCredentialsBannerFormat(t *testing.T) {
	t.Parallel()

	creds := setupCredentials{
		Username: "admin",
		Password: "test",
		Config:   "/path/to/config",
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = outputCredentials(creds, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Check for banner formatting (box characters)
	bannerElements := []string{
		"THE SEED",
		"CREDENTIALS",
	}

	for _, element := range bannerElements {
		if !containsSubstring(output, element) {
			t.Errorf("Banner should contain %q", element)
		}
	}
}

func TestOutputResultErrorFormatting(t *testing.T) {
	t.Parallel()

	result := ValidationResult{
		Valid: false,
		Path:  "/test/config.yaml",
		Errors: []string{
			"error one",
			"error two",
			"error three",
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputResult(result, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Each error should be prefixed with ERROR:
	for _, err := range result.Errors {
		if !containsSubstring(output, err) {
			t.Errorf("Output should contain error: %q", err)
		}
	}
}

func TestOutputResultWarningFormatting(t *testing.T) {
	t.Parallel()

	result := ValidationResult{
		Valid: true,
		Path:  "/test/config.yaml",
		Warnings: []string{
			"warning one",
			"warning two",
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputResult(result, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Each warning should be prefixed with WARNING:
	for _, warn := range result.Warnings {
		if !containsSubstring(output, warn) {
			t.Errorf("Output should contain warning: %q", warn)
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

func TestValidationResultJSONWithContent(t *testing.T) {
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

	// Non-empty slices should be present
	if !containsSubstring(output, `"errors"`) {
		t.Error("Non-empty errors should be present in JSON output")
	}
	if !containsSubstring(output, `"warnings"`) {
		t.Error("Non-empty warnings should be present in JSON output")
	}
}
