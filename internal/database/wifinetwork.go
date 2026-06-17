package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WiFiNetworkDB represents a WiFi network in the database.
type WiFiNetworkDB struct {
	ID                  string    `json:"id"`
	SSID                string    `json:"ssid"`
	IsHidden            bool      `json:"isHidden"`
	SecurityType        string    `json:"securityType"`
	AuthorizationStatus string    `json:"authorizationStatus"`
	FirstSeen           time.Time `json:"firstSeen"`
	LastSeen            time.Time `json:"lastSeen"`
	Metadata            string    `json:"metadata,omitempty"`
}

// CreateWiFiNetwork creates a new WiFi network record.
func (r *DiscoveryRepository) CreateWiFiNetwork(ctx context.Context, network *WiFiNetworkDB) error {
	if network.ID == "" {
		network.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if network.FirstSeen.IsZero() {
		network.FirstSeen = now
	}
	network.LastSeen = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO wifi_networks
		(id, ssid, is_hidden, security_type, authorization_status, first_seen, last_seen, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, network.ID, network.SSID, boolToInt(network.IsHidden), network.SecurityType,
		network.AuthorizationStatus, network.FirstSeen.Format(time.RFC3339),
		network.LastSeen.Format(time.RFC3339), network.Metadata)
	if err != nil {
		return fmt.Errorf("failed to create wifi network: %w", err)
	}
	return nil
}

// GetWiFiNetwork retrieves a WiFi network by ID.
func (r *DiscoveryRepository) GetWiFiNetwork(ctx context.Context, id string) (*WiFiNetworkDB, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, ssid, is_hidden, security_type, authorization_status, first_seen, last_seen, metadata_json
		FROM wifi_networks WHERE id = ?
	`, id)
	return r.scanWiFiNetwork(row)
}

// GetWiFiNetworkBySSID retrieves a WiFi network by SSID.
func (r *DiscoveryRepository) GetWiFiNetworkBySSID(ctx context.Context, ssid string) (*WiFiNetworkDB, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, ssid, is_hidden, security_type, authorization_status, first_seen, last_seen, metadata_json
		FROM wifi_networks WHERE ssid = ?
	`, ssid)
	return r.scanWiFiNetwork(row)
}

// ListWiFiNetworks retrieves all WiFi networks.
func (r *DiscoveryRepository) ListWiFiNetworks(
	ctx context.Context,
	opts WiFiNetworkListOptions,
) ([]*WiFiNetworkDB, error) {
	query := `
		SELECT id, ssid, is_hidden, security_type, authorization_status, first_seen, last_seen, metadata_json
		FROM wifi_networks WHERE 1=1
	`
	var args []any

	if opts.SecurityType != "" {
		query += " AND security_type = ?"
		args = append(args, opts.SecurityType)
	}
	if opts.AuthorizationStatus != "" {
		query += " AND authorization_status = ?"
		args = append(args, opts.AuthorizationStatus)
	}
	if opts.HiddenOnly {
		query += " AND is_hidden = 1"
	}

	query += " ORDER BY last_seen DESC"

	if opts.Limit > 0 {
		query += sqlLimit
		args = append(args, opts.Limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list wifi networks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var networks []*WiFiNetworkDB
	for rows.Next() {
		n, scanErr := r.scanWiFiNetworkFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		networks = append(networks, n)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating wifi networks: %w", rowsErr)
	}
	return networks, nil
}

// WiFiNetworkListOptions specifies criteria for listing WiFi networks.
type WiFiNetworkListOptions struct {
	SecurityType        string
	AuthorizationStatus string
	HiddenOnly          bool
	Limit               int
}

// UpsertWiFiNetwork creates or updates a WiFi network by SSID.
func (r *DiscoveryRepository) UpsertWiFiNetwork(ctx context.Context, network *WiFiNetworkDB) error {
	existing, err := r.GetWiFiNetworkBySSID(ctx, network.SSID)
	if err != nil && !errors.Is(err, ErrWiFiNetworkNotFound) {
		return fmt.Errorf("failed to check existing network: %w", err)
	}

	if existing != nil {
		network.ID = existing.ID
		network.FirstSeen = existing.FirstSeen
		return r.updateWiFiNetwork(ctx, network)
	}
	return r.CreateWiFiNetwork(ctx, network)
}

func (r *DiscoveryRepository) updateWiFiNetwork(ctx context.Context, network *WiFiNetworkDB) error {
	network.LastSeen = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE wifi_networks
		SET is_hidden = ?, security_type = ?, authorization_status = ?, last_seen = ?, metadata_json = ?
		WHERE id = ?
	`, boolToInt(network.IsHidden), network.SecurityType, network.AuthorizationStatus,
		network.LastSeen.Format(time.RFC3339), network.Metadata, network.ID)
	if err != nil {
		return fmt.Errorf("failed to update wifi network: %w", err)
	}
	return nil
}

func (r *DiscoveryRepository) scanWiFiNetwork(row *sql.Row) (*WiFiNetworkDB, error) {
	var n WiFiNetworkDB
	var firstSeen, lastSeen string
	var isHidden int
	var metadata sql.NullString

	err := row.Scan(&n.ID, &n.SSID, &isHidden, &n.SecurityType, &n.AuthorizationStatus,
		&firstSeen, &lastSeen, &metadata)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWiFiNetworkNotFound
		}
		return nil, fmt.Errorf("failed to scan wifi network: %w", err)
	}

	n.IsHidden = isHidden == 1
	n.Metadata = metadata.String
	if t, parseErr := time.Parse(time.RFC3339, firstSeen); parseErr == nil {
		n.FirstSeen = t
	}
	if t, parseErr := time.Parse(time.RFC3339, lastSeen); parseErr == nil {
		n.LastSeen = t
	}
	return &n, nil
}

func (r *DiscoveryRepository) scanWiFiNetworkFromRows(rows *sql.Rows) (*WiFiNetworkDB, error) {
	var n WiFiNetworkDB
	var firstSeen, lastSeen string
	var isHidden int
	var metadata sql.NullString

	err := rows.Scan(&n.ID, &n.SSID, &isHidden, &n.SecurityType, &n.AuthorizationStatus,
		&firstSeen, &lastSeen, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to scan wifi network: %w", err)
	}

	n.IsHidden = isHidden == 1
	n.Metadata = metadata.String
	if t, parseErr := time.Parse(time.RFC3339, firstSeen); parseErr == nil {
		n.FirstSeen = t
	}
	if t, parseErr := time.Parse(time.RFC3339, lastSeen); parseErr == nil {
		n.LastSeen = t
	}
	return &n, nil
}

// WiFi Access Point operations
