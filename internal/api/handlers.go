package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/krisarmstrong/netscope/internal/config"
	"github.com/krisarmstrong/netscope/internal/dhcp"
	"github.com/krisarmstrong/netscope/internal/network"
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
	Status     string `json:"status"`
	Version    string `json:"version"`
	Uptime     int64  `json:"uptime"`
	Interface  string `json:"interface"`
	IsWireless bool   `json:"isWireless"`
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

	// Check if current interface is wireless
	isWireless := false
	if s.wifiManager != nil {
		isWireless = s.wifiManager.IsWireless()
	}

	resp := StatusResponse{
		Status:     "ok",
		Version:    "0.7.3",
		Interface:  s.config.Interface.Default,
		IsWireless: isWireless,
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
			"customPing": map[string]int64{
				"good":    s.config.Thresholds.CustomTests.Ping.Warning.Milliseconds(),
				"warning": s.config.Thresholds.CustomTests.Ping.Critical.Milliseconds(),
			},
			"customTcp": map[string]int64{
				"good":    s.config.Thresholds.CustomTests.TCP.Warning.Milliseconds(),
				"warning": s.config.Thresholds.CustomTests.TCP.Critical.Milliseconds(),
			},
			"customHttp": map[string]int64{
				"good":    s.config.Thresholds.CustomTests.HTTP.Warning.Milliseconds(),
				"warning": s.config.Thresholds.CustomTests.HTTP.Critical.Milliseconds(),
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
		if customPing, ok := thresholds["customPing"].(map[string]interface{}); ok {
			if good, ok := customPing["good"].(float64); ok {
				s.config.Thresholds.CustomTests.Ping.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := customPing["warning"].(float64); ok {
				s.config.Thresholds.CustomTests.Ping.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if customTcp, ok := thresholds["customTcp"].(map[string]interface{}); ok {
			if good, ok := customTcp["good"].(float64); ok {
				s.config.Thresholds.CustomTests.TCP.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := customTcp["warning"].(float64); ok {
				s.config.Thresholds.CustomTests.TCP.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if customHttp, ok := thresholds["customHttp"].(map[string]interface{}); ok {
			if good, ok := customHttp["good"].(float64); ok {
				s.config.Thresholds.CustomTests.HTTP.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := customHttp["warning"].(float64); ok {
				s.config.Thresholds.CustomTests.HTTP.Critical = time.Duration(warning) * time.Millisecond
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ok",
			"interface":  req.Interface,
			"isWireless": isWireless,
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

	// Get DHCP lease info (server, gateway, lease time)
	if leaseInfo, err := dhcp.GetLeaseInfo(currentIface); err == nil && leaseInfo != nil {
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
		// Use DNS from lease if available
		if len(leaseInfo.DNS) > 0 {
			resp.DNS = leaseInfo.DNS
		}
	}

	// Fallback: Try to get DNS servers from system if not from lease
	if len(resp.DNS) == 0 {
		resp.DNS = getSystemDNS()
	}

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

// ExportData represents the full diagnostic export.
type ExportData struct {
	Version   string                 `json:"version"`
	Timestamp string                 `json:"timestamp"`
	Device    ExportDeviceInfo       `json:"device"`
	Cards     map[string]interface{} `json:"cards"`
}

// ExportDeviceInfo contains device information.
type ExportDeviceInfo struct {
	Interface string `json:"interface"`
	MAC       string `json:"mac,omitempty"`
	IPMode    string `json:"ipMode"`
}

// handleExport exports current diagnostic data as JSON.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	currentIface := s.netManager.GetCurrentInterface()
	s.netManager.RefreshInterfaces()

	// Get interface info
	var mac string
	if ifaceInfo, err := s.netManager.GetInterface(currentIface); err == nil {
		mac = ifaceInfo.HardwareAddr
	}

	export := ExportData{
		Version:   "0.7.3",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Device: ExportDeviceInfo{
			Interface: currentIface,
			MAC:       mac,
			IPMode:    s.config.IP.Mode,
		},
		Cards: make(map[string]interface{}),
	}

	// Collect Link data
	if linkStatus, err := s.netManager.GetLinkStatus(currentIface); err == nil {
		export.Cards["link"] = map[string]interface{}{
			"linkUp":  linkStatus.LinkUp,
			"speed":   linkStatus.Speed,
			"duplex":  linkStatus.Duplex,
			"autoNeg": linkStatus.AutoNeg,
		}
	}

	// Collect IP config data
	if ifaceInfo, err := s.netManager.GetInterface(currentIface); err == nil {
		ipData := map[string]interface{}{
			"addresses": ifaceInfo.Addresses,
		}
		if leaseInfo, err := dhcp.GetLeaseInfo(currentIface); err == nil && leaseInfo != nil {
			ipData["dhcpServer"] = leaseInfo.DHCPServer
			ipData["gateway"] = leaseInfo.Gateway
			ipData["leaseTime"] = leaseInfo.LeaseTime
			ipData["dns"] = leaseInfo.DNS
		}
		export.Cards["ipConfig"] = ipData
	}

	// Collect Discovery data
	if s.discoveryManager != nil {
		neighbors := s.discoveryManager.GetNeighbors()
		neighborList := make([]map[string]interface{}, 0, len(neighbors))
		for _, n := range neighbors {
			neighborList = append(neighborList, map[string]interface{}{
				"protocol":          n.Protocol,
				"systemName":        n.SystemName,
				"portId":            n.PortID,
				"portDescription":   n.PortDescription,
				"managementAddress": n.ManagementAddress,
			})
		}
		export.Cards["switch"] = map[string]interface{}{
			"running":   s.discoveryManager.IsRunning(),
			"neighbors": neighborList,
		}
	}

	// Collect DNS data
	if s.dnsTester != nil {
		ctx := r.Context()
		result := s.dnsTester.Test(ctx)
		dnsData := map[string]interface{}{
			"server":       result.Server,
			"testHostname": result.TestHostname,
		}
		if result.Forward != nil {
			dnsData["forward"] = map[string]interface{}{
				"result":   result.Forward.Resolved,
				"time":     result.Forward.Time.Milliseconds(),
				"status":   result.Forward.Status,
				"error":    result.Forward.Error,
			}
		}
		if result.Reverse != nil {
			dnsData["reverse"] = map[string]interface{}{
				"result":   result.Reverse.Resolved,
				"time":     result.Reverse.Time.Milliseconds(),
				"status":   result.Reverse.Status,
				"error":    result.Reverse.Error,
			}
		}
		export.Cards["dns"] = dnsData
	}

	// Collect Gateway data
	if s.gatewayTester != nil {
		stats := s.gatewayTester.GetStats()
		export.Cards["gateway"] = map[string]interface{}{
			"gateway":     stats.Gateway,
			"reachable":   stats.Reachable,
			"sent":        stats.Sent,
			"received":    stats.Received,
			"lossPercent": stats.LossPercent,
			"avgTime":     stats.AvgTime,
			"status":      stats.Status,
		}
	}

	// Collect VLAN data
	if s.vlanManager != nil {
		vlanInfo := s.vlanManager.GetInfo()
		export.Cards["vlan"] = map[string]interface{}{
			"nativeVlan":  vlanInfo.NativeVlan,
			"taggedVlans": vlanInfo.TaggedVlans,
			"voiceVlan":   vlanInfo.VoiceVlan,
			"configured":  vlanInfo.Configured,
		}
	}

	// Collect WiFi data if wireless
	if s.netManager.IsWireless(currentIface) && s.wifiManager != nil {
		wifiInfo := s.wifiManager.GetInfo()
		if wifiInfo.SSID != "" {
			export.Cards["wifi"] = map[string]interface{}{
				"ssid":      wifiInfo.SSID,
				"bssid":     wifiInfo.BSSID,
				"signal":    wifiInfo.Signal,
				"channel":   wifiInfo.Channel,
				"frequency": wifiInfo.Frequency,
				"security":  wifiInfo.Security,
			}
		}
	}

	// Collect Cable test data
	if s.cableTester != nil {
		cableResult := s.cableTester.Test()
		export.Cards["cable"] = map[string]interface{}{
			"supported": cableResult.Supported,
			"length":    cableResult.Length,
			"status":    cableResult.Status,
			"faults":    cableResult.Faults,
		}
	}

	// Pretty print JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=netscope-export.json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(export)
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
	Result   string   `json:"result"`
	Time     int64    `json:"time"` // ms (deprecated, use timeMs)
	TimeMs   int64    `json:"timeMs"`
	Status   string   `json:"status"`
	Error    string   `json:"error,omitempty"`
	Resolved []string `json:"resolved,omitempty"`
}

// DNSResponse represents the DNS test results for the API.
type DNSResponse struct {
	Server       string           `json:"server"`
	Servers      []string         `json:"servers"` // All configured DNS servers
	TestHostname string           `json:"testHostname"`
	Forward      *DNSLookupResult `json:"forward,omitempty"`
	ForwardIpv6  *DNSLookupResult `json:"forwardIpv6,omitempty"`
	Reverse      *DNSLookupResult `json:"reverse,omitempty"`
	ReverseIpv6  *DNSLookupResult `json:"reverseIpv6,omitempty"`
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
		Servers:      result.Servers,
		TestHostname: result.TestHostname,
	}

	if result.Forward != nil {
		resp.Forward = &DNSLookupResult{
			Result:   result.Forward.Result,
			Time:     result.Forward.TimeMs,
			TimeMs:   result.Forward.TimeMs,
			Status:   string(result.Forward.Status),
			Error:    result.Forward.Error,
			Resolved: result.Forward.Resolved,
		}
	}

	if result.ForwardIPv6 != nil {
		resp.ForwardIpv6 = &DNSLookupResult{
			Result:   result.ForwardIPv6.Result,
			Time:     result.ForwardIPv6.TimeMs,
			TimeMs:   result.ForwardIPv6.TimeMs,
			Status:   string(result.ForwardIPv6.Status),
			Error:    result.ForwardIPv6.Error,
			Resolved: result.ForwardIPv6.Resolved,
		}
	}

	if result.Reverse != nil {
		resp.Reverse = &DNSLookupResult{
			Result:   result.Reverse.Result,
			Time:     result.Reverse.TimeMs,
			TimeMs:   result.Reverse.TimeMs,
			Status:   string(result.Reverse.Status),
			Error:    result.Reverse.Error,
			Resolved: result.Reverse.Resolved,
		}
	}

	if result.ReverseIPv6 != nil {
		resp.ReverseIpv6 = &DNSLookupResult{
			Result:   result.ReverseIPv6.Result,
			Time:     result.ReverseIPv6.TimeMs,
			TimeMs:   result.ReverseIPv6.TimeMs,
			Status:   string(result.ReverseIPv6.Status),
			Error:    result.ReverseIPv6.Error,
			Resolved: result.ReverseIPv6.Resolved,
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

// WiFiSettingsResponse represents the WiFi configuration settings.
type WiFiSettingsResponse struct {
	Interface       string   `json:"interface"`
	AvailableWiFi   []string `json:"availableWifi"`
	IsWireless      bool     `json:"isWireless"`
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

func (s *Server) getWiFiSettings(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) updateWiFiSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Interface string `json:"interface"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
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
func (s *Server) handleIPSettingsGet(w http.ResponseWriter, r *http.Request) {
	resp := IPSettingsResponse{
		Mode: s.config.IP.Mode,
	}

	if s.config.IP.Static != nil {
		resp.Address = s.config.IP.Static.Address
		resp.Netmask = s.config.IP.Static.Netmask
		resp.Gateway = s.config.IP.Static.Gateway
		resp.DNS = s.config.IP.Static.DNS
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
	s.netManager.RefreshInterfaces()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "IP configuration updated",
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// TestsSettingsResponse represents the custom tests configuration.
type TestsSettingsResponse struct {
	DNSHostname   string                    `json:"dnsHostname"`
	PingTargets   []PingTargetResponse      `json:"pingTargets"`
	TCPPorts      []TCPPortResponse         `json:"tcpPorts"`
	UDPPorts      []UDPPortResponse         `json:"udpPorts"`
	HTTPEndpoints []HTTPEndpointResponse    `json:"httpEndpoints"`
	Speedtest     SpeedtestSettingsResponse `json:"speedtest"`
}

type PingTargetResponse struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Enabled bool   `json:"enabled"`
}

type TCPPortResponse struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

type UDPPortResponse struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

type HTTPEndpointResponse struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expectedStatus"`
	Enabled        bool   `json:"enabled"`
}

type SpeedtestSettingsResponse struct {
	ServerID      string `json:"serverId"`
	AutoRunOnLink bool   `json:"autoRunOnLink"`
}

// handleTestsSettings handles GET/PUT for custom tests settings.
func (s *Server) handleTestsSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getTestsSettings(w, r)
	case http.MethodPut:
		s.updateTestsSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getTestsSettings(w http.ResponseWriter, r *http.Request) {
	resp := TestsSettingsResponse{
		DNSHostname:   s.config.DNS.TestHostname,
		PingTargets:   make([]PingTargetResponse, 0, len(s.config.Tests.PingTargets)),
		TCPPorts:      make([]TCPPortResponse, 0, len(s.config.Tests.TCPPorts)),
		UDPPorts:      make([]UDPPortResponse, 0, len(s.config.Tests.UDPPorts)),
		HTTPEndpoints: make([]HTTPEndpointResponse, 0, len(s.config.Tests.HTTPEndpoints)),
		Speedtest: SpeedtestSettingsResponse{
			ServerID:      s.config.Speedtest.ServerID,
			AutoRunOnLink: s.config.Speedtest.AutoRunOnLink,
		},
	}

	for _, p := range s.config.Tests.PingTargets {
		resp.PingTargets = append(resp.PingTargets, PingTargetResponse{
			Name:    p.Name,
			Host:    p.Host,
			Enabled: p.Enabled,
		})
	}

	for _, t := range s.config.Tests.TCPPorts {
		resp.TCPPorts = append(resp.TCPPorts, TCPPortResponse{
			Name:    t.Name,
			Host:    t.Host,
			Port:    t.Port,
			Enabled: t.Enabled,
		})
	}

	for _, u := range s.config.Tests.UDPPorts {
		resp.UDPPorts = append(resp.UDPPorts, UDPPortResponse{
			Name:    u.Name,
			Host:    u.Host,
			Port:    u.Port,
			Enabled: u.Enabled,
		})
	}

	for _, h := range s.config.Tests.HTTPEndpoints {
		resp.HTTPEndpoints = append(resp.HTTPEndpoints, HTTPEndpointResponse{
			Name:           h.Name,
			URL:            h.URL,
			ExpectedStatus: h.ExpectedStatus,
			Enabled:        h.Enabled,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) updateTestsSettings(w http.ResponseWriter, r *http.Request) {
	var req TestsSettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update DNS hostname
	if req.DNSHostname != "" {
		s.config.DNS.TestHostname = req.DNSHostname
		// Update the DNS tester with the new hostname
		if s.dnsTester != nil {
			s.dnsTester.SetTestHostname(req.DNSHostname)
		}
	}

	// Update ping targets
	s.config.Tests.PingTargets = make([]config.PingTarget, 0, len(req.PingTargets))
	for _, p := range req.PingTargets {
		s.config.Tests.PingTargets = append(s.config.Tests.PingTargets, config.PingTarget{
			Name:    p.Name,
			Host:    p.Host,
			Enabled: p.Enabled,
		})
	}

	// Update TCP ports
	s.config.Tests.TCPPorts = make([]config.TCPPortTest, 0, len(req.TCPPorts))
	for _, t := range req.TCPPorts {
		s.config.Tests.TCPPorts = append(s.config.Tests.TCPPorts, config.TCPPortTest{
			Name:    t.Name,
			Host:    t.Host,
			Port:    t.Port,
			Enabled: t.Enabled,
		})
	}

	// Update UDP ports
	s.config.Tests.UDPPorts = make([]config.UDPPortTest, 0, len(req.UDPPorts))
	for _, u := range req.UDPPorts {
		s.config.Tests.UDPPorts = append(s.config.Tests.UDPPorts, config.UDPPortTest{
			Name:    u.Name,
			Host:    u.Host,
			Port:    u.Port,
			Enabled: u.Enabled,
		})
	}

	// Update HTTP endpoints
	s.config.Tests.HTTPEndpoints = make([]config.HTTPEndpoint, 0, len(req.HTTPEndpoints))
	for _, h := range req.HTTPEndpoints {
		// Auto-prefix URL with https:// if missing scheme
		url := h.URL
		if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}
		s.config.Tests.HTTPEndpoints = append(s.config.Tests.HTTPEndpoints, config.HTTPEndpoint{
			Name:           h.Name,
			URL:            url,
			ExpectedStatus: h.ExpectedStatus,
			Enabled:        h.Enabled,
		})
	}

	// Update speedtest settings
	s.config.Speedtest.ServerID = req.Speedtest.ServerID
	s.config.Speedtest.AutoRunOnLink = req.Speedtest.AutoRunOnLink
	if s.speedtestTester != nil {
		s.speedtestTester.SetServerID(req.Speedtest.ServerID)
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Tests settings updated",
	})
}

// CustomTestResult represents the result of a single custom test.
type CustomTestResult struct {
	Name        string  `json:"name"`
	Host        string  `json:"host"`
	Port        int     `json:"port,omitempty"`
	URL         string  `json:"url,omitempty"`
	Success     bool    `json:"success"`
	Latency     float64 `json:"latency"` // ms
	Error       string  `json:"error,omitempty"`
	Status      int     `json:"status,omitempty"`     // HTTP status code
	TestStatus  string  `json:"testStatus,omitempty"` // success, warning, error
	// Extended ping fields
	PacketLoss  float64 `json:"packetLoss,omitempty"`  // Percentage
	Jitter      float64 `json:"jitter,omitempty"`      // ms
	MinLatency  float64 `json:"minLatency,omitempty"`  // ms
	MaxLatency  float64 `json:"maxLatency,omitempty"`  // ms
	// Certificate fields
	CertDaysLeft   int    `json:"certDaysLeft,omitempty"`   // Days until cert expires
	CertStatus     string `json:"certStatus,omitempty"`     // success, warning, error
	CertExpiry     string `json:"certExpiry,omitempty"`     // Expiry date string
	CertCommonName string `json:"certCommonName,omitempty"` // Certificate CN
}

// CustomTestsResult represents results from all custom tests.
type CustomTestsResult struct {
	PingResults []CustomTestResult `json:"pingResults"`
	TCPResults  []CustomTestResult `json:"tcpResults"`
	UDPResults  []CustomTestResult `json:"udpResults"`
	HTTPResults []CustomTestResult `json:"httpResults"`
	HasTests    bool               `json:"hasTests"`
}

// handleCustomTests runs all configured custom tests and returns results.
func (s *Server) handleCustomTests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result := CustomTestsResult{
		PingResults: make([]CustomTestResult, 0),
		TCPResults:  make([]CustomTestResult, 0),
		UDPResults:  make([]CustomTestResult, 0),
		HTTPResults: make([]CustomTestResult, 0),
		HasTests:    false,
	}

	// Check if there are any tests configured
	if len(s.config.Tests.PingTargets) > 0 || len(s.config.Tests.TCPPorts) > 0 ||
		len(s.config.Tests.UDPPorts) > 0 || len(s.config.Tests.HTTPEndpoints) > 0 {
		result.HasTests = true
	}

	// Get thresholds
	pingThreshold := s.config.Thresholds.CustomTests.Ping
	tcpThreshold := s.config.Thresholds.CustomTests.TCP
	udpThreshold := s.config.Thresholds.CustomTests.UDP
	httpThreshold := s.config.Thresholds.CustomTests.HTTP
	certThreshold := s.config.Thresholds.CustomTests.CertExpiry

	// Run extended ping tests (with packet loss and jitter)
	for _, target := range s.config.Tests.PingTargets {
		if !target.Enabled {
			continue
		}
		name := target.Name
		if name == "" {
			name = target.Host
		}

		testResult := CustomTestResult{
			Name: name,
			Host: target.Host,
		}

		// Run extended ping (5 pings for stats)
		pingStats, err := runExtendedPing(target.Host, 5)
		if err != nil {
			testResult.Success = false
			testResult.Error = err.Error()
			testResult.TestStatus = "error"
		} else {
			testResult.Success = pingStats.PacketLoss < 100
			testResult.Latency = pingStats.AvgLatency
			testResult.MinLatency = pingStats.MinLatency
			testResult.MaxLatency = pingStats.MaxLatency
			testResult.PacketLoss = pingStats.PacketLoss
			testResult.Jitter = pingStats.Jitter

			// Determine status based on latency or packet loss
			if pingStats.PacketLoss > 50 {
				testResult.TestStatus = "error"
			} else if pingStats.PacketLoss > 10 {
				testResult.TestStatus = "warning"
			} else {
				testResult.TestStatus = getTestStatus(pingStats.AvgLatency, pingThreshold.Warning.Milliseconds(), pingThreshold.Critical.Milliseconds())
			}
		}
		result.PingResults = append(result.PingResults, testResult)
	}

	// Run TCP port tests
	for _, target := range s.config.Tests.TCPPorts {
		if !target.Enabled {
			continue
		}
		name := target.Name
		if name == "" {
			name = fmt.Sprintf("%s:%d", target.Host, target.Port)
		}

		testResult := CustomTestResult{
			Name: name,
			Host: target.Host,
			Port: target.Port,
		}

		latency, err := runTCPTest(target.Host, target.Port)
		if err != nil {
			testResult.Success = false
			testResult.Error = err.Error()
			testResult.TestStatus = "error"
		} else {
			testResult.Success = true
			testResult.Latency = latency
			testResult.TestStatus = getTestStatus(latency, tcpThreshold.Warning.Milliseconds(), tcpThreshold.Critical.Milliseconds())
		}
		result.TCPResults = append(result.TCPResults, testResult)
	}

	// Run UDP port tests
	for _, target := range s.config.Tests.UDPPorts {
		if !target.Enabled {
			continue
		}
		name := target.Name
		if name == "" {
			name = fmt.Sprintf("%s:%d", target.Host, target.Port)
		}

		testResult := CustomTestResult{
			Name: name,
			Host: target.Host,
			Port: target.Port,
		}

		latency, err := runUDPTest(target.Host, target.Port)
		if err != nil {
			testResult.Success = false
			testResult.Error = err.Error()
			testResult.TestStatus = "error"
		} else {
			testResult.Success = true
			testResult.Latency = latency
			testResult.TestStatus = getTestStatus(latency, udpThreshold.Warning.Milliseconds(), udpThreshold.Critical.Milliseconds())
		}
		result.UDPResults = append(result.UDPResults, testResult)
	}

	// Run HTTP endpoint tests with certificate expiry checking
	for _, endpoint := range s.config.Tests.HTTPEndpoints {
		if !endpoint.Enabled {
			continue
		}

		// Auto-prefix URL with https:// if missing scheme
		url := endpoint.URL
		if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}

		name := endpoint.Name
		if name == "" {
			name = endpoint.URL // Show original URL in name
		}

		testResult := CustomTestResult{
			Name: name,
			URL:  url,
		}

		statusCode, latency, err := runHTTPTest(url, endpoint.ExpectedStatus)
		testResult.Status = statusCode
		testResult.Latency = latency
		if err != nil {
			testResult.Success = false
			testResult.Error = err.Error()
			testResult.TestStatus = "error"
		} else {
			testResult.Success = true
			testResult.TestStatus = getTestStatus(latency, httpThreshold.Warning.Milliseconds(), httpThreshold.Critical.Milliseconds())
		}

		// Check certificate expiry for HTTPS URLs
		if strings.HasPrefix(url, "https://") {
			certInfo := checkCertExpiry(url, certThreshold.Warning, certThreshold.Critical)
			testResult.CertDaysLeft = certInfo.DaysLeft
			testResult.CertStatus = certInfo.Status
			testResult.CertExpiry = certInfo.ExpiryDate
			testResult.CertCommonName = certInfo.CommonName

			// Upgrade test status if cert is in bad shape
			if certInfo.Status == "error" && testResult.TestStatus != "error" {
				testResult.TestStatus = "error"
			} else if certInfo.Status == "warning" && testResult.TestStatus == "success" {
				testResult.TestStatus = "warning"
			}
		}

		result.HTTPResults = append(result.HTTPResults, testResult)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// runPingTest runs a ping test and returns latency in ms.
func runPingTest(host string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use Go's net.Dial to test ICMP-like connectivity
	// For actual ICMP ping, we'd need raw sockets (root)
	// Using TCP dial to a common port as fallback indicator
	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "ip:icmp", host)
	if err != nil {
		// Try TCP 80 as fallback (won't work for all hosts but gives connectivity indication)
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", host+":80")
		if err != nil {
			// Try TCP 443
			conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", host+":443")
			if err != nil {
				return 0, fmt.Errorf("host unreachable")
			}
		}
	}
	latency := time.Since(start).Seconds() * 1000
	conn.Close()
	return latency, nil
}

// runTCPTest runs a TCP port test and returns latency in ms.
func runTCPTest(host string, port int) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", host, port)
	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	latency := time.Since(start).Seconds() * 1000
	conn.Close()
	return latency, nil
}

// runHTTPTest runs an HTTP test and returns status code, latency in ms.
func runHTTPTest(url string, expectedStatus int) (int, float64, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	start := time.Now()
	resp, err := client.Get(url)
	latency := time.Since(start).Seconds() * 1000

	if err != nil {
		return 0, latency, err
	}
	defer resp.Body.Close()

	if expectedStatus > 0 && resp.StatusCode != expectedStatus {
		return resp.StatusCode, latency, fmt.Errorf("expected %d, got %d", expectedStatus, resp.StatusCode)
	}

	return resp.StatusCode, latency, nil
}

// getTestStatus returns status based on latency and thresholds.
func getTestStatus(latencyMs float64, warningMs, criticalMs int64) string {
	if latencyMs < float64(warningMs) {
		return "success"
	}
	if latencyMs < float64(criticalMs) {
		return "warning"
	}
	return "error"
}

// PingStats holds extended ping statistics.
type PingStats struct {
	AvgLatency float64 // ms
	MinLatency float64 // ms
	MaxLatency float64 // ms
	PacketLoss float64 // percentage
	Jitter     float64 // ms (standard deviation)
}

// runExtendedPing runs multiple pings and returns statistics.
func runExtendedPing(host string, count int) (*PingStats, error) {
	var latencies []float64
	sent := 0
	received := 0

	for i := 0; i < count; i++ {
		sent++
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		start := time.Now()
		// Try TCP 80/443 as ping alternative (actual ICMP requires root)
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", host+":80")
		if err != nil {
			conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", host+":443")
		}
		cancel()

		if err == nil {
			latency := time.Since(start).Seconds() * 1000
			latencies = append(latencies, latency)
			received++
			conn.Close()
		}

		// Small delay between pings
		if i < count-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if len(latencies) == 0 {
		return &PingStats{PacketLoss: 100}, fmt.Errorf("host unreachable")
	}

	// Calculate statistics
	stats := &PingStats{
		PacketLoss: float64(sent-received) / float64(sent) * 100,
	}

	// Min, max, avg
	stats.MinLatency = latencies[0]
	stats.MaxLatency = latencies[0]
	var sum float64
	for _, lat := range latencies {
		sum += lat
		if lat < stats.MinLatency {
			stats.MinLatency = lat
		}
		if lat > stats.MaxLatency {
			stats.MaxLatency = lat
		}
	}
	stats.AvgLatency = sum / float64(len(latencies))

	// Jitter (standard deviation)
	if len(latencies) > 1 {
		var variance float64
		for _, lat := range latencies {
			diff := lat - stats.AvgLatency
			variance += diff * diff
		}
		stats.Jitter = math.Sqrt(variance / float64(len(latencies)))
	}

	return stats, nil
}

// runUDPTest runs a UDP port test and returns latency in ms.
// Note: UDP is connectionless, so we send a packet and wait for ICMP unreachable
// or application response. For DNS (53), NTP (123), etc. we can get actual responses.
func runUDPTest(host string, port int) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", host, port)

	// For DNS port, try a simple DNS query
	if port == 53 {
		return testDNSPort(ctx, host)
	}

	// For other UDP ports, we try to connect (which on UDP just sets up local state)
	// and send a small probe packet
	start := time.Now()

	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	// Set deadline for response
	conn.SetDeadline(time.Now().Add(3 * time.Second))

	// Send a small probe packet
	_, err = conn.Write([]byte{0x00})
	if err != nil {
		return 0, err
	}

	// Try to read response (may timeout for non-responding services)
	buf := make([]byte, 1024)
	_, err = conn.Read(buf)

	latency := time.Since(start).Seconds() * 1000

	// For UDP, no error on Write means the port is likely open
	// (no ICMP unreachable received)
	if err != nil {
		// Check if it's a timeout (which for UDP often means the port is open but not responding)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Port is likely open but service didn't respond - still count as success
			return latency, nil
		}
		// Connection refused or other error means port is closed
		return 0, fmt.Errorf("port closed or filtered")
	}

	return latency, nil
}

// testDNSPort tests DNS port by sending a simple query.
func testDNSPort(ctx context.Context, host string) (float64, error) {
	// Use Go's resolver to test DNS
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", host+":53")
		},
	}

	start := time.Now()
	_, err := resolver.LookupHost(ctx, "google.com")
	latency := time.Since(start).Seconds() * 1000

	if err != nil {
		return 0, err
	}
	return latency, nil
}

// CertInfo holds certificate expiry information.
type CertInfo struct {
	DaysLeft   int
	Status     string // success, warning, error
	ExpiryDate string
	CommonName string
}

// checkCertExpiry checks the TLS certificate expiry for a URL.
func checkCertExpiry(url string, warningDays, criticalDays int) CertInfo {
	info := CertInfo{Status: "success"}

	// Extract host from URL
	host := strings.TrimPrefix(url, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx == -1 {
		host = host + ":443"
	}

	// Connect with TLS
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 5 * time.Second},
		"tcp",
		host,
		&tls.Config{InsecureSkipVerify: true}, // We want to check expiry even for self-signed
	)
	if err != nil {
		info.Status = "error"
		return info
	}
	defer conn.Close()

	// Get certificate chain
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		info.Status = "error"
		return info
	}

	// Check the leaf certificate
	cert := certs[0]
	info.CommonName = cert.Subject.CommonName
	info.ExpiryDate = cert.NotAfter.Format("2006-01-02")

	// Calculate days until expiry
	daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
	info.DaysLeft = daysLeft

	// Determine status
	if daysLeft <= 0 {
		info.Status = "error" // Expired
	} else if daysLeft <= criticalDays {
		info.Status = "error" // Critical
	} else if daysLeft <= warningDays {
		info.Status = "warning" // Warning
	} else {
		info.Status = "success" // OK
	}

	return info
}

// SpeedtestResponse represents the speedtest results for the API.
type SpeedtestResponse struct {
	Download     float64 `json:"download"`     // Mbps
	Upload       float64 `json:"upload"`       // Mbps
	Latency      float64 `json:"latency"`      // ms
	Server       string  `json:"server"`       // Server name
	Location     string  `json:"location"`     // Server location
	Host         string  `json:"host"`         // Server host
	Distance     float64 `json:"distance"`     // km
	Timestamp    string  `json:"timestamp"`
	TestDuration float64 `json:"testDuration"` // seconds
}

// SpeedtestStatusResponse represents the current speedtest status.
type SpeedtestStatusResponse struct {
	Running  bool                   `json:"running"`
	Phase    string                 `json:"phase"`
	Progress float64                `json:"progress"`
	Last     *SpeedtestResponse     `json:"last,omitempty"`
}

// handleSpeedtest starts a speedtest in the background and returns immediately.
// Use /api/speedtest/status to poll for results.
func (s *Server) handleSpeedtest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed - use POST to start a speedtest", http.StatusMethodNotAllowed)
		return
	}

	if s.speedtestTester == nil {
		http.Error(w, "Speedtest not available", http.StatusServiceUnavailable)
		return
	}

	// Check if already running
	status := s.speedtestTester.GetStatus()
	if status.Running {
		http.Error(w, "Speedtest already in progress", http.StatusConflict)
		return
	}

	// Run the test in the background (takes 30-60 seconds)
	go func() {
		ctx := context.Background()
		_, err := s.speedtestTester.RunTest(ctx)
		if err != nil {
			log.Printf("Speedtest failed: %v", err)
		}
	}()

	// Return immediately with "started" status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "started",
		"message": "Speedtest started. Poll /api/speedtest/status for results.",
	})
}

// handleSpeedtestStatus returns the current speedtest status.
func (s *Server) handleSpeedtestStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.speedtestTester == nil {
		http.Error(w, "Speedtest not available", http.StatusServiceUnavailable)
		return
	}

	status := s.speedtestTester.GetStatus()
	resp := SpeedtestStatusResponse{
		Running:  status.Running,
		Phase:    status.Phase,
		Progress: status.Progress,
	}

	// Include last result if available
	if lastResult := s.speedtestTester.GetLastResult(); lastResult != nil {
		resp.Last = &SpeedtestResponse{
			Download:     lastResult.Download,
			Upload:       lastResult.Upload,
			Latency:      lastResult.Latency,
			Server:       lastResult.Server,
			Location:     lastResult.Location,
			Host:         lastResult.Host,
			Distance:     lastResult.Distance,
			Timestamp:    lastResult.Timestamp.Format(time.RFC3339),
			TestDuration: lastResult.TestDuration,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
