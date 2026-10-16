package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// Wire-shape pins for the polling, topology and alert-rule handlers; response_dto_shape_internal_test.go
// explains why these exist.

func serveClaimed(t *testing.T, h http.HandlerFunc, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h(w, withClaim(httptest.NewRequest(http.MethodGet, target, http.NoBody)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	return w
}

func TestPollingTargetResponseShape(t *testing.T) {
	s, db := newPollingTargetsTestServer(t)
	seeded := seedTarget(t, db, "router-1")

	list := serveClaimed(t, s.handlePollingTargets, pollingTargetsPath)
	require.Equal(t, []string{"count", "targets"}, topLevelKeys(t, list.Body.Bytes()))
	var l struct {
		Count   int               `json:"count"`
		Targets []json.RawMessage `json:"targets"`
	}
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &l))
	require.Equal(t, 1, l.Count)
	require.Len(t, l.Targets, 1)

	one := serveClaimed(t, routed(pollingTargetPath, s.handlePollingTargetByID), pollingTargetsPath+"/"+seeded.ID)
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
	s, _ := newDeviceCredentialsTestServer(t)
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
	targets, _ := newPollingTargetsTestServer(t)
	require.JSONEq(t, `{"count":0,"targets":[]}`,
		serveClaimed(t, targets.handlePollingTargets, pollingTargetsPath).Body.String())

	creds, _ := newDeviceCredentialsTestServer(t)
	require.JSONEq(t, `{"count":0,"credentials":[]}`,
		serveClaimed(t, creds.handleDeviceCredentials, deviceCredentialsPath).Body.String())
}

// A create without a chain stores the default chain; the response is the
// stored target, so it reports that chain rather than null.
func TestCreatedPollingTargetReportsStoredChain(t *testing.T) {
	s, _ := newPollingTargetsTestServer(t)
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, postJSON(t, map[string]any{
		"name": "router-1", "ipAddress": "10.0.0.1", "enabled": true,
	}))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created PollingTargetResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	var stored PollingTargetResponse
	require.NoError(t, json.Unmarshal(
		serveClaimed(
			t,
			routed(pollingTargetPath, s.handlePollingTargetByID),
			pollingTargetsPath+"/"+created.ID,
		).Body.Bytes(),
		&stored,
	))
	require.Equal(t, polling.DefaultCollectorChain(), stored.CollectorChain)
	require.Equal(t, stored.CollectorChain, created.CollectorChain)
}

func TestTopologyResponseShapes(t *testing.T) {
	t.Parallel()
	s, db := newTopologyTestServer(t)
	nodeID := seedTopology(t, db)
	seedARPBindingsForHandler(t, db)

	nodeKeys := []string{
		"chassisId", "clientId", "deviceType", "displayName", "firstSeen", "id",
		"identityHash", "lastSeen", "metadata", "primaryIp", "primaryMac", "sysName",
	}
	linkKeys := []string{
		"evidence", "firstSeen", "id", "lastSeen", "linkType", "sourceInterface",
		"sourceNodeId", "speedMbps", "status", "targetInterface", "targetNodeId",
	}

	nodes := serve(t, s.handleTopologyNodes, APIVersionPrefix+"/topology/nodes")
	require.Equal(t, []string{"count", "nodes"}, topLevelKeys(t, nodes.Body.Bytes()))
	var nl struct {
		Count int               `json:"count"`
		Nodes []json.RawMessage `json:"nodes"`
	}
	require.NoError(t, json.Unmarshal(nodes.Body.Bytes(), &nl))
	require.Equal(t, 2, nl.Count)
	require.Equal(t, nodeKeys, topLevelKeys(t, nl.Nodes[0]))

	detail := serve(t, routed(topologyNodePath, s.handleTopologyNodeByID), APIVersionPrefix+"/topology/nodes/"+nodeID)
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

	links := serve(t, s.handleTopologyLinks, APIVersionPrefix+"/topology/links")
	require.Equal(t, []string{"count", "links"}, topLevelKeys(t, links.Body.Bytes()))
	var ll struct {
		Links []json.RawMessage `json:"links"`
	}
	require.NoError(t, json.Unmarshal(links.Body.Bytes(), &ll))
	require.JSONEq(t, string(d.Links[0]), string(ll.Links[0]))

	arp := serve(t, s.handleTopologyARP, APIVersionPrefix+"/topology/arp")
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
	s, _ := newTopologyTestServer(t)
	require.JSONEq(t, `{"count":0,"nodes":[]}`,
		serve(t, s.handleTopologyNodes, APIVersionPrefix+"/topology/nodes").Body.String())
	require.JSONEq(t, `{"count":0,"links":[]}`,
		serve(t, s.handleTopologyLinks, APIVersionPrefix+"/topology/links").Body.String())
	require.JSONEq(t, `{"count":0,"bindings":[]}`,
		serve(t, s.handleTopologyARP, APIVersionPrefix+"/topology/arp").Body.String())
}

func TestAlertRuleResponseShape(t *testing.T) {
	s, db := newAlertRulesTestServer(t)
	seeded := seedAlertRule(t, db, "rule-1", true)

	list := serve(t, s.handleAlertRules, alertRulesPath)
	require.Equal(t, []string{"count", "rules"}, topLevelKeys(t, list.Body.Bytes()))
	var l struct {
		Count int               `json:"count"`
		Rules []json.RawMessage `json:"rules"`
	}
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &l))
	require.Equal(t, 1, l.Count)
	require.Len(t, l.Rules, 1)

	one := serve(t, routed(alertRulePath, s.handleAlertRuleByID), alertRulesPath+"/"+strconv.FormatInt(seeded.ID, 10))
	require.JSONEq(t, string(l.Rules[0]), one.Body.String())

	// Every key is always sent, empty match filters included.
	require.Equal(t, []string{
		"alertMessage", "alertSeverity", "alertTitle", "alertType", "createdAt", "enabled", "groupId",
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
	s, _ := newAlertRulesTestServer(t)
	w := serve(t, s.handleAlertRules, alertRulesPath)
	require.JSONEq(t, `{"count":0,"rules":[]}`, w.Body.String())
}
