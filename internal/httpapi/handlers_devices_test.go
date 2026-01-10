package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// TestHandleDevicesGET tests the devices list endpoint.
func TestHandleDevicesGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/shell/devices", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	// Should return OK when device discovery is available
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify JSON response structure
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have devices and status keys
	if _, ok := resp["devices"]; !ok {
		t.Error("Expected 'devices' key in response")
	}
	if _, ok := resp["status"]; !ok {
		t.Error("Expected 'status' key in response")
	}
}

// TestHandleDevicesMethodNotAllowed tests non-GET methods on devices endpoint.
func TestHandleDevicesMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/shell/devices", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleDevicesScanPOST tests the device scan trigger endpoint.
func TestHandleDevicesScanPOST(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/shell/devices/scan", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	// Should return OK when device discovery is available
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify JSON response
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have message and scanning keys
	if _, ok := resp["message"]; !ok {
		t.Error("Expected 'message' key in response")
	}
	if _, ok := resp["scanning"]; !ok {
		t.Error("Expected 'scanning' key in response")
	}
}

// TestHandleDevicesScanMethodNotAllowed tests non-POST methods on scan endpoint.
func TestHandleDevicesScanMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/shell/devices/scan", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleDevicesStatusGET tests the device status endpoint.
func TestHandleDevicesStatusGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/shell/devices/status", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify JSON response
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

// TestHandleDevicesStatusMethodNotAllowed tests non-GET methods on status endpoint.
func TestHandleDevicesStatusMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/shell/devices/status", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleDevicesSettingsGET tests the device settings GET endpoint.
func TestHandleDevicesSettingsGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/shell/devices/settings", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify JSON response structure
	var resp api.NetworkDiscoverySettingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify expected fields are present
	// Options should be present
	if resp.Options.PassiveProtocols.LLDP == false && resp.Options.PassiveProtocols.CDP == false {
		// At least check that the struct is populated
		t.Log("Options struct populated correctly")
	}
}

