package api

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/netscope/internal/config"
	"github.com/krisarmstrong/netscope/internal/dhcp"
	"github.com/krisarmstrong/netscope/internal/discovery"
	"github.com/krisarmstrong/netscope/internal/dns"
	"github.com/krisarmstrong/netscope/internal/gateway"
	"github.com/krisarmstrong/netscope/internal/iperf"
	"github.com/krisarmstrong/netscope/internal/network"
	"github.com/krisarmstrong/netscope/internal/validation"
	"github.com/krisarmstrong/netscope/internal/version"
	"github.com/krisarmstrong/netscope/internal/vlan"
)

// sendJSONResponse is a helper to send JSON responses and handle encoding errors.
func sendJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func readLastLines(path string, maxBytes int64, maxLines int) ([]string, error) {
	//nolint:gosec // G304: path is from config for log file location
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	var start int64
	if info.Size() > maxBytes {
		start = info.Size() - maxBytes
	}
	if start > 0 {
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			return nil, err
		}
	}

	scanner := bufio.NewScanner(f)
	// Allow long lines (up to 1MB)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lines := make([]string, 0, maxLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

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
	Status        string `json:"status"`
	Version       string `json:"version"`
	Uptime        int64  `json:"uptime"`
	Interface     string `json:"interface"`
	IsWireless    bool   `json:"isWireless"`
	ICMPAvailable bool   `json:"icmpAvailable"`
}

// handleLogin handles user authentication.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client IP for rate limiting
	clientIP := GetClientIP(r)

	// Check if IP is rate limited
	if s.loginRateLimiter.IsBlocked(clientIP) {
		w.Header().Set("Retry-After", "900") // 15 minutes
		remaining := s.loginRateLimiter.RemainingAttempts(clientIP)
		sendJSONResponse(w, http.StatusTooManyRequests, map[string]interface{}{
			"error":              "Too many failed login attempts",
			"retry_after":        900,
			"remaining_attempts": remaining,
		})
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	var req LoginRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Printf("login decode error: %v body=%q", err, string(bodyBytes))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := s.authManager.Authenticate(req.Username, req.Password)
	if err != nil {
		// Record failed attempt
		blocked := s.loginRateLimiter.RecordAttempt(clientIP, false)
		remaining := s.loginRateLimiter.RemainingAttempts(clientIP)

		if blocked {
			w.Header().Set("Retry-After", "900")
			sendJSONResponse(w, http.StatusTooManyRequests, map[string]interface{}{
				"error":              "Too many failed login attempts. Account temporarily locked.",
				"retry_after":        900,
				"remaining_attempts": 0,
			})
			return
		}

		sendJSONResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"error":              "Invalid credentials",
			"remaining_attempts": remaining,
		})
		return
	}

	// Record successful attempt (clears previous failures)
	s.loginRateLimiter.RecordAttempt(clientIP, true)

	resp := LoginResponse{
		Token:   token,
		Expires: time.Now().Add(s.config.Auth.SessionTimeout).Unix(),
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleLogout handles user logout.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// JWT is stateless, so we just acknowledge the logout
	// Client should discard the token
	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "logged out"})
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
		Status:        "ok",
		Version:       version.Version,
		Interface:     s.config.Interface.Default,
		IsWireless:    isWireless,
		ICMPAvailable: s.icmpAvailable,
	}

	sendJSONResponse(w, http.StatusOK, resp)
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
	// Lock config for read access
	s.config.RLock()
	defer s.config.RUnlock()

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
			"httpTimings": map[string]map[string]int64{
				"dns": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.DNS.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.DNS.Critical.Milliseconds(),
				},
				"tcp": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.TCP.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.TCP.Critical.Milliseconds(),
				},
				"tls": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.TLS.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.TLS.Critical.Milliseconds(),
				},
				"ttfb": {
					"good":    s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Warning.Milliseconds(),
					"warning": s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Critical.Milliseconds(),
				},
			},
		},
	}

	sendJSONResponse(w, http.StatusOK, settings)
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Lock config for write access
	s.config.Lock()
	defer s.config.Unlock()

	// Apply threshold updates
	if thresholds, ok := updates["thresholds"].(map[string]interface{}); ok {
		if dnsThresh, ok := thresholds["dns"].(map[string]interface{}); ok {
			if good, ok := dnsThresh["good"].(float64); ok {
				s.config.Thresholds.DNS.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := dnsThresh["warning"].(float64); ok {
				s.config.Thresholds.DNS.Critical = time.Duration(warning) * time.Millisecond
			}
		}
		if gwThresh, ok := thresholds["gateway"].(map[string]interface{}); ok {
			if good, ok := gwThresh["good"].(float64); ok {
				s.config.Thresholds.Ping.Warning = time.Duration(good) * time.Millisecond
			}
			if warning, ok := gwThresh["warning"].(float64); ok {
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
		if httpTimings, ok := thresholds["httpTimings"].(map[string]interface{}); ok {
			if dnsT, ok := httpTimings["dns"].(map[string]interface{}); ok {
				if good, ok := dnsT["good"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.DNS.Warning = time.Duration(good) * time.Millisecond
				}
				if warning, ok := dnsT["warning"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.DNS.Critical = time.Duration(warning) * time.Millisecond
				}
			}
			if tcpT, ok := httpTimings["tcp"].(map[string]interface{}); ok {
				if good, ok := tcpT["good"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TCP.Warning = time.Duration(good) * time.Millisecond
				}
				if warning, ok := tcpT["warning"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TCP.Critical = time.Duration(warning) * time.Millisecond
				}
			}
			if tlsT, ok := httpTimings["tls"].(map[string]interface{}); ok {
				if good, ok := tlsT["good"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TLS.Warning = time.Duration(good) * time.Millisecond
				}
				if warning, ok := tlsT["warning"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TLS.Critical = time.Duration(warning) * time.Millisecond
				}
			}
			if ttfb, ok := httpTimings["ttfb"].(map[string]interface{}); ok {
				if good, ok := ttfb["good"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Warning = time.Duration(good) * time.Millisecond
				}
				if warning, ok := ttfb["warning"].(float64); ok {
					s.config.Thresholds.CustomTests.HTTPTimings.TTFB.Critical = time.Duration(warning) * time.Millisecond
				}
			}
		}
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "updated"})
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

	if err := s.netManager.RefreshInterfaces(); err != nil {
		http.Error(w, "Failed to refresh interfaces", http.StatusInternalServerError)
		return
	}
	interfaces := s.netManager.GetInterfaces()

	sendJSONResponse(w, http.StatusOK, interfaces)
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

	sendJSONResponse(w, http.StatusOK, resp)
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
	if err := s.netManager.RefreshInterfaces(); err != nil {
		http.Error(w, "Failed to refresh interfaces", http.StatusInternalServerError)
		return
	}

	// Get interface info
	var mac string
	if ifaceInfo, err := s.netManager.GetInterface(currentIface); err == nil {
		mac = ifaceInfo.HardwareAddr
	}

	export := ExportData{
		Version:   version.Version,
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
				"result": result.Forward.Resolved,
				"time":   result.Forward.Time.Milliseconds(),
				"status": result.Forward.Status,
				"error":  result.Forward.Error,
			}
		}
		if result.Reverse != nil {
			dnsData["reverse"] = map[string]interface{}{
				"result": result.Reverse.Resolved,
				"time":   result.Reverse.Time.Milliseconds(),
				"status": result.Reverse.Status,
				"error":  result.Reverse.Error,
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

	// Collect Speedtest data
	if s.speedtestTester != nil {
		if result := s.speedtestTester.GetLastResult(); result != nil {
			export.Cards["speedtest"] = map[string]interface{}{
				"download":     result.Download,
				"upload":       result.Upload,
				"latency":      result.Latency,
				"server":       result.Server,
				"location":     result.Location,
				"host":         result.Host,
				"distance":     result.Distance,
				"timestamp":    result.Timestamp,
				"testDuration": result.TestDuration,
			}
		}
	}

	// Collect iperf3 data
	if s.iperfManager != nil {
		if result := s.iperfManager.GetLastResult(); result != nil {
			export.Cards["iperf"] = map[string]interface{}{
				"bandwidth":   result.Bandwidth,
				"transfer":    result.Transfer,
				"retransmits": result.Retransmits,
				"jitter":      result.Jitter,
				"lostPackets": result.LostPackets,
				"lostPercent": result.LostPercent,
				"protocol":    result.Protocol,
				"direction":   result.Direction,
				"duration":    result.Duration,
				"server":      result.Server,
				"port":        result.Port,
				"timestamp":   result.Timestamp,
			}
		}
	}

	// Pretty print JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=netscope-export.json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(export); err != nil {
		log.Printf("Error encoding export response: %v", err)
	}
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
		})
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// TCPProbeRequest represents a TCP probe request.
type TCPProbeRequest struct {
	Target  string `json:"target"`  // IP or hostname
	Port    int    `json:"port"`    // Single port
	Ports   []int  `json:"ports"`   // Multiple ports
	Timeout int    `json:"timeout"` // Timeout in ms (default 1000)
}

