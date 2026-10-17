package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	enginestatus "github.com/MustardSeedNetworks/seed/internal/engine/status"
	"github.com/MustardSeedNetworks/seed/internal/health/monitoring"
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
		`{"providers":[{"name":"google","enabled":true,"clientId":"id","hasClientSecret":true},`+
			`{"name":"github","enabled":false}]}`,
		w.Body.String())
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

	triaged, _, _, _ := newTriageServer(t)
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

type fixedEngines []engine.Engine

func (fixedEngines) Available() bool            { return true }
func (e fixedEngines) Engines() []engine.Engine { return e }

func TestEnginesResponseShape(t *testing.T) {
	t.Parallel()
	tick := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		engines fixedEngines
		want    string
	}{
		{name: "none", want: `{"count":0,"engines":[]}`},
		{
			name: "reporter and plain",
			engines: fixedEngines{
				&reportingEngine{name: "snmp-poller", status: engine.Status{
					State: engine.StateDegraded, LastTickAt: tick, LastError: "timeout", Inflight: 2,
				}},
				&minimalEngine{name: "retention"},
			},
			want: `{"count":2,"engines":[` +
				`{"name":"snmp-poller","state":"degraded","lastTickAt":"2026-10-10T12:00:00Z","lastError":"timeout","inflight":2},` +
				`{"name":"retention","state":"ok","lastTickAt":"","lastError":"","inflight":0}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{engineStatus: enginestatus.NewService(tt.engines)}
			w := serve(t, s.handleEngines, "/api/v1/engines")
			require.JSONEq(t, tt.want, w.Body.String())
		})
	}
}

func TestDeviceDiscoveryStatusResponseShape(t *testing.T) {
	t.Parallel()
	s := &Server{deviceDisc: enumerate.NewDeviceDiscovery("lo")}
	w := serve(t, s.handleDevicesStatus, "/api/v1/security/devices/status")
	require.Equal(t, []string{"deviceCount", "interface", "lastScan", "localIP", "scanning", "subnet"},
		topLevelKeys(t, w.Body.Bytes()))
}

func TestDevicesResponseShape(t *testing.T) {
	t.Parallel()
	s := &Server{deviceDisc: enumerate.NewDeviceDiscovery("lo")}
	w := serve(t, s.handleDevices, "/api/v1/security/devices")

	var got map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.JSONEq(t, `[]`, string(got["devices"]))
	require.Equal(t, []string{"devices", "status"}, topLevelKeys(t, w.Body.Bytes()))
	require.Equal(t, []string{"deviceCount", "interface", "lastScan", "localIP", "scanning", "subnet"},
		topLevelKeys(t, got["status"]))
}

func TestDiscoveryOptionsResponseShape(t *testing.T) {
	t.Parallel()
	svc := enumerate.NewService(&config.Config{}, enumerate.NewDeviceDiscovery("lo"), nil)
	s := &Server{discoverySvc: svc}
	w := serve(t, s.handleDiscoveryOptions, "/api/v1/security/discovery/options")

	want, err := json.Marshal(map[string]config.DiscoveryOptions{"options": svc.GetOptions()})
	require.NoError(t, err)
	require.JSONEq(t, string(want), w.Body.String())
}

// The service status answered the domain ServiceStatus as it is: the rescan
// interval in nanoseconds, every internal counter, and the last sweep's full
// device lists, on a route the settings page polls every ten seconds.
func TestDiscoveryServiceStatusResponseShape(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cfg.NetworkDiscovery.Timing.RescanInterval = 5 * time.Minute
	s := &Server{discoverySvc: enumerate.NewService(cfg, enumerate.NewDeviceDiscovery("lo"), nil)}
	w := serve(t, s.handleDiscoveryServiceStatus, "/api/v1/security/discovery/service/status")

	require.Equal(t, []string{
		"activeMethods", "degradationStatus", "deviceCount", "interface", "lastScan", "localIP",
		"profilingStatus", "rescanIntervalMs", "running", "scanning", "snmpProbe", "subnet", "subnets",
		"sweepCoverage",
	}, topLevelKeys(t, w.Body.Bytes()))

	var got DiscoveryServiceStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, int64(300_000), got.RescanIntervalMs)
	require.Equal(t, "healthy", got.DegradationStatus.OverallHealth)
	require.Equal(t, []enumerate.SweepCoverage{}, got.SweepCoverage, "an array, not null, before the first sweep")
}

// The port scan sent the domain result, whose scanTime was a Go duration in
// nanoseconds; the answer now carries milliseconds under scanTimeMs.
func TestPortScanResponseShape(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			_, _ = conn.Write([]byte("lab banner\r\n"))
			_ = conn.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port

	w := httptest.NewRecorder()
	body := fmt.Sprintf(`{"target":"127.0.0.1","ports":[%d]}`, port)
	(&Server{}).handlePortScan(w, httptest.NewRequest(http.MethodPost,
		APIVersionPrefix+"/security/discovery/portscan", strings.NewReader(body)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, []string{"ip", "scanTimeMs", "services"}, topLevelKeys(t, w.Body.Bytes()))
	var got PortScanResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Len(t, got.Services, 1)
	require.Equal(t, port, got.Services[0].Port)
	require.Equal(t, discovery.PortOpen, got.Services[0].State)
	require.Less(t, got.ScanTimeMs, int64(10_000))
}

func TestPortScanResponseCarriesMilliseconds(t *testing.T) {
	t.Parallel()
	got := newPortScanResponse(&discovery.PortScanResult{IP: "192.0.2.1", ScanTime: 1500 * time.Millisecond})
	require.Equal(t, int64(1500), got.ScanTimeMs)
}

func TestSubnetWriteAckShape(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	(&Server{}).writeSubnetResult(w,
		httptest.NewRequest(http.MethodDelete, "/api/v1/security/devices/subnets?cidr=10.0.0.0/24", http.NoBody),
		nil, "Subnet deleted")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"status":"success","message":"Subnet deleted"}`, w.Body.String())
}

