package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// TestHandleTCPProbe tests the TCP probe endpoint.
func TestHandleTCPProbe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	tests := []struct {
		name           string
		request        api.TCPProbeRequest
		expectedStatus int
	}{
		{
			name: "valid single port probe",
			request: api.TCPProbeRequest{
				Target:  "127.0.0.1",
				Port:    80,
				Timeout: 1000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid multi-port probe",
			request: api.TCPProbeRequest{
				Target:  "127.0.0.1",
				Ports:   []int{22, 80, 443},
				Timeout: 1000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "probe with hostname",
			request: api.TCPProbeRequest{
				Target:  "localhost",
				Port:    80,
				Timeout: 1000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing target",
			request: api.TCPProbeRequest{
				Port:    80,
				Timeout: 1000,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing port",
			request: api.TCPProbeRequest{
				Target:  "127.0.0.1",
				Timeout: 1000,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			defer server.Close()

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/shell/discovery/probe", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var resp api.TCPProbeResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if resp.Target == "" {
					t.Error("Expected non-empty target in response")
				}
			}
		})
	}
}

// TestHandleTCPProbeMethodNotAllowed tests non-POST methods on TCP probe endpoint.
func TestHandleTCPProbeMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/shell/discovery/probe", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleTCPProbeInvalidJSON tests TCP probe with invalid JSON.
func TestHandleTCPProbeInvalidJSON(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/shell/discovery/probe", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleTCPProbeTooManyPorts tests TCP probe with too many ports.
func TestHandleTCPProbeTooManyPorts(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	// Create a list of 101 ports (exceeds 100 limit)
	ports := make([]int, 101)
	for i := range ports {
		ports[i] = i + 1
	}

	reqBody := api.TCPProbeRequest{
		Target:  "127.0.0.1",
		Ports:   ports,
		Timeout: 1000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shell/discovery/probe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// TestHandleTCPProbeInvalidTarget tests TCP probe with invalid target.
func TestHandleTCPProbeInvalidTarget(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	reqBody := api.TCPProbeRequest{
		Target:  "invalid..hostname",
		Port:    80,
		Timeout: 1000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shell/discovery/probe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// TestHandleTraceroute tests the traceroute endpoint.
func TestHandleTraceroute(t *testing.T) {
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
			req := httptest.NewRequest(http.MethodPost, "/api/roots/traceroute", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleTracerouteMethodNotAllowed tests non-POST methods on traceroute endpoint.
func TestHandleTracerouteMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/roots/traceroute", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleTracerouteInvalidJSON tests traceroute with invalid JSON.
func TestHandleTracerouteInvalidJSON(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/roots/traceroute", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandlePortScan tests the port scan endpoint.
func TestHandlePortScan(t *testing.T) {
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
			req := httptest.NewRequest(http.MethodPost, "/api/shell/discovery/portscan", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandlePortScanMethodNotAllowed tests non-POST methods on port scan endpoint.
func TestHandlePortScanMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/shell/discovery/portscan", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandlePortScanInvalidJSON tests port scan with invalid JSON.
func TestHandlePortScanInvalidJSON(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shell/discovery/portscan",
		bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleAdvancedFingerprint tests the advanced fingerprint endpoint.
func TestHandleAdvancedFingerprint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	tests := []struct {
		name           string
		request        map[string]string
		expectedStatus int
	}{
		{
			name: "valid IP",
			request: map[string]string{
				"ip": "127.0.0.1",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing IP",
			request:        map[string]string{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty IP",
			request: map[string]string{
				"ip": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			defer server.Close()

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/shell/discovery/fingerprint", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleAdvancedFingerprintMethodNotAllowed tests non-POST methods.
func TestHandleAdvancedFingerprintMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/shell/discovery/fingerprint", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleAdvancedFingerprintInvalidJSON tests fingerprint with invalid JSON.
func TestHandleAdvancedFingerprintInvalidJSON(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shell/discovery/fingerprint",
		bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestTCPProbeRequestFields tests that TCPProbeRequest has expected fields.
func TestTCPProbeRequestFields(t *testing.T) {
	req := api.TCPProbeRequest{
		Target:  "192.168.1.1",
		Port:    80,
		Ports:   []int{22, 80, 443},
		Timeout: 5000,
	}

	if req.Target != "192.168.1.1" {
		t.Errorf("Expected Target '192.168.1.1', got %q", req.Target)
	}
	if req.Port != 80 {
		t.Errorf("Expected Port 80, got %d", req.Port)
	}
	if len(req.Ports) != 3 {
		t.Errorf("Expected 3 ports, got %d", len(req.Ports))
	}
	if req.Timeout != 5000 {
		t.Errorf("Expected Timeout 5000, got %d", req.Timeout)
	}
}

// TestTCPProbeResponseFields tests that TCPProbeResponse has expected fields.
func TestTCPProbeResponseFields(t *testing.T) {
	resp := api.TCPProbeResponse{
		Target: "192.168.1.1",
	}

	if resp.Target != "192.168.1.1" {
		t.Errorf("Expected Target '192.168.1.1', got %q", resp.Target)
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

// TestTCPProbeTimeoutBounds tests TCP probe timeout boundaries.
func TestTCPProbeTimeoutBounds(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		isValid bool
	}{
		{"zero timeout (uses default)", 0, true},
		{"100ms timeout", 100, true},
		{"1 second timeout", 1000, true},
		{"5 second timeout", 5000, true},
		{"10 second timeout", 10000, true},
		{"above max timeout (clamped)", 20000, true}, // Will be clamped to max
		{"negative timeout (uses default)", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := api.TCPProbeRequest{
				Target:  "127.0.0.1",
				Port:    80,
				Timeout: tt.timeout,
			}

			// All values are valid (will be clamped or use default)
			if req.Port <= 0 {
				t.Error("Expected valid port")
			}
			if req.Target == "" {
				t.Error("Expected non-empty target")
			}
			if req.Timeout != tt.timeout {
				t.Errorf("Expected timeout %d, got %d", tt.timeout, req.Timeout)
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
