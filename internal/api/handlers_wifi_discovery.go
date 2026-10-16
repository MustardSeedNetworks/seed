package api

// handlers_wifi_discovery.go holds the Wi-Fi discovery handlers: scans
// through the discovery engine and their networks, APs and stats views.

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

// wifiBridge returns the WiFi bridge (nil until initVulnerabilityScanner wires it).
func (s *Server) wifiBridge() *enumerate.WiFiBridge {
	return s.wifiBridgeSvc
}

// WiFiDiscoveryScanResponse contains enhanced WiFi scan results.
type WiFiDiscoveryScanResponse struct {
	Networks    []WiFiNetwork        `json:"networks"`
	APs         []WiFiAccessPoint    `json:"accessPoints"`
	Utilization []ChannelUtilization `json:"channelUtilization"`
	ScanTime    string               `json:"scanTime"`
	Interface   string               `json:"interface"`
}

// WiFiDiscoveryNetworksResponse contains discovered WiFi networks.
type WiFiDiscoveryNetworksResponse struct {
	Networks []WiFiNetwork `json:"networks"`
	Total    int           `json:"total"`
}

// WiFiDiscoveryAPsResponse contains discovered access points.
type WiFiDiscoveryAPsResponse struct {
	AccessPoints []WiFiAccessPoint `json:"accessPoints"`
	Total        int               `json:"total"`
}

// WiFiDiscoveryStatsResponse contains WiFi discovery statistics.
type WiFiDiscoveryStatsResponse struct {
	Stats *WiFiDiscoveryStats `json:"stats"`
}