// TCPProbeResponse represents TCP probe results.
type TCPProbeResponse struct {
	Target  string                     `json:"target"`
	Results []discovery.TCPProbeResult `json:"results"`
}

// handleTCPProbe handles TCP port probe requests.
func (s *Server) handleTCPProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TCPProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate target
	if req.Target == "" {
		http.Error(w, "Target is required", http.StatusBadRequest)
		return
	}

	// Resolve hostname if needed
	ip := net.ParseIP(req.Target)
	if ip == nil {
		// Try to resolve hostname
		ips, err := net.LookupIP(req.Target)
		if err != nil || len(ips) == 0 {
			http.Error(w, "Unable to resolve hostname", http.StatusBadRequest)
			return
		}
		// Use first IPv4 address
		for _, resolvedIP := range ips {
			if resolvedIP.To4() != nil {
				ip = resolvedIP
				break
			}
		}
		if ip == nil {
			ip = ips[0]
		}
	}

	// Build port list
	var ports []int
	if req.Port > 0 {
		ports = append(ports, req.Port)
	}
	ports = append(ports, req.Ports...)
	if len(ports) == 0 {
		http.Error(w, "At least one port is required", http.StatusBadRequest)
		return
	}

	// Limit ports to prevent abuse
	if len(ports) > 100 {
		http.Error(w, "Maximum 100 ports allowed", http.StatusBadRequest)
		return
	}

	// Set timeout
	timeout := time.Second
	if req.Timeout > 0 && req.Timeout <= 10000 {
		timeout = time.Duration(req.Timeout) * time.Millisecond
	}

	// Create prober
	prober, err := discovery.NewTCPProber(timeout)
	if err != nil {
		http.Error(w, "Failed to create prober: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer prober.Close()

	// Run probes
	ctx, cancel := context.WithTimeout(r.Context(), timeout*time.Duration(len(ports))+5*time.Second)
	defer cancel()

	results := prober.ScanPorts(ctx, ip.String(), ports, 10)

	resp := TCPProbeResponse{
		Target:  req.Target,
		Results: results,
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// TracerouteRequest represents a traceroute request.
type TracerouteRequest struct {
	Target   string `json:"target"`   // IP or hostname
	Protocol string `json:"protocol"` // "icmp", "udp", "tcp" (default: icmp)
	Port     int    `json:"port"`     // Port for TCP/UDP (default: 80 for TCP, 33434 for UDP)
	MaxHops  int    `json:"maxHops"`  // Max TTL (default: 30)
	Timeout  int    `json:"timeout"`  // Per-hop timeout in ms (default: 3000)
}

// handleTraceroute handles traceroute requests.
func (s *Server) handleTraceroute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TracerouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate target
	if req.Target == "" {
		http.Error(w, "Target is required", http.StatusBadRequest)
		return
	}

	// Validate target for security (must be valid IP or resolvable hostname)
	ip := net.ParseIP(req.Target)
	if ip == nil {
		// Not an IP, check if it looks like a valid hostname
		if err := validation.ValidateServerAddress(req.Target); err != nil {
			http.Error(w, "Invalid target: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Set defaults
	protocol := req.Protocol
	if protocol == "" {
		protocol = "icmp"
	}
	if protocol != "icmp" && protocol != "udp" && protocol != "tcp" {
		http.Error(w, "Protocol must be icmp, udp, or tcp", http.StatusBadRequest)
		return
	}

	maxHops := req.MaxHops
	if maxHops <= 0 || maxHops > 64 {
		maxHops = 30
	}

	timeout := time.Duration(req.Timeout) * time.Millisecond
	if timeout <= 0 || timeout > 10*time.Second {
		timeout = 3 * time.Second
	}

	port := req.Port
	if port <= 0 {
		if protocol == "tcp" {
			port = 80
		} else {
			port = 33434
		}
	}

	// Create tracer
	tracer := discovery.NewTracer(timeout, maxHops)

	// Set overall timeout for the operation
	ctx, cancel := context.WithTimeout(r.Context(), timeout*time.Duration(maxHops)+10*time.Second)
	defer cancel()

	// Run traceroute based on protocol
	var result *discovery.TracerouteResult
	switch protocol {
	case "icmp":
		result = tracer.TraceICMP(ctx, req.Target)
	case "udp":
		result = tracer.TraceUDP(ctx, req.Target, port)
	case "tcp":
		result = tracer.TraceTCP(ctx, req.Target, port)
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// PortScanRequest represents a port scan request.
type PortScanRequest struct {
	Target  string `json:"target"`            // IP or hostname
	Ports   []int  `json:"ports,omitempty"`   // Specific ports (optional, defaults to common ports)
	Profile string `json:"profile,omitempty"` // "quick", "web", "full" (default: quick)
	Workers int    `json:"workers,omitempty"` // Concurrent workers (default: 20)
}

// handlePortScan handles port scanning with service detection.
func (s *Server) handlePortScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PortScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate target
	if err := validation.ValidateServerAddress(req.Target); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid target: %v", err)})
		return
	}

	// Create scanner
	scanner, err := discovery.NewPortScanner(3 * time.Second)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create scanner: %v", err)})
		return
	}
	defer scanner.Close()

	// Set timeout for operation
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	var result *discovery.PortScanResult

	// Determine scan type
	if len(req.Ports) > 0 {
		workers := req.Workers
		if workers <= 0 {
			workers = 20
		}
		result = scanner.ScanWithBanners(ctx, req.Target, req.Ports, workers)
	} else {
		switch req.Profile {
		case "web":
			result = scanner.WebScan(ctx, req.Target)
		case "full":
			result = scanner.FullScan(ctx, req.Target)
		default: // "quick" or unspecified
			result = scanner.QuickScan(ctx, req.Target)
		}
	}

	sendJSONResponse(w, http.StatusOK, result)
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

// DNSServerTestResult represents per-server DNS test results for the API.
type DNSServerTestResult struct {
	Server      string           `json:"server"`
	Forward     *DNSLookupResult `json:"forward,omitempty"`
	ForwardIpv6 *DNSLookupResult `json:"forwardIpv6,omitempty"`
	Status      string           `json:"status"`
	AvgTimeMs   int64            `json:"avgTimeMs"`
}

// DNSResponse represents the DNS test results for the API.
type DNSResponse struct {
	Server           string                 `json:"server"`
	Servers          []string               `json:"servers"` // All configured DNS servers
	TestHostname     string                 `json:"testHostname"`
	Forward          *DNSLookupResult       `json:"forward,omitempty"`
	ForwardIpv6      *DNSLookupResult       `json:"forwardIpv6,omitempty"`
	Reverse          *DNSLookupResult       `json:"reverse,omitempty"`
	ReverseIpv6      *DNSLookupResult       `json:"reverseIpv6,omitempty"`
	PerServerResults []*DNSServerTestResult `json:"perServerResults,omitempty"`
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

	// Map per-server results
	if len(result.PerServerResults) > 0 {
		for _, serverResult := range result.PerServerResults {
			apiResult := &DNSServerTestResult{
				Server:    serverResult.Server,
				Status:    string(serverResult.Status),
				AvgTimeMs: serverResult.AvgTimeMs,
			}
			if serverResult.Forward != nil {
				apiResult.Forward = &DNSLookupResult{
					Result:   serverResult.Forward.Result,
					Time:     serverResult.Forward.TimeMs,
					TimeMs:   serverResult.Forward.TimeMs,
					Status:   string(serverResult.Forward.Status),
					Error:    serverResult.Forward.Error,
					Resolved: serverResult.Forward.Resolved,
				}
			}
			if serverResult.ForwardIPv6 != nil {
				apiResult.ForwardIpv6 = &DNSLookupResult{
					Result:   serverResult.ForwardIPv6.Result,
					Time:     serverResult.ForwardIPv6.TimeMs,
					TimeMs:   serverResult.ForwardIPv6.TimeMs,
					Status:   string(serverResult.ForwardIPv6.Status),
					Error:    serverResult.ForwardIPv6.Error,
					Resolved: serverResult.ForwardIPv6.Resolved,
				}
			}
			resp.PerServerResults = append(resp.PerServerResults, apiResult)
		}
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// GatewayResponse represents the gateway ping test results for the API.
type GatewayResponse struct {
	Gateway     string           `json:"gateway"`
	Reachable   bool             `json:"reachable"`
	Sent        int              `json:"sent"`
	Received    int              `json:"received"`
	LossPercent float64          `json:"lossPercent"`
	MinTime     float64          `json:"minTime"`
	MaxTime     float64          `json:"maxTime"`
	AvgTime     float64          `json:"avgTime"`
	LastTime    float64          `json:"lastTime"`
	Status      string           `json:"status"`
	IPv6        *GatewayResponse `json:"ipv6,omitempty"`
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

	// Perform IPv4 gateway ping test
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

	// Detect and ping IPv6 gateway if available
	ipv6Gateway, err := gateway.DetectGatewayIPv6()
	if err == nil && ipv6Gateway != "" {
		// Create a temporary tester for IPv6
		ipv6Tester := gateway.NewTester(gateway.DefaultThresholds())
		ipv6Tester.SetGateway(ipv6Gateway)
		ipv6Stats := ipv6Tester.Test()

		resp.IPv6 = &GatewayResponse{
			Gateway:     ipv6Stats.Gateway,
			Reachable:   ipv6Stats.Reachable,
			Sent:        ipv6Stats.Sent,
			Received:    ipv6Stats.Received,
			LossPercent: ipv6Stats.LossPercent,
			MinTime:     ipv6Stats.MinTime,
			MaxTime:     ipv6Stats.MaxTime,
			AvgTime:     ipv6Stats.AvgTime,
			LastTime:    ipv6Stats.LastTime,
			Status:      string(ipv6Stats.Status),
		}
	}

	sendJSONResponse(w, http.StatusOK, resp)
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

// VLANInterfaceRequest represents the request to create/delete a VLAN interface.
type VLANInterfaceRequest struct {
	Interface string `json:"interface"`
	VlanID    int    `json:"vlanId"`
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

// createVLANInterface creates an 802.1Q VLAN subinterface.
func (s *Server) createVLANInterface(w http.ResponseWriter, r *http.Request) {
	var req VLANInterfaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate VLAN ID (1-4094)
	if req.VlanID < 1 || req.VlanID > 4094 {
		http.Error(w, "VLAN ID must be between 1 and 4094", http.StatusBadRequest)
		return
	}

	// Use current interface if not specified
	iface := req.Interface
	if iface == "" {
		iface = s.netManager.GetCurrentInterface()
	}

	// Validate interface name
	if err := validation.ValidateInterface(iface); err != nil {
		http.Error(w, fmt.Sprintf("Invalid interface: %v", err), http.StatusBadRequest)
		return
	}

	// Create the VLAN interface
	if err := vlan.CreateVlanInterface(iface, req.VlanID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create VLAN interface: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "VLAN interface created",
		"interface": iface,
		"vlanId":    req.VlanID,
	})
}

// deleteVLANInterface removes an 802.1Q VLAN subinterface.
func (s *Server) deleteVLANInterface(w http.ResponseWriter, r *http.Request) {
	var req VLANInterfaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate VLAN ID (1-4094)
	if req.VlanID < 1 || req.VlanID > 4094 {
		http.Error(w, "VLAN ID must be between 1 and 4094", http.StatusBadRequest)
		return
	}

	// Use current interface if not specified
	iface := req.Interface
	if iface == "" {
		iface = s.netManager.GetCurrentInterface()
	}

	// Validate interface name
	if err := validation.ValidateInterface(iface); err != nil {
		http.Error(w, fmt.Sprintf("Invalid interface: %v", err), http.StatusBadRequest)
		return
	}

	// Delete the VLAN interface
	if err := vlan.DeleteVlanInterface(iface, req.VlanID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete VLAN interface: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "VLAN interface deleted",
		"interface": iface,
		"vlanId":    req.VlanID,
	})
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

// SetMTURequest represents the request to set interface MTU.
type SetMTURequest struct {
	Interface string `json:"interface"`
	MTU       int    `json:"mtu"`
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

// TestsSettingsResponse represents the custom tests configuration.
type TestsSettingsResponse struct {
	DNSHostname    string                    `json:"dnsHostname"`
	DNSServers     []DNSServerResponse       `json:"dnsServers"`
	PingTargets    []PingTargetResponse      `json:"pingTargets"`
	TCPPorts       []TCPPortResponse         `json:"tcpPorts"`
	UDPPorts       []UDPPortResponse         `json:"udpPorts"`
	HTTPEndpoints  []HTTPEndpointResponse    `json:"httpEndpoints"`
	Speedtest      SpeedtestSettingsResponse `json:"speedtest"`
	Iperf          IperfSettingsResponse     `json:"iperf"`
	RunPerformance bool                      `json:"runPerformance"`
	RunSpeedtest   bool                      `json:"runSpeedtest"`
	RunIperf       bool                      `json:"runIperf"`
	RunDiscovery   bool                      `json:"runDiscovery"`
}

// DNSServerResponse represents a DNS server for testing.
type DNSServerResponse struct {
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
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

type IperfSettingsResponse struct {
	AutoRunOnLink bool `json:"autoRunOnLink"`
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
		DNSHostname:    s.config.DNS.TestHostname,
		DNSServers:     make([]DNSServerResponse, 0, len(s.config.DNS.Servers)),
		PingTargets:    make([]PingTargetResponse, 0, len(s.config.Tests.PingTargets)),
		TCPPorts:       make([]TCPPortResponse, 0, len(s.config.Tests.TCPPorts)),
		UDPPorts:       make([]UDPPortResponse, 0, len(s.config.Tests.UDPPorts)),
		HTTPEndpoints:  make([]HTTPEndpointResponse, 0, len(s.config.Tests.HTTPEndpoints)),
		RunPerformance: s.config.Tests.RunPerformance,
		RunSpeedtest:   s.config.Tests.RunSpeedtest,
		RunIperf:       s.config.Tests.RunIperf,
		RunDiscovery:   s.config.Tests.RunDiscovery,
		Speedtest: SpeedtestSettingsResponse{
			ServerID:      s.config.Speedtest.ServerID,
			AutoRunOnLink: s.config.Speedtest.AutoRunOnLink,
		},
		Iperf: IperfSettingsResponse{
			AutoRunOnLink: s.config.Iperf.AutoRunOnLink,
		},
	}

	// DNS servers
	for _, d := range s.config.DNS.Servers {
		resp.DNSServers = append(resp.DNSServers, DNSServerResponse{
			Address: d.Address,
			Enabled: d.Enabled,
		})
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

	sendJSONResponse(w, http.StatusOK, resp)
}

func (s *Server) updateTestsSettings(w http.ResponseWriter, r *http.Request) {
	var req TestsSettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Lock config for write access
	s.config.Lock()
	defer s.config.Unlock()

	// Update DNS hostname
	if req.DNSHostname != "" {
		s.config.DNS.TestHostname = req.DNSHostname
		// Update the DNS tester with the new hostname
		if s.dnsTester != nil {
			s.dnsTester.SetTestHostname(req.DNSHostname)
		}
	}

	// Update DNS servers
	s.config.DNS.Servers = make([]config.DNSServer, 0, len(req.DNSServers))
	for _, d := range req.DNSServers {
		s.config.DNS.Servers = append(s.config.DNS.Servers, config.DNSServer{
			Address: d.Address,
			Enabled: d.Enabled,
		})
	}
	// Update the DNS tester with the configured servers
	if s.dnsTester != nil {
		configuredServers := make([]dns.ConfiguredServer, 0, len(s.config.DNS.Servers))
		for _, d := range s.config.DNS.Servers {
			configuredServers = append(configuredServers, dns.ConfiguredServer{
				Address: d.Address,
				Enabled: d.Enabled,
			})
		}
		s.dnsTester.SetConfiguredServers(configuredServers)
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
	// Store URL as-is to preserve user intent - scheme-less URLs enable HTTPS->HTTP fallback at test time
	s.config.Tests.HTTPEndpoints = make([]config.HTTPEndpoint, 0, len(req.HTTPEndpoints))
	for _, h := range req.HTTPEndpoints {
		s.config.Tests.HTTPEndpoints = append(s.config.Tests.HTTPEndpoints, config.HTTPEndpoint{
			Name:           h.Name,
			URL:            h.URL,
			ExpectedStatus: h.ExpectedStatus,
			Enabled:        h.Enabled,
		})
	}

	// Update performance toggle
	s.config.Tests.RunPerformance = req.RunPerformance
	s.config.Tests.RunSpeedtest = req.RunSpeedtest
	s.config.Tests.RunIperf = req.RunIperf
	s.config.Tests.RunDiscovery = req.RunDiscovery

	// Update speedtest settings
	s.config.Speedtest.ServerID = req.Speedtest.ServerID
	s.config.Speedtest.AutoRunOnLink = req.Speedtest.AutoRunOnLink
	if s.speedtestTester != nil {
		s.speedtestTester.SetServerID(req.Speedtest.ServerID)
	}

	// Update iperf settings
	s.config.Iperf.AutoRunOnLink = req.Iperf.AutoRunOnLink

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
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
	DNSLatency  float64 `json:"dnsLatency,omitempty"`
	TCPConnect  float64 `json:"tcpConnect,omitempty"`
	TLSLatency  float64 `json:"tlsLatency,omitempty"`
	TTFBLatency float64 `json:"ttfbLatency,omitempty"` // Time to first byte (server processing + wait)
	Error       string  `json:"error,omitempty"`
	Status      int     `json:"status,omitempty"`     // HTTP status code
	TestStatus  string  `json:"testStatus,omitempty"` // success, warning, error
	// Per-phase status fields for HTTP timing breakdown
	DNSStatus  string `json:"dnsStatus,omitempty"`  // success, warning, error
	TCPStatus  string `json:"tcpStatus,omitempty"`  // success, warning, error
	TLSStatus  string `json:"tlsStatus,omitempty"`  // success, warning, error
	TTFBStatus string `json:"ttfbStatus,omitempty"` // success, warning, error
	// Extended ping fields
	PacketLoss float64 `json:"packetLoss,omitempty"` // Percentage
	Jitter     float64 `json:"jitter,omitempty"`     // ms
	MinLatency float64 `json:"minLatency,omitempty"` // ms
	MaxLatency float64 `json:"maxLatency,omitempty"` // ms
	// Certificate fields
	CertDaysLeft   int    `json:"certDaysLeft,omitempty"`   // Days until cert expires
	CertStatus     string `json:"certStatus,omitempty"`     // success, warning, error
	CertExpiry     string `json:"certExpiry,omitempty"`     // Expiry date string
	CertCommonName string `json:"certCommonName,omitempty"` // Certificate CN
	TLSVersion     string `json:"tlsVersion,omitempty"`     // TLS 1.2, TLS 1.3, etc.
	CertIssuer     string `json:"certIssuer,omitempty"`     // Certificate issuer
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
	httpTimingThresholds := s.config.Thresholds.CustomTests.HTTPTimings
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
			switch {
			case pingStats.PacketLoss > 50:
				testResult.TestStatus = "error"
			case pingStats.PacketLoss > 10:
				testResult.TestStatus = "warning"
			default:
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
			name = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
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
			name = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
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

		// Validate URL to prevent SSRF attacks
		if err := validation.ValidateURL(endpoint.URL); err != nil {
			log.Printf("Skipping invalid HTTP endpoint URL %q: %v", endpoint.URL, err)
			continue
		}

		// Determine URL and whether to try fallback
		url := endpoint.URL
		tryHTTPFallback := false

		if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			// No scheme provided - try HTTPS first, can fallback to HTTP
			url = "https://" + url
			tryHTTPFallback = true
		}

		name := endpoint.Name
		if name == "" {
			name = endpoint.URL // Show original URL in name
		}

		testResult := CustomTestResult{
			Name: name,
			URL:  url,
		}

		statusCode, timings, err := runHTTPTest(url, endpoint.ExpectedStatus)

		// If HTTPS failed and we can try HTTP fallback
		if err != nil && tryHTTPFallback {
			httpURL := "http://" + endpoint.URL
			httpStatus, httpTimings, httpErr := runHTTPTest(httpURL, endpoint.ExpectedStatus)
			if httpErr == nil || httpStatus > 0 {
				// HTTP worked (or at least connected) - use those results
				url = httpURL
				testResult.URL = httpURL
				statusCode = httpStatus
				timings = httpTimings
				err = httpErr
			}
		}

		testResult.Status = statusCode
		testResult.Latency = timings.Total
		testResult.DNSLatency = timings.DNS
		testResult.TCPConnect = timings.Connect
		testResult.TLSLatency = timings.TLS
		testResult.TTFBLatency = timings.TTFB
		if err != nil {
			testResult.Success = false
			testResult.Error = err.Error()
			testResult.TestStatus = "error"
		} else {
			testResult.Success = true
			// Evaluate each phase against its threshold
			testResult.DNSStatus = getTestStatus(timings.DNS, httpTimingThresholds.DNS.Warning.Milliseconds(), httpTimingThresholds.DNS.Critical.Milliseconds())
			testResult.TCPStatus = getTestStatus(timings.Connect, httpTimingThresholds.TCP.Warning.Milliseconds(), httpTimingThresholds.TCP.Critical.Milliseconds())
			testResult.TLSStatus = getTestStatus(timings.TLS, httpTimingThresholds.TLS.Warning.Milliseconds(), httpTimingThresholds.TLS.Critical.Milliseconds())
			testResult.TTFBStatus = getTestStatus(timings.TTFB, httpTimingThresholds.TTFB.Warning.Milliseconds(), httpTimingThresholds.TTFB.Critical.Milliseconds())

			// Overall test status: error if any phase is error, warning if any warning, else use total time
			switch {
			case testResult.DNSStatus == "error" || testResult.TCPStatus == "error" ||
				testResult.TLSStatus == "error" || testResult.TTFBStatus == "error":
				testResult.TestStatus = "error"
			case testResult.DNSStatus == "warning" || testResult.TCPStatus == "warning" ||
				testResult.TLSStatus == "warning" || testResult.TTFBStatus == "warning":
				testResult.TestStatus = "warning"
			default:
				testResult.TestStatus = getTestStatus(timings.Total, httpThreshold.Warning.Milliseconds(), httpThreshold.Critical.Milliseconds())
			}
		}

		// Check certificate expiry for HTTPS URLs only
		if strings.HasPrefix(url, "https://") && testResult.Success {
			certInfo := checkCertExpiry(url, certThreshold.Warning, certThreshold.Critical)
			testResult.CertDaysLeft = certInfo.DaysLeft
			testResult.CertStatus = certInfo.Status
			testResult.CertExpiry = certInfo.ExpiryDate
			testResult.CertCommonName = certInfo.CommonName
			testResult.TLSVersion = certInfo.TLSVersion
			testResult.CertIssuer = certInfo.Issuer

			// Upgrade test status if cert is in bad shape
			if certInfo.Status == "error" && testResult.TestStatus != "error" {
				testResult.TestStatus = "error"
			} else if certInfo.Status == "warning" && testResult.TestStatus == "success" {
				testResult.TestStatus = "warning"
			}
		}

		result.HTTPResults = append(result.HTTPResults, testResult)
	}

	sendJSONResponse(w, http.StatusOK, result)
}

// runTCPTest runs a TCP port test and returns latency in ms.
func runTCPTest(host string, port int) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	latency := time.Since(start).Seconds() * 1000
	conn.Close()
	return latency, nil
}

type httpTimings struct {
	DNS     float64
	Connect float64
	TLS     float64
	TTFB    float64 // Time to first byte (from request sent to first response byte)
	Total   float64
}

// runHTTPTest runs an HTTP test and returns status code and timings in ms.
// Uses SafeTransport to prevent DNS rebinding SSRF attacks.
func runHTTPTest(url string, expectedStatus int) (status int, timing httpTimings, err error) {
	// Use SafeTransport to block connections to private IPs (prevents DNS rebinding)
	transport := validation.SafeTransport()
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return 0, timing, err
	}

	var dnsStart, connStart, tlsStart, wroteRequest time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				timing.DNS += time.Since(dnsStart).Seconds() * 1000
			}
		},
		ConnectStart: func(_, _ string) {
			connStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			if !connStart.IsZero() {
				timing.Connect += time.Since(connStart).Seconds() * 1000
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tlsStart.IsZero() {
				timing.TLS += time.Since(tlsStart).Seconds() * 1000
			}
		},
		WroteRequest: func(httptrace.WroteRequestInfo) {
			wroteRequest = time.Now()
		},
		GotFirstResponseByte: func() {
			if !wroteRequest.IsZero() {
				timing.TTFB = time.Since(wroteRequest).Seconds() * 1000
			}
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	start := time.Now()
	resp, err := client.Do(req)
	timing.Total = time.Since(start).Seconds() * 1000

	if err != nil {
		return 0, timing, err
	}
	defer resp.Body.Close()

	status = resp.StatusCode
	if expectedStatus > 0 && status != expectedStatus {
		return status, timing, fmt.Errorf("expected %d, got %d", expectedStatus, status)
	}

	return status, timing, nil
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

	addr := net.JoinHostPort(host, strconv.Itoa(port))

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
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return 0, err
	}

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
	TLSVersion string // TLS 1.0, TLS 1.1, TLS 1.2, TLS 1.3
	Issuer     string // Certificate issuer (for context)
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
		host += ":443"
	}

	// Connect with TLS
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 5 * time.Second},
		"tcp",
		host,
		// #nosec G402 - certificate verification intentionally skipped to inspect expiry
		&tls.Config{InsecureSkipVerify: true}, // We want to check expiry even for self-signed
	)
	if err != nil {
		info.Status = "error"
		return info
	}
	defer conn.Close()

	// Get connection state for TLS info
	connState := conn.ConnectionState()

	// Get TLS version
	info.TLSVersion = getTLSVersionString(connState.Version)

	// Get certificate chain
	certs := connState.PeerCertificates
	if len(certs) == 0 {
		info.Status = "error"
		return info
	}

	// Check the leaf certificate
	cert := certs[0]
	info.CommonName = cert.Subject.CommonName
	info.ExpiryDate = cert.NotAfter.Format("2006-01-02")

	// Get issuer (org or CN)
	if len(cert.Issuer.Organization) > 0 {
		info.Issuer = cert.Issuer.Organization[0]
	} else if cert.Issuer.CommonName != "" {
		info.Issuer = cert.Issuer.CommonName
	}

	// Calculate days until expiry
	daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
	info.DaysLeft = daysLeft

	// Determine status
	switch {
	case daysLeft <= 0:
		info.Status = "error" // Expired
	case daysLeft <= criticalDays:
		info.Status = "error" // Critical
	case daysLeft <= warningDays:
		info.Status = "warning" // Warning
	default:
		info.Status = "success" // OK
	}

	return info
}

// getTLSVersionString converts TLS version to human-readable string.
func getTLSVersionString(tlsVersion uint16) string {
	switch tlsVersion {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}

// SpeedtestResponse represents the speedtest results for the API.
type SpeedtestResponse struct {
	Download     float64 `json:"download"` // Mbps
	Upload       float64 `json:"upload"`   // Mbps
	Latency      float64 `json:"latency"`  // ms
	Server       string  `json:"server"`   // Server name
	Location     string  `json:"location"` // Server location
	Host         string  `json:"host"`     // Server host
	Distance     float64 `json:"distance"` // km
	Timestamp    string  `json:"timestamp"`
	TestDuration float64 `json:"testDuration"` // seconds
}

// SpeedtestStatusResponse represents the current speedtest status.
type SpeedtestStatusResponse struct {
	Running  bool               `json:"running"`
	Phase    string             `json:"phase"`
	Progress float64            `json:"progress"`
	Last     *SpeedtestResponse `json:"last,omitempty"`
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
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
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

	sendJSONResponse(w, http.StatusOK, resp)
}

// iperf3 handlers

// IperfInfoResponse contains iperf3 installation info
type IperfInfoResponse struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
}

// handleIperfInfo returns iperf3 installation status and version
func (s *Server) handleIperfInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := IperfInfoResponse{}
	iperfVersion, err := iperf.GetVersion()
	if err != nil {
		resp.Installed = false
		resp.Error = err.Error()
	} else {
		resp.Installed = true
		resp.Version = iperfVersion
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// IperfClientRequest is the request body for running an iperf3 client test
type IperfClientRequest struct {
	Server    string `json:"server"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`  // "tcp" or "udp"
	Reverse   bool   `json:"reverse"`   // true = download, false = upload (legacy)
	Direction string `json:"direction"` // "upload", "download", "bidirectional"
	Duration  int    `json:"duration"`  // seconds
	Parallel  int    `json:"parallel"`  // number of streams
}

// IperfResultResponse is the response for an iperf3 test result
type IperfResultResponse struct {
	Bandwidth         float64 `json:"bandwidth"`   // Mbps
	Transfer          float64 `json:"transfer"`    // MB
	Retransmits       int     `json:"retransmits"` // TCP only
	Jitter            float64 `json:"jitter"`      // UDP only, ms
	LostPackets       int     `json:"lostPackets"` // UDP only
	LostPercent       float64 `json:"lostPercent"` // UDP only
	Protocol          string  `json:"protocol"`
	Direction         string  `json:"direction"`
	Duration          float64 `json:"duration"`
	Server            string  `json:"server"`
	Port              int     `json:"port"`
	Timestamp         string  `json:"timestamp"`
	DownloadBandwidth float64 `json:"downloadBandwidth,omitempty"`
	UploadBandwidth   float64 `json:"uploadBandwidth,omitempty"`
	DownloadTransfer  float64 `json:"downloadTransfer,omitempty"`
	UploadTransfer    float64 `json:"uploadTransfer,omitempty"`
}

// handleIperfClient runs an iperf3 client test
func (s *Server) handleIperfClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IperfClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Server == "" {
		http.Error(w, "Server address required", http.StatusBadRequest)
		return
	}

	req.Protocol = strings.ToLower(req.Protocol)
	if req.Protocol == "" {
		req.Protocol = "tcp"
	}
	if req.Protocol != "tcp" && req.Protocol != "udp" {
		http.Error(w, "protocol must be tcp or udp", http.StatusBadRequest)
		return
	}

	req.Direction = strings.ToLower(req.Direction)
	if req.Direction == "" {
		if req.Reverse {
			req.Direction = "download"
		} else {
			req.Direction = "upload"
		}
	}
	if req.Direction != "upload" && req.Direction != "download" && req.Direction != "bidirectional" {
		http.Error(w, "direction must be upload, download, or bidirectional", http.StatusBadRequest)
		return
	}

	iperfConfig := iperf.ClientConfig{
		Server:    req.Server,
		Port:      req.Port,
		Protocol:  req.Protocol,
		Reverse:   req.Reverse,
		Direction: req.Direction,
		Duration:  req.Duration,
		Parallel:  req.Parallel,
	}

	// Run test in background and return immediately
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Duration+30)*time.Second)
		defer cancel()
		if _, err := s.iperfManager.RunClient(ctx, &iperfConfig); err != nil {
			log.Printf("iperf client failed: %v", err)
		}
	}()

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"message": "iperf3 test started. Poll /api/iperf/client/status for results.",
	})
}

