package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/capabilities"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/system"
	"github.com/MustardSeedNetworks/seed/internal/version"
)

// Log retrieval constants.
const (
	// maxLogLinesLimit is the maximum number of log lines that can be requested.
	maxLogLinesLimit = 1000
)

// StatusResponse represents the system status.
type StatusResponse struct {
	Status     string `json:"status"`
	Version    string `json:"version"`
	Uptime     int64  `json:"uptime"`
	Interface  string `json:"interface"`
	IsWireless bool   `json:"isWireless"`
	// ICMPAvailable is a privilege check, not a platform one: whether this
	// process can open a raw socket. Kept distinct from Capabilities, which
	// answers whether the operating system supports a feature at all.
	ICMPAvailable bool `json:"icmpAvailable"`
	// Capabilities is what this platform can do, from internal/capabilities
	// (#749). The same source renders HARDWARE.md's matrix, so the UI and the
	// document cannot disagree.
	Capabilities []capabilities.Entry `json:"capabilities"`
}

// handleStatus returns the system status (fixes #544 - split from handlers.go).
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	// Check if current interface is wireless
	isWireless := false
	if s.wifiManager() != nil {
		isWireless = s.wifiManager().IsWireless()
	}

	resp := StatusResponse{
		Status:        "ok",
		Version:       version.GetVersion(),
		Interface:     s.defaultInterface(),
		IsWireless:    isWireless,
		ICMPAvailable: s.icmpAvailable,
		Capabilities:  capabilities.Report(),
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleExport exports current diagnostic data as JSON (fixes #544 - split from
// handlers.go). Accepts optional query parameter: ?interface=eth0. Thin transport
// (ADR-0020): the export-assembly fan-out lives in the export use-case.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	data, err := s.exportService.Build(r.Context(), s.getInterfaceFromRequest(r), version.GetVersion())
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to refresh interfaces", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			"Failed to refresh interfaces", "",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=seed-export.json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if encErr := encoder.Encode(data); encErr != nil {
		logger.ErrorContext(r.Context(), "Error encoding export response", "error", encErr)
	}
}

// LogFileResponse is the answer of GET /api/v1/reporting/logs: the tail of the
// daemon's log file.
type LogFileResponse struct {
	Path  string   `json:"path"`
	Lines []string `json:"lines"`
}

// handleLogs returns the tail of the application log file for troubleshooting (fixes #544 - split from handlers.go).
// Requires JWT authentication (enforced by middleware).
// Security fix #301: Removed insecure LOG_ACCESS_TOKEN - JWT authentication is sufficient.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	// JWT authentication is enforced by the global auth middleware, which
	// stamps the validated identity on the request context.

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
			if n > maxLogLinesLimit {
				n = maxLogLinesLimit
			}
			maxLines = n
		}
	}

	const maxBytes int64 = 500 * 1024 // limit read size to 500KB
	lines, err := readLastLines(s.logPath, maxBytes, maxLines)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to read log file", "error", err)
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

	sendJSONResponse(w, logger, http.StatusOK, LogFileResponse{Path: s.logPath, Lines: lines})
}

// HealthResponse is the liveness answer of GET /api/v1/health.
type HealthResponse struct {
	Status string  `json:"status"`
	Uptime float64 `json:"uptime"` // seconds since the daemon started
}

// handleHealth handles GET /api/health - simple liveness check for load balancers (fixes #540, #544).
// Returns 200 OK if server is running, minimal response for fast health checks.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	// Simple health check - just return OK
	// For detailed health, use /api/system/health
	sendJSONResponse(w, logger, http.StatusOK, HealthResponse{
		Status: "ok",
		Uptime: time.Since(s.startTime).Seconds(),
	})
}

// SystemHealthResponse is GET /telemetry/system/health: the host's resource
// usage, the daemon's version and uptime, and which background services run.
type SystemHealthResponse struct {
	System      system.Health                   `json:"system"`
	Application SystemHealthApplicationResponse `json:"application"`
	Services    SystemHealthServicesResponse    `json:"services"`
}

// SystemHealthApplicationResponse is the daemon half of the health answer.
type SystemHealthApplicationResponse struct {
	Version string  `json:"version"`
	Uptime  float64 `json:"uptime"` // seconds since the daemon started
}

// SystemHealthServicesResponse says which background services are running.
type SystemHealthServicesResponse struct {
	DiscoveryService bool `json:"discoveryService"`
	LinkMonitor      bool `json:"linkMonitor"`
	SSEHub           bool `json:"sseHub"`
}

// handleSystemHealth handles GET /api/v1/telemetry/system/health (#540, #544).
func (s *Server) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	health, err := s.health.Health()
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to get system health", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			"Failed to get system health. Check server logs for details.", "",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, SystemHealthResponse{
		System: *health,
		Application: SystemHealthApplicationResponse{
			Version: version.GetVersion(),
			Uptime:  time.Since(s.startTime).Seconds(),
		},
		Services: SystemHealthServicesResponse{
			DiscoveryService: s.discoveryService() != nil && s.discoveryService().IsRunning(),
			LinkMonitor:      s.linkMonitor() != nil,
			SSEHub:           s.sseHub() != nil,
		},
	})
}
