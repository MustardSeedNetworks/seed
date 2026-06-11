package app

import (
	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// NewWiFiVisibility builds the Wi-Fi airspace visibility component. It is fed
// decoded frames by the capture source (not the DB), but it persists its
// detected-anomaly stream to the unified anomaly store (ADR-0021) when one is
// supplied — pass db.Anomalies() so Wi-Fi anomalies survive restart and feed the
// cross-source platform. A nil store leaves the engine purely in-memory. It
// errors only if the Wi-Fi anomaly catalog is malformed — a build-time
// programming error.
func NewWiFiVisibility(store anomaly.Store) (*visibility.Service, error) {
	return visibility.New(visibility.WithStore(store))
}
