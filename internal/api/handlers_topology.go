package api

// Topology read-only endpoints (Stage A5.1) expose the fat-Node
// graph that the Stage A4 reconcilers maintain. All handlers are
// GET-only.
//
// They do NOT yet respect the authenticated session's client — the client
// filter is read from the `client_id` query parameter below, so a caller
// names its own tenant. The session now carries an unforgeable client claim
// (internal/auth/client_context.go); converting these reads to it is #1797's
// job, which replaces the whole conflated seam rather than patching the
// filter here.
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

	"github.com/invopop/jsonschema"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// writeTopologyError maps a topology read error to its HTTP status: the store
// being unwired → 503 (the prior "Database not initialized"), a missing node →
// 404, anything else → 500 with genericMsg.
func writeTopologyError(w http.ResponseWriter, r *http.Request, err error, genericMsg string) {
	switch {
	case errors.Is(err, topology.ErrUnavailable):
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
	case errors.Is(err, topology.ErrNodeNotFound):
		writeError(w, r, http.StatusNotFound, ErrCodeNotFound, "Node not found")
	default:
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, genericMsg)
	}
}

// topologyPathPrefix is the route root for every topology handler.
// Used to strip the prefix when extracting path parameters.
const topologyPathPrefix = APIVersionPrefix + "/topology/"

// topologyMaxLimit caps the per-page result size for all topology
// list endpoints. Larger values risk exceeding the JSON encoder's
// default buffer on enterprise-scale graphs (5k+ nodes).
const topologyMaxLimit = 1000

// topologyDefaultLimit is the page size when ?limit isn't provided.
const topologyDefaultLimit = 200

// TopologyNode is one node of the topology graph. Timestamps are RFC 3339
// UTC, or "" when never set. Metadata carries the poll's extra scalars
// (vendor, sysObjectID and the like) as a JSON object.
type TopologyNode struct {
	ID           string          `json:"id"`
	ClientID     string          `json:"clientId"`
	IdentityHash string          `json:"identityHash"`
	DisplayName  string          `json:"displayName"`
	DeviceType   string          `json:"deviceType"`
	ChassisID    string          `json:"chassisId"`
	SysName      string          `json:"sysName"`
	PrimaryMAC   string          `json:"primaryMac"`
	PrimaryIP    string          `json:"primaryIp"`
	FirstSeen    string          `json:"firstSeen"`
	LastSeen     string          `json:"lastSeen"`
	Metadata     json.RawMessage `json:"metadata"     jsonschema:"type=object"`
}

// JSONSchemaExtend opens metadata to any keys; the reflector otherwise closes
// every object.
func (TopologyNode) JSONSchemaExtend(s *jsonschema.Schema) {
	s.Properties.Value("metadata").AdditionalProperties = jsonschema.TrueSchema
}

// TopologyInterface is one SNMP interface row of a node.
type TopologyInterface struct {
	ID            int64  `json:"id"`
	NodeID        string `json:"nodeId"`
	IfIndex       uint32 `json:"ifIndex"`
	IfName        string `json:"ifName"`
	IfDescr       string `json:"ifDescr"`
	IfAlias       string `json:"ifAlias"`
	IfType        uint32 `json:"ifType"`
	IfAdminStatus int    `json:"ifAdminStatus"`
	IfOperStatus  int    `json:"ifOperStatus"`
	IfPhysAddr    string `json:"ifPhysAddr"`
	SpeedBps      uint64 `json:"speedBps"`
	LastSeen      string `json:"lastSeen"`
}

// TopologyLink is one edge between two nodes. Evidence is the reconciler's
// record of what established it, as a JSON object.
type TopologyLink struct {
	ID              string          `json:"id"`
	SourceNodeID    string          `json:"sourceNodeId"`
	TargetNodeID    string          `json:"targetNodeId"`
	SourceInterface string          `json:"sourceInterface"`
	TargetInterface string          `json:"targetInterface"`
	LinkType        string          `json:"linkType"`
	Status          string          `json:"status"`
	SpeedMbps       uint32          `json:"speedMbps"`
	UtilizationPct  float64         `json:"utilizationPct"`
	FirstSeen       string          `json:"firstSeen"`
	LastSeen        string          `json:"lastSeen"`
	Evidence        json.RawMessage `json:"evidence"        jsonschema:"type=object"`
}

// JSONSchemaExtend opens evidence to any keys; the reflector otherwise closes
// every object.
func (TopologyLink) JSONSchemaExtend(s *jsonschema.Schema) {
	s.Properties.Value("evidence").AdditionalProperties = jsonschema.TrueSchema
}

