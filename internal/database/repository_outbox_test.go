// SPDX-License-Identifier: BUSL-1.1

package database_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

func setupOutboxTest(t *testing.T) (*database.DB, context.Context) {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "seed-outbox-*.db")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	t.Cleanup(func() { _ = os.Remove(tmpPath) })

	db, openErr := database.Open(tmpPath)
	if openErr != nil {
		t.Fatalf("open db: %v", openErr)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, context.Background()
}

// TestOutboxEnqueueCommitsWithDomainWrite is the load-bearing atomicity test: an
// outbox row enqueued inside WithTx is durable only if the surrounding
// transaction — domain write included — commits.
func TestOutboxEnqueueCommitsWithDomainWrite(t *testing.T) {
	t.Parallel()
	db, ctx := setupOutboxTest(t)

	// A stand-in domain table so the test exercises a real co-committed write.
	if _, err := db.Exec(ctx, `CREATE TABLE scratch_domain (id TEXT NOT NULL PRIMARY KEY)`); err != nil {
		t.Fatalf("create scratch table: %v", err)
	}

	err := db.WithTx(ctx, func(tx *sql.Tx) error {
		if _, execErr := tx.ExecContext(ctx, `INSERT INTO scratch_domain (id) VALUES (?)`, "d1"); execErr != nil {
			return execErr
		}
		return db.Outbox().Enqueue(ctx, tx, "device.discovered", []byte(`{"id":"d1"}`))
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}

	recs, err := db.Outbox().FetchUnpublished(ctx, 10)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d unpublished rows, want 1", len(recs))
	}
	if recs[0].Topic != "device.discovered" || string(recs[0].Payload) != `{"id":"d1"}` {
		t.Errorf("row = topic %q payload %q, want device.discovered / {\"id\":\"d1\"}",
			recs[0].Topic, string(recs[0].Payload))
	}
	if recs[0].ID == 0 {
		t.Error("row ID should be a nonzero autoincrement key")
	}
}

// TestOutboxEnqueueRollbackDropsRow proves the row participates in the caller's
// transaction: if the domain write fails and the tx rolls back, no event is left
// behind — a rolled-back operation never fires an "it happened" fact.
func TestOutboxEnqueueRollbackDropsRow(t *testing.T) {
	t.Parallel()
	db, ctx := setupOutboxTest(t)

	sentinel := errors.New("domain write failed")
	err := db.WithTx(ctx, func(tx *sql.Tx) error {
		if enqErr := db.Outbox().Enqueue(ctx, tx, "device.discovered", []byte(`{"id":"x"}`)); enqErr != nil {
			return enqErr
		}
		return sentinel // forces rollback after the enqueue
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("WithTx error = %v, want sentinel", err)
	}

	recs, err := db.Outbox().FetchUnpublished(ctx, 10)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("rollback left %d outbox rows, want 0", len(recs))
	}
}

func TestOutboxFetchUnpublishedOrdersByInsertionAndLimits(t *testing.T) {
	t.Parallel()
	db, ctx := setupOutboxTest(t)

	for _, topic := range []string{"a", "b", "c"} {
		if err := db.WithTx(ctx, func(tx *sql.Tx) error {
			return db.Outbox().Enqueue(ctx, tx, topic, []byte(topic))
		}); err != nil {
			t.Fatalf("enqueue %s: %v", topic, err)
		}
	}

	recs, err := db.Outbox().FetchUnpublished(ctx, 2)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("limit not honored: got %d, want 2", len(recs))
	}
	if recs[0].Topic != "a" || recs[1].Topic != "b" {
		t.Errorf("order = %q,%q, want a,b (insert order)", recs[0].Topic, recs[1].Topic)
	}
}

func TestOutboxMarkPublishedExcludesFromFetch(t *testing.T) {
	t.Parallel()
	db, ctx := setupOutboxTest(t)

	for _, topic := range []string{"a", "b"} {
		if err := db.WithTx(ctx, func(tx *sql.Tx) error {
			return db.Outbox().Enqueue(ctx, tx, topic, []byte(topic))
		}); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	recs, err := db.Outbox().FetchUnpublished(ctx, 10)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if markErr := db.Outbox().MarkPublished(ctx, []int64{recs[0].ID}); markErr != nil {
		t.Fatalf("mark published: %v", markErr)
	}

	left, err := db.Outbox().FetchUnpublished(ctx, 10)
	if err != nil {
		t.Fatalf("fetch after mark: %v", err)
	}
	if len(left) != 1 || left[0].Topic != "b" {
		t.Fatalf("after marking 'a' published, unpublished = %v, want only b", left)
	}
}

func TestOutboxDeletePublishedBeforeRetainsUnpublished(t *testing.T) {
	t.Parallel()
	db, ctx := setupOutboxTest(t)

	for _, topic := range []string{"old", "keep"} {
		if err := db.WithTx(ctx, func(tx *sql.Tx) error {
			return db.Outbox().Enqueue(ctx, tx, topic, []byte(topic))
		}); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	recs, err := db.Outbox().FetchUnpublished(ctx, 10)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	// Publish only "old".
	if markErr := db.Outbox().MarkPublished(ctx, []int64{recs[0].ID}); markErr != nil {
		t.Fatalf("mark: %v", markErr)
	}

	// Retention sweep with a cutoff in the future removes the published row only.
	n, err := db.Outbox().DeletePublishedBefore(ctx, time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if n != 1 {
		t.Fatalf("deleted %d rows, want 1 (only the published one)", n)
	}
	left, err := db.Outbox().FetchUnpublished(ctx, 10)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(left) != 1 || left[0].Topic != "keep" {
		t.Fatalf("retention removed an unpublished row: left = %v", left)
	}
}
