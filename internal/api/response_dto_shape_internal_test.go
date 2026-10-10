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
	"github.com/MustardSeedNetworks/seed/internal/engine"
	enginestatus "github.com/MustardSeedNetworks/seed/internal/engine/status"
	"github.com/MustardSeedNetworks/seed/internal/polling"
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
