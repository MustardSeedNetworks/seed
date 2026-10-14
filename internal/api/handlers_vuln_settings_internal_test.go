package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

func newVulnSettingsServer(t *testing.T) (*Server, *config.Config) {
	t.Helper()
	cfg := config.DefaultConfig()
	s := &Server{config: cfg, configPath: filepath.Join(t.TempDir(), "config.yaml")}
	s.securitySettings = app.NewSecuritySettings(s.config, s.configPath, nil)
	return s, cfg
}

func vulnSettingsRequest(t *testing.T, s *Server, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleVulnerabilitySettings(w, httptest.NewRequest(method,
		"/api/v1/security/vulnerabilities/settings", strings.NewReader(body)))
	return w
}

// TestVulnerabilitySettingsRoundTrip pins seed#2971: the GET and the PUT share
// one camelCase shape, so what the settings drawer reads it can write back,
// autoScan included.
func TestVulnerabilitySettingsRoundTrip(t *testing.T) {
	t.Parallel()
	s, cfg := newVulnSettingsServer(t)

	w := vulnSettingsRequest(t, s, http.MethodPut,
		`{"enabled":true,"cveDatabase":"local","nvdApiKey":"k","updateInterval":3600,`+
			`"severityThreshold":"high","maxConcurrent":3,"autoScan":false}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.False(t, cfg.Security.VulnerabilityScanning.AutoScan, "autoScan was not written")

	w = vulnSettingsRequest(t, s, http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"enabled":true,"cveDatabase":"local","hasNvdApiKey":true,"updateInterval":3600,`+
		`"severityThreshold":"high","maxConcurrent":3,"autoScan":false}`, w.Body.String())
}

// TestVulnerabilitySettingsNVDKey pins seed#3407: the stored NVD API key never
// leaves the server, a save without a key keeps it, and only an explicit clear
// removes it.
func TestVulnerabilitySettingsNVDKey(t *testing.T) {
	t.Parallel()
	const rest = `"enabled":true,"cveDatabase":"nvd","updateInterval":3600,` +
		`"severityThreshold":"high","maxConcurrent":3,"autoScan":true`
	for _, tc := range []struct {
		name     string
		body     string
		wantCode int
		wantKey  string
	}{
		{"absent key keeps the stored one", `{` + rest + `}`, http.StatusOK, "stored-key"},
		{"empty key keeps the stored one", `{"nvdApiKey":"",` + rest + `}`, http.StatusOK, "stored-key"},
		{"new key replaces it", `{"nvdApiKey":"new-key",` + rest + `}`, http.StatusOK, "new-key"},
		{"clear removes it", `{"clearNvdApiKey":true,` + rest + `}`, http.StatusOK, ""},
		{
			"key and clear together are refused", `{"nvdApiKey":"new-key","clearNvdApiKey":true,` + rest + `}`,
			http.StatusBadRequest, "stored-key",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, cfg := newVulnSettingsServer(t)
			cfg.Security.VulnerabilityScanning.NVDAPIKey = "stored-key"

			w := vulnSettingsRequest(t, s, http.MethodGet, "")
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.NotContains(t, w.Body.String(), "stored-key")
			require.Contains(t, w.Body.String(), `"hasNvdApiKey":true`)

			w = vulnSettingsRequest(t, s, http.MethodPut, tc.body)
			require.Equal(t, tc.wantCode, w.Code, w.Body.String())
			require.Equal(t, tc.wantKey, cfg.Security.VulnerabilityScanning.NVDAPIKey)
		})
	}
}

// TestVulnerabilitySettingsRejectsConfigKeys keeps the file's snake_case keys
// off the wire in both directions.
func TestVulnerabilitySettingsRejectsConfigKeys(t *testing.T) {
	t.Parallel()
	s, _ := newVulnSettingsServer(t)

	w := vulnSettingsRequest(t, s, http.MethodPut, `{"auto_scan":true}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}
