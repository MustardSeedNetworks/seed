package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// TestReplaceProbesByKinds verifies the transactional replace-by-kind
// used by the health-check settings save (ADR-0027 P2): probes of the
// named kinds are swapped wholesale while probes of other kinds are
// left untouched.
func TestReplaceProbesByKinds(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Probes()

	// Seed: two ping probes (health-check kind) and one dns probe (not a
	// health-check kind — must survive the replace).
	seed := []*database.Probe{
		{Kind: "ping", DisplayName: "a", Target: "a.example", IntervalSeconds: 60, Enabled: true},
		{Kind: "ping", DisplayName: "b", Target: "b.example", IntervalSeconds: 60, Enabled: false},
		{Kind: "dns", DisplayName: "dns1", Target: "example.com", IntervalSeconds: 60, Enabled: true},
	}
	for _, p := range seed {
		require.NoError(t, repo.CreateProbe(ctx, p))
	}

	// Replace all ping + tcp probes with a single new tcp probe.
	replacement := []*database.Probe{
		{Kind: "tcp", DisplayName: "c", Target: "c.example", IntervalSeconds: 30, Enabled: true},
	}
	err := repo.ReplaceProbesByKinds(ctx, database.DefaultClientID, []string{"ping", "tcp"}, replacement)
	require.NoError(t, err)

	all, err := repo.ListProbes(ctx, database.DefaultClientID, "")
	require.NoError(t, err)

	byKind := map[string]int{}
	for _, p := range all {
		byKind[p.Kind]++
		require.NotEmpty(t, p.ID, "replaced probe should have an id assigned")
		require.Equal(t, database.DefaultClientID, p.ClientID)
	}
	require.Equal(t, 0, byKind["ping"], "old ping probes should be deleted")
	require.Equal(t, 1, byKind["tcp"], "new tcp probe should be inserted")
	require.Equal(t, 1, byKind["dns"], "dns probe of a non-replaced kind must survive")
}

// TestReplaceProbesByKindsClearsToEmpty verifies that passing no
// replacement probes deletes all probes of the named kinds.
func TestReplaceProbesByKindsClearsToEmpty(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.Probes()

	require.NoError(t, repo.CreateProbe(ctx, &database.Probe{
		Kind: "ping", DisplayName: "a", Target: "a.example", IntervalSeconds: 60, Enabled: true,
	}))

	require.NoError(t, repo.ReplaceProbesByKinds(ctx, database.DefaultClientID, []string{"ping"}, nil))

	count, err := repo.CountProbes(ctx, database.DefaultClientID, "ping")
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
