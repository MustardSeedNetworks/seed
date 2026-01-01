// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Log API Handlers (comprehensive logging enhancement)
// ============================================================================

// ClientLogRequest represents a batch of log entries from the frontend.
type ClientLogRequest struct {
	Entries []ClientLogEntry `json:"entries"`
}

// ClientLogEntry represents a single log entry from the frontend.
type ClientLogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Component string         `json:"component"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Stack     string         `json:"stack,omitempty"`
}

// LogQueryResponse represents the response for log queries.
type LogQueryResponse struct {
	Logs       []*logging.LogEntry `json:"logs"`
	TotalCount int                 `json:"total_count"`
	Offset     int                 `json:"offset"`
	Limit      int                 `json:"limit"`
}

// LogStatsResponse represents log statistics.
type LogStatsResponse struct {
	TotalCount       int            `json:"total_count"`
	ByLevel          map[string]int `json:"by_level"`
	ByLayer          map[string]int `json:"by_layer"`
	ByComponent      map[string]int `json:"by_component"`
	ErrorsLastHour   int            `json:"errors_last_hour"`
	WarningsLastHour int            `json:"warnings_last_hour"`
}

// handleClientLogs receives log entries from the frontend and stores them.
// POST /api/logs/client (fixes #703).
func (s *Server) handleClientLogs(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	// Limit request body size to prevent DoS attacks (fixes #682)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req ClientLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	broadcaster := logging.GetBroadcaster()
	if broadcaster == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.logs.notInitialized"),
			"",
		) // fixes #694
		return
	}

	// Convert frontend entries to LogEntry and broadcast
	for _, entry := range req.Entries {
		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			timestamp = time.Now()
		}

		logEntry := &logging.LogEntry{
			Timestamp: timestamp,
			Level:     strings.ToUpper(entry.Level),
			Layer:     logging.LayerFrontend,
			Component: entry.Component,
			Message:   entry.Message,
			RequestID: entry.RequestID,
			SessionID: entry.SessionID,
			Metadata:  entry.Metadata,
			Stack:     entry.Stack,
		}

		broadcaster.Write(logEntry)
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"status":   "accepted",
		"received": len(req.Entries),
	})
}

// logQueryParams holds parsed log query parameters.
type logQueryParams struct {
	levels     []string
	layers     []string
	components []string
	search     string
	limit      int
	offset     int
}

// parseLogQueryParams extracts and validates query parameters from the request.
func parseLogQueryParams(r *http.Request) logQueryParams {
	query := r.URL.Query()
	params := logQueryParams{
		levels:     parseCSV(query.Get("level")),
		layers:     parseCSV(query.Get("layer")),
		components: parseCSV(query.Get("component")),
		search:     strings.ToLower(query.Get("search")),
		limit:      200, // Default limit
		offset:     0,
	}

	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			params.limit = parsed
		}
	}

	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			params.offset = parsed
		}
	}

	return params
}

// matchesLogFilters checks if a log entry matches the given filter criteria.
func matchesLogFilters(log *logging.LogEntry, params *logQueryParams) bool {
	if len(params.levels) > 0 && !containsIgnoreCase(params.levels, log.Level) {
		return false
	}
	if len(params.layers) > 0 && !containsIgnoreCase(params.layers, log.Layer) {
		return false
	}
	if len(params.components) > 0 && !containsIgnoreCase(params.components, log.Component) {
		return false
	}
	if params.search != "" && !strings.Contains(strings.ToLower(log.Message), params.search) {
		return false
	}
	return true
}

// paginateLogs applies pagination to a slice of logs.
func paginateLogs(logs []*logging.LogEntry, offset, limit int) []*logging.LogEntry {
	if offset >= len(logs) {
		return nil
	}
	end := min(offset+limit, len(logs))
	return logs[offset:end]
}

// handleLogsQuery returns logs matching the specified filters.
// GET /api/logs/query?level=ERROR,WARN&layer=backend,api&component=auth&search=failed&limit=100&offset=0 (fixes #703).
// Reads from database for persistence, falls back to memory buffer.
func (s *Server) handleLogsQuery(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	params := parseLogQueryParams(r)

	// Try database first if available (for persisted logs)
	if s.db != nil {
		dbLogs, err := s.queryLogsFromDB(r.Context(), params)
		if err == nil {
			sendJSONResponse(w, logger, http.StatusOK, LogQueryResponse{
				Logs:       dbLogs,
				TotalCount: len(dbLogs), // TODO: get actual count from DB
				Offset:     params.offset,
				Limit:      params.limit,
			})
			return
		}
		// Fall through to memory buffer on error
		logger.Warn("Failed to query logs from database, falling back to memory", "error", err)
	}

	// Fall back to memory buffer
	broadcaster := logging.GetBroadcaster()
	if broadcaster == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.logs.notInitialized"),
			"",
		) // fixes #694
		return
	}

	allLogs := broadcaster.GetAllLogs()
	filtered := make([]*logging.LogEntry, 0, len(allLogs))

	for _, log := range allLogs {
		if matchesLogFilters(log, &params) {
			filtered = append(filtered, log)
		}
	}

	totalCount := len(filtered)
	filtered = paginateLogs(filtered, params.offset, params.limit)

	sendJSONResponse(w, logger, http.StatusOK, LogQueryResponse{
		Logs:       filtered,
		TotalCount: totalCount,
		Offset:     params.offset,
		Limit:      params.limit,
	})
}

// queryLogsFromDB queries logs from the database with filters.
func (s *Server) queryLogsFromDB(ctx context.Context, params logQueryParams) ([]*logging.LogEntry, error) {
	opts := database.LogListOptions{
		Search: params.search,
		Limit:  params.limit,
		Offset: params.offset,
	}

	// Use first level/layer/component if specified
	if len(params.levels) > 0 {
		opts.Level = params.levels[0]
	}
	if len(params.layers) > 0 {
		opts.Layer = params.layers[0]
	}
	if len(params.components) > 0 {
		opts.Component = params.components[0]
	}

	dbEntries, err := s.db.Logs().List(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Convert database entries to logging.LogEntry
	result := make([]*logging.LogEntry, len(dbEntries))
	for i, e := range dbEntries {
		result[i] = &logging.LogEntry{
			Timestamp:  e.Timestamp,
			Level:      e.Level,
			Layer:      e.Layer,
			Message:    e.Message,
			Component:  e.Component,
			RequestID:  e.RequestID,
			SessionID:  e.SessionID,
			DurationMs: e.DurationMs,
			Stack:      e.Stack,
		}
	}

	return result, nil
}

// handleLogsStats returns aggregated log statistics.
// GET /api/logs/stats (fixes #703).
func (s *Server) handleLogsStats(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	broadcaster := logging.GetBroadcaster()
	if broadcaster == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.logs.notInitialized"),
			"",
		) // fixes #694
		return
	}

	allLogs := broadcaster.GetAllLogs()
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	stats := LogStatsResponse{
		TotalCount:  len(allLogs),
		ByLevel:     make(map[string]int),
		ByLayer:     make(map[string]int),
		ByComponent: make(map[string]int),
	}

	for _, log := range allLogs {
		// Count by level
		stats.ByLevel[log.Level]++

		// Count by layer
		stats.ByLayer[log.Layer]++

		// Count by component
		if log.Component != "" {
			stats.ByComponent[log.Component]++
		}

		// Count recent errors and warnings
		if log.Timestamp.After(oneHourAgo) {
			switch log.Level {
			case "ERROR":
				stats.ErrorsLastHour++
			case "WARN":
				stats.WarningsLastHour++
			}
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, stats)
}

// handleLogsRecent returns the most recent log entries.
// GET /api/logs/recent?limit=100 (fixes #703).
func (s *Server) handleLogsRecent(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		) // fixes #694
		return
	}

	broadcaster := logging.GetBroadcaster()
	if broadcaster == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			localizer.T("errors.logs.notInitialized"),
			"",
		) // fixes #694
		return
	}

	limit := 100 // Default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	logs := broadcaster.GetRecentLogs(limit)
	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"logs":  logs,
		"count": len(logs),
	})
}

// LogSubscription represents a client's log subscription preferences.
type LogSubscription struct {
	Levels     []string `json:"levels"`     // Filter by levels
	Layers     []string `json:"layers"`     // Filter by layers
	Components []string `json:"components"` // Filter by components
}

// Helper functions

// parseCSV splits a comma-separated string into a slice.
func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// containsIgnoreCase checks if a slice contains a string (case-insensitive).
func containsIgnoreCase(slice []string, target string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}
