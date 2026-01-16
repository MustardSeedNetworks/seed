package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrAlertNotFound is returned when an alert is not found.
var ErrAlertNotFound = errors.New("alert not found")

// AlertRepository provides operations for alerts.
type AlertRepository struct {
	db *DB
}

// Create creates a new alert.
func (r *AlertRepository) Create(ctx context.Context, alert *Alert) error {
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}

	result, err := r.db.Exec(ctx, `
		INSERT INTO alerts
		(type, severity, title, message, source, device_id, acknowledged, acknowledged_by,
		 acknowledged_at, resolved, resolved_at, created_at, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, alert.Type, alert.Severity, alert.Title, alert.Message, alert.Source,
		alert.DeviceID, boolToInt(alert.Acknowledged), alert.AcknowledgedBy,
		timeToString(alert.AcknowledgedAt), boolToInt(alert.Resolved),
		timeToString(alert.ResolvedAt), alert.CreatedAt.Format(time.RFC3339), alert.Metadata)
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		alert.ID = id
	}

	return nil
}

// Get retrieves an alert by ID.
func (r *AlertRepository) Get(ctx context.Context, id int64) (*Alert, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, type, severity, title, message, source, device_id, acknowledged,
		       acknowledged_by, acknowledged_at, resolved, resolved_at, created_at, metadata_json
		FROM alerts WHERE id = ?
	`, id)

	return r.scanAlert(row)
}

// List retrieves alerts matching the criteria.
//

func (r *AlertRepository) List(ctx context.Context, opts AlertListOptions) ([]*Alert, error) {
	query := `
		SELECT id, type, severity, title, message, source, device_id, acknowledged,
		       acknowledged_by, acknowledged_at, resolved, resolved_at, created_at, metadata_json
		FROM alerts
		WHERE 1=1
	`
	var args []any

	if opts.Type != "" {
		query += sqlAndType
		args = append(args, opts.Type)
	}

	if opts.Severity != "" {
		query += sqlAndSeverity
		args = append(args, opts.Severity)
	}

	if opts.DeviceID != "" {
		query += sqlAndDeviceID
		args = append(args, opts.DeviceID)
	}

	if opts.UnacknowledgedOnly {
		query += " AND acknowledged = 0"
	}

	if opts.UnresolvedOnly {
		query += " AND resolved = 0"
	}

	if !opts.Since.IsZero() {
		query += " AND created_at >= ?"
		args = append(args, opts.Since.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += sqlLimit
		args = append(args, opts.Limit)
	}

	if opts.Offset > 0 {
		query += sqlOffset
		args = append(args, opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var alerts []*Alert
	for rows.Next() {
		a, scanErr := r.scanAlertFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		alerts = append(alerts, a)
	}

	return alerts, rows.Err()
}

// AlertListOptions specifies criteria for listing alerts.
type AlertListOptions struct {
	Type               string
	Severity           string
	DeviceID           string
	UnacknowledgedOnly bool
	UnresolvedOnly     bool
	Since              time.Time
	Limit              int
	Offset             int
}

// Acknowledge marks an alert as acknowledged.
func (r *AlertRepository) Acknowledge(ctx context.Context, id int64, by string) error {
	now := time.Now().UTC()

	result, err := r.db.Exec(ctx, `
		UPDATE alerts SET acknowledged = 1, acknowledged_by = ?, acknowledged_at = ?
		WHERE id = ?
	`, by, now.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAlertNotFound
	}

	return nil
}

// AcknowledgeAll marks all matching alerts as acknowledged.
//

func (r *AlertRepository) AcknowledgeAll(
	ctx context.Context,
	opts AlertListOptions,
	by string,
) (int64, error) {
	now := time.Now().UTC()

	query := `
		UPDATE alerts SET acknowledged = 1, acknowledged_by = ?, acknowledged_at = ?
		WHERE acknowledged = 0
	`
	args := []any{by, now.Format(time.RFC3339)}

	if opts.Type != "" {
		query += sqlAndType
		args = append(args, opts.Type)
	}

	if opts.Severity != "" {
		query += sqlAndSeverity
		args = append(args, opts.Severity)
	}

	if opts.DeviceID != "" {
		query += sqlAndDeviceID
		args = append(args, opts.DeviceID)
	}

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to acknowledge alerts: %w", err)
	}

	return result.RowsAffected()
}

// Resolve marks an alert as resolved.
func (r *AlertRepository) Resolve(ctx context.Context, id int64) error {
	now := time.Now().UTC()

	result, err := r.db.Exec(ctx, `
		UPDATE alerts SET resolved = 1, resolved_at = ? WHERE id = ?
	`, now.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAlertNotFound
	}

	return nil
}

// Delete removes an alert by ID.
func (r *AlertRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM alerts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAlertNotFound
	}

	return nil
}

// DeleteOlderThan removes alerts older than the given time.
func (r *AlertRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		DELETE FROM alerts WHERE created_at < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("failed to delete old alerts: %w", err)
	}

	return result.RowsAffected()
}

