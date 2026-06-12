package app

// polling.go wires the composition root to the polling-targets CRUD use-case
// (ADR-0020, WS-A7). The adapter implements the targets.Repository port over the
// polling-target repository, resolving the database lazily; a nil database yields
// targets.ErrUnavailable so the handler degrades to 503 rather than panicking.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling/targets"
)

// NewPollingTargets builds the polling-targets CRUD use-case over a lazy database
// accessor.
func NewPollingTargets(db func() *database.DB) *targets.Service {
	return targets.NewService(pollingTargetRepo{db: db})
}

// pollingTargetRepo implements targets.Repository over the polling-target
// repository. A nil database makes every call return targets.ErrUnavailable.
type pollingTargetRepo struct {
	db func() *database.DB
}

func (a pollingTargetRepo) repo() (*database.PollingTargetRepository, error) {
	db := a.db()
	if db == nil {
		return nil, targets.ErrUnavailable
	}
	return db.PollingTargets(), nil
}

func (a pollingTargetRepo) List(ctx context.Context, clientID string) ([]*database.PollingTarget, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.List(ctx, clientID)
}

func (a pollingTargetRepo) Get(ctx context.Context, id string) (*database.PollingTarget, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.Get(ctx, id)
}

func (a pollingTargetRepo) Create(ctx context.Context, t *database.PollingTarget) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.Create(ctx, t)
}

func (a pollingTargetRepo) Update(ctx context.Context, t *database.PollingTarget) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.Update(ctx, t)
}

func (a pollingTargetRepo) Delete(ctx context.Context, id string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.Delete(ctx, id)
}
