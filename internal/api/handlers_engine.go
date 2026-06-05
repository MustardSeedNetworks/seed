package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Discovery Engine Handlers (new unified discovery system)
// ============================================================================

// EngineDiscoveryResponse contains discovery engine results.
type EngineDiscoveryResponse struct {
	Devices      []*discovery.DiscoveredDevice `json:"devices"`
	Stats        *discovery.EngineStats        `json:"stats"`
	ScanResult   *discovery.ScanResult         `json:"scanResult,omitempty"`
	Capabilities map[string]bool               `json:"capabilities"`
}

// EngineScanRequest contains options for an engine scan.
type EngineScanRequest struct {
	// Scan type: "quick" or "full"
	ScanType string `json:"scanType"`

	// Discovery options
	IncludeWired     bool `json:"includeWired"`
	IncludeWiFi      bool `json:"includeWifi"`
	IncludeBluetooth bool `json:"includeBluetooth"`

	// Enrichment options (full scan)
	IncludeSNMP     bool `json:"includeSnmp"`
	IncludePortScan bool `json:"includePortScan"`
	IncludeVulnScan bool `json:"includeVulnScan"`

	// Fresh scan triggers
	FreshWiredScan     bool `json:"freshWiredScan"`
	FreshWiFiScan      bool `json:"freshWifiScan"`
	FreshBluetoothScan bool `json:"freshBluetoothScan"`

	// Port-scan + timing config, folded from the pipeline orchestrator
	// (ADR-0007, Phase 7 S4). Empty PortScanIntensity = engine default
	// (profiler config left unchanged). Intensity is one of
	// off/quick/standard/comprehensive/custom; TimingProfile is one of
	// polite/normal/aggressive.
	PortScanIntensity   string `json:"portScanIntensity,omitempty"`
	PortScanCustomPorts []int  `json:"portScanCustomPorts,omitempty"`
	TimingProfile       string `json:"timingProfile,omitempty"`

	// AcknowledgeIDsRisk is the job-params equivalent of the
	// X-Acknowledge-Ids-Risk header the pipeline endpoint requires for
	// comprehensive scans (which may trip IDS/IPS). The direct
	// /discovery/engine/scan handler reads the header; the engine-scan job
	// reads this field, since a job carries no request headers to its run.
	AcknowledgeIDsRisk bool `json:"acknowledgeIdsRisk,omitempty"`
}

