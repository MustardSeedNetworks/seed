package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/platform/events"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

func newJobStoreTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "jobs-store.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestDBJobStoreSaveLoadRoundTrip proves a terminal job round-trips through the
// adapter: the result is stored as JSON and returned as [json.RawMessage].
func TestDBJobStoreSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()
	store := newDBJobStore(newJobStoreTestDB(t))
	ctx := context.Background()

	want := jobs.Job{
		ID: "j1", Kind: "speedtest", State: jobs.StateSucceeded, Progress: 1,
		Result: map[string]any{"downloadMbps": 100},
	}
	if err := store.Save(ctx, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, ok, err := store.Load(ctx, "j1")
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	if got.Kind != "speedtest" || got.State != jobs.StateSucceeded || got.Progress != 1 {
		t.Errorf("got %+v, want speedtest/succeeded/1", got)
	}
	raw, isRaw := got.Result.(json.RawMessage)
	if !isRaw {
		t.Fatalf("Result type = %T, want json.RawMessage", got.Result)
	}
	var decoded map[string]any
	if uErr := json.Unmarshal(raw, &decoded); uErr != nil {
		t.Fatalf("unmarshal result: %v", uErr)
	}
	if decoded["downloadMbps"] != float64(100) {
		t.Errorf("result = %v, want downloadMbps=100", decoded)
	}
}

// TestDBJobStoreLoadMiss: an unknown id is a clean miss, not an error.
func TestDBJobStoreLoadMiss(t *testing.T) {
	t.Parallel()
	store := newDBJobStore(newJobStoreTestDB(t))

	_, ok, err := store.Load(context.Background(), "absent")
	if err != nil {
		t.Fatalf("load miss returned error: %v", err)
	}
	if ok {
		t.Error("load miss returned ok=true")
	}
}

// TestDBJobStoreMarkInterrupted: only non-terminal jobs are reconciled to failed.
func TestDBJobStoreMarkInterrupted(t *testing.T) {
	t.Parallel()
	store := newDBJobStore(newJobStoreTestDB(t))
	ctx := context.Background()

	for _, j := range []jobs.Job{
		{ID: "run", Kind: "k", State: jobs.StateRunning, Progress: 0.5},
		{ID: "queue", Kind: "k", State: jobs.StateQueued},
		{ID: "done", Kind: "k", State: jobs.StateSucceeded, Progress: 1},
	} {
		if err := store.Save(ctx, j); err != nil {
			t.Fatalf("save %s: %v", j.ID, err)
		}
	}

	n, err := store.MarkInterrupted(ctx)
	if err != nil {
		t.Fatalf("mark interrupted: %v", err)
	}
	if n != 2 {
		t.Errorf("interrupted = %d, want 2", n)
	}
	for _, id := range []string{"run", "queue"} {
		j, _, _ := store.Load(ctx, id)
		if j.State != jobs.StateFailed {
			t.Errorf("%s state = %q, want failed", id, j.State)
		}
	}
	done, _, _ := store.Load(ctx, "done")
	if done.State != jobs.StateSucceeded {
		t.Errorf("terminal job mutated to %q", done.State)
	}
}

// TestRunnerRecoversAcrossRestart wires the adapter into a runner and proves the
// full restart story: a job persisted as running by a prior process is
// reconciled to failed on Recover and is then retrievable via the runner's
// store fallback even though it was never in this runner's memory.
func TestRunnerRecoversAcrossRestart(t *testing.T) {
	t.Parallel()
	db := newJobStoreTestDB(t)
	store := newDBJobStore(db)
	ctx := context.Background()

	// Prior process left a job mid-flight.
	if err := store.Save(ctx, jobs.Job{ID: "stale", Kind: "engine-scan", State: jobs.StateRunning}); err != nil {
		t.Fatalf("seed running job: %v", err)
	}

	// New process: fresh runner over the same durable store.
	bus := events.New(slog.New(slog.DiscardHandler))
	runner := jobs.New(bus, slog.New(slog.DiscardHandler), jobs.Config{Store: store})
	t.Cleanup(func() {
		closeCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = runner.Close(closeCtx)
		_ = bus.Close(closeCtx)
	})

	n, err := runner.Recover(ctx)
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if n != 1 {
		t.Errorf("recovered = %d, want 1", n)
	}

	// The job is not in this runner's memory; Get must fall back to the store
	// and report the reconciled terminal state.
	got, ok := runner.Get("stale")
	if !ok {
		t.Fatal("Get(stale) !ok; expected store fallback")
	}
	if got.State != jobs.StateFailed {
		t.Errorf("recovered job state = %q, want failed", got.State)
	}
}
