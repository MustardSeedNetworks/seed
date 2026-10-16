package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// TestHandleTCPProbe tests the TCP probe endpoint.
func TestHandleTCPProbe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
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
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/security/discovery/probe",
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
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/security/discovery/probe", http.NoBody)
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

// TestHandleTCPProbeInvalidJSON tests TCP probe with invalid JSON.
func TestHandleTCPProbeInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/security/discovery/probe",
		bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleTCPProbeTooManyPorts tests TCP probe with too many ports.
func TestHandleTCPProbeTooManyPorts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
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

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/security/discovery/probe",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
	assertErrorMessage(t, w, "Enter from 1 to 100 ports")
}

// TestHandleTCPProbeInvalidTarget tests TCP probe with invalid target.
func TestHandleTCPProbeInvalidTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	reqBody := api.TCPProbeRequest{
		Target:  "invalid..hostname",
		Port:    80,
		Timeout: 1000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/security/discovery/probe",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
	assertErrorMessage(t, w, "Invalid target: enter a hostname or IP address")
}

// assertErrorMessage pins the message an operator reads. A key with no copy
// used to arrive here verbatim ("errors.tools.invalidTarget") with every
// status-code assertion green (#2299).
func assertErrorMessage(t *testing.T, w *httptest.ResponseRecorder, want string) {
	t.Helper()

	var resp api.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if resp.Error != want {
		t.Errorf("error message = %q, want %q", resp.Error, want)
	}
}

// TestHandleAdvancedFingerprint tests the advanced fingerprint endpoint.
func TestHandleAdvancedFingerprint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
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
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/security/discovery/fingerprint",
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf(
					"Expected status %d, got %d: %s",
					tt.expectedStatus,
					w.Code,
					w.Body.String(),
				)
			}
			if w.Code != http.StatusOK {
				return
			}
			// The documented AdvancedProbeResult is the whole answer.
			dec := json.NewDecoder(w.Body)
			dec.DisallowUnknownFields()
			var got discovery.AdvancedProbeResult
			if err := dec.Decode(&got); err != nil {
				t.Fatalf("decode AdvancedProbeResult: %v", err)
			}
			if got.IP != tt.request["ip"] {
				t.Errorf("ip = %q, want %q", got.IP, tt.request["ip"])
			}
		})
	}
}

// TestHandleAdvancedFingerprintMethodNotAllowed tests non-POST methods.
func TestHandleAdvancedFingerprintMethodNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/security/discovery/fingerprint", http.NoBody)
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

// TestHandleAdvancedFingerprintInvalidJSON tests fingerprint with invalid JSON.
func TestHandleAdvancedFingerprintInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/security/discovery/fingerprint",
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