// TestSettingsWriteAckShape pins the {"status":"updated"} acknowledgement the
// settings writes answer, over a temp config path so each save succeeds.
func TestSettingsWriteAckShape(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Auth.SSO.Providers = []config.SSOProviderConfig{{Name: "google"}}
	s := &Server{config: cfg, configPath: filepath.Join(t.TempDir(), "config.yaml")}
	s.settingsStore = app.NewSettings(s.persistence, s.config)
	s.settingsManagement = app.NewSettingsManagement(s.config, s.configPath,
		func() *alertdelivery.Manager { return nil })
	s.securitySettings = app.NewSecuritySettings(s.config, s.configPath, nil)

	marshal := func(v any) string {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return string(b)
	}
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		target  string
		body    string
	}{
		{"settings", s.handleSettings, "/api/v1/settings", `{}`},
		{"link", s.handleLinkSettings, "/api/v1/settings/link", marshal(s.settingsManagement.Link())},
		{"cable", s.handleCableTestSettings, "/api/v1/settings/cable", marshal(s.settingsManagement.CableTest())},
		{
			"guest audit", s.handleGuestAuditSettings, "/api/v1/security/guest-audit/settings",
			marshal(s.securitySettings.GuestAudit()),
		},
		{
			"vulnerability", s.handleVulnerabilitySettings, "/api/v1/security/vulnerabilities/settings",
			`{"enabled":false}`,
		},
		{"sso", s.handleSSOUpdate, "/api/v1/sso/update", `{"provider":"google","enabled":false}`},
	} {
		w := httptest.NewRecorder()
		tc.handler(w, httptest.NewRequest(http.MethodPut, tc.target, strings.NewReader(tc.body)))
		require.Equal(t, http.StatusOK, w.Code, "%s: %s", tc.name, w.Body.String())
		require.JSONEq(t, `{"status":"updated"}`, w.Body.String(), tc.name)
	}
}

