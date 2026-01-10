package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// TestHandleSettingsDefaultsGET tests the settings defaults endpoint.
func TestHandleSettingsDefaultsGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/settings/defaults", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleSettingsDefaults(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify JSON response
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify content-type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
	}
}

// TestHandleSettingsDefaultsMethodNotAllowed tests non-GET methods.
func TestHandleSettingsDefaultsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/settings/defaults", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleSettingsDefaults(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleSettingsGET tests the settings GET endpoint.
func TestHandleSettingsGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/settings", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify expected fields are present
	expectedFields := []string{"interface", "vlan", "ip", "thresholds", "healthChecks", "speedtest", "iperf"}
	for _, field := range expectedFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("Expected field %s in response", field)
		}
	}
}

// TestHandleSettingsMethodNotAllowed tests non-GET/PUT methods.
func TestHandleSettingsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/settings", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleSettings(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleLinkSettingsGET tests the link settings GET endpoint.
func TestHandleLinkSettingsGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/settings/link", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleLinkSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp config.ProfileLinkSettings
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Default mode should be "auto"
	if resp.Mode != "auto" {
		t.Errorf("Expected default mode 'auto', got %q", resp.Mode)
	}
}

// TestHandleLinkSettingsMethodNotAllowed tests non-GET/PUT methods.
func TestHandleLinkSettingsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/settings/link", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleLinkSettings(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleLinkSettingsPUTInvalidMode tests link settings with invalid mode.
func TestHandleLinkSettingsPUTInvalidMode(t *testing.T) {
	server := api.NewTestServer()

	reqBody := config.ProfileLinkSettings{
		Mode: "invalid-mode",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/link", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.HandleLinkSettings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHandleLinkSettingsPUTValidModes tests link settings with valid modes.
func TestHandleLinkSettingsPUTValidModes(t *testing.T) {
	validModes := []string{"auto", "10/half", "10/full", "100/half", "100/full", "1000/full"}

	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			server := api.NewTestServer()

			reqBody := config.ProfileLinkSettings{
				Mode: mode,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPut, "/api/settings/link", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleLinkSettings(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d for mode %s, got %d", http.StatusOK, mode, w.Code)
			}
		})
	}
}

// TestHandleCableTestSettingsGET tests the cable test settings GET endpoint.
func TestHandleCableTestSettingsGET(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/settings/cable", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleCableTestSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp config.ProfileCableTestSettings
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Default enabled should be true
	if !resp.Enabled {
		t.Error("Expected default enabled to be true")
	}
}

