// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// LLDP/CDP/EDP Discovery Handlers (fixes #544 - handlers split by feature)
// ============================================================================

// DiscoveryResponse contains the status and results of network discovery operations.
type DiscoveryResponse struct {
	Interface string                  `json:"interface"` // Interface used for discovery
	Running   bool                    `json:"running"`   // True if discovery managers are actively capturing
	Neighbors []DiscoveryNeighborInfo `json:"neighbors"` // All discovered neighbors (deduplicated by ChassisID + PortID)
}

// DiscoveryNeighborInfo represents a single discovered neighbor device in the API response.
//
// Information is gathered from link-layer discovery protocol advertisements (LLDP/CDP/EDP).
// Not all fields are present in every protocol - optional fields are omitted when empty.
type DiscoveryNeighborInfo struct {
	Protocol          string   `json:"protocol"`                    // "LLDP", "CDP", or "EDP"
	ChassisID         string   `json:"chassisId"`                   // Unique identifier for the neighbor device
	PortID            string   `json:"portId"`                      // Port/interface identifier on the neighbor
	PortDescription   string   `json:"portDescription,omitempty"`   // Human-readable port description
	SystemName        string   `json:"systemName,omitempty"`        // Hostname or system name of the neighbor
	SystemDescription string   `json:"systemDescription,omitempty"` // Device type, model, or software version
	Capabilities      []string `json:"capabilities,omitempty"`      // Device capabilities (bridge, router, etc.)
	ManagementAddress string   `json:"managementAddress,omitempty"` // IP address for device management
	TTL               int      `json:"ttl"`                         // Time-to-live in seconds until entry expires
	LastSeen          string   `json:"lastSeen"`                    // ISO 8601 timestamp of most recent advertisement
	SourceMAC         string   `json:"sourceMAC"`                   // MAC address of the advertising interface
}

// handleDiscovery returns discovery protocol neighbors from LLDP, CDP, and EDP.
//
// GET /api/discovery aggregates neighbors discovered from all active link-layer
// discovery protocols. The endpoint:
//   - Returns currently cached neighbors (no active scanning)
//   - Includes neighbors from LLDP (IEEE standard), CDP (Cisco), and EDP (Extreme)
//   - Deduplicates entries based on ChassisID + PortID combination
//   - Provides protocol-specific information when available
//   - Indicates whether discovery capture is currently running
//
// Neighbor information includes:
//   - Device identification (chassis ID, system name, management IP)
//   - Port information (port ID, description)
//   - Device capabilities (router, bridge, access point, etc.)
//   - Advertisement metadata (protocol, TTL, last seen timestamp)
//
// Use cases:
//   - Network topology mapping
//   - Switch/router neighbor discovery
//   - VLAN and network troubleshooting
//   - Automatic network documentation
//
// Discovery must be started separately via the discovery service.
// This endpoint only returns already-captured information.
//
// Authentication: Required
// Rate limiting: None (read-only operation)
//
// Response: 200 OK with DiscoveryResponse containing neighbors array.
func (s *Server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "") // fixes #694
		return
	}

	if s.discoveryManager == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable, ErrCodeServiceUnavail, localizer.T("errors.discovery.managerUnavailable"), "") // fixes #694
		return
	}

	// Get interface from query param or use current
	currentIface := s.getInterfaceFromRequest(r)

	neighbors := s.discoveryManager.GetNeighbors()
	resp := DiscoveryResponse{
		Interface: currentIface,
		Running:   s.discoveryManager.IsRunning(),
		Neighbors: make([]DiscoveryNeighborInfo, 0, len(neighbors)),
	}

	for _, n := range neighbors {
		resp.Neighbors = append(resp.Neighbors, DiscoveryNeighborInfo{
			Protocol:          string(n.Protocol),
			ChassisID:         n.ChassisID,
			PortID:            n.PortID,
			PortDescription:   n.PortDescription,
			SystemName:        n.SystemName,
			SystemDescription: n.SystemDescription,
			Capabilities:      n.Capabilities,
			ManagementAddress: n.ManagementAddress,
			TTL:               n.TTL,
			LastSeen:          n.LastSeen.Format("2006-01-02T15:04:05Z07:00"), // Fixes #909
			SourceMAC:         n.SourceMAC,                                    // Fixes #909
		})
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleDiscoveryOptions handles GET/PUT for the discovery options.
func (s *Server) handleDiscoveryOptions(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		s.getDiscoveryOptions(w, r)
	case http.MethodPut:
		s.setDiscoveryOptions(w, r)
	default:
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "") // fixes #694
	}
}

func (s *Server) getDiscoveryOptions(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	opts := s.discoveryService.GetOptions()
	sendJSONResponse(w, logger, http.StatusOK, map[string]interface{}{
		"options": opts,
	})
}

func (s *Server) setDiscoveryOptions(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req config.DiscoveryOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
		return
	}

	// Lock config for write access (fixes #759 - race condition)
	// NOTE: Must unlock before Save() - Save() acquires RLock internally (fixes #783)
	s.config.Lock()
	s.config.NetworkDiscovery.Options = req
	// Unlock before Save() to avoid deadlock
	s.config.Unlock()

	// Apply the options change to the running service
	if err := s.discoveryService.Reload(); err != nil {
		logger.Error("Failed to reload discovery options", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, localizer.T("errors.discovery.failedToApplyOptions"), "")
		return
	}

	// Save config to file (fixes #735 - return error on save failure)
	if err := s.config.Save(s.configPath); err != nil {
		logger.Error("Failed to save config", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, localizer.T("errors.settings.saveFailed"), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Discovery options updated",
	})
}

// handleDiscoveryServiceStatus returns the current status of the discovery service.
func (s *Server) handleDiscoveryServiceStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "") // fixes #694
		return
	}

	status := s.discoveryService.GetStatus()
	sendJSONResponse(w, logger, http.StatusOK, status)
}
