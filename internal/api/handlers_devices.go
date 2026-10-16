package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	discoverysettings "github.com/MustardSeedNetworks/seed/internal/discovery/settings"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/pipeline/publicip"
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

// DevicesResponse is GET /security/devices: every device in the registry
// and the registry's status.
type DevicesResponse struct {
	Devices []*discovery.DiscoveredDevice `json:"devices"`
	Status  DeviceDiscoveryStatusResponse `json:"status"`
}

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

	sendJSONResponse(w, logger, http.StatusOK, DevicesResponse{
		Devices: s.deviceDiscovery().GetDevices(),
		Status:  DeviceDiscoveryStatusResponse(*s.deviceDiscovery().GetStatus()),
	})
}

// DeviceScanResponse is POST /security/devices/scan: the scan runs in the
// background, so the answer only says whether this call started it.
type DeviceScanResponse struct {
	Message  string `json:"message"`
	Scanning bool   `json:"scanning"`
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
		sendJSONResponse(
			w,
			logger,
			http.StatusOK,
			DeviceScanResponse{Message: "Scan already in progress", Scanning: true},
		)
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

		s.afterSweep(ctx, s.discoveryService().Enrich(s.deviceDiscovery().GetDevices()))

		// Notify SSE clients when scan completes
		s.sseHub().Broadcast(Message{
			Type: "deviceScanComplete",
			Payload: map[string]any{
				"deviceCount": s.deviceDiscovery().Count(),
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		})
	}(r.Context())

	sendJSONResponse(w, logger, http.StatusOK, DeviceScanResponse{Message: "Scan started", Scanning: true})
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

// DeviceDiscoveryStatusResponse is GET /security/devices/status: the device
// registry's status served as it is, named for the API.
type DeviceDiscoveryStatusResponse discovery.Status

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

	sendJSONResponse(w, logger, http.StatusOK, DeviceDiscoveryStatusResponse(*s.deviceDiscovery().GetStatus()))
}

// NetworkDiscoverySettingsResponse represents network discovery settings.
type NetworkDiscoverySettingsResponse struct {
	Enabled       bool   `json:"enabled"`
	ScanTimeoutMs int64  `json:"scanTimeoutMs"`
	OUIFilePath   string `json:"ouiFilePath"`

	// Direct options configuration (profiles removed in favor of direct settings).
	Options        OptionsResponse        `json:"options"`
	Timing         TimingResponse         `json:"timing"`
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

// PortScanSettingsResponse represents port scanning settings.
type PortScanSettingsResponse struct {
	Enabled  bool   `json:"enabled"`
	Preset   string `json:"preset"`
	TCPPorts string `json:"tcpPorts"`
}

// OptionsResponse represents discovery options.
type OptionsResponse struct {
	PassiveProtocols PassiveProtocolResponse  `json:"passiveProtocols"`
	ARPScan          bool                     `json:"arpScan"`
	ICMPScan         bool                     `json:"icmpScan"`
	PortScan         PortScanSettingsResponse `json:"portScan"`
	Traceroute       bool                     `json:"traceroute"`
	SNMPQuery        bool                     `json:"snmpQuery"`
}

// TimingResponse represents discovery timing settings.
type TimingResponse struct {
	RescanIntervalMs int64 `json:"rescanIntervalMs"`
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
// the service, and map a bad preset to 400 and a store or reload error to 500.
func (s *Server) updateDevicesSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req NetworkDiscoverySettingsResponse
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}

	err := s.discoverySettings.Update(requestToDiscoveryUpdate(req))
	if errors.Is(err, discoverysettings.ErrInvalidPortPreset) {
		sendErrorResponseWithDetails(
			w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid port-scan preset", "")
		return
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to save discovery settings", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T("errors.settings.saveFailed"), "",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, StatusMessageResponse{
		Status:  statusSuccess,
		Message: "Network discovery settings updated",
	})
}

// discoveryConfigToResponse maps the domain discovery config to the wire DTO
// (duration → milliseconds). Pure transport serialization.
func discoveryConfigToResponse(cfg config.NetworkDiscoveryConfig) NetworkDiscoverySettingsResponse {
	return NetworkDiscoverySettingsResponse{
		Enabled:       cfg.Enabled,
		ScanTimeoutMs: cfg.ScanTimeout.Milliseconds(),
		OUIFilePath:   cfg.OUIFilePath,
		IPv6Enabled:   cfg.IPv6Enabled,
		Options: OptionsResponse{
			PassiveProtocols: PassiveProtocolResponse{
				LLDP: cfg.Options.PassiveProtocols.LLDP,
				CDP:  cfg.Options.PassiveProtocols.CDP,
				EDP:  cfg.Options.PassiveProtocols.EDP,
				NDP:  cfg.Options.PassiveProtocols.NDP,
			},
			ARPScan:  cfg.Options.ARPScan,
			ICMPScan: cfg.Options.ICMPScan,
			PortScan: PortScanSettingsResponse{
				Enabled:  cfg.Options.PortScan.Enabled,
				Preset:   string(cfg.Options.PortScan.Preset),
				TCPPorts: cfg.Options.PortScan.TCPPorts,
			},
			Traceroute: cfg.Options.Traceroute,
			SNMPQuery:  cfg.Options.SNMPQuery,
		},
		Timing: TimingResponse{
			RescanIntervalMs: cfg.Timing.RescanInterval.Milliseconds(),
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
		Enabled:       req.Enabled,
		ScanTimeoutMs: req.ScanTimeoutMs,
		OUIFilePath:   req.OUIFilePath,
		IPv6Enabled:   req.IPv6Enabled,
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
				Enabled:  req.Options.PortScan.Enabled,
				Preset:   req.Options.PortScan.Preset,
				TCPPorts: req.Options.PortScan.TCPPorts,
			},
			Traceroute: req.Options.Traceroute,
			SNMPQuery:  req.Options.SNMPQuery,
		},
		Timing: discoverysettings.TimingUpdate{
			RescanIntervalMs: req.Timing.RescanIntervalMs,
		},
		Fingerprinting: discoverysettings.FingerprintingUpdate{
			Enabled:       req.Fingerprinting.Enabled,
			OSDetection:   req.Fingerprinting.OSDetection,
			ServiceProbes: req.Fingerprinting.ServiceProbes,
		},
	}
}

// PublicIPResponse is GET and POST /telemetry/publicip: the checker's
// result served as it is, named for the API.
type PublicIPResponse publicip.Result

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
		sendJSONResponse(w, logger, http.StatusOK, (*PublicIPResponse)(result))

	case http.MethodPost:
		// Force refresh (fixes #702 - passes context)
		result := s.publicipChecker().Refresh(r.Context())
		sendJSONResponse(w, logger, http.StatusOK, (*PublicIPResponse)(result))

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
