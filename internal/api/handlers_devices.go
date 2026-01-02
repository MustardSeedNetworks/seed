// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Device Discovery Handlers (fixes #544 - split from handlers_discovery.go)
// ============================================================================

// handleDevices returns discovered devices and status (fixes #702 - uses r.Context()).
func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
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

	if s.deviceDiscovery == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Device discovery not available",
			"",
		) // fixes #694, #699
		return
	}

	devices := s.deviceDiscovery.GetDevices()
	status := s.deviceDiscovery.GetStatus()

	resp := map[string]any{
		"devices": devices,
		"status":  status,
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleDevicesScan triggers a network device scan (fixes #702 - uses r.Context()).
func (s *Server) handleDevicesScan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method != http.MethodPost {
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

	if s.deviceDiscovery == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Device discovery not available",
			"",
		) // fixes #694, #699
		return
	}

	// Check if scan is already in progress
	if s.deviceDiscovery.IsScanning() {
		sendJSONResponse(w, logger, http.StatusOK, map[string]any{
			"message":  "Scan already in progress",
			"scanning": true,
		})
		return
	}

	// Start scan in background (fixes #698 - timeout protection)
	go func(reqCtx context.Context) {
		bgLogger := logging.FromContext(reqCtx)
		// Add timeout protection for device scan operations (fixes #698)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		bgLogger.Info("Starting background device scan")
		start := time.Now()
		defer func() {
			bgLogger.Info("Background device scan finished", "duration_ms", time.Since(start).Milliseconds())
		}()

		if err := s.deviceDiscovery.Scan(ctx); err != nil {
			bgLogger.Error("Device scan error", "error", err)
		}

		// Auto-scan for vulnerabilities if enabled
		s.postScanVulnerabilityCheck(bgLogger)

		// Notify WebSocket clients when scan completes
		s.wsHub.Broadcast(Message{
			Type: "deviceScanComplete",
			Payload: map[string]any{
				"deviceCount": s.deviceDiscovery.Count(),
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		})
	}(r.Context())

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"message":  "Scan started",
		"scanning": true,
	})
}

// postScanVulnerabilityCheck runs vulnerability scans after device discovery if auto-scan is enabled.
// This method extracts business logic from the handler for better separation of concerns.
func (s *Server) postScanVulnerabilityCheck(logger *slog.Logger) {
	if s.vulnScanner == nil || !s.config.Security.VulnerabilityScanning.Enabled ||
		!s.config.Security.VulnerabilityScanning.AutoScan {
		return
	}

	logger.Info("Auto-scan: triggering vulnerability scan", "device_count", s.deviceDiscovery.Count())
	devices := s.deviceDiscovery.GetDevices()

	vulnCtx, vulnCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer vulnCancel()

	for _, device := range devices {
		if _, err := s.vulnScanner.ScanDevice(vulnCtx, device); err != nil {
			logger.Warn("Auto vulnerability scan failed", "device_ip", device.IP, "error", err)
		}
	}

	// Broadcast vulnerability results
	results := s.vulnScanner.GetAllVulnerabilities()
	s.wsHub.BroadcastCardUpdate("vulnerabilities", map[string]any{
		"results": results,
		"count":   len(results),
	})
	logger.Info("Auto-scan: completed vulnerability scan", "vulnerable_devices", len(results))
}

// handleDevicesStatus returns the current device discovery status (fixes #702 - uses r.Context()).
func (s *Server) handleDevicesStatus(w http.ResponseWriter, r *http.Request) {
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

	if s.deviceDiscovery == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Device discovery not available",
			"",
		) // fixes #694, #699
		return
	}

	status := s.deviceDiscovery.GetStatus()
	sendJSONResponse(w, logger, http.StatusOK, status)
}

// NetworkDiscoverySettingsResponse represents network discovery settings.
type NetworkDiscoverySettingsResponse struct {
	// Legacy fields (backward compatibility)
	Enabled        bool   `json:"enabled"`
	ARPScanWorkers int    `json:"arpScanWorkers"`
	PingTimeoutMs  int64  `json:"pingTimeoutMs"`
	ScanTimeoutMs  int64  `json:"scanTimeoutMs"`
	AutoScan       bool   `json:"autoScan"`
	ScanIntervalMs int64  `json:"scanIntervalMs"`
	OUIFilePath    string `json:"ouiFilePath"`

	// Direct options configuration (profiles removed in favor of direct settings).
	Options        OptionsResponse        `json:"options"`
	Timing         TimingResponse         `json:"timing"`
	Profiler       ProfilerResponse       `json:"profiler"`
	Fingerprinting FingerprintingResponse `json:"fingerprinting"`
	IPv6Enabled    bool                   `json:"ipv6Enabled"`
}

