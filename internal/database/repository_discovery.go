package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Error definitions for discovery repository.
var (
	ErrWiFiNetworkNotFound     = errors.New("wifi network not found")
	ErrWiFiAccessPointNotFound = errors.New("wifi access point not found")
	ErrNetworkProblemNotFound  = errors.New("network problem not found")
	ErrOUIVendorNotFound       = errors.New("oui vendor not found")
)

// DiscoveryRepository provides operations for unified discovery data.
type DiscoveryRepository struct {
	db *DB
}

// WiFi Network operations

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
		query += " LIMIT ?"
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
		query += " LIMIT ?"
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

// NetworkProblemDB represents a network problem in the database.
type NetworkProblemDB struct {
	ID              string     `json:"id"`
	Category        string     `json:"category"`
	Type            string     `json:"type"`
	Severity        string     `json:"severity"`
	Status          string     `json:"status"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DeviceID        string     `json:"deviceId,omitempty"`
	DeviceMAC       string     `json:"deviceMac,omitempty"`
	InterfaceName   string     `json:"interfaceName,omitempty"`
	IPAddress       string     `json:"ipAddress,omitempty"`
	AffectedMACs    string     `json:"affectedMacs,omitempty"`
	SSID            string     `json:"ssid,omitempty"`
	BSSID           string     `json:"bssid,omitempty"`
	Channel         int        `json:"channel,omitempty"`
	CurrentValue    float64    `json:"currentValue,omitempty"`
	ThresholdValue  float64    `json:"thresholdValue,omitempty"`
	Unit            string     `json:"unit,omitempty"`
	FirstSeen       time.Time  `json:"firstSeen"`
	LastSeen        time.Time  `json:"lastSeen"`
	ResolvedAt      *time.Time `json:"resolvedAt,omitempty"`
	OccurrenceCount int        `json:"occurrenceCount"`
	Metadata        string     `json:"metadata,omitempty"`
}

// CreateNetworkProblem creates a new network problem record.
func (r *DiscoveryRepository) CreateNetworkProblem(ctx context.Context, problem *NetworkProblemDB) error {
	if problem.ID == "" {
		problem.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if problem.FirstSeen.IsZero() {
		problem.FirstSeen = now
	}
	problem.LastSeen = now
	if problem.OccurrenceCount == 0 {
		problem.OccurrenceCount = 1
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO network_problems
		(id, category, type, severity, status, title, description, device_id, device_mac,
		 interface_name, ip_address, affected_macs, ssid, bssid, channel, current_value,
		 threshold_value, unit, first_seen, last_seen, resolved_at, occurrence_count, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, problem.ID, problem.Category, problem.Type, problem.Severity, problem.Status,
		problem.Title, problem.Description, nullString(problem.DeviceID), nullString(problem.DeviceMAC),
		nullString(problem.InterfaceName), nullString(problem.IPAddress), nullString(problem.AffectedMACs),
		nullString(problem.SSID), nullString(problem.BSSID), problem.Channel, problem.CurrentValue,
		problem.ThresholdValue, nullString(problem.Unit), problem.FirstSeen.Format(time.RFC3339),
		problem.LastSeen.Format(time.RFC3339), nullTimeStr(problem.ResolvedAt), problem.OccurrenceCount,
		nullString(problem.Metadata))
	if err != nil {
		return fmt.Errorf("failed to create network problem: %w", err)
	}
	return nil
}

// GetNetworkProblem retrieves a network problem by ID.
func (r *DiscoveryRepository) GetNetworkProblem(ctx context.Context, id string) (*NetworkProblemDB, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, category, type, severity, status, title, description, device_id, device_mac,
		       interface_name, ip_address, affected_macs, ssid, bssid, channel, current_value,
		       threshold_value, unit, first_seen, last_seen, resolved_at, occurrence_count, metadata_json
		FROM network_problems WHERE id = ?
	`, id)
	return r.scanNetworkProblem(row)
}

