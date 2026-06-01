package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
)

func newSuppressionDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(t.TempDir() + "/seed.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestAlertSuppressions_MarkAndIsSuppressedRoundtrip(t *testing.T) {
	t.Parallel()
	db := newSuppressionDB(t)
	ctx := context.Background()
	repo := db.AlertSuppressions()

	now := time.Now().UTC()
	until := now.Add(5 * time.Minute)
	if err := repo.Mark(ctx, "fp-1", "rule-a", "10.0.0.1", until); err != nil {
		t.Fatalf("Mark: %v", err)
	}

	suppressed, err := repo.IsSuppressed(ctx, "fp-1", now)
	if err != nil {
		t.Fatalf("IsSuppressed: %v", err)
	}
	if !suppressed {
		t.Error("fingerprint should be suppressed within window")
	}
}

func TestAlertSuppressions_IsSuppressedFalseWhenExpired(t *testing.T) {
	t.Parallel()
	db := newSuppressionDB(t)
	ctx := context.Background()
	repo := db.AlertSuppressions()

	past := time.Now().UTC().Add(-time.Hour)
	if err := repo.Mark(ctx, "fp-2", "rule-a", "x", past); err != nil {
		t.Fatalf("Mark: %v", err)
	}

	suppressed, _ := repo.IsSuppressed(ctx, "fp-2", time.Now().UTC())
	if suppressed {
		t.Error("expired suppression should report not-suppressed")
	}
}

func TestAlertSuppressions_IsSuppressedFalseWhenAbsent(t *testing.T) {
	t.Parallel()
	db := newSuppressionDB(t)
	suppressed, err := db.AlertSuppressions().IsSuppressed(
		context.Background(), "never-marked", time.Now().UTC())
	if err != nil {
		t.Fatalf("IsSuppressed on absent row: %v", err)
	}
	if suppressed {
		t.Error("absent fingerprint should not be suppressed")
	}
}

func TestAlertSuppressions_MarkOverwrites(t *testing.T) {
	t.Parallel()
	db := newSuppressionDB(t)
	ctx := context.Background()
	repo := db.AlertSuppressions()
	now := time.Now().UTC()

	short := now.Add(time.Second)
	long := now.Add(time.Hour)
	if err := repo.Mark(ctx, "fp", "rule", "x", short); err != nil {
		t.Fatalf("first Mark: %v", err)
	}
	if err := repo.Mark(ctx, "fp", "rule", "x", long); err != nil {
		t.Fatalf("second Mark: %v", err)
	}

	// At a moment past the short deadline but before the long one,
	// the overwriting Mark should still hold.
	check := short.Add(time.Minute)
	suppressed, _ := repo.IsSuppressed(ctx, "fp", check)
	if !suppressed {
		t.Error("second Mark should overwrite first; expected still-suppressed")
	}
}

func TestAlertSuppressions_PurgeExpiredRemovesOnlyExpired(t *testing.T) {
	t.Parallel()
	db := newSuppressionDB(t)
	ctx := context.Background()
	repo := db.AlertSuppressions()
	now := time.Now().UTC()

	if err := repo.Mark(ctx, "expired", "r", "x", now.Add(-time.Hour)); err != nil {
		t.Fatalf("Mark expired: %v", err)
	}
	if err := repo.Mark(ctx, "active", "r", "y", now.Add(time.Hour)); err != nil {
		t.Fatalf("Mark active: %v", err)
	}

	removed, err := repo.PurgeExpired(ctx, now)
	if err != nil {
		t.Fatalf("PurgeExpired: %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}

	stillActive, _ := repo.IsSuppressed(ctx, "active", now)
	if !stillActive {
		t.Error("active suppression should survive purge")
	}
}