// PassiveProtocolResponse represents granular passive protocol settings.
type PassiveProtocolResponse struct {
	LLDP bool `json:"lldp"`
	CDP  bool `json:"cdp"`
	EDP  bool `json:"edp"`
	NDP  bool `json:"ndp"`
}

// PortScanResponse represents port scanning settings.
type PortScanResponse struct {
	Enabled         bool   `json:"enabled"`
	TCPPorts        string `json:"tcpPorts"`
	UDPPorts        string `json:"udpPorts"`
	BannerTimeoutMs int64  `json:"bannerTimeoutMs"`
}

// TCPProbeSettingsResponse represents TCP probe settings in the discovery config.
type TCPProbeSettingsResponse struct {
	TimeoutMs int64 `json:"timeoutMs"`
	Workers   int   `json:"workers"`
}

// OptionsResponse represents discovery options.
type OptionsResponse struct {
	PassiveProtocols PassiveProtocolResponse  `json:"passiveProtocols"`
	ARPScan          bool                     `json:"arpScan"`
	ICMPScan         bool                     `json:"icmpScan"`
	PortScan         PortScanResponse         `json:"portScan"`
	TCPProbe         TCPProbeSettingsResponse `json:"tcpProbe"`
	Traceroute       bool                     `json:"traceroute"`
	SNMPQuery        bool                     `json:"snmpQuery"`
}

// TimingResponse represents discovery timing settings.
type TimingResponse struct {
	ProbeIntervalMs  int64 `json:"probeIntervalMs"`
	RescanIntervalMs int64 `json:"rescanIntervalMs"`
	Workers          int   `json:"workers"`
}

// ProfilerResponse represents device profiler settings.
type ProfilerResponse struct {
	Enabled       bool  `json:"enabled"`
	TimeoutMs     int64 `json:"timeoutMs"`
	MaxConcurrent int   `json:"maxConcurrent"`
	QuickPorts    []int `json:"quickPorts"`
}

// FingerprintingResponse represents fingerprinting settings.
type FingerprintingResponse struct {
	Enabled       bool `json:"enabled"`
	OSDetection   bool `json:"osDetection"`
	ServiceProbes bool `json:"serviceProbes"`
}

// handleDevicesSettings handles GET/PUT for network discovery settings (fixes #702 - uses r.Context()).
func (s *Server) handleDevicesSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		s.getDevicesSettings(w, r)
	case http.MethodPut:
		s.updateDevicesSettings(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
	}
}