// ListNetworkProblems retrieves network problems with filters.
func (r *DiscoveryRepository) ListNetworkProblems(
	ctx context.Context,
	opts ProblemListOptions,
) ([]*NetworkProblemDB, error) {
	query := `
		SELECT id, category, type, severity, status, title, description, device_id, device_mac,
		       interface_name, ip_address, affected_macs, ssid, bssid, channel, current_value,
		       threshold_value, unit, first_seen, last_seen, resolved_at, occurrence_count, metadata_json
		FROM network_problems WHERE 1=1
	`
	var args []any

	if opts.Category != "" {
		query += " AND category = ?"
		args = append(args, opts.Category)
	}
	if opts.Severity != "" {
		query += " AND severity = ?"
		args = append(args, opts.Severity)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.ActiveOnly {
		query += " AND status = 'active'"
	}
	if opts.DeviceID != "" {
		query += " AND device_id = ?"
		args = append(args, opts.DeviceID)
	}

	// Order by severity (critical first), then by last_seen
	query += " ORDER BY CASE severity WHEN 'critical' THEN 1 WHEN 'warning' THEN 2 ELSE 3 END, last_seen DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list network problems: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var problems []*NetworkProblemDB
	for rows.Next() {
		p, scanErr := r.scanNetworkProblemFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		problems = append(problems, p)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating network problems: %w", rowsErr)
	}
	return problems, nil
}

// ProblemListOptions specifies criteria for listing network problems.
type ProblemListOptions struct {
	Category   string
	Severity   string
	Status     string
	ActiveOnly bool
	DeviceID   string
	Limit      int
}

// UpdateNetworkProblem updates a network problem.
func (r *DiscoveryRepository) UpdateNetworkProblem(ctx context.Context, problem *NetworkProblemDB) error {
	problem.LastSeen = time.Now().UTC()

	result, err := r.db.Exec(ctx, `
		UPDATE network_problems
		SET severity = ?, status = ?, title = ?, description = ?, current_value = ?,
		    threshold_value = ?, last_seen = ?, resolved_at = ?, occurrence_count = ?, metadata_json = ?
		WHERE id = ?
	`, problem.Severity, problem.Status, problem.Title, problem.Description, problem.CurrentValue,
		problem.ThresholdValue, problem.LastSeen.Format(time.RFC3339), nullTimeStr(problem.ResolvedAt),
		problem.OccurrenceCount, nullString(problem.Metadata), problem.ID)
	if err != nil {
		return fmt.Errorf("failed to update network problem: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNetworkProblemNotFound
	}
	return nil
}

// ResolveProblem marks a network problem as resolved.
func (r *DiscoveryRepository) ResolveProblem(ctx context.Context, id string) error {
	now := time.Now().UTC()
	result, err := r.db.Exec(ctx, `
		UPDATE network_problems SET status = 'resolved', resolved_at = ? WHERE id = ?
	`, now.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to resolve problem: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNetworkProblemNotFound
	}
	return nil
}

// GetProblemSummary returns a summary of network problems.
func (r *DiscoveryRepository) GetProblemSummary(ctx context.Context) (*ProblemSummaryDB, error) {
	summary := &ProblemSummaryDB{
		BySeverity: make(map[string]int),
		ByCategory: make(map[string]int),
	}

	// Count active problems
	row := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM network_problems WHERE status = 'active'`)
	if err := row.Scan(&summary.TotalActive); err != nil {
		return nil, fmt.Errorf("failed to count active problems: %w", err)
	}

	// Count by severity
	rows, err := r.db.Query(ctx, `
		SELECT severity, COUNT(*) FROM network_problems WHERE status = 'active' GROUP BY severity
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to count by severity: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var severity string
		var count int
		if scanErr := rows.Scan(&severity, &count); scanErr != nil {
			return nil, scanErr
		}
		summary.BySeverity[severity] = count
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating severity counts: %w", err)
	}

	// Count by category
	rowsCat, err := r.db.Query(ctx, `
		SELECT category, COUNT(*) FROM network_problems WHERE status = 'active' GROUP BY category
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to count by category: %w", err)
	}
	defer func() { _ = rowsCat.Close() }()
	for rowsCat.Next() {
		var category string
		var count int
		if scanErr := rowsCat.Scan(&category, &count); scanErr != nil {
			return nil, scanErr
		}
		summary.ByCategory[category] = count
	}
	if err = rowsCat.Err(); err != nil {
		return nil, fmt.Errorf("iterating category counts: %w", err)
	}

	// Recent count (last hour)
	oneHourAgo := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	row = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM network_problems WHERE first_seen >= ?`, oneHourAgo)
	if err = row.Scan(&summary.RecentCount); err != nil {
		return nil, fmt.Errorf("failed to count recent problems: %w", err)
	}

	// Resolved today
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
	row = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM network_problems WHERE resolved_at >= ?`, startOfDay)
	if err = row.Scan(&summary.ResolvedToday); err != nil {
		return nil, fmt.Errorf("failed to count resolved today: %w", err)
	}

	return summary, nil
}

// ProblemSummaryDB represents problem statistics.
type ProblemSummaryDB struct {
	TotalActive   int            `json:"totalActive"`
	BySeverity    map[string]int `json:"bySeverity"`
	ByCategory    map[string]int `json:"byCategory"`
	RecentCount   int            `json:"recentCount"`
	ResolvedToday int            `json:"resolvedToday"`
}

func (r *DiscoveryRepository) scanNetworkProblem(row *sql.Row) (*NetworkProblemDB, error) {
	var p NetworkProblemDB
	var firstSeen, lastSeen string
	var resolvedAt sql.NullString
	var deviceID, deviceMAC, interfaceName, ipAddress, affectedMACs sql.NullString
	var ssid, bssid, unit, metadata sql.NullString

	err := row.Scan(&p.ID, &p.Category, &p.Type, &p.Severity, &p.Status, &p.Title, &p.Description,
		&deviceID, &deviceMAC, &interfaceName, &ipAddress, &affectedMACs, &ssid, &bssid,
		&p.Channel, &p.CurrentValue, &p.ThresholdValue, &unit, &firstSeen, &lastSeen, &resolvedAt,
		&p.OccurrenceCount, &metadata)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNetworkProblemNotFound
		}
		return nil, fmt.Errorf("failed to scan network problem: %w", err)
	}

	p.DeviceID = deviceID.String
	p.DeviceMAC = deviceMAC.String
	p.InterfaceName = interfaceName.String
	p.IPAddress = ipAddress.String
	p.AffectedMACs = affectedMACs.String
	p.SSID = ssid.String
	p.BSSID = bssid.String
	p.Unit = unit.String
	p.Metadata = metadata.String

	if t, parseErr := time.Parse(time.RFC3339, firstSeen); parseErr == nil {
		p.FirstSeen = t
	}
	if t, parseErr := time.Parse(time.RFC3339, lastSeen); parseErr == nil {
		p.LastSeen = t
	}
	if resolvedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, resolvedAt.String); parseErr == nil {
			p.ResolvedAt = &t
		}
	}
	return &p, nil
}

func (r *DiscoveryRepository) scanNetworkProblemFromRows(rows *sql.Rows) (*NetworkProblemDB, error) {
	var p NetworkProblemDB
	var firstSeen, lastSeen string
	var resolvedAt sql.NullString
	var deviceID, deviceMAC, interfaceName, ipAddress, affectedMACs sql.NullString
	var ssid, bssid, unit, metadata sql.NullString

	err := rows.Scan(&p.ID, &p.Category, &p.Type, &p.Severity, &p.Status, &p.Title, &p.Description,
		&deviceID, &deviceMAC, &interfaceName, &ipAddress, &affectedMACs, &ssid, &bssid,
		&p.Channel, &p.CurrentValue, &p.ThresholdValue, &unit, &firstSeen, &lastSeen, &resolvedAt,
		&p.OccurrenceCount, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to scan network problem: %w", err)
	}

	p.DeviceID = deviceID.String
	p.DeviceMAC = deviceMAC.String
	p.InterfaceName = interfaceName.String
	p.IPAddress = ipAddress.String
	p.AffectedMACs = affectedMACs.String
	p.SSID = ssid.String
	p.BSSID = bssid.String
	p.Unit = unit.String
	p.Metadata = metadata.String

	if t, parseErr := time.Parse(time.RFC3339, firstSeen); parseErr == nil {
		p.FirstSeen = t
	}
	if t, parseErr := time.Parse(time.RFC3339, lastSeen); parseErr == nil {
		p.LastSeen = t
	}
	if resolvedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, resolvedAt.String); parseErr == nil {
			p.ResolvedAt = &t
		}
	}
	return &p, nil
}

// OUI Vendor operations

// OUIVendorDB represents an OUI vendor record in the database.
type OUIVendorDB struct {
	OUIPrefix    string `json:"ouiPrefix"` // First 3 bytes of MAC (e.g., "00:50:56")
	VendorName   string `json:"vendorName"`
	VendorAlias  string `json:"vendorAlias,omitempty"`
	IsPrivate    bool   `json:"isPrivate"`
	AddressBlock string `json:"addressBlock,omitempty"` // MA-L, MA-M, MA-S
}

// CreateOUIVendor creates a new OUI vendor record.
func (r *DiscoveryRepository) CreateOUIVendor(ctx context.Context, vendor *OUIVendorDB) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO oui_vendors (oui_prefix, vendor_name, vendor_alias, is_private, address_block)
		VALUES (?, ?, ?, ?, ?)
	`, vendor.OUIPrefix, vendor.VendorName, vendor.VendorAlias,
		boolToInt(vendor.IsPrivate), vendor.AddressBlock)
	if err != nil {
		return fmt.Errorf("failed to create oui vendor: %w", err)
	}
	return nil
}

