// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// LLDP/CDP/EDP Discovery Handlers (fixes #544 - handlers split by feature)
// ============================================================================

// DiscoveryResponse contains the status and results of network discovery operations.
type DiscoveryResponse struct {
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.discoveryManager == nil {
		http.Error(w, "Discovery manager not available", http.StatusServiceUnavailable)
		return
	}

	neighbors := s.discoveryManager.GetNeighbors()
	resp := DiscoveryResponse{
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
		})
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleDiscoveryProfile handles GET/PUT for the discovery profile.
func (s *Server) handleDiscoveryProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getDiscoveryProfile(w, r)
	case http.MethodPut:
		s.setDiscoveryProfile(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getDiscoveryProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	profile := s.discoveryService.GetProfile()
	sendJSONResponse(w, logger, http.StatusOK, map[string]interface{}{
		"profile": profile,
	})
}

func (s *Server) setDiscoveryProfile(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req struct {
		Profile string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert string to DiscoveryProfile
	profile := config.DiscoveryProfile(req.Profile)

	// Validate profile
	switch profile {
	case config.ProfileStealth, config.ProfileStandard, config.ProfileFullScan, config.ProfileCustom:
		// Valid profile
	default:
		http.Error(w, "Invalid profile: must be stealth, standard, full_scan, or custom", http.StatusBadRequest)
		return
	}

	// Update the config
	s.config.NetworkDiscovery.Profile = profile

	// Apply the profile change to the running service
	if err := s.discoveryService.SetProfile(profile); err != nil {
		logger.Error("Failed to set discovery profile", "error", err)
		http.Error(w, "Failed to apply profile", http.StatusInternalServerError)
		return
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		logger.Warn("Failed to save config", "error", err)
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Profile updated to " + string(profile),
	})
}

// handleDiscoveryServiceStatus returns the current status of the discovery service.
func (s *Server) handleDiscoveryServiceStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.discoveryService.GetStatus()
	sendJSONResponse(w, logger, http.StatusOK, status)
}
