package api

// Outbound alert delivery (#368) — the one transport Seed has for sending an
// alert somewhere else. It is opt-in: an operator names a receiver in Settings
// and Seed signs every alert it POSTs there.
//
// No receiver configured means silent. No worker runs and no request is made,
// so an air-gapped deployment loses a feature rather than function.
//
// The signing material is never stored in plaintext. It reaches config.json
// only as keyring ciphertext (ADR-0015, the `enc:` prefix), which answers the
// objection the environment-variable version of this file recorded — that a
// key round-tripping through config.json would be readable wherever that file
// is backed up. It is decrypted once, in the composition root, on its way to
// the notifier (#2605).

import (
	"log/slog"

	alertcorrelation "github.com/MustardSeedNetworks/seed/internal/alerts/correlation"
	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// alertStore returns the writer the alert pipelines emit through: the alert
// repository, wrapped in the webhook decorator. Wrapping the store rather than
// each pipeline means a pipeline added later delivers without being taught to,
// and wrapping the delivery Manager rather than a notifier means the receiver
// behind it can be changed from Settings without rebuilding the chain.
func (s *Server) alertStore(db *database.DB, logger *slog.Logger) alertdelivery.Writer {
	// Correlation sits inside delivery: it annotates the alert with the id of
	// the earlier alert that probably caused it, and it must do so before the
	// row is written, so the cause is on disk and in the delivered payload
	// rather than only in the inbox's later reading of it. It is unconditional
	// — the annotation is worth having whether or not a receiver is
	// configured, and with no webhook this is the whole of it.
	store := alertcorrelation.WrapWriter(db.Alerts(), alertcorrelation.Config{})
	return alertdelivery.WrapWriter(store, s.initAlertDelivery(db.Alerts(), logger))
}

// initAlertDelivery builds the delivery Manager and points it at the
// configured receiver, if there is one. The Manager exists either way: it is
// the indirection a later settings write re-points, and with no receiver it
// stores the alert and sends nothing.
func (s *Server) initAlertDelivery(
	recorder alertdelivery.Recorder,
	logger *slog.Logger,
) *alertdelivery.Manager {
	manager := alertdelivery.NewManager(recorder, logger)
	s.alertDelivery = manager
	// Point it at what the config already says, so a receiver configured in a
	// previous session is live from the first alert rather than from the first
	// settings write. A Server built without a config — the hand-assembled one
	// several internal tests use — has no stored receiver to apply.
	if s.config != nil {
		app.ApplyAlertWebhook(s.config, manager)
	}
	return manager
}