func TestAlertTestSendResponseShape(t *testing.T) {
	t.Parallel()
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer receiver.Close()
	manager := alertdelivery.NewManager(nil, nil, slog.New(slog.DiscardHandler))
	defer manager.Stop(context.Background())
	manager.ApplyWebhook(alertdelivery.WebhookConfig{URL: receiver.URL, Secret: "s3cret"})

	w := httptest.NewRecorder()
	(&Server{alertDelivery: manager}).handleAlertTestSend(w, httptest.NewRequest(http.MethodPost,
		"/api/v1/settings/alerts/test", strings.NewReader(`{"channel":"webhook"}`)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"channel":"webhook","sent":true}`, w.Body.String())
}

// TestVulnerabilityStatusResponseShape pins the scanner status read with no
// KEV provider, with an empty KEV cache, and with a loaded catalog.
func TestVulnerabilityStatusResponseShape(t *testing.T) {
	t.Parallel()
	const catalog = `{"catalogVersion":"2026.10.10","dateReleased":"2026-10-10T00:00:00Z","count":2,` +
		`"vulnerabilities":[{"cveID":"CVE-2099-0001","knownRansomwareCampaignUse":"Known"},` +
		`{"cveID":"CVE-2099-0002","knownRansomwareCampaignUse":"Unknown"}]}`
	statsKeys := []string{
		"activelyExploited", "criticalCount", "cveDatabase", "devicesScanned", "enabled",
		"highCount", "kevEnabled", "lastUpdate", "lowCount", "mediumCount",
		"ransomwareRelated", "running", "totalVulns",
	}
	tests := []struct {
		name     string
		kev      bool
		cache    string
		wantKEV  map[string]any
		wantKeys []string
	}{
		{name: "no KEV", wantKeys: statsKeys},
		{
			name: "KEV not loaded", kev: true,
			wantKEV:  map[string]any{"loaded": false},
			wantKeys: append(slices.Clone(statsKeys), "kevStats"),
		},
		{
			name: "KEV loaded", kev: true, cache: catalog,
			wantKEV: map[string]any{
				"loaded": true, "version": "2026.10.10", "dateReleased": "2026-10-10T00:00:00Z",
				"count": float64(2), "ransomwareCount": float64(1),
			},
			wantKeys: append(slices.Clone(statsKeys), "kevStats"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			cvePath := filepath.Join(dir, "cves.json")
			require.NoError(t, os.WriteFile(cvePath, []byte(cveFixture), 0o600))
			kevDir := ""
			if tt.cache != "" {
				kevDir = dir
				require.NoError(t, os.WriteFile(filepath.Join(dir, "kev_catalog.json"), []byte(tt.cache), 0o600))
			}
			scanner, err := vuln.NewVulnerabilityScanner(&vuln.VulnerabilityScannerConfig{
				Enabled: true, CVEDatabase: "local", LocalDBPath: cvePath,
				SeverityThreshold: "low", KEVEnabled: tt.kev, KEVCachePath: kevDir,
			})
			require.NoError(t, err)
			_, err = scanner.ScanDevice(context.Background(), &discovery.DiscoveredDevice{
				IP: "10.20.30.50", MAC: "02:00:00:00:00:05", Vendor: "Linux", OSGuess: "Linux 5.4",
			})
			require.NoError(t, err)
			cfg := config.DefaultConfig()
			cfg.Security.VulnerabilityScanning.SeverityThreshold = "high"
			s := &Server{vulnScan: scanner}
			s.securitySettings = app.NewSecuritySettings(cfg, filepath.Join(dir, "config.yaml"), nil)

			w := serve(t, s.handleVulnerabilityStatus, "/api/v1/security/vulnerabilities/status")
			require.Equal(t, []string{"enabled", "scanning", "severityFilter", "stats"},
				topLevelKeys(t, w.Body.Bytes()))
			var got struct {
				Enabled        bool                       `json:"enabled"`
				Scanning       bool                       `json:"scanning"`
				SeverityFilter string                     `json:"severityFilter"`
				Stats          map[string]json.RawMessage `json:"stats"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			require.True(t, got.Enabled)
			require.False(t, got.Scanning)
			require.Equal(t, "high", got.SeverityFilter)
			wantKeys := slices.Sorted(slices.Values(tt.wantKeys))
			require.Equal(t, wantKeys, topLevelKeys(t, mustMarshal(t, got.Stats)))
			require.JSONEq(t, "1", string(got.Stats["devicesScanned"]))
			require.JSONEq(t, "1", string(got.Stats["totalVulns"]))
			require.JSONEq(t, "1", string(got.Stats["criticalCount"]))
			require.JSONEq(t, `"local"`, string(got.Stats["cveDatabase"]))
			if tt.wantKEV == nil {
				return
			}
			var kev map[string]any
			require.NoError(t, json.Unmarshal(got.Stats["kevStats"], &kev))
			require.Contains(t, kev, "lastUpdate")
			delete(kev, "lastUpdate")
			require.Equal(t, tt.wantKEV, kev)
		})
	}
}

