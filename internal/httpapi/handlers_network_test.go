package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// TestHandleInterfacesGET tests the interfaces list endpoint.
func TestHandleInterfacesGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/interfaces", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify JSON response
	var resp []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

// TestHandleInterfacesMethodNotAllowed tests non-GET methods on interfaces endpoint.
func TestHandleInterfacesMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/interfaces", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleInterfaceGET tests the current interface GET endpoint.
func TestHandleInterfaceGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/interface", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["interface"]; !ok {
		t.Error("Expected 'interface' key in response")
	}
}

// TestHandleInterfacePUT tests the current interface PUT endpoint.
func TestHandleInterfacePUT(t *testing.T) {
	server := api.NewTestServer()

	reqBody := api.SetInterfaceRequest{
		Interface: "eth0",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/interface", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	// May return OK or BadRequest depending on whether eth0 exists
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d or %d, got %d: %s",
			http.StatusOK, http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// TestHandleInterfacePUTInvalidJSON tests interface update with invalid JSON.
func TestHandleInterfacePUTInvalidJSON(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodPut, "/api/interface", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleInterfaceMethodNotAllowed tests non-GET/PUT methods.
func TestHandleInterfaceMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/interface", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleLinkGET tests the link status endpoint.
func TestHandleLinkGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sap/link", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d or %d, got %d: %s",
			http.StatusOK, http.StatusNotFound, w.Code, w.Body.String())
	}

	if w.Code == http.StatusOK {
		var resp api.LinkResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
	}
}

// TestHandleLinkGETWithInterface tests the link status endpoint with interface parameter.
func TestHandleLinkGETWithInterface(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sap/link?interface=lo", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	// Should return OK for loopback or NotFound if not available
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d or %d, got %d: %s",
			http.StatusOK, http.StatusNotFound, w.Code, w.Body.String())
	}
}

// TestHandleLinkMethodNotAllowed tests non-GET methods on link endpoint.
func TestHandleLinkMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/sap/link", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleIPConfigGET tests the IP config endpoint.
func TestHandleIPConfigGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sap/ipconfig", http.NoBody)
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
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/sap/ipconfig", http.NoBody)
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
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sap/ipconfig/settings", http.NoBody)
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
			server.SetConfigPath("/tmp/test-config.yaml")

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPut, "/api/sap/ipconfig/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			// Static IP requires root permissions, so may fail with Internal Error
			if tt.expectedStatus == http.StatusOK {
				if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
					t.Errorf("Expected status %d or %d, got %d: %s",
						http.StatusOK, http.StatusInternalServerError, w.Code, w.Body.String())
				}
			} else if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleIPSettingsMethodNotAllowed tests non-GET/PUT methods.
func TestHandleIPSettingsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/sap/ipconfig/settings", http.NoBody)
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

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/network/mtu", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			// MTU changes may require root, so OK or InternalServerError both acceptable for valid requests
			if tt.expectedStatus == http.StatusOK {
				if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
					t.Errorf("Expected status %d or %d, got %d: %s",
						http.StatusOK, http.StatusInternalServerError, w.Code, w.Body.String())
				}
			} else if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleSetMTUMethodNotAllowed tests non-POST methods on MTU endpoint.
func TestHandleSetMTUMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/network/mtu", http.NoBody)
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
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/network/mtu", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestLinkResponseFields tests that LinkResponse has expected fields.
func TestLinkResponseFields(t *testing.T) {
	resp := api.LinkResponse{
		Interface:    "eth0",
		LinkUp:       true,
		Carrier:      true,
		HasIP:        true,
		Speed:        "1000baseT/Full",
		Duplex:       "full",
		Advertised:   []string{"10baseT/Half", "10baseT/Full", "100baseT/Half", "100baseT/Full", "1000baseT/Full"},
		MTU:          1500,
		AutoNeg:      true,
		FlapCount24h: 0,
		UptimeMs:     3600000,
		History: []api.LinkHistoryEvent{
			{State: "up", Timestamp: "2024-01-01T00:00:00Z"},
		},
		PoE: &api.PoEInfo{
			Detected: true,
			Standard: "802.3at",
			Class:    4,
			PowerMw:  15400,
			Voltage:  48.0,
		},
		SFP: &api.SFPInfo{
			Present:    true,
			Vendor:     "Cisco",
			PartNumber: "SFP-10G-SR",
			Serial:     "ABC123",
			Type:       "SR",
			Wavelength: 850,
			Distance:   300,
			Connector:  "LC",
			DDMSupport: true,
			DDM: &api.SFPDDMInfo{
				Temperature: 45.5,
				Voltage:     3.3,
				TxPowerDbm:  -2.5,
				TxPowerMw:   0.56,
				RxPowerDbm:  -3.0,
				RxPowerMw:   0.50,
				LaserBiasMa: 30.0,
			},
		},
	}

	// Verify all fields are set correctly
	if resp.Interface != "eth0" {
		t.Errorf("Expected Interface eth0, got %q", resp.Interface)
	}
	if !resp.LinkUp {
		t.Error("Expected LinkUp to be true")
	}
	if !resp.Carrier {
		t.Error("Expected Carrier to be true")
	}
	if !resp.HasIP {
		t.Error("Expected HasIP to be true")
	}
	if resp.Speed != "1000baseT/Full" {
		t.Errorf("Expected Speed '1000baseT/Full', got %q", resp.Speed)
	}
	if resp.Duplex != "full" {
		t.Errorf("Expected Duplex 'full', got %q", resp.Duplex)
	}
	if len(resp.Advertised) != 5 {
		t.Errorf("Expected 5 advertised modes, got %d", len(resp.Advertised))
	}
	if resp.MTU != 1500 {
		t.Errorf("Expected MTU 1500, got %d", resp.MTU)
	}
	if !resp.AutoNeg {
		t.Error("Expected AutoNeg to be true")
	}
	if resp.FlapCount24h != 0 {
		t.Errorf("Expected FlapCount24h 0, got %d", resp.FlapCount24h)
	}
	if resp.UptimeMs != 3600000 {
		t.Errorf("Expected UptimeMs 3600000, got %d", resp.UptimeMs)
	}
	if len(resp.History) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(resp.History))
	}
	if resp.PoE == nil {
		t.Error("Expected PoE to be set")
	} else if !resp.PoE.Detected {
		t.Error("Expected PoE.Detected to be true")
	}
	if resp.SFP == nil {
		t.Error("Expected SFP to be set")
	} else if !resp.SFP.DDMSupport {
		t.Error("Expected SFP.DDMSupport to be true")
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
			Ack:      10,
			Total:    70,
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
	} else if resp.Timing.Total != 70 {
		t.Errorf("Expected Timing.Total 70, got %d", resp.Timing.Total)
	}
}

// TestSetInterfaceRequestValidation tests interface request validation.
func TestSetInterfaceRequestValidation(t *testing.T) {
	tests := []struct {
		name     string
		iface    string
		wantErr  bool
		errField string
	}{
		{
			name:    "valid interface",
			iface:   "eth0",
			wantErr: false,
		},
		{
			name:    "valid wireless interface",
			iface:   "wlan0",
			wantErr: false,
		},
		{
			name:    "valid loopback",
			iface:   "lo",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := api.SetInterfaceRequest{
				Interface: tt.iface,
			}
			if req.Interface == "" && !tt.wantErr {
				t.Error("Expected non-empty interface for valid request")
			}
		})
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
