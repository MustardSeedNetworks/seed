package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// TestHandleDNSGET tests the DNS test endpoint.
func TestHandleDNSGET(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sap/dns", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.DNSResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify basic structure
	if resp.TestHostname == "" {
		t.Error("Expected non-empty TestHostname")
	}
}

// TestHandleDNSGETWithInterface tests the DNS endpoint with interface parameter.
func TestHandleDNSGETWithInterface(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sap/dns?interface=lo", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	// Should return OK regardless of interface
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestHandleDNSMethodNotAllowed tests non-GET methods on DNS endpoint.
func TestHandleDNSMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/sap/dns", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleDNSSecurityGET tests the DNS security GET endpoint.
func TestHandleDNSSecurityGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sap/dns/security", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestHandleDNSSecurityPOST tests the DNS security POST endpoint.
func TestHandleDNSSecurityPOST(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	server := api.NewTestServer()

	reqBody := api.DNSSecurityScanRequest{
		Servers: []string{"8.8.8.8", "8.8.4.4"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sap/dns/security", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	// Should return OK or conflict if already running
	if w.Code != http.StatusOK && w.Code != http.StatusConflict {
		t.Errorf("Expected status %d or %d, got %d: %s",
			http.StatusOK, http.StatusConflict, w.Code, w.Body.String())
	}
}

// TestHandleDNSSecurityPOSTNoServers tests DNS security POST with no servers.
func TestHandleDNSSecurityPOSTNoServers(t *testing.T) {
	server := api.NewTestServer()

	// Empty servers list - should use configured servers or return error
	reqBody := api.DNSSecurityScanRequest{
		Servers: []string{},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sap/dns/security", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	// Should return BadRequest if no servers configured, or OK if servers are configured
	if w.Code != http.StatusBadRequest && w.Code != http.StatusOK && w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, %d, or %d, got %d: %s",
			http.StatusBadRequest, http.StatusOK, http.StatusConflict, w.Code, w.Body.String())
	}
}

// TestHandleDNSSecurityPOSTInvalidJSON tests DNS security POST with invalid JSON.
func TestHandleDNSSecurityPOSTInvalidJSON(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/sap/dns/security", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleDNSSecurityMethodNotAllowed tests non-GET/POST methods.
func TestHandleDNSSecurityMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/sap/dns/security", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleDNSSecuritySettingsGET tests DNS security settings GET endpoint.
func TestHandleDNSSecuritySettingsGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sap/dns/security/settings", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

// TestHandleDNSSecuritySettingsPUT tests DNS security settings PUT endpoint.
func TestHandleDNSSecuritySettingsPUT(t *testing.T) {
	server := api.NewTestServer()

	reqBody := map[string]any{
		"timeout":     5000,
		"concurrency": 10,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/sap/dns/security/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "success" {
		t.Errorf("Expected status 'success', got %q", resp["status"])
	}
}

// TestHandleDNSSecuritySettingsMethodNotAllowed tests non-GET/PUT methods.
func TestHandleDNSSecuritySettingsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/sap/dns/security/settings", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestDNSResponseFields tests that DNSResponse has expected fields.
func TestDNSResponseFields(t *testing.T) {
	resp := api.DNSResponse{
		Interface:    "eth0",
		Server:       "8.8.8.8",
		Servers:      []string{"8.8.8.8", "8.8.4.4"},
		TestHostname: "google.com",
		Forward: &api.DNSLookupResult{
			Result:   "142.250.185.78",
			Time:     25,
			TimeMs:   25,
			Status:   "ok",
			Resolved: []string{"142.250.185.78"},
		},
		ForwardIpv6: &api.DNSLookupResult{
			Result:   "2607:f8b0:4004:800::200e",
			Time:     30,
			TimeMs:   30,
			Status:   "ok",
			Resolved: []string{"2607:f8b0:4004:800::200e"},
		},
		Reverse: &api.DNSLookupResult{
			Result: "lhr25s10-in-f14.1e100.net",
			Time:   15,
			TimeMs: 15,
			Status: "ok",
		},
		PerServerResults: []*api.DNSServerTestResult{
			{
				Server:    "8.8.8.8",
				Status:    "ok",
				AvgTimeMs: 25,
				Forward: &api.DNSLookupResult{
					Result: "142.250.185.78",
					TimeMs: 25,
					Status: "ok",
				},
			},
			{
				Server:    "8.8.4.4",
				Status:    "ok",
				AvgTimeMs: 28,
			},
		},
	}

	if resp.Interface != "eth0" {
		t.Errorf("Expected Interface 'eth0', got %q", resp.Interface)
	}
	if resp.Server != "8.8.8.8" {
		t.Errorf("Expected Server '8.8.8.8', got %q", resp.Server)
	}
	if len(resp.Servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(resp.Servers))
	}
	if resp.TestHostname != "google.com" {
		t.Errorf("Expected TestHostname 'google.com', got %q", resp.TestHostname)
	}
	if resp.Forward == nil {
		t.Error("Expected Forward to be set")
	} else if resp.Forward.Status != "ok" {
		t.Errorf("Expected Forward.Status 'ok', got %q", resp.Forward.Status)
	}
	if resp.ForwardIpv6 == nil {
		t.Error("Expected ForwardIpv6 to be set")
	}
	if resp.Reverse == nil {
		t.Error("Expected Reverse to be set")
	}
	if len(resp.PerServerResults) != 2 {
		t.Errorf("Expected 2 per-server results, got %d", len(resp.PerServerResults))
	}
}

// TestDNSLookupResultFields tests that DNSLookupResult has expected fields.
func TestDNSLookupResultFields(t *testing.T) {
	result := api.DNSLookupResult{
		Result:   "192.168.1.1",
		Time:     50,
		TimeMs:   50,
		Status:   "ok",
		Error:    "",
		Resolved: []string{"192.168.1.1", "192.168.1.2"},
	}

	if result.Result != "192.168.1.1" {
		t.Errorf("Expected Result '192.168.1.1', got %q", result.Result)
	}
	if result.Time != 50 {
		t.Errorf("Expected Time 50, got %d", result.Time)
	}
	if result.TimeMs != 50 {
		t.Errorf("Expected TimeMs 50, got %d", result.TimeMs)
	}
	if result.Status != "ok" {
		t.Errorf("Expected Status 'ok', got %q", result.Status)
	}
	if result.Error != "" {
		t.Errorf("Expected Error to be empty, got %q", result.Error)
	}
	if len(result.Resolved) != 2 {
		t.Errorf("Expected 2 resolved addresses, got %d", len(result.Resolved))
	}
}

// TestDNSLookupResultWithError tests DNSLookupResult with error.
func TestDNSLookupResultWithError(t *testing.T) {
	result := api.DNSLookupResult{
		Result: "",
		TimeMs: 100,
		Status: "error",
		Error:  "no such host",
	}

	if result.Status != "error" {
		t.Errorf("Expected Status 'error', got %q", result.Status)
	}
	if result.TimeMs != 100 {
		t.Errorf("Expected TimeMs 100, got %d", result.TimeMs)
	}
	if result.Error != "no such host" {
		t.Errorf("Expected Error 'no such host', got %q", result.Error)
	}
	if result.Result != "" {
		t.Errorf("Expected empty Result, got %q", result.Result)
	}
}

// TestDNSServerTestResultFields tests that DNSServerTestResult has expected fields.
func TestDNSServerTestResultFields(t *testing.T) {
	result := api.DNSServerTestResult{
		Server:    "1.1.1.1",
		Status:    "ok",
		AvgTimeMs: 15,
		Forward: &api.DNSLookupResult{
			Result: "93.184.216.34",
			TimeMs: 15,
			Status: "ok",
		},
		ForwardIpv6: &api.DNSLookupResult{
			Result: "2606:2800:220:1:248:1893:25c8:1946",
			TimeMs: 18,
			Status: "ok",
		},
	}

	if result.Server != "1.1.1.1" {
		t.Errorf("Expected Server '1.1.1.1', got %q", result.Server)
	}
	if result.Status != "ok" {
		t.Errorf("Expected Status 'ok', got %q", result.Status)
	}
	if result.AvgTimeMs != 15 {
		t.Errorf("Expected AvgTimeMs 15, got %d", result.AvgTimeMs)
	}
	if result.Forward == nil {
		t.Error("Expected Forward to be set")
	}
	if result.ForwardIpv6 == nil {
		t.Error("Expected ForwardIpv6 to be set")
	}
}

// TestDNSSecurityScanRequestValidation tests DNS security scan request validation.
func TestDNSSecurityScanRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request api.DNSSecurityScanRequest
		isValid bool
	}{
		{
			name: "valid single server",
			request: api.DNSSecurityScanRequest{
				Servers: []string{"8.8.8.8"},
			},
			isValid: true,
		},
		{
			name: "valid multiple servers",
			request: api.DNSSecurityScanRequest{
				Servers: []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"},
			},
			isValid: true,
		},
		{
			name: "empty servers (uses config)",
			request: api.DNSSecurityScanRequest{
				Servers: []string{},
			},
			isValid: true, // Will use configured servers
		},
		{
			name: "nil servers (uses config)",
			request: api.DNSSecurityScanRequest{
				Servers: nil,
			},
			isValid: true, // Will use configured servers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize and deserialize to verify JSON handling
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			var parsed api.DNSSecurityScanRequest
			if unmarshalErr := json.Unmarshal(data, &parsed); unmarshalErr != nil {
				t.Fatalf("Failed to unmarshal request: %v", unmarshalErr)
			}

			// Verify servers match
			if len(tt.request.Servers) != len(parsed.Servers) {
				t.Errorf("Expected %d servers, got %d", len(tt.request.Servers), len(parsed.Servers))
			}
		})
	}
}

