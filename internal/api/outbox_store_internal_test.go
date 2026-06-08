package api

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/outbox"
)

func newOutboxStoreTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "outbox-store.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestOutboxRelayEndToEnd is the wiring proof for ADR-0017: an event enqueued in
// a domain transaction is drained by the relay through the real database adapter
// and delivered on its original topic, exactly once to an idempotent consumer.
func TestOutboxRelayEndToEnd(t *testing.T) {
	t.Parallel()
	db := newOutboxStoreTestDB(t)
	ctx := context.Background()

	// Producer: enqueue an event in the same transaction as a domain write.
	if err := db.WithTx(ctx, func(tx *sql.Tx) error {
		return db.Outbox().Enqueue(ctx, tx, "device.discovered", []byte(`{"id":"d1"}`))
	}); err != nil {
		t.Fatalf("enqueue in tx: %v", err)
	}

	bus := events.New(slog.New(slog.DiscardHandler))
	var got []string
	dedup := outbox.NewDeduper(64)
	bus.Subscribe("device.discovered", outbox.Dedupe(dedup, func(_ context.Context, ev events.Event) {
		if m, ok := ev.(outbox.Message); ok {
			got = append(got, string(m.Payload))
		}
	}))

	relay := outbox.NewRelay(newDBOutboxStore(db), bus, slog.New(slog.DiscardHandler))

	// First drain delivers it.
	if n, err := relay.Drain(ctx); err != nil || n != 1 {
		t.Fatalf("drain = (%d,%v), want (1,nil)", n, err)
	}
	// A second drain finds nothing (the row was marked published).
	if n, err := relay.Drain(ctx); err != nil || n != 0 {
		t.Fatalf("second drain = (%d,%v), want (0,nil)", n, err)
	}

	closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := bus.Close(closeCtx); err != nil {
		t.Fatalf("bus close: %v", err)
	}

	if len(got) != 1 || got[0] != `{"id":"d1"}` {
		t.Fatalf("consumer got %v, want one delivery of the payload", got)
	}

	// Retention respects age: a freshly delivered row (younger than the window)
	// is retained. (Deletion past the window is covered at the repository layer
	// by TestOutboxDeletePublishedBeforeRetainsUnpublished.)
	if n, err := relay.Cleanup(ctx, time.Hour); err != nil || n != 0 {
		t.Fatalf("cleanup of a fresh row = (%d,%v), want (0,nil)", n, err)
	}
}
