package api

// Topology read-only endpoints (Stage A5.1) expose the fat-Node
// graph that the Stage A4 reconcilers maintain. All handlers are
// GET-only and respect the authenticated session's client_id.
//
//   GET /api/v1/topology/nodes            — list nodes
//   GET /api/v1/topology/nodes/{id}       — single node with interfaces + links
//   GET /api/v1/topology/links            — list links (optionally per-node)
//   GET /api/v1/topology/arp              — ARP bindings (optionally per-node)
//
// Each handler runs through the standard auth + i18n middleware so
// they live under the same JWT/PAT surface as the rest of /api/v1.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
)

// topologyPathPrefix is the route root for every topology handler.
// Used to strip the prefix when extracting path parameters.
const topologyPathPrefix = APIVersionPrefix + "/topology/"

// topologyMaxLimit caps the per-page result size for all topology
// list endpoints. Larger values risk exceeding the JSON encoder's
// default buffer on enterprise-scale graphs (5k+ nodes).
const topologyMaxLimit = 1000

// topologyDefaultLimit is the page size when ?limit isn't provided.
const topologyDefaultLimit = 200

// handleTopologyNodes serves GET /api/v1/topology/nodes. Filters via
// ?device_type, ?since (RFC3339), ?limit. Returns 200 with a JSON
// array. Empty results are not 404 — that's reserved for malformed
// requests.
func (s *Server) handleTopologyNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	opts, parseErr := parseTopologyListOptions(r)
	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusBadRequest)
		return
	}

	nodes, err := db.Topology().List(r.Context(), opts)
	if err != nil {
		logger.ErrorContext(r.Context(), "list topology_nodes failed", "error", err)
		http.Error(w, "Failed to list nodes", http.StatusInternalServerError)
		return
	}

	writeTopologyJSON(w, r, "nodes", encodeNodes(nodes))
}

// handleTopologyNodeByID serves GET /api/v1/topology/nodes/{id}.
// Returns the node plus its interfaces and links — one HTTP call
// per "device detail" page in the UI.
func (s *Server) handleTopologyNodeByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, topologyPathPrefix+"nodes/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "Missing or invalid node id", http.StatusBadRequest)
		return
	}
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	// We don't have a GetByID — look up via List with no filter,
	// then match in-memory. List is paged; with at most N=1000 nodes
	// per page this is fine. A future GetByID method on the repo
	// becomes a worthwhile optimization when the graph exceeds the
	// page cap.
	nodes, err := db.Topology().List(r.Context(), database.TopologyListOptions{Limit: topologyMaxLimit})
	if err != nil {
		logger.ErrorContext(r.Context(), "list topology_nodes failed", "error", err)
		http.Error(w, "Failed to load node", http.StatusInternalServerError)
		return
	}
	var node *database.TopologyNode
	for _, n := range nodes {
		if n.ID == id {
			node = n
			break
		}
	}
	if node == nil {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	interfaces, err := db.Topology().ListInterfaces(r.Context(), node.ID)
	if err != nil {
		logger.ErrorContext(r.Context(), "list interfaces failed", "node_id", node.ID, "error", err)
	}
	links, err := db.Topology().ListLinks(r.Context(), node.ID)
	if err != nil {
		logger.ErrorContext(r.Context(), "list links failed", "node_id", node.ID, "error", err)
	}

	writeJSON(w, r, map[string]any{
		"node":       encodeNode(node),
		"interfaces": encodeInterfaces(interfaces),
		"links":      encodeLinks(links),
	})
}

// handleTopologyLinks serves GET /api/v1/topology/links. Without
// ?node_id, returns the global edge list; with ?node_id, returns
// the edges incident to that node.
func (s *Server) handleTopologyLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		http.Error(w, "node_id query parameter required", http.StatusBadRequest)
		return
	}

	links, err := db.Topology().ListLinks(r.Context(), nodeID)
	if err != nil {
		logger.ErrorContext(r.Context(), "list links failed", "node_id", nodeID, "error", err)
		http.Error(w, "Failed to list links", http.StatusInternalServerError)
		return
	}
	writeTopologyJSON(w, r, "links", encodeLinks(links))
}

