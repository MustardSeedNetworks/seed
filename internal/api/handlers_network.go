// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/krisarmstrong/luminetiq/internal/config"
	"github.com/krisarmstrong/luminetiq/internal/dhcp"
	"github.com/krisarmstrong/luminetiq/internal/network"
	"github.com/krisarmstrong/luminetiq/internal/validation"
	"github.com/krisarmstrong/luminetiq/internal/vlan"
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

// WiFiResponse represents the Wi-Fi information for the API.
type WiFiResponse struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	Signal    int    `json:"signal"` // dBm
	Channel   int    `json:"channel"`
	Frequency int    `json:"frequency"` // MHz
	Security  string `json:"security"`
}

// WiFiSettingsResponse represents the WiFi configuration settings.
type WiFiSettingsResponse struct {
	Interface     string   `json:"interface"`
	AvailableWiFi []string `json:"availableWifi"`
	IsWireless    bool     `json:"isWireless"`
}

// CableResponse represents the cable test results for the API.
type CableResponse struct {
	Supported bool     `json:"supported"`
	Length    *float64 `json:"length,omitempty"` // meters
	Status    string   `json:"status"`
	Faults    []string `json:"faults"`
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

func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.netManager == nil {
		http.Error(w, "Network manager not available", http.StatusServiceUnavailable)
		return
	}

	if err := s.netManager.RefreshInterfaces(); err != nil {
		http.Error(w, "Failed to refresh interfaces", http.StatusInternalServerError)
		return
	}
	interfaces := s.netManager.GetInterfaces()

	sendJSONResponse(w, http.StatusOK, interfaces)
}

