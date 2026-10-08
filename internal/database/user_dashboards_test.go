package database_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
)

// Layouts are per user: one user's save is invisible to another, a saved
// empty dashboard is not the same as never having saved, and the row goes
// with its user.
func TestUserDashboardRoundTrip(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := t.Context()

	for _, name := range []string{"alice", "bob", "carol"} {
		_, err := db.CreateUser(ctx, name, "$2a$10$x", roles.Viewer)
		require.NoError(t, err)
	}
	// DeleteUser refuses the last admin; this one keeps alice deletable.
	_, err := db.CreateUser(ctx, "root", "$2a$10$x", roles.Admin)
	require.NoError(t, err)

	widgets, saved, err := db.GetUserDashboard(ctx, "alice")
	require.NoError(t, err)
	require.False(t, saved, "a user who never saved has no layout")
	require.Nil(t, widgets)

	require.NoError(t, db.SetUserDashboard(ctx, "alice", []string{"gateway", "link"}))
	require.NoError(t, db.SetUserDashboard(ctx, "alice", []string{"link", "dns", "gateway"}))
	require.NoError(t, db.SetUserDashboard(ctx, "bob", []string{}))

	widgets, saved, err = db.GetUserDashboard(ctx, "alice")
	require.NoError(t, err)
	require.True(t, saved)
	require.Equal(t, []string{"link", "dns", "gateway"}, widgets, "the last save wins, in order")

	widgets, saved, err = db.GetUserDashboard(ctx, "bob")
	require.NoError(t, err)
	require.True(t, saved, "an empty dashboard is a saved layout")
	require.Empty(t, widgets)

	_, saved, err = db.GetUserDashboard(ctx, "carol")
	require.NoError(t, err)
	require.False(t, saved, "another user's save is not this user's layout")

	require.ErrorIs(t, db.SetUserDashboard(ctx, "nobody", []string{"link"}), users.ErrUserNotFound)
	_, _, err = db.GetUserDashboard(ctx, "nobody")
	require.ErrorIs(t, err, users.ErrUserNotFound)

	require.NoError(t, db.DeleteUser(ctx, "alice"))
	var rows int
	require.NoError(t, db.WriteConn().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_dashboards`).Scan(&rows))
	require.Equal(t, 1, rows, "the layout goes with the deleted account; bob's stays")
}
