package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

func newAlertRulesTestServer(t *testing.T) *Server {
	t.Helper()
	db := newTestDB(t)
	s := &Server{services: NewServiceContainer()}
	s.services.Database = &DatabaseServices{DB: db}
	s.alertRules = app.NewAlertRules(s.db)
	return s
}

func seedAlertRule(t *testing.T, db *database.DB, name string, enabled bool) *database.AlertRule {
	t.Helper()
	rule := &database.AlertRule{
		Name:          name,
		Enabled:       enabled,
		MatchKind:     "syslog-udp",
		MatchSeverity: "error",
		AlertType:     database.AlertTypeSystem,
		AlertSeverity: database.AlertSeverityError,
		AlertTitle:    "Test rule",
		AlertMessage:  "Triggered by test seed",
	}
	if err := db.AlertRules().Create(context.Background(), rule); err != nil {
		t.Fatalf("seed alert_rule: %v", err)
	}
	return rule
}

func TestHandleAlertRules_GETReturnsList(t *testing.T) {
	s := newAlertRulesTestServer(t)
	seedAlertRule(t, s.db(), "rule-1", true)
	seedAlertRule(t, s.db(), "rule-2", false)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/alert-rules", http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlertRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp struct {
		Count int              `json:"count"`
		Rules []map[string]any `json:"rules"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2", resp.Count)
	}
}

func TestHandleAlertRules_EnabledOnlyFilter(t *testing.T) {
	s := newAlertRulesTestServer(t)
	seedAlertRule(t, s.db(), "rule-enabled", true)
	seedAlertRule(t, s.db(), "rule-disabled", false)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/alert-rules?enabled_only=true", http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlertRules(w, req)
	var resp struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count != 1 {
		t.Errorf("count = %d, want 1 (enabled_only)", resp.Count)
	}
}

func TestHandleAlertRules_POSTCreates(t *testing.T) {
	s := newAlertRulesTestServer(t)
	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(map[string]any{
		"name":          "new-rule",
		"enabled":       true,
		"matchKind":     "syslog-udp",
		"matchSeverity": "critical",
		"alertType":     "system",
		"alertSeverity": "critical",
		"alertTitle":    "Critical syslog from {{.SourceAddr}}",
		"alertMessage":  "Severity critical event observed",
	})
	req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/alert-rules", buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleAlertRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Location") == "" {
		t.Error("missing Location header")
	}
	list, _ := s.db().AlertRules().List(context.Background(), false)
	if len(list) != 1 {
		t.Errorf("db rows = %d, want 1", len(list))
	}
}

func TestHandleAlertRules_POSTMissingFieldsReturns400(t *testing.T) {
	s := newAlertRulesTestServer(t)
	tests := []struct {
		name string
		body map[string]any
	}{
		{"missing name", map[string]any{"alertType": "system", "alertSeverity": "info", "alertTitle": "t"}},
		{"missing alertType", map[string]any{"name": "r", "alertSeverity": "info", "alertTitle": "t"}},
		{"missing alertSeverity", map[string]any{"name": "r", "alertType": "system", "alertTitle": "t"}},
		{"missing alertTitle", map[string]any{"name": "r", "alertType": "system", "alertSeverity": "info"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			_ = json.NewEncoder(buf).Encode(tt.body)
			req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/alert-rules", buf)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.handleAlertRules(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", w.Code)
			}
		})
	}
}

func TestHandleAlertRuleByID_GET(t *testing.T) {
	s := newAlertRulesTestServer(t)
	rule := seedAlertRule(t, s.db(), "rule-detail", true)

	url := APIVersionPrefix + "/alert-rules/" + strconv.FormatInt(rule.ID, 10)
	req := httptest.NewRequest(http.MethodGet, url, http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlertRuleByID(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "rule-detail" {
		t.Errorf("name = %v", resp["name"])
	}
}

func TestHandleAlertRuleByID_GETUnknownReturns404(t *testing.T) {
	s := newAlertRulesTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/alert-rules/9999", http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlertRuleByID(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleAlertRuleByID_PUTUpdates(t *testing.T) {
	s := newAlertRulesTestServer(t)
	rule := seedAlertRule(t, s.db(), "to-update", true)

	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(map[string]any{
		"name":          "updated-name",
		"enabled":       false,
		"alertType":     "performance",
		"alertSeverity": "warning",
		"alertTitle":    "updated title",
		"alertMessage":  "updated message",
	})
	url := APIVersionPrefix + "/alert-rules/" + strconv.FormatInt(rule.ID, 10)
	req := httptest.NewRequest(http.MethodPut, url, buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleAlertRuleByID(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}

	got, _ := s.db().AlertRules().Get(context.Background(), rule.ID)
	if got.Name != "updated-name" || got.Enabled {
		t.Errorf("update didn't persist: %+v", got)
	}
}

func TestHandleAlertRuleByID_DELETERemoves(t *testing.T) {
	s := newAlertRulesTestServer(t)
	rule := seedAlertRule(t, s.db(), "to-delete", true)

	url := APIVersionPrefix + "/alert-rules/" + strconv.FormatInt(rule.ID, 10)
	req := httptest.NewRequest(http.MethodDelete, url, http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlertRuleByID(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
	if _, err := s.db().AlertRules().Get(context.Background(), rule.ID); err == nil {
		t.Error("rule should be deleted")
	}
}

func TestHandleAlertRuleByID_NonIntIDReturns400(t *testing.T) {
	s := newAlertRulesTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/alert-rules/abc", http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlertRuleByID(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleAlertRules_UnknownMethodReturns405(t *testing.T) {
	s := newAlertRulesTestServer(t)
	req := httptest.NewRequest(http.MethodPatch, APIVersionPrefix+"/alert-rules", http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlertRules(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}
