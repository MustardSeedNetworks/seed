package api

// handlers_network.go carries the interface handlers: enumerating interfaces,
// selecting the current one, and MTU. Link/PHY state is in
// handlers_network_link.go and addressing in handlers_network_ip.go.

import (
	"log/slog"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// SetInterfaceRequest represents a request to change the current interface.
type SetInterfaceRequest struct {
	Interface string `json:"interface"`
}

// SetMTURequest represents the request to set interface MTU.
//
// MTU range 68..65535: 68 is the IPv4 minimum per RFC 791; 65535 is the
// absolute cap (jumbo frames live below 9216 in practice).
type SetMTURequest struct {
	Interface string `json:"interface" validate:"required"`
	MTU       int    `json:"mtu"       validate:"required,gte=68,lte=65535"`
}

// ============================================================================
// Handler Functions
// ============================================================================

// CategorizedInterfacesResponse groups interfaces by type for UI display.
// #756: Interfaces are categorized so WiFi only shows under WiFi, Ethernet under Ethernet.
type CategorizedInterfacesResponse struct {
	Ethernet            []InterfaceInfo `json:"ethernet"`
	WiFi                []InterfaceInfo `json:"wifi"`
	RecommendedEthernet string          `json:"recommendedEthernet,omitempty"`
	RecommendedWiFi     string          `json:"recommendedWifi,omitempty"`
	CurrentInterface    string          `json:"currentInterface"`
	CurrentType         string          `json:"currentType"`
}

// InterfaceInfo is the flat transport view of a network interface, mirroring
// netif.InterfaceInfo's wire shape so the published schema does not depend on
// the netif domain package. Type is a plain string (the domain's InterfaceType
// is a string enum).
type InterfaceInfo struct {
	Name          string   `json:"name"`
	FriendlyName  string   `json:"friendlyName,omitempty"`
	Description   string   `json:"description,omitempty"`
	Type          string   `json:"type"`
	Up            bool     `json:"up"`
	Running       bool     `json:"running"`
	HardwareAddr  string   `json:"hardwareAddr"`
	MTU           int      `json:"mtu"`
	Addresses     []string `json:"addresses"`
	Speed         int64    `json:"speed,omitempty"`
	SpeedDisplay  string   `json:"speedDisplay,omitempty"`
	ChipsetVendor string   `json:"chipsetVendor,omitempty"`
	ChipsetModel  string   `json:"chipsetModel,omitempty"`
	HasTDR        bool     `json:"hasTDR,omitempty"`
	HasDOM        bool     `json:"hasDOM,omitempty"`
	Score         int      `json:"score,omitempty"`
}

// toInterfaceInfos maps physical interfaces onto their flat transport view. It
// always returns a non-nil slice so an empty category serializes as [] not null.
func toInterfaceInfos(ifaces []*netif.InterfaceInfo) []InterfaceInfo {
	out := make([]InterfaceInfo, 0, len(ifaces))
	for _, iface := range ifaces {
		if iface == nil {
			continue
		}
		out = append(out, InterfaceInfo{
			Name:          iface.Name,
			FriendlyName:  iface.FriendlyName,
			Description:   iface.Description,
			Type:          string(iface.Type),
			Up:            iface.Up,
			Running:       iface.Running,
			HardwareAddr:  iface.HardwareAddr,
			MTU:           iface.MTU,
			Addresses:     iface.Addresses,
			Speed:         iface.Speed,
			SpeedDisplay:  iface.SpeedDisplay,
			ChipsetVendor: iface.ChipsetVendor,
			ChipsetModel:  iface.ChipsetModel,
			HasTDR:        iface.HasTDR,
			HasDOM:        iface.HasDOM,
			Score:         iface.Score,
		})
	}
	return out
}

func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.netManager() == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.TWithData(
				"errors.service.notAvailable",
				map[string]any{"service": "Network manager"},
			),
			"",
		) // fixes #694
		return
	}

	if err := s.netManager().RefreshInterfaces(); err != nil {
		logger.ErrorContext(r.Context(), "Failed to refresh interfaces", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.network.refreshFailed"),
			"",
		)
		return
	}

	// #756: Check if categorized response is requested
	if r.URL.Query().Get("categorized") == "true" {
		s.handleCategorizedInterfaces(w, r)
		return
	}

	// Return only physical interfaces (ethernet and wifi) - excludes loopback, docker, veth, etc.
	interfaces := s.netManager().GetPhysicalInterfaces()

	sendJSONResponse(w, nil, http.StatusOK, interfaces)
}

