package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrProbeNotFound is returned when a probe is not found.
var ErrProbeNotFound = errors.New("probe not found")

// ProbeRepository provides CRUD for probes and append/query for
// probe_results. Future Stage A2 rollups (probe_rollups_hourly,
// probe_rollups_daily) will land as additional methods on this
// repository, following the same raw-plus-rollups multi-table pattern.
type ProbeRepository struct {
	db *DB
}

// ---------------------------------------------------------------
// probes — CRUD
// ---------------------------------------------------------------

// CreateProbe inserts a new probe. Generates an id when Probe.ID is
// empty.
func (r *ProbeRepository) CreateProbe(ctx context.Context, p *Probe) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.ClientID == "" {
		p.ClientID = DefaultClientID
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO probes (id, client_id, kind, display_name, target, params_json,
			interval_seconds, enabled, warning_json, critical_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		p.ID, p.ClientID, p.Kind, p.DisplayName, p.Target,
		toNullString(p.ParamsJSON),
		p.IntervalSeconds, boolToInt(p.Enabled),
		toNullString(p.WarningJSON), toNullString(p.CriticalJSON),
		p.CreatedAt.Format(time.RFC3339Nano), p.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("create probe: %w", err)
	}
	return nil
}

// GetProbe returns a probe by id.
func (r *ProbeRepository) GetProbe(ctx context.Context, id string) (*Probe, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, client_id, kind, display_name, target, params_json,
			interval_seconds, enabled, warning_json, critical_json, created_at, updated_at
		FROM probes WHERE id = ?
	`, id)
	return scanProbe(row.Scan)
}

// UpdateProbe replaces all mutable fields. UpdatedAt is set to now.
func (r *ProbeRepository) UpdateProbe(ctx context.Context, p *Probe) error {
	p.UpdatedAt = time.Now().UTC()
	res, err := r.db.Exec(ctx, `
		UPDATE probes
		SET kind = ?, display_name = ?, target = ?, params_json = ?,
			interval_seconds = ?, enabled = ?, warning_json = ?, critical_json = ?,
			updated_at = ?
		WHERE id = ?
	`,
		p.Kind, p.DisplayName, p.Target,
		toNullString(p.ParamsJSON),
		p.IntervalSeconds, boolToInt(p.Enabled),
		toNullString(p.WarningJSON), toNullString(p.CriticalJSON),
		p.UpdatedAt.Format(time.RFC3339Nano), p.ID,
	)
	if err != nil {
		return fmt.Errorf("update probe: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrProbeNotFound
	}
	return nil
}

// DeleteProbe removes a probe; ON DELETE CASCADE on probe_results
// removes its history.
func (r *ProbeRepository) DeleteProbe(ctx context.Context, id string) error {
	res, err := r.db.Exec(ctx, `DELETE FROM probes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete probe: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrProbeNotFound
	}
	return nil
}

// ListProbes returns probes for a client, optionally filtered by
// kind. Empty clientID returns all clients' probes.
func (r *ProbeRepository) ListProbes(ctx context.Context, clientID, kind string) ([]*Probe, error) {
	query := `
		SELECT id, client_id, kind, display_name, target, params_json,
			interval_seconds, enabled, warning_json, critical_json, created_at, updated_at
		FROM probes
		WHERE 1=1
	`
	var args []any
	if clientID != "" {
		query += " AND client_id = ?"
		args = append(args, clientID)
	}
	if kind != "" {
		query += " AND kind = ?"
		args = append(args, kind)
	}
	query += " ORDER BY display_name ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list probes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*Probe
	for rows.Next() {
		p, scanErr := scanProbe(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, p)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("list probes iter: %w", rowsErr)
	}
	return out, nil
}

// CountProbes returns the number of probes for a client. Optionally
// filters by kind. Empty clientID counts across all clients.
func (r *ProbeRepository) CountProbes(ctx context.Context, clientID, kind string) (int, error) {
	query := `SELECT COUNT(*) FROM probes WHERE 1=1`
	var args []any
	if clientID != "" {
		query += " AND client_id = ?"
		args = append(args, clientID)
	}
	if kind != "" {
		query += " AND kind = ?"
		args = append(args, kind)
	}

	var n int
	err := r.db.QueryRow(ctx, query, args...).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count probes: %w", err)
	}
	return n, nil
}

// ---------------------------------------------------------------
// probe_results — write + query
// ---------------------------------------------------------------

