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

	"github.com/MustardSeedNetworks/seed/internal/alerts/rules"
	"github.com/MustardSeedNetworks/seed/internal/logging"
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
	WindowSeconds        int    `json:"windowSeconds,omitempty"`
	ThresholdCount       int    `json:"thresholdCount,omitempty"`
}

func (s *Server) handleAlertRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listAlertRules(w, r)
	case http.MethodPost:
		s.createAlertRule(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleAlertRuleByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, alertRulesPathPrefix)
	if rest == "" || strings.Contains(rest, "/") {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Missing or invalid rule id")
		return
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, "Rule id must be a positive integer")
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
		writeError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) listAlertRules(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if !s.alertRules.Available() {
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
		return
	}
	enabledOnly := r.URL.Query().Get("enabled_only") == "true"
	ruleList, err := s.alertRules.List(r.Context(), enabledOnly)
	if err != nil {
		logger.ErrorContext(r.Context(), "list alert_rules failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to list rules")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		jsonKeyCount: len(ruleList),
		"rules":      encodeAlertRules(ruleList),
	})
}

func (s *Server) getAlertRule(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	if !s.alertRules.Available() {
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
		return
	}
	rule, err := s.alertRules.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, rules.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, ErrCodeNotFound, "Rule not found")
			return
		}
		logger.ErrorContext(r.Context(), "get alert_rule failed", "id", id, "error", err)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to load rule")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, encodeAlertRule(rule))
}

func (s *Server) createAlertRule(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if !s.alertRules.Available() {
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
		return
	}
	in, err := decodeAlertRuleInput(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, err.Error())
		return
	}
	rule, createErr := s.alertRules.Create(r.Context(), inputToRule(in))
	if createErr != nil {
		if ve, ok := errors.AsType[*rules.ValidationError](createErr); ok {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation, ve.Msg)
			return
		}
		logger.ErrorContext(r.Context(), "create alert_rule failed", "error", createErr)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to create rule")
		return
	}
	w.Header().Set("Location", alertRulesPathPrefix+strconv.FormatInt(rule.ID, 10))
	sendJSONResponse(w, logger, http.StatusOK, encodeAlertRule(rule))
}

func (s *Server) updateAlertRule(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	if !s.alertRules.Available() {
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
		return
	}
	in, err := decodeAlertRuleInput(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, err.Error())
		return
	}
	// The use-case writes the row and echoes the freshly-read record so the
	// JSON reflects updated_at (falling back to the written shape on re-read).
	rule, updateErr := s.alertRules.Update(r.Context(), id, inputToRule(in))
	if updateErr != nil {
		if errors.Is(updateErr, rules.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, ErrCodeNotFound, "Rule not found")
			return
		}
		logger.ErrorContext(r.Context(), "update alert_rule failed", "id", id, "error", updateErr)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to update rule")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, encodeAlertRule(rule))
}

func (s *Server) deleteAlertRule(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	if !s.alertRules.Available() {
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
		return
	}
	if err := s.alertRules.Delete(r.Context(), id); err != nil {
		if errors.Is(err, rules.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, ErrCodeNotFound, "Rule not found")
			return
		}
		logger.ErrorContext(r.Context(), "delete alert_rule failed", "id", id, "error", err)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Failed to delete rule")
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

// inputToRule maps the wire input to the use-case model. The ThresholdCount
// floor (>=1) is applied by the use-case, so it is passed through raw here.
func inputToRule(in *alertRuleInput) rules.Rule {
	return rules.Rule{
		Name:                 in.Name,
		Enabled:              in.Enabled,
		MatchKind:            in.MatchKind,
		MatchSeverity:        in.MatchSeverity,
		MatchPayloadContains: in.MatchPayloadContains,
		AlertType:            in.AlertType,
		AlertSeverity:        in.AlertSeverity,
		AlertTitle:           in.AlertTitle,
		AlertMessage:         in.AlertMessage,
		WindowSeconds:        in.WindowSeconds,
		ThresholdCount:       in.ThresholdCount,
	}
}

func encodeAlertRule(rule rules.Rule) map[string]any {
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
		"windowSeconds":        rule.WindowSeconds,
		"thresholdCount":       rule.ThresholdCount,
		"createdAt":            formatTime(rule.CreatedAt),
		"updatedAt":            formatTime(rule.UpdatedAt),
	}
}

func encodeAlertRules(ruleList []rules.Rule) []map[string]any {
	out := make([]map[string]any, 0, len(ruleList))
	for _, r := range ruleList {
		out = append(out, encodeAlertRule(r))
	}
	return out
}
