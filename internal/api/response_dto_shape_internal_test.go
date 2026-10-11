package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	enginestatus "github.com/MustardSeedNetworks/seed/internal/engine/status"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
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

func serveClaimed(t *testing.T, h http.HandlerFunc, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h(w, withClaim(httptest.NewRequest(http.MethodGet, target, http.NoBody)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	return w
}

func TestPollingTargetResponseShape(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	seeded := seedTarget(t, s.db(), "router-1")

	list := serveClaimed(t, s.handlePollingTargets, pollingTargetsPath)
	require.Equal(t, []string{"count", "targets"}, topLevelKeys(t, list.Body.Bytes()))
	var l struct {
		Count   int               `json:"count"`
		Targets []json.RawMessage `json:"targets"`
	}
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &l))
	require.Equal(t, 1, l.Count)
	require.Len(t, l.Targets, 1)

	one := serveClaimed(t, s.handlePollingTargetByID, pollingTargetsPathPrefix+seeded.ID)
	require.JSONEq(t, string(l.Targets[0]), one.Body.String())

	// lastPolledAt is absent until the first poll; every other key is always sent.
	require.Equal(t, []string{
		"clientId", "collectorChain", "createdAt", "credentialsId", "enabled", "id",
		"ipAddress", "lastError", "lastStatus", "name", "pollIntervalSeconds",
		"snmpVersion", "updatedAt",
	}, topLevelKeys(t, one.Body.Bytes()))
	var got map[string]any
	require.NoError(t, json.Unmarshal(one.Body.Bytes(), &got))
	require.Equal(t, seeded.ID, got["id"])
	require.Equal(t, testClientID, got["clientId"])
	require.Equal(t, "10.0.0.1", got["ipAddress"])
	require.Equal(t, []any{"sys_info"}, got["collectorChain"])
	require.InDelta(t, 60, got["pollIntervalSeconds"], 0)
	require.Empty(t, got["credentialsId"])
	created, err := time.Parse(time.RFC3339Nano, got["createdAt"].(string))
	require.NoError(t, err)
	require.Equal(t, time.UTC, created.Location())
}

