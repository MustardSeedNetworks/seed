package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// deliveriesColumn reads an alert's per-channel delivery outcomes as one JSON
// array, NULL when no channel was ever offered the alert, so List stays one
// query rather than one per alert. Ordered by channel so the wire order is
// stable.
const deliveriesColumn = `(
		SELECT json_group_array(json_object(
			'channel', d.channel, 'status', d.status,
			'attemptedAt', d.attempted_at, 'error', NULLIF(d.error, '')
		) ORDER BY d.channel)
		FROM alert_deliveries d WHERE d.alert_id = alerts.id
		HAVING count(*) > 0)`

// ErrAlertNotFound is returned when an alert is not found.
var ErrAlertNotFound = errors.New("alert not found")

// AlertRepository provides operations for alerts.
type AlertRepository struct {
	db *DB
}

// Create creates a new alert together with the delivery state it was stamped
// with, in one transaction, so the inbox never shows an alert whose pending
// delivery is missing.
func (r *AlertRepository) Create(ctx context.Context, alert *alerts.Alert) error {
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO alerts
			(type, severity, title, message, source, device_id, acknowledged, acknowledged_by,
			 acknowledged_at, resolved, resolved_at, created_at, metadata_json, rule, root_cause_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, alert.Type, alert.Severity, alert.Title, alert.Message, alert.Source,
			alert.DeviceID, boolToInt(alert.Acknowledged), alert.AcknowledgedBy,
			timeToString(alert.AcknowledgedAt), boolToInt(alert.Resolved),
			timeToString(alert.ResolvedAt), alert.CreatedAt.Format(time.RFC3339), alert.Metadata,
			alert.Rule, alert.RootCauseID)
		if err != nil {
			return fmt.Errorf("failed to create alert: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to read new alert id: %w", err)
		}
		for _, d := range alert.Deliveries {
			if _, err = tx.ExecContext(ctx, `
				INSERT INTO alert_deliveries (alert_id, channel, status, attempted_at, error)
				VALUES (?, ?, ?, ?, ?)
			`, id, d.Channel, d.Status, timeToString(d.AttemptedAt), d.Error); err != nil {
				return fmt.Errorf("failed to record alert %s delivery: %w", d.Channel, err)
			}
		}
		alert.ID = id
		return nil
	})
}

