package database_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
)

func autotestRecord(id string, started time.Time) *autotest.Report {
	return &autotest.Report{
		ID: id, Interface: "eth0", JackLabel: "B2-3F-R301-P07", AssetTag: "PP-0042",
		Verdict: autotest.OutcomeFail, StartedAt: started, DurationMs: 4210,
		Stages: []autotest.StageReport{
			{
				Stage: autotest.StageLink, Outcome: autotest.OutcomePass, Reason: autotest.ReasonOK,
				Readings: []autotest.Reading{{Label: "speed", Value: "1000 Mb/s"}}, DurationMs: 3,
			},
			{
				Stage: autotest.StageDHCP, Outcome: autotest.OutcomeFail, Reason: autotest.ReasonNoLeaseFound,
				Message: "no lease", DurationMs: 4000,
			},
		},
	}
}

// A stored record reads back exactly, newest first, and an id is stored once.
func TestAutotestResultsRoundTrip(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := t.Context()
	repo := db.AutotestResults()

	older := autotestRecord("0b6c3f0e-6a53-4c43-9a43-2f1d0f6b8a01",
		time.Date(2026, 10, 16, 9, 0, 0, 123456789, time.UTC))
	newer := autotestRecord("0b6c3f0e-6a53-4c43-9a43-2f1d0f6b8a02",
		time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC))
	require.NoError(t, repo.Insert(ctx, older))
	require.NoError(t, repo.Insert(ctx, newer))

	got, err := repo.Get(ctx, older.ID)
	require.NoError(t, err)
	require.Equal(t, older, got)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, newer.ID, list[0].ID)

	require.ErrorIs(t, repo.Insert(ctx, older), autotest.ErrExists)
	_, err = repo.Get(ctx, "0b6c3f0e-6a53-4c43-9a43-2f1d0f6b8a03")
	require.ErrorIs(t, err, autotest.ErrNotFound)
}
