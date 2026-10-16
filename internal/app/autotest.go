package app

// autotest.go wires the composition root to the AutoTest record store (P-E2).
// A nil database makes every call autotest.ErrUnavailable, so a run fails
// rather than reporting a record that was never kept, and the routes answer 503.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
)

// NewAutotestResults builds the record store over a lazy database accessor.
func NewAutotestResults(persist func() *Persistence) autotest.Store {
	return autotestStore{db: lazyHandle(persist)}
}

type autotestStore struct {
	db func() *database.DB
}

func (a autotestStore) repo() (*database.AutotestResultsRepository, error) {
	db := a.db()
	if db == nil {
		return nil, autotest.ErrUnavailable
	}
	return db.AutotestResults(), nil
}

func (a autotestStore) Insert(ctx context.Context, r *autotest.Report) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.Insert(ctx, r)
}

func (a autotestStore) Get(ctx context.Context, id string) (*autotest.Report, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.Get(ctx, id)
}

func (a autotestStore) List(ctx context.Context) ([]autotest.Report, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}