// GetOUIVendor retrieves a vendor by OUI prefix.
func (r *DiscoveryRepository) GetOUIVendor(ctx context.Context, ouiPrefix string) (*OUIVendorDB, error) {
	// Normalize the prefix (uppercase, colon-separated)
	normalized := normalizeOUI(ouiPrefix)

	row := r.db.QueryRow(ctx, `
		SELECT oui_prefix, vendor_name, vendor_alias, is_private, address_block
		FROM oui_vendors WHERE oui_prefix = ?
	`, normalized)

	var v OUIVendorDB
	var alias, addressBlock sql.NullString
	var isPrivate int

	err := row.Scan(&v.OUIPrefix, &v.VendorName, &alias, &isPrivate, &addressBlock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOUIVendorNotFound
		}
		return nil, fmt.Errorf("failed to scan oui vendor: %w", err)
	}

	v.VendorAlias = alias.String
	v.AddressBlock = addressBlock.String
	v.IsPrivate = isPrivate == 1
	return &v, nil
}

// LookupVendorByMAC looks up the vendor for a MAC address.
func (r *DiscoveryRepository) LookupVendorByMAC(ctx context.Context, mac string) (string, error) {
	// Extract OUI prefix from MAC address
	ouiPrefix := extractOUIPrefix(mac)
	if ouiPrefix == "" {
		return "", nil
	}

	vendor, err := r.GetOUIVendor(ctx, ouiPrefix)
	if err != nil {
		if errors.Is(err, ErrOUIVendorNotFound) {
			return "", nil
		}
		return "", err
	}
	return vendor.VendorName, nil
}

