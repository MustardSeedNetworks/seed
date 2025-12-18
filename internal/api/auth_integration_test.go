// Package api provides the HTTP/WebSocket server.
// Integration tests validate JWT authentication enforcement across API endpoints.
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestEndpointAuthentication verifies that all API endpoints properly enforce JWT authentication
// This is a comprehensive test suite for Issue #452: API Endpoints Skip JWT Auth Checks.
func TestEndpointAuthentication(t *testing.T) {
	// Create a test server without auth token
	server := createTestServer(t)
	defer server.cleanup()

	// Define all API endpoints and their expected auth behavior
	tests := []struct {
		name           string
		method         string
		path           string
		shouldSkipAuth bool
		reason         string
	}{
		// Auth endpoints - should skip
		{"Login", "POST", "/api/auth/login", true, "Login endpoint needs to be public"},
		{"Logout", "POST", "/api/auth/logout", false, "Logout requires auth to know who to logout"},

		// Setup endpoints - should skip
		{"Setup Status", "GET", "/api/setup/status", true, "Initial setup status check"},
		{"Setup Complete", "POST", "/api/setup/complete", true, "First-boot setup completion"},

		// Status and configuration
		{"Status", "GET", "/api/status", false, "Server status requires auth"},
		{"Settings GET", "GET", "/api/settings", false, "Reading settings requires auth"},
		{"Settings PUT", "PUT", "/api/settings", false, "Updating settings requires auth"},

		// Network interfaces
		{"Interfaces", "GET", "/api/interfaces", false, "Listing interfaces requires auth"},
		{"Interface GET", "GET", "/api/interface", false, "Current interface requires auth"},
		{"Interface PUT", "PUT", "/api/interface", false, "Switching interface requires auth"},

		// Export and logs
		{"Export", "GET", "/api/export", false, "Exporting diagnostics requires auth"},
		{"Logs", "GET", "/api/logs", false, "Viewing logs requires auth"},

		// Network diagnostics
		{"Link", "GET", "/api/link", false, "Link status requires auth"},
		{"IPConfig", "GET", "/api/ipconfig", false, "IP configuration requires auth"},
		{"IPConfig Settings GET", "GET", "/api/ipconfig/settings", false, "IP settings require auth"},
		{"IPConfig Settings PUT", "PUT", "/api/ipconfig/settings", false, "Changing IP settings requires auth"},
		{"MTU", "PUT", "/api/network/mtu", false, "Setting MTU requires auth"},

		// Discovery
		{"Discovery", "GET", "/api/discovery", false, "LLDP/CDP discovery requires auth"},
		{"TCP Probe", "POST", "/api/discovery/probe", false, "TCP probing requires auth"},
		{"Traceroute", "POST", "/api/discovery/traceroute", false, "Traceroute requires auth"},
		{"Port Scan", "POST", "/api/discovery/portscan", false, "Port scanning requires auth"},
		{"Discovery Profile GET", "GET", "/api/discovery/profile", false, "Discovery profile requires auth"},
		{"Discovery Profile PUT", "PUT", "/api/discovery/profile", false, "Changing profile requires auth"},
		{"Discovery Service Status", "GET", "/api/discovery/service/status", false, "Service status requires auth"},
		{"Advanced Fingerprint", "POST", "/api/discovery/fingerprint", false, "Fingerprinting requires auth"},

		// DNS and Gateway
		{"DNS", "GET", "/api/dns", false, "DNS test results require auth"},
		{"Gateway", "GET", "/api/gateway", false, "Gateway status requires auth"},

		// VLAN
		{"VLAN", "GET", "/api/vlan", false, "VLAN status requires auth"},
		{"VLAN Traffic", "GET", "/api/vlan/traffic", false, "VLAN traffic requires auth"},
		{"VLAN Interface", "PUT", "/api/vlan/interface", false, "Setting VLAN requires auth"},

		// WiFi
		{"WiFi", "GET", "/api/wifi", false, "WiFi status requires auth"},
		{"WiFi Settings GET", "GET", "/api/wifi/settings", false, "WiFi settings require auth"},
		{"WiFi Settings PUT", "PUT", "/api/wifi/settings", false, "Changing WiFi settings requires auth"},

		// Cable diagnostics
		{"Cable", "GET", "/api/cable", false, "Cable diagnostics require auth"},

		// Public IP
		{"Public IP", "GET", "/api/publicip", false, "Public IP lookup requires auth"},

		// Speed tests
		{"Speedtest", "POST", "/api/speedtest", false, "Starting speedtest requires auth"},
		{"Speedtest Status", "GET", "/api/speedtest/status", false, "Speedtest status requires auth"},

		// Health checks
		{"Health Checks Settings GET", "GET", "/api/health-checks/settings", false, "Health check settings require auth"},
		{"Health Checks Settings PUT", "PUT", "/api/health-checks/settings", false, "Changing health check settings requires auth"},
		{"Run Health Checks", "GET", "/api/health-checks/run", false, "Running health checks requires auth"},

		// iperf3
		{"iperf Info", "GET", "/api/iperf/info", false, "iperf info requires auth"},
		{"iperf Client", "POST", "/api/iperf/client", false, "Running iperf client requires auth"},
		{"iperf Client Status", "GET", "/api/iperf/client/status", false, "iperf client status requires auth"},
		{"iperf Server", "POST", "/api/iperf/server", false, "Managing iperf server requires auth"},
		{"iperf Server Status", "GET", "/api/iperf/server/status", false, "iperf server status requires auth"},
		{"iperf Suggestions", "GET", "/api/iperf/suggestions", false, "iperf suggestions require auth"},

		// Network device discovery
		{"Devices", "GET", "/api/devices", false, "Discovered devices require auth"},
		{"Devices Scan", "POST", "/api/devices/scan", false, "Triggering scan requires auth"},
		{"Devices Status", "GET", "/api/devices/status", false, "Scan status requires auth"},
		{"Devices Settings GET", "GET", "/api/devices/settings", false, "Discovery settings require auth"},
		{"Devices Settings PUT", "PUT", "/api/devices/settings", false, "Changing discovery settings requires auth"},
		{"Devices Subnets GET", "GET", "/api/devices/subnets", false, "Subnet config requires auth"},
		{"Devices Subnets PUT", "PUT", "/api/devices/subnets", false, "Changing subnets requires auth"},

		// System health
		{"System Health", "GET", "/api/system/health", false, "System health requires auth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test without auth token
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			w := httptest.NewRecorder()
			server.handler.ServeHTTP(w, req)

			if tt.shouldSkipAuth {
				// Endpoints that skip auth should NOT return 401
				if w.Code == http.StatusUnauthorized {
					t.Errorf("%s %s should skip auth (%s) but returned 401", tt.method, tt.path, tt.reason)
				}
			} else {
				// Endpoints that require auth MUST return 401 without token
				if w.Code != http.StatusUnauthorized {
					t.Errorf("%s %s should require auth but returned %d (expected 401). Reason: %s",
						tt.method, tt.path, w.Code, tt.reason)
				}
			}
		})
	}
}

