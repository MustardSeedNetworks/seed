package api

// handlers_network_link.go carries link state for an interface: speed/duplex,
// PoE, SFP and its DDM readings, and the recent link-change history.

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/phy"
)

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
		logger.ErrorContext(r.Context(), "Failed to refresh interfaces", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.network.refreshFailed"), "")
		return
	}

	currentIface := s.getInterfaceFromRequest(r)
	ifaceInfo, err := s.netManager().GetInterface(currentIface)
	if err != nil {
		logger.WarnContext(r.Context(), "Interface not found", "error", err, "interface", currentIface)
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, localizer.T("errors.network.interfaceNotFound"), "")
		return
	}

	linkStatus, err := s.netManager().GetLinkStatus(currentIface)
	if err != nil {
		logging.GetLogger().
			WarnContext(r.Context(), "Failed to get link status", "interface", currentIface, "error", err)
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
