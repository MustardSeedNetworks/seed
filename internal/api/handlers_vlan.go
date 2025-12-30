// Package api provides the HTTP/WebSocket server.
// handlers_vlan.go contains VLAN management handlers.
// Split from handlers_network.go for code organization (Plan F).
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
	"github.com/krisarmstrong/seed/internal/sap/vlan"
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

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed", "")
		return
	}

	if s.vlanManager == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "VLAN manager not available", "")
		return
	}

	// Get VLAN info from LLDP/CDP if available
	var nativeVlan, voiceVlan *int
	if s.discoveryService != nil {
		neighbors := s.discoveryService.GetNeighbors()
		// Use first neighbor for VLAN information
		if len(neighbors) > 0 {
			n := neighbors[0]
			// LLDP can carry VLAN information in TLVs
			if n.NativeVLAN > 0 {
				v := n.NativeVLAN
				nativeVlan = &v
			}
			if n.VoiceVLAN > 0 {
				v := n.VoiceVLAN
				voiceVlan = &v
			}
		}
	}

	// Get VLAN info (including detected subinterfaces)
	info := s.vlanManager.GetInfoWithLLDP(nativeVlan, voiceVlan)

	resp := VLANResponse{
		NativeVlan:  info.NativeVlan,
		TaggedVlans: info.TaggedVlans,
		VoiceVlan:   info.VoiceVlan,
	}
	resp.Configured.Enabled = info.Configured.Enabled
	resp.Configured.ID = info.Configured.ID

	sendJSONResponse(w, nil, http.StatusOK, resp)
}

// handleVLANTraffic returns VLAN traffic statistics from frame capture.
func (s *Server) handleVLANTraffic(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	if s.vlanTrafficMonitor == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable, ErrCodeServiceUnavail, localizer.TWithData("errors.service.notAvailable", map[string]interface{}{"service": "VLAN traffic monitor"}), "")
		return
	}

	stats := s.vlanTrafficMonitor.GetStats()
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
		Running: s.vlanTrafficMonitor.IsRunning(),
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
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
	}
}

// parseVLANRequest parses and validates a VLAN interface request.
// Returns the validated interface name, VLAN ID, and success boolean.
func (s *Server) parseVLANRequest(w http.ResponseWriter, r *http.Request, logger *slog.Logger, localizer *i18n.Localizer) (iface string, vlanID int, ok bool) {
	// Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req VLANInterfaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
		return "", 0, false
	}

	// Validate VLAN ID
	if err := validation.ValidateVLANID(req.VlanID); err != nil {
		logger.Warn("Invalid VLAN ID", "error", err, "vlanID", req.VlanID)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation, localizer.T("errors.vlan.invalidId"), "")
		return "", 0, false
	}

	// Use current interface if not specified
	iface = req.Interface
	if iface == "" {
		iface = s.netManager.GetCurrentInterface()
	}

	// Validate interface name
	if err := validation.ValidateInterface(iface); err != nil {
		logger.Warn("Invalid interface", "error", err, "interface", iface)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation, localizer.T("errors.network.invalidInterface"), "")
		return "", 0, false
	}

	vlanID = req.VlanID
	ok = true
	return iface, vlanID, ok
}

// createVLANInterface creates an 802.1Q VLAN subinterface.
//
//nolint:dupl // Intentionally similar to deleteVLANInterface - different CRUD operations
func (s *Server) createVLANInterface(w http.ResponseWriter, r *http.Request, logger *slog.Logger, localizer *i18n.Localizer) {
	iface, vlanID, ok := s.parseVLANRequest(w, r, logger, localizer)
	if !ok {
		return
	}

	if err := vlan.CreateVlanInterface(iface, vlanID); err != nil {
		logger.Error("Failed to create VLAN interface", "error", err, "interface", iface, "vlanID", vlanID)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, localizer.T("errors.vlan.failedToCreate"), "")
		return
	}

	sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "VLAN interface created",
		"interface": iface,
		"vlanId":    vlanID,
	})
}

// deleteVLANInterface removes an 802.1Q VLAN subinterface.
//
//nolint:dupl // Intentionally similar to createVLANInterface - different CRUD operations
func (s *Server) deleteVLANInterface(w http.ResponseWriter, r *http.Request, logger *slog.Logger, localizer *i18n.Localizer) {
	iface, vlanID, ok := s.parseVLANRequest(w, r, logger, localizer)
	if !ok {
		return
	}

	if err := vlan.DeleteVlanInterface(iface, vlanID); err != nil {
		logger.Error("Failed to delete VLAN interface", "error", err, "interface", iface, "vlanID", vlanID)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, localizer.T("errors.vlan.failedToDelete"), "")
		return
	}

	sendJSONResponse(w, nil, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "VLAN interface deleted",
		"interface": iface,
		"vlanId":    vlanID,
	})
}