// TestEndpointAuthWithValidToken verifies that endpoints work correctly with valid auth tokens.
func TestEndpointAuthWithValidToken(t *testing.T) {
	server := createTestServer(t)
	defer server.cleanup()

	// Generate a valid token
	token, err := server.server.authManager.GenerateToken("testuser")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Test a few representative endpoints with valid token
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"Status with auth", "GET", "/api/status"},
		{"Link with auth", "GET", "/api/link"},
		{"Devices with auth", "GET", "/api/devices"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			server.handler.ServeHTTP(w, req)

			// Should NOT return 401 with valid token
			if w.Code == http.StatusUnauthorized {
				t.Errorf("%s %s with valid token returned 401", tt.method, tt.path)
			}
		})
	}
}

// TestEndpointAuthWithInvalidToken verifies that invalid tokens are rejected.
func TestEndpointAuthWithInvalidToken(t *testing.T) {
	server := createTestServer(t)
	defer server.cleanup()

	tests := []struct {
		name        string
		tokenValue  string
		description string
	}{
		{"Invalid token", "invalid-token-here", "Completely invalid token"},
		{"Empty token", "", "Empty authorization header"},
		{"Malformed Bearer", "NotBearer token123", "Wrong auth scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/status", http.NoBody)
			if tt.tokenValue != "" {
				req.Header.Set("Authorization", tt.tokenValue)
			}
			w := httptest.NewRecorder()
			server.handler.ServeHTTP(w, req)

			// Should return 401 for invalid tokens
			if w.Code != http.StatusUnauthorized {
				t.Errorf("Invalid token (%s) should return 401 but got %d", tt.description, w.Code)
			}
		})
	}
}

