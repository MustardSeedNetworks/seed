package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

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
			defer server.Close()
			server.SetConfigPath("/tmp/test-config.json")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
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
			defer server.Close()
			server.SetConfigPath("/tmp/test-config.json")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
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
			defer server.Close()
			server.SetConfigPath("/tmp/test-config.json")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
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
			defer server.Close()
			server.SetConfigPath("/tmp/test-config.json")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
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
			defer server.Close()
			server.SetConfigPath("/tmp/test-config.json")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
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
			defer server.Close()
			server.SetConfigPath("/tmp/test-config.json")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
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
			defer server.Close()
			server.SetConfigPath("/tmp/test-config.json")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
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
	defer server.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.HandleSettings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestBuildThresholdSettings tests that threshold settings builder returns expected structure.
func TestBuildThresholdSettings(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", http.NoBody)
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
			defer server.Close()
			server.SetConfigPath("/tmp/test-config.json")

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
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
	defer server.Close()

	// Set thresholds in config
	cfg := server.Config()
	cfg.Thresholds.DNS.Warning = 100 * time.Millisecond
	cfg.Thresholds.DNS.Critical = 500 * time.Millisecond

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", http.NoBody)
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
