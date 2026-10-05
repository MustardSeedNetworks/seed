package app

// history.go wires the composition root to the history read use-case (#175,
// ADR-0020). The adapter implements history.Store over the database history
// repository, resolving the handle lazily so a later-set value (the api test
// harness) is honored.

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/history"
)

// NewHistory builds the history read use-case over a lazy database accessor.
func NewHistory(db func() *database.DB) *history.Service {
	return history.NewService(historyStore{db: db})
}

type historyStore struct {
	db func() *database.DB
}

func (a historyStore) ProbeTrendRaw(
	ctx context.Context, clientID, probeID string, from, to time.Time, daily bool,
) ([]history.ProbeTrendPoint, error) {
	return a.db().History().ProbeTrendRaw(ctx, clientID, probeID, from, to, daily)
}

func (a historyStore) ProbeTrendRollup(
	ctx context.Context, clientID, probeID string, from, to time.Time, daily bool,
) ([]history.ProbeTrendPoint, error) {
	return a.db().History().ProbeTrendRollup(ctx, clientID, probeID, from, to, daily)
}

func (a historyStore) AnomalyCountsByDayLive(
	ctx context.Context,
	from, to time.Time,
) ([]history.AnomalyDayCount, error) {
	return a.db().History().AnomalyCountsByDayLive(ctx, from, to)
}

func (a historyStore) AnomalyCountsByDayRollup(
	ctx context.Context,
	from, to time.Time,
) ([]history.AnomalyDayCount, error) {
	return a.db().History().AnomalyCountsByDayRollup(ctx, from, to)
}