// BulkUpsertOUIVendors inserts or updates multiple OUI vendors efficiently.
func (r *DiscoveryRepository) BulkUpsertOUIVendors(ctx context.Context, vendors []OUIVendorDB) error {
	if len(vendors) == 0 {
		return nil
	}

	// Use INSERT OR REPLACE for SQLite
	query := `INSERT OR REPLACE INTO oui_vendors (oui_prefix, vendor_name, vendor_alias, is_private, address_block) VALUES `
	var args []any
	var placeholders []string

	for _, v := range vendors {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?)")
		args = append(args, v.OUIPrefix, v.VendorName, v.VendorAlias, boolToInt(v.IsPrivate), v.AddressBlock)
	}

	query += strings.Join(placeholders, ", ")

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert oui vendors: %w", err)
	}
	return nil
}

// Channel Utilization operations

// ChannelUtilizationDB represents channel utilization data in the database.
type ChannelUtilizationDB struct {
	ID                 string    `json:"id"`
	Channel            int       `json:"channel"`
	Band               string    `json:"band"`
	FrequencyMHz       int       `json:"frequencyMhz"`
	UtilizationPercent float64   `json:"utilizationPercent"`
	NonWiFiPercent     float64   `json:"nonWifiPercent"`
	RetryPercent       float64   `json:"retryPercent"`
	APCount            int       `json:"apCount"`
	ClientCount        int       `json:"clientCount"`
	RecordedAt         time.Time `json:"recordedAt"`
}