// handleInterface handles GET/PUT for current interface.
func (s *Server) handleInterface(w http.ResponseWriter, r *http.Request) {
	if s.netManager == nil {
		http.Error(w, "Network manager not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, http.StatusOK, map[string]string{
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
			log.Printf("failed to set discovery interface: %v", err)
		}

		// Update device discovery (ARP/protocol scans)
		if s.deviceDiscovery != nil {
			if err := s.deviceDiscovery.SetInterface(req.Interface); err != nil {
				log.Printf("failed to set device discovery interface: %v", err)
			}
		}

		// Update WiFi manager interface and check if wireless
		if s.wifiManager != nil {
			s.wifiManager.SetInterface(req.Interface)
		}

		// Update link monitor interface
		if s.linkMonitor != nil {
			s.linkMonitor.SetInterface(req.Interface)
		}

		// Check if new interface is wireless
		isWireless := false
		if s.wifiManager != nil {
			isWireless = s.wifiManager.IsWireless()
		}

		sendJSONResponse(w, http.StatusOK, map[string]interface{}{
			"status":     "ok",
			"interface":  req.Interface,
			"isWireless": isWireless,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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

	if err := s.netManager.RefreshInterfaces(); err != nil {
		http.Error(w, "Failed to refresh interfaces", http.StatusInternalServerError)
		return
	}
	currentIface := s.netManager.GetCurrentInterface()

	ifaceInfo, err := s.netManager.GetInterface(currentIface)
	if err != nil {
		http.Error(w, "Interface not found", http.StatusNotFound)
		return
	}

	linkStatus, err := s.netManager.GetLinkStatus(currentIface)
	if err != nil {
		log.Printf("failed to get link status for %s: %v", currentIface, err)
	}

	resp := LinkResponse{
		Interface: currentIface,
		LinkUp:    false,
		MTU:       ifaceInfo.MTU,
	}

	if linkStatus != nil {
		resp.LinkUp = linkStatus.LinkUp
		resp.Carrier = linkStatus.Carrier
		resp.HasIP = linkStatus.HasIP
		resp.Speed = linkStatus.Speed
		resp.Duplex = linkStatus.Duplex
		resp.Advertised = linkStatus.Advertised
		resp.AutoNeg = linkStatus.AutoNeg
	}

	// Add link flap history from monitor
	if s.linkMonitor != nil {
		resp.FlapCount24h = s.linkMonitor.GetFlapCount24h()
		resp.UptimeMs = s.linkMonitor.GetUptime().Milliseconds()

		// Convert history events to API format
		history := s.linkMonitor.GetHistory()
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

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleIPConfig returns IP configuration for the current interface.
func (s *Server) handleIPConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.netManager == nil {
		http.Error(w, "Network manager not available", http.StatusServiceUnavailable)
		return
	}

	if err := s.netManager.RefreshInterfaces(); err != nil {
		http.Error(w, "Failed to refresh interfaces", http.StatusInternalServerError)
		return
	}
	currentIface := s.netManager.GetCurrentInterface()

	ifaceInfo, err := s.netManager.GetInterface(currentIface)
	if err != nil {
		http.Error(w, "Interface not found", http.StatusNotFound)
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

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleVLAN returns VLAN information for the current interface.
func (s *Server) handleVLAN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.vlanManager == nil {
		http.Error(w, "VLAN manager not available", http.StatusServiceUnavailable)
		return
	}

	// Get VLAN info from LLDP/CDP if available
	var nativeVlan, voiceVlan *int
	if s.discoveryManager != nil {
		neighbors := s.discoveryManager.GetNeighbors()
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

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleVLANTraffic returns VLAN traffic statistics from frame capture.
func (s *Server) handleVLANTraffic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.vlanTrafficMonitor == nil {
		http.Error(w, "VLAN traffic monitor not available", http.StatusServiceUnavailable)
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

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleVLANInterface handles POST (create) and DELETE (remove) for VLAN subinterfaces.
func (s *Server) handleVLANInterface(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createVLANInterface(w, r)
	case http.MethodDelete:
		s.deleteVLANInterface(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// parseVLANRequest parses and validates a VLAN interface request.
// Returns the validated interface name, VLAN ID, and success boolean.
func (s *Server) parseVLANRequest(w http.ResponseWriter, r *http.Request) (iface string, vlanID int, ok bool) {
	var req VLANInterfaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return "", 0, false
	}

	// Validate VLAN ID (fixes #522)
	if err := validation.ValidateVLANID(req.VlanID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", 0, false
	}

	// Use current interface if not specified
	iface = req.Interface
	if iface == "" {
		iface = s.netManager.GetCurrentInterface()
	}

	// Validate interface name
	if err := validation.ValidateInterface(iface); err != nil {
		http.Error(w, fmt.Sprintf("Invalid interface: %v", err), http.StatusBadRequest)
		return "", 0, false
	}

	vlanID = req.VlanID
	ok = true
	return
}

// createVLANInterface creates an 802.1Q VLAN subinterface.
func (s *Server) createVLANInterface(w http.ResponseWriter, r *http.Request) {
	iface, vlanID, ok := s.parseVLANRequest(w, r)
	if !ok {
		return
	}

	if err := vlan.CreateVlanInterface(iface, vlanID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create VLAN interface: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "VLAN interface created",
		"interface": iface,
		"vlanId":    vlanID,
	})
}

// deleteVLANInterface removes an 802.1Q VLAN subinterface.
func (s *Server) deleteVLANInterface(w http.ResponseWriter, r *http.Request) {
	iface, vlanID, ok := s.parseVLANRequest(w, r)
	if !ok {
		return
	}

	if err := vlan.DeleteVlanInterface(iface, vlanID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete VLAN interface: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "VLAN interface deleted",
		"interface": iface,
		"vlanId":    vlanID,
	})
}

// handleWiFiSettings handles GET/PUT for WiFi settings.
func (s *Server) handleWiFiSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getWiFiSettings(w, r)
	case http.MethodPut:
		s.updateWiFiSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getWiFiSettings(w http.ResponseWriter, _ *http.Request) {
	// Get configured WiFi interface (or fall back to current)
	wifiIface := s.config.Interface.WiFi
	if wifiIface == "" {
		wifiIface = s.config.Interface.Default
	}

	// Get list of available wireless interfaces
	availableWiFi := []string{}
	if s.netManager != nil {
		for _, iface := range s.netManager.GetInterfaces() {
			if s.netManager.IsWireless(iface.Name) {
				availableWiFi = append(availableWiFi, iface.Name)
			}
		}
	}

	resp := WiFiSettingsResponse{
		Interface:     wifiIface,
		AvailableWiFi: availableWiFi,
		IsWireless:    s.wifiManager != nil && s.wifiManager.IsWireless(),
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

func (s *Server) updateWiFiSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Interface string `json:"interface"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Lock config for write access
	s.config.Lock()
	defer s.config.Unlock()

	// Update WiFi interface in config
	s.config.Interface.WiFi = req.Interface

	// Update WiFi manager to use new interface
	if s.wifiManager != nil && req.Interface != "" {
		s.wifiManager.SetInterface(req.Interface)
	}

	// Save config
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "WiFi settings updated",
	})
}

// handleWiFi returns Wi-Fi information for the current interface.
func (s *Server) handleWiFi(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.wifiManager == nil {
		http.Error(w, "Wi-Fi manager not available", http.StatusServiceUnavailable)
		return
	}

	// Check if interface is wireless
	if !s.wifiManager.IsWireless() {
		sendJSONResponse(w, http.StatusOK, map[string]interface{}{
			"wireless": false,
			"message":  "Current interface is not a wireless adapter",
		})
		return
	}

	info := s.wifiManager.GetInfo()
	if info == nil {
		w.Header().Set("Content-Type", "application/json")
		sendJSONResponse(w, http.StatusOK, map[string]interface{}{
			"wireless":  true,
			"connected": false,
			"message":   "Not connected to a wireless network",
		})
		return
	}

	resp := WiFiResponse{
		SSID:      info.SSID,
		BSSID:     info.BSSID,
		Signal:    info.Signal,
		Channel:   info.Channel,
		Frequency: info.Frequency,
		Security:  info.Security,
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleIPSettings handles GET/PUT for IP configuration settings.
func (s *Server) handleIPSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleIPSettingsGet(w, r)
	case http.MethodPut:
		s.handleIPSettingsPut(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleIPSettingsPut updates the IP configuration settings.
func (s *Server) handleIPSettingsPut(w http.ResponseWriter, r *http.Request) {
	var req IPSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate mode
	if req.Mode != "dhcp" && req.Mode != "static" {
		http.Error(w, "Mode must be 'dhcp' or 'static'", http.StatusBadRequest)
		return
	}

	// Lock config for write access
	s.config.Lock()
	defer s.config.Unlock()

	currentIface := s.netManager.GetCurrentInterface()

	if req.Mode == "static" {
		// Apply static IP configuration
		cfg := &network.StaticIPConfig{
			Address: req.Address,
			Netmask: req.Netmask,
			Gateway: req.Gateway,
			DNS:     req.DNS,
		}

		if err := s.netManager.ConfigureStaticIP(currentIface, cfg); err != nil {
			http.Error(w, fmt.Sprintf("Failed to configure static IP: %v", err), http.StatusInternalServerError)
			return
		}

		// Update config
		s.config.IP.Mode = "static"
		s.config.IP.Static = &config.StaticIP{
			Address: req.Address,
			Netmask: req.Netmask,
			Gateway: req.Gateway,
			DNS:     req.DNS,
		}
	} else {
		// Switch to DHCP
		if err := s.netManager.ConfigureDHCP(currentIface); err != nil {
			http.Error(w, fmt.Sprintf("Failed to configure DHCP: %v", err), http.StatusInternalServerError)
			return
		}

		// Update config
		s.config.IP.Mode = "dhcp"
		s.config.IP.Static = nil
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		// Log but don't fail - the config was applied
		log.Printf("Warning: Failed to save config: %v", err)
	}

	// Refresh interface data
	if err := s.netManager.RefreshInterfaces(); err != nil {
		http.Error(w, "Failed to refresh interfaces", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "IP configuration updated",
	})
}

// handleSetMTU handles POST requests to set interface MTU.
func (s *Server) handleSetMTU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetMTURequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use current interface if not specified
	iface := req.Interface
	if iface == "" {
		iface = s.netManager.GetCurrentInterface()
	}

	// Set the MTU
	if err := s.netManager.SetMTU(iface, req.MTU); err != nil {
		http.Error(w, fmt.Sprintf("Failed to set MTU: %v", err), http.StatusInternalServerError)
		return
	}

	// Refresh interface data
	if err := s.netManager.RefreshInterfaces(); err != nil {
		log.Printf("Warning: Failed to refresh interfaces after MTU change: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "MTU updated",
		"interface": iface,
		"mtu":       req.MTU,
	})
}

// handleCable performs a cable test and returns results.
func (s *Server) handleCable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.cableTester == nil {
		http.Error(w, "Cable tester not available", http.StatusServiceUnavailable)
		return
	}

	result := s.cableTester.Test()

	resp := CableResponse{
		Supported: result.Supported,
		Length:    result.Length,
		Status:    string(result.Status),
		Faults:    result.Faults,
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// ============================================================================
// Helper Functions
// ============================================================================

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
			resp.Mode = "dhcp"
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
	if s.dhcpMonitor == nil {
		return
	}
	timing := s.dhcpMonitor.GetLastTiming()
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
			result = result*10 + int(c-'0')
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
	if len(addr) < 2 {
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