// IperfClientStatusResponse is the status of an iperf3 client test
type IperfClientStatusResponse struct {
	Running  bool                 `json:"running"`
	Phase    string               `json:"phase"`
	Progress float64              `json:"progress"`
	Last     *IperfResultResponse `json:"last,omitempty"`
}

// handleIperfClientStatus returns the status of the iperf3 client test
func (s *Server) handleIperfClientStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.iperfManager.GetClientStatus()
	resp := IperfClientStatusResponse{
		Running:  status.Running,
		Phase:    status.Phase,
		Progress: status.Progress,
	}

	if lastResult := s.iperfManager.GetLastResult(); lastResult != nil {
		resp.Last = &IperfResultResponse{
			Bandwidth:         lastResult.Bandwidth,
			Transfer:          lastResult.Transfer,
			Retransmits:       lastResult.Retransmits,
			Jitter:            lastResult.Jitter,
			LostPackets:       lastResult.LostPackets,
			LostPercent:       lastResult.LostPercent,
			Protocol:          lastResult.Protocol,
			Direction:         lastResult.Direction,
			Duration:          lastResult.Duration,
			Server:            lastResult.Server,
			Port:              lastResult.Port,
			Timestamp:         lastResult.Timestamp.Format(time.RFC3339),
			DownloadBandwidth: lastResult.DownloadBandwidth,
			UploadBandwidth:   lastResult.UploadBandwidth,
			DownloadTransfer:  lastResult.DownloadTransfer,
			UploadTransfer:    lastResult.UploadTransfer,
		}
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// IperfServerRequest is the request body for starting/stopping the iperf3 server
type IperfServerRequest struct {
	Action string `json:"action"` // "start" or "stop"
	Port   int    `json:"port"`
}

// handleIperfServer starts or stops the iperf3 server
func (s *Server) handleIperfServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IperfServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "start":
		port := req.Port
		if port == 0 {
			port = 5201
		}
		if err := s.iperfManager.StartServer(port); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sendJSONResponse(w, http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("iperf3 server started on port %d", port),
			"port":    port,
		})
	case "stop":
		if err := s.iperfManager.StopServer(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sendJSONResponse(w, http.StatusOK, map[string]string{
			"message": "iperf3 server stopped",
		})
	default:
		http.Error(w, "Invalid action (use 'start' or 'stop')", http.StatusBadRequest)
	}
}

