package api

// /api/v1/alert-rules — Stage A5.10. Operator-defined alert rules.
// The Stage A4.5/A4.6 pipelines retain their hardcoded fallback
// rule set; rows in alert_rules are applied additively.
//
//   GET    /api/v1/alert-rules            list (filter ?enabled_only)
//   POST   /api/v1/alert-rules            create
//   GET    /api/v1/alert-rules/{id}       fetch one
//   PUT    /api/v1/alert-rules/{id}       full update
//   DELETE /api/v1/alert-rules/{id}       delete

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
)

const (
	alertRulesPath       = APIVersionPrefix + "/alert-rules"
	alertRulesPathPrefix = alertRulesPath + "/"
)

// alertRuleInput is the wire shape for POST + PUT. Mirrors the
// repo struct minus the audit columns.
type alertRuleInput struct {
	Name                 string `json:"name"`
	Enabled              bool   `json:"enabled"`
	MatchKind            string `json:"matchKind,omitempty"`
	MatchSeverity        string `json:"matchSeverity,omitempty"`
	MatchPayloadContains string `json:"matchPayloadContains,omitempty"`
	AlertType            string `json:"alertType"`
	AlertSeverity        string `json:"alertSeverity"`
	AlertTitle           string `json:"alertTitle"`
	AlertMessage         string `json:"alertMessage"`
}

func (s *Server) handleAlertRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listAlertRules(w, r)
	case http.MethodPost:
		s.createAlertRule(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAlertRuleByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, alertRulesPathPrefix)
	if rest == "" || strings.Contains(rest, "/") {
		http.Error(w, "Missing or invalid rule id", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Rule id must be a positive integer", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.getAlertRule(w, r, id)
	case http.MethodPut:
		s.updateAlertRule(w, r, id)
	case http.MethodDelete:
		s.deleteAlertRule(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listAlertRules(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	enabledOnly := r.URL.Query().Get("enabled_only") == "true"
	rules, err := db.AlertRules().List(r.Context(), enabledOnly)
	if err != nil {
		logger.ErrorContext(r.Context(), "list alert_rules failed", "error", err)
		http.Error(w, "Failed to list rules", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, map[string]any{
		jsonKeyCount: len(rules),
		"rules":      encodeAlertRules(rules),
	})
}

func (s *Server) getAlertRule(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	rule, err := db.AlertRules().Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrAlertRuleNotFound) {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		logger.ErrorContext(r.Context(), "get alert_rule failed", "id", id, "error", err)
		http.Error(w, "Failed to load rule", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, encodeAlertRule(rule))
}

func (s *Server) createAlertRule(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	in, err := decodeAlertRuleInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rule := inputToRule(in, 0)
	if createErr := db.AlertRules().Create(r.Context(), rule); createErr != nil {
		logger.ErrorContext(r.Context(), "create alert_rule failed", "error", createErr)
		if strings.HasPrefix(createErr.Error(), "alert_rules:") {
			http.Error(w, createErr.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to create rule", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", alertRulesPathPrefix+strconv.FormatInt(rule.ID, 10))
	writeJSON(w, r, encodeAlertRule(rule))
}

func (s *Server) updateAlertRule(w http.ResponseWriter, r *http.Request, id int64) {
	in, err := s.parseRuleUpdate(w, r)
	if err != nil {
		return
	}
	rule := inputToRule(in, id)
	if !s.applyRuleUpdate(w, r, id, rule) {
		return
	}
	// Echo the freshly-read row so the JSON reflects updated_at.
	// Get failures fall back to echoing the request shape.
	current, getErr := s.db().AlertRules().Get(r.Context(), id)
	if getErr != nil || current == nil {
		writeJSON(w, r, encodeAlertRule(rule))
		return
	}
	writeJSON(w, r, encodeAlertRule(current))
}

// parseRuleUpdate consolidates the precondition checks (db ready +
// body decode) the PUT handler shares with no other handler. Splits
// out from updateAlertRule so the call site reads top-down and so
// the structural shape doesn't mirror handlers_polling_targets.
func (s *Server) parseRuleUpdate(w http.ResponseWriter, r *http.Request) (*alertRuleInput, error) {
	if s.db() == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return nil, errors.New("db not ready")
	}
	in, err := decodeAlertRuleInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, err
	}
	return in, nil
}

// applyRuleUpdate runs the repo Update + maps errors to HTTP
// statuses. Returns true on success so the caller proceeds to
// echo; false when an error response has already been written.
func (s *Server) applyRuleUpdate(
	w http.ResponseWriter, r *http.Request, id int64, rule *database.AlertRule,
) bool {
	logger := logging.FromContext(r.Context())
	if err := s.db().AlertRules().Update(r.Context(), rule); err != nil {
		if errors.Is(err, database.ErrAlertRuleNotFound) {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return false
		}
		logger.ErrorContext(r.Context(), "update alert_rule failed",
			"id", id, "error", err)
		http.Error(w, "Failed to update rule", http.StatusInternalServerError)
		return false
	}
	return true
}

func (s *Server) deleteAlertRule(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	db := s.db()
	if db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	if err := db.AlertRules().Delete(r.Context(), id); err != nil {
		if errors.Is(err, database.ErrAlertRuleNotFound) {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		logger.ErrorContext(r.Context(), "delete alert_rule failed", "id", id, "error", err)
		http.Error(w, "Failed to delete rule", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeAlertRuleInput(r *http.Request) (*alertRuleInput, error) {
	if r.Body == nil {
		return nil, errors.New("body required")
	}
	var in alertRuleInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return nil, errors.New("invalid JSON body: " + err.Error())
	}
	if in.Name == "" {
		return nil, errors.New("'name' required")
	}
	if in.AlertType == "" || in.AlertSeverity == "" || in.AlertTitle == "" {
		return nil, errors.New("'alertType' + 'alertSeverity' + 'alertTitle' required")
	}
	return &in, nil
}

func inputToRule(in *alertRuleInput, id int64) *database.AlertRule {
	return &database.AlertRule{
		ID:                   id,
		Name:                 in.Name,
		Enabled:              in.Enabled,
		MatchKind:            in.MatchKind,
		MatchSeverity:        in.MatchSeverity,
		MatchPayloadContains: in.MatchPayloadContains,
		AlertType:            in.AlertType,
		AlertSeverity:        in.AlertSeverity,
		AlertTitle:           in.AlertTitle,
		AlertMessage:         in.AlertMessage,
	}
}

func encodeAlertRule(rule *database.AlertRule) map[string]any {
	return map[string]any{
		"id":                   rule.ID,
		jsonKeyName:            rule.Name,
		jsonKeyEnabled:         rule.Enabled,
		"matchKind":            rule.MatchKind,
		"matchSeverity":        rule.MatchSeverity,
		"matchPayloadContains": rule.MatchPayloadContains,
		"alertType":            rule.AlertType,
		"alertSeverity":        rule.AlertSeverity,
		"alertTitle":           rule.AlertTitle,
		"alertMessage":         rule.AlertMessage,
		"createdAt":            formatTime(rule.CreatedAt),
		"updatedAt":            formatTime(rule.UpdatedAt),
	}
}

func encodeAlertRules(rules []*database.AlertRule) []map[string]any {
	out := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		out = append(out, encodeAlertRule(r))
	}
	return out
}
