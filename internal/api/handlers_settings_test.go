package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

// TestHandleSettingsDefaultsGET tests the settings defaults endpoint.
func TestHandleSettingsDefaultsGET(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/defaults", http.NoBody)
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

// TestHandleSettingsGET tests the settings GET endpoint.
func TestHandleSettingsGET(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", http.NoBody)
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
	expectedFields := []string{"vlan", "ip", "thresholds", "healthChecks", "speedtest", "iperf"}
	for _, field := range expectedFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("Expected field %s in response", field)
		}
	}
}

// TestHandleSettingsMethodNotAllowed tests non-GET/PUT methods.
func TestHandleSettingsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/settings", http.NoBody)
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
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/link", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleLinkSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp config.LinkConfig
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
	defer server.Close()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/settings/link", http.NoBody)
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
	defer server.Close()

	reqBody := config.LinkConfig{
		Mode: "invalid-mode",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/link", bytes.NewReader(body))
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
			defer server.Close()

			reqBody := config.LinkConfig{
				Mode: mode,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/link", bytes.NewReader(body))
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
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/cable", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleCableTestSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp config.CableTestConfig
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response decoded successfully (structure is correct)
	// Default Enabled is false per Go zero value - test just validates structure
	_ = resp.Enabled // Access field to verify it exists
}

// TestHandleCableTestSettingsMethodNotAllowed tests non-GET/PUT methods.
func TestHandleCableTestSettingsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	methods := []string{http.MethodPost, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/settings/cable", http.NoBody)
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
	defer server.Close()

	reqBody := config.CableTestConfig{
		Enabled: false,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/cable", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.HandleCableTestSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestHandleSettingsETagReflectsConfig is the regression for the pre-existing
// getSettings mismatch: GET must report the live config, not hardcoded values.
// Under the default config RunIperf is true, so the response healthChecks.runIperf
// must be true (it was previously hardcoded false).
func TestHandleSettingsETagReflectsConfig(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", http.NoBody)
	w := httptest.NewRecorder()
	server.HandleSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET settings: expected 200, got %d", w.Code)
	}

	// GET emits a quoted, non-empty ETag (optimistic-concurrency token).
	etag := w.Header().Get("ETag")
	if len(etag) < 3 || etag[0] != '"' || etag[len(etag)-1] != '"' {
		t.Fatalf("GET settings: expected a quoted ETag, got %q", etag)
	}
	if want := server.Config().SettingsETag(); etag != want {
		t.Fatalf("GET ETag %q != config SettingsETag %q", etag, want)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	health := resp["healthChecks"].(map[string]any)
	if runIperf, ok := health["runIperf"].(bool); !ok || !runIperf {
		t.Errorf("healthChecks.runIperf should reflect config (true), got %v", health["runIperf"])
	}
}

// TestHandleSettingsIfMatch covers optimistic concurrency on PUT /settings:
// a stale If-Match is rejected with 412 and leaves config untouched; the current
// token writes through and returns a fresh (different) ETag; an absent header is
// unconditional (prior behavior).
func TestHandleSettingsIfMatch(t *testing.T) {
	body := []byte(`{"thresholds":{"dns":{"good":777}}}`)

	t.Run("stale If-Match -> 412, config untouched", func(t *testing.T) {
		server := api.NewTestServer()
		defer server.Close()

		before := server.Config().Thresholds.DNS.Warning // "good" maps to DNS.Warning

		req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
		req.Header.Set("If-Match", `"stale-token"`)
		w := httptest.NewRecorder()
		server.HandleSettings(w, req)

		if w.Code != http.StatusPreconditionFailed {
			t.Fatalf("expected 412, got %d", w.Code)
		}
		if after := server.Config().Thresholds.DNS.Warning; after != before {
			t.Errorf("config mutated on 412: before=%v after=%v", before, after)
		}
	})

	t.Run("current If-Match -> 200 + fresh ETag", func(t *testing.T) {
		server := api.NewTestServer()
		defer server.Close()

		current := server.Config().SettingsETag()

		req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
		req.Header.Set("If-Match", current)
		w := httptest.NewRecorder()
		server.HandleSettings(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body %s)", w.Code, w.Body.String())
		}
		if got := server.Config().Thresholds.DNS.Warning; got != 777*time.Millisecond {
			t.Errorf("threshold not applied: got %v", got)
		}
		fresh := w.Header().Get("ETag")
		if fresh == "" || fresh == current {
			t.Errorf("expected a fresh ETag different from %q, got %q", current, fresh)
		}
	})

	t.Run("absent If-Match -> 200 unconditional", func(t *testing.T) {
		server := api.NewTestServer()
		defer server.Close()

		req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
		w := httptest.NewRecorder()
		server.HandleSettings(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if got := server.Config().Thresholds.DNS.Warning; got != 777*time.Millisecond {
			t.Errorf("threshold not applied: got %v", got)
		}
	})
}