// handleIperfServerStatus returns the iperf3 server status
func (s *Server) handleIperfServerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.iperfManager.GetServerStatus()
	sendJSONResponse(w, http.StatusOK, status)
}

// IperfSuggestion represents a discovered host that responds on the iperf port.
type IperfSuggestion struct {
	Host      string  `json:"host"`
	Hostname  string  `json:"hostname,omitempty"`
	Source    string  `json:"source,omitempty"`
	LatencyMs float64 `json:"latencyMs,omitempty"`
}

// handleIperfSuggestions returns discovered devices that respond on the iperf port.
func (s *Server) handleIperfSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.deviceDiscovery == nil {
		http.Error(w, "Device discovery not available", http.StatusServiceUnavailable)
		return
	}

	port := 5201
	if p := r.URL.Query().Get("port"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			port = parsed
		}
	}

	devices := s.deviceDiscovery.GetDevices()
	suggestions := make([]IperfSuggestion, 0, len(devices))

	for _, d := range devices {
		if d.IP == "" {
			continue
		}

		addr := net.JoinHostPort(d.IP, strconv.Itoa(port))
		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, 400*time.Millisecond)
		if err != nil {
			continue
		}
		latency := time.Since(start).Seconds() * 1000
		_ = conn.Close()

		var source string
		if len(d.DiscoveryMethod) > 0 {
			methods := make([]string, 0, len(d.DiscoveryMethod))
			for _, m := range d.DiscoveryMethod {
				methods = append(methods, string(m))
			}
			source = strings.Join(methods, ",")
		}

		suggestions = append(suggestions, IperfSuggestion{
			Host:      d.IP,
			Hostname:  d.Hostname,
			Source:    source,
			LatencyMs: latency,
		})

		if len(suggestions) >= 10 {
			break
		}
	}

	sendJSONResponse(w, http.StatusOK, suggestions)
}

