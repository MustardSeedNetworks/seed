// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/dhcp"
	"github.com/krisarmstrong/luminetiq/internal/system"
	"github.com/krisarmstrong/luminetiq/internal/version"
)

// StatusResponse represents the system status.
type StatusResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	Uptime        int64  `json:"uptime"`
	Interface     string `json:"interface"`
	IsWireless    bool   `json:"isWireless"`
	ICMPAvailable bool   `json:"icmpAvailable"`
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

// handleStatus returns the system status (fixes #544 - split from handlers.go).
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

// handleExport exports current diagnostic data as JSON (fixes #544 - split from handlers.go).
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

// handleLogs returns the tail of the application log file for troubleshooting (fixes #544 - split from handlers.go).
// Requires JWT authentication (enforced by middleware).
// Optionally requires an additional log access token for extra protection.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// JWT authentication is enforced by the global auth middleware
	// X-Username header is set by the middleware after validating the JWT

	// OPTIONAL: Additional log access token for extra protection
	if s.logAccessToken != "" {
		headerName := s.logAccessHeader
		if headerName == "" {
			headerName = "X-Log-Token"
		}
		token := r.Header.Get(headerName)
		if token == "" {
			// Allow query param fallback (deprecated, for backwards compatibility)
			token = r.URL.Query().Get("log_token")
		}
		if token != s.logAccessToken {
			http.Error(w, "Additional log access token required", http.StatusForbidden)
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

// handleHealth handles GET /api/health - simple liveness check for load balancers (fixes #540, #544).
// Returns 200 OK if server is running, minimal response for fast health checks.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simple health check - just return OK
	// For detailed health, use /api/system/health
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"uptime": time.Since(s.startTime).Seconds(),
	})
}

// handleSystemHealth handles GET /api/system/health - returns comprehensive health metrics (fixes #540, #544).
func (s *Server) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get system health metrics
	health, err := system.GetHealth()
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get system health: " + err.Error(),
		})
		return
	}

	// Add application-specific health information
	appHealth := map[string]interface{}{
		"system": health,
		"application": map[string]interface{}{
			"version":     version.Version,
			"uptime":      time.Since(s.startTime).Seconds(),
			"uptime_text": time.Since(s.startTime).String(),
		},
		"services": map[string]interface{}{
			"discovery_service": s.discoveryService != nil && s.discoveryService.IsRunning(),
			"link_monitor":      s.linkMonitor != nil,
			"websocket_hub":     s.wsHub != nil,
			"vlan_monitor":      s.vlanTrafficMonitor != nil,
		},
	}

	sendJSONResponse(w, http.StatusOK, appHealth)
}