// TopologyARPBinding is one IP-to-MAC binding harvested from a node's ARP
// table.
type TopologyARPBinding struct {
	ID           int64  `json:"id"`
	ClientID     string `json:"clientId"`
	SourceNodeID string `json:"sourceNodeId"`
	IfIndex      uint32 `json:"ifIndex"`
	IPAddress    string `json:"ipAddress"`
	MACAddress   string `json:"macAddress"`
	MediaType    int    `json:"mediaType"`
	LastSeen     string `json:"lastSeen"`
}

// TopologyNodeListResponse is the GET /topology/nodes envelope.
type TopologyNodeListResponse struct {
	Count int            `json:"count"`
	Nodes []TopologyNode `json:"nodes"`
}

// TopologyNodeDetailResponse is GET /topology/nodes/{id}: the node with its
// interfaces and incident links.
type TopologyNodeDetailResponse struct {
	Node       TopologyNode        `json:"node"`
	Interfaces []TopologyInterface `json:"interfaces"`
	Links      []TopologyLink      `json:"links"`
}

// TopologyLinkListResponse is the GET /topology/links envelope.
type TopologyLinkListResponse struct {
	Count int            `json:"count"`
	Links []TopologyLink `json:"links"`
}

// TopologyARPListResponse is the GET /topology/arp envelope.
type TopologyARPListResponse struct {
	Count    int                  `json:"count"`
	Bindings []TopologyARPBinding `json:"bindings"`
}

// handleTopologyNodes serves GET /api/v1/topology/nodes. Filters via
// ?device_type, ?since (RFC3339), ?limit. Returns 200 with a JSON
// array. Empty results are not 404 — that's reserved for malformed
// requests.
func (s *Server) handleTopologyNodes(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	opts, parseErr := parseTopologyListOptions(r)
	if parseErr != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, parseErr.Error())
		return
	}

	nodes, err := s.topologyQueries.Nodes(r.Context(), opts)
	if err != nil {
		logger.ErrorContext(r.Context(), "list topology_nodes failed", "error", err)
		writeTopologyError(w, r, err, "Failed to list nodes")
		return
	}

	enc := encodeNodes(nodes)
	sendJSONResponse(w, logger, http.StatusOK, TopologyNodeListResponse{Count: len(enc), Nodes: enc})
}

// handleTopologyNodeByID serves GET /api/v1/topology/nodes/{id}.
// Returns the node plus its interfaces and links — one HTTP call
// per "device detail" page in the UI.
func (s *Server) handleTopologyNodeByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, topologyPathPrefix+"nodes/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid node id")
		return
	}
	logger := logging.FromContext(r.Context())

	detail, err := s.topologyQueries.Node(r.Context(), id)
	if err != nil {
		logger.ErrorContext(r.Context(), "load topology node failed", "node_id", id, "error", err)
		writeTopologyError(w, r, err, "Failed to load node")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, TopologyNodeDetailResponse{
		Node:       encodeNode(detail.Node),
		Interfaces: encodeInterfaces(detail.Interfaces),
		Links:      encodeLinks(detail.Links),
	})
}

// handleTopologyLinks serves GET /api/v1/topology/links. Without
// ?node_id, returns the global edge list; with ?node_id, returns
// the edges incident to that node.
func (s *Server) handleTopologyLinks(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	nodeID := r.URL.Query().Get("node_id")

	links, err := s.topologyQueries.Links(r.Context(), nodeID)
	if err != nil {
		logger.ErrorContext(r.Context(), "list links failed", "node_id", nodeID, "error", err)
		writeTopologyError(w, r, err, "Failed to list links")
		return
	}
	enc := encodeLinks(links)
	sendJSONResponse(w, logger, http.StatusOK, TopologyLinkListResponse{Count: len(enc), Links: enc})
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
	q := r.URL.Query()
	opts := topology.ARPListOptions{
		ClientID:     q.Get("client_id"),
		SourceNodeID: q.Get("node_id"),
		Limit:        topologyDefaultLimit,
	}
	if sinceRaw := q.Get("since"); sinceRaw != "" {
		t, err := time.Parse(time.RFC3339, sinceRaw)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "invalid 'since' (expect RFC3339)")
			return
		}
		opts.Since = t
	}
	if limitRaw := q.Get("limit"); limitRaw != "" {
		n, err := strconv.Atoi(limitRaw)
		if err != nil || n < 1 {
			writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "invalid 'limit' (positive integer)")
			return
		}
		if n > topologyMaxLimit {
			n = topologyMaxLimit
		}
		opts.Limit = n
	}

	bindings, err := s.topologyQueries.ARP(r.Context(), opts)
	if err != nil {
		logging.FromContext(r.Context()).ErrorContext(r.Context(),
			"list topology_arp_bindings failed", "error", err)
		writeTopologyError(w, r, err, "Failed to list ARP bindings")
		return
	}
	enc := encodeARPBindings(bindings)
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
		TopologyARPListResponse{Count: len(enc), Bindings: enc})
}

