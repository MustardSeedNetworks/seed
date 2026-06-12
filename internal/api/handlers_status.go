package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	Status        string `json:"status"`
	Version       string `json:"version"`
	Uptime        int64  `json:"uptime"`
	Interface     string `json:"interface"`
	IsWireless    bool   `json:"isWireless"`
	ICMPAvailable bool   `json:"icmpAvailable"`
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

// handleLogs returns the tail of the application log file for troubleshooting (fixes #544 - split from handlers.go).
// Requires JWT authentication (enforced by middleware).
// Security fix #301: Removed insecure LOG_ACCESS_TOKEN - JWT authentication is sufficient.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
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

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"path":  s.logPath,
		"lines": lines,
	})
}

// handleHealth handles GET /api/health - simple liveness check for load balancers (fixes #540, #544).
// Returns 200 OK if server is running, minimal response for fast health checks.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
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
	// Get system health metrics
	health, err := system.GetHealth()
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to get system health", "error", err)
		sendJSONResponse(w, logger, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get system health. Check server logs for details.",
		})
		return
	}

	// Add application-specific health information
	appHealth := map[string]any{
		"system": health,
		"application": map[string]any{
			"version":     version.GetVersion(),
			"uptime":      time.Since(s.startTime).Seconds(),
			"uptime_text": time.Since(s.startTime).String(),
		},
		"services": map[string]any{
			"discovery_service": s.discoveryService() != nil && s.discoveryService().IsRunning(),
			"link_monitor":      s.linkMonitor() != nil,
			"sse_hub":           s.sseHub() != nil,
			"vlan_monitor":      s.vlanTrafficMonitor() != nil,
		},
	}

	sendJSONResponse(w, logger, http.StatusOK, appHealth)
}