func TestDeviceCredentialListShape(t *testing.T) {
	s := newDeviceCredentialsTestServer(t)
	require.Equal(t, http.StatusOK, postCredential(t, s, `{"name":"core","community":"c"}`).Code)

	w := serveClaimed(t, s.handleDeviceCredentials, deviceCredentialsPath)
	require.Equal(t, []string{"count", "credentials"}, topLevelKeys(t, w.Body.Bytes()))
	var l struct {
		Count       int              `json:"count"`
		Credentials []map[string]any `json:"credentials"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &l))
	require.Equal(t, 1, l.Count)
	require.Len(t, l.Credentials, 1)
	require.Equal(t, "core", l.Credentials[0]["name"])
}

// An empty estate is an empty array, never null: the documented type is an
// array, and the UI maps over it.
func TestEmptyPollingListsAreArrays(t *testing.T) {
	targets := newPollingTargetsTestServer(t)
	require.JSONEq(t, `{"count":0,"targets":[]}`,
		serveClaimed(t, targets.handlePollingTargets, pollingTargetsPath).Body.String())

	creds := newDeviceCredentialsTestServer(t)
	require.JSONEq(t, `{"count":0,"credentials":[]}`,
		serveClaimed(t, creds.handleDeviceCredentials, deviceCredentialsPath).Body.String())
}

// A create without a chain stores the default chain; the response is the
// stored target, so it reports that chain rather than null.
func TestCreatedPollingTargetReportsStoredChain(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, postJSON(t, map[string]any{
		"name": "router-1", "ipAddress": "10.0.0.1", "enabled": true,
	}))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created PollingTargetResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	var stored PollingTargetResponse
	require.NoError(t, json.Unmarshal(
		serveClaimed(t, s.handlePollingTargetByID, pollingTargetsPathPrefix+created.ID).Body.Bytes(),
		&stored))
	require.Equal(t, polling.DefaultCollectorChain(), stored.CollectorChain)
	require.Equal(t, stored.CollectorChain, created.CollectorChain)
}

func TestTopologyResponseShapes(t *testing.T) {
	t.Parallel()
	s := newTopologyTestServer(t)
	nodeID := seedTopology(t, s.db())
	seedARPBindingsForHandler(t, s.db())

	nodeKeys := []string{
		"chassisId", "clientId", "deviceType", "displayName", "firstSeen", "id",
		"identityHash", "lastSeen", "metadata", "primaryIp", "primaryMac", "sysName",
	}
	linkKeys := []string{
		"evidence", "firstSeen", "id", "lastSeen", "linkType", "sourceInterface",
		"sourceNodeId", "speedMbps", "status", "targetInterface", "targetNodeId", "utilizationPct",
	}

	nodes := serve(t, s.handleTopologyNodes, topologyPathPrefix+"nodes")
	require.Equal(t, []string{"count", "nodes"}, topLevelKeys(t, nodes.Body.Bytes()))
	var nl struct {
		Count int               `json:"count"`
		Nodes []json.RawMessage `json:"nodes"`
	}
	require.NoError(t, json.Unmarshal(nodes.Body.Bytes(), &nl))
	require.Equal(t, 2, nl.Count)
	require.Equal(t, nodeKeys, topLevelKeys(t, nl.Nodes[0]))

	detail := serve(t, s.handleTopologyNodeByID, topologyPathPrefix+"nodes/"+nodeID)
	require.Equal(t, []string{"interfaces", "links", "node"}, topLevelKeys(t, detail.Body.Bytes()))
	var d struct {
		Node       json.RawMessage   `json:"node"`
		Interfaces []json.RawMessage `json:"interfaces"`
		Links      []json.RawMessage `json:"links"`
	}
	require.NoError(t, json.Unmarshal(detail.Body.Bytes(), &d))
	require.Equal(t, nodeKeys, topLevelKeys(t, d.Node))
	require.Len(t, d.Interfaces, 1)
	require.Equal(t, []string{
		"id", "ifAdminStatus", "ifAlias", "ifDescr", "ifIndex", "ifName", "ifOperStatus",
		"ifPhysAddr", "ifType", "lastSeen", "nodeId", "speedBps",
	}, topLevelKeys(t, d.Interfaces[0]))
	require.Len(t, d.Links, 1)
	require.Equal(t, linkKeys, topLevelKeys(t, d.Links[0]))

	var node map[string]any
	require.NoError(t, json.Unmarshal(d.Node, &node))
	require.Equal(t, "router-1", node["displayName"])
	require.Equal(t, map[string]any{}, node["metadata"])
	var iface map[string]any
	require.NoError(t, json.Unmarshal(d.Interfaces[0], &iface))
	require.InDelta(t, 1_000_000_000, iface["speedBps"], 0)

	links := serve(t, s.handleTopologyLinks, topologyPathPrefix+"links")
	require.Equal(t, []string{"count", "links"}, topLevelKeys(t, links.Body.Bytes()))
	var ll struct {
		Links []json.RawMessage `json:"links"`
	}
	require.NoError(t, json.Unmarshal(links.Body.Bytes(), &ll))
	require.JSONEq(t, string(d.Links[0]), string(ll.Links[0]))

	arp := serve(t, s.handleTopologyARP, topologyPathPrefix+"arp")
	require.Equal(t, []string{"bindings", "count"}, topLevelKeys(t, arp.Body.Bytes()))
	var al struct {
		Bindings []json.RawMessage `json:"bindings"`
	}
	require.NoError(t, json.Unmarshal(arp.Body.Bytes(), &al))
	require.Equal(t, []string{
		"clientId", "id", "ifIndex", "ipAddress", "lastSeen", "macAddress", "mediaType", "sourceNodeId",
	}, topLevelKeys(t, al.Bindings[0]))
}

func TestEmptyTopologyListsAreArrays(t *testing.T) {
	t.Parallel()
	s := newTopologyTestServer(t)
	require.JSONEq(t, `{"count":0,"nodes":[]}`,
		serve(t, s.handleTopologyNodes, topologyPathPrefix+"nodes").Body.String())
	require.JSONEq(t, `{"count":0,"links":[]}`,
		serve(t, s.handleTopologyLinks, topologyPathPrefix+"links").Body.String())
	require.JSONEq(t, `{"count":0,"bindings":[]}`,
		serve(t, s.handleTopologyARP, topologyPathPrefix+"arp").Body.String())
}

func TestAlertRuleResponseShape(t *testing.T) {
	s := newAlertRulesTestServer(t)
	seeded := seedAlertRule(t, s.db(), "rule-1", true)

	list := serve(t, s.handleAlertRules, alertRulesPath)
	require.Equal(t, []string{"count", "rules"}, topLevelKeys(t, list.Body.Bytes()))
	var l struct {
		Count int               `json:"count"`
		Rules []json.RawMessage `json:"rules"`
	}
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &l))
	require.Equal(t, 1, l.Count)
	require.Len(t, l.Rules, 1)

	one := serve(t, s.handleAlertRuleByID, alertRulesPathPrefix+strconv.FormatInt(seeded.ID, 10))
	require.JSONEq(t, string(l.Rules[0]), one.Body.String())

	// Every key is always sent, empty match filters included.
	require.Equal(t, []string{
		"alertMessage", "alertSeverity", "alertTitle", "alertType", "createdAt", "enabled",
		"id", "matchKind", "matchPayloadContains", "matchSeverity", "name",
		"thresholdCount", "updatedAt", "windowSeconds",
	}, topLevelKeys(t, one.Body.Bytes()))
	var got map[string]any
	require.NoError(t, json.Unmarshal(one.Body.Bytes(), &got))
	require.InDelta(t, seeded.ID, got["id"], 0)
	require.Equal(t, "rule-1", got["name"])
	require.Equal(t, "syslog-udp", got["matchKind"])
	require.Empty(t, got["matchPayloadContains"])
	created, err := time.Parse(time.RFC3339Nano, got["createdAt"].(string))
	require.NoError(t, err)
	require.Equal(t, time.UTC, created.Location())
}

func TestEmptyAlertRuleListIsArray(t *testing.T) {
	s := newAlertRulesTestServer(t)
	w := serve(t, s.handleAlertRules, alertRulesPath)
	require.JSONEq(t, `{"count":0,"rules":[]}`, w.Body.String())
}

func TestDeviceDiscoveryStatusResponseShape(t *testing.T) {
	t.Parallel()
	s := &Server{deviceDisc: enumerate.NewDeviceDiscovery("lo")}
	w := serve(t, s.handleDevicesStatus, "/api/v1/security/devices/status")
	require.Equal(t, []string{"deviceCount", "interface", "lastScan", "localIP", "scanning", "subnet"},
		topLevelKeys(t, w.Body.Bytes()))
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

type noScanRadio struct{ statusRadio }

func (noScanRadio) ScannerAvailable() bool { return false }

type scanRadio struct{ statusRadio }

func (scanRadio) ScannerAvailable() bool { return true }
func (scanRadio) SetInterface(string)    {}
func (scanRadio) Scan() ([]*wifi.ScannedNetwork, error) {
	return []*wifi.ScannedNetwork{{
		SSID: "lab", BSSID: "aa:bb:cc:dd:ee:ff", Signal: -51, Channel: 36,
		Frequency: 5180, Security: "WPA2", ChannelWidth: 80, LastSeen: time.Unix(0, 0).UTC(),
	}}, nil
}

func (scanRadio) Disconnect() (*wifi.ConnectionResult, error) {
	return &wifi.ConnectionResult{Success: true, Message: "Disconnected", SSID: "lab"}, nil
}

func TestWiFiScanResponseShape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		radio troubleshooting.Hardware
		want  string
	}{
		{
			name:  "scanned",
			radio: scanRadio{},
			want: `{"interface":"wlan0","available":true,"networks":[{"ssid":"lab","bssid":"aa:bb:cc:dd:ee:ff",` +
				`"signal":-51,"channel":36,"frequency":5180,"security":"WPA2","channelWidth":80,"noiseFloor":0,` +
				`"snr":0,"htMode":"","isDFS":false,"lastSeen":"1970-01-01T00:00:00Z","hidden":false,` +
				`"pmfRequired":false,"wpsEnabled":false,"rrmNeighbor":false,"btmSupported":false,` +
				`"ftSupported":false,"hasBssLoad":false,"channelUtil":0,"advertisedStations":0}]}`,
		},
		{
			name:  "no scanner",
			radio: noScanRadio{},
			want:  `{"interface":"wlan0","available":false,"networks":[],"error":"WiFi scanner not initialized"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{wifiManagement: troubleshooting.NewManagement(tt.radio, nil, fixedWiFiInterface("wlan0"))}
			w := serve(t, s.handleWiFiScan, "/api/v1/wifi/wifi/scan")
			require.JSONEq(t, tt.want, w.Body.String())
		})
	}
}