func (s *Server) getDevicesSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	cfg := s.config.NetworkDiscovery

	resp := NetworkDiscoverySettingsResponse{
		// Legacy fields (backward compatibility)
		Enabled:        cfg.Enabled,
		ARPScanWorkers: cfg.ARPScanWorkers,
		PingTimeoutMs:  cfg.PingTimeout.Milliseconds(),
		ScanTimeoutMs:  cfg.ScanTimeout.Milliseconds(),
		AutoScan:       cfg.AutoScan,
		ScanIntervalMs: cfg.ScanInterval.Milliseconds(),
		OUIFilePath:    cfg.OUIFilePath,

		// Direct options configuration (profiles removed).
		IPv6Enabled: cfg.IPv6Enabled,
		Options: OptionsResponse{
			PassiveProtocols: PassiveProtocolResponse{
				LLDP: cfg.Options.PassiveProtocols.LLDP,
				CDP:  cfg.Options.PassiveProtocols.CDP,
				EDP:  cfg.Options.PassiveProtocols.EDP,
				NDP:  cfg.Options.PassiveProtocols.NDP,
			},
			ARPScan:  cfg.Options.ARPScan,
			ICMPScan: cfg.Options.ICMPScan,
			PortScan: PortScanResponse{
				Enabled:         cfg.Options.PortScan.Enabled,
				TCPPorts:        cfg.Options.PortScan.TCPPorts,
				UDPPorts:        cfg.Options.PortScan.UDPPorts,
				BannerTimeoutMs: cfg.Options.PortScan.BannerTimeout.Milliseconds(),
			},
			TCPProbe: TCPProbeSettingsResponse{
				TimeoutMs: cfg.Options.TCPProbe.Timeout.Milliseconds(),
				Workers:   cfg.Options.TCPProbe.Workers,
			},
			Traceroute: cfg.Options.Traceroute,
			SNMPQuery:  cfg.Options.SNMPQuery,
		},
		Timing: TimingResponse{
			ProbeIntervalMs:  cfg.Timing.ProbeInterval.Milliseconds(),
			RescanIntervalMs: cfg.Timing.RescanInterval.Milliseconds(),
			Workers:          cfg.Timing.Workers,
		},
		Profiler: ProfilerResponse{
			Enabled:       cfg.Profiler.Enabled,
			TimeoutMs:     cfg.Profiler.Timeout.Milliseconds(),
			MaxConcurrent: cfg.Profiler.MaxConcurrent,
			QuickPorts:    cfg.Profiler.QuickPorts,
		},
		Fingerprinting: FingerprintingResponse{
			Enabled:       cfg.Fingerprinting.Enabled,
			OSDetection:   cfg.Fingerprinting.OSDetection,
			ServiceProbes: cfg.Fingerprinting.ServiceProbes,
		},
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// applyLegacyDiscoveryConfig applies legacy/backward-compatible discovery settings.
func (s *Server) applyLegacyDiscoveryConfig(req *NetworkDiscoverySettingsResponse) {
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
	s.config.NetworkDiscovery.IPv6Enabled = req.IPv6Enabled
}

// applyDiscoveryOptions applies discovery options including protocols, port scan, and TCP probe.
func (s *Server) applyDiscoveryOptions(req *NetworkDiscoverySettingsResponse) {
	opts := &s.config.NetworkDiscovery.Options
	opts.PassiveProtocols.LLDP = req.Options.PassiveProtocols.LLDP
	opts.PassiveProtocols.CDP = req.Options.PassiveProtocols.CDP
	opts.PassiveProtocols.EDP = req.Options.PassiveProtocols.EDP
	opts.PassiveProtocols.NDP = req.Options.PassiveProtocols.NDP
	opts.ARPScan = req.Options.ARPScan
	opts.ICMPScan = req.Options.ICMPScan
	opts.Traceroute = req.Options.Traceroute
	opts.SNMPQuery = req.Options.SNMPQuery

	// Port scan config
	opts.PortScan.Enabled = req.Options.PortScan.Enabled
	if req.Options.PortScan.TCPPorts != "" {
		opts.PortScan.TCPPorts = req.Options.PortScan.TCPPorts
	}
	if req.Options.PortScan.UDPPorts != "" {
		opts.PortScan.UDPPorts = req.Options.PortScan.UDPPorts
	}
	if req.Options.PortScan.BannerTimeoutMs > 0 {
		opts.PortScan.BannerTimeout = time.Duration(req.Options.PortScan.BannerTimeoutMs) * time.Millisecond
	}

	// TCP probe config
	if req.Options.TCPProbe.TimeoutMs > 0 {
		opts.TCPProbe.Timeout = time.Duration(req.Options.TCPProbe.TimeoutMs) * time.Millisecond
	}
	if req.Options.TCPProbe.Workers > 0 {
		opts.TCPProbe.Workers = req.Options.TCPProbe.Workers
	}
}

// applyAdvancedDiscoverySettings applies timing, profiler, and fingerprinting settings.
func (s *Server) applyAdvancedDiscoverySettings(req *NetworkDiscoverySettingsResponse) {
	// Timing config
	if req.Timing.ProbeIntervalMs > 0 {
		s.config.NetworkDiscovery.Timing.ProbeInterval = time.Duration(req.Timing.ProbeIntervalMs) * time.Millisecond
	}
	if req.Timing.RescanIntervalMs > 0 {
		s.config.NetworkDiscovery.Timing.RescanInterval = time.Duration(req.Timing.RescanIntervalMs) * time.Millisecond
	}
	if req.Timing.Workers > 0 {
		s.config.NetworkDiscovery.Timing.Workers = req.Timing.Workers
	}

	// Profiler config
	s.config.NetworkDiscovery.Profiler.Enabled = req.Profiler.Enabled
	if req.Profiler.TimeoutMs > 0 {
		s.config.NetworkDiscovery.Profiler.Timeout = time.Duration(req.Profiler.TimeoutMs) * time.Millisecond
	}
	if req.Profiler.MaxConcurrent > 0 {
		s.config.NetworkDiscovery.Profiler.MaxConcurrent = req.Profiler.MaxConcurrent
	}
	if len(req.Profiler.QuickPorts) > 0 {
		s.config.NetworkDiscovery.Profiler.QuickPorts = req.Profiler.QuickPorts
	}

	// Fingerprinting config
	s.config.NetworkDiscovery.Fingerprinting.Enabled = req.Fingerprinting.Enabled
	s.config.NetworkDiscovery.Fingerprinting.OSDetection = req.Fingerprinting.OSDetection
	s.config.NetworkDiscovery.Fingerprinting.ServiceProbes = req.Fingerprinting.ServiceProbes
}

func (s *Server) updateDevicesSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req NetworkDiscoverySettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body", "")
		return
	}

	// Lock config for write access (fixes #759 - race condition)
	// NOTE: Must unlock before Save() - Save() acquires RLock internally (fixes #783)
	s.config.Lock()
	s.applyLegacyDiscoveryConfig(&req)
	s.applyDiscoveryOptions(&req)
	s.applyAdvancedDiscoverySettings(&req)
	s.config.Unlock()

	// Save config to file (fixes #735 - return error on save failure)
	if err := s.config.Save(s.configPath); err != nil {
		logger.Error("Failed to save config", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T("errors.settings.saveFailed"), "",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
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

// handleDevicesSubnets handles GET/POST/DELETE for additional subnets (fixes #702 - uses r.Context()).
func (s *Server) handleDevicesSubnets(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
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
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
	}
}

func (s *Server) getDevicesSubnets(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	subnets := make([]SubnetResponse, 0, len(s.config.NetworkDiscovery.AdditionalSubnets))
	for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
		subnets = append(subnets, SubnetResponse{
			CIDR:    subnet.CIDR,
			Name:    subnet.Name,
			Enabled: subnet.Enabled,
		})
	}

	sendJSONResponse(w, logger, http.StatusOK, subnets)
}

