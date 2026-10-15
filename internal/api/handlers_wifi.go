package api

// handlers_wifi.go contains WiFi management and scanning handlers.
// Split from handlers_network.go for code organization (Plan F).

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/validation"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

// ============================================================================
// WiFi Types
// ============================================================================

// WiFiResponse represents the Wi-Fi information for the API.
type WiFiResponse struct {
	Interface   string      `json:"interface"`
	Wireless    bool        `json:"wireless"`
	Status      wifi.Status `json:"status"                jsonschema:"enum=associated,enum=detailsWithheld,enum=notAssociated"`
	Connected   *bool       `json:"connected,omitempty"`
	SSID        string      `json:"ssid,omitempty"`
	BSSID       string      `json:"bssid,omitempty"`
	Signal      int         `json:"signal,omitempty"` // dBm
	Channel     int         `json:"channel,omitempty"`
	Frequency   int         `json:"frequency,omitempty"` // MHz
	Security    string      `json:"security,omitempty"`
	Reason      string      `json:"reason,omitempty"`
	Remediation string      `json:"remediation,omitempty"`
}

// A withheld observation cannot establish whether the interface is connected.
func wiFiStateResponse(iface string, observed wifi.Observation) WiFiResponse {
	response := WiFiResponse{
		Interface: iface, Wireless: true, Status: observed.Status,
		Reason: observed.Reason, Remediation: observed.Remediation,
	}
	if observed.Status == wifi.StatusDetailsWithheld {
		return response
	}
	connected := observed.Status == wifi.StatusAssociated
	response.Connected = &connected
	if info := observed.Info; info != nil {
		response.SSID = info.SSID
		response.BSSID = info.BSSID
		response.Signal = info.Signal
		response.Channel = info.Channel
		response.Frequency = info.Frequency
		response.Security = info.Security
	}
	return response
}

// WiFiSettingsResponse represents the WiFi configuration settings.
type WiFiSettingsResponse struct {
	Interface     string   `json:"interface"`
	AvailableWiFi []string `json:"availableWifi"`
	IsWireless    bool     `json:"isWireless"`
}

// WiFiSettingsRequest selects the Wi-Fi interface. An empty interface clears
// the selection, falling back to the default interface.
type WiFiSettingsRequest struct {
	Interface string `json:"interface"`
}

// WiFiScanResponse is a neighbor-AP scan. Error carries the reason the scan
// could not run or complete; Remediation is set only when an operator can act
// on it.
type WiFiScanResponse struct {
	Interface   string                 `json:"interface"`
	Available   bool                   `json:"available"`
	Networks    []*wifi.ScannedNetwork `json:"networks"`
	Remediation string                 `json:"remediation,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// WiFiChannelGraphResponse is the last neighbor scan grouped by band for the
// channel overlap graph. Data is absent when the scan could not run or failed,
// and Error then says why.
type WiFiChannelGraphResponse struct {
	Interface   string                 `json:"interface"`
	Available   bool                   `json:"available"`
	Error       string                 `json:"error,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
	Data        *wifi.ChannelGraphData `json:"data,omitempty"`
}

// WiFiSavedNetworksResponse lists the networks the host has credentials for.
// Error is set when the platform could not list them.
type WiFiSavedNetworksResponse struct {
	Networks []wifi.SavedNetwork `json:"networks"`
	Error    string              `json:"error,omitempty"`
}

