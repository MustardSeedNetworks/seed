package api

// jobs_wifi_discovery.go registers the Wi-Fi discovery scan as a unified job
// kind (ADR-0005). Thin additive wrapper over the existing ctx-aware bridge
// behind an interface seam — no discovery-internal refactor. The scan is
// synchronous and exposes no progress fraction, so the kind takes no progress
// callback. The legacy /security/wifi/discovery/scan endpoint is unchanged
// (retire at Phase-7).

import (
	"context"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/platform/jobs"
)

// wifiDiscoveryScanJobKind is the registered kind name for a Wi-Fi discovery scan.
const wifiDiscoveryScanJobKind = "wifi-discovery-scan"

// wifiDiscoveryBridge is the slice of *discovery.WiFiBridge behaviour the kind
// needs.
type wifiDiscoveryBridge interface {
	Scan(ctx context.Context) (*discovery.WiFiScanResult, error)
}

// newWiFiDiscoveryScanHandler returns the job Handler for the
// "wifi-discovery-scan" kind. It runs one scan (cancellable via the job context)
// and returns the same WiFiDiscoveryScanResponse the legacy endpoint produces.
func newWiFiDiscoveryScanHandler(newBridge func() wifiDiscoveryBridge) jobs.Handler {
	return func(ctx context.Context, _ any, _ func(float64)) (any, error) {
		bridge := newBridge()
		result, err := bridge.Scan(ctx)
		if err != nil {
			return nil, err
		}
		return toWiFiDiscoveryScanResponse(result), nil
	}
}

// registerWiFiDiscoveryScanKind registers the wifi-discovery-scan kind with an
// injectable bridge factory (the seam that makes the wiring testable without
// hardware).
func (s *Server) registerWiFiDiscoveryScanKind(newBridge func() wifiDiscoveryBridge) {
	if err := s.jobsRunner().Register(
		wifiDiscoveryScanJobKind, newWiFiDiscoveryScanHandler(newBridge),
	); err != nil {
		logging.GetLogger().Error("failed to register wifi-discovery-scan job kind", "error", err)
	}
}
