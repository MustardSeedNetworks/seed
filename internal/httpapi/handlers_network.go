package httpapi

// handlers_network.go contains core network interface and IP configuration handlers.
// Split from original handlers_network.go for code organization (Plan F).
// Related handlers moved to: handlers_wifi.go, handlers_vlan.go, handlers_cable.go

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/phy"
	"github.com/krisarmstrong/seed/internal/validation"
)

// IP configuration mode constants.
const (
	ipModeDHCP   = "dhcp"
	ipModeStatic = "static"
)

// Parsing and validation constants.
const (
	// decimalBase is the base for decimal number parsing (0-9 digits).
	decimalBase = 10

	// minUniqueLocalAddrLen is the minimum length for IPv6 unique local address prefix check.
	minUniqueLocalAddrLen = 2
)

// ============================================================================
// Request/Response Types
// ============================================================================

// SetInterfaceRequest represents a request to change the current interface.
type SetInterfaceRequest struct {
	Interface string `json:"interface"`
}

// LinkHistoryEvent represents a link state change event for the API.
type LinkHistoryEvent struct {
	State     string `json:"state"`
	Timestamp string `json:"timestamp"`
}

// LinkResponse represents the link status for an interface.
type LinkResponse struct {
	Interface    string             `json:"interface"`
	LinkUp       bool               `json:"linkUp"`  // Deprecated: use Carrier && HasIP for accurate status
	Carrier      bool               `json:"carrier"` // Physical link/carrier detected (Layer 2)
	HasIP        bool               `json:"hasIP"`   // Has routable IP address (Layer 3)
	Speed        string             `json:"speed"`
	Duplex       string             `json:"duplex"`
	Advertised   []string           `json:"advertisedSpeeds"`
	MTU          int                `json:"mtu"`
	AutoNeg      bool               `json:"autoNeg"`
	FlapCount24h int                `json:"flapCount24h"`       // Link flap count in last 24 hours
	History      []LinkHistoryEvent `json:"history,omitempty"`  // Recent link state changes
	UptimeMs     int64              `json:"uptimeMs,omitempty"` // Monitor uptime in milliseconds
	PoE          *PoEInfo           `json:"poe,omitempty"`      // Power over Ethernet status
	SFP          *SFPInfo           `json:"sfp,omitempty"`      // SFP module and DDM info
}

// PoEInfo represents Power over Ethernet status.
type PoEInfo struct {
	Detected bool    `json:"detected"`
	Standard string  `json:"standard,omitempty"` // 802.3af, 802.3at, 802.3bt
	Class    int     `json:"class,omitempty"`
	PowerMw  float64 `json:"powerMw,omitempty"`
	Voltage  float64 `json:"voltage,omitempty"`
}

// SFPInfo represents SFP module information and DDM.
type SFPInfo struct {
	Present    bool        `json:"present"`
	Vendor     string      `json:"vendor,omitempty"`
	PartNumber string      `json:"partNumber,omitempty"`
	Serial     string      `json:"serial,omitempty"`
	Type       string      `json:"type,omitempty"`       // SR, LR, ER
	Wavelength int         `json:"wavelength,omitempty"` // nm
	Distance   int         `json:"distance,omitempty"`   // meters
	Connector  string      `json:"connector,omitempty"`  // LC, SC
	DDMSupport bool        `json:"ddmSupport"`
	DDM        *SFPDDMInfo `json:"ddm,omitempty"`
}

