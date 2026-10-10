package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

// These pin the wire shape of handlers whose responses the OpenAPI document
// describes with a registered DTO (S6-4, #196), so the document and the bytes
// cannot drift apart.

func serve(t *testing.T, h http.HandlerFunc, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, target, http.NoBody))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	return w
}

func topLevelKeys(t *testing.T, body []byte) []string {
	t.Helper()
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &m))
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func TestHealthResponseShape(t *testing.T) {
	t.Parallel()
	s := &Server{startTime: time.Now().Add(-time.Minute)}
	w := serve(t, s.handleHealth, "/api/v1/health")

	var got struct {
		Status string  `json:"status"`
		Uptime float64 `json:"uptime"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, []string{"status", "uptime"}, topLevelKeys(t, w.Body.Bytes()))
	require.Equal(t, "ok", got.Status)
	require.GreaterOrEqual(t, got.Uptime, 60.0)
}

func TestSSOSettingsResponseShape(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cfg.Auth.SSO.Providers = []config.SSOProviderConfig{
		{Name: "google", Enabled: true, ClientID: "id", ClientSecret: "secret"},
		{Name: "github", Enabled: true},
	}
	s := &Server{config: cfg}
	w := serve(t, s.handleSSOSettings, "/api/v1/sso/settings")

	require.JSONEq(t,
		`{"providers":[{"name":"google","enabled":true},{"name":"github","enabled":false}]}`,
		w.Body.String())
}

type statusRadio struct{ troubleshooting.Hardware }

func (statusRadio) ManagerAvailable() bool { return true }
func (statusRadio) IsWireless() bool       { return true }

type wirelessNames []string

func (n wirelessNames) WirelessInterfaceNames() []string { return n }

type fixedWiFiInterface string

func (f fixedWiFiInterface) ResolvedWiFiInterface() string { return string(f) }
func (fixedWiFiInterface) SaveWiFiInterface(string) error  { return nil }

func TestWiFiStatusResponseShape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		adapters wirelessNames
		want     string
	}{
		{
			name:     "ready",
			adapters: wirelessNames{"wlan0"},
			want: `{"status":"ready","message":"Wireless adapter ready for scanning.",` +
				`"currentInterface":"wlan0","isWireless":true,"availableAdapters":["wlan0"],"canScan":true}`,
		},
		{
			name: "no adapter",
			want: `{"status":"unavailable",` +
				`"message":"No wireless adapter detected. Connect a WiFi adapter to perform surveys.",` +
				`"currentInterface":"wlan0","isWireless":true,"availableAdapters":[],"canScan":false}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{wifiManagement: troubleshooting.NewManagement(
				statusRadio{}, tt.adapters, fixedWiFiInterface("wlan0"))}
			w := serve(t, s.handleWiFiStatus, "/api/v1/wifi/wifi/status")
			require.JSONEq(t, tt.want, w.Body.String())
		})
	}
}

func TestVulnerabilityListResponseShapes(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	scanner := newStoredVulnScanner(t, db)
	_, err := scanner.ScanDevice(context.Background(), &discovery.DiscoveredDevice{
		IP: "10.20.30.50", MAC: "02:00:00:00:00:05", Vendor: "Linux", OSGuess: "Linux 5.4",
	})
	require.NoError(t, err)
	s := &Server{vulnScan: scanner}

	results := serve(t, s.handleVulnerabilityResults, "/api/v1/security/vulnerabilities/results")
	require.Equal(t, []string{"count", "results"}, topLevelKeys(t, results.Body.Bytes()))
	var r struct {
		Count   int                                `json:"count"`
		Results []*discovery.DeviceVulnerabilities `json:"results"`
	}
	require.NoError(t, json.Unmarshal(results.Body.Bytes(), &r))
	require.Equal(t, 1, r.Count)
	require.Len(t, r.Results, 1)
	require.Equal(t, "10.20.30.50", r.Results[0].DeviceIP)

	triaged, _, _ := newTriageServer(t)
	findings := serve(t, triaged.handleVulnFindings, "/api/v1/security/vulnerabilities/findings")
	require.Equal(t, []string{"count", "findings"}, topLevelKeys(t, findings.Body.Bytes()))
	var f struct {
		Count    int                   `json:"count"`
		Findings []VulnFindingResponse `json:"findings"`
	}
	require.NoError(t, json.Unmarshal(findings.Body.Bytes(), &f))
	require.Equal(t, 1, f.Count)
	require.Len(t, f.Findings, 1)
}