// handleEngineDiscovery returns all devices from the discovery engine registry.
//
// GET /api/v1/discovery/engine returns all discovered devices.
//
// The response includes:
// - All devices from the unified registry
// - Engine statistics
// - Last scan result summary
// - Available capabilities
//
// Authentication: Required
// Rate limiting: None (read-only operation).
func (s *Server) handleEngineDiscovery(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	engine := s.services.Discovery.Engine
	if engine == nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Discovery engine not available",
			"",
		)
		return
	}

	resp := EngineDiscoveryResponse{
		Devices:      engine.GetDevices(),
		Stats:        engine.GetStats(),
		ScanResult:   engine.GetLastScan(),
		Capabilities: engine.GetCapabilities(),
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// maxEngineCustomPorts bounds an engine-scan custom port list so a single
// request can't queue an unbounded scan.
const maxEngineCustomPorts = 1024

// errIDsRiskUnacknowledged signals that a comprehensive scan was requested
// without the IDS-risk acknowledgment. Callers map it to 428 (the direct
// endpoint) or surface it as a job error; ordinary validation failures map to
// 400.
var errIDsRiskUnacknowledged = errors.New(
	"comprehensive port scanning may trigger IDS/IPS alerts; acknowledge the risk to proceed",
)

// validateEngineScanConfig enforces, for the engine scan paths, the same guards
// the pipeline applies to risky scan configuration (ADR-0007 fold, S4):
//   - comprehensive intensity requires an explicit IDS-risk acknowledgment
//     (acknowledged = X-Acknowledge-Ids-Risk header on the direct endpoint, or
//     the acknowledgeIdsRisk param on the engine-scan job);
//   - custom ports are only valid with custom intensity, must each be in
//     1..65535, and are bounded in number.
func validateEngineScanConfig(req EngineScanRequest, acknowledged bool) error {
	if req.PortScanIntensity == string(discovery.PortScanComprehensive) && !acknowledged {
		return errIDsRiskUnacknowledged
	}
	if len(req.PortScanCustomPorts) == 0 {
		return nil
	}
	if req.PortScanIntensity != string(discovery.PortScanCustom) {
		return fmt.Errorf("custom ports require portScanIntensity=%q", discovery.PortScanCustom)
	}
	if len(req.PortScanCustomPorts) > maxEngineCustomPorts {
		return fmt.Errorf("too many custom ports: %d (max %d)",
			len(req.PortScanCustomPorts), maxEngineCustomPorts)
	}
	for _, p := range req.PortScanCustomPorts {
		if p < 1 || p > 65535 {
			return fmt.Errorf("custom port out of range: %d (must be 1..65535)", p)
		}
	}
	return nil
}

// dedupePorts returns the input with duplicates removed, order preserved.
func dedupePorts(ports []int) []int {
	if len(ports) == 0 {
		return ports
	}
	seen := make(map[int]struct{}, len(ports))
	out := make([]int, 0, len(ports))
	for _, p := range ports {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// scanOptsFromRequest builds discovery scan options from a request: the
// quick/full default plus the per-feature overrides. Shared by the engine-scan
// handler and the engine-scan job kind so both produce identical options.
func scanOptsFromRequest(req EngineScanRequest) *discovery.ScanOptions {
	var opts *discovery.ScanOptions
	if req.ScanType == "full" {
		opts = discovery.DefaultFullScanOpts()
	} else {
		opts = discovery.DefaultQuickScanOpts()
	}
	opts.IncludeWired = req.IncludeWired
	opts.IncludeWiFi = req.IncludeWiFi
	opts.IncludeBluetooth = req.IncludeBluetooth
	opts.IncludeSNMP = req.IncludeSNMP
	opts.IncludePortScan = req.IncludePortScan
	opts.IncludeVulnScan = req.IncludeVulnScan
	opts.FreshWiredScan = req.FreshWiredScan
	opts.FreshWiFiScan = req.FreshWiFiScan
	opts.FreshBluetoothScan = req.FreshBluetoothScan
	opts.PortScanIntensity = discovery.PortScanIntensity(req.PortScanIntensity)
	opts.PortScanCustomPorts = dedupePorts(req.PortScanCustomPorts)
	opts.TimingProfile = discovery.ScanTimingProfile(req.TimingProfile)
	return opts
}

// parseEngineScanOpts decodes and validates the optional scan body, applying
// the same IDS-risk + custom-port guards as the engine-scan job. On a bad or
// unacknowledged request it writes the error response (428 for the IDS gate,
// 400 otherwise) and returns ok=false. An empty body yields a quick scan.
func parseEngineScanOpts(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
) (*discovery.ScanOptions, bool) {
	if r.ContentLength <= 0 {
		return discovery.DefaultQuickScanOpts(), true
	}
	// Note: previously this handler leaked the json parser error into the
	// `details` field; the strict helper drops that (the structured WARN log
	// retains the diagnostic).
	var req EngineScanRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return nil, false
	}
	acknowledged := r.Header.Get("X-Acknowledge-Ids-Risk") == "true"
	if err := validateEngineScanConfig(req, acknowledged); err != nil {
		status, code := http.StatusBadRequest, ErrCodeValidation
		if errors.Is(err, errIDsRiskUnacknowledged) {
			status, code = http.StatusPreconditionRequired, ErrCodePreconditionFail
		}
		sendErrorResponseWithDetails(w, logger, status, code, err.Error(), "")
		return nil, false
	}
	return scanOptsFromRequest(req), true
}

// handleEngineScan triggers a discovery engine scan.
//
// POST /api/v1/discovery/engine/scan triggers a new scan.
// Without a body, performs a quick scan.
// With a body, can specify scan options.
//
// Quick scan: Correlation of existing data, fast
// Full scan: Fresh discovery + enrichment + assessment
//
// Request body (optional):
//
//	{
//	  "scanType": "quick",          // or "full"
//	  "includeWired": true,
//	  "includeWifi": true,
//	  "includeBluetooth": true,
//	  "includeSnmp": true,
//	  "includePortScan": true,
//	  "includeVulnScan": true,
//	  "freshWiredScan": true,
//	  "freshWifiScan": true,
//	  "freshBluetoothScan": true
//	}
//
// Authentication: Required
// Rate limiting: Yes (scans can be resource intensive).
func (s *Server) handleEngineScan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	engine := s.services.Discovery.Engine
	if engine == nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Discovery engine not available",
			"",
		)
		return
	}

	// Check if already scanning
	if engine.IsScanning() {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusConflict,
			ErrCodeConflict,
			"A scan is already in progress",
			"",
		)
		return
	}

	// Parse + validate options from body, default to quick scan.
	opts, ok := parseEngineScanOpts(w, r, logger)
	if !ok {
		return
	}

	// Run scan
	result, err := engine.Scan(r.Context(), opts)
	if err != nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Scan failed",
			err.Error(),
		)
		return
	}

	resp := EngineDiscoveryResponse{
		Devices:      engine.GetDevices(),
		Stats:        engine.GetStats(),
		ScanResult:   result,
		Capabilities: engine.GetCapabilities(),
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// scanType indicates the type of scan to perform.
type scanType int

const (
	scanTypeQuick scanType = iota
	scanTypeFull
)

// executeScan is a helper that handles common scan logic.
func (s *Server) executeScan(w http.ResponseWriter, r *http.Request, st scanType) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	engine := s.services.Discovery.Engine
	if engine == nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Discovery engine not available",
			"",
		)
		return
	}

	if engine.IsScanning() {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusConflict,
			ErrCodeConflict,
			"A scan is already in progress",
			"",
		)
		return
	}

	var result *discovery.ScanResult
	var err error
	var scanName string

	switch st {
	case scanTypeQuick:
		scanName = "Quick scan"
		result, err = engine.QuickScan(r.Context())
	case scanTypeFull:
		scanName = "Full scan"
		result, err = engine.FullScan(r.Context())
	}

	if err != nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			scanName+" failed",
			err.Error(),
		)
		return
	}

	resp := EngineDiscoveryResponse{
		Devices:      engine.GetDevices(),
		Stats:        engine.GetStats(),
		ScanResult:   result,
		Capabilities: engine.GetCapabilities(),
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleEngineQuickScan triggers a quick discovery scan.
//
// POST /api/v1/discovery/engine/quick triggers a quick scan.
// Uses cached data and performs correlation only.
//
// Authentication: Required
// Rate limiting: Yes.
func (s *Server) handleEngineQuickScan(w http.ResponseWriter, r *http.Request) {
	s.executeScan(w, r, scanTypeQuick)
}

// handleEngineFullScan triggers a comprehensive full scan.
//
// POST /api/v1/discovery/engine/full triggers a full discovery scan:
// - Fresh wired/WiFi/Bluetooth discovery
// - SNMP data collection
// - Port scanning
// - Vulnerability assessment
// - Device correlation
//
// This can take several minutes depending on network size.
//
// Authentication: Required
// Rate limiting: Yes (resource intensive).
func (s *Server) handleEngineFullScan(w http.ResponseWriter, r *http.Request) {
	s.executeScan(w, r, scanTypeFull)
}

// handleEngineStats returns discovery engine statistics.
//
// GET /api/v1/discovery/engine/stats returns engine metrics.
//
// Authentication: Required
// Rate limiting: None (read-only).
func (s *Server) handleEngineStats(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	engine := s.services.Discovery.Engine
	if engine == nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Discovery engine not available",
			"",
		)
		return
	}

	stats := engine.GetStats()
	sendJSONResponse(w, logger, http.StatusOK, stats)
}