func encodeARPBindings(bindings []*topology.ARPBinding) []TopologyARPBinding {
	out := make([]TopologyARPBinding, 0, len(bindings))
	for _, b := range bindings {
		out = append(out, TopologyARPBinding{
			ID:           b.ID,
			ClientID:     b.ClientID,
			SourceNodeID: b.SourceNodeID,
			IfIndex:      b.IfIndex,
			IPAddress:    b.IPAddress,
			MACAddress:   b.MACAddress,
			MediaType:    b.MediaType,
			LastSeen:     formatTime(b.LastSeen),
		})
	}
	return out
}

// parseTopologyListOptions extracts query-string filters for the
// nodes endpoint. Returns 400-shaped errors via plain text.
func parseTopologyListOptions(r *http.Request) (topology.ListOptions, error) {
	q := r.URL.Query()
	opts := topology.ListOptions{
		ClientID:   q.Get("client_id"),
		DeviceType: q.Get("device_type"),
		Limit:      topologyDefaultLimit,
	}
	if sinceRaw := q.Get("since"); sinceRaw != "" {
		t, err := time.Parse(time.RFC3339, sinceRaw)
		if err != nil {
			return topology.ListOptions{}, errors.New("invalid 'since' (expect RFC3339)")
		}
		opts.SeenSince = t
	}
	if limitRaw := q.Get("limit"); limitRaw != "" {
		n, err := strconv.Atoi(limitRaw)
		if err != nil || n < 1 {
			return topology.ListOptions{}, errors.New("invalid 'limit' (positive integer)")
		}
		if n > topologyMaxLimit {
			n = topologyMaxLimit
		}
		opts.Limit = n
	}
	return opts, nil
}

// encodeNode flattens a Node for JSON. Done explicitly
// (rather than tagging the DB struct) so the wire format stays
// stable when DB columns evolve.
func encodeNode(n *topology.Node) TopologyNode {
	return TopologyNode{
		ID:           n.ID,
		ClientID:     n.ClientID,
		IdentityHash: n.IdentityHash,
		DisplayName:  n.DisplayName,
		DeviceType:   n.DeviceType,
		ChassisID:    n.ChassisID,
		SysName:      n.SysName,
		PrimaryMAC:   n.PrimaryMAC,
		PrimaryIP:    n.PrimaryIP,
		FirstSeen:    formatTime(n.FirstSeen),
		LastSeen:     formatTime(n.LastSeen),
		Metadata:     rawJSON(n.MetadataJSON),
	}
}

func encodeNodes(nodes []*topology.Node) []TopologyNode {
	out := make([]TopologyNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, encodeNode(n))
	}
	return out
}

func encodeInterfaces(ifaces []*topology.Interface) []TopologyInterface {
	out := make([]TopologyInterface, 0, len(ifaces))
	for _, i := range ifaces {
		out = append(out, TopologyInterface{
			ID:            i.ID,
			NodeID:        i.NodeID,
			IfIndex:       i.IfIndex,
			IfName:        i.IfName,
			IfDescr:       i.IfDescr,
			IfAlias:       i.IfAlias,
			IfType:        i.IfType,
			IfAdminStatus: i.IfAdminStatus,
			IfOperStatus:  i.IfOperStatus,
			IfPhysAddr:    i.IfPhysAddr,
			SpeedBps:      i.SpeedBps,
			LastSeen:      formatTime(i.LastSeen),
		})
	}
	return out
}

func encodeLinks(links []*topology.Link) []TopologyLink {
	out := make([]TopologyLink, 0, len(links))
	for _, l := range links {
		out = append(out, TopologyLink{
			ID:              l.ID,
			SourceNodeID:    l.SourceNodeID,
			TargetNodeID:    l.TargetNodeID,
			SourceInterface: l.SourceInterface,
			TargetInterface: l.TargetInterface,
			LinkType:        l.LinkType,
			Status:          l.Status,
			SpeedMbps:       l.SpeedMbps,
			UtilizationPct:  l.UtilizationPct,
			FirstSeen:       formatTime(l.FirstSeen),
			LastSeen:        formatTime(l.LastSeen),
			Evidence:        rawJSON(l.EvidenceJSON),
		})
	}
	return out
}

// rawJSON returns the payload as a [json.RawMessage] when it is a JSON
// object, otherwise an empty object, so metadata and evidence always match
// their documented object type.
func rawJSON(s string) json.RawMessage {
	var obj map[string]json.RawMessage
	if json.Unmarshal([]byte(s), &obj) != nil || obj == nil {
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
