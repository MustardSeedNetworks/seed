package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

// TestHandleInterfacesGET tests the interfaces list endpoint.
func TestHandleInterfacesGET(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/interfaces", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify JSON response
	var resp api.CategorizedInterfacesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

// TestHandleInterfacesMethodNotAllowed tests non-GET methods on interfaces endpoint.
func TestHandleInterfacesMethodNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/interfaces", http.NoBody)
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
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/interface", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Response includes interface (string), isWireless (bool), available (bool), etc.
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["interface"]; !ok {
		t.Error("Expected 'interface' key in response")
	}
}

// TestHandleInterfacePUT tests the current interface PUT endpoint.
func TestHandleInterfacePUT(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	reqBody := api.SetInterfaceRequest{
		Interface: "eth0",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/interface", bytes.NewReader(body))
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
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/interface", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleInterfaceMethodNotAllowed tests non-GET/PUT methods.
func TestHandleInterfaceMethodNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/interface", http.NoBody)
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
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/link", http.NoBody)
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
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/link?interface=lo", http.NoBody)
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
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/telemetry/link", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
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
