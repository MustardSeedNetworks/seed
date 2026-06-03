package api

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/platform/events"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

// TestSweepJobsDurable: only terminal jobs older than the cutoff are deleted
// from the durable store; recent terminal jobs survive.
func TestSweepJobsDurable(t *testing.T) {
	t.Parallel()
	repo := newJobStoreTestDB(t).Jobs()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.Save(ctx, &database.JobRecord{
		ID: "old", Kind: "k", State: "succeeded", Progress: 1,
		CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour),
		CompletedAt: now.Add(-2 * time.Hour),
	}); err != nil {
		t.Fatalf("save old: %v", err)
	}
	if err := repo.Save(ctx, &database.JobRecord{
		ID: "recent", Kind: "k", State: "failed", Error: "x",
		CreatedAt: now, UpdatedAt: now, CompletedAt: now,
	}); err != nil {
		t.Fatalf("save recent: %v", err)
	}

	sweepJobs(ctx, nil, repo, now.Add(-time.Hour), slog.New(slog.DiscardHandler))

	if _, err := repo.Get(ctx, "old"); !errors.Is(err, database.ErrJobNotFound) {
		t.Errorf("old terminal job not evicted (err=%v)", err)
	}
	if _, err := repo.Get(ctx, "recent"); err != nil {
		t.Errorf("recent terminal job wrongly evicted: %v", err)
	}
}

// TestSweepJobsInMemory: a terminal job past the (zero) retention window is
// evicted from the runner's in-memory map.
func TestSweepJobsInMemory(t *testing.T) {
	t.Parallel()
	bus := events.New(slog.New(slog.DiscardHandler))
	runner := jobs.New(bus, slog.New(slog.DiscardHandler), jobs.Config{Retention: 0})
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = runner.Close(ctx)
		_ = bus.Close(ctx)
	})
	if err := runner.Register("noop", func(context.Context, any, func(float64)) (any, error) {
		return "ok", nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	id, err := runner.Submit("noop", nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	// Wait for the job to reach a terminal state before sweeping.
	deadline := time.After(5 * time.Second)
	for {
		j, ok := runner.Get(id)
		if ok && (j.State == jobs.StateSucceeded || j.State == jobs.StateFailed) {
			break
		}
		select {
		case <-deadline:
			t.Fatal("job never reached terminal state")
		case <-time.After(2 * time.Millisecond):
		}
	}

	sweepJobs(context.Background(), runner, nil, time.Now().UTC(), slog.New(slog.DiscardHandler))

	if _, ok := runner.Get(id); ok {
		t.Error("terminal job not evicted from the in-memory runner")
	}
}

// TestSweepJobsNilSafe: nil runner and nil repo are tolerated (no-op, no panic).
func TestSweepJobsNilSafe(t *testing.T) {
	t.Parallel()
	sweepJobs(context.Background(), nil, nil, time.Now(), slog.New(slog.DiscardHandler))
}
