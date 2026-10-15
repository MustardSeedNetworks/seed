package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// newTopologyTestServer builds a minimal Server backed by a temp
// SQLite DB. Only the fields the topology handlers touch are
// initialized — everything else stays nil so unrelated handlers
// would crash if called.
func newTopologyTestServer(t *testing.T) (*Server, *database.DB) {
	t.Helper()
	db := newTestDB(t)
	s := &Server{}
	s.persist = app.NewPersistence(db)
	s.topologyQueries = app.NewTopologyQueries(s.persistence, topologyMaxLimit)
	return s, db
}

// seedTopology inserts one node + one interface + one link so the
// handlers have something to return. Returns the node ID so tests
// can hit /nodes/{id}.
func seedTopology(t *testing.T, db *database.DB) string {
	t.Helper()
	ctx := context.Background()
	node, err := db.Topology().Upsert(ctx, &topology.Node{
		ID:           "node-test-1",
		ClientID:     "default",
		IdentityHash: "hash-test-1",
		DisplayName:  "router-1",
		DeviceType:   "cisco",
		SysName:      "router-1.example.com",
		PrimaryMAC:   "aa:bb:cc:dd:ee:01",
		LastSeen:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}
	if upErr := db.Topology().UpsertInterface(ctx, &topology.Interface{
		NodeID: node.ID, IfIndex: 1, IfName: "Gi0/0",
		IfAdminStatus: 1, IfOperStatus: 1, SpeedBps: 1_000_000_000,
		LastSeen: time.Now().UTC(),
	}); upErr != nil {
		t.Fatalf("seed interface: %v", upErr)
	}
	// Create a second node so the edge has both endpoints.
	other, _ := db.Topology().Upsert(ctx, &topology.Node{
		ID:           "node-test-2",
		ClientID:     "default",
		IdentityHash: "hash-test-2",
		DisplayName:  "core-sw",
		SysName:      "core-sw",
		LastSeen:     time.Now().UTC(),
	})
	if linkErr := db.Topology().UpsertLink(ctx, &topology.Link{
		ID:           "link-test-1",
		SourceNodeID: node.ID,
		TargetNodeID: other.ID,
		LinkType:     "lldp",
		Status:       "up",
		LastSeen:     time.Now().UTC(),
	}); linkErr != nil {
		t.Fatalf("seed link: %v", linkErr)
	}
	return node.ID
}

// A link reports the utilization of its source port's latest rated poll
// (seed#3024); a link whose port has none omits the field rather than
// claiming 0.
func TestHandleTopologyLinks_ReportsPortUtilization(t *testing.T) {
	s, db := newTopologyTestServer(t)
	nodeID := seedTopology(t, db)
	ctx := context.Background()
	now := time.Now().UTC()

	target := &polling.Target{
		ClientID: database.DefaultClientID, Name: "router-1", IPAddress: "10.0.0.1",
		SNMPVersion: "v2c", PollIntervalSec: 60, Enabled: true, CollectorChain: []string{"if_table"},
	}
	if err := db.PollingTargets().Create(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if err := db.Topology().UpsertTargetNode(ctx, database.DefaultClientID, target.ID, nodeID, now); err != nil {
		t.Fatalf("map target: %v", err)
	}
	if err := db.Topology().UpsertLink(ctx, &topology.Link{
		ID: "link-test-1", SourceNodeID: nodeID, TargetNodeID: "node-test-2",
		SourceInterface: "Gi0/0", LinkType: "lldp", Status: "up", LastSeen: now,
	}); err != nil {
		t.Fatalf("name the link's port: %v", err)
	}

	links := func() []map[string]any {
		t.Helper()
		w := serve(t, s.handleTopologyLinks, APIVersionPrefix+"/topology/links")
		var resp struct {
			Links []map[string]any `json:"links"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Links) != 1 {
			t.Fatalf("links = %+v, want one", resp.Links)
		}
		return resp.Links
	}
	if got, ok := links()[0]["utilizationPct"]; ok {
		t.Errorf("utilizationPct = %v before any rated poll, want absent", got)
	}

	util := ifrate.Utilization{In: 12.5, Out: 30}
	if err := db.Metrics().RecordInterfaceRates(ctx, []ifrate.Rate{{
		ClientID: database.DefaultClientID, TargetID: target.ID, IfIndex: 1, At: now,
		Octets: &ifrate.Octets{In: 1.5e6, Out: 3.75e6, Utilization: &util},
	}}); err != nil {
		t.Fatalf("RecordInterfaceRates: %v", err)
	}
	if got := links()[0]["utilizationPct"]; got != 30.0 {
		t.Errorf("utilizationPct = %v, want 30 (the busier direction)", got)
	}
}

func TestHandleTopologyNodes_ReturnsSeedededNodes(t *testing.T) {
	s, db := newTopologyTestServer(t)
	seedTopology(t, db)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Count int              `json:"count"`
		Nodes []map[string]any `json:"nodes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2", resp.Count)
	}
	if len(resp.Nodes) != 2 {
		t.Fatalf("nodes = %d, want 2", len(resp.Nodes))
	}
}

func TestHandleTopologyNodes_DeviceTypeFilter(t *testing.T) {
	s, db := newTopologyTestServer(t)
	seedTopology(t, db)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes?device_type=cisco", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodes(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count != 1 {
		t.Errorf("count = %d, want 1 (only the cisco node)", resp.Count)
	}
}

func TestHandleTopologyNodes_InvalidSinceReturns400(t *testing.T) {
	s, _ := newTopologyTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes?since=not-a-date", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodes(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleTopologyNodes_InvalidLimitReturns400(t *testing.T) {
	s, _ := newTopologyTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes?limit=zero", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodes(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleTopologyNodeByID_ReturnsNodeWithInterfacesAndLinks(t *testing.T) {
	s, db := newTopologyTestServer(t)
	nodeID := seedTopology(t, db)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes/"+nodeID, http.NoBody)
	w := httptest.NewRecorder()
	routed(topologyNodePath, s.handleTopologyNodeByID)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Node       map[string]any   `json:"node"`
		Interfaces []map[string]any `json:"interfaces"`
		Links      []map[string]any `json:"links"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Node["id"] != nodeID {
		t.Errorf("node.id = %v, want %q", resp.Node["id"], nodeID)
	}
	if len(resp.Interfaces) != 1 {
		t.Errorf("interfaces = %d, want 1", len(resp.Interfaces))
	}
	if len(resp.Links) != 1 {
		t.Errorf("links = %d, want 1", len(resp.Links))
	}
}

func TestHandleTopologyNodeByID_UnknownReturns404(t *testing.T) {
	s, db := newTopologyTestServer(t)
	seedTopology(t, db)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes/no-such-node", http.NoBody)
	w := httptest.NewRecorder()
	routed(topologyNodePath, s.handleTopologyNodeByID)(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// An empty or nested node id is not a node: the router answers 404.
func TestHandleTopologyNodeByID_EmptyOrNestedIDReturns404(t *testing.T) {
	s, _ := newTopologyTestServer(t)
	for _, path := range []string{"/topology/nodes/", "/topology/nodes/a/b"} {
		req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+path, http.NoBody)
		w := httptest.NewRecorder()
		routed(topologyNodePath, s.handleTopologyNodeByID)(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", path, w.Code)
		}
	}
}

// The handler's own contract — and the topology map that consumes it
// (seed#2700) — is that ?node_id is a filter, not a requirement: without it
// the endpoint serves the global edge list. It used to answer 400, so the map
// would have had to fetch the edges of every node one request at a time.
func TestHandleTopologyLinks_WithoutNodeIDReturnsEveryLink(t *testing.T) {
	s, db := newTopologyTestServer(t)
	seedTopology(t, db)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/links", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyLinks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var resp struct {
		Count int              `json:"count"`
		Links []map[string]any `json:"links"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 1 || len(resp.Links) != 1 {
		t.Errorf("count = %d, links = %d, want 1 and 1", resp.Count, len(resp.Links))
	}
}

func TestHandleTopologyLinks_ReturnsLinksForNode(t *testing.T) {
	s, db := newTopologyTestServer(t)
	nodeID := seedTopology(t, db)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/links?node_id="+nodeID, http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyLinks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"links"`) {
		t.Errorf("response missing links envelope; got %s", w.Body.String())
	}
}

