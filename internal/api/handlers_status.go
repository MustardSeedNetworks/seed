// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/system"
	"github.com/krisarmstrong/seed/internal/version"
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
	Version   string           `json:"version"`
	Timestamp string           `json:"timestamp"`
	Device    ExportDeviceInfo `json:"device"`
	Cards     map[string]any   `json:"cards"`
}

// ExportDeviceInfo contains device information.
type ExportDeviceInfo struct {
	Interface string `json:"interface"`
	MAC       string `json:"mac,omitempty"`
	IPMode    string `json:"ipMode"`
}

// handleStatus returns the system status (fixes #544 - split from handlers.go).
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
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

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleExport exports current diagnostic data as JSON (fixes #544 - split from handlers.go).
// Accepts optional query parameter: ?interface=eth0.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
		return
	}

	// Get interface from query param or fallback to current.
	currentIface := s.getInterfaceFromRequest(r)
	if err := s.netManager.RefreshInterfaces(); err != nil {
		logger.Error("Failed to refresh interfaces", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Failed to refresh interfaces",
			"",
		)
		return
	}

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
		Cards: make(map[string]any),
	}

	s.exportLinkCard(currentIface, export.Cards)
	s.exportIPConfigCard(currentIface, export.Cards)
	s.exportDiscoveryCard(export.Cards)
	s.exportDNSCard(r.Context(), export.Cards)
	s.exportGatewayCard(export.Cards)
	s.exportVLANCard(export.Cards)
	s.exportWiFiCard(currentIface, export.Cards)
	s.exportCableCard(export.Cards)
	s.exportSpeedtestCard(export.Cards)
	s.exportIperfCard(export.Cards)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=seed-export.json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(export); err != nil {
		logger.Error("Error encoding export response", "error", err)
	}
}

func (s *Server) exportLinkCard(iface string, cards map[string]any) {
	if linkStatus, err := s.netManager.GetLinkStatus(iface); err == nil {
		cards["link"] = map[string]any{
			"linkUp": linkStatus.LinkUp, "speed": linkStatus.Speed,
			"duplex": linkStatus.Duplex, "autoNeg": linkStatus.AutoNeg,
		}
	}
}

func (s *Server) exportIPConfigCard(iface string, cards map[string]any) {
	ifaceInfo, err := s.netManager.GetInterface(iface)
	if err != nil {
		return
	}
	ipData := map[string]any{"addresses": ifaceInfo.Addresses}
	if leaseInfo, leaseErr := dhcp.GetLeaseInfo(iface); leaseErr == nil && leaseInfo != nil {
		ipData["dhcpServer"] = leaseInfo.DHCPServer
		ipData["gateway"] = leaseInfo.Gateway
		ipData["leaseTime"] = leaseInfo.LeaseTime
		ipData["dns"] = leaseInfo.DNS
	}
	cards["ipConfig"] = ipData
}

func (s *Server) exportDiscoveryCard(cards map[string]any) {
	if s.discoveryService == nil {
		return
	}
	neighbors := s.discoveryService.GetNeighbors()
	neighborList := make([]map[string]any, 0, len(neighbors))
	for _, n := range neighbors {
		neighborList = append(neighborList, map[string]any{
			"protocol": n.Protocol, "systemName": n.SystemName, "portId": n.PortID,
			"portDescription": n.PortDescription, "managementAddress": n.ManagementAddress,
		})
	}
	cards["switch"] = map[string]any{"running": s.discoveryService.IsRunning(), "neighbors": neighborList}
}

func (s *Server) exportDNSCard(ctx context.Context, cards map[string]any) {
	if s.dnsTester == nil {
		return
	}
	result := s.dnsTester.Test(ctx)
	dnsData := map[string]any{"server": result.Server, "testHostname": result.TestHostname}
	if result.Forward != nil {
		dnsData["forward"] = map[string]any{
			"result": result.Forward.Resolved, "time": result.Forward.Time.Milliseconds(),
			"status": result.Forward.Status, "error": result.Forward.Error,
		}
	}
	if result.Reverse != nil {
		dnsData["reverse"] = map[string]any{
			"result": result.Reverse.Resolved, "time": result.Reverse.Time.Milliseconds(),
			"status": result.Reverse.Status, "error": result.Reverse.Error,
		}
	}
	cards["dns"] = dnsData
}

func (s *Server) exportGatewayCard(cards map[string]any) {
	if s.gatewayTester == nil {
		return
	}
	stats := s.gatewayTester.GetStats()
	cards["gateway"] = map[string]any{
		"gateway": stats.Gateway, "reachable": stats.Reachable, "sent": stats.Sent,
		"received": stats.Received, "lossPercent": stats.LossPercent,
		"avgTime": stats.AvgTime, "status": stats.Status,
	}
}

