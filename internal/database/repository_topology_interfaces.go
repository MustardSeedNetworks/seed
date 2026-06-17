package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// UpsertInterface inserts or updates one topology_interfaces row.
// Reconcilers call this once per if_table row per node per poll.
func (r *TopologyRepository) UpsertInterface(ctx context.Context, iface *topology.Interface) error {
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
) ([]*topology.Interface, error) {
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

	var out []*topology.Interface
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

func scanTopologyInterface(scan func(...any) error) (*topology.Interface, error) {
	var (
		iface       topology.Interface
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
		return nil, topology.ErrTopologyNodeNotFound
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