type fakeProbeAnomalies []anomaly.Anomaly

func (fakeProbeAnomalies) Available() bool { return true }

func (f fakeProbeAnomalies) ActiveAnomalies(context.Context) ([]anomaly.Anomaly, error) {
	return f, nil
}

func TestProbeAnomaliesResponseShape(t *testing.T) {
	t.Parallel()
	seen := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		active fakeProbeAnomalies
		want   string
	}{
		{name: "none", want: `[]`},
		{
			name: "one",
			active: fakeProbeAnomalies{{
				DefKey: "probe.latency", Category: "performance", Severity: "warning",
				Subject: anomaly.SubjectRef{Kind: "probe", ID: "p1"}, Title: "Latency high",
				Description: "d", Recommendation: "r", FirstSeen: seen, LastSeen: seen, Count: 1,
			}},
			want: `[{"defKey":"probe.latency","category":"performance","severity":"warning",` +
				`"subject":{"kind":"probe","id":"p1"},"title":"Latency high","description":"d",` +
				`"recommendation":"r","firstSeen":"2026-10-11T00:00:00Z",` +
				`"lastSeen":"2026-10-11T00:00:00Z","count":1}]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{healthMonitoring: monitoring.NewService(tt.active)}
			w := serve(t, s.handleHealthCheckAnomalies, "/api/v1/telemetry/probes/anomalies")
			require.Equal(t, []string{"activeCount", "anomalies", "timestamp"}, topLevelKeys(t, w.Body.Bytes()))
			var got struct {
				Timestamp   string          `json:"timestamp"`
				Anomalies   json.RawMessage `json:"anomalies"`
				ActiveCount int             `json:"activeCount"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			_, err := time.Parse(time.RFC3339, got.Timestamp)
			require.NoError(t, err)
			require.Equal(t, len(tt.active), got.ActiveCount)
			require.JSONEq(t, tt.want, string(got.Anomalies))
		})
	}
}

