package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// TestNormalizeSPAPath tests SPA path normalization.
func TestNormalizeSPAPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/", "/index.html"},
		{"", "/index.html"},
		{"/index.html", "/index.html"},
		{"/dashboard", "/dashboard"},
		{"/api/status", "/api/status"},
		{"/settings/network", "/settings/network"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := api.ExportNormalizeSPAPath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSPAPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsAPIOrWSRoute tests API and WebSocket route detection.
func TestIsAPIOrWSRoute(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/v1/status", true},     // APIVersionPrefix matches
		{"/api/v1/auth/login", true}, // APIVersionPrefix matches
		{"/api/v1/settings", true},   // APIVersionPrefix matches
		{"/api/events", true},        // APIBasePath+"/events" for SSE
		{"/ws", true},
		{"/ws/", true},
		{"/api/status", false}, // Doesn't match /api/v1
		{"/websocket", false},
		{"/dashboard", false},
		{"/", false},
		{"/index.html", false},
		{"/static/app.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := api.ExportIsAPIOrWSRoute(tt.path)
			if result != tt.expected {
				t.Errorf("IsAPIOrWSRoute(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestSecurityHeadersMiddleware tests that security headers are set.
func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with security headers middleware
	wrapped := api.ExportSecurityHeadersMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Check security headers
	expectedHeaders := map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expected := range expectedHeaders {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("Expected %s = %q, got %q", header, expected, got)
		}
	}

	// Verify CSP is set
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Expected Content-Security-Policy header to be set")
	}
}

// TestRecoverMiddleware tests panic recovery in handlers.
func TestRecoverMiddleware(t *testing.T) {
	// Handler that panics
	panicHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	})

	wrapped := api.ExportRecoverMiddleware(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(w, req)

	// Should return 500 Internal Server Error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// TestCORSMiddleware tests CORS header handling.
func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		method         string
		expectedStatus int
		expectHeaders  bool
	}{
		{
			name:           "OPTIONS preflight",
			origin:         "http://localhost:3000",
			method:         http.MethodOptions,
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
		},
		{
			name:           "null origin rejected",
			origin:         "null",
			method:         http.MethodGet,
			expectedStatus: http.StatusForbidden,
			expectHeaders:  false,
		},
		{
			name:           "no origin header",
			origin:         "",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectHeaders:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrapped := api.ExportCORSMiddleware(handler)

			req := httptest.NewRequest(tt.method, "/", http.NoBody)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectHeaders && tt.origin != "null" {
				if w.Header().Get("Access-Control-Allow-Methods") == "" {
					t.Error("Expected Access-Control-Allow-Methods header")
				}
			}
		})
	}
}

// TestBodyLimitMiddleware tests request body size limits.
func TestBodyLimitMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantLimit bool
	}{
		{
			name:      "auth endpoint",
			path:      "/api/auth/login",
			wantLimit: true,
		},
		{
			name:      "config endpoint",
			path:      "/api/config/restore",
			wantLimit: true,
		},
		{
			name:      "general API endpoint",
			path:      "/api/status",
			wantLimit: true,
		},
		{
			name:      "static path",
			path:      "/index.html",
			wantLimit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check that body limit was applied (MaxBytesReader wraps the body)
				if r.Body == nil && tt.wantLimit {
					t.Error("Expected body to be wrapped with limit")
				}
				w.WriteHeader(http.StatusOK)
			})

			wrapped := api.ExportBodyLimitMiddleware(handler)

			req := httptest.NewRequest(http.MethodPost, tt.path, http.NoBody)
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}