// CreateChannelUtilization creates a new channel utilization record.
func (r *DiscoveryRepository) CreateChannelUtilization(ctx context.Context, util *ChannelUtilizationDB) error {
	if util.ID == "" {
		util.ID = uuid.New().String()
	}
	if util.RecordedAt.IsZero() {
		util.RecordedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO channel_utilization
		(id, channel, band, frequency_mhz, utilization_percent, non_wifi_percent, retry_percent,
		 ap_count, client_count, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, util.ID, util.Channel, util.Band, util.FrequencyMHz, util.UtilizationPercent,
		util.NonWiFiPercent, util.RetryPercent, util.APCount, util.ClientCount,
		util.RecordedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to create channel utilization: %w", err)
	}
	return nil
}

// GetChannelUtilization retrieves the latest utilization for a channel.
func (r *DiscoveryRepository) GetChannelUtilization(
	ctx context.Context,
	channel int,
	band string,
) (*ChannelUtilizationDB, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, channel, band, frequency_mhz, utilization_percent, non_wifi_percent, retry_percent,
		       ap_count, client_count, recorded_at
		FROM channel_utilization
		WHERE channel = ? AND band = ?
		ORDER BY recorded_at DESC
		LIMIT 1
	`, channel, band)

	var u ChannelUtilizationDB
	var recordedAt string

	err := row.Scan(&u.ID, &u.Channel, &u.Band, &u.FrequencyMHz, &u.UtilizationPercent,
		&u.NonWiFiPercent, &u.RetryPercent, &u.APCount, &u.ClientCount, &recordedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan channel utilization: %w", err)
	}

	if t, parseErr := time.Parse(time.RFC3339, recordedAt); parseErr == nil {
		u.RecordedAt = t
	}
	return &u, nil
}

// ListChannelUtilization retrieves channel utilization for all channels.
func (r *DiscoveryRepository) ListChannelUtilization(
	ctx context.Context,
	band string,
) ([]*ChannelUtilizationDB, error) {
	query := `
		SELECT id, channel, band, frequency_mhz, utilization_percent, non_wifi_percent, retry_percent,
		       ap_count, client_count, recorded_at
		FROM channel_utilization
		WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY channel, band ORDER BY recorded_at DESC) as rn
				FROM channel_utilization
			) WHERE rn = 1
		)
	`
	var args []any

	if band != "" {
		query += " AND band = ?"
		args = append(args, band)
	}

	query += " ORDER BY channel"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list channel utilization: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var utils []*ChannelUtilizationDB
	for rows.Next() {
		var u ChannelUtilizationDB
		var recordedAt string

		if scanErr := rows.Scan(&u.ID, &u.Channel, &u.Band, &u.FrequencyMHz, &u.UtilizationPercent,
			&u.NonWiFiPercent, &u.RetryPercent, &u.APCount, &u.ClientCount, &recordedAt); scanErr != nil {
			return nil, fmt.Errorf("failed to scan channel utilization: %w", scanErr)
		}

		if t, parseErr := time.Parse(time.RFC3339, recordedAt); parseErr == nil {
			u.RecordedAt = t
		}
		utils = append(utils, &u)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating channel utilization: %w", rowsErr)
	}
	return utils, nil
}

// Discovery Statistics

// WiFiDiscoveryStats returns WiFi discovery statistics.
func (r *DiscoveryRepository) GetWiFiStats(ctx context.Context) (*WiFiStatsDB, error) {
	stats := &WiFiStatsDB{
		SecurityBreakdown: make(map[string]int),
		BandBreakdown:     make(map[string]int),
	}

	// Total networks
	row := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wifi_networks`)
	if err := row.Scan(&stats.TotalNetworks); err != nil {
		return nil, fmt.Errorf("failed to count networks: %w", err)
	}

	// Hidden networks
	row = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wifi_networks WHERE is_hidden = 1`)
	if err := row.Scan(&stats.HiddenNetworks); err != nil {
		return nil, fmt.Errorf("failed to count hidden networks: %w", err)
	}

	// Total APs
	row = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wifi_access_points`)
	if err := row.Scan(&stats.TotalAPs); err != nil {
		return nil, fmt.Errorf("failed to count APs: %w", err)
	}

	// Authorized/Unauthorized APs
	row = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wifi_access_points WHERE is_authorized = 1`)
	if err := row.Scan(&stats.AuthorizedAPs); err != nil {
		return nil, fmt.Errorf("failed to count authorized APs: %w", err)
	}
	stats.UnauthorizedAPs = stats.TotalAPs - stats.AuthorizedAPs

	// Security breakdown
	rows, err := r.db.Query(ctx, `SELECT security_type, COUNT(*) FROM wifi_networks GROUP BY security_type`)
	if err != nil {
		return nil, fmt.Errorf("failed to get security breakdown: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var secType string
		var count int
		if scanErr := rows.Scan(&secType, &count); scanErr != nil {
			return nil, scanErr
		}
		stats.SecurityBreakdown[secType] = count
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating security breakdown: %w", err)
	}

	// Band breakdown
	rowsBand, err := r.db.Query(ctx, `SELECT band, COUNT(*) FROM wifi_access_points GROUP BY band`)
	if err != nil {
		return nil, fmt.Errorf("failed to get band breakdown: %w", err)
	}
	defer func() { _ = rowsBand.Close() }()
	for rowsBand.Next() {
		var band string
		var count int
		if scanErr := rowsBand.Scan(&band, &count); scanErr != nil {
			return nil, scanErr
		}
		stats.BandBreakdown[band] = count
	}
	if err = rowsBand.Err(); err != nil {
		return nil, fmt.Errorf("iterating band breakdown: %w", err)
	}

	return stats, nil
}

// WiFiStatsDB represents WiFi discovery statistics.
type WiFiStatsDB struct {
	TotalNetworks     int            `json:"totalNetworks"`
	HiddenNetworks    int            `json:"hiddenNetworks"`
	TotalAPs          int            `json:"totalAPs"`
	AuthorizedAPs     int            `json:"authorizedAPs"`
	UnauthorizedAPs   int            `json:"unauthorizedAPs"`
	SecurityBreakdown map[string]int `json:"securityBreakdown"`
	BandBreakdown     map[string]int `json:"bandBreakdown"`
}

// Helper functions

func normalizeOUI(oui string) string {
	// Remove common separators and convert to uppercase
	oui = strings.ToUpper(oui)
	oui = strings.ReplaceAll(oui, "-", "")
	oui = strings.ReplaceAll(oui, ":", "")
	oui = strings.ReplaceAll(oui, ".", "")

	// Take first 6 hex chars and format with colons
	if len(oui) >= 6 {
		return oui[0:2] + ":" + oui[2:4] + ":" + oui[4:6]
	}
	return oui
}

func extractOUIPrefix(mac string) string {
	// Normalize MAC address
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, ".", "")

	// Extract first 6 hex chars (OUI)
	if len(mac) >= 6 {
		return mac[0:2] + ":" + mac[2:4] + ":" + mac[4:6]
	}
	return ""
}

func nullTimeStr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
