package httpapi

import (
	"encoding/json"
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

	// Parse options from body, default to quick scan
	var opts *discovery.ScanOptions
	if r.ContentLength > 0 {
		var req EngineScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendErrorResponseWithDetails(
				w, logger,
				http.StatusBadRequest,
				ErrCodeBadRequest,
				"Invalid request body",
				err.Error(),
			)
			return
		}

		if req.ScanType == "full" {
			opts = discovery.DefaultFullScanOpts()
		} else {
			opts = discovery.DefaultQuickScanOpts()
		}

		// Override with request values
		opts.IncludeWired = req.IncludeWired
		opts.IncludeWiFi = req.IncludeWiFi
		opts.IncludeBluetooth = req.IncludeBluetooth
		opts.IncludeSNMP = req.IncludeSNMP
		opts.IncludePortScan = req.IncludePortScan
		opts.IncludeVulnScan = req.IncludeVulnScan
		opts.FreshWiredScan = req.FreshWiredScan
		opts.FreshWiFiScan = req.FreshWiFiScan
		opts.FreshBluetoothScan = req.FreshBluetoothScan
	} else {
		opts = discovery.DefaultQuickScanOpts()
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
