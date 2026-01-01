// Package api provides the HTTP/WebSocket server.
// Integration tests validate API endpoints for correct responses, configs, and data handling.
package api

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// testEndpointServer holds shared test server state.
type testEndpointServer struct {
	ts *httptest.Server
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

	server := NewServer(cfg, configPath, "", netMgr, false, nil, nil, nil)
	// Force IPv4 listener to avoid environments where IPv6 binds are disallowed
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create IPv4 listener: %v", err)
	}
	ts := httptest.NewUnstartedServer(server.mux)
	ts.Listener = ln
	ts.Start()
	return &testEndpointServer{ts: ts}
}

func (s *testEndpointServer) close() {
	s.ts.Close()
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

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Errorf("Failed to decode status response: %v", err)
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
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Errorf("Failed to decode settings response: %v", err)
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
	if err := json.NewDecoder(resp.Body).Decode(&interfaces); err != nil {
		t.Errorf("Failed to decode interfaces response: %v", err)
	}
}

func (s *testEndpointServer) testSNMPSettingsEndpoint(t *testing.T) {
	t.Helper()
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Get(s.ts.URL + "/api/snmp/settings")
	if err != nil {
		t.Fatalf("GET /api/snmp/settings failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var settings SNMPSettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Errorf("Failed to decode SNMP settings: %v", err)
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
	resp, err := http.Get(s.ts.URL + "/api/wifi/settings")
	if err != nil {
		t.Fatalf("GET /api/wifi/settings failed: %v", err)
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
		var ipconfig IPConfigResponse
		if err := json.NewDecoder(resp.Body).Decode(&ipconfig); err != nil {
			t.Errorf("Failed to decode ipconfig response: %v", err)
		}
	}
}

func (s *testEndpointServer) testSystemHealthEndpoint(t *testing.T) {
	t.Helper()
	resp, err := http.Get(s.ts.URL + "/api/system/health")
	if err != nil {
		t.Fatalf("GET /api/system/health failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
	}

	requiredFields := []string{"system", "application", "services"}
	for _, field := range requiredFields {
		if _, ok := health[field]; !ok {
			t.Errorf("Missing %s field in health response", field)
		}
	}

	if systemHealth, ok := health["system"].(map[string]any); ok {
		systemFields := []string{"cpuPercent", "memoryPercent"}
		for _, field := range systemFields {
			if _, ok := systemHealth[field]; !ok {
				t.Errorf("Missing %s field in system health", field)
			}
		}
	}
}

// TestAPIEndpoints provides comprehensive integration tests for all major API endpoints.
func TestAPIEndpoints(t *testing.T) {
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
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().Build()

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	netMgr, _ := network.NewManager("")
	server := NewServer(cfg, configPath, "", netMgr, false, nil, nil, nil)

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

func TestDevicesEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().Build()
	cfg.NetworkDiscovery.Enabled = true

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	netMgr, _ := network.NewManager("")
	server := NewServer(cfg, configPath, "", netMgr, false, nil, nil, nil)

	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	t.Run("GetDevices", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/devices")
		if err != nil {
			t.Fatalf("GET /api/devices failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("GetDevicesStatus", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/devices/status")
		if err != nil {
			t.Fatalf("GET /api/devices/status failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("GetDevicesSettings", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/devices/settings")
		if err != nil {
			t.Fatalf("GET /api/devices/settings failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var settings NetworkDiscoverySettingsResponse
		if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
			t.Errorf("Failed to decode discovery settings: %v", err)
		}

		if !settings.Enabled {
			t.Error("Expected discovery to be enabled")
		}
	})

	t.Run("ScanDevices", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/devices/scan", http.NoBody)
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("POST /api/devices/scan failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Should accept the request (may fail to actually scan without network)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Unexpected status code: %d", resp.StatusCode)
		}
	})
}

func TestTestsSettingsEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Use testutil for consistent test configuration
	cfg := testutil.NewConfigBuilder().Build()

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	netMgr, _ := network.NewManager("")
	server := NewServer(cfg, configPath, "", netMgr, false, nil, nil, nil)

	ts := httptest.NewServer(server.mux)
	defer ts.Close()

	t.Run("GetHealthChecksSettings", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/health-checks/settings")
		if err != nil {
			t.Fatalf("GET /api/health-checks/settings failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var settings TestsSettingsResponse
		if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
			t.Errorf("Failed to decode health checks settings: %v", err)
		}
	})

	t.Run("UpdateHealthChecksSettings", func(t *testing.T) {
		// Issue #605 fixed: config.Save() deadlock resolved by unlocking before Save()
		settings := TestsSettingsResponse{
			DNSHostname: "example.com",
			DNSServers:  []DNSServerResponse{{Address: "8.8.8.8", Enabled: true}},
			PingTargets: []PingTargetResponse{{Name: "Google", Host: "8.8.8.8", Enabled: true}},
		}

		body, _ := json.Marshal(settings)
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/health-checks/settings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("PUT /api/health-checks/settings failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}
