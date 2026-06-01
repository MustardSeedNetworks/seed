package database

// repository_alert_suppressions persists the per-(rule, entity) "do
// not re-fire until" marks the alert pipelines used to keep in
// memory. Restart-safe so a Seed restart mid-incident doesn't
// re-fire alerts that were already emitted (#1380).
//
// fingerprint is the same sha256-derived key the pipelines compute
// when they decide whether to emit; passing it through unchanged
// means existing in-memory call sites only need to swap the map
// for this repo without recomputing.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AlertSuppressionsRepository is the read/write surface for the
// alert_suppressions table.
type AlertSuppressionsRepository struct {
	db *DB
}

// IsSuppressed returns true when fingerprint has a non-expired row.
// Expired rows are treated as if absent and left for PurgeExpired
// to clean up.
func (r *AlertSuppressionsRepository) IsSuppressed(
	ctx context.Context, fingerprint string, now time.Time,
) (bool, error) {
	row := r.db.QueryRow(ctx, `
		SELECT suppress_until FROM alert_suppressions
		WHERE fingerprint = ?
	`, fingerprint)
	var until string
	if err := row.Scan(&until); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("alert_suppressions IsSuppressed: %w", err)
	}
	parsed, perr := time.Parse(time.RFC3339Nano, until)
	if perr != nil {
		return false, fmt.Errorf("alert_suppressions parse suppress_until %q: %w", until, perr)
	}
	return now.Before(parsed), nil
}

// Mark sets / extends a suppression. Repeating Mark on the same
// fingerprint overwrites suppress_until — useful when the same
// rule fires again before its prior window expires (extends the
// window rather than stacking).
func (r *AlertSuppressionsRepository) Mark(
	ctx context.Context, fingerprint, ruleID, entityKey string,
	suppressUntil time.Time,
) error {
	if fingerprint == "" {
		return errors.New("alert_suppressions: fingerprint required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.Exec(ctx, `
		INSERT INTO alert_suppressions
		  (fingerprint, rule_id, entity_key, suppress_until, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(fingerprint) DO UPDATE SET
			suppress_until = excluded.suppress_until
	`,
		fingerprint, ruleID, entityKey,
		suppressUntil.UTC().Format(time.RFC3339Nano), now,
	)
	if err != nil {
		return fmt.Errorf("alert_suppressions Mark: %w", err)
	}
	return nil
}

// PurgeExpired deletes rows whose suppress_until is in the past.
// Returns the number of rows removed.
func (r *AlertSuppressionsRepository) PurgeExpired(
	ctx context.Context, now time.Time,
) (int64, error) {
	res, err := r.db.Exec(ctx, `
		DELETE FROM alert_suppressions WHERE suppress_until < ?
	`, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("alert_suppressions PurgeExpired: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
