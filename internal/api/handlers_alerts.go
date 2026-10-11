package api

// /api/v1/alerts endpoints — Stage A5.2. The Stage A4.5 / A4.6
// pipelines write alerts; these handlers let the operator UI list
// them, acknowledge ("seen") them, and resolve ("fixed") them.
//
//   GET    /api/v1/alerts                            list with filters
//   POST   /api/v1/alerts/{alertId}/acknowledge      mark acknowledged
//   POST   /api/v1/alerts/{alertId}/resolve          mark resolved
//
// List is read-only and runs through the normal auth surface. The
// two mutating endpoints go through writeGated so only operator+
// roles can mark alerts handled.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/invopop/jsonschema"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/inbox"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const (
	alertsPath = APIVersionPrefix + "/alerts"

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
			Path:    APIVersionPrefix + "/alerts/{alertId}/acknowledge",
			Handler: s.handleAlertAcknowledge,
			Methods: post,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/alerts/{alertId}/resolve",
			Handler: s.handleAlertResolve,
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
	enc := encodeAlerts(alerts, narratives, i18n.FromRequest(r))
	sendJSONResponse(w, logger, http.StatusOK, AlertListResponse{Count: len(enc), Alerts: enc})
}

// handleAlertAcknowledge serves POST /api/v1/alerts/{alertId}/acknowledge,
// recording the caller as the one who saw it.
func (s *Server) handleAlertAcknowledge(w http.ResponseWriter, r *http.Request) {
	id, ok := alertIDFromPath(w, r)
	if !ok {
		return
	}
	logger := logging.FromContext(r.Context())
	username := s.usernameFromRequest(r)
	if err := s.alertInbox.Acknowledge(r.Context(), id, username); err != nil {
		logger.ErrorContext(r.Context(), "alert acknowledge failed", "id", id, "error", err)
		writeAlertError(w, r, err, "Failed to acknowledge alert")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, AlertAcknowledgeResponse{
		ID: id, Acknowledged: true, AcknowledgedBy: username,
	})
}

// handleAlertResolve serves POST /api/v1/alerts/{alertId}/resolve.
func (s *Server) handleAlertResolve(w http.ResponseWriter, r *http.Request) {
	id, ok := alertIDFromPath(w, r)
	if !ok {
		return
	}
	logger := logging.FromContext(r.Context())
	if err := s.alertInbox.Resolve(r.Context(), id); err != nil {
		logger.ErrorContext(r.Context(), "alert resolve failed", "id", id, "error", err)
		writeAlertError(w, r, err, "Failed to resolve alert")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, AlertResolveResponse{ID: id, Resolved: true})
}

// alertIDFromPath reads the positive alert id from the {alertId} segment,
// answering 400 when it is not one.
func alertIDFromPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("alertId"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Alert id must be a positive integer")
		return 0, false
	}
	return id, true
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

// AlertListResponse is the GET /alerts envelope.
type AlertListResponse struct {
	Count  int     `json:"count"`
	Alerts []Alert `json:"alerts"`
}

// Alert is one inbox row. Each optional key is absent until it applies.
type Alert struct {
	ID             int64           `json:"id"`
	Type           string          `json:"type"`
	Severity       string          `json:"severity"`
	Title          string          `json:"title"`
	Message        string          `json:"message"`
	Source         string          `json:"source"`
	Acknowledged   bool            `json:"acknowledged"`
	Resolved       bool            `json:"resolved"`
	CreatedAt      string          `json:"createdAt"`
	Metadata       json.RawMessage `json:"metadata"                 jsonschema:"type=object"`
	DeviceID       *string         `json:"deviceId,omitempty"`
	AcknowledgedBy *string         `json:"acknowledgedBy,omitempty"`
	AcknowledgedAt string          `json:"acknowledgedAt,omitempty"`
	ResolvedAt     string          `json:"resolvedAt,omitempty"`
	// Deliveries is each receiver channel's outcome. It is what makes a
	// receiver that stopped accepting visible in the inbox (#2606, #2997); an
	// alert never offered to a receiver carries no key at all, and that must
	// never read as a failure.
	Deliveries []AlertDelivery `json:"deliveries,omitempty"`
	// Rule is the pipeline rule that raised the alert, e.g. "bgp.flap". An
	// escalation ladder is keyed by it (P-B2).
	Rule string `json:"rule,omitempty"`
	// RootCauseID is an earlier alert that probably caused this one (#409),
	// whose narrative covers both.
	RootCauseID *int64 `json:"rootCauseId,omitempty"`
	// Narrative explains a cause and the alerts it caused, in the reader's
	// language.
	Narrative *AlertNarrative `json:"narrative,omitempty"`
	// EscalationStage is the stage last sent for this alert, absent until one
	// is; EscalatedAt is when.
	EscalationStage int    `json:"escalationStage,omitempty"`
	EscalatedAt     string `json:"escalatedAt,omitempty"`
}

