package api

// handlers_vlan.go contains VLAN management handlers.
// Split from handlers_network.go for code organization (Plan F).

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// ============================================================================
// VLAN Types
// ============================================================================

// VLANResponse contains VLAN configuration and detection information for an interface.
type VLANResponse struct {
	NativeVlan  *int  `json:"nativeVlan"`
	TaggedVlans []int `json:"taggedVlans"`
	VoiceVlan   *int  `json:"voiceVlan"`
	Configured  struct {
		Enabled bool `json:"enabled"`
		ID      int  `json:"id"`
	} `json:"configured"`
}

// VLANTrafficResponse represents the VLAN traffic statistics for the API.
type VLANTrafficResponse struct {
	VLANs   []VLANTrafficEntry `json:"vlans"`
	Running bool               `json:"running"`
}

// VLANTrafficEntry represents traffic statistics for a single VLAN.
type VLANTrafficEntry struct {
	ID       int    `json:"id"`
	Packets  uint64 `json:"packets"`
	Bytes    uint64 `json:"bytes"`
	LastSeen string `json:"lastSeen"`
}

// VLANInterfaceRequest represents the request to create/delete a VLAN interface.
type VLANInterfaceRequest struct {
	Interface string `json:"interface"`
	VlanID    int    `json:"vlanId"`
}

// ============================================================================
// VLAN Handlers
// ============================================================================

// handleVLAN returns VLAN information for the current interface.
func (s *Server) handleVLAN(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if s.vlanManager() == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"VLAN manager not available",
			"",
		)
		return
	}

	// Get VLAN info from LLDP/CDP if available
	nativeVlan, voiceVlan := s.getVLANsFromDiscovery()

	// Get VLAN info (including detected subinterfaces)
	info := s.vlanManager().GetInfoWithLLDP(nativeVlan, voiceVlan)

	resp := VLANResponse{
		NativeVlan:  info.NativeVlan,
		TaggedVlans: info.TaggedVlans,
		VoiceVlan:   info.VoiceVlan,
	}
	resp.Configured.Enabled = info.Configured.Enabled
	resp.Configured.ID = info.Configured.ID

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// getVLANsFromDiscovery extracts VLAN information from LLDP/CDP neighbors.
func (s *Server) getVLANsFromDiscovery() (*int, *int) {
	if s.discoveryService() == nil {
		return nil, nil
	}

	neighbors := s.discoveryService().GetNeighbors()
	if len(neighbors) == 0 {
		return nil, nil
	}

	// Use first neighbor for VLAN information
	n := neighbors[0]
	var nativeVlan, voiceVlan *int

	// LLDP can carry VLAN information in TLVs
	if n.NativeVLAN > 0 {
		v := n.NativeVLAN
		nativeVlan = &v
	}
	if n.VoiceVLAN > 0 {
		v := n.VoiceVLAN
		voiceVlan = &v
	}

	return nativeVlan, voiceVlan
}

