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
				"X-Api-Key":      []string{"secret"},
				"Content-Length": []string{"1234"},
			},
			expected: map[string]string{
				"X-Api-Key":      "[REDACTED]",
				"Content-Length": "1234",
			},
		},
		{
			name: "safe headers unchanged",
			headers: http.Header{
				"Accept":       []string{"*/*"},
				"Content-Type": []string{"text/plain"},
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
		name        string
		err         error
		context     string
		contains    string
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
			if tt.contains != "" && !testContains(resultStr, tt.contains) {
				t.Errorf("SafeError() = %q, should contain %q", resultStr, tt.contains)
			}
			if tt.notContains != "" && testContains(resultStr, tt.notContains) {
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

func testContains(s, substr string) bool {
	return substr != "" && len(s) >= len(substr) &&
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

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    http.Header
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    http.Header{"X-Forwarded-For": []string{"192.168.1.100"}},
			remoteAddr: "10.0.0.1:8080",
			expected:   "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			headers:    http.Header{"X-Forwarded-For": []string{"192.168.1.100, 10.0.0.50"}},
			remoteAddr: "10.0.0.1:8080",
			expected:   "192.168.1.100",
		},
		{
			name:       "X-Real-IP",
			headers:    http.Header{"X-Real-IP": []string{"172.16.0.5"}},
			remoteAddr: "10.0.0.1:8080",
			expected:   "172.16.0.5",
		},
		{
			name:       "No proxy headers - use RemoteAddr",
			headers:    http.Header{},
			remoteAddr: "192.168.1.50:12345",
			expected:   "192.168.1.50",
		},
		{
			name:       "RemoteAddr without port",
			headers:    http.Header{},
			remoteAddr: "192.168.1.50",
			expected:   "192.168.1.50",
		},
		{
			name: "X-Forwarded-For takes precedence over X-Real-IP",
			headers: http.Header{
				"X-Forwarded-For": []string{"192.168.1.100"},
				"X-Real-IP":       []string{"172.16.0.5"},
			},
			remoteAddr: "10.0.0.1:8080",
			expected:   "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", http.NoBody)
			// Copy headers to request (don't replace the Header map)
			for k, v := range tt.headers {
				for _, val := range v {
					req.Header.Add(k, val)
				}
			}
			req.RemoteAddr = tt.remoteAddr

			result := GetClientIP(req)
			if result != tt.expected {
				t.Errorf("GetClientIP() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLogf(t *testing.T) {
	// Test that Logf doesn't panic with various argument types
	t.Run("string argument", func(_ *testing.T) {
		Logf("test message: %s", "password=secret")
	})

	t.Run("http.Header argument", func(_ *testing.T) {
		h := http.Header{"Authorization": []string{"Bearer token"}}
		Logf("headers: %v", h)
	})

	t.Run("map argument", func(_ *testing.T) {
		m := map[string]interface{}{"password": "secret", "user": "admin"}
		Logf("data: %v", m)
	})

	t.Run("int argument", func(_ *testing.T) {
		Logf("count: %d", 42)
	})

	t.Run("mixed arguments", func(_ *testing.T) {
		Logf("user=%s count=%d", "admin", 10)
	})
}

func TestLogRequest(_ *testing.T) {
	// Just verify it doesn't panic
	req, _ := http.NewRequest("POST", "/api/login", http.NoBody)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.100:54321"

	// This should log but not panic
	LogRequest(req, "test request")
}

func TestRedactMapWithNonStringValues(t *testing.T) {
	input := map[string]interface{}{
		"count":    42,
		"enabled":  true,
		"password": "secret",
		"data":     []int{1, 2, 3},
		"nested":   map[string]string{"key": "value"},
	}

	result := RedactMap(input)

	// Non-string values should be preserved (except sensitive keys)
	if result["count"] != 42 {
		t.Errorf("RedactMap()[count] = %v, want 42", result["count"])
	}
	if result["enabled"] != true {
		t.Errorf("RedactMap()[enabled] = %v, want true", result["enabled"])
	}
	if result["password"] != "[REDACTED]" {
		t.Errorf("RedactMap()[password] = %v, want [REDACTED]", result["password"])
	}
	// Non-string types are preserved as-is
	if _, ok := result["data"].([]int); !ok {
		t.Errorf("RedactMap()[data] should be []int")
	}
}

func TestSensitivePatterns(t *testing.T) {
	// Test various sensitive patterns
	tests := []struct {
		input        string
		shouldRedact bool
	}{
		{"password=hunter2", true},
		{"Password:secret", true},
		{"PASSWD=abc123", true},
		{"pwd=test", true},
		{"token=eyJhbGci", true},
		{"auth=bearer_token", true},
		{"api-key=abc123", true},
		{"API_KEY=xyz789", true},
		{"secret=classified", true},
		{"Bearer eyJhbGciOiJIUzI1NiJ9", true},
		{"Basic dXNlcjpwYXNz", true},
		{"username=admin", false},
		{"email=test@example.com", false},
		{"status=active", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := RedactString(tt.input)
			hasRedacted := result != tt.input
			if hasRedacted != tt.shouldRedact {
				t.Errorf("RedactString(%q) redacted=%v, want %v (result=%q)",
					tt.input, hasRedacted, tt.shouldRedact, result)
			}
		})
	}
}

func TestSensitiveHeaders(t *testing.T) {
	// All sensitive headers should be redacted
	sensitiveHeaderNames := []string{
		"Authorization",
		"X-Api-Key",
		"X-Auth-Token",
		"Cookie",
		"Set-Cookie",
		"X-CSRF-Token",
		"X-XSRF-Token",
		"Proxy-Authorization",
	}

	for _, headerName := range sensitiveHeaderNames {
		t.Run(headerName, func(t *testing.T) {
			headers := http.Header{headerName: []string{"secret-value"}}
			result := RedactHeaders(headers)
			if result[headerName] != "[REDACTED]" {
				t.Errorf("RedactHeaders()[%q] = %q, want [REDACTED]", headerName, result[headerName])
			}
		})
	}
}

func TestEmptyInputs(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		result := RedactString("")
		if result != "" {
			t.Errorf("RedactString(\"\") = %q, want \"\"", result)
		}
	})

	t.Run("empty headers", func(t *testing.T) {
		result := RedactHeaders(http.Header{})
		if len(result) != 0 {
			t.Errorf("RedactHeaders(empty) = %v, want empty map", result)
		}
	})

	t.Run("empty map", func(t *testing.T) {
		result := RedactMap(map[string]interface{}{})
		if len(result) != 0 {
			t.Errorf("RedactMap(empty) = %v, want empty map", result)
		}
	})
}