// handleDevices returns all discovered network devices.
func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.deviceDiscovery == nil {
		http.Error(w, "Device discovery not available", http.StatusServiceUnavailable)
		return
	}

	devices := s.deviceDiscovery.GetDevices()
	status := s.deviceDiscovery.GetStatus()

	resp := map[string]interface{}{
		"devices": devices,
		"status":  status,
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleDevicesScan triggers a network device scan.
func (s *Server) handleDevicesScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.deviceDiscovery == nil {
		http.Error(w, "Device discovery not available", http.StatusServiceUnavailable)
		return
	}

	// Check if scan is already in progress
	if s.deviceDiscovery.IsScanning() {
		sendJSONResponse(w, http.StatusOK, map[string]interface{}{
			"message":  "Scan already in progress",
			"scanning": true,
		})
		return
	}

	// Start scan in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := s.deviceDiscovery.Scan(ctx); err != nil {
			log.Printf("Device scan error: %v", err)
		}

		// Notify WebSocket clients when scan completes
		s.wsHub.Broadcast(Message{
			Type: "deviceScanComplete",
			Payload: map[string]interface{}{
				"deviceCount": s.deviceDiscovery.Count(),
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		})
	}()

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":  "Scan started",
		"scanning": true,
	})
}

// handleDevicesStatus returns the current device discovery status.
func (s *Server) handleDevicesStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.deviceDiscovery == nil {
		http.Error(w, "Device discovery not available", http.StatusServiceUnavailable)
		return
	}

	status := s.deviceDiscovery.GetStatus()
	sendJSONResponse(w, http.StatusOK, status)
}

