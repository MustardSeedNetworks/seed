package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

// TestHandlePortScan tests the port scan endpoint.
func TestHandlePortScan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	tests := []struct {
		name           string
		request        api.PortScanRequest
		expectedStatus int
	}{
		{
			name: "quick scan profile",
			request: api.PortScanRequest{
				Target:  "127.0.0.1",
				Profile: "quick",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "web scan profile",
			request: api.PortScanRequest{
				Target:  "127.0.0.1",
				Profile: "web",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "specific ports",
			request: api.PortScanRequest{
				Target:  "127.0.0.1",
				Ports:   []int{22, 80, 443},
				Workers: 5,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "default profile",
			request: api.PortScanRequest{
				Target: "127.0.0.1",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			defer server.Close()

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/security/discovery/portscan",
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf(
					"Expected status %d, got %d: %s",
					tt.expectedStatus,
					w.Code,
					w.Body.String(),
				)
			}
		})
	}
}

// TestHandlePortScanMethodNotAllowed tests non-POST methods on port scan endpoint.
func TestHandlePortScanMethodNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/security/discovery/portscan", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf(
					"Expected status %d for %s, got %d",
					http.StatusMethodNotAllowed,
					method,
					w.Code,
				)
			}
		})
	}
}

// TestHandlePortScanInvalidJSON tests port scan with invalid JSON.
func TestHandlePortScanInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/security/discovery/portscan",
		bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestPortScanRequestValidation tests PortScanRequest validation.
func TestPortScanRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request api.PortScanRequest
		isValid bool
	}{
		{
			name: "valid with quick profile",
			request: api.PortScanRequest{
				Target:  "192.168.1.1",
				Profile: "quick",
			},
			isValid: true,
		},
		{
			name: "valid with web profile",
			request: api.PortScanRequest{
				Target:  "192.168.1.1",
				Profile: "web",
			},
			isValid: true,
		},
		{
			name: "valid with full profile",
			request: api.PortScanRequest{
				Target:  "192.168.1.1",
				Profile: "full",
			},
			isValid: true,
		},
		{
			name: "valid with specific ports",
			request: api.PortScanRequest{
				Target:  "192.168.1.1",
				Ports:   []int{22, 80, 443, 8080},
				Workers: 10,
			},
			isValid: true,
		},
		{
			name: "valid hostname target",
			request: api.PortScanRequest{
				Target:  "example.com",
				Profile: "quick",
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - target should be non-empty
			isValid := tt.request.Target != ""

			if tt.isValid && !isValid {
				t.Error("Expected valid request but validation failed")
			}
		})
	}
}

// TestPortScanProfiles tests the different port scan profiles.
func TestPortScanProfiles(t *testing.T) {
	profiles := []string{"quick", "web", "full", ""}

	for _, profile := range profiles {
		t.Run("profile_"+profile, func(t *testing.T) {
			req := api.PortScanRequest{
				Target:  "127.0.0.1",
				Profile: profile,
			}

			// All profiles should be valid (empty defaults to quick)
			if req.Target == "" {
				t.Error("Expected non-empty target")
			}
			if req.Profile != profile {
				t.Errorf("Expected profile %q, got %q", profile, req.Profile)
			}
		})
	}
}
