package api

import (
	"net/http"

	"github.com/krisarmstrong/seed/internal/anomaly"
	"github.com/krisarmstrong/seed/internal/wifi/airspace"
	"github.com/krisarmstrong/seed/internal/wifi/visibility"
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

// wifiVisibility returns the live visibility service, or nil when no capture
// component is wired (e.g. no monitor-capable interface). Handlers degrade to an
// empty-but-valid response in that case rather than erroring.
func (s *Server) wifiVisibility() *visibility.Service {
	if s.background == nil {
		return nil
	}
	return s.background.WiFiVisibility
}

// handleWiFiAirspace serves GET /api/v1/wifi/airspace — the airspace tree. The
// route is feature-gated (wifi_management_capture); an absent capture component
// yields an empty tree with captureActive=false, not an error.
func (s *Server) handleWiFiAirspace(w http.ResponseWriter, _ *http.Request) {
	resp := WiFiAirspaceResponse{SSIDs: []airspace.SSIDGroup{}}
	if svc := s.wifiVisibility(); svc != nil {
		resp.SSIDs = svc.Tree()
		resp.Status = svc.Status()
	}
	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// handleWiFiAnomalies serves GET /api/v1/wifi/anomalies — the Wi-Fi anomaly
// stream. Feature-gated (wifi_association_forensics); degrades to an empty
// stream when no capture component is present.
func (s *Server) handleWiFiAnomalies(w http.ResponseWriter, _ *http.Request) {
	resp := WiFiAnomaliesResponse{Anomalies: []anomaly.Anomaly{}}
	if svc := s.wifiVisibility(); svc != nil {
		resp.Anomalies = svc.Anomalies()
		resp.Status = svc.Status()
	}
	sendJSONResponse(w, nil, http.StatusOK, resp)
}