// Get retrieves an alert by ID.
func (r *AlertRepository) Get(ctx context.Context, id int64) (*alerts.Alert, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, type, severity, title, message, source, device_id, acknowledged,
		       acknowledged_by, acknowledged_at, resolved, resolved_at, created_at, metadata_json,
		       rule, root_cause_id, escalation_stage, escalated_at, `+deliveriesColumn+`
		FROM alerts WHERE id = ?
	`, id)

	return r.scanAlert(row)
}

// List retrieves alerts matching opts (type, severity, device, unacknowledged/
// unresolved-only, since, limit/offset), newest first.
func (r *AlertRepository) List(ctx context.Context, opts alerts.ListOptions) ([]*alerts.Alert, error) {
	query := `
		SELECT id, type, severity, title, message, source, device_id, acknowledged,
		       acknowledged_by, acknowledged_at, resolved, resolved_at, created_at, metadata_json,
		       rule, root_cause_id, escalation_stage, escalated_at, ` + deliveriesColumn + `
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

	out, err := r.queryAlerts(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	return out, nil
}

// ListEffects returns the alerts whose root cause is one of causeIDs, oldest
// first — the rest of each cluster a page of the inbox shows the cause of.
func (r *AlertRepository) ListEffects(ctx context.Context, causeIDs []int64) ([]*alerts.Alert, error) {
	if len(causeIDs) == 0 {
		return nil, nil
	}
	args := make([]any, len(causeIDs))
	for i, id := range causeIDs {
		args[i] = id
	}
	out, err := r.queryAlerts(ctx, `
		SELECT id, type, severity, title, message, source, device_id, acknowledged,
		       acknowledged_by, acknowledged_at, resolved, resolved_at, created_at, metadata_json,
		       rule, root_cause_id, escalation_stage, escalated_at, NULL
		FROM alerts
		WHERE root_cause_id IN (?`+strings.Repeat(", ?", len(causeIDs)-1)+`)
		ORDER BY created_at, id
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert effects: %w", err)
	}
	return out, nil
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

// AcknowledgeAll marks every currently-unacknowledged alert matching opts as
// acknowledged by the given user, in one statement, and returns the count of
// rows updated.
func (r *AlertRepository) AcknowledgeAll(
	ctx context.Context,
	opts alerts.ListOptions,
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

// RecordDelivery writes the outcome of one channel's delivery attempt onto the
// alert it was for (#368, #2997). The alert is where an operator already
// looks, so it is where a receiver that stopped accepting becomes visible.
//
// A missing alert is not an error here. Retention prunes by age, and a
// delivery that exhausted its bounded retries can finish after the alert it
// was for was pruned; failing that write would log an error about an alert
// nobody can see any more. The INSERT ... SELECT writes nothing in that case.
func (r *AlertRepository) RecordDelivery(
	ctx context.Context,
	id int64,
	channel alerts.Channel,
	status string,
	attemptedAt time.Time,
	deliveryErr string,
) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO alert_deliveries (alert_id, channel, status, attempted_at, error)
		SELECT id, ?, ?, ?, ? FROM alerts WHERE id = ?
		ON CONFLICT (alert_id, channel) DO UPDATE SET
			status = excluded.status,
			attempted_at = excluded.attempted_at,
			error = excluded.error
	`, channel, status, attemptedAt.UTC().Format(time.RFC3339), deliveryErr, id)
	if err != nil {
		return fmt.Errorf("failed to record alert delivery: %w", err)
	}
	return nil
}

// ListEscalating returns the open alerts — neither acknowledged nor resolved —
// raised by any of rules, oldest first (P-B2). It is the escalator's read, so
// it skips the per-channel delivery history the inbox needs.
func (r *AlertRepository) ListEscalating(ctx context.Context, rules []string) ([]*alerts.Alert, error) {
	if len(rules) == 0 {
		return nil, nil
	}
	args := make([]any, len(rules))
	for i, rule := range rules {
		args[i] = rule
	}
	out, err := r.queryAlerts(ctx, `
		SELECT id, type, severity, title, message, source, device_id, acknowledged,
		       acknowledged_by, acknowledged_at, resolved, resolved_at, created_at, metadata_json,
		       rule, root_cause_id, escalation_stage, escalated_at, NULL
		FROM alerts
		WHERE acknowledged = 0 AND resolved = 0
		  AND rule IN (?`+strings.Repeat(", ?", len(rules)-1)+`)
		ORDER BY created_at, id
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list escalating alerts: %w", err)
	}
	return out, nil
}

// queryAlerts runs a query selecting the scanAlertFromRows columns and reads
// every row.
func (r *AlertRepository) queryAlerts(ctx context.Context, query string, args ...any) ([]*alerts.Alert, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*alerts.Alert
	for rows.Next() {
		a, scanErr := r.scanAlertFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AdvanceEscalation records that alert id was sent at stage at at (P-B2). It
// is a compare-and-set on the stage and time the caller read, and it requires
// the alert to be open, so an alert acknowledged or resolved after the read,
// or advanced by another pass, is left alone and false is returned.
func (r *AlertRepository) AdvanceEscalation(
	ctx context.Context,
	id int64,
	fromStage int,
	fromAt *time.Time,
	stage int,
	at time.Time,
) (bool, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE alerts SET escalation_stage = ?, escalated_at = ?
		WHERE id = ? AND acknowledged = 0 AND resolved = 0
		  AND escalation_stage = ? AND escalated_at IS ?
	`, stage, at.UTC().Format(time.RFC3339), id, fromStage, timeToString(fromAt))
	if err != nil {
		return false, fmt.Errorf("failed to advance alert escalation: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return n == 1, nil
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

// Count returns the number of alerts matching opts, using the same filters
// as List but without Limit/Offset applied.
func (r *AlertRepository) Count(ctx context.Context, opts alerts.ListOptions) (int64, error) {
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
	return r.Count(ctx, alerts.ListOptions{UnacknowledgedOnly: true})
}

// GetCriticalCount returns the number of unresolved critical alerts.
func (r *AlertRepository) GetCriticalCount(ctx context.Context) (int64, error) {
	return r.Count(ctx, alerts.ListOptions{
		Severity:       alerts.SeverityCritical,
		UnresolvedOnly: true,
	})
}

// scanAlert scans an alert from a single row, mapping [sql.ErrNoRows] to
// [ErrAlertNotFound].
func (r *AlertRepository) scanAlert(row *sql.Row) (*alerts.Alert, error) {
	a, err := scanAlertInto(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAlertNotFound
		}
		return nil, err
	}
	return a, nil
}

// scanAlertFromRows scans an alert from a result set.
func (r *AlertRepository) scanAlertFromRows(rows *sql.Rows) (*alerts.Alert, error) {
	return scanAlertInto(rows.Scan)
}

// scanAlertInto is the one place the alert column list is decoded. Both
// callers pass their own Scan method so a column added to the SELECTs is
// added to exactly one scanner; they were separate copies before, which is
// how a new column drifts into being read by one path and not the other.
func scanAlertInto(scan func(...any) error) (*alerts.Alert, error) {
	var a alerts.Alert
	var createdAt string
	var acked, resolved int
	var source, deviceID, ackedBy, ackedAt, resolvedAt, metadata, rule sql.NullString
	var deliveries sql.NullString
	var rootCauseID sql.NullInt64
	var escalatedAt sql.NullString

	if err := scan(&a.ID, &a.Type, &a.Severity, &a.Title, &a.Message, &source, &deviceID,
		&acked, &ackedBy, &ackedAt, &resolved, &resolvedAt, &createdAt, &metadata,
		&rule, &rootCauseID, &a.EscalationStage, &escalatedAt, &deliveries); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to scan alert: %w", err)
	}

	a.Source = source.String
	a.Metadata = metadata.String
	a.Rule = rule.String
	a.Acknowledged = acked == 1
	a.Resolved = resolved == 1
	if t, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil {
		a.CreatedAt = t
	}

	if rootCauseID.Valid {
		a.RootCauseID = &rootCauseID.Int64
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
	if escalatedAt.Valid {
		if t, parseErr := time.Parse(time.RFC3339, escalatedAt.String); parseErr == nil {
			a.EscalatedAt = &t
		}
	}
	if deliveries.Valid {
		if err := json.Unmarshal([]byte(deliveries.String), &a.Deliveries); err != nil {
			return nil, fmt.Errorf("failed to decode alert %d deliveries: %w", a.ID, err)
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
