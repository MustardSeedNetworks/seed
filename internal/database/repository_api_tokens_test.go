// SPDX-License-Identifier: BUSL-1.1

package database_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
)

func setupAPITokenTest(t *testing.T, owners ...string) (*database.APITokenRepository, context.Context) {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "seed-test-*.db")
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

	// The hardening migration added an ON DELETE CASCADE FK from
	// api_tokens.owner_username -> users.username. Tests must seed any
	// owner referenced by an inserted token, otherwise the FK rejects
	// the row. Default ("alice","bob") covers the existing cases; extra
	// owners can be passed in by name.
	ctx := context.Background()
	if len(owners) == 0 {
		owners = []string{"alice", "bob"}
	}
	for _, name := range owners {
		_, createErr := db.CreateUser(ctx, name, "$2a$10$x", database.RoleAdmin)
		if createErr != nil {
			t.Fatalf("seed user %q: %v", name, createErr)
		}
	}

	return database.NewAPITokenRepository(db), ctx
}

func TestAPITokenInsertAndLookup(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAPITokenTest(t)

	rec := database.APITokenRecord{
		ID:            "tokenid01",
		OwnerUsername: "alice",
		Name:          "ci-bot",
		TokenHash:     "deadbeef",
		Prefix:        "sd_pat_abcd",
		CreatedAt:     time.Now().UTC(),
	}
	if err := repo.Insert(ctx, rec); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	got, err := repo.FindActiveByHash(ctx, "deadbeef")
	if err != nil {
		t.Fatalf("FindActiveByHash: %v", err)
	}
	if got.OwnerUsername != "alice" || got.Name != "ci-bot" {
		t.Errorf("unexpected record: %+v", got)
	}
	if !got.IsActive() {
		t.Error("expected IsActive == true on fresh token")
	}
}

func TestAPITokenRevokeMakesInactive(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAPITokenTest(t)

	rec := database.APITokenRecord{
		ID: "tokenid02", OwnerUsername: "bob", Name: "old",
		TokenHash: "cafe", Prefix: "sd_pat_cafe", CreatedAt: time.Now().UTC(),
	}
	if err := repo.Insert(ctx, rec); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := repo.Revoke(ctx, rec.ID, "bob"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if _, err := repo.FindActiveByHash(ctx, "cafe"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows after revoke, got: %v", err)
	}

	// Second revoke of the same token is a no-op (already revoked).
	if err := repo.Revoke(ctx, rec.ID, "bob"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows on double-revoke, got: %v", err)
	}
}

func TestAPITokenRevokeRequiresOwner(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAPITokenTest(t)

	rec := database.APITokenRecord{
		ID: "tokenid03", OwnerUsername: "alice", Name: "mine",
		TokenHash: "beef", Prefix: "sd_pat_beef", CreatedAt: time.Now().UTC(),
	}
	if err := repo.Insert(ctx, rec); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// Bob cannot revoke Alice's token.
	if err := repo.Revoke(ctx, rec.ID, "bob"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows for wrong owner, got: %v", err)
	}
	// Token is still active for Alice.
	if _, err := repo.FindActiveByHash(ctx, "beef"); err != nil {
		t.Errorf("token should still be active for owner: %v", err)
	}
}

func TestAPITokenListByOwnerSorted(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAPITokenTest(t)

	now := time.Now().UTC()
	tokens := []database.APITokenRecord{
		{
			ID: "t1", OwnerUsername: "alice", Name: "first",
			TokenHash: "h1", Prefix: "sd_pat_aaa", CreatedAt: now.Add(-3 * time.Hour),
		},
		{
			ID: "t2", OwnerUsername: "alice", Name: "second",
			TokenHash: "h2", Prefix: "sd_pat_bbb", CreatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID: "t3", OwnerUsername: "bob", Name: "other",
			TokenHash: "h3", Prefix: "sd_pat_ccc", CreatedAt: now,
		},
	}
	for _, t0 := range tokens {
		if err := repo.Insert(ctx, t0); err != nil {
			t.Fatalf("Insert %s: %v", t0.ID, err)
		}
	}

	rows, err := repo.ListByOwner(ctx, "alice")
	if err != nil {
		t.Fatalf("ListByOwner: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows for alice, got %d", len(rows))
	}
	// DESC by created_at ⇒ "second" first.
	if rows[0].Name != "second" || rows[1].Name != "first" {
		t.Errorf("unexpected order: [%s, %s]", rows[0].Name, rows[1].Name)
	}
}

func TestAPITokenTouchLastUsedUpdatesTimestamp(t *testing.T) {
	t.Parallel()
	repo, ctx := setupAPITokenTest(t, "carol")

	rec := database.APITokenRecord{
		ID: "tokenid04", OwnerUsername: "carol", Name: "k",
		TokenHash: "abcd", Prefix: "sd_pat_xxx", CreatedAt: time.Now().UTC(),
	}
	if err := repo.Insert(ctx, rec); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	before, _ := repo.FindActiveByHash(ctx, "abcd")
	if !before.LastUsedAt.IsZero() {
		t.Error("expected LastUsedAt to be zero before TouchLastUsed")
	}
	if err := repo.TouchLastUsed(ctx, rec.ID); err != nil {
		t.Fatalf("TouchLastUsed: %v", err)
	}
	after, _ := repo.FindActiveByHash(ctx, "abcd")
	if after.LastUsedAt.IsZero() {
		t.Error("expected LastUsedAt to be set after TouchLastUsed")
	}
}