// WiFiNetwork is the flat transport view of discovery.WiFiNetwork. It and its
// siblings (WiFiAccessPoint, ChannelUtilization, WiFiDiscoveryStats) mirror the
// discovery domain's Wi-Fi types so the published schemas do not depend on the
// discovery package; named string enums (security/authorization/band) collapse
// to string.
type WiFiNetwork struct {
	ID                  string         `json:"id"`
	SSID                string         `json:"ssid"`
	IsHidden            bool           `json:"isHidden"`
	SecurityType        string         `json:"securityType"`
	AuthorizationStatus string         `json:"authorizationStatus"`
	FirstSeen           time.Time      `json:"firstSeen"`
	LastSeen            time.Time      `json:"lastSeen"`
	APCount             int            `json:"apCount,omitempty"`
	BestSignal          int            `json:"bestSignal,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

// WiFiAccessPoint is the flat transport view of discovery.WiFiAccessPoint.
type WiFiAccessPoint struct {
	ID           string         `json:"id"`
	DeviceID     string         `json:"deviceId,omitempty"`
	BSSID        string         `json:"bssid"`
	SSIDID       string         `json:"ssidId,omitempty"`
	SSIDName     string         `json:"ssidName,omitempty"`
	APName       string         `json:"apName,omitempty"`
	Vendor       string         `json:"vendor,omitempty"`
	Channel      int            `json:"channel"`
	ChannelWidth int            `json:"channelWidth"`
	FrequencyMHz int            `json:"frequencyMhz"`
	Band         string         `json:"band"`
	WiFiStandard []string       `json:"wifiStandard,omitempty"`
	SignalDBm    int            `json:"signalDbm"`
	NoiseDBm     int            `json:"noiseDbm,omitempty"`
	ClientCount  int            `json:"clientCount"`
	MaxClients   int            `json:"maxClients,omitempty"`
	IsAuthorized bool           `json:"isAuthorized"`
	FirstSeen    time.Time      `json:"firstSeen"`
	LastSeen     time.Time      `json:"lastSeen"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// ChannelUtilization is the flat transport view of discovery.ChannelUtilization.
type ChannelUtilization struct {
	ID                 string    `json:"id"`
	Channel            int       `json:"channel"`
	Band               string    `json:"band"`
	FrequencyMHz       int       `json:"frequencyMhz"`
	UtilizationPercent float64   `json:"utilizationPercent"`
	NonWiFiPercent     float64   `json:"nonWifiPercent"`
	RetryPercent       float64   `json:"retryPercent"`
	APCount            int       `json:"apCount"`
	ClientCount        int       `json:"clientCount"`
	RecordedAt         time.Time `json:"recordedAt"`
}

// WiFiDiscoveryStats is the flat transport view of discovery.WiFiDiscoveryStats.
type WiFiDiscoveryStats struct {
	TotalNetworks     int            `json:"totalNetworks"`
	HiddenNetworks    int            `json:"hiddenNetworks"`
	TotalAPs          int            `json:"totalAps"`
	AuthorizedAPs     int            `json:"authorizedAps"`
	UnauthorizedAPs   int            `json:"unauthorizedAps"`
	TotalClients      int            `json:"totalClients"`
	ChannelsByBand    map[string]int `json:"channelsByBand"`
	SecurityBreakdown map[string]int `json:"securityBreakdown"`
	VendorBreakdown   map[string]int `json:"vendorBreakdown"`
	LastScanTime      time.Time      `json:"lastScanTime"`
}

// toWiFiNetworks maps discovered Wi-Fi networks onto their flat transport view,
// always returning a non-nil slice so an empty result serializes as [].
func toWiFiNetworks(networks []discovery.WiFiNetwork) []WiFiNetwork {
	out := make([]WiFiNetwork, 0, len(networks))
	for _, n := range networks {
		out = append(out, WiFiNetwork{
			ID:                  n.ID,
			SSID:                n.SSID,
			IsHidden:            n.IsHidden,
			SecurityType:        string(n.SecurityType),
			AuthorizationStatus: string(n.AuthorizationStatus),
			FirstSeen:           n.FirstSeen,
			LastSeen:            n.LastSeen,
			APCount:             n.APCount,
			BestSignal:          n.BestSignal,
			Metadata:            n.Metadata,
		})
	}
	return out
}

// toWiFiAccessPoints maps discovered access points onto their flat transport
// view, always returning a non-nil slice.
func toWiFiAccessPoints(aps []discovery.WiFiAccessPoint) []WiFiAccessPoint {
	out := make([]WiFiAccessPoint, 0, len(aps))
	for _, ap := range aps {
		out = append(out, WiFiAccessPoint{
			ID:           ap.ID,
			DeviceID:     ap.DeviceID,
			BSSID:        ap.BSSID,
			SSIDID:       ap.SSIDID,
			SSIDName:     ap.SSIDName,
			APName:       ap.APName,
			Vendor:       ap.Vendor,
			Channel:      ap.Channel,
			ChannelWidth: ap.ChannelWidth,
			FrequencyMHz: ap.FrequencyMHz,
			Band:         string(ap.Band),
			WiFiStandard: ap.WiFiStandard,
			SignalDBm:    ap.SignalDBm,
			NoiseDBm:     ap.NoiseDBm,
			ClientCount:  ap.ClientCount,
			MaxClients:   ap.MaxClients,
			IsAuthorized: ap.IsAuthorized,
			FirstSeen:    ap.FirstSeen,
			LastSeen:     ap.LastSeen,
			Metadata:     ap.Metadata,
		})
	}
	return out
}

// toChannelUtilizations maps channel-utilization records onto their flat
// transport view, always returning a non-nil slice.
func toChannelUtilizations(utils []discovery.ChannelUtilization) []ChannelUtilization {
	out := make([]ChannelUtilization, 0, len(utils))
	for _, u := range utils {
		out = append(out, ChannelUtilization{
			ID:                 u.ID,
			Channel:            u.Channel,
			Band:               string(u.Band),
			FrequencyMHz:       u.FrequencyMHz,
			UtilizationPercent: u.UtilizationPercent,
			NonWiFiPercent:     u.NonWiFiPercent,
			RetryPercent:       u.RetryPercent,
			APCount:            u.APCount,
			ClientCount:        u.ClientCount,
			RecordedAt:         u.RecordedAt,
		})
	}
	return out
}

// toWiFiDiscoveryStats maps Wi-Fi discovery statistics onto their flat
// transport view, preserving nil so an absent stats block stays omitted.
func toWiFiDiscoveryStats(stats *discovery.WiFiDiscoveryStats) *WiFiDiscoveryStats {
	if stats == nil {
		return nil
	}
	return &WiFiDiscoveryStats{
		TotalNetworks:     stats.TotalNetworks,
		HiddenNetworks:    stats.HiddenNetworks,
		TotalAPs:          stats.TotalAPs,
		AuthorizedAPs:     stats.AuthorizedAPs,
		UnauthorizedAPs:   stats.UnauthorizedAPs,
		TotalClients:      stats.TotalClients,
		ChannelsByBand:    stats.ChannelsByBand,
		SecurityBreakdown: stats.SecurityBreakdown,
		VendorBreakdown:   stats.VendorBreakdown,
		LastScanTime:      stats.LastScanTime,
	}
}

// handleWiFiDiscoveryScan performs an enhanced WiFi scan using the WiFiBridge.
//
// POST /api/v1/security/wifi/discovery/scan
//
// Triggers a WiFi scan with enhanced metadata including vendor lookup,
// authorization status, and channel utilization.
//
// Response: 200 OK with WiFiDiscoveryScanResponse.
func (s *Server) handleWiFiDiscoveryScan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	result, err := s.wifiDiscovery.Scan(r.Context())
	switch {
	case errors.Is(err, troubleshooting.ErrDiscoveryUnavailable):
		s.respondWiFiDiscoveryUnavailable(w, logger)
		return
	case err != nil:
		logger.ErrorContext(r.Context(), "WiFi discovery scan failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"WiFi discovery scan failed: "+err.Error(),
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, toWiFiDiscoveryScanResponse(result))
}

// respondWiFiDiscoveryUnavailable emits the shared 503 for the enhanced Wi-Fi
// discovery handlers when no bridge is wired.
func (s *Server) respondWiFiDiscoveryUnavailable(w http.ResponseWriter, logger *slog.Logger) {
	sendErrorResponseWithDetails(
		w,
		logger,
		http.StatusServiceUnavailable,
		ErrCodeServiceUnavail,
		"WiFi discovery bridge not available",
		"",
	)
}

// toWiFiDiscoveryScanResponse maps a Wi-Fi scan result to the API wire shape.
// Shared by the scan handler and the wifi-discovery-scan job kind so both paths
// produce an identical response.
func toWiFiDiscoveryScanResponse(result *discovery.WiFiScanResult) WiFiDiscoveryScanResponse {
	return WiFiDiscoveryScanResponse{
		Networks:    toWiFiNetworks(result.Networks),
		APs:         toWiFiAccessPoints(result.APs),
		Utilization: toChannelUtilizations(result.Utilization),
		ScanTime:    result.ScanTime.Format("2006-01-02T15:04:05Z07:00"),
		Interface:   result.Interface,
	}
}

// handleWiFiDiscoveryNetworks returns discovered WiFi networks.
//
// GET /api/v1/security/wifi/discovery/networks
//
// Returns the list of WiFi networks from the most recent enhanced scan.
//
// Response: 200 OK with WiFiDiscoveryNetworksResponse.
func (s *Server) handleWiFiDiscoveryNetworks(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	networks, err := s.wifiDiscovery.Networks()
	if errors.Is(err, troubleshooting.ErrDiscoveryUnavailable) {
		s.respondWiFiDiscoveryUnavailable(w, logger)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, WiFiDiscoveryNetworksResponse{
		Networks: toWiFiNetworks(networks),
		Total:    len(networks),
	})
}

// handleWiFiDiscoveryAPs returns discovered WiFi access points.
//
// GET /api/v1/security/wifi/discovery/aps
//
// Returns the list of WiFi access points with extended metadata.
//
// Response: 200 OK with WiFiDiscoveryAPsResponse.
func (s *Server) handleWiFiDiscoveryAPs(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	aps, err := s.wifiDiscovery.AccessPoints()
	if errors.Is(err, troubleshooting.ErrDiscoveryUnavailable) {
		s.respondWiFiDiscoveryUnavailable(w, logger)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, WiFiDiscoveryAPsResponse{
		AccessPoints: toWiFiAccessPoints(aps),
		Total:        len(aps),
	})
}

// handleWiFiDiscoveryStats returns WiFi discovery statistics.
//
// GET /api/v1/security/wifi/discovery/stats
//
// Returns aggregated statistics from WiFi discovery.
//
// Response: 200 OK with WiFiDiscoveryStatsResponse.
func (s *Server) handleWiFiDiscoveryStats(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	stats, err := s.wifiDiscovery.Stats()
	if errors.Is(err, troubleshooting.ErrDiscoveryUnavailable) {
		s.respondWiFiDiscoveryUnavailable(w, logger)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, WiFiDiscoveryStatsResponse{
		Stats: toWiFiDiscoveryStats(stats),
	})
}