// handleTopologyARP serves GET /api/v1/topology/arp. Filters via
// ?node_id (source node), ?since (RFC3339), ?limit. Returns 200 with
// a JSON envelope {count, bindings}. The bindings come from the ARP
// reconciler which folds repeated observations into one row per
// (source_node, if_index, ip_address).
//
// Query string is parsed inline (rather than via a parseFn helper)
// because /topology/arp filters on node_id rather than device_type,
// so the parser shape differs from parseTopologyListOptions enough
// that sharing would just push the divergence into the helper.
func (s *Server) handleTopologyARP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	opts := database.TopologyARPListOptions{
		ClientID:     q.Get("client_id"),
		SourceNodeID: q.Get("node_id"),
		Limit:        topologyDefaultLimit,
	}
	if sinceRaw := q.Get("since"); sinceRaw != "" {
		t, err := time.Parse(time.RFC3339, sinceRaw)
		if err != nil {
			http.Error(w, "invalid 'since' (expect RFC3339)", http.StatusBadRequest)
			return
		}
		opts.Since = t
	}
	if limitRaw := q.Get("limit"); limitRaw != "" {
		n, err := strconv.Atoi(limitRaw)
		if err != nil || n < 1 {
			http.Error(w, "invalid 'limit' (positive integer)", http.StatusBadRequest)
			return
		}
		if n > topologyMaxLimit {
			n = topologyMaxLimit
		}
		opts.Limit = n
	}

	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	bindings, err := db.Topology().ListARPBindings(r.Context(), opts)
	if err != nil {
		logging.FromContext(r.Context()).ErrorContext(r.Context(),
			"list topology_arp_bindings failed", "error", err)
		http.Error(w, "Failed to list ARP bindings", http.StatusInternalServerError)
		return
	}
	writeTopologyJSON(w, r, "bindings", encodeARPBindings(bindings))
}

// encodeARPBindings flattens TopologyARPBinding rows for JSON.
func encodeARPBindings(bindings []*database.TopologyARPBinding) []map[string]any {
	out := make([]map[string]any, 0, len(bindings))
	for _, b := range bindings {
		out = append(out, map[string]any{
			"id":           b.ID,
			"clientId":     b.ClientID,
			"sourceNodeId": b.SourceNodeID,
			"ifIndex":      b.IfIndex,
			"ipAddress":    b.IPAddress,
			"macAddress":   b.MACAddress,
			"mediaType":    b.MediaType,
			"lastSeen":     formatTime(b.LastSeen),
		})
	}
	return out
}

// parseTopologyListOptions extracts query-string filters for the
// nodes endpoint. Returns 400-shaped errors via plain text.
func parseTopologyListOptions(r *http.Request) (database.TopologyListOptions, error) {
	q := r.URL.Query()
	opts := database.TopologyListOptions{
		ClientID:   q.Get("client_id"),
		DeviceType: q.Get("device_type"),
		Limit:      topologyDefaultLimit,
	}
	if sinceRaw := q.Get("since"); sinceRaw != "" {
		t, err := time.Parse(time.RFC3339, sinceRaw)
		if err != nil {
			return database.TopologyListOptions{}, errors.New("invalid 'since' (expect RFC3339)")
		}
		opts.SeenSince = t
	}
	if limitRaw := q.Get("limit"); limitRaw != "" {
		n, err := strconv.Atoi(limitRaw)
		if err != nil || n < 1 {
			return database.TopologyListOptions{}, errors.New("invalid 'limit' (positive integer)")
		}
		if n > topologyMaxLimit {
			n = topologyMaxLimit
		}
		opts.Limit = n
	}
	return opts, nil
}

