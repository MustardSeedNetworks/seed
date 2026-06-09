package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// WiFiAirspaceResponse is the Pro-gated airspace read model: the cross-referenced
// SSID → AP → BSSID → client tree plus a capture/entity-count status summary.
type WiFiAirspaceResponse struct {
	SSIDs  []airspace.SSIDGroup `json:"ssids"`
	Status visibility.Status    `json:"status"`
}

// WiFiAnomaliesResponse is the Pro-gated Wi-Fi anomaly stream: the deduped,
// severity-ranked detections plus the same status summary.
type WiFiAnomaliesResponse struct {
	Anomalies []anomaly.Anomaly `json:"anomalies"`
	Status    visibility.Status `json:"status"`
}

// wifiVisibility returns the live Wi-Fi visibility component, or nil when no
// capture component is wired (e.g. the test harness). The composition root
// (app.NewWiFiQueries) maps a nil component to a genuinely-absent use-case
// source, so the read use-case degrades to empty-but-valid results.
func (s *Server) wifiVisibility() *visibility.Service {
	if s.background == nil {
		return nil
	}
	return s.background.WiFiVisibility
}

// handleWiFiAirspace serves GET /api/v1/wifi/airspace. Thin handler (ADR-0020):
// it delegates to the Wi-Fi visibility read use-case and encodes the result. The
// route is feature-gated (wifi_management_capture); an absent capture component
// yields an empty tree with captureActive=false, not an error.
func (s *Server) handleWiFiAirspace(w http.ResponseWriter, _ *http.Request) {
	res := s.wifiQueries.Airspace()
	sendJSONResponse(w, nil, http.StatusOK, WiFiAirspaceResponse{SSIDs: res.SSIDs, Status: res.Status})
}

// handleWiFiAnomalies serves GET /api/v1/wifi/anomalies. Thin handler delegating
// to the use-case; feature-gated (wifi_association_forensics); degrades to an
// empty stream when no capture component is present.
func (s *Server) handleWiFiAnomalies(w http.ResponseWriter, _ *http.Request) {
	res := s.wifiQueries.Anomalies()
	sendJSONResponse(w, nil, http.StatusOK, WiFiAnomaliesResponse{Anomalies: res.Anomalies, Status: res.Status})
}
