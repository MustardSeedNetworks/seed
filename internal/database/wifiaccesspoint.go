package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WiFiAccessPointDB represents a WiFi access point in the database.
type WiFiAccessPointDB struct {
	ID           string    `json:"id"`
	BSSID        string    `json:"bssid"`
	SSIDID       string    `json:"ssidId,omitempty"`
	SSIDName     string    `json:"ssidName,omitempty"`
	APName       string    `json:"apName,omitempty"`
	Vendor       string    `json:"vendor,omitempty"`
	Channel      int       `json:"channel"`
	ChannelWidth int       `json:"channelWidth"`
	FrequencyMHz int       `json:"frequencyMhz"`
	Band         string    `json:"band"`
	SignalDBm    int       `json:"signalDbm"`
	NoiseDBm     int       `json:"noiseDbm,omitempty"`
	ClientCount  int       `json:"clientCount"`
	IsAuthorized bool      `json:"isAuthorized"`
	FirstSeen    time.Time `json:"firstSeen"`
	LastSeen     time.Time `json:"lastSeen"`
	Metadata     string    `json:"metadata,omitempty"`
}

// CreateWiFiAccessPoint creates a new WiFi access point record.
func (r *DiscoveryRepository) CreateWiFiAccessPoint(ctx context.Context, ap *WiFiAccessPointDB) error {
	if ap.ID == "" {
		ap.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if ap.FirstSeen.IsZero() {
		ap.FirstSeen = now
	}
	ap.LastSeen = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO wifi_access_points
		(id, bssid, ssid_id, ssid_name, ap_name, vendor, channel, channel_width, frequency_mhz, band,
		 signal_dbm, noise_dbm, client_count, is_authorized, first_seen, last_seen, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ap.ID, ap.BSSID, ap.SSIDID, ap.SSIDName, ap.APName, ap.Vendor, ap.Channel, ap.ChannelWidth,
		ap.FrequencyMHz, ap.Band, ap.SignalDBm, ap.NoiseDBm, ap.ClientCount,
		boolToInt(ap.IsAuthorized), ap.FirstSeen.Format(time.RFC3339),
		ap.LastSeen.Format(time.RFC3339), ap.Metadata)
	if err != nil {
		return fmt.Errorf("failed to create wifi access point: %w", err)
	}
	return nil
}

// GetWiFiAccessPoint retrieves a WiFi access point by ID.
func (r *DiscoveryRepository) GetWiFiAccessPoint(ctx context.Context, id string) (*WiFiAccessPointDB, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, bssid, ssid_id, ssid_name, ap_name, vendor, channel, channel_width, frequency_mhz, band,
		       signal_dbm, noise_dbm, client_count, is_authorized, first_seen, last_seen, metadata_json
		FROM wifi_access_points WHERE id = ?
	`, id)
	return r.scanWiFiAccessPoint(row)
}

// GetWiFiAccessPointByBSSID retrieves a WiFi access point by BSSID.
func (r *DiscoveryRepository) GetWiFiAccessPointByBSSID(ctx context.Context, bssid string) (*WiFiAccessPointDB, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, bssid, ssid_id, ssid_name, ap_name, vendor, channel, channel_width, frequency_mhz, band,
		       signal_dbm, noise_dbm, client_count, is_authorized, first_seen, last_seen, metadata_json
		FROM wifi_access_points WHERE bssid = ?
	`, bssid)
	return r.scanWiFiAccessPoint(row)
}