// handleEngineCapabilities returns discovery engine capabilities.
//
// GET /api/v1/discovery/engine/capabilities returns what the engine can do.
//
// Authentication: Required
// Rate limiting: None (read-only).
func (s *Server) handleEngineCapabilities(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	engine := s.services.Discovery.Engine
	if engine == nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Discovery engine not available",
			"",
		)
		return
	}

	caps := engine.GetCapabilities()
	sendJSONResponse(w, logger, http.StatusOK, caps)
}

// handleEngineDevice returns a specific device by MAC address.
//
// GET /api/v1/discovery/engine/device/{mac} returns device details.
//
// Path parameter:
//   - mac: Device MAC address (any format: AA:BB:CC:DD:EE:FF or aa-bb-cc-dd-ee-ff)
//
// Authentication: Required
// Rate limiting: None (read-only).
func (s *Server) handleEngineDevice(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	engine := s.services.Discovery.Engine
	if engine == nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Discovery engine not available",
			"",
		)
		return
	}

	// Extract MAC from URL path
	// URL format: /api/v1/discovery/engine/device/{mac}
	path := r.URL.Path
	prefix := APIVersionPrefix + "/discovery/engine/device/"
	if !strings.HasPrefix(path, prefix) {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"Invalid request path",
			"",
		)
		return
	}
	mac := strings.TrimPrefix(path, prefix)
	if mac == "" {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			"MAC address required",
			"",
		)
		return
	}

	device := engine.GetDevice(mac)
	if device == nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusNotFound,
			ErrCodeNotFound,
			"Device not found",
			"",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, device)
}

// handleEngineEvents subscribes to engine events via SSE.
//
// GET /api/v1/discovery/engine/events opens an SSE stream for real-time updates.
//
// Event types:
// - device.discovered: New device found
// - device.updated: Device information changed
// - device.lost: Device went offline
// - scan.started: Scan began
// - scan.completed: Scan finished
//
// Authentication: Required
// Rate limiting: None (streaming endpoint).
func (s *Server) handleEngineEvents(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	engine := s.services.Discovery.Engine
	if engine == nil {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Discovery engine not available",
			"",
		)
		return
	}

	// Check if SSE is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		sendErrorResponseWithDetails(
			w, logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Streaming not supported",
			"",
		)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Subscribe to all events
	sub := engine.SubscribeAll(func(event *discovery.Event) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		// Write SSE format
		_, _ = w.Write([]byte("event: " + string(event.Type) + "\n"))
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()
	})
	defer engine.Unsubscribe(sub.ID())

	// Keep connection open until client disconnects
	<-r.Context().Done()
}
