package app

// ipsla.go wires the composition root to the IP-SLA read use-case (P-D8,
// #3463). The adapter implements sla.Reader over the metrics repository,
// resolving the database lazily; a nil database yields sla.ErrUnavailable so
// the handler degrades to 503.

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/sla"
)

// NewIPSLA builds the IP-SLA read use-case over a lazy database accessor.
func NewIPSLA(db func() *database.DB) *sla.Service {
	return sla.NewService(ipslaReader{db: db})
}

type ipslaReader struct {
	db func() *database.DB
}

// metrics resolves the repository, or sla.ErrUnavailable with no database.
func (a ipslaReader) metrics() (*database.MetricsRepository, error) {
	db := a.db()
	if db == nil {
		return nil, sla.ErrUnavailable
	}
	return db.Metrics(), nil
}

func (a ipslaReader) Operations(ctx context.Context) ([]sla.Operation, error) {
	m, err := a.metrics()
	if err != nil {
		return nil, err
	}
	return m.IPSLAOperations(ctx, database.DefaultClientID)
}

func (a ipslaReader) Runs(ctx context.Context, targetID string, index int, from, to time.Time) ([]sla.Run, error) {
	m, err := a.metrics()
	if err != nil {
		return nil, err
	}
	return m.IPSLARuns(ctx, database.DefaultClientID, targetID, index, from, to)
}