// SFPDDMInfo contains DDM readings from SFP module.
type SFPDDMInfo struct {
	Temperature float64  `json:"temperature"` // Celsius
	Voltage     float64  `json:"voltage"`     // Volts
	TxPowerDbm  float64  `json:"txPowerDbm"`
	TxPowerMw   float64  `json:"txPowerMw"`
	RxPowerDbm  float64  `json:"rxPowerDbm"`
	RxPowerMw   float64  `json:"rxPowerMw"`
	LaserBiasMa float64  `json:"laserBiasMa"`
	Alarms      []string `json:"alarms,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

// IPv4Info represents IPv4 address configuration.
type IPv4Info struct {
	Address    string `json:"address"`
	Subnet     string `json:"subnet"`
	Gateway    string `json:"gateway,omitempty"`
	DHCPServer string `json:"dhcpServer,omitempty"`
	LeaseTime  int    `json:"leaseTime,omitempty"`
}

// IPv6Info represents an IPv6 address configuration.
type IPv6Info struct {
	Address string `json:"address"`
	Prefix  int    `json:"prefix"`
	Scope   string `json:"scope"`  // global, link-local, unique-local
	Source  string `json:"source"` // slaac, dhcpv6, static, temporary
}

// DHCPTimingInfo represents DHCP transaction timing.
type DHCPTimingInfo struct {
	Discover int64 `json:"discover"` // ms
	Offer    int64 `json:"offer"`
	Request  int64 `json:"request"`
	Ack      int64 `json:"ack"`
	Total    int64 `json:"total"`
}

// IPConfigResponse represents the full IP configuration.
type IPConfigResponse struct {
	Interface string          `json:"interface"`
	MAC       string          `json:"mac"`
	Mode      string          `json:"mode"` // dhcp, static, auto
	IPv4      *IPv4Info       `json:"ipv4,omitempty"`
	IPv6      []IPv6Info      `json:"ipv6"`
	DNS       []string        `json:"dns"`
	Timing    *DHCPTimingInfo `json:"timing,omitempty"`
}

// ipAddrInfo holds parsed IP address information.
type ipAddrInfo struct {
	isIPv4  bool
	address string
	subnet  string
	prefix  int
	scope   string
	source  string
}

// IPSettingsRequest represents a request to change IP configuration.
type IPSettingsRequest struct {
	Mode    string   `json:"mode"`    // "dhcp" or "static"
	Address string   `json:"address"` // IP address (static mode)
	Netmask string   `json:"netmask"` // Subnet mask (static mode)
	Gateway string   `json:"gateway"` // Gateway (static mode, optional)
	DNS     []string `json:"dns"`     // DNS servers (static mode, optional)
}

// IPSettingsResponse represents the current IP configuration settings.
type IPSettingsResponse struct {
	Mode    string   `json:"mode"`
	Address string   `json:"address,omitempty"`
	Netmask string   `json:"netmask,omitempty"`
	Gateway string   `json:"gateway,omitempty"`
	DNS     []string `json:"dns,omitempty"`
}

// SetMTURequest represents the request to set interface MTU.
type SetMTURequest struct {
	Interface string `json:"interface"`
	MTU       int    `json:"mtu"`
}

// ============================================================================
// Handler Functions
// ============================================================================

// CategorizedInterfacesResponse groups interfaces by type for UI display.
// #756: Interfaces are categorized so WiFi only shows under WiFi, Ethernet under Ethernet.
type CategorizedInterfacesResponse struct {
	Ethernet            []*network.InterfaceInfo `json:"ethernet"`
	WiFi                []*network.InterfaceInfo `json:"wifi"`
	RecommendedEthernet string                   `json:"recommendedEthernet,omitempty"`
	RecommendedWiFi     string                   `json:"recommendedWifi,omitempty"`
	CurrentInterface    string                   `json:"currentInterface"`
	CurrentType         string                   `json:"currentType"`
}

func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
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
		) // fixes #694
		return
	}

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
		logger.Error("Failed to refresh interfaces", "error", err)
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

// handleCategorizedInterfaces returns interfaces grouped by type (ethernet vs WiFi).
// #756: Helps UI show ethernet interfaces under Ethernet dropdown, WiFi under WiFi dropdown.
func (s *Server) handleCategorizedInterfaces(w http.ResponseWriter, _ *http.Request) {
	interfaces := s.netManager().GetPhysicalInterfaces()

	resp := CategorizedInterfacesResponse{
		Ethernet:         make([]*network.InterfaceInfo, 0),
		WiFi:             make([]*network.InterfaceInfo, 0),
		CurrentInterface: s.netManager().GetCurrentInterface(),
	}

	// Categorize interfaces and find best in each category
	var bestEthernet, bestWiFi *network.InterfaceInfo

	for _, iface := range interfaces {
		switch iface.Type {
		case network.InterfaceTypeEthernet:
			resp.Ethernet = append(resp.Ethernet, iface)
			// Track best ethernet: prefer up with IP, highest score
			if bestEthernet == nil ||
				(iface.Up && !bestEthernet.Up) ||
				(iface.Up && bestEthernet.Up && iface.Score > bestEthernet.Score) {
				bestEthernet = iface
			}
		case network.InterfaceTypeWiFi:
			resp.WiFi = append(resp.WiFi, iface)
			// Track best WiFi: prefer up with IP, highest score
			if bestWiFi == nil ||
				(iface.Up && !bestWiFi.Up) ||
				(iface.Up && bestWiFi.Up && iface.Score > bestWiFi.Score) {
				bestWiFi = iface
			}
		}
	}

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

	case http.MethodPut:
		// Limit request body size to prevent DoS attacks (fixes #693)
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

		var req SetInterfaceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("Invalid request body", "error", err)
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusBadRequest,
				ErrCodeBadRequest,
				localizer.T("errors.api.invalidRequestBody"),
				"",
			)
			return
		}

		// #756: Validate interface exists and is available
		ifaceInfo, err := s.netManager().GetInterface(req.Interface)
		if err != nil {
			logger.Warn("Invalid interface", "error", err, "interface", req.Interface)
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

		if err := s.netManager().SetCurrentInterface(req.Interface); err != nil {
			logger.Warn("Failed to set interface", "error", err, "interface", req.Interface)
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
			if err := s.discoveryService().SetInterface(req.Interface); err != nil {
				// Log but don't fail - discovery may not work without root
				logging.GetLogger().Warn("Failed to set discovery interface", "error", err)
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

// addLinkHistory adds link flap history from monitor to response.
func (s *Server) addLinkHistory(resp *LinkResponse) {
	if s.linkMonitor() == nil {
		return
	}
	resp.FlapCount24h = s.linkMonitor().GetFlapCount24h()
	resp.UptimeMs = s.linkMonitor().GetUptime().Milliseconds()

	history := s.linkMonitor().GetHistory()
	if len(history) > 0 {
		resp.History = make([]LinkHistoryEvent, len(history))
		for i, event := range history {
			resp.History[i] = LinkHistoryEvent{
				State:     event.State.String(),
				Timestamp: event.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			}
		}
	}
}

// addPhyInfo adds PoE and SFP/DDM info to response.
func addPhyInfo(resp *LinkResponse, iface string) {
	phyDetector := phy.NewDetector(iface)

	if poeStatus := phyDetector.GetPoEStatus(); poeStatus != nil && poeStatus.Detected {
		resp.PoE = &PoEInfo{
			Detected: poeStatus.Detected, Standard: poeStatus.Standard,
			Class: poeStatus.Class, PowerMw: poeStatus.PowerMw, Voltage: poeStatus.Voltage,
		}
	}

	sfpInfo := phyDetector.GetSFPInfo()
	if sfpInfo == nil || !sfpInfo.Present {
		return
	}
	resp.SFP = &SFPInfo{
		Present: sfpInfo.Present, Vendor: sfpInfo.Vendor, PartNumber: sfpInfo.PartNumber,
		Serial: sfpInfo.Serial, Type: sfpInfo.Type, Wavelength: sfpInfo.Wavelength,
		Distance: sfpInfo.Distance, Connector: sfpInfo.Connector, DDMSupport: sfpInfo.DDMSupport,
	}
	if sfpInfo.DDM != nil {
		resp.SFP.DDM = &SFPDDMInfo{
			Temperature: sfpInfo.DDM.Temperature, Voltage: sfpInfo.DDM.Voltage,
			TxPowerDbm: sfpInfo.DDM.TxPowerDbm, TxPowerMw: sfpInfo.DDM.TxPowerMw,
			RxPowerDbm: sfpInfo.DDM.RxPowerDbm, RxPowerMw: sfpInfo.DDM.RxPowerMw,
			LaserBiasMa: sfpInfo.DDM.LaserBiasMa, Alarms: sfpInfo.DDM.Alarms, Warnings: sfpInfo.DDM.Warnings,
		}
	}
}

// handleLink returns link status for the specified or current interface.
// Accepts optional query parameter: ?interface=eth0.
func (s *Server) handleLink(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

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
		)
		return
	}

	if err := s.netManager().RefreshInterfaces(); err != nil {
		logger.Error("Failed to refresh interfaces", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.network.refreshFailed"), "")
		return
	}

	currentIface := s.getInterfaceFromRequest(r)
	ifaceInfo, err := s.netManager().GetInterface(currentIface)
	if err != nil {
		logger.Warn("Interface not found", "error", err, "interface", currentIface)
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, localizer.T("errors.network.interfaceNotFound"), "")
		return
	}

	linkStatus, err := s.netManager().GetLinkStatus(currentIface)
	if err != nil {
		logging.GetLogger().
			Warn("Failed to get link status", "interface", currentIface, "error", err)
	}

	resp := LinkResponse{Interface: currentIface, LinkUp: false, MTU: ifaceInfo.MTU}
	if linkStatus != nil {
		resp.LinkUp = linkStatus.LinkUp
		resp.Carrier = linkStatus.Carrier
		resp.HasIP = linkStatus.HasIP
		resp.Speed = linkStatus.Speed
		resp.Duplex = linkStatus.Duplex
		resp.Advertised = linkStatus.Advertised
		resp.AutoNeg = linkStatus.AutoNeg
	}

	s.addLinkHistory(&resp)
	addPhyInfo(&resp, currentIface)

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// handleIPConfig returns IP configuration for the specified or current interface.
// Accepts optional query parameter: ?interface=eth0.
func (s *Server) handleIPConfig(w http.ResponseWriter, r *http.Request) {
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
		) // fixes #694
		return
	}

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
		logger.Error("Failed to refresh interfaces", "error", err)
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

	// Get interface from query param or fallback to current.
	currentIface := s.getInterfaceFromRequest(r)

	ifaceInfo, err := s.netManager().GetInterface(currentIface)
	if err != nil {
		logger.Warn("Interface not found", "error", err, "interface", currentIface)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusNotFound,
			ErrCodeNotFound,
			localizer.T("errors.network.interfaceNotFound"),
			"",
		)
		return
	}

	resp := IPConfigResponse{
		Interface: currentIface,
		MAC:       ifaceInfo.HardwareAddr,
		Mode:      "auto", // We'll detect this properly later
		IPv6:      []IPv6Info{},
		DNS:       []string{},
	}

	// Parse addresses into IPv4 and IPv6
	for _, addr := range ifaceInfo.Addresses {
		ipInfo := parseIPAddress(addr)
		if ipInfo.isIPv4 {
			resp.IPv4 = &IPv4Info{
				Address: ipInfo.address,
				Subnet:  ipInfo.subnet,
			}
		} else {
			resp.IPv6 = append(resp.IPv6, IPv6Info{
				Address: ipInfo.address,
				Prefix:  ipInfo.prefix,
				Scope:   ipInfo.scope,
				Source:  ipInfo.source,
			})
		}
	}

	// Get DHCP lease info and DNS
	applyDHCPLeaseInfo(&resp, currentIface)

	// Add DHCP timing if available
	s.applyDHCPTiming(&resp)

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// handleIPSettings handles GET/PUT for IP configuration settings.
func (s *Server) handleIPSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		s.handleIPSettingsGet(w, r)
	case http.MethodPut:
		s.handleIPSettingsPut(w, r, logger, localizer)
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

// handleIPSettingsGet returns the current IP configuration settings.
func (s *Server) handleIPSettingsGet(w http.ResponseWriter, _ *http.Request) {
	resp := IPSettingsResponse{
		Mode: s.config.IP.Mode,
	}

	if s.config.IP.Static != nil {
		resp.Address = s.config.IP.Static.Address
		resp.Netmask = s.config.IP.Static.Netmask
		resp.Gateway = s.config.IP.Static.Gateway
		resp.DNS = s.config.IP.Static.DNS
	}

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// applyStaticIPConfig applies static IP configuration, returns error message on failure.
func (s *Server) applyStaticIPConfig(
	iface string,
	req *IPSettingsRequest,
	logger *slog.Logger,
) error {
	cfg := &network.StaticIPConfig{
		Address: req.Address, Netmask: req.Netmask, Gateway: req.Gateway, DNS: req.DNS,
	}
	if err := s.netManager().ConfigureStaticIP(iface, cfg); err != nil {
		logger.Error("Failed to configure static IP", "error", err, "interface", iface)
		return err
	}
	s.config.IP.Mode = ipModeStatic
	s.config.IP.Static = &config.StaticIP{
		Address: req.Address, Netmask: req.Netmask, Gateway: req.Gateway, DNS: req.DNS,
	}
	return nil
}

// applyDHCPConfig applies DHCP configuration, returns error on failure.
func (s *Server) applyDHCPConfig(iface string, logger *slog.Logger) error {
	if err := s.netManager().ConfigureDHCP(iface); err != nil {
		logger.Error("Failed to configure DHCP", "error", err, "interface", iface)
		return err
	}
	s.config.IP.Mode = ipModeDHCP
	s.config.IP.Static = nil
	return nil
}

// handleIPSettingsPut updates the IP configuration settings.
// Accepts optional query parameter: ?interface=eth0.
func (s *Server) handleIPSettingsPut(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req IPSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	if req.Mode != ipModeDHCP && req.Mode != ipModeStatic {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.network.invalidMode"),
			"",
		)
		return
	}

	s.config.Lock()
	currentIface := s.getInterfaceFromRequest(r)

	var configErr error
	if req.Mode == ipModeStatic {
		configErr = s.applyStaticIPConfig(currentIface, &req, logger)
	} else {
		configErr = s.applyDHCPConfig(currentIface, logger)
	}

	s.config.Unlock()

	if configErr != nil {
		errMsg := localizer.T("errors.network.staticConfigFailed")
		if req.Mode == ipModeDHCP {
			errMsg = localizer.T("errors.network.dhcpConfigFailed")
		}
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			errMsg,
			"",
		)
		return
	}

	if err := s.config.Save(s.configPath); err != nil {
		logger.Error("Failed to save config", "error", err)
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

	if err := s.netManager().RefreshInterfaces(); err != nil {
		logger.Error("Failed to refresh interfaces", "error", err)
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

	sendJSONResponse(
		w,
		logger,
		http.StatusOK,
		map[string]string{"status": "success", "message": "IP configuration updated"},
	)
}

// handleSetMTU handles POST requests to set interface MTU.
func (s *Server) handleSetMTU(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req SetMTURequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	// Validate MTU value
	if err := validation.ValidateMTU(req.MTU); err != nil {
		logger.Warn("Invalid MTU value", "error", err, "mtu", req.MTU)
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

	// Use current interface if not specified
	iface := req.Interface
	if iface == "" {
		iface = s.netManager().GetCurrentInterface()
	}

	// Set the MTU
	if err := s.netManager().SetMTU(iface, req.MTU); err != nil {
		logger.Error("Failed to set MTU", "error", err, "interface", iface, "mtu", req.MTU)
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

	// Refresh interface data
	if err := s.netManager().RefreshInterfaces(); err != nil {
		logging.GetLogger().Warn("Failed to refresh interfaces after MTU change", "error", err)
	}

	sendJSONResponse(w, nil, http.StatusOK, map[string]any{
		"status":    "success",
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
				Warn("Invalid interface name in request", "interface", iface, "error", err)
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

// applyDHCPLeaseInfo populates the response with DHCP lease information.
func applyDHCPLeaseInfo(resp *IPConfigResponse, currentIface string) {
	leaseInfo, err := dhcp.GetLeaseInfo(currentIface)
	if err != nil || leaseInfo == nil {
		// Fallback: Try to get DNS servers from system
		resp.DNS = getSystemDNS()
		return
	}

	if resp.IPv4 != nil {
		if leaseInfo.Gateway != "" {
			resp.IPv4.Gateway = leaseInfo.Gateway
		}
		if leaseInfo.DHCPServer != "" {
			resp.IPv4.DHCPServer = leaseInfo.DHCPServer
			resp.Mode = ipModeDHCP
		}
		if leaseInfo.LeaseTime > 0 {
			resp.IPv4.LeaseTime = leaseInfo.LeaseTime
		}
	}

	// Use DNS from lease if available, otherwise fallback to system
	if len(leaseInfo.DNS) > 0 {
		resp.DNS = leaseInfo.DNS
	} else {
		resp.DNS = getSystemDNS()
	}
}

// applyDHCPTiming adds DHCP timing information to the response.
func (s *Server) applyDHCPTiming(resp *IPConfigResponse) {
	if s.dhcpMonitor() == nil {
		return
	}
	timing := s.dhcpMonitor().GetLastTiming()
	if timing == nil {
		return
	}
	ms := timing.ToMs()
	resp.Timing = &DHCPTimingInfo{
		Discover: ms.Discover,
		Offer:    ms.Offer,
		Request:  ms.Request,
		Total:    ms.Total,
	}
}

// parseIPAddress parses an IP address string (with CIDR) into components.
func parseIPAddress(addr string) ipAddrInfo {
	info := ipAddrInfo{
		scope:  "global",
		source: "static",
	}

	// Split address and prefix
	parts := splitCIDR(addr)
	info.address = parts[0]
	prefixStr := parts[1]

	// Determine if IPv4 or IPv6
	if isIPv4Address(info.address) {
		info.isIPv4 = true
		info.subnet = prefixStr
	} else {
		info.isIPv4 = false
		info.prefix = parsePrefix(prefixStr)

		// Determine IPv6 scope
		switch {
		case isLinkLocal(info.address):
			info.scope = "link-local"
		case isUniqueLocal(info.address):
			info.scope = "unique-local"
		default:
			info.scope = "global"
		}

		// Determine source (simplified - would need more info for accurate detection)
		info.source = "slaac"
	}

	return info
}

func splitCIDR(addr string) [2]string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == '/' {
			return [2]string{addr[:i], addr[i+1:]}
		}
	}
	return [2]string{addr, ""}
}

func isIPv4Address(addr string) bool {
	for _, c := range addr {
		if c == ':' {
			return false
		}
	}
	return true
}

func parsePrefix(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*decimalBase + int(c-'0')
		}
	}
	return result
}

func isLinkLocal(addr string) bool {
	// IPv6 link-local starts with fe80::
	return len(addr) >= 4 && (addr[:4] == "fe80" || addr[:4] == "FE80")
}

func isUniqueLocal(addr string) bool {
	// IPv6 unique local starts with fc or fd
	if len(addr) < minUniqueLocalAddrLen {
		return false
	}
	c := addr[0]
	c2 := addr[1]
	return (c == 'f' || c == 'F') && (c2 == 'c' || c2 == 'C' || c2 == 'd' || c2 == 'D')
}

func getSystemDNS() []string {
	// This is platform-specific. For now, return common defaults.
	// A full implementation would read /etc/resolv.conf on Linux
	// or use scutil on macOS.
	return []string{}
}
