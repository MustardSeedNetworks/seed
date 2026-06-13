package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	logquery "github.com/MustardSeedNetworks/seed/internal/logs/query"
)

// Log query constants.
const (
	// logQueryDefaultLimit is the default number of log entries to return in queries.
	logQueryDefaultLimit = 200
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
	RequestID string         `json:"requestId,omitempty"`
	SessionID string         `json:"sessionId,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Stack     string         `json:"stack,omitempty"`
}

// LogQueryResponse represents the response for log queries.
type LogQueryResponse struct {
	Logs       []LogEntry `json:"logs"`
	TotalCount int        `json:"totalCount"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
}

// LogEntry is the flat transport view of a log record, mirroring
// logging.LogEntry's wire shape so the published schema does not depend on the
// logging domain package.
type LogEntry struct {
	Timestamp  time.Time      `json:"timestamp"`
	Level      string         `json:"level"`
	Layer      string         `json:"layer"`
	RequestID  string         `json:"requestId,omitempty"`
	SessionID  string         `json:"sessionId,omitempty"`
	Message    string         `json:"message"`
	Component  string         `json:"component,omitempty"`
	DurationMs int64          `json:"durationMs,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Stack      string         `json:"stack,omitempty"`
}

// toLogEntries maps logging records onto their flat transport view.
func toLogEntries(entries []*logging.LogEntry) []LogEntry {
	out := make([]LogEntry, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		out = append(out, LogEntry{
			Timestamp:  e.Timestamp,
			Level:      e.Level,
			Layer:      e.Layer,
			RequestID:  e.RequestID,
			SessionID:  e.SessionID,
			Message:    e.Message,
			Component:  e.Component,
			DurationMs: e.DurationMs,
			Metadata:   e.Metadata,
			Stack:      e.Stack,
		})
	}
	return out
}

// LogStatsResponse represents log statistics.
type LogStatsResponse struct {
	TotalCount       int            `json:"totalCount"`
	ByLevel          map[string]int `json:"byLevel"`
	ByLayer          map[string]int `json:"byLayer"`
	ByComponent      map[string]int `json:"byComponent"`
	ErrorsLastHour   int            `json:"errorsLastHour"`
	WarningsLastHour int            `json:"warningsLastHour"`
}

// handleClientLogs receives log entries from the frontend and stores them.
// POST /api/logs/client (fixes #703).
func (s *Server) handleClientLogs(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req ClientLogRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
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
// parseLogQueryParams extracts and validates the log-query filter from the
// request, producing the transport-neutral input the log-query use-case consumes.
func parseLogQueryParams(r *http.Request) logquery.Params {
	query := r.URL.Query()
	params := logquery.Params{
		Levels:     parseCSV(query.Get("level")),
		Layers:     parseCSV(query.Get("layer")),
		Components: parseCSV(query.Get("component")),
		Search:     strings.ToLower(query.Get("search")),
		Limit:      logQueryDefaultLimit,
		Offset:     0,
	}

	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			params.Limit = parsed
		}
	}

	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			params.Offset = parsed
		}
	}

	return params
}

// handleLogsQuery returns logs matching the specified filters.
// GET /api/logs/query?level=ERROR,WARN&layer=backend,api&component=auth&search=failed&limit=100&offset=0 (fixes #703).
// Thin handler (ADR-0020): the log-query use-case owns the "persisted store first,
// memory buffer fallback" decision and filtering/pagination.
func (s *Server) handleLogsQuery(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	params := parseLogQueryParams(r)

	res, err := s.logQuery.Query(r.Context(), params)
	if err != nil {
		// The only error is ErrNoSource (neither store nor buffer available).
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

	sendJSONResponse(w, logger, http.StatusOK, LogQueryResponse{
		Logs:       toLogEntries(res.Entries),
		TotalCount: res.TotalCount,
		Offset:     params.Offset,
		Limit:      params.Limit,
	})
}

// handleLogsStats returns aggregated log statistics.
// GET /api/logs/stats (fixes #703).
func (s *Server) handleLogsStats(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

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
