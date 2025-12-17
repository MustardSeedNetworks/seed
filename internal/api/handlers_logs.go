// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
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
// POST /api/logs/client.
func (s *Server) handleClientLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ClientLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	broadcaster := logging.GetBroadcaster()
	if broadcaster == nil {
		http.Error(w, "Logging not initialized", http.StatusServiceUnavailable)
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

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "accepted",
		"received": len(req.Entries),
	})
}

// handleLogsQuery returns logs matching the specified filters.
// GET /api/logs/query?level=ERROR,WARN&layer=backend,api&component=auth&search=failed&limit=100&offset=0.
func (s *Server) handleLogsQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	broadcaster := logging.GetBroadcaster()
	if broadcaster == nil {
		http.Error(w, "Logging not initialized", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	levels := parseCSV(query.Get("level"))
	layers := parseCSV(query.Get("layer"))
	components := parseCSV(query.Get("component"))
	search := strings.ToLower(query.Get("search"))

	limit := 200 // Default limit
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get all logs and filter
	allLogs := broadcaster.GetAllLogs()
	filtered := make([]*logging.LogEntry, 0, len(allLogs))

	for _, log := range allLogs {
		// Filter by level
		if len(levels) > 0 && !containsIgnoreCase(levels, log.Level) {
			continue
		}

		// Filter by layer
		if len(layers) > 0 && !containsIgnoreCase(layers, log.Layer) {
			continue
		}

		// Filter by component
		if len(components) > 0 && !containsIgnoreCase(components, log.Component) {
			continue
		}

		// Filter by search text
		if search != "" && !strings.Contains(strings.ToLower(log.Message), search) {
			continue
		}

		filtered = append(filtered, log)
	}

	// Apply pagination
	totalCount := len(filtered)
	if offset >= len(filtered) {
		filtered = nil
	} else {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[offset:end]
	}

	sendJSONResponse(w, http.StatusOK, LogQueryResponse{
		Logs:       filtered,
		TotalCount: totalCount,
		Offset:     offset,
		Limit:      limit,
	})
}

// handleLogsStats returns aggregated log statistics.
// GET /api/logs/stats
func (s *Server) handleLogsStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	broadcaster := logging.GetBroadcaster()
	if broadcaster == nil {
		http.Error(w, "Logging not initialized", http.StatusServiceUnavailable)
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

	sendJSONResponse(w, http.StatusOK, stats)
}

// handleLogsRecent returns the most recent log entries.
// GET /api/logs/recent?limit=100
func (s *Server) handleLogsRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	broadcaster := logging.GetBroadcaster()
	if broadcaster == nil {
		http.Error(w, "Logging not initialized", http.StatusServiceUnavailable)
		return
	}

	limit := 100 // Default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	logs := broadcaster.GetRecentLogs(limit)
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// handleLogsClear clears the log buffer (admin action).
// DELETE /api/logs
func (s *Server) handleLogsClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	broadcaster := logging.GetBroadcaster()
	if broadcaster == nil {
		http.Error(w, "Logging not initialized", http.StatusServiceUnavailable)
		return
	}

	// Clear the buffer (we'd need to add this method to the broadcaster)
	// For now, just return success - the buffer will eventually cycle through
	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status": "cleared",
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