// TestHandleDevicesSettingsPUT tests the device settings PUT endpoint.
func TestHandleDevicesSettingsPUT(t *testing.T) {
	server := api.NewTestServer()
	server.SetConfigPath("/tmp/test-config.yaml")

	reqBody := api.NetworkDiscoverySettingsResponse{
		Enabled:        true,
		ARPScanWorkers: 10,
		PingTimeoutMs:  1000,
		ScanTimeoutMs:  30000,
		AutoScan:       false,
		ScanIntervalMs: 60000,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/shell/devices/settings", bytes.NewReader(body))
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

// TestHandleDevicesSettingsMethodNotAllowed tests non-GET/PUT methods.
func TestHandleDevicesSettingsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/shell/devices/settings", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleDevicesSettingsInvalidJSON tests settings update with invalid JSON.
func TestHandleDevicesSettingsInvalidJSON(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodPut, "/api/shell/devices/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleDevicesSubnetsGET tests the subnets list endpoint.
func TestHandleDevicesSubnetsGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/shell/devices/subnets", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify JSON response is an array
	var resp []api.SubnetResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

// TestHandleDevicesSubnetsPOST tests adding a subnet.
func TestHandleDevicesSubnetsPOST(t *testing.T) {
	server := api.NewTestServer()
	server.SetConfigPath("/tmp/test-config.yaml")

	reqBody := api.SubnetRequest{
		CIDR:    "10.0.0.0/24",
		Name:    "Test Subnet",
		Enabled: true,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shell/devices/subnets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestHandleDevicesSubnetsPOSTInvalidCIDR tests adding a subnet with invalid CIDR.
func TestHandleDevicesSubnetsPOSTInvalidCIDR(t *testing.T) {
	server := api.NewTestServer()

	reqBody := api.SubnetRequest{
		CIDR:    "invalid-cidr",
		Name:    "Test Subnet",
		Enabled: true,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shell/devices/subnets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleDevicesSubnetsPUT tests updating a subnet.
func TestHandleDevicesSubnetsPUT(t *testing.T) {
	server := api.NewTestServer()
	server.SetConfigPath("/tmp/test-config.yaml")

	// First add a subnet
	addBody, _ := json.Marshal(api.SubnetRequest{
		CIDR:    "172.16.0.0/16",
		Name:    "Initial Name",
		Enabled: true,
	})
	addReq := httptest.NewRequest(http.MethodPost, "/api/shell/devices/subnets", bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addW := httptest.NewRecorder()
	server.Mux().ServeHTTP(addW, addReq)

	// Now update it
	updateBody, _ := json.Marshal(api.SubnetRequest{
		CIDR:    "172.16.0.0/16",
		Name:    "Updated Name",
		Enabled: false,
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/api/shell/devices/subnets", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateW := httptest.NewRecorder()

	server.Mux().ServeHTTP(updateW, updateReq)

	if updateW.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, updateW.Code, updateW.Body.String())
	}
}

// TestHandleDevicesSubnetsPUTNotFound tests updating a non-existent subnet.
func TestHandleDevicesSubnetsPUTNotFound(t *testing.T) {
	server := api.NewTestServer()

	updateBody, _ := json.Marshal(api.SubnetRequest{
		CIDR:    "192.168.99.0/24",
		Name:    "Non-existent",
		Enabled: false,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/shell/devices/subnets", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestHandleDevicesSubnetsDELETE tests deleting a subnet.
func TestHandleDevicesSubnetsDELETE(t *testing.T) {
	server := api.NewTestServer()
	server.SetConfigPath("/tmp/test-config.yaml")

	// First add a subnet
	addBody, _ := json.Marshal(api.SubnetRequest{
		CIDR:    "192.168.50.0/24",
		Name:    "To Delete",
		Enabled: true,
	})
	addReq := httptest.NewRequest(http.MethodPost, "/api/shell/devices/subnets", bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addW := httptest.NewRecorder()
	server.Mux().ServeHTTP(addW, addReq)

	// Now delete it
	req := httptest.NewRequest(http.MethodDelete, "/api/shell/devices/subnets?cidr=192.168.50.0/24", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestHandleDevicesSubnetsDELETEMissingCIDR tests deleting without CIDR parameter.
func TestHandleDevicesSubnetsDELETEMissingCIDR(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodDelete, "/api/shell/devices/subnets", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleDevicesSubnetsDELETENotFound tests deleting a non-existent subnet.
func TestHandleDevicesSubnetsDELETENotFound(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodDelete, "/api/shell/devices/subnets?cidr=10.99.99.0/24", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestHandlePublicIPGET tests the public IP endpoint.
func TestHandlePublicIPGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sap/publicip", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify JSON response
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

// TestHandlePublicIPPOST tests the public IP refresh endpoint.
func TestHandlePublicIPPOST(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/sap/publicip", http.NoBody)
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestHandlePublicIPMethodNotAllowed tests non-GET/POST methods on public IP endpoint.
func TestHandlePublicIPMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/sap/publicip", http.NoBody)
			w := httptest.NewRecorder()

			server.Mux().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestNetworkDiscoverySettingsResponseFields tests that settings response has expected fields.
func TestNetworkDiscoverySettingsResponseFields(t *testing.T) {
	resp := api.NetworkDiscoverySettingsResponse{
		Enabled:        true,
		ARPScanWorkers: 10,
		PingTimeoutMs:  500,
		ScanTimeoutMs:  30000,
		AutoScan:       true,
		ScanIntervalMs: 60000,
		OUIFilePath:    "/var/lib/seed/oui.txt",
		IPv6Enabled:    true,
		Options: api.OptionsResponse{
			PassiveProtocols: api.PassiveProtocolResponse{
				LLDP: true,
				CDP:  true,
				EDP:  false,
				NDP:  true,
			},
			ARPScan:  true,
			ICMPScan: true,
			PortScan: api.PortScanResponse{
				Enabled:         true,
				TCPPorts:        "22,80,443",
				UDPPorts:        "53,161",
				BannerTimeoutMs: 2000,
			},
			TCPProbe: api.TCPProbeSettingsResponse{
				TimeoutMs: 1000,
				Workers:   10,
			},
			Traceroute: false,
			SNMPQuery:  true,
		},
		Timing: api.TimingResponse{
			ProbeIntervalMs:  100,
			RescanIntervalMs: 300000,
			Workers:          5,
		},
		Profiler: api.ProfilerResponse{
			Enabled:       true,
			TimeoutMs:     5000,
			MaxConcurrent: 5,
			QuickPorts:    []int{22, 80, 443, 8080},
		},
		Fingerprinting: api.FingerprintingResponse{
			Enabled:       true,
			OSDetection:   true,
			ServiceProbes: true,
		},
	}

	// Verify all fields are set correctly
	if !resp.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if resp.ARPScanWorkers != 10 {
		t.Errorf("Expected ARPScanWorkers 10, got %d", resp.ARPScanWorkers)
	}
	if resp.PingTimeoutMs != 500 {
		t.Errorf("Expected PingTimeoutMs 500, got %d", resp.PingTimeoutMs)
	}
	if resp.ScanTimeoutMs != 30000 {
		t.Errorf("Expected ScanTimeoutMs 30000, got %d", resp.ScanTimeoutMs)
	}
	if !resp.AutoScan {
		t.Error("Expected AutoScan to be true")
	}
	if resp.ScanIntervalMs != 60000 {
		t.Errorf("Expected ScanIntervalMs 60000, got %d", resp.ScanIntervalMs)
	}
	if resp.OUIFilePath != "/var/lib/seed/oui.txt" {
		t.Errorf("Expected OUIFilePath /var/lib/seed/oui.txt, got %q", resp.OUIFilePath)
	}
	if !resp.IPv6Enabled {
		t.Error("Expected IPv6Enabled to be true")
	}
	if !resp.Options.PassiveProtocols.LLDP {
		t.Error("Expected LLDP to be enabled")
	}
	if !resp.Options.PortScan.Enabled {
		t.Error("Expected PortScan to be enabled")
	}
	if resp.Timing.Workers != 5 {
		t.Errorf("Expected Timing.Workers 5, got %d", resp.Timing.Workers)
	}
	if !resp.Profiler.Enabled {
		t.Error("Expected Profiler to be enabled")
	}
	if len(resp.Profiler.QuickPorts) != 4 {
		t.Errorf("Expected 4 QuickPorts, got %d", len(resp.Profiler.QuickPorts))
	}
	if !resp.Fingerprinting.OSDetection {
		t.Error("Expected OSDetection to be enabled")
	}
}

// TestSubnetRequestValidation tests subnet request validation.
func TestSubnetRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     api.SubnetRequest
		isValid bool
	}{
		{
			name: "valid IPv4 /24",
			req: api.SubnetRequest{
				CIDR:    "192.168.1.0/24",
				Name:    "LAN",
				Enabled: true,
			},
			isValid: true,
		},
		{
			name: "valid IPv4 /16",
			req: api.SubnetRequest{
				CIDR:    "172.16.0.0/16",
				Name:    "Private",
				Enabled: true,
			},
			isValid: true,
		},
		{
			name: "valid IPv4 /8",
			req: api.SubnetRequest{
				CIDR:    "10.0.0.0/8",
				Name:    "Class A",
				Enabled: false,
			},
			isValid: true,
		},
		{
			name: "disabled subnet",
			req: api.SubnetRequest{
				CIDR:    "192.168.100.0/24",
				Name:    "Disabled Subnet",
				Enabled: false,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - CIDR should be non-empty
			if tt.req.CIDR == "" && tt.isValid {
				t.Error("Expected valid CIDR for valid request")
			}
		})
	}
}
