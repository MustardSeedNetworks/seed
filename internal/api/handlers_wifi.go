package api

// handlers_wifi.go contains WiFi management and scanning handlers.
// Split from handlers_network.go for code organization (Plan F).

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

// ============================================================================
// WiFi Types
// ============================================================================

// WiFiResponse represents the Wi-Fi information for the API.
type WiFiResponse struct {
	Interface string `json:"interface"` // WiFi interface used
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	Signal    int    `json:"signal"` // dBm
	Channel   int    `json:"channel"`
	Frequency int    `json:"frequency"` // MHz
	Security  string `json:"security"`
}

// WiFiSettingsResponse represents the WiFi configuration settings.
type WiFiSettingsResponse struct {
	Interface     string   `json:"interface"`
	AvailableWiFi []string `json:"availableWifi"`
	IsWireless    bool     `json:"isWireless"`
}

// ============================================================================
// WiFi Settings Handlers
// ============================================================================

// handleWiFiSettings handles GET/PUT for WiFi settings.
func (s *Server) handleWiFiSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		s.getWiFiSettings(w, r)
	case http.MethodPut:
		s.updateWiFiSettings(w, r, logger, localizer)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
	}
}

func (s *Server) getWiFiSettings(w http.ResponseWriter, _ *http.Request) {
	res := s.wifiManagement.Settings()
	sendJSONResponse(w, nil, http.StatusOK, WiFiSettingsResponse{
		Interface:     res.Interface,
		AvailableWiFi: res.AvailableWiFi,
		IsWireless:    res.IsWireless,
	})
}

func (s *Server) updateWiFiSettings(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) {
	var req struct {
		Interface string `json:"interface"`
	}
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	if err := s.wifiManagement.UpdateInterface(req.Interface); err != nil {
		logger.ErrorContext(r.Context(), "Failed to save config", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.config.failedToSave"),
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status":  statusSuccess,
		"message": "WiFi settings updated",
	})
}

// ============================================================================
// WiFi Info Handlers
// ============================================================================

// handleWiFi returns Wi-Fi information for the current interface.
func (s *Server) handleWiFi(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	if s.wifiManager() == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.TWithData(
				"errors.service.notAvailable",
				map[string]any{"service": "Wi-Fi manager"},
			),
			"",
		)
		return
	}

	// Get interface from query param or use current/default
	wlanIface := s.resolveWiFiInterface(r)

	// Update WiFi manager to use the requested interface
	s.wifiManager().SetInterface(wlanIface)

	// Check if interface is wireless
	if !s.wifiManager().IsWireless() {
		sendJSONResponse(w, nil, http.StatusOK, map[string]any{
			"interface": wlanIface,
			"wireless":  false,
			"message":   "Current interface is not a wireless adapter",
		})
		return
	}

	info := s.wifiManager().GetInfo()
	if info == nil {
		w.Header().Set("Content-Type", "application/json")
		sendJSONResponse(w, nil, http.StatusOK, map[string]any{
			"interface": wlanIface,
			"wireless":  true,
			"connected": false,
			"message":   "Not connected to a wireless network",
		})
		return
	}

	resp := WiFiResponse{
		Interface: wlanIface,
		SSID:      info.SSID,
		BSSID:     info.BSSID,
		Signal:    info.Signal,
		Channel:   info.Channel,
		Frequency: info.Frequency,
		Security:  info.Security,
	}

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// ============================================================================
// WiFi Scan Handlers
// ============================================================================