// Count returns the number of alerts matching the criteria.
//

func (r *AlertRepository) Count(ctx context.Context, opts AlertListOptions) (int64, error) {
	query := "SELECT COUNT(*) FROM alerts WHERE 1=1"
	var args []any

	if opts.Type != "" {
		query += sqlAndType
		args = append(args, opts.Type)
	}

	if opts.Severity != "" {
		query += sqlAndSeverity
		args = append(args, opts.Severity)
	}

	if opts.UnacknowledgedOnly {
		query += " AND acknowledged = 0"
	}

	if opts.UnresolvedOnly {
		query += " AND resolved = 0"
	}

	var count int64
	row := r.db.QueryRow(ctx, query, args...)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count alerts: %w", err)
	}
	return count, nil
}

// GetUnacknowledgedCount returns the number of unacknowledged alerts.
func (r *AlertRepository) GetUnacknowledgedCount(ctx context.Context) (int64, error) {
	return r.Count(ctx, AlertListOptions{UnacknowledgedOnly: true})
}

// GetCriticalCount returns the number of unresolved critical alerts.
func (r *AlertRepository) GetCriticalCount(ctx context.Context) (int64, error) {
	return r.Count(ctx, AlertListOptions{
		Severity:       AlertSeverityCritical,
		UnresolvedOnly: true,
	})
}

// scanAlert scans an alert from a row.
func (r *AlertRepository) scanAlert(row *sql.Row) (*Alert, error) {
	var a Alert
	var createdAt string
	var acked, resolved int
	var source, deviceID, ackedBy, ackedAt, resolvedAt, metadata sql.NullString

	err := row.Scan(&a.ID, &a.Type, &a.Severity, &a.Title, &a.Message, &source, &deviceID,
		&acked, &ackedBy, &ackedAt, &resolved, &resolvedAt, &createdAt, &metadata)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAlertNotFound
		}
		return nil, fmt.Errorf("failed to scan alert: %w", err)
	}

	a.Source = source.String
	a.Metadata = metadata.String
	a.Acknowledged = acked == 1
	a.Resolved = resolved == 1
	if t, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil {
		a.CreatedAt = t
	}

	if deviceID.Valid {
		a.DeviceID = &deviceID.String
	}
	if ackedBy.Valid {
		a.AcknowledgedBy = &ackedBy.String
	}
	if ackedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, ackedAt.String); parseErr == nil {
			a.AcknowledgedAt = &t
		}
	}
	if resolvedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, resolvedAt.String); parseErr == nil {
			a.ResolvedAt = &t
		}
	}

	return &a, nil
}

// scanAlertFromRows scans an alert from rows.
func (r *AlertRepository) scanAlertFromRows(rows *sql.Rows) (*Alert, error) {
	var a Alert
	var createdAt string
	var acked, resolved int
	var source, deviceID, ackedBy, ackedAt, resolvedAt, metadata sql.NullString

	err := rows.Scan(&a.ID, &a.Type, &a.Severity, &a.Title, &a.Message, &source, &deviceID,
		&acked, &ackedBy, &ackedAt, &resolved, &resolvedAt, &createdAt, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to scan alert: %w", err)
	}

	a.Source = source.String
	a.Metadata = metadata.String
	a.Acknowledged = acked == 1
	a.Resolved = resolved == 1
	if t, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil {
		a.CreatedAt = t
	}

	if deviceID.Valid {
		a.DeviceID = &deviceID.String
	}
	if ackedBy.Valid {
		a.AcknowledgedBy = &ackedBy.String
	}
	if ackedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, ackedAt.String); parseErr == nil {
			a.AcknowledgedAt = &t
		}
	}
	if resolvedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, resolvedAt.String); parseErr == nil {
			a.ResolvedAt = &t
		}
	}

	return &a, nil
}

// timeToString converts a time pointer to a SQL-compatible string.
func timeToString(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.Format(time.RFC3339), Valid: true}
}
