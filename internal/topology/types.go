package topology

import (
	"errors"
	"time"
)

// ErrTopologyNodeNotFound is returned when a node lookup misses.
var ErrTopologyNodeNotFound = errors.New("topology node not found")

// Node is one row of topology_nodes. The fat-Node carries
// every observation source's contribution: identity from sys_info,
// per-interface state from if_table (folded into MetadataJSON),
// addresses from arp, neighbors from lldp/cdp/fdp. Reconcilers in
// internal/topology own writes; alert rules + the operator UI own
// reads.
type Node struct {
	ID           string
	ClientID     string
	IdentityHash string
	DisplayName  string
	DeviceType   string
	ChassisID    string
	SysName      string
	PrimaryMAC   string
	PrimaryIP    string
	FirstSeen    time.Time
	LastSeen     time.Time
	ExpiresAt    time.Time
	MetadataJSON string
}

// ListOptions narrows a topology_nodes query.
type ListOptions struct {
	ClientID   string
	DeviceType string
	SeenSince  time.Time
	Limit      int
}

// ARPBinding mirrors one row of topology_arp_bindings.
// Reconciled from arp observations; the row's MAC is the join key
// used to backfill node.primary_ip when a binding matches a known
// node's chassis/primary MAC.
type ARPBinding struct {
	ID           int64
	ClientID     string
	SourceNodeID string
	IfIndex      uint32
	IPAddress    string
	MACAddress   string
	MediaType    int
	LastSeen     time.Time
}

// ARPListOptions filters the ListARPBindings query.
// Empty fields mean "no filter".
type ARPListOptions struct {
	ClientID     string
	SourceNodeID string
	Since        time.Time
	Limit        int
}

// Link mirrors one row of topology_links. Edges are
// identity-merged the same way nodes are — the link ID is derived
// from (source_node, source_interface, target_node, target_interface)
// so re-observations of the same physical cable upsert rather than
// duplicate.
type Link struct {
	ID              string
	SourceNodeID    string
	TargetNodeID    string
	SourceInterface string
	TargetInterface string
	LinkType        string // "lldp", "cdp", "fdp"
	Status          string // "up", "down", "unknown"
	SpeedMbps       uint32
	UtilizationPct  float64
	FirstSeen       time.Time
	LastSeen        time.Time
	EvidenceJSON    string
}

// Interface mirrors one row of topology_interfaces.
// Reconciled from if_table observations; columns shaped so alert
// rules can index admin/oper status and speed without parsing JSON.
type Interface struct {
	ID            int64
	NodeID        string
	IfIndex       uint32
	IfName        string
	IfDescr       string
	IfAlias       string
	IfType        uint32
	IfAdminStatus int
	IfOperStatus  int
	IfPhysAddr    string
	SpeedBps      uint64
	LastSeen      time.Time
}
