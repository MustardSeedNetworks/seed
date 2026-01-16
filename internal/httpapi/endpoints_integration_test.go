package httpapi_test

// Integration tests validate API endpoints for correct responses, configs, and data handling.

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	api "github.com/krisarmstrong/seed/internal/httpapi"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// testEndpointServer holds shared test server state.
type testEndpointServer struct {
	ts     *httptest.Server
	server *api.Server
}

func newTestEndpointServer(t *testing.T) *testEndpointServer {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().
		WithPort(8080).
		Build()

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	netMgr, err := network.NewManager("")
	if err != nil {
		t.Logf("Warning: Could not create network manager: %v", err)
	}

	server := api.NewServer(cfg, configPath, "", netMgr, false, nil, nil, nil)
	// Force IPv4 listener to avoid environments where IPv6 binds are disallowed
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create IPv4 listener: %v", err)
	}
	ts := httptest.NewUnstartedServer(server.Mux())
	ts.Listener = ln
	ts.Start()
	return &testEndpointServer{ts: ts, server: server}
}

func (s *testEndpointServer) close() {
	s.ts.Close()
	s.server.Close() // Stop goroutines (rate limiters, WebSocket hub, etc.)
}

func (s *testEndpointServer) testStatusEndpoint(t *testing.T) {
	t.Helper()
	resp, err := http.Get(s.ts.URL + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var status api.StatusResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&status); decodeErr != nil {
		t.Errorf("Failed to decode status response: %v", decodeErr)
	}

	if status.Status == "" {
		t.Error("Status field is empty")
	}
}

