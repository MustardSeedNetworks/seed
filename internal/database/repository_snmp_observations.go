package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SNMPObservation is one row of snmp_observations. PayloadJSON
// carries the typed collector Observation struct serialized as JSON
// — callers decode it back into the kind-specific shape they own.
type SNMPObservation struct {
	ID          int64
	ClientID    string
	TargetID    string
	Kind        string
	ObservedAt  time.Time
	PayloadJSON string
	IngestedAt  time.Time
}

// SNMPObservationsRepository owns CRUD over snmp_observations.
type SNMPObservationsRepository struct {
	db *DB
}

// Insert records one observation. ObservedAt and IngestedAt are
// stamped if zero.
func (r *SNMPObservationsRepository) Insert(ctx context.Context, obs *SNMPObservation) error {
	if obs.Kind == "" {
		return errors.New("snmp_observations: Kind required")
	}
	if obs.TargetID == "" {
		return errors.New("snmp_observations: TargetID required")
	}
	now := time.Now().UTC()
	if obs.ObservedAt.IsZero() {
		obs.ObservedAt = now
	}
	if obs.IngestedAt.IsZero() {
		obs.IngestedAt = now
	}
	if obs.ClientID == "" {
		obs.ClientID = "default"
	}

	res, err := r.db.Exec(ctx, `
		INSERT INTO snmp_observations (client_id, target_id, kind, observed_at, payload_json, ingested_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		obs.ClientID,
		obs.TargetID,
		obs.Kind,
		obs.ObservedAt.UTC().Format(time.RFC3339Nano),
		obs.PayloadJSON,
		obs.IngestedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert snmp_observation: %w", err)
	}
	if id, idErr := res.LastInsertId(); idErr == nil {
		obs.ID = id
	}
	return nil
}

// ListOptions narrows the query — empty values disable that filter.
type ListOptions struct {
	ClientID string
	TargetID string
	Kind     string
	Since    time.Time
	Until    time.Time
	Limit    int
}

// List returns observations matching opts ordered by ObservedAt
// descending (newest first). Limit is clamped to 1000 if unset.
func (r *SNMPObservationsRepository) List(ctx context.Context, opts ListOptions) ([]*SNMPObservation, error) {
	const defaultLimit, maxLimit = 100, 1000

	query := `
		SELECT id, client_id, target_id, kind, observed_at, payload_json, ingested_at
		FROM snmp_observations
		WHERE 1=1
	`
	args := []any{}
	if opts.ClientID != "" {
		query += " AND client_id = ?"
		args = append(args, opts.ClientID)
	}
	if opts.TargetID != "" {
		query += " AND target_id = ?"
		args = append(args, opts.TargetID)
	}
	if opts.Kind != "" {
		query += " AND kind = ?"
		args = append(args, opts.Kind)
	}
	if !opts.Since.IsZero() {
		query += " AND observed_at >= ?"
		args = append(args, opts.Since.UTC().Format(time.RFC3339Nano))
	}
	if !opts.Until.IsZero() {
		query += " AND observed_at <= ?"
		args = append(args, opts.Until.UTC().Format(time.RFC3339Nano))
	}
	query += " ORDER BY observed_at DESC LIMIT ?"
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	args = append(args, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list snmp_observations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*SNMPObservation
	for rows.Next() {
		obs, scanErr := scanSNMPObservation(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, obs)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("list snmp_observations iter: %w", rowsErr)
	}
	return out, nil
}

// DeleteOlderThan removes observations whose observed_at is before
// cutoff. Returns the number of rows deleted.
func (r *SNMPObservationsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(ctx,
		`DELETE FROM snmp_observations WHERE observed_at < ?`,
		cutoff.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("purge snmp_observations: %w", err)
	}
	if res == nil {
		return 0, errors.New("purge snmp_observations: nil sql.Result")
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// scanSNMPObservation reads a row via a [sql.Row.Scan] or
// [sql.Rows.Scan] signature.
func scanSNMPObservation(scan func(...any) error) (*SNMPObservation, error) {
	var (
		obs           SNMPObservation
		observedAtStr string
		ingestedAtStr string
	)
	err := scan(
		&obs.ID, &obs.ClientID, &obs.TargetID, &obs.Kind,
		&observedAtStr, &obs.PayloadJSON, &ingestedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("scan snmp_observation: %w", err)
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, observedAtStr); perr == nil {
		obs.ObservedAt = parsed
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, ingestedAtStr); perr == nil {
		obs.IngestedAt = parsed
	}
	return &obs, nil
}