// RecordResult appends one probe_results row. The id is auto-assigned
// by the database; the rest of the row is taken from ProbeResult.
func (r *ProbeRepository) RecordResult(ctx context.Context, pr *ProbeResult) error {
	if pr.ClientID == "" {
		pr.ClientID = DefaultClientID
	}
	if pr.Timestamp.IsZero() {
		pr.Timestamp = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO probe_results (probe_id, client_id, kind, timestamp,
			success, latency_ms, error, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		pr.ProbeID, pr.ClientID, pr.Kind,
		pr.Timestamp.Format(time.RFC3339Nano),
		boolToInt(pr.Success), pr.LatencyMs,
		toNullString(pr.Error), toNullString(pr.MetadataJSON),
	)
	if err != nil {
		return fmt.Errorf("record probe result: %w", err)
	}
	return nil
}

// QueryResults returns probe_results filtered by ProbeQueryOptions.
// Defaults: Limit=100, Offset=0, no time bounds.
func (r *ProbeRepository) QueryResults(ctx context.Context, opts ProbeQueryOptions) ([]*ProbeResult, error) {
	query := `
		SELECT id, probe_id, client_id, kind, timestamp,
			success, latency_ms, error, metadata_json
		FROM probe_results
		WHERE 1=1
	`
	var args []any
	if opts.ClientID != "" {
		query += " AND client_id = ?"
		args = append(args, opts.ClientID)
	}
	if opts.ProbeID != "" {
		query += " AND probe_id = ?"
		args = append(args, opts.ProbeID)
	}
	if opts.Kind != "" {
		query += " AND kind = ?"
		args = append(args, opts.Kind)
	}
	if !opts.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, opts.StartTime.Format(time.RFC3339Nano))
	}
	if !opts.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, opts.EndTime.Format(time.RFC3339Nano))
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultPaginationLimit
	}
	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, opts.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query probe results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*ProbeResult
	for rows.Next() {
		pr, scanErr := scanProbeResult(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, pr)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("query probe results iter: %w", rowsErr)
	}
	return out, nil
}

// DeleteResultsOlderThan removes probe_results with timestamp <
// cutoff. Returns the number of rows deleted. Used by the retention
// engine.
func (r *ProbeRepository) DeleteResultsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(ctx,
		`DELETE FROM probe_results WHERE timestamp < ?`,
		cutoff.Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("delete old probe results: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ---------------------------------------------------------------
// scanners + helpers
// ---------------------------------------------------------------

// scanProbe reads a Probe via the scan function signature shared by
// [sql.Row.Scan] and [sql.Rows.Scan].
func scanProbe(scan func(...any) error) (*Probe, error) {
	var (
		p          Probe
		params     sql.NullString
		warning    sql.NullString
		critical   sql.NullString
		enabledInt int
		createdAt  string
		updatedAt  string
	)
	err := scan(
		&p.ID, &p.ClientID, &p.Kind, &p.DisplayName, &p.Target,
		&params,
		&p.IntervalSeconds, &enabledInt,
		&warning, &critical,
		&createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProbeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan probe: %w", err)
	}
	if params.Valid {
		p.ParamsJSON = params.String
	}
	if warning.Valid {
		p.WarningJSON = warning.String
	}
	if critical.Valid {
		p.CriticalJSON = critical.String
	}
	p.Enabled = enabledInt != 0
	if t, parseErr := time.Parse(time.RFC3339Nano, createdAt); parseErr == nil {
		p.CreatedAt = t
	}
	if t, parseErr := time.Parse(time.RFC3339Nano, updatedAt); parseErr == nil {
		p.UpdatedAt = t
	}
	return &p, nil
}

// scanProbeResult reads a ProbeResult via the shared scan signature.
func scanProbeResult(scan func(...any) error) (*ProbeResult, error) {
	var (
		pr         ProbeResult
		latencyMs  sql.NullFloat64
		errMsg     sql.NullString
		metadata   sql.NullString
		successInt int
		ts         string
	)
	err := scan(
		&pr.ID, &pr.ProbeID, &pr.ClientID, &pr.Kind, &ts,
		&successInt, &latencyMs, &errMsg, &metadata,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("scan probe result: %w", err)
	}
	pr.Success = successInt != 0
	if latencyMs.Valid {
		pr.LatencyMs = latencyMs.Float64
	}
	if errMsg.Valid {
		pr.Error = errMsg.String
	}
	if metadata.Valid {
		pr.MetadataJSON = metadata.String
	}
	if t, parseErr := time.Parse(time.RFC3339Nano, ts); parseErr == nil {
		pr.Timestamp = t
	}
	return &pr, nil
}

// boolToInt is defined in repository_profiles.go.
