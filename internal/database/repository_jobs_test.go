// SPDX-License-Identifier: BUSL-1.1

package database_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
)

func setupJobsTest(t *testing.T) (*database.JobRepository, context.Context) {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "seed-jobs-*.db")
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
	return db.Jobs(), context.Background()
}

func TestJobsSaveInsertThenGet(t *testing.T) {
	t.Parallel()
	repo, ctx := setupJobsTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	rec := &database.JobRecord{
		ID:        "job-1",
		Kind:      "speedtest",
		State:     "queued",
		Progress:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Save(ctx, rec); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Get(ctx, "job-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Kind != "speedtest" || got.State != "queued" {
		t.Errorf("got kind=%q state=%q, want speedtest/queued", got.Kind, got.State)
	}
	if !got.CompletedAt.IsZero() {
		t.Errorf("active job should have zero CompletedAt, got %v", got.CompletedAt)
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
}

func TestJobsSaveUpsertPreservesCreatedAtAndKind(t *testing.T) {
	t.Parallel()
	repo, ctx := setupJobsTest(t)

	created := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	if err := repo.Save(ctx, &database.JobRecord{
		ID: "job-1", Kind: "iperf", State: "queued",
		CreatedAt: created, UpdatedAt: created,
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// A later transition writes the full snapshot; created_at and kind must
	// NOT be overwritten by the upsert, mutable lifecycle columns must be.
	completed := time.Now().UTC().Truncate(time.Second)
	if err := repo.Save(ctx, &database.JobRecord{
		ID: "job-1", Kind: "SHOULD-BE-IGNORED", State: "succeeded",
		Progress: 1, ResultJSON: `{"ok":true}`,
		CreatedAt: completed, UpdatedAt: completed, CompletedAt: completed,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.Get(ctx, "job-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Kind != "iperf" {
		t.Errorf("kind = %q, want iperf (immutable after insert)", got.Kind)
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v (immutable)", got.CreatedAt, created)
	}
	if got.State != "succeeded" || got.Progress != 1 || got.ResultJSON != `{"ok":true}` {
		t.Errorf("mutable cols not updated: state=%q progress=%v result=%q",
			got.State, got.Progress, got.ResultJSON)
	}
	if got.CompletedAt.IsZero() {
		t.Error("terminal job should have non-zero CompletedAt")
	}
}

func TestJobsGetNotFound(t *testing.T) {
	t.Parallel()
	repo, ctx := setupJobsTest(t)

	_, err := repo.Get(ctx, "missing")
	if !errors.Is(err, database.ErrJobNotFound) {
		t.Errorf("got %v, want ErrJobNotFound", err)
	}
}

func TestJobsListNewestFirst(t *testing.T) {
	t.Parallel()
	repo, ctx := setupJobsTest(t)

	base := time.Now().UTC().Truncate(time.Second)
	for i, id := range []string{"old", "mid", "new"} {
		ts := base.Add(time.Duration(i) * time.Minute)
		if err := repo.Save(ctx, &database.JobRecord{
			ID: id, Kind: "k", State: "queued", CreatedAt: ts, UpdatedAt: ts,
		}); err != nil {
			t.Fatalf("save %s: %v", id, err)
		}
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}
	if list[0].ID != "new" || list[1].ID != "mid" || list[2].ID != "old" {
		t.Errorf("order = %s,%s,%s, want new,mid,old",
			list[0].ID, list[1].ID, list[2].ID)
	}
}

func TestJobsDeleteCompletedBefore(t *testing.T) {
	t.Parallel()
	repo, ctx := setupJobsTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	// Terminal, old -> eligible.
	mustSave(ctx, t, repo, &database.JobRecord{
		ID: "old-done", Kind: "k", State: "succeeded", Progress: 1,
		CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour),
		CompletedAt: now.Add(-2 * time.Hour),
	})
	// Terminal, recent -> kept.
	mustSave(ctx, t, repo, &database.JobRecord{
		ID: "recent-done", Kind: "k", State: "failed", Error: "boom",
		CreatedAt: now, UpdatedAt: now, CompletedAt: now,
	})
	// Active (no completed_at) -> never deleted regardless of age.
	mustSave(ctx, t, repo, &database.JobRecord{
		ID: "active", Kind: "k", State: "running",
		CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour),
	})

	removed, err := repo.DeleteCompletedBefore(ctx, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}

	list, _ := repo.List(ctx)
	if len(list) != 2 {
		t.Fatalf("remaining = %d, want 2", len(list))
	}
	for _, j := range list {
		if j.ID == "old-done" {
			t.Error("old terminal job should have been deleted")
		}
	}
}

// TestJobsStateCheckRejectsInvalid proves the STRICT-table CHECK on state
// (Phase 5b/5c hardening) rejects an out-of-domain value at the DB, not just in
// Go. Save surfaces the SQLite constraint failure as an error.
func TestJobsStateCheckRejectsInvalid(t *testing.T) {
	t.Parallel()
	repo, ctx := setupJobsTest(t)

	now := time.Now().UTC()
	err := repo.Save(ctx, &database.JobRecord{
		ID: "bad", Kind: "k", State: "bogus", CreatedAt: now, UpdatedAt: now,
	})
	if err == nil {
		t.Fatal("expected CHECK violation for invalid state, got nil")
	}
}

// TestJobsProgressCheckRejectsOutOfRange proves the CHECK (progress 0..1).
func TestJobsProgressCheckRejectsOutOfRange(t *testing.T) {
	t.Parallel()
	repo, ctx := setupJobsTest(t)

	now := time.Now().UTC()
	err := repo.Save(ctx, &database.JobRecord{
		ID: "bad", Kind: "k", State: "running", Progress: 1.5,
		CreatedAt: now, UpdatedAt: now,
	})
	if err == nil {
		t.Fatal("expected CHECK violation for progress > 1, got nil")
	}
}

func TestJobsMarkInterrupted(t *testing.T) {
	t.Parallel()
	repo, ctx := setupJobsTest(t)

	now := time.Now().UTC().Truncate(time.Second)
	mustSave(ctx, t, repo, &database.JobRecord{ID: "q", Kind: "k", State: "queued", CreatedAt: now, UpdatedAt: now})
	mustSave(
		ctx,
		t,
		repo,
		&database.JobRecord{ID: "r", Kind: "k", State: "running", Progress: 0.5, CreatedAt: now, UpdatedAt: now},
	)
	mustSave(ctx, t, repo, &database.JobRecord{
		ID: "ok", Kind: "k", State: "succeeded", Progress: 1,
		CreatedAt: now, UpdatedAt: now, CompletedAt: now,
	})

	n, err := repo.MarkInterrupted(ctx)
	if err != nil {
		t.Fatalf("mark interrupted: %v", err)
	}
	if n != 2 {
		t.Errorf("interrupted = %d, want 2 (queued + running)", n)
	}
	for _, id := range []string{"q", "r"} {
		got, gErr := repo.Get(ctx, id)
		if gErr != nil {
			t.Fatalf("get %s: %v", id, gErr)
		}
		if got.State != "failed" || got.Error == "" || got.CompletedAt.IsZero() {
			t.Errorf("%s: state=%q error=%q completedAt=%v, want failed/non-empty/non-zero",
				id, got.State, got.Error, got.CompletedAt)
		}
	}
	if got, _ := repo.Get(ctx, "ok"); got.State != "succeeded" {
		t.Errorf("terminal job mutated to %q", got.State)
	}
}

func mustSave(ctx context.Context, t *testing.T, repo *database.JobRepository, rec *database.JobRecord) {
	t.Helper()
	if err := repo.Save(ctx, rec); err != nil {
		t.Fatalf("save %s: %v", rec.ID, err)
	}
}