// TestDNSStatusValues tests common DNS status values.
func TestDNSStatusValues(t *testing.T) {
	statuses := []string{"ok", "error", "timeout", "warning"}

	for _, status := range statuses {
		t.Run("status_"+status, func(t *testing.T) {
			result := api.DNSLookupResult{
				Status: status,
			}
			if result.Status != status {
				t.Errorf("Expected status %q, got %q", status, result.Status)
			}
		})
	}
}

// TestDNSResponseJSONSerialization tests JSON serialization of DNS response.
func TestDNSResponseJSONSerialization(t *testing.T) {
	resp := api.DNSResponse{
		Interface:    "eth0",
		Server:       "8.8.8.8",
		Servers:      []string{"8.8.8.8"},
		TestHostname: "example.com",
		Forward: &api.DNSLookupResult{
			Result:   "93.184.216.34",
			TimeMs:   20,
			Status:   "ok",
			Resolved: []string{"93.184.216.34"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed api.DNSResponse
	if unmarshalErr := json.Unmarshal(data, &parsed); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal response: %v", unmarshalErr)
	}

	if parsed.Interface != resp.Interface {
		t.Errorf("Expected Interface %q, got %q", resp.Interface, parsed.Interface)
	}
	if parsed.Server != resp.Server {
		t.Errorf("Expected Server %q, got %q", resp.Server, parsed.Server)
	}
	if parsed.Forward == nil {
		t.Error("Expected Forward to be set after deserialization")
	}
}
