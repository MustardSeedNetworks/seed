package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
)

// newTopologyTestServer builds a minimal Server backed by a temp
// SQLite DB. Only the fields the topology handlers touch are
// initialized — everything else stays nil so unrelated handlers
// would crash if called.
func newTopologyTestServer(t *testing.T) *Server {
	t.Helper()
	db := newTestDB(t)
	s := &Server{
		services: NewServiceContainer(),
	}
	s.services.Database = &DatabaseServices{DB: db}
	return s
}

// seedTopology inserts one node + one interface + one link so the
// handlers have something to return. Returns the node ID so tests
// can hit /nodes/{id}.
func seedTopology(t *testing.T, db *database.DB) string {
	t.Helper()
	ctx := context.Background()
	node, err := db.Topology().Upsert(ctx, &database.TopologyNode{
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
	if upErr := db.Topology().UpsertInterface(ctx, &database.TopologyInterface{
		NodeID: node.ID, IfIndex: 1, IfName: "Gi0/0",
		IfAdminStatus: 1, IfOperStatus: 1, SpeedBps: 1_000_000_000,
		LastSeen: time.Now().UTC(),
	}); upErr != nil {
		t.Fatalf("seed interface: %v", upErr)
	}
	// Create a second node so the edge has both endpoints.
	other, _ := db.Topology().Upsert(ctx, &database.TopologyNode{
		ID:           "node-test-2",
		ClientID:     "default",
		IdentityHash: "hash-test-2",
		DisplayName:  "core-sw",
		SysName:      "core-sw",
		LastSeen:     time.Now().UTC(),
	})
	if linkErr := db.Topology().UpsertLink(ctx, &database.TopologyLink{
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

func TestHandleTopologyNodes_ReturnsSeedededNodes(t *testing.T) {
	s := newTopologyTestServer(t)
	seedTopology(t, s.db())

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

func TestHandleTopologyNodes_RejectsNonGET(t *testing.T) {
	s := newTopologyTestServer(t)
	req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/topology/nodes", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodes(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestHandleTopologyNodes_DeviceTypeFilter(t *testing.T) {
	s := newTopologyTestServer(t)
	seedTopology(t, s.db())

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
	s := newTopologyTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes?since=not-a-date", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodes(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleTopologyNodes_InvalidLimitReturns400(t *testing.T) {
	s := newTopologyTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes?limit=zero", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodes(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleTopologyNodeByID_ReturnsNodeWithInterfacesAndLinks(t *testing.T) {
	s := newTopologyTestServer(t)
	nodeID := seedTopology(t, s.db())

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes/"+nodeID, http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodeByID(w, req)

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
	s := newTopologyTestServer(t)
	seedTopology(t, s.db())

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes/no-such-node", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodeByID(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleTopologyNodeByID_EmptyIDReturns400(t *testing.T) {
	s := newTopologyTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/nodes/", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyNodeByID(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleTopologyLinks_RequiresNodeID(t *testing.T) {
	s := newTopologyTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/links", http.NoBody)
	w := httptest.NewRecorder()
	s.handleTopologyLinks(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (missing node_id)", w.Code)
	}
}

func TestHandleTopologyLinks_ReturnsLinksForNode(t *testing.T) {
	s := newTopologyTestServer(t)
	nodeID := seedTopology(t, s.db())

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

func TestRawJSON_InvalidFallsBackToEmpty(t *testing.T) {
	if got := string(rawJSON("not json {")); got != "{}" {
		t.Errorf("invalid JSON should fall back to {}, got %q", got)
	}
	if got := string(rawJSON("")); got != "{}" {
		t.Errorf("empty should fall back to {}, got %q", got)
	}
	if got := string(rawJSON(`{"ok":1}`)); got != `{"ok":1}` {
		t.Errorf("valid JSON should pass through, got %q", got)
	}
}

func TestFormatTime_ZeroReturnsEmpty(t *testing.T) {
	if got := formatTime(time.Time{}); got != "" {
		t.Errorf("zero time should format as empty string, got %q", got)
	}
}
