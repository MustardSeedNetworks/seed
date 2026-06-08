// SPDX-License-Identifier: BUSL-1.1

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// OutboxRecord is one row of the transactional outbox (ADR-0017): a domain event
// awaiting delivery. ID is the monotonic autoincrement key the relay uses as the
// stable dedup token; Payload is the opaque serialized event. PublishedAt is the
// zero time while the row is pending and the relay's publish timestamp once
// drained.
type OutboxRecord struct {
	ID          int64
	Topic       string
	Payload     []byte
	CreatedAt   time.Time
	PublishedAt time.Time
}

// OutboxRepository owns the outbox table (ADR-0017). Enqueue is called inside a
// caller-supplied transaction so the event row commits atomically with the
// domain change; the relay (internal/platform/outbox) drains via FetchUnpublished
// / MarkPublished and the maintenance loop prunes with DeletePublishedBefore.
type OutboxRepository struct {
	db *DB
}

// Outbox returns the transactional-outbox repository (ADR-0017). Producers that
// need durable, cross-restart delivery enqueue in-transaction; a relay
// republishes post-commit onto the in-process bus.
func (db *DB) Outbox() *OutboxRepository {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.outbox == nil {
		db.outbox = &OutboxRepository{db: db}
	}
	return db.outbox
}

// Enqueue writes one pending event on the caller's transaction. It deliberately
// runs on tx (not db) so the row is committed or rolled back together with the
// domain write that produced the event — the outbox's entire correctness
// guarantee. The row is left unpublished (published_at NULL) for the relay.
func (r *OutboxRepository) Enqueue(ctx context.Context, tx *sql.Tx, topic string, payload []byte) error {
	if tx == nil {
		return errors.New("outbox: Enqueue requires a transaction")
	}
	if topic == "" {
		return errors.New("outbox: empty topic")
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outbox (topic, payload, created_at)
		VALUES (?, ?, ?)
	`, topic, payload, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("enqueue outbox row: %w", err)
	}
	return nil
}

// FetchUnpublished returns up to limit pending rows in insert order (the partial
// index keeps this scan bounded to the backlog). The relay publishes them, then
// MarkPublished the ones it delivered.
func (r *OutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]OutboxRecord, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, topic, payload, created_at
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unpublished outbox rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []OutboxRecord
	for rows.Next() {
		var (
			rec          OutboxRecord
			createdAtStr string
		)
		if scanErr := rows.Scan(&rec.ID, &rec.Topic, &rec.Payload, &createdAtStr); scanErr != nil {
			return nil, fmt.Errorf("scan outbox row: %w", scanErr)
		}
		rec.CreatedAt = parseTime(createdAtStr)
		out = append(out, rec)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", rowsErr)
	}
	return out, nil
}

// MarkPublished stamps published_at on the given rows. It is called after the
// relay has published them to the bus; a crash between publish and mark leaves
// the rows pending, so they replay (at-least-once). An empty slice is a no-op.
func (r *OutboxRepository) MarkPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx,
			`UPDATE outbox SET published_at = ? WHERE id = ? AND published_at IS NULL`)
		if err != nil {
			return fmt.Errorf("prepare mark published: %w", err)
		}
		defer func() { _ = stmt.Close() }()
		for _, id := range ids {
			if _, execErr := stmt.ExecContext(ctx, now, id); execErr != nil {
				return fmt.Errorf("mark outbox row %d published: %w", id, execErr)
			}
		}
		return nil
	})
}

// DeletePublishedBefore removes published rows whose published_at is before
// cutoff and returns the count removed. Pending rows (NULL published_at) are
// never touched — the durable analogue of jobs retention.
func (r *OutboxRepository) DeletePublishedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.Exec(ctx, `
		DELETE FROM outbox
		WHERE published_at IS NOT NULL AND published_at < ?
	`, cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("delete published outbox rows: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