// NetworkDiscoverySettingsResponse represents network discovery settings.
type NetworkDiscoverySettingsResponse struct {
	Enabled        bool   `json:"enabled"`
	ARPScanWorkers int    `json:"arpScanWorkers"`
	PingTimeoutMs  int64  `json:"pingTimeoutMs"`
	ScanTimeoutMs  int64  `json:"scanTimeoutMs"`
	AutoScan       bool   `json:"autoScan"`
	ScanIntervalMs int64  `json:"scanIntervalMs"`
	OUIFilePath    string `json:"ouiFilePath"`
}

// handleDevicesSettings handles GET/PUT for network discovery settings.
func (s *Server) handleDevicesSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getDevicesSettings(w, r)
	case http.MethodPut:
		s.updateDevicesSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getDevicesSettings(w http.ResponseWriter, r *http.Request) {
	resp := NetworkDiscoverySettingsResponse{
		Enabled:        s.config.NetworkDiscovery.Enabled,
		ARPScanWorkers: s.config.NetworkDiscovery.ARPScanWorkers,
		PingTimeoutMs:  s.config.NetworkDiscovery.PingTimeout.Milliseconds(),
		ScanTimeoutMs:  s.config.NetworkDiscovery.ScanTimeout.Milliseconds(),
		AutoScan:       s.config.NetworkDiscovery.AutoScan,
		ScanIntervalMs: s.config.NetworkDiscovery.ScanInterval.Milliseconds(),
		OUIFilePath:    s.config.NetworkDiscovery.OUIFilePath,
	}

	sendJSONResponse(w, http.StatusOK, resp)
}

