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

// TestVulnerabilitySettingsRoundTrip pins seed#2971: the GET and the PUT share
// one camelCase body, so what the settings drawer reads it can write back,
// autoScan included.
func TestVulnerabilitySettingsRoundTrip(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	s := &Server{config: cfg, configPath: filepath.Join(t.TempDir(), "config.yaml")}
	s.securitySettings = app.NewSecuritySettings(s.config, s.configPath, nil)

	const body = `{"enabled":true,"cveDatabase":"local","nvdApiKey":"k","updateInterval":3600,` +
		`"severityThreshold":"high","maxConcurrent":3,"autoScan":false}`

	w := httptest.NewRecorder()
	s.handleVulnerabilitySettings(w, httptest.NewRequest(http.MethodPut,
		"/api/v1/security/vulnerabilities/settings", strings.NewReader(body)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.False(t, cfg.Security.VulnerabilityScanning.AutoScan, "autoScan was not written")

	w = httptest.NewRecorder()
	s.handleVulnerabilitySettings(w, httptest.NewRequest(http.MethodGet,
		"/api/v1/security/vulnerabilities/settings", http.NoBody))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, body, w.Body.String())
}

// TestVulnerabilitySettingsRejectsConfigKeys keeps the file's snake_case keys
// off the wire in both directions.
func TestVulnerabilitySettingsRejectsConfigKeys(t *testing.T) {
	t.Parallel()
	s := &Server{config: config.DefaultConfig(), configPath: filepath.Join(t.TempDir(), "config.yaml")}
	s.securitySettings = app.NewSecuritySettings(s.config, s.configPath, nil)

	w := httptest.NewRecorder()
	s.handleVulnerabilitySettings(w, httptest.NewRequest(http.MethodPut,
		"/api/v1/security/vulnerabilities/settings", strings.NewReader(`{"auto_scan":true}`)))
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}
