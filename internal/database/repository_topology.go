package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// TopologyNode is one row of topology_nodes. The fat-Node carries
// every observation source's contribution: identity from sys_info,
// per-interface state from if_table (folded into MetadataJSON),
// addresses from arp, neighbors from lldp/cdp/fdp. Reconcilers in
// internal/topology own writes; alert rules + the operator UI own
// reads.
type TopologyNode struct {
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

// TopologyRepository owns CRUD over topology_nodes + topology_links.
// Stage A4.1 wires the nodes side; links arrive in A4.3.
type TopologyRepository struct {
	db *DB
}

// GetByIdentityHash returns the node with the given identity hash
// or ErrTopologyNodeNotFound when absent.
func (r *TopologyRepository) GetByIdentityHash(
	ctx context.Context,
	hash string,
) (*TopologyNode, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, client_id, identity_hash, display_name, device_type,
		       chassis_id, sys_name, primary_mac, primary_ip,
		       first_seen, last_seen, expires_at, metadata_json
		FROM topology_nodes WHERE identity_hash = ?
	`, hash)
	return scanTopologyNode(row.Scan)
}

// ErrTopologyNodeNotFound is returned when a node lookup misses.
var ErrTopologyNodeNotFound = errors.New("topology node not found")

// Upsert inserts or updates a node by identity_hash. The hash is
// the merge key — same hash = same physical device regardless of
// how it was observed. FirstSeen is preserved on update; LastSeen
// is always set to node.LastSeen (which the caller should set to
// the observation's ObservedAt).
//
// Returns the up-to-date row including any FirstSeen the existing
// record carried.
func (r *TopologyRepository) Upsert(
	ctx context.Context,
	node *TopologyNode,
) (*TopologyNode, error) {
	if node.IdentityHash == "" {
		return nil, errors.New("topology_nodes: IdentityHash required")
	}
	if node.ID == "" {
		return nil, errors.New("topology_nodes: ID required")
	}
	if node.ClientID == "" {
		node.ClientID = "default"
	}
	if node.LastSeen.IsZero() {
		node.LastSeen = time.Now().UTC()
	}
	if node.FirstSeen.IsZero() {
		node.FirstSeen = node.LastSeen
	}

	existing, err := r.GetByIdentityHash(ctx, node.IdentityHash)
	switch {
	case err == nil:
		// Update — preserve FirstSeen from existing row.
		node.FirstSeen = existing.FirstSeen
		node.ID = existing.ID // keep stable id across upserts
		if updErr := r.update(ctx, node); updErr != nil {
			return nil, updErr
		}
	case errors.Is(err, ErrTopologyNodeNotFound):
		if insErr := r.insert(ctx, node); insErr != nil {
			return nil, insErr
		}
	default:
		return nil, err
	}
	return node, nil
}

func (r *TopologyRepository) insert(ctx context.Context, node *TopologyNode) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO topology_nodes
		  (id, client_id, identity_hash, display_name, device_type,
		   chassis_id, sys_name, primary_mac, primary_ip,
		   first_seen, last_seen, expires_at, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		node.ID, node.ClientID, node.IdentityHash, node.DisplayName,
		toNullString(node.DeviceType), toNullString(node.ChassisID),
		toNullString(node.SysName), toNullString(node.PrimaryMAC),
		toNullString(node.PrimaryIP),
		node.FirstSeen.UTC().Format(time.RFC3339Nano),
		node.LastSeen.UTC().Format(time.RFC3339Nano),
		toNullTime(node.ExpiresAt),
		toNullString(node.MetadataJSON),
	)
	if err != nil {
		return fmt.Errorf("insert topology_node: %w", err)
	}
	return nil
}

func (r *TopologyRepository) update(ctx context.Context, node *TopologyNode) error {
	_, err := r.db.Exec(ctx, `
		UPDATE topology_nodes SET
			display_name = ?, device_type = ?, chassis_id = ?, sys_name = ?,
			primary_mac = ?, primary_ip = ?, last_seen = ?, expires_at = ?,
			metadata_json = ?
		WHERE identity_hash = ?
	`,
		node.DisplayName,
		toNullString(node.DeviceType), toNullString(node.ChassisID),
		toNullString(node.SysName), toNullString(node.PrimaryMAC),
		toNullString(node.PrimaryIP),
		node.LastSeen.UTC().Format(time.RFC3339Nano),
		toNullTime(node.ExpiresAt),
		toNullString(node.MetadataJSON),
		node.IdentityHash,
	)
	if err != nil {
		return fmt.Errorf("update topology_node: %w", err)
	}
	return nil
}

// TopologyListOptions narrows a topology_nodes query.
type TopologyListOptions struct {
	ClientID   string
	DeviceType string
	SeenSince  time.Time
	Limit      int
}

// List returns nodes matching opts ordered by LastSeen desc.
func (r *TopologyRepository) List(
	ctx context.Context,
	opts TopologyListOptions,
) ([]*TopologyNode, error) {
	const defaultLimit, maxLimit = 100, 5000

	query, args := newListQueryBuilder(`
		SELECT id, client_id, identity_hash, display_name, device_type,
		       chassis_id, sys_name, primary_mac, primary_ip,
		       first_seen, last_seen, expires_at, metadata_json
		FROM topology_nodes
		WHERE 1=1
	`).
		Where("AND client_id = ?", opts.ClientID).
		Where("AND device_type = ?", opts.DeviceType).
		WhereTime("AND last_seen >= ?", opts.SeenSince).
		OrderLimit("ORDER BY last_seen DESC", opts.Limit, defaultLimit, maxLimit).
		Build()

	return queryRows(ctx, r.db, query, args, scanTopologyNode, "list topology_nodes")
}

// toNullTime returns a NullString that's invalid when t is zero.
// Stored times use RFC3339Nano UTC.
func toNullTime(t time.Time) sql.NullString {
	if t.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(time.RFC3339Nano), Valid: true}
}

// TopologyLink mirrors one row of topology_links. Edges are
// identity-merged the same way nodes are — the link ID is derived
// from (source_node, source_interface, target_node, target_interface)
// so re-observations of the same physical cable upsert rather than
// duplicate.
type TopologyLink struct {
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

// NodeIDForSysName resolves a sys_name back to its node_id by
// looking up topology_nodes. Used by the edge reconciler to map
// an LLDP/CDP neighbor's reported hostname to a known node.
// Returns ErrTopologyNodeNotFound when no node has that sys_name.
func (r *TopologyRepository) NodeIDForSysName(ctx context.Context, clientID, sysName string) (string, error) {
	if clientID == "" {
		clientID = "default"
	}
	if sysName == "" {
		return "", ErrTopologyNodeNotFound
	}
	row := r.db.QueryRow(ctx, `
		SELECT id FROM topology_nodes
		WHERE client_id = ? AND sys_name = ?
		ORDER BY last_seen DESC
		LIMIT 1
	`, clientID, sysName)
	var id string
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrTopologyNodeNotFound
		}
		return "", fmt.Errorf("nodeIDForSysName: %w", err)
	}
	return id, nil
}

// UpsertLink inserts or updates a topology_links row. The link's
// ID is the merge key — the edge reconciler computes it
// deterministically from (source_node, source_interface,
// target_node, target_interface) so two LLDP polls of the same
// physical cable update one row instead of inserting two.
func (r *TopologyRepository) UpsertLink(ctx context.Context, link *TopologyLink) error {
	if link.ID == "" {
		return errors.New("topology_links: ID required")
	}
	if link.SourceNodeID == "" || link.TargetNodeID == "" {
		return errors.New("topology_links: SourceNodeID + TargetNodeID required")
	}
	if link.LinkType == "" {
		link.LinkType = "unknown"
	}
	if link.Status == "" {
		link.Status = "up"
	}
	if link.LastSeen.IsZero() {
		link.LastSeen = time.Now().UTC()
	}
	if link.FirstSeen.IsZero() {
		link.FirstSeen = link.LastSeen
	}

	// ON CONFLICT preserves first_seen; everything else refreshes
	// from the new observation.
	_, err := r.db.Exec(ctx, `
		INSERT INTO topology_links
		  (id, source_node_id, target_node_id, source_interface, target_interface,
		   link_type, status, speed_mbps, utilization_pct,
		   first_seen, last_seen, evidence_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source_interface = excluded.source_interface,
			target_interface = excluded.target_interface,
			link_type = excluded.link_type,
			status = excluded.status,
			speed_mbps = excluded.speed_mbps,
			utilization_pct = excluded.utilization_pct,
			last_seen = excluded.last_seen,
			evidence_json = excluded.evidence_json
	`,
		link.ID, link.SourceNodeID, link.TargetNodeID,
		toNullString(link.SourceInterface), toNullString(link.TargetInterface),
		link.LinkType, link.Status,
		nullableUint32(link.SpeedMbps),
		nullableFloat(link.UtilizationPct),
		link.FirstSeen.UTC().Format(time.RFC3339Nano),
		link.LastSeen.UTC().Format(time.RFC3339Nano),
		toNullString(link.EvidenceJSON),
	)
	if err != nil {
		return fmt.Errorf("upsert topology_link: %w", err)
	}
	return nil
}

// ListLinks returns every link involving nodeID (either source or
// target) ordered by LastSeen desc.
func (r *TopologyRepository) ListLinks(ctx context.Context, nodeID string) ([]*TopologyLink, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, source_node_id, target_node_id, source_interface, target_interface,
		       link_type, status, speed_mbps, utilization_pct,
		       first_seen, last_seen, evidence_json
		FROM topology_links
		WHERE source_node_id = ? OR target_node_id = ?
		ORDER BY last_seen DESC
	`, nodeID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list topology_links: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*TopologyLink
	for rows.Next() {
		link, scanErr := scanTopologyLink(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, link)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("list topology_links iter: %w", rowsErr)
	}
	return out, nil
}

// nullableUint32 returns [sql.NullInt32] with valid=false when v == 0.
// SpeedMbps + UtilizationPct are nullable in the schema so the UI
// can distinguish "no measurement yet" from "0 Mbps". Values above
// [math.MaxInt32] clamp down — SQLite stores INTEGER as int64 in
// the driver but the NullInt32 wrapper is signed, so clamping keeps
// the conversion well-defined for the (rare) terabit-per-second link.
func nullableUint32(v uint32) sql.NullInt32 {
	if v == 0 {
		return sql.NullInt32{}
	}
	const maxInt32 uint32 = 1<<31 - 1
	if v > maxInt32 {
		v = maxInt32
	}
	return sql.NullInt32{Int32: int32(v), Valid: true}
}

func nullableFloat(v float64) sql.NullFloat64 {
	if v == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: v, Valid: true}
}

func scanTopologyLink(scan func(...any) error) (*TopologyLink, error) {
	var (
		link         TopologyLink
		srcIface     sql.NullString
		tgtIface     sql.NullString
		speedMbps    sql.NullInt32
		utilization  sql.NullFloat64
		evidenceJSON sql.NullString
		firstSeenStr string
		lastSeenStr  string
	)
	err := scan(
		&link.ID, &link.SourceNodeID, &link.TargetNodeID,
		&srcIface, &tgtIface, &link.LinkType, &link.Status,
		&speedMbps, &utilization,
		&firstSeenStr, &lastSeenStr, &evidenceJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTopologyNodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan topology_link: %w", err)
	}
	if srcIface.Valid {
		link.SourceInterface = srcIface.String
	}
	if tgtIface.Valid {
		link.TargetInterface = tgtIface.String
	}
	if speedMbps.Valid && speedMbps.Int32 >= 0 {
		link.SpeedMbps = uint32(speedMbps.Int32)
	}
	if utilization.Valid {
		link.UtilizationPct = utilization.Float64
	}
	if evidenceJSON.Valid {
		link.EvidenceJSON = evidenceJSON.String
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, firstSeenStr); perr == nil {
		link.FirstSeen = parsed
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, lastSeenStr); perr == nil {
		link.LastSeen = parsed
	}
	return &link, nil
}

// TopologyInterface mirrors one row of topology_interfaces.
// Reconciled from if_table observations; columns shaped so alert
// rules can index admin/oper status and speed without parsing JSON.
type TopologyInterface struct {
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

// UpsertTargetNode records the (client_id, target_id) -> node_id
// mapping so A4.2+ reconcilers (if_table, lldp, arp, fdb, routing,
// bgp4) can resolve their observations to the right topology node
// without re-decoding sysinfo. The sysinfo reconciler calls this on
// every node upsert.
func (r *TopologyRepository) UpsertTargetNode(
	ctx context.Context,
	clientID, targetID, nodeID string,
	lastSeen time.Time,
) error {
	if targetID == "" {
		return errors.New("topology_target_nodes: TargetID required")
	}
	if nodeID == "" {
		return errors.New("topology_target_nodes: NodeID required")
	}
	if clientID == "" {
		clientID = "default"
	}
	if lastSeen.IsZero() {
		lastSeen = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO topology_target_nodes (client_id, target_id, node_id, last_seen)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(client_id, target_id) DO UPDATE SET
			node_id = excluded.node_id,
			last_seen = excluded.last_seen
	`,
		clientID, targetID, nodeID,
		lastSeen.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("upsert topology_target_nodes: %w", err)
	}
	return nil
}

// NodeIDForTarget resolves (client_id, target_id) -> node_id. Returns
// "" + ErrTopologyNodeNotFound when no mapping exists yet (the
// sysinfo reconciler hasn't seen this target).
func (r *TopologyRepository) NodeIDForTarget(
	ctx context.Context,
	clientID, targetID string,
) (string, error) {
	if clientID == "" {
		clientID = "default"
	}
	row := r.db.QueryRow(ctx,
		`SELECT node_id FROM topology_target_nodes WHERE client_id = ? AND target_id = ?`,
		clientID, targetID,
	)
	var nodeID string
	if err := row.Scan(&nodeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrTopologyNodeNotFound
		}
		return "", fmt.Errorf("nodeIDForTarget: %w", err)
	}
	return nodeID, nil
}

// UpsertInterface inserts or updates one topology_interfaces row.
// Reconcilers call this once per if_table row per node per poll.
func (r *TopologyRepository) UpsertInterface(ctx context.Context, iface *TopologyInterface) error {
	if iface.NodeID == "" {
		return errors.New("topology_interfaces: NodeID required")
	}
	if iface.LastSeen.IsZero() {
		iface.LastSeen = time.Now().UTC()
	}

	const ifaceUpsertSQL = `
		INSERT INTO topology_interfaces
		  (node_id, if_index, if_name, if_descr, if_alias, if_type,
		   if_admin_status, if_oper_status, if_phys_addr, speed_bps,
		   last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id, if_index) DO UPDATE SET
			if_name = excluded.if_name,
			if_descr = excluded.if_descr,
			if_alias = excluded.if_alias,
			if_type = excluded.if_type,
			if_admin_status = excluded.if_admin_status,
			if_oper_status = excluded.if_oper_status,
			if_phys_addr = excluded.if_phys_addr,
			speed_bps = excluded.speed_bps,
			last_seen = excluded.last_seen
	`
	_, err := r.db.Exec(ctx, ifaceUpsertSQL,
		iface.NodeID, iface.IfIndex,
		toNullString(iface.IfName), toNullString(iface.IfDescr),
		toNullString(iface.IfAlias), iface.IfType,
		iface.IfAdminStatus, iface.IfOperStatus,
		toNullString(iface.IfPhysAddr),
		iface.SpeedBps,
		iface.LastSeen.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("upsert topology_interface: %w", err)
	}
	return nil
}

// ListInterfaces returns every interface for a node ordered by
// IfIndex ascending.
func (r *TopologyRepository) ListInterfaces(
	ctx context.Context,
	nodeID string,
) ([]*TopologyInterface, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, node_id, if_index, if_name, if_descr, if_alias,
		       if_type, if_admin_status, if_oper_status, if_phys_addr,
		       speed_bps, last_seen
		FROM topology_interfaces
		WHERE node_id = ?
		ORDER BY if_index ASC
	`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list topology_interfaces: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*TopologyInterface
	for rows.Next() {
		iface, scanErr := scanTopologyInterface(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, iface)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("list topology_interfaces iter: %w", rowsErr)
	}
	return out, nil
}

func scanTopologyInterface(scan func(...any) error) (*TopologyInterface, error) {
	var (
		iface       TopologyInterface
		ifName      sql.NullString
		ifDescr     sql.NullString
		ifAlias     sql.NullString
		ifPhysAddr  sql.NullString
		lastSeenStr string
	)
	err := scan(
		&iface.ID, &iface.NodeID, &iface.IfIndex,
		&ifName, &ifDescr, &ifAlias,
		&iface.IfType, &iface.IfAdminStatus, &iface.IfOperStatus,
		&ifPhysAddr, &iface.SpeedBps, &lastSeenStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTopologyNodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan topology_interface: %w", err)
	}
	if ifName.Valid {
		iface.IfName = ifName.String
	}
	if ifDescr.Valid {
		iface.IfDescr = ifDescr.String
	}
	if ifAlias.Valid {
		iface.IfAlias = ifAlias.String
	}
	if ifPhysAddr.Valid {
		iface.IfPhysAddr = ifPhysAddr.String
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, lastSeenStr); perr == nil {
		iface.LastSeen = parsed
	}
	return &iface, nil
}

func scanTopologyNode(scan func(...any) error) (*TopologyNode, error) {
	var (
		node         TopologyNode
		deviceType   sql.NullString
		chassisID    sql.NullString
		sysName      sql.NullString
		primaryMAC   sql.NullString
		primaryIP    sql.NullString
		expiresAt    sql.NullString
		metadataJSON sql.NullString
		firstSeenStr string
		lastSeenStr  string
	)
	err := scan(
		&node.ID, &node.ClientID, &node.IdentityHash, &node.DisplayName,
		&deviceType, &chassisID, &sysName, &primaryMAC, &primaryIP,
		&firstSeenStr, &lastSeenStr, &expiresAt, &metadataJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTopologyNodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan topology_node: %w", err)
	}
	if deviceType.Valid {
		node.DeviceType = deviceType.String
	}
	if chassisID.Valid {
		node.ChassisID = chassisID.String
	}
	if sysName.Valid {
		node.SysName = sysName.String
	}
	if primaryMAC.Valid {
		node.PrimaryMAC = primaryMAC.String
	}
	if primaryIP.Valid {
		node.PrimaryIP = primaryIP.String
	}
	if metadataJSON.Valid {
		node.MetadataJSON = metadataJSON.String
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, firstSeenStr); perr == nil {
		node.FirstSeen = parsed
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, lastSeenStr); perr == nil {
		node.LastSeen = parsed
	}
	if expiresAt.Valid {
		if parsed, perr := time.Parse(time.RFC3339Nano, expiresAt.String); perr == nil {
			node.ExpiresAt = parsed
		}
	}
	return &node, nil
}
