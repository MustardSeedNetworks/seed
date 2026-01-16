package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// Engine tests verify handlers return proper errors when engine is unavailable.
// The test server doesn't initialize the discovery engine for performance reasons
// (OUI database loading is slow). These tests verify the error handling paths.

func TestHandleEngineDiscovery(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET returns service unavailable when engine nil",
			method:         http.MethodGet,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "POST not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/discovery/engine", nil)
			rec := httptest.NewRecorder()

			server.HandleEngineDiscovery(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify proper error response format for unavailable
			if tt.expectedStatus == http.StatusServiceUnavailable {
				var resp map[string]any
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if resp["error"] == nil || resp["error"] == "" {
					t.Error("expected error message in response")
				}
			}
		})
	}
}

func TestHandleEngineStats(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/engine/stats", nil)
	rec := httptest.NewRecorder()

	server.HandleEngineStats(rec, req)

	// Engine is nil in test server, should return 503
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestHandleEngineCapabilities(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/engine/capabilities", nil)
	rec := httptest.NewRecorder()

	server.HandleEngineCapabilities(rec, req)

	// Engine is nil in test server, should return 503
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestHandleEngineScan(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "POST returns service unavailable when engine nil",
			method:         http.MethodPost,
			body:           "",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "GET not allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "POST with options returns service unavailable",
			method:         http.MethodPost,
			body:           `{"scanType":"quick","includeWired":true}`,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/api/v1/discovery/engine/scan", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/api/v1/discovery/engine/scan", nil)
			}
			rec := httptest.NewRecorder()

			server.HandleEngineScan(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandleEngineQuickScan(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/engine/quick", nil)
	rec := httptest.NewRecorder()

	server.HandleEngineQuickScan(rec, req)

	// Engine is nil, expect 503
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d: %s", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}
}

func TestHandleEngineFullScan(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/engine/full", nil)
	rec := httptest.NewRecorder()

	server.HandleEngineFullScan(rec, req)

	// Engine is nil, expect 503
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d: %s", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}
}

func TestHandleEngineDevice(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	tests := []struct {
		name           string
		mac            string
		expectedStatus int
	}{
		{
			name:           "device lookup returns unavailable when engine nil",
			mac:            "AA:BB:CC:DD:EE:FF",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "empty mac returns unavailable (engine check first)",
			mac:            "",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := api.ExportAPIVersionPrefix + "/discovery/engine/device/" + tt.mac
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			server.HandleEngineDevice(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandleEngineMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	// Test that non-GET methods return 405 for read endpoints
	readEndpoints := []string{
		"/api/v1/discovery/engine",
		"/api/v1/discovery/engine/stats",
		"/api/v1/discovery/engine/capabilities",
	}

	for _, endpoint := range readEndpoints {
		t.Run("POST "+endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, endpoint, nil)
			rec := httptest.NewRecorder()

			switch endpoint {
			case "/api/v1/discovery/engine":
				server.HandleEngineDiscovery(rec, req)
			case "/api/v1/discovery/engine/stats":
				server.HandleEngineStats(rec, req)
			case "/api/v1/discovery/engine/capabilities":
				server.HandleEngineCapabilities(rec, req)
			}

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}

	// Test that non-POST methods return 405 for scan endpoints
	scanEndpoints := []string{
		"/api/v1/discovery/engine/scan",
		"/api/v1/discovery/engine/quick",
		"/api/v1/discovery/engine/full",
	}

	for _, endpoint := range scanEndpoints {
		t.Run("GET "+endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			rec := httptest.NewRecorder()

			switch endpoint {
			case "/api/v1/discovery/engine/scan":
				server.HandleEngineScan(rec, req)
			case "/api/v1/discovery/engine/quick":
				server.HandleEngineQuickScan(rec, req)
			case "/api/v1/discovery/engine/full":
				server.HandleEngineFullScan(rec, req)
			}

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}
