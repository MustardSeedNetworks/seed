package app

// ifstats.go wires the composition root to the interface statistics read
// use-case (UI-SEED-21, #3191, ADR-0020). The adapter implements ifstats.Store
// over the metrics repository, resolving the database lazily; a nil database
// yields ifstats.ErrUnavailable so the handler degrades to 503.

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
)

// NewInterfaceStats builds the interface statistics read use-case over a lazy
// database accessor.
func NewInterfaceStats(db func() *database.DB) *ifstats.Service {
	return ifstats.NewService(interfaceStatsStore{db: db})
}

type interfaceStatsStore struct {
	db func() *database.DB
}

func (a interfaceStatsStore) Interfaces(ctx context.Context) ([]ifstats.Interface, error) {
	db := a.db()
	if db == nil {
		return nil, ifstats.ErrUnavailable
	}
	return db.Metrics().InterfaceStats(ctx, database.DefaultClientID)
}

func (a interfaceStatsStore) InterfaceRates(
	ctx context.Context, targetID string, ifIndex uint32, from, to time.Time,
) ([]ifstats.Rates, error) {
	db := a.db()
	if db == nil {
		return nil, ifstats.ErrUnavailable
	}
	return db.Metrics().InterfaceRates(ctx, database.DefaultClientID, targetID, ifIndex, from, to)
}