// TestHandleCableTestSettingsMethodNotAllowed tests non-GET/PUT methods.
func TestHandleCableTestSettingsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/settings/cable", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleCableTestSettings(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleCableTestSettingsPUT tests the cable test settings PUT endpoint.
func TestHandleCableTestSettingsPUT(t *testing.T) {
	server := api.NewTestServer()

	reqBody := config.ProfileCableTestSettings{
		Enabled: false,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/cable", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.HandleCableTestSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// settingsUpdateTestCase defines a test case for settings update validation.
type settingsUpdateTestCase struct {
	name           string
	payload        map[string]any
	expectedStatus int
}

// TestApplyThresholdUpdates tests threshold update validation.
func TestApplyThresholdUpdates(t *testing.T) {
	tests := []settingsUpdateTestCase{
		{
			name: "valid dns thresholds",
			payload: map[string]any{
				"thresholds": map[string]any{"dns": map[string]any{"good": 50.0, "warning": 100.0}},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid gateway thresholds",
			payload: map[string]any{
				"thresholds": map[string]any{"gateway": map[string]any{"good": 20.0, "warning": 50.0}},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid wifi thresholds",
			payload: map[string]any{
				"thresholds": map[string]any{"wifi": map[string]any{"good": -60.0, "warning": -70.0}},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid thresholds type",
			payload:        map[string]any{"thresholds": "not-an-object"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid dns type",
			payload:        map[string]any{"thresholds": map[string]any{"dns": "not-an-object"}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid gateway type",
			payload:        map[string]any{"thresholds": map[string]any{"gateway": "not-an-object"}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid wifi type",
			payload:        map[string]any{"thresholds": map[string]any{"wifi": "not-an-object"}},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			server.SetConfigPath("/tmp/test-config.yaml")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleSettings(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestApplyHealthChecksUpdates tests health check update validation.
func TestApplyHealthChecksUpdates(t *testing.T) {
	tests := []settingsUpdateTestCase{
		{
			name: "valid health checks",
			payload: map[string]any{
				"healthChecks": map[string]any{"runPerformance": true, "runSpeedtest": false},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid healthChecks type",
			payload:        map[string]any{"healthChecks": "not-an-object"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid runPerformance type",
			payload:        map[string]any{"healthChecks": map[string]any{"runPerformance": "not-a-bool"}},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			server.SetConfigPath("/tmp/test-config.yaml")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleSettings(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestApplySpeedtestUpdates tests speedtest update validation.
func TestApplySpeedtestUpdates(t *testing.T) {
	tests := []settingsUpdateTestCase{
		{
			name:           "valid speedtest settings",
			payload:        map[string]any{"speedtest": map[string]any{"serverId": "12345", "autoRunOnLink": true}},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid speedtest type",
			payload:        map[string]any{"speedtest": "not-an-object"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid serverId type",
			payload:        map[string]any{"speedtest": map[string]any{"serverId": 12345}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid autoRunOnLink type",
			payload:        map[string]any{"speedtest": map[string]any{"autoRunOnLink": "not-a-bool"}},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			server.SetConfigPath("/tmp/test-config.yaml")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleSettings(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestApplyIperfUpdates tests iperf update validation.
func TestApplyIperfUpdates(t *testing.T) {
	tests := []settingsUpdateTestCase{
		{
			name:           "valid iperf settings",
			payload:        map[string]any{"iperf": map[string]any{"server": "192.168.1.100", "port": 5201.0}},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid iperf type",
			payload:        map[string]any{"iperf": "not-an-object"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid server type",
			payload:        map[string]any{"iperf": map[string]any{"server": 12345}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid port - out of range",
			payload:        map[string]any{"iperf": map[string]any{"port": 70000.0}},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			server.SetConfigPath("/tmp/test-config.yaml")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleSettings(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestApplyDisplayOptionsUpdates tests display options update validation.
func TestApplyDisplayOptionsUpdates(t *testing.T) {
	tests := []settingsUpdateTestCase{
		{
			name: "valid display options",
			payload: map[string]any{
				"displayOptions": map[string]any{"showPublicIP": true, "unitSystem": "metric"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid displayOptions type",
			payload:        map[string]any{"displayOptions": "not-an-object"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid showPublicIP type",
			payload:        map[string]any{"displayOptions": map[string]any{"showPublicIP": "not-a-bool"}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid unitSystem type",
			payload:        map[string]any{"displayOptions": map[string]any{"unitSystem": 123}},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			server.SetConfigPath("/tmp/test-config.yaml")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleSettings(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestApplyHTTPTimingThresholds tests HTTP timing threshold update validation.
func TestApplyHTTPTimingThresholds(t *testing.T) {
	tests := []settingsUpdateTestCase{
		{
			name: "valid http timing thresholds",
			payload: map[string]any{
				"thresholds": map[string]any{
					"httpTimings": map[string]any{
						"dns":  map[string]any{"good": 50.0, "warning": 100.0},
						"tcp":  map[string]any{"good": 50.0, "warning": 100.0},
						"tls":  map[string]any{"good": 100.0, "warning": 200.0},
						"ttfb": map[string]any{"good": 100.0, "warning": 300.0},
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid httpTimings type",
			payload: map[string]any{
				"thresholds": map[string]any{
					"httpTimings": "not-an-object",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid dns timing type",
			payload: map[string]any{
				"thresholds": map[string]any{
					"httpTimings": map[string]any{
						"dns": "not-an-object",
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			server.SetConfigPath("/tmp/test-config.yaml")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleSettings(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestApplyCustomTestThresholds tests custom test threshold update validation.
func TestApplyCustomTestThresholds(t *testing.T) {
	tests := []settingsUpdateTestCase{
		{
			name: "valid custom ping thresholds",
			payload: map[string]any{
				"thresholds": map[string]any{
					"customPing": map[string]any{"good": 50.0, "warning": 100.0},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid custom tcp thresholds",
			payload: map[string]any{
				"thresholds": map[string]any{
					"customTcp": map[string]any{"good": 50.0, "warning": 100.0},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid custom http thresholds",
			payload: map[string]any{
				"thresholds": map[string]any{
					"customHttp": map[string]any{"good": 200.0, "warning": 500.0},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid customPing type",
			payload: map[string]any{
				"thresholds": map[string]any{
					"customPing": "not-an-object",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			server.SetConfigPath("/tmp/test-config.yaml")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleSettings(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestSettingsUpdateInvalidJSON tests settings update with invalid JSON.
func TestSettingsUpdateInvalidJSON(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.HandleSettings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestBuildCardSettings tests that card settings builder returns expected structure.
func TestBuildCardSettings(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/settings", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleSettings(w, req)

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	cardSettings, ok := resp["cardSettings"].(map[string]any)
	if !ok {
		t.Fatal("Expected cardSettings in response")
	}

	expectedCards := []string{"link", "switch", "vlan", "network", "gateway", "dns"}
	for _, card := range expectedCards {
		if _, cardOK := cardSettings[card]; !cardOK {
			t.Errorf("Expected card %s in cardSettings", card)
		}
	}
}

// TestBuildThresholdSettings tests that threshold settings builder returns expected structure.
func TestBuildThresholdSettings(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/settings", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleSettings(w, req)

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	thresholds, ok := resp["thresholds"].(map[string]any)
	if !ok {
		t.Fatal("Expected thresholds in response")
	}

	expectedKeys := []string{"dns", "gateway", "wifi", "customPing", "customTcp", "customHttp", "httpTimings"}
	for _, key := range expectedKeys {
		if _, thresholdOK := thresholds[key]; !thresholdOK {
			t.Errorf("Expected threshold key %s", key)
		}
	}

	// Verify dns threshold structure
	dnsThreshold, ok := thresholds["dns"].(map[string]any)
	if !ok {
		t.Fatal("Expected dns threshold to be an object")
	}

	if _, goodOK := dnsThreshold["good"]; !goodOK {
		t.Error("Expected 'good' key in dns threshold")
	}
	if _, warningOK := dnsThreshold["warning"]; !warningOK {
		t.Error("Expected 'warning' key in dns threshold")
	}
}

// TestApplyFABOptionsUpdates tests FAB options update validation.
func TestApplyFABOptionsUpdates(t *testing.T) {
	tests := []settingsUpdateTestCase{
		{
			name: "valid fab options",
			payload: map[string]any{
				"fabOptions": map[string]any{
					"runLink":             true,
					"runSwitch":           true,
					"runVLAN":             false,
					"autoScanOnLink":      true,
					"runHealthChecks":     true,
					"runNetworkDiscovery": true,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid fabOptions type",
			payload:        map[string]any{"fabOptions": "not-an-object"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid runLink type",
			payload: map[string]any{
				"fabOptions": map[string]any{"runLink": "not-a-bool"},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			server.SetConfigPath("/tmp/test-config.yaml")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.HandleSettings(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestSettingsThresholdDurationConversion tests that thresholds are properly converted to milliseconds.
func TestSettingsThresholdDurationConversion(t *testing.T) {
	server := api.NewTestServer()

	// Set thresholds in config
	cfg := server.Config()
	cfg.Thresholds.DNS.Warning = 100 * time.Millisecond
	cfg.Thresholds.DNS.Critical = 500 * time.Millisecond

	req := httptest.NewRequest(http.MethodGet, "/api/settings", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleSettings(w, req)

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	thresholds := resp["thresholds"].(map[string]any)
	dnsThreshold := thresholds["dns"].(map[string]any)

	// Verify values are in milliseconds
	good := dnsThreshold["good"].(float64)
	warning := dnsThreshold["warning"].(float64)

	if good != 100 {
		t.Errorf("Expected dns.good to be 100, got %v", good)
	}
	if warning != 500 {
		t.Errorf("Expected dns.warning to be 500, got %v", warning)
	}
}