// ListWiFiAccessPoints retrieves WiFi access points with filters.
func (r *DiscoveryRepository) ListWiFiAccessPoints(
	ctx context.Context,
	opts WiFiAPListOptions,
) ([]*WiFiAccessPointDB, error) {
	query := `
		SELECT id, bssid, ssid_id, ssid_name, ap_name, vendor, channel, channel_width, frequency_mhz, band,
		       signal_dbm, noise_dbm, client_count, is_authorized, first_seen, last_seen, metadata_json
		FROM wifi_access_points WHERE 1=1
	`
	var args []any

	if opts.SSIDID != "" {
		query += " AND ssid_id = ?"
		args = append(args, opts.SSIDID)
	}
	if opts.Band != "" {
		query += " AND band = ?"
		args = append(args, opts.Band)
	}
	if opts.Channel > 0 {
		query += " AND channel = ?"
		args = append(args, opts.Channel)
	}
	if opts.AuthorizedOnly {
		query += " AND is_authorized = 1"
	}
	if opts.UnauthorizedOnly {
		query += " AND is_authorized = 0"
	}
	if opts.MinSignalDBm != 0 {
		query += " AND signal_dbm >= ?"
		args = append(args, opts.MinSignalDBm)
	}

	query += " ORDER BY signal_dbm DESC"

	if opts.Limit > 0 {
		query += sqlLimit
		args = append(args, opts.Limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list wifi access points: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var aps []*WiFiAccessPointDB
	for rows.Next() {
		ap, scanErr := r.scanWiFiAccessPointFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		aps = append(aps, ap)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating wifi access points: %w", rowsErr)
	}
	return aps, nil
}

// WiFiAPListOptions specifies criteria for listing WiFi access points.
type WiFiAPListOptions struct {
	SSIDID           string
	Band             string
	Channel          int
	AuthorizedOnly   bool
	UnauthorizedOnly bool
	MinSignalDBm     int
	Limit            int
}

// UpsertWiFiAccessPoint creates or updates a WiFi access point by BSSID.
func (r *DiscoveryRepository) UpsertWiFiAccessPoint(ctx context.Context, ap *WiFiAccessPointDB) error {
	existing, err := r.GetWiFiAccessPointByBSSID(ctx, ap.BSSID)
	if err != nil && !errors.Is(err, ErrWiFiAccessPointNotFound) {
		return fmt.Errorf("failed to check existing AP: %w", err)
	}

	if existing != nil {
		ap.ID = existing.ID
		ap.FirstSeen = existing.FirstSeen
		return r.updateWiFiAccessPoint(ctx, ap)
	}
	return r.CreateWiFiAccessPoint(ctx, ap)
}

func (r *DiscoveryRepository) updateWiFiAccessPoint(ctx context.Context, ap *WiFiAccessPointDB) error {
	ap.LastSeen = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE wifi_access_points
		SET ssid_id = ?, ssid_name = ?, ap_name = ?, vendor = ?, channel = ?, channel_width = ?,
		    frequency_mhz = ?, band = ?, signal_dbm = ?, noise_dbm = ?, client_count = ?,
		    is_authorized = ?, last_seen = ?, metadata_json = ?
		WHERE id = ?
	`, ap.SSIDID, ap.SSIDName, ap.APName, ap.Vendor, ap.Channel, ap.ChannelWidth,
		ap.FrequencyMHz, ap.Band, ap.SignalDBm, ap.NoiseDBm, ap.ClientCount,
		boolToInt(ap.IsAuthorized), ap.LastSeen.Format(time.RFC3339), ap.Metadata, ap.ID)
	if err != nil {
		return fmt.Errorf("failed to update wifi access point: %w", err)
	}
	return nil
}

func (r *DiscoveryRepository) scanWiFiAccessPoint(row *sql.Row) (*WiFiAccessPointDB, error) {
	var ap WiFiAccessPointDB
	var firstSeen, lastSeen string
	var isAuthorized int
	var ssidID, ssidName, apName, vendor, metadata sql.NullString
	var noiseDBm sql.NullInt64

	err := row.Scan(&ap.ID, &ap.BSSID, &ssidID, &ssidName, &apName, &vendor, &ap.Channel,
		&ap.ChannelWidth, &ap.FrequencyMHz, &ap.Band, &ap.SignalDBm, &noiseDBm,
		&ap.ClientCount, &isAuthorized, &firstSeen, &lastSeen, &metadata)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWiFiAccessPointNotFound
		}
		return nil, fmt.Errorf("failed to scan wifi access point: %w", err)
	}

	ap.SSIDID = ssidID.String
	ap.SSIDName = ssidName.String
	ap.APName = apName.String
	ap.Vendor = vendor.String
	ap.Metadata = metadata.String
	ap.NoiseDBm = int(noiseDBm.Int64)
	ap.IsAuthorized = isAuthorized == 1
	if t, parseErr := time.Parse(time.RFC3339, firstSeen); parseErr == nil {
		ap.FirstSeen = t
	}
	if t, parseErr := time.Parse(time.RFC3339, lastSeen); parseErr == nil {
		ap.LastSeen = t
	}
	return &ap, nil
}

func (r *DiscoveryRepository) scanWiFiAccessPointFromRows(rows *sql.Rows) (*WiFiAccessPointDB, error) {
	var ap WiFiAccessPointDB
	var firstSeen, lastSeen string
	var isAuthorized int
	var ssidID, ssidName, apName, vendor, metadata sql.NullString
	var noiseDBm sql.NullInt64

	err := rows.Scan(&ap.ID, &ap.BSSID, &ssidID, &ssidName, &apName, &vendor, &ap.Channel,
		&ap.ChannelWidth, &ap.FrequencyMHz, &ap.Band, &ap.SignalDBm, &noiseDBm,
		&ap.ClientCount, &isAuthorized, &firstSeen, &lastSeen, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to scan wifi access point: %w", err)
	}

	ap.SSIDID = ssidID.String
	ap.SSIDName = ssidName.String
	ap.APName = apName.String
	ap.Vendor = vendor.String
	ap.Metadata = metadata.String
	ap.NoiseDBm = int(noiseDBm.Int64)
	ap.IsAuthorized = isAuthorized == 1
	if t, parseErr := time.Parse(time.RFC3339, firstSeen); parseErr == nil {
		ap.FirstSeen = t
	}
	if t, parseErr := time.Parse(time.RFC3339, lastSeen); parseErr == nil {
		ap.LastSeen = t
	}
	return &ap, nil
}

// Network Problem operations