// TestEndpointAuthWithExpiredToken verifies that expired tokens are rejected.
func TestEndpointAuthWithExpiredToken(t *testing.T) {
	server := createTestServer(t)
	defer server.cleanup()

	// Create a token with 0 duration (immediately expired)
	// Note: This requires access to the auth manager's GenerateTokenWithDuration method
	// For now, we'll skip this test if the method doesn't exist
	t.Skip("Expired token test requires GenerateTokenWithDuration method - implement in auth package")
}

// TestWebSocketAuth verifies WebSocket authentication via query parameter.
func TestWebSocketAuth(t *testing.T) {
	server := createTestServer(t)
	defer server.cleanup()

	// Test without token
	t.Run("WebSocket without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ws", http.NoBody)
		// Upgrade headers for WebSocket
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "test-key")

		w := httptest.NewRecorder()
		server.handler.ServeHTTP(w, req)

		// Should return 401 without token
		if w.Code != http.StatusUnauthorized {
			t.Errorf("WebSocket without token should return 401 but got %d", w.Code)
		}
	})

	// Test with token in query (DISABLED for security fix #706)
	// Query parameter authentication is no longer supported to prevent token leakage via logs/referer
	t.Run("WebSocket with token in query (disabled #706)", func(t *testing.T) {
		token, err := server.server.authManager.GenerateToken("testuser")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		req := httptest.NewRequest("GET", "/ws?token="+token, http.NoBody)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "test-key")

		w := httptest.NewRecorder()
		server.handler.ServeHTTP(w, req)

		// Security fix #706: Query param auth is disabled, should return 401
		if w.Code != http.StatusUnauthorized {
			t.Errorf("WebSocket with query param token should return 401 (query auth disabled for security), got %d", w.Code)
		}
	})

	// Test with token in Sec-WebSocket-Protocol header (new secure method)
	t.Run("WebSocket with token in subprotocol", func(t *testing.T) {
		token, err := server.server.authManager.GenerateToken("testuser")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		req := httptest.NewRequest("GET", "/ws", http.NoBody)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "test-key")
		req.Header.Set("Sec-WebSocket-Protocol", "access_token, "+token)

		w := httptest.NewRecorder()
		server.handler.ServeHTTP(w, req)

		// Should NOT return 401 with valid token in subprotocol
		if w.Code == http.StatusUnauthorized {
			t.Errorf("WebSocket with valid token in subprotocol should not return 401")
		}
	})
}

// TestStaticFilesNoAuth verifies that static files don't require authentication.
func TestStaticFilesNoAuth(t *testing.T) {
	server := createTestServer(t)
	defer server.cleanup()

	// Test various static file paths
	staticPaths := []string{
		"/",
		"/index.html",
		"/assets/style.css",
		"/favicon.ico",
	}

	for _, path := range staticPaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, http.NoBody)
			w := httptest.NewRecorder()
			server.handler.ServeHTTP(w, req)

			// Static files should NOT return 401
			if w.Code == http.StatusUnauthorized {
				t.Errorf("Static file %s should not require auth but got 401", path)
			}
		})
	}
}

// Helper to create a test server.
type testServer struct {
	server  *Server
	handler http.Handler
}

func (ts *testServer) cleanup() {
	// Cleanup resources if needed
	// WebSocket hub doesn't require explicit cleanup in tests
}

func createTestServer(_ *testing.T) *testServer {
	server := NewTestServer()
	handler := server.GetAuthenticatedHandler()

	return &testServer{
		server:  server,
		handler: handler,
	}
}
