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

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/inbox"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const (
	alertsPath       = APIVersionPrefix + "/alerts"
	alertsPathPrefix = alertsPath + "/"

	alertsMaxLimit     = 1000
	alertsDefaultLimit = 100
)

// alertRoutes returns the inbox routes and the receiver test-send. Listing
// is a safe read; acknowledging, resolving and test-sending are operator
// actions, and the test-send is rate-limited because each call is an outbound
// connection.
func (s *Server) alertRoutes() []route.Route {
	op := roles.Operator
	post := []string{http.MethodPost}
	return []route.Route{
		{Path: APIVersionPrefix + "/alerts", Handler: s.handleAlerts, Methods: []string{http.MethodGet}, Auth: true},
		{
			Path:    APIVersionPrefix + "/alerts/",
			Handler: s.handleAlertAction,
			Methods: post,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:         APIVersionPrefix + "/settings/alerts/test",
			Handler:      s.handleAlertTestSend,
			Methods:      post,
			Scope:        op,
			MaxBodyBytes: MaxBodySizeConfig,
			Limiter:      limitEndpoint,
			Auth:         true,
			CSRF:         true,
		},
	}
}

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

	opts, parseErr := parseAlertListOptions(r)
	if parseErr != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, parseErr.Error())
		return
	}
	alerts, err := s.alertInbox.List(r.Context(), opts)
	if err != nil {
		logger.ErrorContext(r.Context(), "list alerts failed", "error", err)
		writeAlertError(w, r, err, "Failed to list alerts")
		return
	}
	narratives, err := s.alertInbox.Narratives(r.Context(), alerts)
	if err != nil {
		logger.ErrorContext(r.Context(), "explain alerts failed", "error", err)
		writeAlertError(w, r, err, "Failed to list alerts")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		jsonKeyCount: len(alerts),
		"alerts":     encodeAlerts(alerts, narratives, i18n.FromRequest(r)),
	})
}

// handleAlertAction routes /api/v1/alerts/{id}/{action} to either
// Acknowledge or Resolve. Splitting the path here keeps the route
// registration to two entries (alertsPath + alertsPathPrefix)
// instead of one per (id, action) combinator.
func (s *Server) handleAlertAction(w http.ResponseWriter, r *http.Request) {
	id, action, ok := splitAlertActionPath(r.URL.Path)
	if !ok {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Path must be /alerts/{id}/{acknowledge|resolve}")
		return
	}
	logger := logging.FromContext(r.Context())

	switch action {
	case "acknowledge":
		username := s.usernameFromRequest(r)
		if err := s.alertInbox.Acknowledge(r.Context(), id, username); err != nil {
			logger.ErrorContext(r.Context(), "alert acknowledge failed", "id", id, "error", err)
			writeAlertError(w, r, err, "Failed to acknowledge alert")
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, map[string]any{
			"id":             id,
			"acknowledged":   true,
			"acknowledgedBy": username,
		})
	case "resolve":
		if err := s.alertInbox.Resolve(r.Context(), id); err != nil {
			logger.ErrorContext(r.Context(), "alert resolve failed", "id", id, "error", err)
			writeAlertError(w, r, err, "Failed to resolve alert")
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, map[string]any{
			"id":       id,
			"resolved": true,
		})
	default:
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Unknown action; use acknowledge or resolve")
	}
}

// writeAlertError maps an alert-inbox error to its HTTP status: the store unwired
// → 503 (the prior "Database not initialized"), anything else → 500 with genericMsg.
func writeAlertError(w http.ResponseWriter, r *http.Request, err error, genericMsg string) {
	if errors.Is(err, inbox.ErrUnavailable) {
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
		return
	}
	writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, genericMsg)
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

// usernameFromRequest pulls the authenticated user that the apiTokenMiddleware
// or the JWT middleware stamped on the request context (#2632; it was a
// spoofable header). Falls back to "system" when the surrounding middleware
// was bypassed (e.g. tests).
func (s *Server) usernameFromRequest(r *http.Request) string {
	if u := usernameFromContext(r); u != "" {
		return u
	}
	return "system"
}

// parseAlertListOptions reads query-string filters. Returns 400-
// shaped errors via plain text.
func parseAlertListOptions(r *http.Request) (alerts.ListOptions, error) {
	q := r.URL.Query()
	opts := alerts.ListOptions{
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
			return alerts.ListOptions{}, errors.New("invalid 'since' (expect RFC3339)")
		}
		opts.Since = t
	}
	if limitRaw := q.Get("limit"); limitRaw != "" {
		n, err := strconv.Atoi(limitRaw)
		if err != nil || n < 1 {
			return alerts.ListOptions{}, errors.New("invalid 'limit' (positive integer)")
		}
		if n > alertsMaxLimit {
			n = alertsMaxLimit
		}
		opts.Limit = n
	}
	if offsetRaw := q.Get("offset"); offsetRaw != "" {
		n, err := strconv.Atoi(offsetRaw)
		if err != nil || n < 0 {
			return alerts.ListOptions{}, errors.New("invalid 'offset' (non-negative integer)")
		}
		opts.Offset = n
	}
	return opts, nil
}

// encodeAlerts shapes the AlertRepository rows into the JSON
// envelope. Done explicitly so the wire format stays stable when
// DB columns evolve.
func encodeAlerts(
	alerts []*alerts.Alert, narratives map[int64]narrative.Narrative, t *i18n.Localizer,
) []map[string]any {
	out := make([]map[string]any, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, encodeAlert(a, narratives, t))
	}
	return out
}

func encodeAlert(
	a *alerts.Alert, narratives map[int64]narrative.Narrative, t *i18n.Localizer,
) map[string]any {
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
	if len(a.Deliveries) > 0 {
		row["deliveries"] = encodeDeliveries(a.Deliveries)
	}
	// The rule is what an escalation ladder is keyed by (P-B2), so the
	// operator reads it here to configure one.
	if a.Rule != "" {
		row["rule"] = a.Rule
	}
	// A caused alert points at its cause, whose narrative covers both.
	if a.RootCauseID != nil {
		row["rootCauseId"] = *a.RootCauseID
	}
	if n, ok := narratives[a.ID]; ok {
		row["narrative"] = n.Render(t.TWithData)
	}
	if a.EscalationStage > 0 {
		row["escalationStage"] = a.EscalationStage
	}
	if a.EscalatedAt != nil {
		row["escalatedAt"] = formatTime(*a.EscalatedAt)
	}
	return row
}

// encodeDeliveries is each channel's outcome. It is what makes a receiver
// that stopped accepting visible in the inbox (#2606, #2997); an alert with no
// entry was never offered to a receiver and carries no key at all.
func encodeDeliveries(deliveries []alerts.Delivery) []map[string]any {
	out := make([]map[string]any, 0, len(deliveries))
	for _, d := range deliveries {
		row := map[string]any{"channel": d.Channel, "status": d.Status}
		if d.AttemptedAt != nil {
			row["attemptedAt"] = formatTime(*d.AttemptedAt)
		}
		if d.Error != "" {
			row["error"] = d.Error
		}
		out = append(out, row)
	}
	return out
}
