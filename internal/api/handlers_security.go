package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	securitysettings "github.com/MustardSeedNetworks/seed/internal/security/settings"
)

// passwordPlaceholder is used to mask sensitive values in API responses.
const passwordPlaceholder = "*****"

// ============================================================================
// Request/Response Types and Handlers (fixes #544 - split from handlers.go)
// ============================================================================

// RogueDHCPResponse represents rogue DHCP detection status.
type RogueDHCPResponse struct {
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// RogueServersResponse contains the list of detected DHCP servers.
type RogueServersResponse struct {
	Servers         []RogueServer `json:"servers"`
	RogueCount      int           `json:"rogueCount"`
	AuthorizedCount int           `json:"authorizedCount"`
}

// RogueServer is the flat transport view of a detected DHCP server, mirroring
// dhcp.RogueServer's wire shape so the published schema does not depend on the
// dhcp domain package.
type RogueServer struct {
	IP           string    `json:"ip"`
	MAC          string    `json:"mac"`
	FirstSeen    time.Time `json:"firstSeen"`
	LastSeen     time.Time `json:"lastSeen"`
	OfferCount   int       `json:"offerCount"`
	IsAuthorized bool      `json:"isAuthorized"`
}

// toRogueServers maps detected DHCP servers onto their flat transport view.
func toRogueServers(servers []*dhcp.RogueServer) []RogueServer {
	out := make([]RogueServer, 0, len(servers))
	for _, srv := range servers {
		if srv == nil {
			continue
		}
		out = append(out, RogueServer{
			IP:           srv.IP,
			MAC:          srv.MAC,
			FirstSeen:    srv.FirstSeen,
			LastSeen:     srv.LastSeen,
			OfferCount:   srv.OfferCount,
			IsAuthorized: srv.IsAuthorized,
		})
	}
	return out
}

// RogueDHCPConfigResponse contains the rogue DHCP detector configuration.
type RogueDHCPConfigResponse struct {
	Enabled          bool     `json:"enabled"`
	KnownServers     []string `json:"knownServers"`
	AlertOnDetection bool     `json:"alertOnDetection"`
	Interface        string   `json:"interface"`
}

// handleRogueDHCP starts/stops rogue DHCP detection (POST) or gets status (GET).
func (s *Server) handleRogueDHCP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		resp := RogueDHCPResponse{
			Enabled: s.securitySettings.RogueEnabled(),
			Running: s.rogueDetector().IsRunning(),
		}
		sendJSONResponse(w, logger, http.StatusOK, resp)

	case http.MethodPost:
		s.handleRogueDHCPAction(w, r, logger, localizer)

	default:
		sendErrorResponseWithDetails(
			w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"), "",
		)
	}
}

// handleRogueDHCPAction handles POST requests to start/stop rogue DHCP detection.
func (s *Server) handleRogueDHCPAction(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) {
	var req struct {
		Action string `json:"action"`
	}
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	resp := RogueDHCPResponse{Enabled: s.securitySettings.RogueEnabled()}

	switch strings.ToLower(req.Action) {
	case "start":
		s.handleRogueDHCPStart(w, logger, &resp)
	case "stop":
		s.handleRogueDHCPStop(w, logger, &resp)
	default:
		sendErrorResponseWithDetails(
			w, logger, http.StatusBadRequest, ErrCodeBadRequest,
			localizer.T("errors.security.invalidAction"), "",
		)
	}
}

// handleRogueDHCPStart starts the rogue DHCP detector.
func (s *Server) handleRogueDHCPStart(
	w http.ResponseWriter,
	logger *slog.Logger,
	resp *RogueDHCPResponse,
) {
	if !s.securitySettings.RogueEnabled() {
		resp.Error = "Rogue DHCP detection is disabled in configuration"
		sendJSONResponse(w, logger, http.StatusBadRequest, *resp)
		return
	}
	if s.rogueDetector().IsRunning() {
		resp.Running = true
		resp.Message = "Rogue DHCP detector already running"
		sendJSONResponse(w, logger, http.StatusOK, *resp)
		return
	}
	if err := s.rogueDetector().Start(); err != nil {
		logger.Error("Failed to start rogue DHCP detector", "error", err)
		resp.Error = "internal server error"
		sendJSONResponse(w, logger, http.StatusInternalServerError, *resp)
		return
	}
	resp.Running = true
	resp.Message = "Rogue DHCP detector started"
	sendJSONResponse(w, logger, http.StatusOK, *resp)
}

// handleRogueDHCPStop stops the rogue DHCP detector.
func (s *Server) handleRogueDHCPStop(
	w http.ResponseWriter,
	logger *slog.Logger,
	resp *RogueDHCPResponse,
) {
	if !s.rogueDetector().IsRunning() {
		resp.Running = false
		resp.Message = "Rogue DHCP detector not running"
		sendJSONResponse(w, logger, http.StatusOK, *resp)
		return
	}
	if err := s.rogueDetector().Stop(); err != nil {
		logger.Error("Failed to stop rogue DHCP detector", "error", err)
		resp.Error = "internal server error"
		sendJSONResponse(w, logger, http.StatusInternalServerError, *resp)
		return
	}
	resp.Running = false
	resp.Message = "Rogue DHCP detector stopped"
	sendJSONResponse(w, logger, http.StatusOK, *resp)
}

