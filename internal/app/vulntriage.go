package app

// vulntriage.go wires the composition root to the vulnerability-triage
// use-case (S6-2, #899, ADR-0020). The adapter implements vulntriage.Store over
// the database vulnerability repository, resolving the handle lazily so a
// later-set value (the api test harness) is honored.

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/security/vulntriage"
)

// NewVulnTriage builds the vulnerability-triage use-case over a lazy database
// accessor.
func NewVulnTriage(db func() *database.DB) *vulntriage.Service {
	return vulntriage.NewService(vulnTriageStore{db: db})
}

type vulnTriageStore struct {
	db func() *database.DB
}

func (a vulnTriageStore) ListFindings(
	ctx context.Context, opts vulntriage.ListOptions,
) ([]vulntriage.Finding, error) {
	return a.db().Vulnerabilities().ListFindings(ctx, opts)
}

func (a vulnTriageStore) SetStatus(
	ctx context.Context, id int64, to vulntriage.Status, actor, reason string, at time.Time,
) error {
	return a.db().Vulnerabilities().SetStatus(ctx, id, to, actor, reason, at)
}

func (a vulnTriageStore) History(ctx context.Context, id int64) ([]vulntriage.StatusChange, error) {
	return a.db().Vulnerabilities().History(ctx, id)
}
