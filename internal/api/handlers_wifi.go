// Package api provides the HTTP/WebSocket server.
// handlers_wifi.go contains WiFi management and scanning handlers.
// Split from handlers_network.go for code organization (Plan F).
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
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
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
	}
}

func (s *Server) getWiFiSettings(w http.ResponseWriter, _ *http.Request) {
	// Get configured WLAN interface (or fall back to current) - IEEE 802.11
	wlanIface := s.config.Interface.WiFi
	if wlanIface == "" {
		wlanIface = s.config.Interface.Default
	}

	// Get list of available wireless interfaces
	availableWLAN := []string{}
	if s.netManager != nil {
		for _, iface := range s.netManager.GetInterfaces() {
			if s.netManager.IsWireless(iface.Name) {
				availableWLAN = append(availableWLAN, iface.Name)
			}
		}
	}

	resp := WiFiSettingsResponse{
		Interface:     wlanIface,
		AvailableWiFi: availableWLAN,
		IsWireless:    s.wifiManager != nil && s.wifiManager.IsWireless(),
	}

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

func (s *Server) updateWiFiSettings(w http.ResponseWriter, r *http.Request, logger *slog.Logger, localizer *i18n.Localizer) {
	// Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req struct {
		Interface string `json:"interface"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
		return
	}

	// Lock config for write access
	// NOTE: Must unlock before Save() - Save() acquires RLock internally
	s.config.Lock()

	// Update WiFi interface in config
	s.config.Interface.WiFi = req.Interface

	// Update WiFi manager to use new interface
	if s.wifiManager != nil && req.Interface != "" {
		s.wifiManager.SetInterface(req.Interface)
	}

	// Unlock before Save() to avoid deadlock - Save() acquires RLock internally
	s.config.Unlock()

	// Save config
	if err := s.config.Save(s.configPath); err != nil {
		logger.Error("Failed to save config", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, localizer.T("errors.config.failedToSave"), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status":  "success",
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
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	if s.wifiManager == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable, ErrCodeServiceUnavail, localizer.TWithData("errors.service.notAvailable", map[string]interface{}{"service": "Wi-Fi manager"}), "")
		return
	}

	// Get interface from query param or use current/default
	wlanIface := s.getInterfaceFromRequest(r)
	if wlanIface == "" {
		wlanIface = s.config.Interface.WiFi
		if wlanIface == "" {
			wlanIface = s.config.Interface.Default
		}
	}

	// Check if interface is wireless
	if !s.wifiManager.IsWireless() {
		sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
			"interface": wlanIface,
			"wireless":  false,
			"message":   "Current interface is not a wireless adapter",
		})
		return
	}

	info := s.wifiManager.GetInfo()
	if info == nil {
		w.Header().Set("Content-Type", "application/json")
		sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
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
func (s *Server) handleWiFiScan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	// Get interface from query param or use current/default
	wlanIface := s.getInterfaceFromRequest(r)
	if wlanIface == "" {
		wlanIface = s.config.Interface.WiFi
		if wlanIface == "" {
			wlanIface = s.config.Interface.Default
		}
	}

	if s.wifiScanner == nil {
		sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
			"interface": wlanIface,
			"available": false,
			"error":     "WiFi scanner not initialized",
			"networks":  []interface{}{},
		})
		return
	}

	// Check if interface is wireless
	if s.wifiManager == nil || !s.wifiManager.IsWireless() {
		sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
			"interface": wlanIface,
			"available": false,
			"error":     "No wireless adapter available. Connect a WiFi adapter to scan networks.",
			"networks":  []interface{}{},
		})
		return
	}

	// Perform scan
	networks, err := s.wifiScanner.Scan()
	if err != nil {
		sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
			"interface": wlanIface,
			"available": true,
			"error":     "Wi-Fi scan failed. Check permissions and interface availability.",
			"networks":  []interface{}{},
		})
		return
	}

	sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
		"interface": wlanIface,
		"available": true,
		"networks":  networks,
	})
}

// handleWiFiStatus returns the WiFi adapter status without performing a scan.
func (s *Server) handleWiFiStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	// Get list of available wireless interfaces
	availableAdapters := []string{}
	if s.netManager != nil {
		for _, iface := range s.netManager.GetInterfaces() {
			if s.netManager.IsWireless(iface.Name) {
				availableAdapters = append(availableAdapters, iface.Name)
			}
		}
	}

	// Get interface from query param or use current/default
	currentInterface := s.getInterfaceFromRequest(r)
	if currentInterface == "" {
		currentInterface = s.config.Interface.WiFi
		if currentInterface == "" {
			currentInterface = s.config.Interface.Default
		}
	}

	// Check if current interface is wireless
	isWireless := false
	if s.wifiManager != nil {
		isWireless = s.wifiManager.IsWireless()
	}

	// Determine status message
	var status, message string
	switch {
	case len(availableAdapters) == 0:
		status = "unavailable"
		message = "No wireless adapter detected. Connect a WiFi adapter to perform surveys."
	case !isWireless:
		status = "available"
		message = "Wireless adapter available but not selected as current interface."
	default:
		status = "ready"
		message = "Wireless adapter ready for scanning."
	}

	sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
		"status":            status,
		"message":           message,
		"currentInterface":  currentInterface,
		"isWireless":        isWireless,
		"availableAdapters": availableAdapters,
		"canScan":           isWireless && len(availableAdapters) > 0,
	})
}

// ============================================================================
// WiFi Channel Graph Handler
// ============================================================================

// handleWiFiChannelGraph returns channel overlap graph data for WiFi visualization.
// It scans available networks and organizes them by frequency band with channel overlap information.
func (s *Server) handleWiFiChannelGraph(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	// Get interface from query param or use current/default
	wlanIface := s.getInterfaceFromRequest(r)
	if wlanIface == "" {
		wlanIface = s.config.Interface.WiFi
		if wlanIface == "" {
			wlanIface = s.config.Interface.Default
		}
	}

	if s.wifiScanner == nil {
		sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
			"interface": wlanIface,
			"available": false,
			"error":     "WiFi scanner not initialized",
			"data":      nil,
		})
		return
	}

	// Check if interface is wireless
	if s.wifiManager == nil || !s.wifiManager.IsWireless() {
		sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
			"interface": wlanIface,
			"available": false,
			"error":     "No wireless adapter available. Connect a WiFi adapter to scan networks.",
			"data":      nil,
		})
		return
	}

	// Perform scan
	networks, err := s.wifiScanner.Scan()
	if err != nil {
		sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
			"interface": wlanIface,
			"available": true,
			"error":     "Wi-Fi scan failed. Check permissions and interface availability.",
			"data":      nil,
		})
		return
	}

	// Get connected network BSSID
	connectedBSSID := ""
	if info := s.wifiManager.GetInfo(); info != nil {
		connectedBSSID = info.BSSID
	}

	// Generate channel graph data
	data := wifi.GetChannelGraphData(networks, connectedBSSID)

	sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
		"interface": wlanIface,
		"available": true,
		"data":      data,
	})
}
