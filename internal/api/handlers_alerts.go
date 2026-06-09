package api

// /api/v1/alerts endpoints — Stage A5.2. The Stage A4.5 / A4.6
// pipelines write alerts; these handlers let the operator UI list
// them, acknowledge ("seen") them, and resolve ("fixed") them.
//
//   GET    /api/v1/alerts                       list with filters
//   POST   /api/v1/alerts/{id}/acknowledge      mark acknowledged
//   POST   /api/v1/alerts/{id}/resolve          mark resolved
//
// List is read-only and runs through the normal auth surface. The
// two mutating endpoints go through writeGated so only operator+
// roles can mark alerts handled.

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const (
	alertsPath       = APIVersionPrefix + "/alerts"
	alertsPathPrefix = alertsPath + "/"

	alertsMaxLimit     = 1000
	alertsDefaultLimit = 100
)

// handleAlerts serves GET /api/v1/alerts. Supports the same filter
// vocabulary as AlertRepository.List:
//
//	type=connectivity            string (any AlertType*)
//	severity=warning             string (any AlertSeverity*)
//	device_id=node-abc           filter to one device
//	unacknowledged_only=true     hide acknowledged
//	unresolved_only=true         hide resolved
//	since=2026-01-01T00:00:00Z   RFC3339
//	limit=100   offset=0         pagination
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	opts, parseErr := parseAlertListOptions(r)
	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusBadRequest)
		return
	}
	alerts, err := db.Alerts().List(r.Context(), opts)
	if err != nil {
		logger.ErrorContext(r.Context(), "list alerts failed", "error", err)
		http.Error(w, "Failed to list alerts", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, map[string]any{
		jsonKeyCount: len(alerts),
		"alerts":     encodeAlerts(alerts),
	})
}

// handleAlertAction routes /api/v1/alerts/{id}/{action} to either
// Acknowledge or Resolve. Splitting the path here keeps the route
// registration to two entries (alertsPath + alertsPathPrefix)
// instead of one per (id, action) combinator.
func (s *Server) handleAlertAction(w http.ResponseWriter, r *http.Request) {
	id, action, ok := splitAlertActionPath(r.URL.Path)
	if !ok {
		http.Error(w, "Path must be /alerts/{id}/{acknowledge|resolve}", http.StatusBadRequest)
		return
	}
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	switch action {
	case "acknowledge":
		username := s.usernameFromRequest(r)
		if err := db.Alerts().Acknowledge(r.Context(), id, username); err != nil {
			logger.ErrorContext(r.Context(), "alert acknowledge failed",
				"id", id, "error", err)
			http.Error(w, "Failed to acknowledge alert", http.StatusInternalServerError)
			return
		}
		writeJSON(w, r, map[string]any{
			"id":             id,
			"acknowledged":   true,
			"acknowledgedBy": username,
		})
	case "resolve":
		if err := db.Alerts().Resolve(r.Context(), id); err != nil {
			logger.ErrorContext(r.Context(), "alert resolve failed",
				"id", id, "error", err)
			http.Error(w, "Failed to resolve alert", http.StatusInternalServerError)
			return
		}
		writeJSON(w, r, map[string]any{
			"id":       id,
			"resolved": true,
		})
	default:
		http.Error(w, "Unknown action; use acknowledge or resolve",
			http.StatusBadRequest)
	}
}

// splitAlertActionPath parses /api/v1/alerts/{id}/{action} into
// (id, action). Returns ok=false for malformed paths.
func splitAlertActionPath(urlPath string) (int64, string, bool) {
	rest := strings.TrimPrefix(urlPath, alertsPathPrefix)
	if rest == urlPath {
		return 0, "", false
	}
	// path is "{id}/{action}" — two slash-separated tokens.
	const actionPathParts = 2
	parts := strings.SplitN(rest, "/", actionPathParts)
	if len(parts) != actionPathParts || parts[0] == "" || parts[1] == "" {
		return 0, "", false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	return id, parts[1], true
}

// usernameFromRequest pulls the authenticated user from the X-Username
// header that the apiTokenMiddleware + JWT middleware set. Falls
// back to "system" when the surrounding middleware was bypassed
// (e.g. tests).
func (s *Server) usernameFromRequest(r *http.Request) string {
	if u := r.Header.Get("X-Username"); u != "" {
		return u
	}
	return "system"
}

// parseAlertListOptions reads query-string filters. Returns 400-
// shaped errors via plain text.
func parseAlertListOptions(r *http.Request) (database.AlertListOptions, error) {
	q := r.URL.Query()
	opts := database.AlertListOptions{
		Type:     q.Get("type"),
		Severity: q.Get("severity"),
		DeviceID: q.Get("device_id"),
		Limit:    alertsDefaultLimit,
	}
	if v := q.Get("unacknowledged_only"); v == "true" || v == "1" {
		opts.UnacknowledgedOnly = true
	}
	if v := q.Get("unresolved_only"); v == "true" || v == "1" {
		opts.UnresolvedOnly = true
	}
	if since := q.Get("since"); since != "" {
		t, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return database.AlertListOptions{}, errors.New("invalid 'since' (expect RFC3339)")
		}
		opts.Since = t
	}
	if limitRaw := q.Get("limit"); limitRaw != "" {
		n, err := strconv.Atoi(limitRaw)
		if err != nil || n < 1 {
			return database.AlertListOptions{}, errors.New("invalid 'limit' (positive integer)")
		}
		if n > alertsMaxLimit {
			n = alertsMaxLimit
		}
		opts.Limit = n
	}
	if offsetRaw := q.Get("offset"); offsetRaw != "" {
		n, err := strconv.Atoi(offsetRaw)
		if err != nil || n < 0 {
			return database.AlertListOptions{}, errors.New("invalid 'offset' (non-negative integer)")
		}
		opts.Offset = n
	}
	return opts, nil
}

// encodeAlerts shapes the AlertRepository rows into the JSON
// envelope. Done explicitly so the wire format stays stable when
// DB columns evolve.
func encodeAlerts(alerts []*database.Alert) []map[string]any {
	out := make([]map[string]any, 0, len(alerts))
	for _, a := range alerts {
		row := map[string]any{
			"id":           a.ID,
			"type":         a.Type,
			"severity":     a.Severity,
			"title":        a.Title,
			"message":      a.Message,
			"source":       a.Source,
			"acknowledged": a.Acknowledged,
			"resolved":     a.Resolved,
			"createdAt":    formatTime(a.CreatedAt),
			"metadata":     rawJSON(a.Metadata),
		}
		if a.DeviceID != nil {
			row["deviceId"] = *a.DeviceID
		}
		if a.AcknowledgedBy != nil {
			row["acknowledgedBy"] = *a.AcknowledgedBy
		}
		if a.AcknowledgedAt != nil {
			row["acknowledgedAt"] = formatTime(*a.AcknowledgedAt)
		}
		if a.ResolvedAt != nil {
			row["resolvedAt"] = formatTime(*a.ResolvedAt)
		}
		out = append(out, row)
	}
	return out
}
