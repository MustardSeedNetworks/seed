package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// UpsertLink inserts or updates a topology_links row. The link's
// ID is the merge key — the edge reconciler computes it
// deterministically from (source_node, source_interface,
// target_node, target_interface) so two LLDP polls of the same
// physical cable update one row instead of inserting two.
func (r *TopologyRepository) UpsertLink(ctx context.Context, link *topology.Link) error {
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
func (r *TopologyRepository) ListLinks(ctx context.Context, nodeID string) ([]*topology.Link, error) {
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

	var out []*topology.Link
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

func scanTopologyLink(scan func(...any) error) (*topology.Link, error) {
	var (
		link         topology.Link
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
		return nil, topology.ErrTopologyNodeNotFound
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
// "" + topology.ErrTopologyNodeNotFound when no mapping exists yet (the
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
			return "", topology.ErrTopologyNodeNotFound
		}
		return "", fmt.Errorf("nodeIDForTarget: %w", err)
	}
	return nodeID, nil
}
