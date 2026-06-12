package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/health/probemap"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// TestSeedDefaultHealthCheckProbes_FreshInstall verifies that a genuine
// first run seeds the factory health-check targets into the probes table,
// restoring the pre-ADR-0027 out-of-box behavior (a non-empty health-check
// card). Before this seed, getHealthChecksSettings read an empty probes
// table because nothing populated it outside the settings-PUT path.
func TestSeedDefaultHealthCheckProbes_FreshInstall(t *testing.T) {
	db := newTestDB(t)
	s := &Server{config: &config.Config{}, dbConn: db}
	wireHealthSettings(s)
	ctx := context.Background()

	require.NoError(t, s.seedDefaultHealthCheckProbes(ctx, db))

	// The factory ping targets are now persisted as probe rows.
	pings, err := db.Probes().ListProbes(ctx, database.DefaultClientID, probe.KindPing)
	require.NoError(t, err)
	hosts := map[string]bool{}
	for _, p := range pings {
		hosts[p.Target] = true
	}
	require.True(t, hosts["8.8.8.8"], "factory Google DNS ping target should be seeded")
	require.True(t, hosts["1.1.1.1"], "factory Cloudflare ping target should be seeded")

	// The GET handler now reconstructs a non-empty endpoint set.
	gw := httptest.NewRecorder()
	gr := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	s.getHealthChecksSettings(gw, gr)
	require.Equal(t, http.StatusOK, gw.Code, "GET body: %s", gw.Body.String())
	require.Greater(t, len(gw.Body.String()), 2, "settings response must not be empty")
}

// TestSeedDefaultHealthCheckProbes_Idempotent verifies that seeding twice
// does not duplicate the factory set: the persistent marker short-circuits
// the second call.
func TestSeedDefaultHealthCheckProbes_Idempotent(t *testing.T) {
	db := newTestDB(t)
	s := &Server{config: &config.Config{}, dbConn: db}
	ctx := context.Background()

	require.NoError(t, s.seedDefaultHealthCheckProbes(ctx, db))
	first, err := db.Probes().CountProbes(ctx, database.DefaultClientID, probe.KindPing)
	require.NoError(t, err)
	require.Equal(t, 2, first, "factory set has two ping targets")

	require.NoError(t, s.seedDefaultHealthCheckProbes(ctx, db))
	second, err := db.Probes().CountProbes(ctx, database.DefaultClientID, probe.KindPing)
	require.NoError(t, err)
	require.Equal(t, first, second, "second seed must not duplicate the factory set")
}

// TestSeedDefaultHealthCheckProbes_NoReseedAfterDeleteAll verifies the
// persistent-marker contract: once seeded, an operator who deletes every
// health-check probe stays empty across restarts rather than having the
// factory set silently reappear.
func TestSeedDefaultHealthCheckProbes_NoReseedAfterDeleteAll(t *testing.T) {
	db := newTestDB(t)
	s := &Server{config: &config.Config{}, dbConn: db}
	ctx := context.Background()

	require.NoError(t, s.seedDefaultHealthCheckProbes(ctx, db))

	// Operator clears the whole health-check set (settings save with no targets).
	require.NoError(t, db.Probes().ReplaceProbesByKinds(
		ctx, database.DefaultClientID, probemap.Kinds(), nil,
	))

	// A subsequent boot must NOT re-seed.
	require.NoError(t, s.seedDefaultHealthCheckProbes(ctx, db))
	count, err := db.Probes().CountProbes(ctx, database.DefaultClientID, probe.KindPing)
	require.NoError(t, err)
	require.Equal(t, 0, count, "deleting all probes must not trigger a re-seed")
}

// TestSeedDefaultHealthCheckProbes_PreservesExistingProbes verifies the
// upgrade guard: an install that already holds health-check probes (saved
// before this seeding existed, so the marker is absent) is left untouched —
// the operator's set is never overwritten with factory defaults.
func TestSeedDefaultHealthCheckProbes_PreservesExistingProbes(t *testing.T) {
	db := newTestDB(t)
	s := &Server{config: &config.Config{}, dbConn: db}
	ctx := context.Background()

	// Pre-existing operator-configured set, no seed marker, written straight to
	// the probes table (the store of record) via the shared mapping.
	custom := TestsSettingsResponse{
		PingTargets: []PingTargetResponse{{Name: "lab", Host: "10.0.0.1", Enabled: true}},
	}
	hc := requestEndpointTargets(&custom)
	customProbes, err := probemap.ProbesFromConfig(&hc)
	require.NoError(t, err)
	require.NoError(t, db.Probes().ReplaceProbesByKinds(
		ctx, database.DefaultClientID, probemap.Kinds(), customProbes,
	))

	require.NoError(t, s.seedDefaultHealthCheckProbes(ctx, db))

	pings, err := db.Probes().ListProbes(ctx, database.DefaultClientID, probe.KindPing)
	require.NoError(t, err)
	require.Len(t, pings, 1, "existing set must not be replaced by the factory defaults")
	require.Equal(t, "10.0.0.1", pings[0].Target)
}
