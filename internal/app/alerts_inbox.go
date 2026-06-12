package app

// alerts_inbox.go wires the composition root to the alert-inbox use-case
// (ADR-0020, WS-A8). The adapter implements the inbox.Repository port over the
// alert repository, resolving the database lazily; a nil database yields
// inbox.ErrUnavailable so the handler degrades to 503 rather than panicking.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/alerts/inbox"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// NewAlertInbox builds the alert-inbox use-case over a lazy database accessor.
func NewAlertInbox(db func() *database.DB) *inbox.Service {
	return inbox.NewService(alertInboxRepo{db: db})
}

// alertInboxRepo implements inbox.Repository over the alert repository. A nil
// database makes every call return inbox.ErrUnavailable.
type alertInboxRepo struct {
	db func() *database.DB
}

func (a alertInboxRepo) repo() (*database.AlertRepository, error) {
	db := a.db()
	if db == nil {
		return nil, inbox.ErrUnavailable
	}
	return db.Alerts(), nil
}

func (a alertInboxRepo) List(
	ctx context.Context, opts database.AlertListOptions,
) ([]*database.Alert, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.List(ctx, opts)
}

func (a alertInboxRepo) Acknowledge(ctx context.Context, id int64, username string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.Acknowledge(ctx, id, username)
}

func (a alertInboxRepo) Resolve(ctx context.Context, id int64) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.Resolve(ctx, id)
}
