package app

import "github.com/krisarmstrong/seed/internal/wifi/visibility"

// NewWiFiVisibility builds the Wi-Fi airspace visibility component. It owns no
// persistence (it is fed decoded frames by the capture source, not the DB), so
// unlike NewReporting it takes no store adapters. It errors only if the Wi-Fi
// anomaly catalog is malformed — a build-time programming error.
func NewWiFiVisibility() (*visibility.Service, error) {
	return visibility.New()
}
