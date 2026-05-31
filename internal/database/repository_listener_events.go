package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ListenerEvent is one row of listener_events.
type ListenerEvent struct {
	ID          int64
	ClientID    string
	Kind        string
	SourceAddr  string
	TargetKind  string
	TargetID    string
	Severity    string
	ObservedAt  time.Time
	PayloadJSON string
	IngestedAt  time.Time
}

// ListenerEventsRepository owns CRUD over listener_events.
type ListenerEventsRepository struct {
	db *DB
}

// Insert records one listener event. ObservedAt and IngestedAt are
// stamped with the current time if zero.
func (r *ListenerEventsRepository) Insert(ctx context.Context, evt *ListenerEvent) error {
	if evt.Kind == "" {
		return errors.New("listener_events: Kind required")
	}
	if evt.SourceAddr == "" {
		return errors.New("listener_events: SourceAddr required")
	}
	now := time.Now().UTC()
	if evt.ObservedAt.IsZero() {
		evt.ObservedAt = now
	}
	if evt.IngestedAt.IsZero() {
		evt.IngestedAt = now
	}
	if evt.ClientID == "" {
		evt.ClientID = "default"
	}

	res, err := r.db.Exec(ctx, `
		INSERT INTO listener_events
		  (client_id, kind, source_addr, target_kind, target_id,
		   severity, observed_at, payload_json, ingested_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		evt.ClientID,
		evt.Kind,
		evt.SourceAddr,
		toNullString(evt.TargetKind),
		toNullString(evt.TargetID),
		toNullString(evt.Severity),
		evt.ObservedAt.UTC().Format(time.RFC3339Nano),
		evt.PayloadJSON,
		evt.IngestedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert listener_event: %w", err)
	}
	if id, idErr := res.LastInsertId(); idErr == nil {
		evt.ID = id
	}
	return nil
}

// ListenerEventListOptions narrows the query — empty values disable
// that filter.
type ListenerEventListOptions struct {
	ClientID   string
	Kind       string
	SourceAddr string
	Since      time.Time
	Until      time.Time
	Limit      int
}

// List returns events matching opts ordered by ObservedAt desc.
// Limit defaults to 100; clamped to 1000.
func (r *ListenerEventsRepository) List(
	ctx context.Context,
	opts ListenerEventListOptions,
) ([]*ListenerEvent, error) {
	const defaultLimit, maxLimit = 100, 1000

	query, args := newListQueryBuilder(`
		SELECT id, client_id, kind, source_addr, target_kind, target_id,
		       severity, observed_at, payload_json, ingested_at
		FROM listener_events
		WHERE 1=1
	`).
		Where("AND client_id = ?", opts.ClientID).
		Where("AND kind = ?", opts.Kind).
		Where("AND source_addr = ?", opts.SourceAddr).
		WhereTime("AND observed_at >= ?", opts.Since).
		WhereTime("AND observed_at <= ?", opts.Until).
		OrderLimit("ORDER BY observed_at DESC", opts.Limit, defaultLimit, maxLimit).
		Build()

	return queryRows(ctx, r.db, query, args, scanListenerEvent, "list listener_events")
}

// DeleteOlderThan removes events whose observed_at is before cutoff.
// Returns the number of rows deleted.
func (r *ListenerEventsRepository) DeleteOlderThan(
	ctx context.Context,
	cutoff time.Time,
) (int64, error) {
	res, err := r.db.Exec(ctx,
		`DELETE FROM listener_events WHERE observed_at < ?`,
		cutoff.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("purge listener_events: %w", err)
	}
	if res == nil {
		return 0, errors.New("purge listener_events: nil sql.Result")
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// scanListenerEvent reads a row via either [sql.Row.Scan] or
// [sql.Rows.Scan] signatures.
func scanListenerEvent(scan func(...any) error) (*ListenerEvent, error) {
	var (
		evt           ListenerEvent
		targetKind    sql.NullString
		targetID      sql.NullString
		severity      sql.NullString
		observedAtStr string
		ingestedAtStr string
	)
	err := scan(
		&evt.ID, &evt.ClientID, &evt.Kind, &evt.SourceAddr,
		&targetKind, &targetID, &severity,
		&observedAtStr, &evt.PayloadJSON, &ingestedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("scan listener_event: %w", err)
	}
	if targetKind.Valid {
		evt.TargetKind = targetKind.String
	}
	if targetID.Valid {
		evt.TargetID = targetID.String
	}
	if severity.Valid {
		evt.Severity = severity.String
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, observedAtStr); perr == nil {
		evt.ObservedAt = parsed
	}
	if parsed, perr := time.Parse(time.RFC3339Nano, ingestedAtStr); perr == nil {
		evt.IngestedAt = parsed
	}
	return &evt, nil
}
