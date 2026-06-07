package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	wifiapp "github.com/MustardSeedNetworks/seed/internal/wifi/app"
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

// wifiVisibilitySource adapts the background visibility component to the use-case
// port, returning a nil interface (not a typed-nil) when no capture component is
// wired — so wifiapp.Queries sees a genuinely-absent source.
func wifiVisibilitySource(bg *BackgroundComponents) wifiapp.VisibilitySource {
	if bg == nil || bg.WiFiVisibility == nil {
		return nil
	}
	return bg.WiFiVisibility
}

// handleWiFiAirspace serves GET /api/v1/wifi/airspace. Thin handler (ADR-0016):
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
