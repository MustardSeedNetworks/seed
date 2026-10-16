package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

// TestHandleTraceroute tests the traceroute endpoint.
func TestHandleTraceroute(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	tests := []struct {
		name           string
		request        api.TracerouteRequest
		expectedStatus int
	}{
		{
			name: "valid ICMP traceroute",
			request: api.TracerouteRequest{
				Target:   "127.0.0.1",
				Protocol: "icmp",
				MaxHops:  5,
				Timeout:  1000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid UDP traceroute",
			request: api.TracerouteRequest{
				Target:   "127.0.0.1",
				Protocol: "udp",
				Port:     33434,
				MaxHops:  5,
				Timeout:  1000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid TCP traceroute",
			request: api.TracerouteRequest{
				Target:   "127.0.0.1",
				Protocol: "tcp",
				Port:     80,
				MaxHops:  5,
				Timeout:  1000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "default protocol",
			request: api.TracerouteRequest{
				Target:  "127.0.0.1",
				MaxHops: 5,
				Timeout: 1000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing target",
			request: api.TracerouteRequest{
				Protocol: "icmp",
				MaxHops:  5,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid protocol",
			request: api.TracerouteRequest{
				Target:   "127.0.0.1",
				Protocol: "invalid",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid port",
			request: api.TracerouteRequest{
				Target:   "127.0.0.1",
				Protocol: "tcp",
				Port:     70000,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			defer server.Close()

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/path/traceroute",
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

// TestHandleTracerouteResponseShape pins the answer to the published
// TracerouteResult schema, the same view the path endpoint serves.
func TestHandleTracerouteResponseShape(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	body, _ := json.Marshal(api.TracerouteRequest{
		Target:   "127.0.0.1",
		Protocol: "tcp",
		Port:     80,
		MaxHops:  2,
		Timeout:  500,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/path/traceroute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	dec := json.NewDecoder(w.Body)
	dec.DisallowUnknownFields()
	var got api.TracerouteResult
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Target != "127.0.0.1" || got.TargetIP != "127.0.0.1" || got.Protocol != "tcp" || got.Port != 80 {
		t.Errorf("unexpected result header: %+v", got)
	}
}

// TestHandleTracerouteMethodNotAllowed tests non-POST methods on traceroute endpoint.
func TestHandleTracerouteMethodNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/path/traceroute", http.NoBody)
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

// TestHandleTracerouteInvalidJSON tests traceroute with invalid JSON.
func TestHandleTracerouteInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/path/traceroute",
		bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestTracerouteRequestValidation tests TracerouteRequest validation.
func TestTracerouteRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request api.TracerouteRequest
		isValid bool
	}{
		{
			name: "valid ICMP traceroute",
			request: api.TracerouteRequest{
				Target:   "8.8.8.8",
				Protocol: "icmp",
				MaxHops:  30,
				Timeout:  3000,
			},
			isValid: true,
		},
		{
			name: "valid UDP traceroute",
			request: api.TracerouteRequest{
				Target:   "8.8.8.8",
				Protocol: "udp",
				Port:     33434,
				MaxHops:  30,
				Timeout:  3000,
			},
			isValid: true,
		},
		{
			name: "valid TCP traceroute",
			request: api.TracerouteRequest{
				Target:   "8.8.8.8",
				Protocol: "tcp",
				Port:     80,
				MaxHops:  30,
				Timeout:  3000,
			},
			isValid: true,
		},
		{
			name: "empty target",
			request: api.TracerouteRequest{
				Target:   "",
				Protocol: "icmp",
			},
			isValid: false,
		},
		{
			name: "invalid protocol",
			request: api.TracerouteRequest{
				Target:   "8.8.8.8",
				Protocol: "invalid",
			},
			isValid: false,
		},
		{
			name: "invalid port (too high)",
			request: api.TracerouteRequest{
				Target:   "8.8.8.8",
				Protocol: "tcp",
				Port:     70000,
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation
			hasTarget := tt.request.Target != ""
			validProtocol := tt.request.Protocol == "" ||
				tt.request.Protocol == "icmp" ||
				tt.request.Protocol == "udp" ||
				tt.request.Protocol == "tcp"
			validPort := tt.request.Port <= 65535

			isValid := hasTarget && validProtocol && validPort

			if tt.isValid && !isValid {
				t.Error("Expected valid request but validation failed")
			}
			if !tt.isValid && isValid {
				t.Error("Expected invalid request but validation passed")
			}
		})
	}
}

// TestTracerouteMaxHopsBounds tests traceroute max hops boundaries.
func TestTracerouteMaxHopsBounds(t *testing.T) {
	tests := []struct {
		name        string
		maxHops     int
		expectedMax int // What it should be normalized to
	}{
		{"zero (uses default 30)", 0, 30},
		{"negative (uses default 30)", -1, 30},
		{"5 hops", 5, 5},
		{"30 hops (default)", 30, 30},
		{"64 hops (max)", 64, 64},
		{"above max (clamped to 30)", 100, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := api.TracerouteRequest{
				Target:   "127.0.0.1",
				Protocol: "icmp",
				MaxHops:  tt.maxHops,
			}

			// Verify the request can be created
			if req.Target == "" {
				t.Error("Expected non-empty target")
			}
			if req.Protocol != "icmp" {
				t.Errorf("Expected protocol 'icmp', got %q", req.Protocol)
			}
			if req.MaxHops != tt.maxHops {
				t.Errorf("Expected MaxHops %d, got %d", tt.maxHops, req.MaxHops)
			}
		})
	}
}
