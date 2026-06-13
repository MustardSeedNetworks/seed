package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// SNMPObservationsRepository owns CRUD over snmp_observations.
type SNMPObservationsRepository struct {
	db *DB
}

// Insert records one observation. ObservedAt and IngestedAt are
// stamped if zero.
func (r *SNMPObservationsRepository) Insert(ctx context.Context, obs *observation.SNMPObservation) error {
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

// List returns observations matching opts ordered by ObservedAt
// descending (newest first). Limit defaults to 100; clamped to 1000.
func (r *SNMPObservationsRepository) List(
	ctx context.Context,
	opts observation.ListOptions,
) ([]*observation.SNMPObservation, error) {
	const defaultLimit, maxLimit = 100, 1000

	query, args := newListQueryBuilder(`
		SELECT id, client_id, target_id, kind, observed_at, payload_json, ingested_at
		FROM snmp_observations
		WHERE 1=1
	`).
		Where("AND client_id = ?", opts.ClientID).
		Where("AND target_id = ?", opts.TargetID).
		Where("AND kind = ?", opts.Kind).
		WhereTime("AND observed_at >= ?", opts.Since).
		WhereTime("AND observed_at <= ?", opts.Until).
		OrderLimit("ORDER BY observed_at DESC", opts.Limit, defaultLimit, maxLimit).
		Build()

	return queryRows(ctx, r.db, query, args, scanSNMPObservation, "list snmp_observations")
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
func scanSNMPObservation(scan func(...any) error) (*observation.SNMPObservation, error) {
	var (
		obs           observation.SNMPObservation
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
