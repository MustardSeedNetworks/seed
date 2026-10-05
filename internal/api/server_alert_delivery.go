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
	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/escalation"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// useAlertDelivery keeps the delivery Manager app.NewAlertDelivery built, for
// settings writes to re-point, and returns the writer the alert pipelines emit
// through.
func (s *Server) useAlertDelivery(manager *alertdelivery.Manager, writer alertdelivery.Writer) alertdelivery.Writer {
	s.alertDelivery = manager
	return writer
}

// registerAlertEscalator registers the escalator that re-sends an alert nobody
// acknowledged (P-B2). It is built after useAlertDelivery, whose Manager it
// sends through and reads the operator's ladders from.
func (s *Server) registerAlertEscalator(e *escalation.Escalator, err error) {
	if err != nil {
		logging.GetLogger().Warn("alert escalation init failed", "error", err)
		return
	}
	if regErr := s.registerEngineIfLicensed(e); regErr != nil {
		logging.GetLogger().Warn("alert escalation registry registration failed", "error", regErr)
	}
}