// handleRogueDHCPServers returns detected DHCP servers (GET) or clears the list (DELETE).
func (s *Server) handleRogueDHCPServers(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		// Get all detected servers
		servers := s.rogueDetector().GetDetectedServers()
		rogues := s.rogueDetector().GetRogueServers()

		resp := RogueServersResponse{
			Servers:         toRogueServers(servers),
			RogueCount:      len(rogues),
			AuthorizedCount: len(servers) - len(rogues),
		}
		sendJSONResponse(w, logger, http.StatusOK, resp)

	case http.MethodDelete:
		// Clear detected servers list
		s.rogueDetector().ClearDetectedServers()
		sendJSONResponse(w, logger, http.StatusOK, map[string]string{
			"message": "Detected servers list cleared",
		})

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

// handleRogueDHCPConfig gets (GET) or updates (PUT) the rogue DHCP detector
// configuration. Thin transport (ADR-0020): delegates to the security-settings
// service, which owns the config merge/persist + live-detector sync.
func (s *Server) handleRogueDHCPConfig(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, logger, http.StatusOK, rogueViewToResponse(s.securitySettings.RogueDHCP()))

	case http.MethodPut:
		var req struct {
			Enabled          *bool    `json:"enabled,omitempty"`
			KnownServers     []string `json:"knownServers,omitempty"`
			AlertOnDetection *bool    `json:"alertOnDetection,omitempty"`
		}
		if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
			return
		}
		err := s.securitySettings.UpdateRogueDHCP(securitysettings.RogueUpdate{
			Enabled:          req.Enabled,
			KnownServers:     req.KnownServers,
			AlertOnDetection: req.AlertOnDetection,
		})
		if err != nil {
			logger.ErrorContext(r.Context(), "Failed to save rogue DHCP config", "error", err)
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.config.failedToSave"), "")
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, rogueViewToResponse(s.securitySettings.RogueDHCP()))

	default:
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
	}
}

// rogueViewToResponse maps the service read model to the wire DTO.
func rogueViewToResponse(v securitysettings.RogueView) RogueDHCPConfigResponse {
	return RogueDHCPConfigResponse{
		Enabled:          v.Enabled,
		KnownServers:     v.KnownServers,
		AlertOnDetection: v.AlertOnDetection,
		Interface:        v.Interface,
	}
}

// GatewayPingResult is the ping outcome for a single gateway. It is the flat,
// self-contained value object shared by the IPv4 and IPv6 sides of a gateway
// test, so the transport contract never has to reference itself.
type GatewayPingResult struct {
	Gateway     string  `json:"gateway"`
	Reachable   bool    `json:"reachable"`
	Sent        int     `json:"sent"`
	Received    int     `json:"received"`
	LossPercent float64 `json:"lossPercent"`
	MinTime     float64 `json:"minTime"`
	MaxTime     float64 `json:"maxTime"`
	AvgTime     float64 `json:"avgTime"`
	LastTime    float64 `json:"lastTime"`
	Status      string  `json:"status"`
}

// GatewayResponse represents the gateway ping test results for the API: the
// IPv4 gateway result (promoted to the top level) plus the IPv6 gateway result
// when one is detected. It is non-recursive — the IPv6 sibling is a flat
// GatewayPingResult and carries no further nesting — which keeps the published
// schema acyclic.
type GatewayResponse struct {
	GatewayPingResult

	IPv6 *GatewayPingResult `json:"ipv6,omitempty"`
}