// The lease reply drops the domain result's nanosecond leaseTime and
// responseTime; the seconds and milliseconds fields carry the same values.
func TestDHCPLeaseResponseShape(t *testing.T) {
	t.Parallel()
	tested := time.Date(2026, 10, 11, 1, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		result dhcp.TestResult
		want   string
	}{
		{
			name: "lease",
			result: dhcp.TestResult{
				Interface: "eth0", Success: true, Status: dhcp.StatusSuccess,
				ServerIP: "10.0.0.1", OfferedIP: "10.0.0.20", SubnetMask: "255.255.255.0",
				Gateway: "10.0.0.1", DNSServers: []string{"10.0.0.53"}, DomainName: "lab",
				LeaseTime: time.Hour, LeaseTimeSec: 3600,
				ResponseTime: 2 * time.Millisecond, ResponseMs: 2, TestedAt: tested,
			},
			want: `{"interface":"eth0","success":true,"status":"success","serverIp":"10.0.0.1",` +
				`"offeredIp":"10.0.0.20","subnetMask":"255.255.255.0","gateway":"10.0.0.1",` +
				`"dnsServers":["10.0.0.53"],"domainName":"lab","leaseTimeSec":3600,` +
				`"responseTimeMs":2,"testedAt":"2026-10-11T01:00:00Z"}`,
		},
		{
			name: "no lease",
			result: dhcp.TestResult{
				Interface: "eth0", Status: dhcp.StatusError, Error: "interface is down", TestedAt: tested,
			},
			want: `{"interface":"eth0","success":false,"status":"error","responseTimeMs":0,` +
				`"error":"interface is down","testedAt":"2026-10-11T01:00:00Z"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.JSONEq(t, tt.want, string(mustMarshal(t, newDHCPLeaseResponse(&tt.result))))
		})
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestSettingsResponseShape(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Identity = config.IdentityConfig{Name: "seed-idf-3b", Location: "IDF 3B"}
	cfg.Alerts.Webhook = config.AlertWebhookConfig{URL: "https://receiver.example.com/hook", Secret: "enc:s3cret"}
	cfg.Alerts.Escalations = []config.AlertEscalationConfig{{
		Rule: "iface.down", RepeatSeconds: 3600,
		Stages: []config.AlertEscalationStage{{AfterSeconds: 300, Channels: []string{"webhook"}}},
	}}
	s := &Server{config: cfg}
	s.settingsManagement = app.NewSettingsManagement(s.config, "", func() *alertdelivery.Manager { return nil })

	w := serve(t, s.handleSettings, "/api/v1/settings")
	require.NotEmpty(t, w.Header().Get("ETag"))
	require.JSONEq(t, `{
		"vlan": {"enabled": false, "id": 0},
		"ip": {"mode": "dhcp"},
		"thresholds": {
			"dns": {"good": 50, "warning": 200},
			"gateway": {"good": 10, "warning": 50},
			"wifi": {"good": -70, "warning": -80},
			"customPing": {"good": 10, "warning": 100},
			"customTcp": {"good": 100, "warning": 500},
			"customHttp": {"good": 500, "warning": 2000},
			"httpTimings": {
				"dns": {"good": 50, "warning": 500},
				"tcp": {"good": 100, "warning": 500},
				"tls": {"good": 150, "warning": 500},
				"ttfb": {"good": 500, "warning": 2000}
			}
		},
		"healthChecks": {"runPerformance": true, "runSpeedtest": true, "runIperf": true, "runDiscovery": true},
		"speedtest": {"serverId": "", "autoRunOnLink": true},
		"iperf": {
			"autoRunOnLink": false, "server": "", "port": 5201, "protocol": "tcp",
			"direction": "download", "duration": 10, "serverPort": 5201, "enableServer": true
		},
		"alerts": {
			"webhook": {"url": "https://receiver.example.com/hook", "secretSet": true},
			"email": {"host": "", "port": 0, "tls": "", "username": "", "passwordSet": false, "from": "", "to": []},
			"escalations": [
				{"rule": "iface.down", "stages": [{"afterSeconds": 300, "channels": ["webhook"]}], "repeatSeconds": 3600}
			],
			"syslog": {"host": "", "port": 0, "transport": ""},
			"trap": {"host": "", "port": 0, "communitySet": false}
		},
		"displayOptions": {"showPublicIP": true, "unitSystem": "sae"},
		"identity": {"name": "seed-idf-3b", "location": "IDF 3B"}
	}`, w.Body.String())
}
