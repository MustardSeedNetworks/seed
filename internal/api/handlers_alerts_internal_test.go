package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// newAlertsTestServer mirrors newTopologyTestServer but exists as
// its own helper so the per-handler test file stays self-contained.
func newAlertsTestServer(t *testing.T) *Server {
	t.Helper()
	db := newTestDB(t)
	s := &Server{}
	s.dbConn = db
	s.alertInbox = app.NewAlertInbox(s.db)
	return s
}

// seedAlert inserts one alert via the repository and returns its
// generated ID.
func seedAlert(t *testing.T, db *database.DB, severity string) int64 {
	t.Helper()
	a := &database.Alert{
		Type:     database.AlertTypeConnectivity,
		Severity: severity,
		Title:    "test alert",
		Message:  "for testing",
		Source:   "10.0.0.1",
	}
	if err := db.Alerts().Create(context.Background(), a); err != nil {
		t.Fatalf("seed alert: %v", err)
	}
	return a.ID
}

func TestHandleAlerts_ReturnsList(t *testing.T) {
	s := newAlertsTestServer(t)
	seedAlert(t, s.db(), database.AlertSeverityWarning)
	seedAlert(t, s.db(), database.AlertSeverityError)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/alerts", http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlerts(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Count  int              `json:"count"`
		Alerts []map[string]any `json:"alerts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2", resp.Count)
	}
}

func TestHandleAlerts_SeverityFilter(t *testing.T) {
	s := newAlertsTestServer(t)
	seedAlert(t, s.db(), database.AlertSeverityWarning)
	seedAlert(t, s.db(), database.AlertSeverityError)

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/alerts?severity=error", http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlerts(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count != 1 {
		t.Errorf("count = %d, want 1 (only error severity)", resp.Count)
	}
}

func TestHandleAlerts_InvalidSinceReturns400(t *testing.T) {
	s := newAlertsTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/alerts?since=not-a-date", http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlerts(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleAlertAction_Acknowledge(t *testing.T) {
	s := newAlertsTestServer(t)
	id := seedAlert(t, s.db(), database.AlertSeverityWarning)

	url := APIVersionPrefix + "/alerts/" + strconv.FormatInt(id, 10) + "/acknowledge"
	req := httptest.NewRequest(http.MethodPost, url, http.NoBody)
	req.Header.Set("X-Username", "alice")
	w := httptest.NewRecorder()
	s.handleAlertAction(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	// Verify the row actually flipped via the repo.
	alert, err := s.db().Alerts().Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get alert: %v", err)
	}
	if !alert.Acknowledged {
		t.Error("alert should be acknowledged in DB")
	}
	if alert.AcknowledgedBy == nil || *alert.AcknowledgedBy != "alice" {
		t.Errorf("acknowledgedBy = %v, want alice", alert.AcknowledgedBy)
	}
}

func TestHandleAlertAction_Resolve(t *testing.T) {
	s := newAlertsTestServer(t)
	id := seedAlert(t, s.db(), database.AlertSeverityWarning)

	url := APIVersionPrefix + "/alerts/" + strconv.FormatInt(id, 10) + "/resolve"
	req := httptest.NewRequest(http.MethodPost, url, http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlertAction(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	alert, _ := s.db().Alerts().Get(context.Background(), id)
	if !alert.Resolved {
		t.Error("alert should be resolved in DB")
	}
}

func TestHandleAlertAction_BadPathReturns400(t *testing.T) {
	s := newAlertsTestServer(t)
	tests := []struct{ name, path string }{
		{"no action", APIVersionPrefix + "/alerts/1"},
		{"non-int id", APIVersionPrefix + "/alerts/abc/acknowledge"},
		{"zero id", APIVersionPrefix + "/alerts/0/acknowledge"},
		{"missing id", APIVersionPrefix + "/alerts//acknowledge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, http.NoBody)
			w := httptest.NewRecorder()
			s.handleAlertAction(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", w.Code)
			}
		})
	}
}

func TestHandleAlertAction_UnknownActionReturns400(t *testing.T) {
	s := newAlertsTestServer(t)
	id := seedAlert(t, s.db(), database.AlertSeverityWarning)
	url := APIVersionPrefix + "/alerts/" + strconv.FormatInt(id, 10) + "/explode"
	req := httptest.NewRequest(http.MethodPost, url, http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlertAction(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleAlerts_PaginationLimitClamped(t *testing.T) {
	s := newAlertsTestServer(t)
	for range 5 {
		seedAlert(t, s.db(), database.AlertSeverityInfo)
	}
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/alerts?limit=2", http.NoBody)
	w := httptest.NewRecorder()
	s.handleAlerts(w, req)
	var resp struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2 (limited)", resp.Count)
	}
}

func TestUsernameFromRequest_FallsBackToSystem(t *testing.T) {
	s := &Server{}
	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	if got := s.usernameFromRequest(r); got != "system" {
		t.Errorf("got %q, want system", got)
	}
}

// Make sure seedAlert + the underlying repo handle the createdAt
// default — without this test a regression elsewhere could leave
// the JSON encoder serializing a zero time and clients seeing
// "0001-01-01T00:00:00Z" instead of "".
func TestSeedAlert_HasNonZeroCreatedAt(t *testing.T) {
	s := newAlertsTestServer(t)
	id := seedAlert(t, s.db(), database.AlertSeverityWarning)
	alert, _ := s.db().Alerts().Get(context.Background(), id)
	if alert.CreatedAt.IsZero() {
		t.Error("CreatedAt should be auto-populated by repo")
	}
	if !alert.CreatedAt.Before(time.Now().UTC().Add(time.Minute)) {
		t.Errorf("CreatedAt looks wrong: %v", alert.CreatedAt)
	}
}