// isBetterInterface returns true if candidate is better than current best.
// Prefers: interfaces that are up, then higher score.
func isBetterInterface(candidate, best *netif.InterfaceInfo) bool {
	if best == nil {
		return true
	}
	if candidate.Up && !best.Up {
		return true
	}
	return candidate.Up && best.Up && candidate.Score > best.Score
}

// handleCategorizedInterfaces returns interfaces grouped by type (ethernet vs WiFi).
// #756: Helps UI show ethernet interfaces under Ethernet dropdown, WiFi under WiFi dropdown.
func (s *Server) handleCategorizedInterfaces(w http.ResponseWriter, _ *http.Request) {
	interfaces := s.netManager().GetPhysicalInterfaces()

	resp := CategorizedInterfacesResponse{
		CurrentInterface: s.netManager().GetCurrentInterface(),
	}

	// Categorize interfaces and find best in each category
	var ethernet, wifi []*netif.InterfaceInfo
	var bestEthernet, bestWiFi *netif.InterfaceInfo

	for _, iface := range interfaces {
		switch iface.Type {
		case netif.InterfaceTypeEthernet:
			ethernet = append(ethernet, iface)
			if isBetterInterface(iface, bestEthernet) {
				bestEthernet = iface
			}
		case netif.InterfaceTypeWiFi:
			wifi = append(wifi, iface)
			if isBetterInterface(iface, bestWiFi) {
				bestWiFi = iface
			}
		case netif.InterfaceTypeLoopback, netif.InterfaceTypeVirtual, netif.InterfaceTypeOther:
			// Skip non-physical interfaces for categorization
			continue
		}
	}

	resp.Ethernet = toInterfaceInfos(ethernet)
	resp.WiFi = toInterfaceInfos(wifi)

	// Set recommended interfaces
	if bestEthernet != nil {
		resp.RecommendedEthernet = bestEthernet.Name
	}
	if bestWiFi != nil {
		resp.RecommendedWiFi = bestWiFi.Name
	}

	// Determine current interface type
	if currentInfo, err := s.netManager().GetInterface(resp.CurrentInterface); err == nil && currentInfo != nil {
		resp.CurrentType = string(currentInfo.Type)
	}

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// handleInterface handles GET/PUT for current interface.
// #756: Interfaces are auto-detected and categorized; user can select from available options.
func (s *Server) handleInterface(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.netManager() == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.TWithData(
				"errors.service.notAvailable",
				map[string]any{"service": "Network manager"},
			),
			"",
		) // fixes #694
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetInterface(w)
	case http.MethodPut:
		s.handlePutInterface(w, r, logger, localizer)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
	}
}