// JSONSchemaExtend opens metadata to any keys; the reflector otherwise closes
// every object.
func (Alert) JSONSchemaExtend(s *jsonschema.Schema) {
	s.Properties.Value("metadata").AdditionalProperties = jsonschema.TrueSchema
}

// AlertNarrative is the rendered narrative the inbox serves; a named type so
// the schema component does not take narrative's generic name.
type AlertNarrative narrative.Text

// AlertDelivery is one channel's delivery outcome for one alert.
type AlertDelivery struct {
	Channel alerts.Channel `json:"channel" jsonschema:"enum=webhook,enum=email,enum=syslog"`
	Status  string         `json:"status"  jsonschema:"enum=pending,enum=delivered,enum=failed,enum=dropped"`
	// AttemptedAt is when the last attempt finished; absent while queued.
	AttemptedAt string `json:"attemptedAt,omitempty"`
	// Error is the last attempt's error text.
	Error string `json:"error,omitempty"`
}

// AlertAcknowledgeResponse is POST /alerts/{alertId}/acknowledge.
type AlertAcknowledgeResponse struct {
	ID             int64  `json:"id"`
	Acknowledged   bool   `json:"acknowledged"`
	AcknowledgedBy string `json:"acknowledgedBy"`
}

// AlertResolveResponse is POST /alerts/{alertId}/resolve.
type AlertResolveResponse struct {
	ID       int64 `json:"id"`
	Resolved bool  `json:"resolved"`
}

func encodeAlerts(
	rows []*alerts.Alert, narratives map[int64]narrative.Narrative, t *i18n.Localizer,
) []Alert {
	out := make([]Alert, 0, len(rows))
	for _, a := range rows {
		out = append(out, encodeAlert(a, narratives, t))
	}
	return out
}

func encodeAlert(a *alerts.Alert, narratives map[int64]narrative.Narrative, t *i18n.Localizer) Alert {
	row := Alert{
		ID:              a.ID,
		Type:            a.Type,
		Severity:        a.Severity,
		Title:           a.Title,
		Message:         a.Message,
		Source:          a.Source,
		Acknowledged:    a.Acknowledged,
		Resolved:        a.Resolved,
		CreatedAt:       formatTime(a.CreatedAt),
		Metadata:        rawJSON(a.Metadata),
		DeviceID:        a.DeviceID,
		AcknowledgedBy:  a.AcknowledgedBy,
		AcknowledgedAt:  formatTimePtr(a.AcknowledgedAt),
		ResolvedAt:      formatTimePtr(a.ResolvedAt),
		Deliveries:      encodeDeliveries(a.Deliveries),
		Rule:            a.Rule,
		RootCauseID:     a.RootCauseID,
		EscalationStage: a.EscalationStage,
		EscalatedAt:     formatTimePtr(a.EscalatedAt),
	}
	if n, ok := narratives[a.ID]; ok {
		text := AlertNarrative(n.Render(t.TWithData))
		row.Narrative = &text
	}
	return row
}

func encodeDeliveries(deliveries []alerts.Delivery) []AlertDelivery {
	if len(deliveries) == 0 {
		return nil
	}
	out := make([]AlertDelivery, 0, len(deliveries))
	for _, d := range deliveries {
		out = append(out, AlertDelivery{
			Channel:     d.Channel,
			Status:      d.Status,
			AttemptedAt: formatTimePtr(d.AttemptedAt),
			Error:       d.Error,
		})
	}
	return out
}
