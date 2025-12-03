package api

import (
	"encoding/json"
	"net/http"
)

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

// StatusResponse represents the system status.
type StatusResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    int64  `json:"uptime"`
	Interface string `json:"interface"`
}

// handleLogin handles user authentication.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := s.authManager.Authenticate(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	resp := LoginResponse{
		Token:   token,
		Expires: int64(s.config.Auth.SessionTimeout.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleLogout handles user logout.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// JWT is stateless, so we just acknowledge the logout
	// Client should discard the token
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "logged out"})
}

// handleStatus returns the system status.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := StatusResponse{
		Status:    "ok",
		Version:   "0.1.0",
		Interface: s.config.Interface.Default,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleSettings handles settings get/update.
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getSettings(w, r)
	case http.MethodPut:
		s.updateSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	settings := map[string]interface{}{
		"interface": map[string]interface{}{
			"current":   s.config.Interface.Default,
			"available": []string{}, // Will be populated by network module
		},
		"vlan": map[string]interface{}{
			"enabled": s.config.VLAN.Enabled,
			"id":      s.config.VLAN.ID,
		},
		"ip": map[string]interface{}{
			"mode": s.config.IP.Mode,
		},
		"thresholds": s.config.Thresholds,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Apply settings updates
	// This will be implemented when we add the settings management

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// handleInterfaces returns available network interfaces.
func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.netManager == nil {
		http.Error(w, "Network manager not available", http.StatusServiceUnavailable)
		return
	}

	s.netManager.RefreshInterfaces()
	interfaces := s.netManager.GetInterfaces()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interfaces)
}

// SetInterfaceRequest represents a request to change the current interface.
type SetInterfaceRequest struct {
	Interface string `json:"interface"`
}

// handleInterface handles GET/PUT for current interface.
func (s *Server) handleInterface(w http.ResponseWriter, r *http.Request) {
	if s.netManager == nil {
		http.Error(w, "Network manager not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"interface": s.netManager.GetCurrentInterface(),
		})
	case http.MethodPut:
		var req SetInterfaceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := s.netManager.SetCurrentInterface(req.Interface); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Also update discovery manager to use new interface
		if err := s.discoveryManager.SetInterface(req.Interface); err != nil {
			// Log but don't fail - discovery may not work without root
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "ok",
			"interface": req.Interface,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// LinkResponse represents the link status for an interface.
type LinkResponse struct {
	Interface   string   `json:"interface"`
	LinkUp      bool     `json:"linkUp"`
	Speed       string   `json:"speed"`
	Duplex      string   `json:"duplex"`
	Advertised  []string `json:"advertisedSpeeds"`
	MAC         string   `json:"mac"`
	MTU         int      `json:"mtu"`
	Addresses   []string `json:"addresses"`
}

// handleLink returns link status for the current interface.
func (s *Server) handleLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.netManager == nil {
		http.Error(w, "Network manager not available", http.StatusServiceUnavailable)
		return
	}

	s.netManager.RefreshInterfaces()
	currentIface := s.netManager.GetCurrentInterface()

	ifaceInfo, err := s.netManager.GetInterface(currentIface)
	if err != nil {
		http.Error(w, "Interface not found", http.StatusNotFound)
		return
	}

	linkStatus, _ := s.netManager.GetLinkStatus(currentIface)

	resp := LinkResponse{
		Interface: currentIface,
		LinkUp:    false, // Default to false
		MAC:       ifaceInfo.HardwareAddr,
		MTU:       ifaceInfo.MTU,
		Addresses: ifaceInfo.Addresses,
	}

	if linkStatus != nil {
		resp.LinkUp = linkStatus.LinkUp // Use the improved detection
		resp.Speed = linkStatus.Speed
		resp.Duplex = linkStatus.Duplex
		resp.Advertised = linkStatus.Advertised
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleExport exports current diagnostic data as JSON.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Get actual card data
	export := map[string]interface{}{
		"version":   "0.1.0",
		"timestamp": "2024-01-01T00:00:00Z",
		"interface": s.config.Interface.Default,
		"cards":     []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=netscope-export.json")
	json.NewEncoder(w).Encode(export)
}

// DiscoveryResponse contains all discovered neighbors.
type DiscoveryResponse struct {
	Running   bool                    `json:"running"`
	Neighbors []DiscoveryNeighborInfo `json:"neighbors"`
}

// DiscoveryNeighborInfo represents a discovered neighbor in the API.
type DiscoveryNeighborInfo struct {
	Protocol          string   `json:"protocol"`
	ChassisID         string   `json:"chassisId"`
	PortID            string   `json:"portId"`
	PortDescription   string   `json:"portDescription,omitempty"`
	SystemName        string   `json:"systemName,omitempty"`
	SystemDescription string   `json:"systemDescription,omitempty"`
	Capabilities      []string `json:"capabilities,omitempty"`
	ManagementAddress string   `json:"managementAddress,omitempty"`
	TTL               int      `json:"ttl"`
	LastSeen          string   `json:"lastSeen"`
	SourceMAC         string   `json:"sourceMAC"`
}

// handleDiscovery returns discovery protocol neighbors.
func (s *Server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
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
			LastSeen:          n.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
			SourceMAC:         n.SourceMAC,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