// handleGetInterface returns current interface information.
func (s *Server) handleGetInterface(w http.ResponseWriter) {
	currentIface := s.netManager().GetCurrentInterface()
	ifaceInfo, _ := s.netManager().GetInterface(currentIface)

	// Check if current interface is wireless
	isWireless := false
	if s.wifiManager() != nil && currentIface != "" {
		isWireless = s.wifiManager().IsWireless()
	}

	// #756: Check if the configured interface is still available
	interfaceAvailable := ifaceInfo != nil && ifaceInfo.Up

	resp := map[string]any{
		"interface":  currentIface,
		"isWireless": isWireless,
		"available":  interfaceAvailable,
	}

	// Add interface details if available
	if ifaceInfo != nil {
		resp["type"] = string(ifaceInfo.Type)
		resp["up"] = ifaceInfo.Up
		if ifaceInfo.FriendlyName != "" {
			resp["friendlyName"] = ifaceInfo.FriendlyName
		}
	}

	// #756: If interface is unavailable, suggest alternatives
	if !interfaceAvailable && currentIface != "" {
		resp["warning"] = "Selected interface is no longer available"
		// Find best alternative
		if suggested := s.netManager().FindFirstAvailable(nil); suggested != "" {
			resp["suggestedInterface"] = suggested
		}
	}

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// handlePutInterface sets the current interface.
func (s *Server) handlePutInterface(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) {
	var req SetInterfaceRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	// #756: Validate interface exists and is available
	ifaceInfo, err := s.netManager().GetInterface(req.Interface)
	if err != nil {
		logger.WarnContext(r.Context(), "Invalid interface", "error", err, "interface", req.Interface)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.network.invalidInterface"),
			"",
		)
		return
	}

	// Warn if interface is down but allow selection (user may be preparing)
	warning := ""
	if !ifaceInfo.Up {
		warning = "Selected interface is currently down"
	}

	if setErr := s.netManager().SetCurrentInterface(req.Interface); setErr != nil {
		logger.WarnContext(r.Context(), "Failed to set interface", "error", setErr, "interface", req.Interface)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.network.invalidInterface"),
			"",
		)
		return
	}

	// Update unified discovery service to use new interface (handles protocol restarts)
	if s.discoveryService() != nil {
		if discErr := s.discoveryService().SetInterface(req.Interface); discErr != nil {
			// Log but don't fail - discovery may not work without root
			logging.GetLogger().WarnContext(r.Context(), "Failed to set discovery interface", "error", discErr)
		}
	}

	// Update WiFi manager interface and check if wireless
	if s.wifiManager() != nil {
		s.wifiManager().SetInterface(req.Interface)
	}

	// Update link monitor interface
	if s.linkMonitor() != nil {
		s.linkMonitor().SetInterface(req.Interface)
	}

	// Check if new interface is wireless
	isWireless := false
	if s.wifiManager() != nil {
		isWireless = s.wifiManager().IsWireless()
	}

	resp := map[string]any{
		"status":     "ok",
		"interface":  req.Interface,
		"isWireless": isWireless,
		"type":       string(ifaceInfo.Type),
	}
	if warning != "" {
		resp["warning"] = warning
	}

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// handleSetMTU handles POST requests to set interface MTU.
func (s *Server) handleSetMTU(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req SetMTURequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	// Validate MTU value
	if err := validation.ValidateMTU(req.MTU); err != nil {
		logger.WarnContext(r.Context(), "Invalid MTU value", "error", err, "mtu", req.MTU)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.mtu.invalidRange"),
			"",
		)
		return
	}

	// Set the MTU (the use-case resolves the current interface when unset and
	// refreshes best-effort).
	iface, err := s.networkIP.SetMTU(req.Interface, req.MTU)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to set MTU", "error", err, "interface", iface, "mtu", req.MTU)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.api.internalError"),
			"",
		)
		return
	}

	sendJSONResponse(w, nil, http.StatusOK, map[string]any{
		"status":    statusSuccess,
		"message":   "MTU updated",
		"interface": iface,
		"mtu":       req.MTU,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// getInterfaceFromRequest extracts the interface name from request query params.
// Falls back to the netManager's current interface if not specified.
// Validates the interface name to prevent injection attacks.
func (s *Server) getInterfaceFromRequest(r *http.Request) string {
	if iface := r.URL.Query().Get("interface"); iface != "" {
		// Validate interface name to prevent path traversal/injection
		if err := validation.ValidateInterface(iface); err != nil {
			logging.GetLogger().
				WarnContext(r.Context(), "Invalid interface name in request", "interface", iface, "error", err)
			// Fall back to current interface instead of returning invalid input
			if s.netManager() != nil {
				return s.netManager().GetCurrentInterface()
			}
			return ""
		}
		return iface
	}
	if s.netManager() != nil {
		return s.netManager().GetCurrentInterface()
	}
	return ""
}
