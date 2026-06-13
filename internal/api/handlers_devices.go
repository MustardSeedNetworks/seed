package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	discoverysettings "github.com/MustardSeedNetworks/seed/internal/discovery/settings"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Device discovery timing constants.
const (
	// deviceScanTimeoutMin is the timeout in minutes for device scanning operations.
	deviceScanTimeoutMin = 2

	// vulnScanTimeoutMin is the timeout in minutes for vulnerability scanning operations.
	vulnScanTimeoutMin = 5
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

	if s.deviceDiscovery() == nil {
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

	devices := s.deviceDiscovery().GetDevices()
	status := s.deviceDiscovery().GetStatus()

	resp := map[string]any{
		"devices": devices,
		"status":  status,
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleDevicesScan triggers a network device scan (fixes #702 - uses r.Context()).
func (s *Server) handleDevicesScan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if s.deviceDiscovery() == nil {
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
	if s.deviceDiscovery().IsScanning() {
		sendJSONResponse(w, logger, http.StatusOK, map[string]any{
			"message":  "Scan already in progress",
			"scanning": true,
		})
		return
	}

	// Start scan in background (fixes #698 - timeout protection).
	// WithoutCancel inherits logging values from the request context but
	// detaches lifecycle so the scan outlives the HTTP request — users
	// shouldn't be able to kill an in-flight scan by closing their tab.
	go func(reqCtx context.Context) {
		bgLogger := logging.FromContext(reqCtx)
		ctx, cancel := context.WithTimeout(context.WithoutCancel(reqCtx), deviceScanTimeoutMin*time.Minute)
		defer cancel()

		bgLogger.InfoContext(reqCtx, "Starting background device scan")
		start := time.Now()
		defer func() {
			bgLogger.InfoContext(reqCtx,
				"Background device scan finished",
				"duration_ms",
				time.Since(start).Milliseconds(),
			)
		}()

		if err := s.deviceDiscovery().Scan(ctx); err != nil {
			bgLogger.ErrorContext(reqCtx, "Device scan error", "error", err)
		}

		// Auto-scan for vulnerabilities if enabled
		s.postScanVulnerabilityCheck(bgLogger)

		// Notify SSE clients when scan completes
		s.sseHub().Broadcast(Message{
			Type: "deviceScanComplete",
			Payload: map[string]any{
				"deviceCount": s.deviceDiscovery().Count(),
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
	vuln := s.securitySettings.Vuln()
	if s.vulnScanner() == nil || !vuln.Enabled || !vuln.AutoScan {
		return
	}

	logger.Info(
		"Auto-scan: triggering vulnerability scan",
		"device_count",
		s.deviceDiscovery().Count(),
	)
	devices := s.deviceDiscovery().GetDevices()

	vulnCtx, vulnCancel := context.WithTimeout(context.Background(), vulnScanTimeoutMin*time.Minute)
	defer vulnCancel()

	for _, device := range devices {
		if _, err := s.vulnScanner().ScanDevice(vulnCtx, device); err != nil {
			logger.Warn("Auto vulnerability scan failed", "device_ip", device.IP, "error", err)
		}
	}

	// Broadcast vulnerability results
	results := s.vulnScanner().GetAllVulnerabilities()
	s.sseHub().BroadcastCardUpdate("vulnerabilities", map[string]any{
		"results": results,
		"count":   len(results),
	})
	logger.Info("Auto-scan: completed vulnerability scan", "vulnerable_devices", len(results))
}

// handleDevicesStatus returns the current device discovery status (fixes #702 - uses r.Context()).
func (s *Server) handleDevicesStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if s.deviceDiscovery() == nil {
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

	status := s.deviceDiscovery().GetStatus()
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

// getDevicesSettings serves the current network-discovery settings. Thin
// transport (ADR-0020): read from the discovery-settings service, map the domain
// config to the wire DTO (duration → ms).
func (s *Server) getDevicesSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	sendJSONResponse(w, logger, http.StatusOK, discoveryConfigToResponse(s.discoverySettings.Settings()))
}

// updateDevicesSettings persists the network-discovery settings. Thin transport:
// decode the wire DTO, map it to the domain update, delegate the merge/persist to
// the service, and map a store error to 500.
func (s *Server) updateDevicesSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req NetworkDiscoverySettingsResponse
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}

	if err := s.discoverySettings.Update(requestToDiscoveryUpdate(req)); err != nil {
		logger.ErrorContext(r.Context(), "Failed to save discovery settings", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T("errors.settings.saveFailed"), "",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status":  statusSuccess,
		"message": "Network discovery settings updated",
	})
}

// discoveryConfigToResponse maps the domain discovery config to the wire DTO
// (duration → milliseconds). Pure transport serialization.
func discoveryConfigToResponse(cfg config.NetworkDiscoveryConfig) NetworkDiscoverySettingsResponse {
	return NetworkDiscoverySettingsResponse{
		Enabled:        cfg.Enabled,
		ARPScanWorkers: cfg.ARPScanWorkers,
		PingTimeoutMs:  cfg.PingTimeout.Milliseconds(),
		ScanTimeoutMs:  cfg.ScanTimeout.Milliseconds(),
		AutoScan:       cfg.AutoScan,
		ScanIntervalMs: cfg.ScanInterval.Milliseconds(),
		OUIFilePath:    cfg.OUIFilePath,
		IPv6Enabled:    cfg.IPv6Enabled,
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
}

// requestToDiscoveryUpdate maps the wire DTO to the domain update model. Pure
// transport mapping; the field-specific merge rules live in the service.
func requestToDiscoveryUpdate(req NetworkDiscoverySettingsResponse) discoverysettings.Update {
	return discoverysettings.Update{
		Enabled:        req.Enabled,
		ARPScanWorkers: req.ARPScanWorkers,
		PingTimeoutMs:  req.PingTimeoutMs,
		ScanTimeoutMs:  req.ScanTimeoutMs,
		AutoScan:       req.AutoScan,
		ScanIntervalMs: req.ScanIntervalMs,
		OUIFilePath:    req.OUIFilePath,
		IPv6Enabled:    req.IPv6Enabled,
		Options: discoverysettings.OptionsUpdate{
			PassiveProtocols: discoverysettings.PassiveProtocolsUpdate{
				LLDP: req.Options.PassiveProtocols.LLDP,
				CDP:  req.Options.PassiveProtocols.CDP,
				EDP:  req.Options.PassiveProtocols.EDP,
				NDP:  req.Options.PassiveProtocols.NDP,
			},
			ARPScan:  req.Options.ARPScan,
			ICMPScan: req.Options.ICMPScan,
			PortScan: discoverysettings.PortScanUpdate{
				Enabled:         req.Options.PortScan.Enabled,
				TCPPorts:        req.Options.PortScan.TCPPorts,
				UDPPorts:        req.Options.PortScan.UDPPorts,
				BannerTimeoutMs: req.Options.PortScan.BannerTimeoutMs,
			},
			TCPProbe: discoverysettings.TCPProbeUpdate{
				TimeoutMs: req.Options.TCPProbe.TimeoutMs,
				Workers:   req.Options.TCPProbe.Workers,
			},
			Traceroute: req.Options.Traceroute,
			SNMPQuery:  req.Options.SNMPQuery,
		},
		Timing: discoverysettings.TimingUpdate{
			ProbeIntervalMs:  req.Timing.ProbeIntervalMs,
			RescanIntervalMs: req.Timing.RescanIntervalMs,
			Workers:          req.Timing.Workers,
		},
		Profiler: discoverysettings.ProfilerUpdate{
			Enabled:       req.Profiler.Enabled,
			TimeoutMs:     req.Profiler.TimeoutMs,
			MaxConcurrent: req.Profiler.MaxConcurrent,
			QuickPorts:    req.Profiler.QuickPorts,
		},
		Fingerprinting: discoverysettings.FingerprintingUpdate{
			Enabled:       req.Fingerprinting.Enabled,
			OSDetection:   req.Fingerprinting.OSDetection,
			ServiceProbes: req.Fingerprinting.ServiceProbes,
		},
	}
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

// getDevicesSubnets lists the configured additional subnets. Thin transport.
func (s *Server) getDevicesSubnets(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	cfgSubnets := s.discoverySettings.Subnets()
	subnets := make([]SubnetResponse, 0, len(cfgSubnets))
	for _, subnet := range cfgSubnets {
		subnets = append(subnets, SubnetResponse{
			CIDR:    subnet.CIDR,
			Name:    subnet.Name,
			Enabled: subnet.Enabled,
		})
	}
	sendJSONResponse(w, logger, http.StatusOK, subnets)
}

func (s *Server) addDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req SubnetRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}

	err := s.discoverySettings.AddSubnet(config.SubnetConfig{
		CIDR: req.CIDR, Name: req.Name, Enabled: req.Enabled,
	})
	s.writeSubnetResult(w, r, err, "Subnet added")
}

func (s *Server) updateDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req SubnetRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}

	err := s.discoverySettings.UpdateSubnet(config.SubnetConfig{
		CIDR: req.CIDR, Name: req.Name, Enabled: req.Enabled,
	})
	s.writeSubnetResult(w, r, err, "Subnet updated")
}

func (s *Server) deleteDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	cidr := r.URL.Query().Get("cidr")
	if cidr == "" {
		sendErrorResponseWithDetails(
			w, logger, http.StatusBadRequest, ErrCodeBadRequest, "CIDR parameter required", "",
		)
		return
	}
	s.writeSubnetResult(w, r, s.discoverySettings.DeleteSubnet(cidr), "Subnet deleted")
}

// writeSubnetResult maps a discovery-settings subnet error to an HTTP response,
// or writes the success message. Centralizes the validation/conflict/not-found/
// save error mapping shared by the subnet add/update/delete handlers.
func (s *Server) writeSubnetResult(
	w http.ResponseWriter, r *http.Request, err error, successMsg string,
) {
	logger := logging.FromContext(r.Context())
	switch {
	case errors.Is(err, discoverysettings.ErrInvalidCIDR):
		sendErrorResponseWithDetails(
			w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid CIDR format", "")
	case errors.Is(err, discoverysettings.ErrSubnetExists):
		sendErrorResponseWithDetails(
			w, logger, http.StatusConflict, ErrCodeConflict, "Subnet already exists", "")
	case errors.Is(err, discoverysettings.ErrSubnetNotFound):
		sendErrorResponseWithDetails(
			w, logger, http.StatusNotFound, ErrCodeNotFound, "Subnet not found", "")
	case err != nil:
		logger.ErrorContext(r.Context(), "Failed to save subnet change", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			i18n.FromRequest(r).T("errors.settings.saveFailed"), "")
	default:
		sendJSONResponse(w, logger, http.StatusOK, map[string]string{
			"status": statusSuccess, "message": successMsg,
		})
	}
}

// handlePublicIP returns the public IPv4 and IPv6 addresses (fixes #702 - uses r.Context() for service calls).
func (s *Server) handlePublicIP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if s.publicipChecker() == nil {
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
		result := s.publicipChecker().GetPublicIP(r.Context())
		sendJSONResponse(w, logger, http.StatusOK, result)

	case http.MethodPost:
		// Force refresh (fixes #702 - passes context)
		result := s.publicipChecker().Refresh(r.Context())
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
