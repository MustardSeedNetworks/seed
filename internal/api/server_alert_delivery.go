package api

// Outbound alert delivery (#368) — the one transport Seed has for sending an
// alert somewhere else. It is opt-in by environment, the same bring-your-own
// shape as the Wi-Fi monitor interface: SEED_ALERT_WEBHOOK_URL names the
// receiver and SEED_ALERT_WEBHOOK_SECRET is the HMAC material it verifies with.
//
// Unset means silent. Nothing is constructed, no goroutine runs, and no request
// is made, so an air-gapped deployment loses a feature rather than function.
//
// The secret is read from the environment and never written to the config file:
// the daemon's environment is operator-owned, and a signing key that round-trips
// through config.json would be readable wherever that file is backed up.

import (
	"log/slog"
	"os"

	alertcorrelation "github.com/MustardSeedNetworks/seed/internal/alerts/correlation"
	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// Environment variables that configure the outbound alert webhook.
const (
	alertWebhookURLEnv = "SEED_ALERT_WEBHOOK_URL"
	alertWebhookKeyEnv = "SEED_ALERT_WEBHOOK_SECRET"
)

// alertStore returns the writer the alert pipelines emit through: the alert
// repository, wrapped in the webhook decorator when a receiver is configured.
// Wrapping the store rather than each pipeline means a pipeline added later
// delivers without being taught to.
func (s *Server) alertStore(db *database.DB, logger *slog.Logger) alertdelivery.Writer {
	// Correlation sits inside delivery: it annotates the alert with the id of
	// the earlier alert that probably caused it, and it must do so before the
	// row is written, so the cause is on disk and in the delivered payload
	// rather than only in the inbox's later reading of it. It is unconditional
	// — the annotation is worth having whether or not a receiver is
	// configured, and with no webhook this is the whole of it.
	store := alertcorrelation.WrapWriter(db.Alerts(), alertcorrelation.Config{})
	if n := s.initAlertDelivery(logger); n != nil {
		return alertdelivery.WrapWriter(store, n)
	}
	return store
}

// initAlertDelivery builds the webhook notifier when the operator configured
// one, and returns nil otherwise. A URL that is set but unusable is logged at
// error level rather than made fatal: a webhook that cannot be built is a lost
// feature, and refusing to start would turn a typo in an optional integration
// into an outage of the diagnostics the operator actually installed Seed for.
func (s *Server) initAlertDelivery(logger *slog.Logger) *alertdelivery.Notifier {
	receiver := os.Getenv(alertWebhookURLEnv)
	if receiver == "" {
		return nil
	}

	notifier, err := alertdelivery.New(alertdelivery.Config{
		URL:    receiver,
		Secret: os.Getenv(alertWebhookKeyEnv),
		Logger: logger,
	})
	if err != nil {
		logger.Error("alert webhook not configured; alerts will not be delivered",
			"error", err, "url_env", alertWebhookURLEnv, "key_env", alertWebhookKeyEnv)
		return nil
	}

	notifier.Start()
	s.alertDelivery = notifier
	logger.Info("alert webhook configured", "endpoint", notifier.Endpoint())
	return notifier
}
