package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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
		query += sqlLimit
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
	startOfDay := time.Now().UTC().Truncate(hoursPerDay * time.Hour).Format(time.RFC3339)
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
