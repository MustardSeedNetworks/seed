package api

// handlers_network_ip.go carries interface addressing: the reported IPv4/IPv6
// configuration, DHCP lease and timing, and the static/DHCP settings handlers.

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/network/ipconfig"
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
//
// Three intervals, not four: DORA's four packets yield Discover->Offer,
// Offer->Request and Request->Ack. There was an Ack field here that nothing
// could populate — applyDHCPTiming never set it — so it went out as 0 on every
// response, describing a phase that does not exist.
type DHCPTimingInfo struct {
	Discover int64 `json:"discover"` // ms, Discover -> Offer
	Offer    int64 `json:"offer"`    // Offer -> Request
	Request  int64 `json:"request"`  // Request -> Ack
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

// handleIPConfig returns IP configuration for the specified or current interface.
// Accepts optional query parameter: ?interface=eth0.
func (s *Server) handleIPConfig(w http.ResponseWriter, r *http.Request) {
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

	// Get interface from query param or fallback to current.
	currentIface := s.getInterfaceFromRequest(r)

	ifaceInfo, err := s.netManager().GetInterface(currentIface)
	if err != nil {
		logger.WarnContext(r.Context(), "Interface not found", "error", err, "interface", currentIface)
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
	st := s.networkIP.Settings()
	sendJSONResponse(w, nil, http.StatusOK, IPSettingsResponse{
		Mode:    st.Mode,
		Address: st.Address,
		Netmask: st.Netmask,
		Gateway: st.Gateway,
		DNS:     st.DNS,
	})
}

// handleIPSettingsPut updates the IP configuration settings.
// Accepts optional query parameter: ?interface=eth0.
func (s *Server) handleIPSettingsPut(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) {
	var req IPSettingsRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	iface := s.getInterfaceFromRequest(r)
	err := s.networkIP.Apply(iface, req.Mode, ipconfig.StaticIP{
		Address: req.Address, Netmask: req.Netmask, Gateway: req.Gateway, DNS: req.DNS,
	})
	if err != nil {
		s.writeIPApplyError(w, r, logger, localizer, err)
		return
	}

	sendJSONResponse(
		w,
		logger,
		http.StatusOK,
		map[string]string{"status": statusSuccess, "message": "IP configuration updated"},
	)
}

// writeIPApplyError maps an IPService.Apply sentinel to the pre-strangle HTTP
// response (status + localized message).
func (s *Server) writeIPApplyError(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
	err error,
) {
	switch {
	case errors.Is(err, ipconfig.ErrInvalidMode):
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.network.invalidMode"), "")
	case errors.Is(err, ipconfig.ErrInvalidConfig):
		// Rejected before anything was applied, so it is the request that is
		// wrong, not the server. The reason names the offending field and is
		// the only way the operator learns which one (#50).
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.network.invalidConfig"), err.Error())
	case errors.Is(err, ipconfig.ErrStaticConfig):
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.network.staticConfigFailed"), "")
	case errors.Is(err, ipconfig.ErrDHCPConfig):
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.network.dhcpConfigFailed"), "")
	case errors.Is(err, ipconfig.ErrSave):
		logger.ErrorContext(r.Context(), "Failed to save config", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.config.failedToSave"), "")
	default: // ErrRefresh
		logger.ErrorContext(r.Context(), "Failed to refresh interfaces", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.network.refreshFailed"), "")
	}
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

// getSystemDNS returns the host's configured resolvers, used as the IP config
// fallback when a DHCP lease carries none of its own.
func getSystemDNS() []string {
	return dns.GetSystemDNS()
}