func (s *Server) addDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req SubnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body", "")
		return
	}

	// Validate CIDR format
	_, _, err := net.ParseCIDR(req.CIDR)
	if err != nil {
		logger.Warn("Invalid CIDR format", "error", err, "cidr", req.CIDR)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid CIDR format", "")
		return
	}

	// Check for duplicates
	for _, existing := range s.config.NetworkDiscovery.AdditionalSubnets {
		if existing.CIDR == req.CIDR {
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusConflict,
				ErrCodeConflict,
				"Subnet already exists",
				"",
			) // fixes #694, #699
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
	s.syncDeviceDiscoverySubnets(logger)

	// Save config to file (fixes #735 - return error on save failure)
	if saveErr := s.config.Save(s.configPath); saveErr != nil {
		logger.Error("Failed to save config", "error", saveErr)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.settings.saveFailed"),
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Subnet added",
	})
}

func (s *Server) updateDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	// Limit request body size to prevent DoS attacks (fixes #693)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req SubnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid request body", "")
		return
	}

	// Validate CIDR format
	if _, _, err := net.ParseCIDR(req.CIDR); err != nil {
		logger.Warn("Invalid CIDR format", "error", err, "cidr", req.CIDR)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid CIDR format", "")
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
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusNotFound,
			ErrCodeNotFound,
			"Subnet not found",
			"",
		) // fixes #694, #699
		return
	}

	// Update the device discovery scanner
	s.syncDeviceDiscoverySubnets(logger)

	// Save config to file (fixes #735 - return error on save failure)
	if err := s.config.Save(s.configPath); err != nil {
		logger.Error("Failed to save config", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.settings.saveFailed"),
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Subnet updated",
	})
}

func (s *Server) deleteDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	cidr := r.URL.Query().Get("cidr")
	if cidr == "" {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"CIDR parameter required",
			"",
		) // fixes #694, #699
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
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusNotFound,
			ErrCodeNotFound,
			"Subnet not found",
			"",
		) // fixes #694, #699
		return
	}

	s.config.NetworkDiscovery.AdditionalSubnets = newSubnets

	// Update the device discovery scanner
	s.syncDeviceDiscoverySubnets(logger)

	// Save config to file (fixes #735 - return error on save failure)
	if err := s.config.Save(s.configPath); err != nil {
		logger.Error("Failed to save config", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.settings.saveFailed"),
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Subnet deleted",
	})
}

// syncDeviceDiscoverySubnets synchronizes enabled subnets from config to the device discovery scanner.
// This helper method eliminates DRY violation across add/update/delete subnet handlers.
func (s *Server) syncDeviceDiscoverySubnets(logger *slog.Logger) {
	if s.deviceDiscovery == nil {
		return
	}

	enabledCIDRs := make([]string, 0, len(s.config.NetworkDiscovery.AdditionalSubnets))
	for _, subnet := range s.config.NetworkDiscovery.AdditionalSubnets {
		if subnet.Enabled {
			enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
		}
	}

	if err := s.deviceDiscovery.SetAdditionalSubnets(enabledCIDRs); err != nil {
		logger.Warn("Failed to update scanner subnets", "error", err)
	}
}

// handlePublicIP returns the public IPv4 and IPv6 addresses (fixes #702 - uses r.Context() for service calls).
func (s *Server) handlePublicIP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if s.publicipChecker == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Public IP checker not available",
			"",
		) // fixes #694, #699
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Return cached result or fetch if cache expired (fixes #702 - passes context)
		result := s.publicipChecker.GetPublicIP(r.Context())
		sendJSONResponse(w, logger, http.StatusOK, result)

	case http.MethodPost:
		// Force refresh (fixes #702 - passes context)
		result := s.publicipChecker.Refresh(r.Context())
		sendJSONResponse(w, logger, http.StatusOK, result)

	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
	}
}
