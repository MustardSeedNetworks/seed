package app

import (
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// NewWiFiVisibility builds the Wi-Fi airspace visibility component, fed decoded
// frames by the capture source (not the DB). It is a producer into the shared,
// server-owned anomaly Coordinator (ADR-0029): the cmd layer builds it before the
// server owns the merged engine, so the server injects the Coordinator later via
// Service.SetCoordinator. Until then it runs airspace-only.
func NewWiFiVisibility() *visibility.Service {
	return visibility.New()
}