// handleWiFiScan performs a WiFi network scan and returns discovered networks.
// Thin handler (ADR-0016): it resolves the requested interface from the request
// and delegates to the Wi-Fi management use-case, which owns the degrade logic.
func (s *Server) handleWiFiScan(w http.ResponseWriter, r *http.Request) {
	res := s.wifiManagement.Scan(s.getInterfaceFromRequest(r))

	resp := map[string]any{
		"interface": res.Interface,
		"available": res.Available,
		"networks":  res.Networks,
	}
	if res.Error != "" {
		resp["error"] = res.Error
	}
	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// handleWiFiStatus returns the WiFi adapter status without performing a scan.
// Thin handler (ADR-0016) delegating to the Wi-Fi management use-case.
func (s *Server) handleWiFiStatus(w http.ResponseWriter, r *http.Request) {
	res := s.wifiManagement.Status(s.getInterfaceFromRequest(r))

	sendJSONResponse(w, nil, http.StatusOK, map[string]any{
		"status":            res.Status,
		"message":           res.Message,
		"currentInterface":  res.CurrentInterface,
		"isWireless":        res.IsWireless,
		"availableAdapters": res.AvailableAdapters,
		"canScan":           res.CanScan,
	})
}

// ============================================================================
// WiFi Channel Graph Handler
// ============================================================================

// ============================================================================
// WiFi Connection Handlers
// ============================================================================

// WiFiConnectRequest represents a request to connect to a WiFi network.
type WiFiConnectRequest struct {
	SSID     string `json:"ssid"               validate:"required,min=1,max=32"` // 802.11 SSID max is 32 bytes
	Password string `json:"password,omitempty" validate:"omitempty,min=8"`       // WPA2 minimum is 8
}

// handleWiFiConnect handles WiFi connection requests. Thin handler (ADR-0016):
// decode the request, delegate to the management use-case, map domain errors to
// status codes.
func (s *Server) handleWiFiConnect(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req WiFiConnectRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	if req.SSID == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"SSID is required",
			"",
		)
		return
	}

	result, err := s.wifiManagement.Connect(req.SSID, req.Password)
	switch {
	case errors.Is(err, troubleshooting.ErrRadioUnavailable):
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"WiFi manager not available",
			"",
		)
		return
	case err != nil:
		logger.ErrorContext(r.Context(), "WiFi connection failed", "error", err, "ssid", req.SSID)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Connection failed",
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// handleWiFiDisconnect handles WiFi disconnection requests. Thin handler
// (ADR-0016) delegating to the management use-case.
func (s *Server) handleWiFiDisconnect(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	result, err := s.wifiManagement.Disconnect()
	switch {
	case errors.Is(err, troubleshooting.ErrRadioUnavailable):
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"WiFi manager not available",
			"",
		)
		return
	case err != nil:
		logger.ErrorContext(r.Context(), "WiFi disconnection failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Disconnection failed",
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// handleWiFiSavedNetworks returns a list of saved WiFi networks.
func (s *Server) handleWiFiSavedNetworks(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if s.wifiManager() == nil {
		sendJSONResponse(w, nil, http.StatusOK, map[string]any{
			"networks": []any{},
		})
		return
	}

	networks, err := s.wifiManager().GetSavedNetworks()
	if err != nil {
		logger.WarnContext(r.Context(), "Failed to get saved networks", "error", err)
		sendJSONResponse(w, nil, http.StatusOK, map[string]any{
			"networks": []any{},
			"error":    err.Error(),
		})
		return
	}

	sendJSONResponse(w, nil, http.StatusOK, map[string]any{
		"networks": networks,
	})
}

// handleWiFiForgetNetwork handles requests to forget a saved WiFi network.
func (s *Server) handleWiFiForgetNetwork(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if s.wifiManager() == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"WiFi manager not available",
			"",
		)
		return
	}

	// Get SSID from query parameter
	ssid := r.URL.Query().Get("ssid")
	if ssid == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"SSID is required",
			"",
		)
		return
	}

	if err := s.wifiManager().ForgetNetwork(ssid); err != nil {
		logger.ErrorContext(r.Context(), "Failed to forget network", "error", err, "ssid", ssid)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			err.Error(),
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status":  statusSuccess,
		"message": "Network forgotten",
	})
}

// handleWiFiChannelGraph returns channel overlap graph data for WiFi visualization.
// It scans available networks and organizes them by frequency band with channel overlap information.
func (s *Server) handleWiFiChannelGraph(w http.ResponseWriter, r *http.Request) {
	// Get interface from query param or use current/default
	wlanIface := s.resolveWiFiInterface(r)

	if s.wifiScanner() == nil {
		sendJSONResponse(w, nil, http.StatusOK, map[string]any{
			"interface": wlanIface,
			"available": false,
			"error":     "WiFi scanner not initialized",
			"data":      nil,
		})
		return
	}

	// Check if interface is wireless
	if s.wifiManager() == nil || !s.wifiManager().IsWireless() {
		sendJSONResponse(w, nil, http.StatusOK, map[string]any{
			"interface": wlanIface,
			"available": false,
			"error":     "No wireless adapter available. Connect a WiFi adapter to scan networks.",
			"data":      nil,
		})
		return
	}

	// Perform scan
	networks, err := s.wifiScanner().Scan()
	if err != nil {
		sendJSONResponse(w, nil, http.StatusOK, map[string]any{
			"interface": wlanIface,
			"available": true,
			"error":     "Wi-Fi scan failed. Check permissions and interface availability.",
			"data":      nil,
		})
		return
	}

	// Get connected network BSSID
	connectedBSSID := ""
	if info := s.wifiManager().GetInfo(); info != nil {
		connectedBSSID = info.BSSID
	}

	// Generate channel graph data
	data := wifi.GetChannelGraphData(networks, connectedBSSID)

	sendJSONResponse(w, nil, http.StatusOK, map[string]any{
		"interface": wlanIface,
		"available": true,
		"data":      data,
	})
}

// ============================================================================
// Enhanced WiFi Discovery Handlers (using WiFiBridge)
// ============================================================================

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
