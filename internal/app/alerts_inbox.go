package app

// alerts_inbox.go wires the composition root to the alert-inbox use-case
// (ADR-0020, WS-A8). The adapter implements the inbox.Repository port over the
// alert repository, resolving the database lazily; a nil database yields
// inbox.ErrUnavailable so the handler degrades to 503 rather than panicking.

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/inbox"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
)

// NewAlertInbox builds the alert-inbox use-case over a lazy database accessor.
func NewAlertInbox(db func() *database.DB) *inbox.Service {
	return inbox.NewService(alertInboxRepo{db: db})
}

// NewAlertNarrator explains alerts for the delivery channels with the inbox's
// own narratives (P-B7), rendered in the default language: a receiver has no
// reader whose language Seed could follow, and the rest of what each channel
// sends is English.
func NewAlertNarrator(db *database.DB) alertdelivery.Narrator {
	return alertNarrator{
		inbox: NewAlertInbox(func() *database.DB { return db }),
		t:     i18n.NewLocalizer(i18n.DefaultLanguage),
	}
}

type alertNarrator struct {
	inbox *inbox.Service
	t     *i18n.Localizer
}

func (n alertNarrator) Narrate(ctx context.Context, a *alerts.Alert) (narrative.Text, bool, error) {
	stories, err := n.inbox.Narratives(ctx, []*alerts.Alert{a})
	if err != nil {
		return narrative.Text{}, false, err
	}
	story, ok := stories[a.ID]
	if !ok {
		return narrative.Text{}, false, nil
	}
	return story.Render(n.t.TWithData), true, nil
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
	ctx context.Context, opts alerts.ListOptions,
) ([]*alerts.Alert, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.List(ctx, opts)
}

func (a alertInboxRepo) ListEffects(
	ctx context.Context, causeIDs []int64,
) ([]*alerts.Alert, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.ListEffects(ctx, causeIDs)
}

// DeviceNames reads the names of the default client's polling targets, the
// ids the SNMP pipelines stamp on an alert as its Source.
func (a alertInboxRepo) DeviceNames(ctx context.Context) (map[string]string, error) {
	db := a.db()
	if db == nil {
		return nil, inbox.ErrUnavailable
	}
	targets, err := db.PollingTargets().ListAll(ctx, database.DefaultClientID)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(targets))
	for _, t := range targets {
		names[t.ID] = t.Name
	}
	return names, nil
}

// InterfaceErrorPeaks reads the rates the SNMP pipeline stored for the
// default client's target, the client DeviceNames reads targets from.
func (a alertInboxRepo) InterfaceErrorPeaks(
	ctx context.Context, targetID string, ifIndex uint32, from, to time.Time,
) (ifrate.ErrorPeaks, error) {
	db := a.db()
	if db == nil {
		return ifrate.ErrorPeaks{}, inbox.ErrUnavailable
	}
	return db.Metrics().InterfaceErrorPeaks(ctx, database.DefaultClientID, targetID, ifIndex, from, to)
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