func (s *Server) updateDevicesSettings(w http.ResponseWriter, r *http.Request) {
	var req NetworkDiscoverySettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update config
	s.config.NetworkDiscovery.Enabled = req.Enabled
	if req.ARPScanWorkers > 0 {
		s.config.NetworkDiscovery.ARPScanWorkers = req.ARPScanWorkers
	}
	if req.PingTimeoutMs > 0 {
		s.config.NetworkDiscovery.PingTimeout = time.Duration(req.PingTimeoutMs) * time.Millisecond
	}
	if req.ScanTimeoutMs > 0 {
		s.config.NetworkDiscovery.ScanTimeout = time.Duration(req.ScanTimeoutMs) * time.Millisecond
	}
	s.config.NetworkDiscovery.AutoScan = req.AutoScan
	s.config.NetworkDiscovery.ScanInterval = time.Duration(req.ScanIntervalMs) * time.Millisecond
	if req.OUIFilePath != "" {
		s.config.NetworkDiscovery.OUIFilePath = req.OUIFilePath
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Network discovery settings updated",
	})
}

// SubnetRequest represents a subnet configuration request.
type SubnetRequest struct {
	CIDR    string `json:"cidr"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// SubnetResponse represents a subnet in API responses.
type SubnetResponse struct {
	CIDR    string `json:"cidr"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// handleDevicesSubnets handles GET/POST/DELETE for additional subnets.
func (s *Server) handleDevicesSubnets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getDevicesSubnets(w, r)
	case http.MethodPost:
		s.addDevicesSubnet(w, r)
	case http.MethodPut:
		s.updateDevicesSubnet(w, r)
	case http.MethodDelete:
		s.deleteDevicesSubnet(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getDevicesSubnets(w http.ResponseWriter, r *http.Request) {
	subnets := make([]SubnetResponse, 0, len(s.config.NetworkDiscovery.AdditionalSubnets))
	for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
		subnets = append(subnets, SubnetResponse{
			CIDR:    subnet.CIDR,
			Name:    subnet.Name,
			Enabled: subnet.Enabled,
		})
	}

	sendJSONResponse(w, http.StatusOK, subnets)
}

func (s *Server) addDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	var req SubnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate CIDR format
	_, _, err := net.ParseCIDR(req.CIDR)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid CIDR format: %v", err), http.StatusBadRequest)
		return
	}

	// Check for duplicates
	for _, existing := range s.config.NetworkDiscovery.AdditionalSubnets {
		if existing.CIDR == req.CIDR {
			http.Error(w, "Subnet already exists", http.StatusConflict)
			return
		}
	}

	// Add the new subnet
	newSubnet := config.SubnetConfig{
		CIDR:    req.CIDR,
		Name:    req.Name,
		Enabled: req.Enabled,
	}
	s.config.NetworkDiscovery.AdditionalSubnets = append(
		s.config.NetworkDiscovery.AdditionalSubnets,
		newSubnet,
	)

	// Update the device discovery scanner
	if s.deviceDiscovery != nil {
		enabledCIDRs := make([]string, 0)
		for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled {
				enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
			}
		}
		if err := s.deviceDiscovery.SetAdditionalSubnets(enabledCIDRs); err != nil {
			log.Printf("Warning: Failed to update scanner subnets: %v", err)
		}
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Subnet added",
	})
}

