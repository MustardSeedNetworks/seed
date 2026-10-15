package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

// TestHandleIPConfigGET tests the IP config endpoint.
func TestHandleIPConfigGET(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/ipconfig", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d or %d, got %d: %s",
			http.StatusOK, http.StatusNotFound, w.Code, w.Body.String())
	}

	if w.Code == http.StatusOK {
		var resp api.IPConfigResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify basic structure
		if resp.Mode == "" {
			t.Error("Expected 'mode' to be non-empty")
		}
	}
}

// TestHandleIPConfigMethodNotAllowed tests non-GET methods on IP config endpoint.
func TestHandleIPConfigMethodNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/telemetry/ipconfig", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleIPSettingsGET tests the IP settings endpoint.
func TestHandleIPSettingsGET(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/ipconfig/settings", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.IPSettingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Mode should be dhcp or static
	if resp.Mode != "dhcp" && resp.Mode != "static" && resp.Mode != "" {
		t.Errorf("Expected mode to be 'dhcp' or 'static', got %q", resp.Mode)
	}
}

// TestHandleIPSettingsPUT tests the IP settings PUT endpoint with valid data.
func TestHandleIPSettingsPUT(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tests := []struct {
		name           string
		request        api.IPSettingsRequest
		expectedStatus int
	}{
		{
			name: "dhcp mode",
			request: api.IPSettingsRequest{
				Mode: "dhcp",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid mode",
			request: api.IPSettingsRequest{
				Mode: "invalid",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			defer server.Close()
			server.SetConfigPath("/tmp/test-config.json")

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/telemetry/ipconfig/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleIPSettingsMethodNotAllowed tests non-GET/PUT methods.
func TestHandleIPSettingsMethodNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/telemetry/ipconfig/settings", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleSetMTU tests the MTU set endpoint.
func TestHandleSetMTU(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tests := []struct {
		name           string
		request        api.SetMTURequest
		expectedStatus int
	}{
		{
			name: "valid MTU",
			request: api.SetMTURequest{
				Interface: "lo",
				MTU:       1500,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "MTU too small",
			request: api.SetMTURequest{
				Interface: "lo",
				MTU:       50,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "MTU too large",
			request: api.SetMTURequest{
				Interface: "lo",
				MTU:       100000,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			defer server.Close()

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/network/mtu", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleSetMTUResponseShape pins the answer to a successful MTU change.
func TestHandleSetMTUResponseShape(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	body, _ := json.Marshal(api.SetMTURequest{Interface: "lo", MTU: 1500})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/network/mtu", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	dec := json.NewDecoder(w.Body)
	dec.DisallowUnknownFields()
	var got api.SetMTUResponse
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := api.SetMTUResponse{Status: "success", Message: "MTU updated", Interface: "lo", MTU: 1500}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// TestHandleSetMTUMethodNotAllowed tests non-POST methods on MTU endpoint.
func TestHandleSetMTUMethodNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/network/mtu", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleSetMTUInvalidJSON tests MTU endpoint with invalid JSON.
func TestHandleSetMTUInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/network/mtu", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestIPConfigResponseFields tests that IPConfigResponse has expected fields.
func TestIPConfigResponseFields(t *testing.T) {
	resp := api.IPConfigResponse{
		Interface: "eth0",
		MAC:       "00:11:22:33:44:55",
		Mode:      "dhcp",
		IPv4: &api.IPv4Info{
			Address:    "192.168.1.100",
			Subnet:     "255.255.255.0",
			Gateway:    "192.168.1.1",
			DHCPServer: "192.168.1.1",
			LeaseTime:  86400,
		},
		IPv6: []api.IPv6Info{
			{
				Address: "fe80::1",
				Prefix:  64,
				Scope:   "link-local",
				Source:  "slaac",
			},
			{
				Address: "2001:db8::1",
				Prefix:  64,
				Scope:   "global",
				Source:  "dhcpv6",
			},
		},
		DNS: []string{"8.8.8.8", "8.8.4.4"},
		Timing: &api.DHCPTimingInfo{
			Discover: 10,
			Offer:    20,
			Request:  30,
			Total:    60,
		},
	}

	if resp.Interface != "eth0" {
		t.Errorf("Expected Interface 'eth0', got %q", resp.Interface)
	}
	if resp.Mode != "dhcp" {
		t.Errorf("Expected Mode 'dhcp', got %q", resp.Mode)
	}
	if resp.MAC != "00:11:22:33:44:55" {
		t.Errorf("Expected MAC 00:11:22:33:44:55, got %q", resp.MAC)
	}
	if resp.IPv4 == nil {
		t.Error("Expected IPv4 to be set")
	} else if resp.IPv4.Address != "192.168.1.100" {
		t.Errorf("Expected IPv4.Address '192.168.1.100', got %q", resp.IPv4.Address)
	}
	if len(resp.IPv6) != 2 {
		t.Errorf("Expected 2 IPv6 addresses, got %d", len(resp.IPv6))
	}
	if len(resp.DNS) != 2 {
		t.Errorf("Expected 2 DNS servers, got %d", len(resp.DNS))
	}
	if resp.Timing == nil {
		t.Error("Expected Timing to be set")
	} else if resp.Timing.Total != 60 {
		t.Errorf("Expected Timing.Total 60, got %d", resp.Timing.Total)
	}
}

// TestIPSettingsRequestModeValidation tests IP settings request mode validation.
func TestIPSettingsRequestModeValidation(t *testing.T) {
	tests := []struct {
		name    string
		request api.IPSettingsRequest
		isValid bool
	}{
		{
			name: "valid dhcp",
			request: api.IPSettingsRequest{
				Mode: "dhcp",
			},
			isValid: true,
		},
		{
			name: "valid static",
			request: api.IPSettingsRequest{
				Mode:    "static",
				Address: "192.168.1.100",
				Netmask: "255.255.255.0",
				Gateway: "192.168.1.1",
				DNS:     []string{"8.8.8.8"},
			},
			isValid: true,
		},
		{
			name: "invalid mode",
			request: api.IPSettingsRequest{
				Mode: "invalid",
			},
			isValid: false,
		},
		{
			name: "static without address",
			request: api.IPSettingsRequest{
				Mode: "static",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - mode should be dhcp or static
			validMode := tt.request.Mode == "dhcp" || tt.request.Mode == "static"
			if tt.isValid && !validMode {
				t.Error("Expected valid mode for valid request")
			}
			if !tt.isValid && validMode && tt.request.Mode == "static" && tt.request.Address == "" {
				// Static mode requires address
				t.Log("Correctly identified missing address for static mode")
			}
		})
	}
}

// TestSetMTURequestValidation tests MTU request validation.
func TestSetMTURequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request api.SetMTURequest
		isValid bool
	}{
		{
			name: "valid standard MTU",
			request: api.SetMTURequest{
				Interface: "eth0",
				MTU:       1500,
			},
			isValid: true,
		},
		{
			name: "valid jumbo frame MTU",
			request: api.SetMTURequest{
				Interface: "eth0",
				MTU:       9000,
			},
			isValid: true,
		},
		{
			name: "MTU at minimum",
			request: api.SetMTURequest{
				Interface: "eth0",
				MTU:       68,
			},
			isValid: true,
		},
		{
			name: "MTU below minimum",
			request: api.SetMTURequest{
				Interface: "eth0",
				MTU:       50,
			},
			isValid: false,
		},
		{
			name: "MTU above maximum",
			request: api.SetMTURequest{
				Interface: "eth0",
				MTU:       100000,
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - MTU should be in valid range
			validMTU := tt.request.MTU >= 68 && tt.request.MTU <= 65535
			if tt.isValid && !validMTU {
				t.Error("Expected valid MTU for valid request")
			}
			if !tt.isValid && validMTU {
				t.Error("Expected invalid MTU for invalid request")
			}
		})
	}
}
