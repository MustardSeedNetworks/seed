package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// TestDBJobIdempotency exercises the durable Idempotency-Key store through its
// interface: miss -> store -> hit (same body) -> conflict (changed body).
func TestDBJobIdempotency(t *testing.T) {
	t.Parallel()
	store := app.NewJobStore(newJobStoreTestDB(t))
	ctx := context.Background()

	// The key's FK requires the job row to exist first (the runner persists on
	// Submit before the handler records the key).
	if err := store.Save(ctx, jobs.Job{ID: "j1", Kind: "speedtest", State: jobs.StateQueued}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	var idem jobIdempotencyStore = newDBJobIdempotency(store, slog.New(slog.DiscardHandler))
	req := CreateJobRequest{Kind: "speedtest", Params: json.RawMessage(`{"server":"a"}`)}

	if res := idem.check(ctx, "key-1", req); res.kind != idemMiss {
		t.Fatalf("first check = %v, want idemMiss", res.kind)
	}
	idem.store(ctx, "key-1", req, "j1")

	if res := idem.check(ctx, "key-1", req); res.kind != idemHit || res.id != "j1" {
		t.Errorf("replay check = (%v,%q), want (idemHit,j1)", res.kind, res.id)
	}

	other := CreateJobRequest{Kind: "speedtest", Params: json.RawMessage(`{"server":"DIFFERENT"}`)}
	if res := idem.check(ctx, "key-1", other); res.kind != idemConflict {
		t.Errorf("changed-body check = %v, want idemConflict", res.kind)
	}
}

func newJobStoreTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(dbtest.Path(t))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
