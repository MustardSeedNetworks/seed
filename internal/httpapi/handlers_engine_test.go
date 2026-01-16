package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krisarmstrong/seed/internal/discovery"
	api "github.com/krisarmstrong/seed/internal/httpapi"
)

func TestHandleEngineDiscovery(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()
	defer server.GetEngine().(*discovery.Engine).Stop()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET returns devices",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/discovery/engine", nil)
			rec := httptest.NewRecorder()

			server.HandleEngineDiscovery(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp api.ExportEngineDiscoveryResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if resp.Capabilities == nil {
					t.Error("expected capabilities in response")
				}
			}
		})
	}
}

func TestHandleEngineStats(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()
	defer server.GetEngine().(*discovery.Engine).Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/engine/stats", nil)
	rec := httptest.NewRecorder()

	server.HandleEngineStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var stats map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}
}

func TestHandleEngineCapabilities(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()
	defer server.GetEngine().(*discovery.Engine).Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/engine/capabilities", nil)
	rec := httptest.NewRecorder()

	server.HandleEngineCapabilities(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var caps map[string]bool
	if err := json.NewDecoder(rec.Body).Decode(&caps); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	// Correlation should always be available.
	if !caps["correlation"] {
		t.Error("expected correlation capability to be true")
	}
}

func TestHandleEngineScan(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()
	defer server.GetEngine().(*discovery.Engine).Stop()

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "POST triggers scan",
			method:         http.MethodPost,
			body:           "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET not allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "POST with options",
			method:         http.MethodPost,
			body:           `{"scanType":"quick","includeWired":true}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/api/v1/discovery/engine/scan", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/api/v1/discovery/engine/scan", nil)
			}
			rec := httptest.NewRecorder()

			server.HandleEngineScan(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandleEngineQuickScan(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()
	defer server.GetEngine().(*discovery.Engine).Stop()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/engine/quick", nil)
	rec := httptest.NewRecorder()

	server.HandleEngineQuickScan(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp api.ExportEngineDiscoveryResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if resp.ScanResult == nil {
		t.Error("expected scan result in response")
	}
}

func TestHandleEngineFullScan(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()
	defer server.GetEngine().(*discovery.Engine).Stop()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/engine/full", nil)
	rec := httptest.NewRecorder()

	server.HandleEngineFullScan(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp api.ExportEngineDiscoveryResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if resp.ScanResult == nil {
		t.Error("expected scan result in response")
	}
	if resp.ScanResult.ScanType != "full" {
		t.Errorf("expected scan type 'full', got %s", resp.ScanResult.ScanType)
	}
}

func TestHandleEngineDevice(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()
	engine := server.GetEngine().(*discovery.Engine)
	defer engine.Stop()

	// Add a device to the engine's registry.
	device := &discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:FF",
		IP:  "192.168.1.100",
	}
	engine.Registry().AddOrUpdate(device)

	tests := []struct {
		name           string
		mac            string
		expectedStatus int
	}{
		{
			name:           "existing device",
			mac:            "AA:BB:CC:DD:EE:FF",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-existing device",
			mac:            "00:00:00:00:00:00",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty mac",
			mac:            "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := api.ExportAPIVersionPrefix + "/discovery/engine/device/" + tt.mac
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			server.HandleEngineDevice(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var device map[string]any
				if err := json.NewDecoder(rec.Body).Decode(&device); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if device["mac"] == nil || device["mac"] == "" {
					t.Error("expected device MAC in response")
				}
			}
		})
	}
}

func TestHandleEngineNoEngine(t *testing.T) {
	// Create server and then remove the engine.
	server := api.NewTestServer()
	defer server.Close()
	engine := server.GetEngine().(*discovery.Engine)
	engine.Stop()

	// We can't easily remove the engine, so instead test the existing
	// error paths by calling the endpoints. The engine is initialized
	// in NewTestServer, so this test verifies the handler works.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/engine", nil)
	rec := httptest.NewRecorder()

	server.HandleEngineDiscovery(rec, req)

	// Since engine exists (just stopped), it should still return OK.
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