func seedARPBindingsForHandler(t *testing.T, db *database.DB) {
	t.Helper()
	// Reuse seedTopology to satisfy the FK on source_node_id, then
	// insert two bindings under node-test-1.
	_ = seedTopology(t, db)
	ctx := context.Background()
	for _, b := range []*topology.ARPBinding{
		{
			ClientID: "default", SourceNodeID: "node-test-1", IfIndex: 1,
			IPAddress: "10.0.0.1", MACAddress: "aa:bb:cc:dd:ee:01",
			LastSeen: time.Now().UTC(),
		},
		{
			ClientID: "default", SourceNodeID: "node-test-1", IfIndex: 1,
			IPAddress: "10.0.0.2", MACAddress: "aa:bb:cc:dd:ee:02",
			LastSeen: time.Now().UTC(),
		},
	} {
		if err := db.Topology().UpsertARPBinding(ctx, b); err != nil {
			t.Fatalf("seed binding: %v", err)
		}
	}
}

func TestHandleTopologyARP_ReturnsBindings(t *testing.T) {
	s, db := newTopologyTestServer(t)
	seedARPBindingsForHandler(t, db)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/arp", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyARP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Count    int              `json:"count"`
		Bindings []map[string]any `json:"bindings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 2 || len(resp.Bindings) != 2 {
		t.Errorf("count=%d / bindings=%d, want 2/2", resp.Count, len(resp.Bindings))
	}
}

func TestHandleTopologyARP_FilterByNodeID(t *testing.T) {
	s, db := newTopologyTestServer(t)
	seedARPBindingsForHandler(t, db)

	req := httptest.NewRequest(http.MethodGet,
		APIVersionPrefix+"/topology/arp?node_id=node-test-1", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyARP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2 (both bindings live on node-test-1)", resp.Count)
	}
}

func TestHandleTopologyARP_InvalidSinceReturns400(t *testing.T) {
	s, _ := newTopologyTestServer(t)
	req := httptest.NewRequest(http.MethodGet,
		APIVersionPrefix+"/topology/arp?since=garbage", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyARP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleTopologyARP_LimitClampsToMax(t *testing.T) {
	s, db := newTopologyTestServer(t)
	seedARPBindingsForHandler(t, db)

	req := httptest.NewRequest(http.MethodGet,
		APIVersionPrefix+"/topology/arp?limit=5000", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyARP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (limit should clamp, not 400)", w.Code)
	}
}

func TestRawJSON_NonObjectFallsBackToEmpty(t *testing.T) {
	for in, want := range map[string]string{
		"not json {": "{}",
		"":           "{}",
		"null":       "{}",
		"[1,2]":      "{}",
		`"text"`:     "{}",
		`{"ok":1}`:   `{"ok":1}`,
	} {
		if got := string(rawJSON(in)); got != want {
			t.Errorf("rawJSON(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatTime_ZeroReturnsEmpty(t *testing.T) {
	if got := formatTime(time.Time{}); got != "" {
		t.Errorf("zero time should format as empty string, got %q", got)
	}
}
