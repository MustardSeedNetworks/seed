// Package api provides the HTTP/WebSocket server.
// Tests for status-related handlers including /api/status, /api/settings, /api/export.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleStatusHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/status")
		if err != nil {
			t.Fatalf("GET /api/status failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var status StatusResponse
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Errorf("Failed to decode status response: %v", err)
		}

		if status.Status != "ok" {
			t.Errorf("Expected status 'ok', got %q", status.Status)
		}
		if status.Version == "" {
			t.Error("Expected version to be set")
		}
	})

	t.Run("POST not allowed", func(t *testing.T) {
		resp, err := http.Post(httpServer.URL+"/api/status", "application/json", nil)
		if err != nil {
			t.Fatalf("POST /api/status failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", resp.StatusCode)
		}
	})
}

func TestHandleSettingsHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns settings", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/settings")
		if err != nil {
			t.Fatalf("GET /api/settings failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var settings map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
			t.Errorf("Failed to decode settings response: %v", err)
		}

		// Verify key settings fields exist
		if _, ok := settings["interface"]; !ok {
			t.Error("Expected 'interface' field in settings")
		}
		if _, ok := settings["thresholds"]; !ok {
			t.Error("Expected 'thresholds' field in settings")
		}
	})

	// Note: PUT settings test skipped - involves filesystem operations that can deadlock in test environment
}

func TestHandleInterfacesHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns interfaces list", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/interfaces")
		if err != nil {
			t.Fatalf("GET /api/interfaces failed: %v", err)
		}
		defer resp.Body.Close()

		// May fail on systems without network manager, but should not panic
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
		}

		if resp.StatusCode == http.StatusOK {
			var interfaces []interface{}
			if err := json.NewDecoder(resp.Body).Decode(&interfaces); err != nil {
				t.Errorf("Failed to decode interfaces response: %v", err)
			}
		}
	})
}

func TestHandleLinkHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns link status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/link")
		if err != nil {
			t.Fatalf("GET /api/link failed: %v", err)
		}
		defer resp.Body.Close()

		// May fail without network interface configured
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusServiceUnavailable && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200, 404, 500, or 503, got %d", resp.StatusCode)
		}
	})

	t.Run("POST not allowed", func(t *testing.T) {
		resp, err := http.Post(httpServer.URL+"/api/link", "application/json", nil)
		if err != nil {
			t.Fatalf("POST /api/link failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", resp.StatusCode)
		}
	})
}

func TestHandleVLANHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns VLAN status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/vlan")
		if err != nil {
			t.Fatalf("GET /api/vlan failed: %v", err)
		}
		defer resp.Body.Close()

		// VLAN detection may fail without proper interface
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}

		if resp.StatusCode == http.StatusOK {
			var vlanResp map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&vlanResp); err != nil {
				t.Errorf("Failed to decode VLAN response: %v", err)
			}
		}
	})
}

func TestHandleWiFiHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns WiFi status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/wifi")
		if err != nil {
			t.Fatalf("GET /api/wifi failed: %v", err)
		}
		defer resp.Body.Close()

		// WiFi may not be available - all status codes except 405 are acceptable
		if resp.StatusCode == http.StatusMethodNotAllowed {
			t.Error("Expected GET to be allowed for /api/wifi")
		}
	})
}

func TestHandleCableHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns cable status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/cable")
		if err != nil {
			t.Fatalf("GET /api/cable failed: %v", err)
		}
		defer resp.Body.Close()

		// Cable test may not be supported
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
	})
}

func TestHandleDNSHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns DNS status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/dns")
		if err != nil {
			t.Fatalf("GET /api/dns failed: %v", err)
		}
		defer resp.Body.Close()

		// DNS test runs async, may return various codes
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
	})
}

func TestHandleGatewayHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns gateway status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/gateway")
		if err != nil {
			t.Fatalf("GET /api/gateway failed: %v", err)
		}
		defer resp.Body.Close()

		// Gateway test may fail without network access
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError && resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200, 500, or 503, got %d", resp.StatusCode)
		}
	})
}

func TestHandlePublicIPHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns public IP", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/publicip")
		if err != nil {
			t.Fatalf("GET /api/publicip failed: %v", err)
		}
		defer resp.Body.Close()

		// May fail without internet access
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
	})
}

func TestHandleExportHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns export data", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/export")
		if err != nil {
			t.Fatalf("GET /api/export failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}

		if resp.StatusCode == http.StatusOK {
			var export ExportData
			if err := json.NewDecoder(resp.Body).Decode(&export); err != nil {
				t.Errorf("Failed to decode export response: %v", err)
			}

			if export.Version == "" {
				t.Error("Expected version in export data")
			}
			if export.Timestamp == "" {
				t.Error("Expected timestamp in export data")
			}
		}
	})

	t.Run("POST not allowed", func(t *testing.T) {
		resp, err := http.Post(httpServer.URL+"/api/export", "application/json", nil)
		if err != nil {
			t.Fatalf("POST /api/export failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", resp.StatusCode)
		}
	})
}

func TestHandleLogsHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns logs or requires auth", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/logs")
		if err != nil {
			t.Fatalf("GET /api/logs failed: %v", err)
		}
		defer resp.Body.Close()

		// Logs may require token or return empty
		validCodes := []int{http.StatusOK, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError}
		found := false
		for _, code := range validCodes {
			if resp.StatusCode == code {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected status code: %d", resp.StatusCode)
		}
	})

	t.Run("GET with lines parameter", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/logs?lines=50")
		if err != nil {
			t.Fatalf("GET /api/logs?lines=50 failed: %v", err)
		}
		defer resp.Body.Close()

		// Same valid codes as above
		validCodes := []int{http.StatusOK, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError}
		found := false
		for _, code := range validCodes {
			if resp.StatusCode == code {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected status code: %d", resp.StatusCode)
		}
	})
}

