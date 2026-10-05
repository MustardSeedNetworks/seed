package app

// alerts_delivery.go assembles the alert write chain and the escalator over the
// alerts repository (#368, P-B2, ADR-0020), so the composition root wires them
// without reaching into persistence.

import (
	"log/slog"

	alertcorrelation "github.com/MustardSeedNetworks/seed/internal/alerts/correlation"
	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/escalation"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// NewAlertDelivery returns the delivery Manager and the writer the alert
// pipelines emit through: the alert repository, wrapped in correlation, wrapped
// in delivery. Wrapping the store rather than each pipeline means a pipeline
// added later delivers without being taught to, and wrapping the Manager rather
// than a notifier means the receiver behind it can be changed from Settings
// without rebuilding the chain.
//
// Correlation sits inside delivery: it annotates the alert with the id of the
// earlier alert that probably caused it, and it must do so before the row is
// written, so the cause is on disk and in the delivered payload rather than
// only in the inbox's later reading of it. It is unconditional — the annotation
// is worth having whether or not a receiver is configured.
//
// The Manager exists either way: it is the indirection a later settings write
// re-points, and with no receiver it stores the alert and sends nothing. It is
// pointed at what cfg already says, so a receiver configured in a previous
// session is live from the first alert. A nil cfg (the hand-assembled Server
// several api tests use) has no stored receiver to apply.
func NewAlertDelivery(
	db *database.DB,
	cfg *config.Config,
	logger *slog.Logger,
) (*alertdelivery.Manager, alertdelivery.Writer) {
	manager := alertdelivery.NewManager(db.Alerts(), NewAlertNarrator(db), logger)
	if cfg != nil {
		ApplyAlertReceivers(cfg, manager)
	}
	store := alertcorrelation.WrapWriter(db.Alerts(), alertcorrelation.Config{})
	return manager, alertdelivery.WrapWriter(store, manager)
}

// NewAlertEscalator builds the escalator that re-sends an alert nobody
// acknowledged (P-B2), through manager and on the ladders it holds.
func NewAlertEscalator(
	db *database.DB,
	manager *alertdelivery.Manager,
	logger *slog.Logger,
) (*escalation.Escalator, error) {
	return escalation.New(escalation.Config{
		Store:   db.Alerts(),
		Sender:  manager,
		Ladders: manager.Escalations,
		Logger:  logger,
	})
}
