package api

import (
	"encoding/json"
	"net/http"
	"time"
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
		"thresholds": map[string]interface{}{
			"dns": map[string]int64{
				"good":    s.config.Thresholds.DNS.Warning.Milliseconds(),
				"warning": s.config.Thresholds.DNS.Critical.Milliseconds(),
			},
			"gateway": map[string]int64{
				"good":    s.config.Thresholds.Ping.Warning.Milliseconds(),
				"warning": s.config.Thresholds.Ping.Critical.Milliseconds(),
			},
			"wifi": map[string]int{
				"good":    s.config.Thresholds.WiFi.Signal.Warning,
				"warning": s.config.Thresholds.WiFi.Signal.Critical,
			},
		},
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

	// Apply threshold updates
	if thresholds, ok := updates["thresholds"].(map[string]interface{}); ok {
		if dns, ok := thresholds["dns"].(map[string]interface{}); ok {
			if good, ok := dns["good"].(float64); ok {
				s.config.Thresholds.DNS.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := dns["warning"].(float64); ok {
				s.config.Thresholds.DNS.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if gateway, ok := thresholds["gateway"].(map[string]interface{}); ok {
			if good, ok := gateway["good"].(float64); ok {
				s.config.Thresholds.Ping.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := gateway["warning"].(float64); ok {
				s.config.Thresholds.Ping.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if wifi, ok := thresholds["wifi"].(map[string]interface{}); ok {
			if good, ok := wifi["good"].(float64); ok {
				s.config.Thresholds.WiFi.Signal.Warning = int(good)
			}
			if warning, ok := wifi["warning"].(float64); ok {
				s.config.Thresholds.WiFi.Signal.Critical = int(warning)
			}
		}
	}

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

// LinkResponse represents the link status for an interface (Layer 2 only).
type LinkResponse struct {
	Interface  string   `json:"interface"`
	LinkUp     bool     `json:"linkUp"`
	Speed      string   `json:"speed"`
	Duplex     string   `json:"duplex"`
	Advertised []string `json:"advertisedSpeeds"`
	MTU        int      `json:"mtu"`
	AutoNeg    bool     `json:"autoNeg"`
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
		LinkUp:    false,
		MTU:       ifaceInfo.MTU,
	}

	if linkStatus != nil {
		resp.LinkUp = linkStatus.LinkUp
		resp.Speed = linkStatus.Speed
		resp.Duplex = linkStatus.Duplex
		resp.Advertised = linkStatus.Advertised
		resp.AutoNeg = linkStatus.AutoNeg
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

	s.netManager.RefreshInterfaces()
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

	// Try to get DNS servers from system
	resp.DNS = getSystemDNS()

	// Add DHCP timing if available
	if s.dhcpMonitor != nil {
		if timing := s.dhcpMonitor.GetLastTiming(); timing != nil {
			ms := timing.ToMs()
			resp.Timing = &DHCPTimingInfo{
				Discover: ms.Discover,
				Offer:    ms.Offer,
				Request:  ms.Request,
				Total:    ms.Total,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
		if isLinkLocal(info.address) {
			info.scope = "link-local"
		} else if isUniqueLocal(info.address) {
			info.scope = "unique-local"
		} else {
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

// DNSLookupResult represents a DNS lookup result for the API.
type DNSLookupResult struct {
	Result string `json:"result"`
	Time   int64  `json:"time"` // ms
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// DNSResponse represents the DNS test results for the API.
type DNSResponse struct {
	Server       string           `json:"server"`
	TestHostname string           `json:"testHostname"`
	Forward      *DNSLookupResult `json:"forward,omitempty"`
	Reverse      *DNSLookupResult `json:"reverse,omitempty"`
}

// handleDNS performs DNS testing and returns results.
func (s *Server) handleDNS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.dnsTester == nil {
		http.Error(w, "DNS tester not available", http.StatusServiceUnavailable)
		return
	}

	// Perform DNS test
	result := s.dnsTester.Test(r.Context())

	resp := DNSResponse{
		Server:       result.Server,
		TestHostname: result.TestHostname,
	}

	if result.Forward != nil {
		resp.Forward = &DNSLookupResult{
			Result: result.Forward.Result,
			Time:   result.Forward.TimeMs,
			Status: string(result.Forward.Status),
			Error:  result.Forward.Error,
		}
	}

	if result.Reverse != nil {
		resp.Reverse = &DNSLookupResult{
			Result: result.Reverse.Result,
			Time:   result.Reverse.TimeMs,
			Status: string(result.Reverse.Status),
			Error:  result.Reverse.Error,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GatewayResponse represents the gateway ping test results for the API.
type GatewayResponse struct {
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

// handleGateway performs gateway ping testing and returns results.
func (s *Server) handleGateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.gatewayTester == nil {
		http.Error(w, "Gateway tester not available", http.StatusServiceUnavailable)
		return
	}

	// Perform gateway ping test
	stats := s.gatewayTester.Test()

	resp := GatewayResponse{
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
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// VLANResponse represents the VLAN information for the API.
type VLANResponse struct {
	NativeVlan  *int  `json:"nativeVlan"`
	TaggedVlans []int `json:"taggedVlans"`
	VoiceVlan   *int  `json:"voiceVlan"`
	Configured  struct {
		Enabled bool `json:"enabled"`
		ID      int  `json:"id"`
	} `json:"configured"`
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
		for _, n := range neighbors {
			// LLDP can carry VLAN information in TLVs
			if n.NativeVLAN > 0 {
				v := n.NativeVLAN
				nativeVlan = &v
			}
			if n.VoiceVLAN > 0 {
				v := n.VoiceVLAN
				voiceVlan = &v
			}
			break // Use first neighbor
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// WiFiResponse represents the Wi-Fi information for the API.
type WiFiResponse struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	Signal    int    `json:"signal"`    // dBm
	Channel   int    `json:"channel"`
	Frequency int    `json:"frequency"` // MHz
	Security  string `json:"security"`
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"wireless": false,
			"message":  "Current interface is not a wireless adapter",
		})
		return
	}

	info := s.wifiManager.GetInfo()
	if info == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CableResponse represents the cable test results for the API.
type CableResponse struct {
	Supported bool     `json:"supported"`
	Length    *float64 `json:"length,omitempty"` // meters
	Status    string   `json:"status"`
	Faults    []string `json:"faults"`
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