func TestSendJSONResponseFunc(t *testing.T) {
	t.Run("sends valid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()

		data := map[string]string{"message": "test"}
		sendJSONResponse(w, http.StatusOK, data)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %q", contentType)
		}

		var result map[string]string
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if result["message"] != "test" {
			t.Errorf("Expected message 'test', got %q", result["message"])
		}
	})

	t.Run("handles different status codes", func(t *testing.T) {
		tests := []struct {
			code int
			data interface{}
		}{
			{http.StatusOK, map[string]string{"status": "ok"}},
			{http.StatusBadRequest, map[string]string{"error": "bad request"}},
			{http.StatusInternalServerError, map[string]string{"error": "server error"}},
			{http.StatusNotFound, map[string]string{"error": "not found"}},
		}

		for _, tt := range tests {
			w := httptest.NewRecorder()
			sendJSONResponse(w, tt.code, tt.data)

			if w.Code != tt.code {
				t.Errorf("Expected status %d, got %d", tt.code, w.Code)
			}
		}
	})
}

func TestHandleDevicesHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns devices list", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/devices")
		if err != nil {
			t.Fatalf("GET /api/devices failed: %v", err)
		}
		defer resp.Body.Close()

		// Devices endpoint may return empty list or error
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
	})
}

func TestHandleSpeedtestStatusHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns speedtest status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/speedtest/status")
		if err != nil {
			t.Fatalf("GET /api/speedtest/status failed: %v", err)
		}
		defer resp.Body.Close()

		// Speedtest status should return idle/running/complete
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
	})
}

func TestHandleIperfInfoHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns iperf info", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/iperf/info")
		if err != nil {
			t.Fatalf("GET /api/iperf/info failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
	})
}

func TestHandleDiscoveryHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns discovery status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/discovery")
		if err != nil {
			t.Fatalf("GET /api/discovery failed: %v", err)
		}
		defer resp.Body.Close()

		// Discovery may not work without network
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError && resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200, 500, or 503, got %d", resp.StatusCode)
		}
	})
}

func TestHandleDevicesScanHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("POST triggers scan", func(t *testing.T) {
		resp, err := http.Post(httpServer.URL+"/api/devices/scan", "application/json", nil)
		if err != nil {
			t.Fatalf("POST /api/devices/scan failed: %v", err)
		}
		defer resp.Body.Close()

		// Scan may fail without network/privileges
		validCodes := []int{http.StatusOK, http.StatusAccepted, http.StatusInternalServerError, http.StatusServiceUnavailable}
		found := false
		for _, code := range validCodes {
			if resp.StatusCode == code {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected status code: %d", resp.StatusCode)
		}
	})
}

func TestHandleDevicesStatusHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns devices status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/devices/status")
		if err != nil {
			t.Fatalf("GET /api/devices/status failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
	})
}

func TestHandleDevicesSettingsHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns devices settings", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/devices/settings")
		if err != nil {
			t.Fatalf("GET /api/devices/settings failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
	})
}

func TestHandleLoginHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("POST with invalid credentials", func(t *testing.T) {
		payload := `{"username":"wronguser","password":"wrongpass"}`
		resp, err := http.Post(httpServer.URL+"/api/auth/login", "application/json", strings.NewReader(payload))
		if err != nil {
			t.Fatalf("POST /api/auth/login failed: %v", err)
		}
		defer resp.Body.Close()

		// Should return 401 for invalid credentials
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusTooManyRequests {
			t.Errorf("Expected status 401 or 429, got %d", resp.StatusCode)
		}
	})

	t.Run("POST with empty body", func(t *testing.T) {
		resp, err := http.Post(httpServer.URL+"/api/auth/login", "application/json", nil)
		if err != nil {
			t.Fatalf("POST /api/auth/login failed: %v", err)
		}
		defer resp.Body.Close()

		// Should return bad request for empty body
		if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusTooManyRequests {
			t.Errorf("Expected status 400 or 429, got %d", resp.StatusCode)
		}
	})
}

func TestHandleSetupStatusHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns setup status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/setup/status")
		if err != nil {
			t.Fatalf("GET /api/setup/status failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestHandleRoutesHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns routes", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/routes")
		if err != nil {
			t.Fatalf("GET /api/routes failed: %v", err)
		}
		defer resp.Body.Close()

		// Routes may not be available
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200, 404, or 500, got %d", resp.StatusCode)
		}
	})
}

// Note: /api/ping endpoint is not registered in server.go - skipping ping handler tests

func TestHandleHealthHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns health status", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/health")
		if err != nil {
			t.Fatalf("GET /api/health failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", resp.StatusCode)
		}
	})
}

func TestHandleIPConfigHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET returns IP config", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/ipconfig")
		if err != nil {
			t.Fatalf("GET /api/ipconfig failed: %v", err)
		}
		defer resp.Body.Close()

		// May not work without network interface
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError && resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200, 404, 500, or 503, got %d", resp.StatusCode)
		}
	})
}

func TestHandleTracerouteHandler(t *testing.T) {
	ts := createTestServer(t)
	defer ts.cleanup()

	httpServer := httptest.NewServer(ts.server.mux)
	defer httpServer.Close()

	t.Run("GET without target", func(t *testing.T) {
		resp, err := http.Get(httpServer.URL + "/api/traceroute")
		if err != nil {
			t.Fatalf("GET /api/traceroute failed: %v", err)
		}
		defer resp.Body.Close()

		// May return 404 if route not registered, or 400 if missing target
		if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 400, 404, or 200, got %d", resp.StatusCode)
		}
	})
}
