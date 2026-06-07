package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// TestProfileUpdateIfMatch covers the optimistic-concurrency variant: a wrong
// ETag conflicts, the current ETag writes through, and a missing row is 404.
func TestProfileUpdateIfMatch(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Profiles()

	require.NoError(t, repo.Create(ctx, &database.Profile{ID: "p1", Name: "orig", ConfigJSON: "{}"}))
	cur, err := repo.Get(ctx, "p1")
	require.NoError(t, err)
	etag := cur.UpdatedAt.Format(time.RFC3339)

	// A stale/wrong ETag is a conflict — the row exists but has a different version.
	staleErr := repo.UpdateIfMatch(
		ctx,
		&database.Profile{ID: "p1", Name: "x", ConfigJSON: "{}"},
		"1999-01-01T00:00:00Z",
	)
	require.ErrorIs(t, staleErr, database.ErrProfileConflict)

	// The current ETag writes through.
	require.NoError(
		t,
		repo.UpdateIfMatch(ctx, &database.Profile{ID: "p1", Name: "updated", ConfigJSON: `{"x":1}`}, etag),
	)
	got, err := repo.Get(ctx, "p1")
	require.NoError(t, err)
	require.Equal(t, "updated", got.Name)
	require.JSONEq(t, `{"x":1}`, got.ConfigJSON)

	// A missing row is not-found, not a conflict.
	missingErr := repo.UpdateIfMatch(ctx, &database.Profile{ID: "ghost", Name: "y", ConfigJSON: "{}"}, etag)
	require.ErrorIs(t, missingErr, database.ErrProfileNotFound)
}