func (s *testEndpointServer) testSettingsEndpoint(t *testing.T) {
	t.Helper()
	resp, err := http.Get(s.ts.URL + "/api/settings")
	if err != nil {
		t.Fatalf("GET /api/settings failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var settings map[string]any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&settings); decodeErr != nil {
		t.Errorf("Failed to decode settings response: %v", decodeErr)
	}
}

func (s *testEndpointServer) testInterfacesEndpoint(t *testing.T) {
	t.Helper()
	resp, err := http.Get(s.ts.URL + "/api/interfaces")
	if err != nil {
		t.Fatalf("GET /api/interfaces failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var interfaces []any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&interfaces); decodeErr != nil {
		t.Errorf("Failed to decode interfaces response: %v", decodeErr)
	}
}

func (s *testEndpointServer) testSNMPSettingsEndpoint(t *testing.T) {
	t.Helper()
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Get(s.ts.URL + "/api/sap/snmp/settings")
	if err != nil {
		t.Fatalf("GET /api/sap/snmp/settings failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var settings api.SNMPSettingsResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&settings); decodeErr != nil {
		t.Errorf("Failed to decode SNMP settings: %v", decodeErr)
	}

	if settings.Port != 161 {
		t.Errorf("Expected port 161, got %d", settings.Port)
	}
	if settings.Timeout != 5000 {
		t.Errorf("Expected timeout 5000ms, got %d", settings.Timeout)
	}
}

func (s *testEndpointServer) testWiFiSettingsEndpoint(t *testing.T) {
	t.Helper()
	resp, err := http.Get(s.ts.URL + "/api/canopy/wifi/settings")
	if err != nil {
		t.Fatalf("GET /api/canopy/wifi/settings failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

func (s *testEndpointServer) testIPConfigEndpoint(t *testing.T) {
	t.Helper()
	resp, err := http.Get(s.ts.URL + "/api/ipconfig")
	if err != nil {
		t.Fatalf("GET /api/ipconfig failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	validCodes := resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusNotFound ||
		resp.StatusCode == http.StatusServiceUnavailable
	if !validCodes {
		t.Errorf("Expected status 200, 404, or 503, got %d", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusOK {
		var ipconfig api.IPConfigResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&ipconfig); decodeErr != nil {
			t.Errorf("Failed to decode ipconfig response: %v", decodeErr)
		}
	}
}

func (s *testEndpointServer) testSystemHealthEndpoint(t *testing.T) {
	t.Helper()
	resp, err := http.Get(s.ts.URL + "/api/sap/system/health")
	if err != nil {
		t.Fatalf("GET /api/sap/system/health failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health map[string]any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&health); decodeErr != nil {
		t.Errorf("Failed to decode health response: %v", decodeErr)
	}

	requiredFields := []string{"system", "application", "services"}
	for _, field := range requiredFields {
		if _, ok := health[field]; !ok {
			t.Errorf("Missing %s field in health response", field)
		}
	}

	if systemHealth, sysOK := health["system"].(map[string]any); sysOK {
		systemFields := []string{"cpuPercent", "memoryPercent"}
		for _, field := range systemFields {
			if _, fieldOK := systemHealth[field]; !fieldOK {
				t.Errorf("Missing %s field in system health", field)
			}
		}
	}
}

// TestAPIEndpoints provides comprehensive integration tests for all major API endpoints.
// Skipped in fast test mode - uses full server with OUI database loading.
func TestAPIEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	srv := newTestEndpointServer(t)
	defer srv.close()

	t.Run("StatusEndpoint", srv.testStatusEndpoint)
	t.Run("SettingsEndpoint", srv.testSettingsEndpoint)
	t.Run("InterfacesEndpoint", srv.testInterfacesEndpoint)
	t.Run("SNMPSettingsEndpoint", srv.testSNMPSettingsEndpoint)
	t.Run("WiFiSettingsEndpoint", srv.testWiFiSettingsEndpoint)
	t.Run("IPConfigEndpoint", srv.testIPConfigEndpoint)
	t.Run("SystemHealthEndpoint", srv.testSystemHealthEndpoint)
}

func TestWebSocketHub(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().Build()

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	netMgr, _ := network.NewManager("")
	server := api.NewServer(cfg, configPath, "", netMgr, false, nil, nil, nil)
	defer server.Close()

	// Test hub exists
	hub := server.Hub()
	if hub == nil {
		t.Error("WebSocket hub is nil")
	}
}

// func TestThresholdsUpdate(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	configPath := filepath.Join(tmpDir, "test-config.yaml")
//
// 	cfg := config.DefaultConfig()
// 	cfg.Interface.Default = "lo"
//
// 	if err := cfg.Save(configPath); err != nil {
// 		t.Fatalf("Failed to save test config: %v", err)
// 	}
//
// 	netMgr, _ := network.NewManager("")
// 	server := NewServer(cfg, configPath, "", netMgr, false, nil, nil, nil)
//
// 	ts := httptest.NewServer(server.mux)
// 	defer ts.Close()
//
// 	// Test threshold update
// 	thresholds := map[string]interface{}{
// 		"thresholds": map[string]interface{}{
// 			"dns": map[string]int64{
// 				"good":    50,
// 				"warning": 100,
// 			},
// 		},
// 	}
//
// 	body, _ := json.Marshal(thresholds)
// 	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/settings", bytes.NewReader(body))
// 	req.Header.Set("Content-Type", "application/json")
//
// 	client := &http.Client{Timeout: 15 * time.Second}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		t.Fatalf("PUT /api/settings failed: %v", err)
// 	}
// 	defer func() { _ = resp.Body.Close() }()
//
// 	if resp.StatusCode != http.StatusOK {
// 		t.Errorf("Expected status 200, got %d", resp.StatusCode)
// 	}
// }

// devicesTestServer wraps test server state for device endpoint tests.
type devicesTestServer struct {
	ts     *httptest.Server
	server *api.Server
}

// newDevicesTestServer creates a test server configured for device endpoints.
func newDevicesTestServer(t *testing.T) *devicesTestServer {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	cfg := testutil.NewConfigBuilder().Build()
	cfg.NetworkDiscovery.Enabled = true

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	netMgr, _ := network.NewManager("")
	server := api.NewServer(cfg, configPath, "", netMgr, false, nil, nil, nil)

	return &devicesTestServer{ts: httptest.NewServer(server.Mux()), server: server}
}

func (s *devicesTestServer) close() {
	s.ts.Close()
	s.server.Close() // Stop goroutines (rate limiters, WebSocket hub, etc.)
}

// assertGetEndpointOK performs a GET request and asserts HTTP 200 response.
func (s *devicesTestServer) assertGetEndpointOK(t *testing.T, endpoint string) {
	t.Helper()

	resp, err := http.Get(s.ts.URL + endpoint)
	if err != nil {
		t.Fatalf("GET %s failed: %v", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func (s *devicesTestServer) testGetDevices(t *testing.T) {
	t.Helper()
	s.assertGetEndpointOK(t, "/api/shell/devices")
}

func (s *devicesTestServer) testGetDevicesStatus(t *testing.T) {
	t.Helper()
	s.assertGetEndpointOK(t, "/api/shell/devices/status")
}

func (s *devicesTestServer) testGetDevicesSettings(t *testing.T) {
	t.Helper()

	resp, err := http.Get(s.ts.URL + "/api/shell/devices/settings")
	if err != nil {
		t.Fatalf("GET /api/shell/devices/settings failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var settings api.NetworkDiscoverySettingsResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&settings); decodeErr != nil {
		t.Errorf("Failed to decode discovery settings: %v", decodeErr)
	}

	if !settings.Enabled {
		t.Error("Expected discovery to be enabled")
	}
}

func (s *devicesTestServer) testScanDevices(t *testing.T) {
	t.Helper()

	req, _ := http.NewRequest(http.MethodPost, s.ts.URL+"/api/shell/devices/scan", http.NoBody)
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/shell/devices/scan failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should accept the request (may fail to actually scan without network)
	validCodes := resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable
	if !validCodes {
		t.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

func TestDevicesEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	srv := newDevicesTestServer(t)
	defer srv.close()

	t.Run("GetDevices", srv.testGetDevices)
	t.Run("GetDevicesStatus", srv.testGetDevicesStatus)
	t.Run("GetDevicesSettings", srv.testGetDevicesSettings)
	t.Run("ScanDevices", srv.testScanDevices)
}

func TestTestsSettingsEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().Build()

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	netMgr, _ := network.NewManager("")
	server := api.NewServer(cfg, configPath, "", netMgr, false, nil, nil, nil)
	defer server.Close()

	ts := httptest.NewServer(server.Mux())
	defer ts.Close()

	t.Run("GetHealthChecksSettings", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/sap/health-checks/settings")
		if err != nil {
			t.Fatalf("GET /api/sap/health-checks/settings failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var settings api.TestsSettingsResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&settings); decodeErr != nil {
			t.Errorf("Failed to decode health checks settings: %v", decodeErr)
		}
	})

	t.Run("UpdateHealthChecksSettings", func(t *testing.T) {
		// Issue #605 fixed: config.Save() deadlock resolved by unlocking before Save()
		settings := api.TestsSettingsResponse{
			DNSHostname: "example.com",
			DNSServers:  []api.DNSServerResponse{{Address: "8.8.8.8", Enabled: true}},
			PingTargets: []api.PingTargetResponse{{Name: "Google", Host: "8.8.8.8", Enabled: true}},
		}

		body, _ := json.Marshal(settings)
		req, _ := http.NewRequest(
			http.MethodPut,
			ts.URL+"/api/sap/health-checks/settings",
			bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("PUT /api/sap/health-checks/settings failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}