// WiFiStatusResponse is the wireless adapter state, read without a scan.
type WiFiStatusResponse struct {
	Status            string   `json:"status"`
	Message           string   `json:"message"`
	CurrentInterface  string   `json:"currentInterface"`
	IsWireless        bool     `json:"isWireless"`
	AvailableAdapters []string `json:"availableAdapters"`
	CanScan           bool     `json:"canScan"`
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
	var req WiFiSettingsRequest
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

	sendJSONResponse(w, logger, http.StatusOK, StatusMessageResponse{
		Status:  statusSuccess,
		Message: "WiFi settings updated",
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

	if !s.wifiManager().IsWireless() {
		resp := wiFiStateResponse(wlanIface, wifi.Observation{Status: wifi.StatusNotAssociated})
		resp.Wireless = false
		sendJSONResponse(w, nil, http.StatusOK, resp)
		return
	}

	sendJSONResponse(w, nil, http.StatusOK, wiFiStateResponse(wlanIface, s.wifiManager().Observe()))
}

// ============================================================================
// WiFi Scan Handlers
// ============================================================================

// handleWiFiScan performs a WiFi network scan and returns discovered networks.
// Thin handler (ADR-0016): it resolves the requested interface from the request
// and delegates to the Wi-Fi management use-case, which owns the degrade logic.
func (s *Server) handleWiFiScan(w http.ResponseWriter, r *http.Request) {
	iface, ok := wifiScanInterface(w, r)
	if !ok {
		return
	}
	res := s.wifiManagement.Scan(iface)
	sendJSONResponse(w, nil, http.StatusOK, WiFiScanResponse{
		Interface:   res.Interface,
		Available:   res.Available,
		Networks:    res.Networks,
		Remediation: res.Remediation,
		Error:       res.Error,
	})
}

// wifiScanInterface returns the radio a scan request names. Without the
// parameter it is empty, which the use-case resolves to the configured Wi-Fi
// interface; getInterfaceFromRequest would fall back to the wired one and
// report it as the scanned radio (#2881). A malformed name is refused with 400.
func wifiScanInterface(w http.ResponseWriter, r *http.Request) (string, bool) {
	iface := r.URL.Query().Get("interface")
	if iface == "" {
		return "", true
	}
	if err := validation.ValidateInterface(iface); err != nil {
		logger := logging.FromContext(r.Context())
		// Log err, not the name, which arrives straight from the query (CWE-117).
		logger.WarnContext(r.Context(), "Invalid interface", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, i18n.FromRequest(r).T("errors.network.invalidInterface"), "")
		return "", false
	}
	return iface, true
}

// handleWiFiStatus returns the WiFi adapter status without performing a scan.
// Thin handler (ADR-0016) delegating to the Wi-Fi management use-case.
func (s *Server) handleWiFiStatus(w http.ResponseWriter, r *http.Request) {
	res := s.wifiManagement.Status(s.getInterfaceFromRequest(r))

	// A host with no adapter lists none rather than sending null.
	adapters := res.AvailableAdapters
	if adapters == nil {
		adapters = []string{}
	}
	sendJSONResponse(w, nil, http.StatusOK, WiFiStatusResponse{
		Status:            res.Status,
		Message:           res.Message,
		CurrentInterface:  res.CurrentInterface,
		IsWireless:        res.IsWireless,
		AvailableAdapters: adapters,
		CanScan:           res.CanScan,
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
		sendJSONResponse(w, nil, http.StatusOK, WiFiSavedNetworksResponse{Networks: []wifi.SavedNetwork{}})
		return
	}

	networks, err := s.wifiManager().GetSavedNetworks()
	if err != nil {
		logger.WarnContext(r.Context(), "Failed to get saved networks", "error", err)
		sendJSONResponse(w, nil, http.StatusOK, WiFiSavedNetworksResponse{
			Networks: []wifi.SavedNetwork{},
			Error:    err.Error(),
		})
		return
	}
	// A host with no saved networks lists none rather than sending null.
	if networks == nil {
		networks = []wifi.SavedNetwork{}
	}
	sendJSONResponse(w, nil, http.StatusOK, WiFiSavedNetworksResponse{Networks: networks})
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

	sendJSONResponse(w, logger, http.StatusOK, StatusMessageResponse{
		Status:  statusSuccess,
		Message: "Network forgotten",
	})
}

// handleWiFiChannelGraph returns channel overlap graph data for WiFi visualization.
// It scans available networks and organizes them by frequency band with channel overlap information.
func (s *Server) handleWiFiChannelGraph(w http.ResponseWriter, r *http.Request) {
	iface, ok := wifiScanInterface(w, r)
	if !ok {
		return
	}
	res := s.wifiManagement.Scan(iface)
	response := WiFiChannelGraphResponse{
		Interface:   res.Interface,
		Available:   res.Available,
		Error:       res.Error,
		Remediation: res.Remediation,
	}
	if res.Available && res.Error == "" {
		connectedBSSID := ""
		if info := s.wifiManager().GetInfo(); info != nil {
			connectedBSSID = info.BSSID
		}
		response.Data = wifi.GetChannelGraphData(res.Networks, connectedBSSID)
	}
	sendJSONResponse(w, nil, http.StatusOK, response)
}

// ============================================================================
// Enhanced WiFi Discovery Handlers (using WiFiBridge)
// ============================================================================