// handleGateway performs gateway ping testing and returns results.
func (s *Server) handleGateway(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.gatewayTester() == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.security.gatewayTesterUnavailable"),
			"",
		) // fixes #694
		return
	}

	// Check if requested interface has connectivity via link monitor
	currentIface := s.getInterfaceFromRequest(r)
	if currentIface != "" && s.linkMonitor() != nil {
		// If link is down, return disconnected status
		if !s.linkMonitor().IsUp() {
			resp := GatewayResponse{
				GatewayPingResult: GatewayPingResult{
					Gateway:   "",
					Reachable: false,
					Status:    "disconnected",
				},
			}
			sendJSONResponse(w, logger, http.StatusOK, resp)
			return
		}
	}

	// Perform IPv4 gateway ping test
	stats := s.gatewayTester().Test()

	resp := GatewayResponse{
		GatewayPingResult: GatewayPingResult{
			Gateway:     stats.Gateway,
			Reachable:   stats.Reachable,
			Sent:        stats.Sent,
			Received:    stats.Received,
			LossPercent: stats.LossPercent,
			MinTime:     stats.MinTime,
			MaxTime:     stats.MaxTime,
			AvgTime:     stats.AvgTime,
			LastTime:    stats.LastTime,
			Status:      string(stats.Status),
		},
	}

	// Detect and ping IPv6 gateway if available
	ipv6Gateway, err := gateway.DetectGatewayIPv6()
	if err == nil && ipv6Gateway != "" {
		// Create a temporary tester for IPv6
		ipv6Tester := gateway.NewTester(gateway.DefaultThresholds())
		ipv6Tester.SetGateway(ipv6Gateway)
		ipv6Stats := ipv6Tester.Test()

		resp.IPv6 = &GatewayPingResult{
			Gateway:     ipv6Stats.Gateway,
			Reachable:   ipv6Stats.Reachable,
			Sent:        ipv6Stats.Sent,
			Received:    ipv6Stats.Received,
			LossPercent: ipv6Stats.LossPercent,
			MinTime:     ipv6Stats.MinTime,
			MaxTime:     ipv6Stats.MaxTime,
			AvgTime:     ipv6Stats.AvgTime,
			LastTime:    ipv6Stats.LastTime,
			Status:      string(ipv6Stats.Status),
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// VLANResponse represents the VLAN information for the API.

// SNMPSettingsResponse represents the SNMP configuration settings.
type SNMPSettingsResponse struct {
	Communities   []string                   `json:"communities"`
	V3Credentials []SNMPv3CredentialResponse `json:"v3Credentials"`
	Timeout       int                        `json:"timeout"` // milliseconds
	Retries       int                        `json:"retries"`
	Port          int                        `json:"port"`
}

// SNMPv3CredentialResponse represents an SNMPv3 credential for API responses.
type SNMPv3CredentialResponse struct {
	Name          string `json:"name"`
	Username      string `json:"username"`
	AuthProtocol  string `json:"authProtocol"`
	AuthPassword  string `json:"authPassword"`
	PrivProtocol  string `json:"privProtocol"`
	PrivPassword  string `json:"privPassword"`
	ContextName   string `json:"contextName"`
	SecurityLevel string `json:"securityLevel"`
}

// handleSNMPSettings handles GET/PUT for SNMP settings.
func (s *Server) handleSNMPSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		s.getSNMPSettings(w, r)
	case http.MethodPut:
		s.updateSNMPSettings(w, r)
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

// getSNMPSettings serves the SNMP settings with passwords masked. Thin transport.
func (s *Server) getSNMPSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	sendJSONResponse(w, logger, http.StatusOK, snmpViewToResponse(s.securitySettings.SNMP()))
}

// updateSNMPSettings persists SNMP settings (encrypting new passwords). Thin
// transport: decode, map to the domain update, delegate to the service, and map
// the typed encrypt errors to the distinct i18n messages.
func (s *Server) updateSNMPSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req SNMPSettingsResponse
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	switch err := s.securitySettings.UpdateSNMP(snmpRequestToUpdate(req)); {
	case errors.Is(err, securitysettings.ErrEncryptPriv):
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.security.failedToEncryptPriv"), "")
	case errors.Is(err, securitysettings.ErrEncryptAuth):
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.security.failedToEncryptAuth"), "")
	case err != nil:
		logger.ErrorContext(r.Context(), "Failed to save SNMP settings", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.config.failedToSave"), "")
	default:
		sendJSONResponse(w, logger, http.StatusOK, map[string]string{
			"status": statusSuccess, "message": "SNMP settings updated",
		})
	}
}

// snmpViewToResponse maps the service read model to the wire DTO.
func snmpViewToResponse(v securitysettings.SNMPView) SNMPSettingsResponse {
	creds := make([]SNMPv3CredentialResponse, len(v.V3Credentials))
	for i, c := range v.V3Credentials {
		creds[i] = SNMPv3CredentialResponse{
			Name:          c.Name,
			Username:      c.Username,
			AuthProtocol:  c.AuthProtocol,
			AuthPassword:  c.AuthPassword,
			PrivProtocol:  c.PrivProtocol,
			PrivPassword:  c.PrivPassword,
			ContextName:   c.ContextName,
			SecurityLevel: c.SecurityLevel,
		}
	}
	return SNMPSettingsResponse{
		Communities:   v.Communities,
		V3Credentials: creds,
		Timeout:       v.TimeoutMs,
		Retries:       v.Retries,
		Port:          v.Port,
	}
}

// snmpRequestToUpdate maps the wire DTO to the domain update model.
func snmpRequestToUpdate(req SNMPSettingsResponse) securitysettings.SNMPUpdate {
	creds := make([]securitysettings.SNMPv3Credential, len(req.V3Credentials))
	for i, c := range req.V3Credentials {
		creds[i] = securitysettings.SNMPv3Credential{
			Name:          c.Name,
			Username:      c.Username,
			AuthProtocol:  c.AuthProtocol,
			AuthPassword:  c.AuthPassword,
			PrivProtocol:  c.PrivProtocol,
			PrivPassword:  c.PrivPassword,
			ContextName:   c.ContextName,
			SecurityLevel: c.SecurityLevel,
		}
	}
	return securitysettings.SNMPUpdate{
		Communities:   req.Communities,
		V3Credentials: creds,
		TimeoutMs:     req.Timeout,
		Retries:       req.Retries,
		Port:          req.Port,
	}
}
