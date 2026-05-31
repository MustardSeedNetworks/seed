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
func (r *TopologyRepository) GetByIdentityHash(ctx context.Context, hash string) (*TopologyNode, error) {
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
func (r *TopologyRepository) Upsert(ctx context.Context, node *TopologyNode) (*TopologyNode, error) {
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
func (r *TopologyRepository) List(ctx context.Context, opts TopologyListOptions) ([]*TopologyNode, error) {
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