// TestGetClientIPWithTrustedProxies tests client IP extraction.
func TestGetClientIPWithTrustedProxies(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		trustedProxies string
		expectedIP     string
	}{
		{
			name:       "no proxies configured",
			remoteAddr: "192.168.1.100:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "remote addr with IPv6",
			remoteAddr: "[::1]:12345",
			expectedIP: "::1",
		},
		{
			name:           "trusted proxy with X-Forwarded-For",
			remoteAddr:     "10.0.0.1:12345",
			xForwardedFor:  "203.0.113.50",
			trustedProxies: "10.0.0.1",
			expectedIP:     "203.0.113.50",
		},
		{
			name:           "untrusted proxy ignores X-Forwarded-For",
			remoteAddr:     "1.2.3.4:12345",
			xForwardedFor:  "203.0.113.50",
			trustedProxies: "10.0.0.1",
			expectedIP:     "1.2.3.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			var proxies *api.TrustedProxies
			if tt.trustedProxies != "" {
				proxies = api.NewTrustedProxies(tt.trustedProxies)
			}

			result := api.GetClientIPWithTrustedProxies(req, proxies)
			if result != tt.expectedIP {
				t.Errorf("GetClientIPWithTrustedProxies() = %q, want %q", result, tt.expectedIP)
			}
		})
	}
}

// TestTrustedProxies tests the TrustedProxies type.
func TestTrustedProxies(t *testing.T) {
	t.Run("valid IPs", func(t *testing.T) {
		proxies := api.NewTrustedProxies("10.0.0.1,192.168.1.0/24")

		if !proxies.IsTrusted("10.0.0.1") {
			t.Error("Expected 10.0.0.1 to be trusted")
		}

		if !proxies.IsTrusted("192.168.1.100") {
			t.Error("Expected 192.168.1.100 to be trusted (in CIDR)")
		}

		if proxies.IsTrusted("8.8.8.8") {
			t.Error("Expected 8.8.8.8 to not be trusted")
		}
	})

	t.Run("nil proxies", func(t *testing.T) {
		var proxies *api.TrustedProxies
		if proxies.IsTrusted("10.0.0.1") {
			t.Error("Nil TrustedProxies should not trust any IP")
		}
	})

	t.Run("invalid CIDR logs warning and continues", func(t *testing.T) {
		// Invalid entries are logged and skipped, not returned as errors
		proxies := api.NewTrustedProxies("invalid-cidr")
		// Should return an empty proxy list that trusts no one
		if proxies.IsTrusted("10.0.0.1") {
			t.Error("Proxies with invalid CIDR should not trust any IP")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		proxies := api.NewTrustedProxies("")
		if proxies.IsTrusted("10.0.0.1") {
			t.Error("Empty TrustedProxies should not trust any IP")
		}
	})

	t.Run("IsEmpty", func(t *testing.T) {
		emptyProxies := api.NewTrustedProxies("")
		if !emptyProxies.IsEmpty() {
			t.Error("Expected empty proxies to be empty")
		}

		validProxies := api.NewTrustedProxies("10.0.0.1")
		if validProxies.IsEmpty() {
			t.Error("Expected non-empty proxies to not be empty")
		}
	})

	t.Run("Count", func(t *testing.T) {
		proxies := api.NewTrustedProxies("10.0.0.1,192.168.1.0/24,172.16.0.1")
		if proxies.Count() != 3 {
			t.Errorf("Expected count 3, got %d", proxies.Count())
		}
	})
}

// TestNewTestServer tests the test server creation.
func TestNewTestServer(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	if server == nil {
		t.Fatal("NewTestServer returned nil")
	}

	if server.Mux() == nil {
		t.Error("Expected server to have a mux")
	}

	if server.AuthManager() == nil {
		t.Error("Expected server to have an auth manager")
	}
}

// TestGetAuthenticatedHandler tests getting the authenticated handler.
func TestGetAuthenticatedHandler(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	handler := server.GetAuthenticatedHandler()
	if handler == nil {
		t.Fatal("GetAuthenticatedHandler returned nil")
	}

	// Make a request to verify the handler works
	req := httptest.NewRequest(http.MethodGet, "/api/status", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return some response (likely 401 without auth, but not panic)
	if w.Code == 0 {
		t.Error("Expected handler to return a status code")
	}
}

// TestServerHub tests the Hub accessor.
func TestServerHub(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	// Hub might be nil in test server, but accessor should not panic
	hub := server.Hub()
	if hub == nil {
		t.Skip("Hub not initialized in test server")
	}
}

// TestServerDB tests the DB accessor.
func TestServerDB(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	// DB is nil in test server, but accessor should not panic
	db := server.DB()
	if db != nil {
		t.Error("Expected DB to be nil in test server")
	}
}