// encodeNode flattens a TopologyNode for JSON. Done explicitly
// (rather than tagging the DB struct) so the wire format stays
// stable when DB columns evolve.
func encodeNode(n *database.TopologyNode) map[string]any {
	return map[string]any{
		"id":           n.ID,
		"clientId":     n.ClientID,
		"identityHash": n.IdentityHash,
		"displayName":  n.DisplayName,
		"deviceType":   n.DeviceType,
		"chassisId":    n.ChassisID,
		"sysName":      n.SysName,
		"primaryMac":   n.PrimaryMAC,
		"primaryIp":    n.PrimaryIP,
		"firstSeen":    formatTime(n.FirstSeen),
		"lastSeen":     formatTime(n.LastSeen),
		"metadata":     rawJSON(n.MetadataJSON),
	}
}

func encodeNodes(nodes []*database.TopologyNode) []map[string]any {
	out := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, encodeNode(n))
	}
	return out
}

func encodeInterfaces(ifaces []*database.TopologyInterface) []map[string]any {
	out := make([]map[string]any, 0, len(ifaces))
	for _, i := range ifaces {
		out = append(out, map[string]any{
			"id":            i.ID,
			"nodeId":        i.NodeID,
			"ifIndex":       i.IfIndex,
			"ifName":        i.IfName,
			"ifDescr":       i.IfDescr,
			"ifAlias":       i.IfAlias,
			"ifType":        i.IfType,
			"ifAdminStatus": i.IfAdminStatus,
			"ifOperStatus":  i.IfOperStatus,
			"ifPhysAddr":    i.IfPhysAddr,
			"speedBps":      i.SpeedBps,
			"lastSeen":      formatTime(i.LastSeen),
		})
	}
	return out
}

func encodeLinks(links []*database.TopologyLink) []map[string]any {
	out := make([]map[string]any, 0, len(links))
	for _, l := range links {
		out = append(out, map[string]any{
			"id":              l.ID,
			"sourceNodeId":    l.SourceNodeID,
			"targetNodeId":    l.TargetNodeID,
			"sourceInterface": l.SourceInterface,
			"targetInterface": l.TargetInterface,
			"linkType":        l.LinkType,
			"status":          l.Status,
			"speedMbps":       l.SpeedMbps,
			"utilizationPct":  l.UtilizationPct,
			"firstSeen":       formatTime(l.FirstSeen),
			"lastSeen":        formatTime(l.LastSeen),
			"evidence":        rawJSON(l.EvidenceJSON),
		})
	}
	return out
}

// rawJSON returns the payload as a [json.RawMessage] when it
// parses, otherwise an empty object. Keeps the wire format
// predictable for clients that walk the response shape.
func rawJSON(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage("{}")
	}
	if !json.Valid([]byte(s)) {
		return json.RawMessage("{}")
	}
	return json.RawMessage(s)
}

// formatTime returns ISO-8601 UTC, or "" for zero values so the
// client can distinguish "never seen" from a real timestamp.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// writeTopologyJSON wraps a result list with a count + envelope so
// the UI doesn't have to parse a bare array. Standard shape across
// every topology list endpoint.
func writeTopologyJSON(w http.ResponseWriter, r *http.Request, key string, payload any) {
	writeJSON(w, r, map[string]any{
		jsonKeyCount: lenOf(payload),
		key:          payload,
	})
}

// writeJSON sets Content-Type + 200, then encodes body. Wraps the
// common pattern used across every other handler so we don't repeat
// the header/encode/log triple at each call site. Callers that need
// a non-200 status set the header themselves before calling — none
// do today, which is why the helper hard-codes 200.
func writeJSON(w http.ResponseWriter, r *http.Request, body any) {
	logger := logging.FromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.WarnContext(r.Context(), "writeJSON encode failed", "error", err)
	}
}

// lenOf returns len(v) for slice / map types or 0 otherwise.
// Used by the envelope helper to surface count without forcing
// callers to type-assert.
func lenOf(v any) int {
	switch s := v.(type) {
	case []map[string]any:
		return len(s)
	case map[string]any:
		return len(s)
	}
	return 0
}