func (s *Server) exportVLANCard(cards map[string]any) {
	if s.vlanManager == nil {
		return
	}
	vlanInfo := s.vlanManager.GetInfo()
	cards["vlan"] = map[string]any{
		"nativeVlan": vlanInfo.NativeVlan, "taggedVlans": vlanInfo.TaggedVlans,
		"voiceVlan": vlanInfo.VoiceVlan, "configured": vlanInfo.Configured,
	}
}

func (s *Server) exportWiFiCard(iface string, cards map[string]any) {
	if !s.netManager.IsWireless(iface) || s.wifiManager == nil {
		return
	}
	wifiInfo := s.wifiManager.GetInfo()
	if wifiInfo.SSID != "" {
		cards["wifi"] = map[string]any{
			"ssid": wifiInfo.SSID, "bssid": wifiInfo.BSSID, "signal": wifiInfo.Signal,
			"channel": wifiInfo.Channel, "frequency": wifiInfo.Frequency, "security": wifiInfo.Security,
		}
	}
}

func (s *Server) exportCableCard(cards map[string]any) {
	if s.cableTester == nil {
		return
	}
	cableResult := s.cableTester.Test()
	cards["cable"] = map[string]any{
		"supported": cableResult.Supported, "length": cableResult.Length,
		"status": cableResult.Status, "faults": cableResult.Faults,
	}
}

func (s *Server) exportSpeedtestCard(cards map[string]any) {
	if s.speedtestTester == nil {
		return
	}
	result := s.speedtestTester.GetLastResult()
	if result == nil {
		return
	}
	cards["speedtest"] = map[string]any{
		"download": result.Download, "upload": result.Upload, "latency": result.Latency,
		"server": result.Server, "location": result.Location, "host": result.Host,
		"distance": result.Distance, "timestamp": result.Timestamp, "testDuration": result.TestDuration,
	}
}

func (s *Server) exportIperfCard(cards map[string]any) {
	if s.iperfManager == nil {
		return
	}
	result := s.iperfManager.GetLastResult()
	if result == nil {
		return
	}
	cards["iperf"] = map[string]any{
		"bandwidth": result.Bandwidth, "transfer": result.Transfer, "retransmits": result.Retransmits,
		"jitter": result.Jitter, "lostPackets": result.LostPackets, "lostPercent": result.LostPercent,
		"protocol": result.Protocol, "direction": result.Direction, "duration": result.Duration,
		"server": result.Server, "port": result.Port, "timestamp": result.Timestamp,
	}
}

// handleLogs returns the tail of the application log file for troubleshooting (fixes #544 - split from handlers.go).
// Requires JWT authentication (enforced by middleware).
// Security fix #301: Removed insecure LOG_ACCESS_TOKEN - JWT authentication is sufficient.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
		return
	}

	// JWT authentication is enforced by the global auth middleware
	// X-Username header is set by the middleware after validating the JWT

	if s.logPath == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Log path not configured",
			"",
		) // fixes #694, #699
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
		logger.Error("Failed to read log file", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Failed to read log file",
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"path":  s.logPath,
		"lines": lines,
	})
}

// handleHealth handles GET /api/health - simple liveness check for load balancers (fixes #540, #544).
// Returns 200 OK if server is running, minimal response for fast health checks.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
		return
	}

	// Simple health check - just return OK
	// For detailed health, use /api/system/health
	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"status": "ok",
		"uptime": time.Since(s.startTime).Seconds(),
	})
}

// handleSystemHealth handles GET /api/system/health - returns comprehensive health metrics (fixes #540, #544).
func (s *Server) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
		return
	}

	// Get system health metrics
	health, err := system.GetHealth()
	if err != nil {
		logger.Error("Failed to get system health", "error", err)
		sendJSONResponse(w, logger, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get system health. Check server logs for details.",
		})
		return
	}

	// Add application-specific health information
	appHealth := map[string]any{
		"system": health,
		"application": map[string]any{
			"version":     version.Version,
			"uptime":      time.Since(s.startTime).Seconds(),
			"uptime_text": time.Since(s.startTime).String(),
		},
		"services": map[string]any{
			"discovery_service": s.discoveryService != nil && s.discoveryService.IsRunning(),
			"link_monitor":      s.linkMonitor != nil,
			"websocket_hub":     s.wsHub != nil,
			"vlan_monitor":      s.vlanTrafficMonitor != nil,
		},
	}

	sendJSONResponse(w, logger, http.StatusOK, appHealth)
}
