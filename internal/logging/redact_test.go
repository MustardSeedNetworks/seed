package logging

import (
	"net/http"
	"testing"
)

func TestRedactString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password in query",
			input:    "user login failed with password=secret123",
			expected: "user login failed with [REDACTED]",
		},
		{
			name:     "token in header format",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Authorization: [REDACTED]",
		},
		{
			name:     "api key",
			input:    "api_key=abc123def456",
			expected: "[REDACTED]",
		},
		{
			name:     "basic auth",
			input:    "Basic dXNlcjpwYXNzd29yZA==",
			expected: "[REDACTED]",
		},
		{
			name:     "safe content unchanged",
			input:    "user logged in successfully",
			expected: "user logged in successfully",
		},
		{
			name:     "password with colon",
			input:    "password: hunter2",
			expected: "[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactString(tt.input)
			if result != tt.expected {
				t.Errorf("RedactString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRedactHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  http.Header
		expected map[string]string
	}{
		{
			name: "authorization header redacted",
			headers: http.Header{
				"Authorization": []string{"Bearer token123"},
				"Content-Type":  []string{"application/json"},
			},
			expected: map[string]string{
				"Authorization": "[REDACTED]",
				"Content-Type":  "application/json",
			},
		},
		{
			name: "cookie header redacted",
			headers: http.Header{
				"Cookie":     []string{"session=abc123"},
				"User-Agent": []string{"Mozilla/5.0"},
			},
			expected: map[string]string{
				"Cookie":     "[REDACTED]",
				"User-Agent": "Mozilla/5.0",
			},
		},
		{
			name: "api key header redacted",
			headers: http.Header{
				"X-Api-Key":   []string{"secret"},
				"Content-Length": []string{"1234"},
			},
			expected: map[string]string{
				"X-Api-Key":   "[REDACTED]",
				"Content-Length": "1234",
			},
		},
		{
			name: "safe headers unchanged",
			headers: http.Header{
				"Accept":        []string{"*/*"},
				"Content-Type":  []string{"text/plain"},
			},
			expected: map[string]string{
				"Accept":       "*/*",
				"Content-Type": "text/plain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactHeaders(tt.headers)
			for key, expectedVal := range tt.expected {
				if result[key] != expectedVal {
					t.Errorf("RedactHeaders()[%q] = %q, want %q", key, result[key], expectedVal)
				}
			}
		})
	}
}

func TestRedactMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		checkKey string
		expected interface{}
	}{
		{
			name: "password field redacted",
			input: map[string]interface{}{
				"username": "admin",
				"password": "secret123",
			},
			checkKey: "password",
			expected: "[REDACTED]",
		},
		{
			name: "token field redacted",
			input: map[string]interface{}{
				"user":  "alice",
				"token": "eyJhbGci...",
			},
			checkKey: "token",
			expected: "[REDACTED]",
		},
		{
			name: "api_key field redacted",
			input: map[string]interface{}{
				"api_key": "abc123",
				"status":  "active",
			},
			checkKey: "api_key",
			expected: "[REDACTED]",
		},
		{
			name: "safe fields unchanged",
			input: map[string]interface{}{
				"username": "admin",
				"status":   "active",
			},
			checkKey: "username",
			expected: "admin",
		},
		{
			name: "string with password pattern redacted",
			input: map[string]interface{}{
				"error": "login failed with password=secret",
			},
			checkKey: "error",
			expected: "login failed with [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactMap(tt.input)
			if result[tt.checkKey] != tt.expected {
				t.Errorf("RedactMap()[%q] = %v, want %v", tt.checkKey, result[tt.checkKey], tt.expected)
			}
		})
	}
}

func TestSafeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		context  string
		contains string
		notContains string
	}{
		{
			name:        "nil error",
			err:         nil,
			context:     "test",
			contains:    "",
			notContains: "",
		},
		{
			name:        "error with password redacted",
			err:         &testError{msg: "authentication failed: password=secret123"},
			context:     "login",
			contains:    "[REDACTED]",
			notContains: "secret123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeError(tt.err, tt.context)
			if tt.err == nil {
				if result != nil {
					t.Errorf("SafeError(nil) = %v, want nil", result)
				}
				return
			}

			resultStr := result.Error()
			if tt.contains != "" && !contains(resultStr, tt.contains) {
				t.Errorf("SafeError() = %q, should contain %q", resultStr, tt.contains)
			}
			if tt.notContains != "" && contains(resultStr, tt.notContains) {
				t.Errorf("SafeError() = %q, should NOT contain %q", resultStr, tt.notContains)
			}
		})
	}
}

// Helper types and functions for tests

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