func TestWiFiDisconnectResponseShape(t *testing.T) {
	t.Parallel()
	s := &Server{wifiManagement: troubleshooting.NewManagement(scanRadio{}, nil, fixedWiFiInterface("wlan0"))}
	w := serve(t, s.handleWiFiDisconnect, "/api/v1/wifi/wifi/disconnect")
	require.JSONEq(t, `{"success":true,"message":"Disconnected","ssid":"lab"}`, w.Body.String())
}

func TestWiFiSettingsUpdateResponseShape(t *testing.T) {
	t.Parallel()
	s := &Server{wifiManagement: troubleshooting.NewManagement(scanRadio{}, nil, fixedWiFiInterface("wlan0"))}
	w := httptest.NewRecorder()
	s.handleWiFiSettings(w, httptest.NewRequest(http.MethodPut, "/api/v1/wifi/wifi/settings",
		strings.NewReader(`{"interface":"wlan1"}`)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"status":"success","message":"WiFi settings updated"}`, w.Body.String())
}

func TestWiFiSavedNetworksWithoutManagerIsEmptyArray(t *testing.T) {
	t.Parallel()
	w := serve(t, (&Server{}).handleWiFiSavedNetworks, "/api/v1/wifi/wifi/saved")
	require.JSONEq(t, `{"networks":[]}`, w.Body.String())
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
	s.settingsStore = app.NewSettings(s.db, s.config)
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

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