func (s *Server) updateDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	var req SubnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find and update the subnet
	found := false
	for i, existing := range s.config.NetworkDiscovery.AdditionalSubnets {
		if existing.CIDR == req.CIDR {
			s.config.NetworkDiscovery.AdditionalSubnets[i].Name = req.Name
			s.config.NetworkDiscovery.AdditionalSubnets[i].Enabled = req.Enabled
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Subnet not found", http.StatusNotFound)
		return
	}

	// Update the device discovery scanner
	if s.deviceDiscovery != nil {
		enabledCIDRs := make([]string, 0)
		for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled {
				enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
			}
		}
		if err := s.deviceDiscovery.SetAdditionalSubnets(enabledCIDRs); err != nil {
			log.Printf("Warning: Failed to update scanner subnets: %v", err)
		}
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Subnet updated",
	})
}

func (s *Server) deleteDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	cidr := r.URL.Query().Get("cidr")
	if cidr == "" {
		http.Error(w, "CIDR parameter required", http.StatusBadRequest)
		return
	}

	// Find and remove the subnet
	found := false
	newSubnets := make([]config.SubnetConfig, 0, len(s.config.NetworkDiscovery.AdditionalSubnets))
	for _, existing := range s.config.NetworkDiscovery.AdditionalSubnets {
		if existing.CIDR == cidr {
			found = true
			continue
		}
		newSubnets = append(newSubnets, existing)
	}

	if !found {
		http.Error(w, "Subnet not found", http.StatusNotFound)
		return
	}

	s.config.NetworkDiscovery.AdditionalSubnets = newSubnets

	// Update the device discovery scanner
	if s.deviceDiscovery != nil {
		enabledCIDRs := make([]string, 0)
		for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled {
				enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
			}
		}
		if err := s.deviceDiscovery.SetAdditionalSubnets(enabledCIDRs); err != nil {
			log.Printf("Warning: Failed to update scanner subnets: %v", err)
		}
	}

	// Save config to file
	if err := s.config.Save(s.configPath); err != nil {
		log.Printf("Warning: Failed to save config: %v", err)
	}

	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Subnet deleted",
	})
}

// handlePublicIP returns the public IPv4 and IPv6 addresses.
func (s *Server) handlePublicIP(w http.ResponseWriter, r *http.Request) {
	if s.publicipChecker == nil {
		http.Error(w, "Public IP checker not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Return cached result or fetch if cache expired
		result := s.publicipChecker.GetPublicIP(r.Context())
		sendJSONResponse(w, http.StatusOK, result)

	case http.MethodPost:
		// Force refresh
		result := s.publicipChecker.Refresh(r.Context())
		sendJSONResponse(w, http.StatusOK, result)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleLogs returns the tail of the application log file for troubleshooting.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Optional token gate
	if s.logAccessToken != "" || s.requireLogToken {
		headerName := s.logAccessHeader
		if headerName == "" {
			headerName = "X-Log-Token"
		}
		token := r.Header.Get(headerName)
		if token == "" {
			// allow query param fallback
			token = r.URL.Query().Get("token")
		}
		if token == "" || (s.logAccessToken != "" && token != s.logAccessToken) {
			http.Error(w, "Log access requires token", http.StatusForbidden)
			return
		}
	}

	if s.logPath == "" {
		http.Error(w, "Log path not configured", http.StatusInternalServerError)
		return
	}

	linesParam := r.URL.Query().Get("lines")
	maxLines := 200
	if linesParam != "" {
		if n, err := strconv.Atoi(linesParam); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			maxLines = n
		}
	}

	const maxBytes int64 = 500 * 1024 // limit read size to 500KB
	lines, err := readLastLines(s.logPath, maxBytes, maxLines)
	if err != nil {
		log.Printf("failed to read log file: %v", err)
		http.Error(w, "Failed to read log file", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"path":  s.logPath,
		"lines": lines,
	})
}