// handleVLANTraffic returns VLAN traffic statistics from frame capture.
func (s *Server) handleVLANTraffic(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.vlanTrafficMonitor() == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.TWithData(
				"errors.service.notAvailable",
				map[string]any{"service": "VLAN traffic monitor"},
			),
			"",
		)
		return
	}

	stats := s.vlanTrafficMonitor().GetStats()
	entries := make([]VLANTrafficEntry, 0, len(stats))
	for _, stat := range stats {
		entries = append(entries, VLANTrafficEntry{
			ID:       stat.ID,
			Packets:  stat.Packets,
			Bytes:    stat.Bytes,
			LastSeen: stat.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	resp := VLANTrafficResponse{
		VLANs:   entries,
		Running: s.vlanTrafficMonitor().IsRunning(),
	}

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// ============================================================================
// VLAN Interface Management
// ============================================================================

// handleVLANInterface handles POST (create) and DELETE (remove) for VLAN subinterfaces.
func (s *Server) handleVLANInterface(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodPost:
		s.createVLANInterface(w, r, logger, localizer)
	case http.MethodDelete:
		s.deleteVLANInterface(w, r, logger, localizer)
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

// parseVLANRequest parses and validates a VLAN interface request.
// Returns the validated interface name, VLAN ID, and success boolean.
func (s *Server) parseVLANRequest(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) (string, int, bool) {
	var req VLANInterfaceRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return "", 0, false
	}

	// Validate VLAN ID
	if err := validation.ValidateVLANID(req.VlanID); err != nil {
		logger.WarnContext(r.Context(), "Invalid VLAN ID", "error", err, "vlanID", req.VlanID)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.vlan.invalidId"),
			"",
		)
		return "", 0, false
	}

	// Use current interface if not specified
	iface := req.Interface
	if iface == "" {
		iface = s.netManager().GetCurrentInterface()
	}

	// Validate interface name
	if err := validation.ValidateInterface(iface); err != nil {
		logger.WarnContext(r.Context(), "Invalid interface", "error", err, "interface", iface)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeValidation,
			localizer.T("errors.network.invalidInterface"),
			"",
		)
		return "", 0, false
	}

	return iface, req.VlanID, true
}

// createVLANInterface creates an 802.1Q VLAN subinterface.
func (s *Server) createVLANInterface(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) {
	s.applyVLANOp(w, r, logger, localizer, vlanOp{
		fn:         s.vlanCreate,
		logMsg:     "Failed to create VLAN interface",
		failureKey: "errors.vlan.failedToCreate",
		successMsg: "VLAN interface created",
	})
}

// deleteVLANInterface removes an 802.1Q VLAN subinterface.
func (s *Server) deleteVLANInterface(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) {
	s.applyVLANOp(w, r, logger, localizer, vlanOp{
		fn:         s.vlanDelete,
		logMsg:     "Failed to delete VLAN interface",
		failureKey: "errors.vlan.failedToDelete",
		successMsg: "VLAN interface deleted",
	})
}

// vlanOp names the create/delete-specific pieces applyVLANOp needs: the
// platform call itself and what to say about it on failure and success.
type vlanOp struct {
	fn         func(parentIface string, vlanID int) error
	logMsg     string
	failureKey string
	successMsg string
}

// applyVLANOp is the shared body of createVLANInterface and
// deleteVLANInterface: parse the request, decline up front on a platform that
// cannot do this at all, run the operation, and map its result to a response.
// An unsupported platform can also surface from the attempt itself when the
// up-front check and the attempt disagree; that is still a 501, not a 500.
func (s *Server) applyVLANOp(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
	op vlanOp,
) {
	iface, vlanID, ok := s.parseVLANRequest(w, r, logger, localizer)
	if !ok {
		return
	}

	if !s.vlanCreateSupported() {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusNotImplemented,
			ErrCodeNotImplemented,
			localizer.T("errors.vlan.notSupportedOnPlatform"),
			"",
		)
		return
	}

	if err := op.fn(iface, vlanID); err != nil {
		status, code := http.StatusInternalServerError, ErrCodeInternal
		message := localizer.T(op.failureKey)
		if errors.Is(err, vlan.ErrUnsupported) {
			status, code = http.StatusNotImplemented, ErrCodeNotImplemented
			message = localizer.T("errors.vlan.notSupportedOnPlatform")
		}

		logger.ErrorContext(r.Context(), op.logMsg,
			"error", err, "interface", iface, "vlanID", vlanID)
		sendErrorResponseWithDetails(w, logger, status, code, message, "")
		return
	}

	sendJSONResponse(w, nil, http.StatusOK, map[string]any{
		"status":    statusSuccess,
		"message":   op.successMsg,
		"interface": iface,
		"vlanId":    vlanID,
	})
}

// vlanSeam is the VLAN create/delete test seam, held as a single Server
// field (server.go) rather than three -- that file is baselined by the
// file-size gate and may not grow. Zero value means production, where the
// vlan package is called directly; a test sets createSupported/create/delete
// because the real functions would create or remove an 802.1Q subinterface
// on the machine running the tests.
type vlanSeam struct {
	createSupported func() bool
	create          func(parentIface string, vlanID int) error
	delete          func(parentIface string, vlanID int) error
}

// vlanCreateSupported reports platform support, through the server's seam
// when a test has set one.
func (s *Server) vlanCreateSupported() bool {
	if s.vlanSeam.createSupported != nil {
		return s.vlanSeam.createSupported()
	}

	return vlan.CreateSupported()
}

// vlanCreate creates the subinterface, through the server's seam when a test
// has set one.
func (s *Server) vlanCreate(parentIface string, vlanID int) error {
	if s.vlanSeam.create != nil {
		return s.vlanSeam.create(parentIface, vlanID)
	}

	return vlan.CreateVlanInterface(parentIface, vlanID)
}

// vlanDelete removes the subinterface, through the server's seam when a test
// has set one.
func (s *Server) vlanDelete(parentIface string, vlanID int) error {
	if s.vlanSeam.delete != nil {
		return s.vlanSeam.delete(parentIface, vlanID)
	}

	return vlan.DeleteVlanInterface(parentIface, vlanID)
}
