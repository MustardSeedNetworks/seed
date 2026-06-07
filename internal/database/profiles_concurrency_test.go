package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// TestProfileUpdateIfMatch covers the optimistic-concurrency variant: a wrong
// row_version conflicts, the current row_version writes through (and bumps the
// version), and a missing row is 404. The token is the monotonic row_version
// (not updated_at) so a sub-second double-write is still detected.
func TestProfileUpdateIfMatch(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Profiles()

	require.NoError(t, repo.Create(ctx, &database.Profile{ID: "p1", Name: "orig", ConfigJSON: "{}"}))
	cur, err := repo.Get(ctx, "p1")
	require.NoError(t, err)
	require.Equal(t, int64(1), cur.RowVersion, "a fresh profile starts at row_version 1")

	// A stale/wrong version is a conflict — the row exists but has moved on.
	staleErr := repo.UpdateIfMatch(
		ctx,
		&database.Profile{ID: "p1", Name: "x", ConfigJSON: "{}"},
		999,
	)
	require.ErrorIs(t, staleErr, database.ErrProfileConflict)

	// The current version writes through and bumps row_version.
	updated := &database.Profile{ID: "p1", Name: "updated", ConfigJSON: `{"x":1}`}
	require.NoError(t, repo.UpdateIfMatch(ctx, updated, cur.RowVersion))
	require.Equal(t, int64(2), updated.RowVersion, "a write bumps the in-memory row_version")

	got, err := repo.Get(ctx, "p1")
	require.NoError(t, err)
	require.Equal(t, "updated", got.Name)
	require.JSONEq(t, `{"x":1}`, got.ConfigJSON)
	require.Equal(t, int64(2), got.RowVersion, "the persisted row_version advanced")

	// Re-using the now-stale version conflicts (the sub-second window #1559 left open).
	reuseErr := repo.UpdateIfMatch(ctx, &database.Profile{ID: "p1", Name: "z", ConfigJSON: "{}"}, cur.RowVersion)
	require.ErrorIs(t, reuseErr, database.ErrProfileConflict)

	// A missing row is not-found, not a conflict.
	missingErr := repo.UpdateIfMatch(ctx, &database.Profile{ID: "ghost", Name: "y", ConfigJSON: "{}"}, 1)
	require.ErrorIs(t, missingErr, database.ErrProfileNotFound)

	// An unconditional Update also bumps the version (so a held token goes stale).
	uncond := &database.Profile{ID: "p1", Name: "uncond", ConfigJSON: "{}"}
	require.NoError(t, repo.Update(ctx, uncond))
	require.Equal(t, int64(3), uncond.RowVersion)
}
