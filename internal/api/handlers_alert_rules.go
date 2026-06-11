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
	if s.db() == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	enabledOnly := r.URL.Query().Get("enabled_only") == "true"
	ruleList, err := s.alertRules.List(r.Context(), enabledOnly)
	if err != nil {
		logger.ErrorContext(r.Context(), "list alert_rules failed", "error", err)
		http.Error(w, "Failed to list rules", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, map[string]any{
		jsonKeyCount: len(ruleList),
		"rules":      encodeAlertRules(ruleList),
	})
}

func (s *Server) getAlertRule(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	if s.db() == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	rule, err := s.alertRules.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, rules.ErrNotFound) {
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
	if s.db() == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	in, err := decodeAlertRuleInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rule, createErr := s.alertRules.Create(r.Context(), inputToRule(in))
	if createErr != nil {
		var ve *rules.ValidationError
		if errors.As(createErr, &ve) {
			http.Error(w, ve.Msg, http.StatusBadRequest)
			return
		}
		logger.ErrorContext(r.Context(), "create alert_rule failed", "error", createErr)
		http.Error(w, "Failed to create rule", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", alertRulesPathPrefix+strconv.FormatInt(rule.ID, 10))
	writeJSON(w, r, encodeAlertRule(rule))
}

func (s *Server) updateAlertRule(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	if s.db() == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	in, err := decodeAlertRuleInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// The use-case writes the row and echoes the freshly-read record so the
	// JSON reflects updated_at (falling back to the written shape on re-read).
	rule, updateErr := s.alertRules.Update(r.Context(), id, inputToRule(in))
	if updateErr != nil {
		if errors.Is(updateErr, rules.ErrNotFound) {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		logger.ErrorContext(r.Context(), "update alert_rule failed", "id", id, "error", updateErr)
		http.Error(w, "Failed to update rule", http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, encodeAlertRule(rule))
}

func (s *Server) deleteAlertRule(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	if s.db() == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}
	if err := s.alertRules.Delete(r.Context(), id); err != nil {
		if errors.Is(err, rules.ErrNotFound) {
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
