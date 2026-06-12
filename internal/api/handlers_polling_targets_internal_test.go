package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// newPollingTargetsTestServer returns a Server with only the bits
// the polling-targets handlers touch.
func newPollingTargetsTestServer(t *testing.T) *Server {
	t.Helper()
	db := newTestDB(t)
	s := &Server{}
	s.dbConn = db
	s.pollingTargets = app.NewPollingTargets(s.db)
	return s
}

// seedTarget inserts a polling target via the repo and returns it.
func seedTarget(t *testing.T, db *database.DB, name string) *database.PollingTarget {
	t.Helper()
	target := &database.PollingTarget{
		Name:            name,
		IPAddress:       "10.0.0.1",
		SNMPVersion:     "v2c",
		PollIntervalSec: 60,
		Enabled:         true,
		CollectorChain:  []string{"sys_info"},
	}
	if err := db.PollingTargets().Create(context.Background(), target); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	return target
}

func postJSON(t *testing.T, body any) *http.Request {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		t.Fatalf("encode body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/polling-targets", buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandlePollingTargets_GETReturnsList(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	seedTarget(t, s.db(), "router-1")
	seedTarget(t, s.db(), "router-2")

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/polling-targets", http.NoBody)
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Count   int              `json:"count"`
		Targets []map[string]any `json:"targets"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2", resp.Count)
	}
}

func TestHandlePollingTargets_POSTCreates(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	req := postJSON(t, map[string]any{
		"name":                "router-1",
		"ipAddress":           "10.0.0.1",
		"snmpVersion":         "v2c",
		"pollIntervalSeconds": 60,
		"enabled":             true,
	})
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc == "" {
		t.Error("missing Location header on create")
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "router-1" || resp["ipAddress"] != "10.0.0.1" {
		t.Errorf("response = %+v", resp)
	}
	// Confirm the row landed in the DB.
	list, _ := s.db().PollingTargets().List(context.Background(), "")
	if len(list) != 1 {
		t.Errorf("db rows = %d, want 1", len(list))
	}
}

func TestHandlePollingTargets_POSTMissingNameReturns400(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	req := postJSON(t, map[string]any{"ipAddress": "10.0.0.1"})
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlePollingTargets_POSTMissingIPReturns400(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	req := postJSON(t, map[string]any{"name": "router-1"})
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlePollingTargets_POSTUnknownFieldsReturns400(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	// "extra" isn't a known input field; the disallow-unknown-fields
	// decoder rejects the request.
	body := []byte(`{"name":"r","ipAddress":"10.0.0.1","extra":true}`)
	req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/polling-targets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlePollingTargets_UnknownMethodReturns405(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	req := httptest.NewRequest(http.MethodPatch, APIVersionPrefix+"/polling-targets", http.NoBody)
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestHandlePollingTargetByID_GET(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	target := seedTarget(t, s.db(), "router-1")

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/polling-targets/"+target.ID, http.NoBody)
	w := httptest.NewRecorder()
	s.handlePollingTargetByID(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["id"] != target.ID {
		t.Errorf("id = %v, want %v", resp["id"], target.ID)
	}
}

func TestHandlePollingTargetByID_GETUnknownReturns404(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/polling-targets/no-such-id", http.NoBody)
	w := httptest.NewRecorder()
	s.handlePollingTargetByID(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlePollingTargetByID_PUTUpdates(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	target := seedTarget(t, s.db(), "old-name")

	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(map[string]any{
		"name":                "new-name",
		"ipAddress":           "10.0.0.99",
		"snmpVersion":         "v2c",
		"pollIntervalSeconds": 120,
		"enabled":             false,
		"collectorChain":      []string{"sys_info", "if_table"},
	})
	req := httptest.NewRequest(http.MethodPut, APIVersionPrefix+"/polling-targets/"+target.ID, buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handlePollingTargetByID(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	// Re-fetch from DB to confirm.
	got, _ := s.db().PollingTargets().Get(context.Background(), target.ID)
	if got.Name != "new-name" || got.IPAddress != "10.0.0.99" {
		t.Errorf("update didn't persist: %+v", got)
	}
	if got.Enabled {
		t.Error("Enabled should be false after update")
	}
}

func TestHandlePollingTargetByID_PUTUnknownReturns404(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(map[string]any{
		"name":      "x",
		"ipAddress": "10.0.0.1",
	})
	req := httptest.NewRequest(http.MethodPut, APIVersionPrefix+"/polling-targets/no-such-id", buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handlePollingTargetByID(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlePollingTargetByID_DELETERemoves(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	target := seedTarget(t, s.db(), "to-delete")

	req := httptest.NewRequest(http.MethodDelete, APIVersionPrefix+"/polling-targets/"+target.ID, http.NoBody)
	w := httptest.NewRecorder()
	s.handlePollingTargetByID(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}

	if _, err := s.db().PollingTargets().Get(context.Background(), target.ID); err == nil {
		t.Error("target should be deleted from DB")
	}
}

func TestHandlePollingTargetByID_DELETEUnknownReturns404(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, APIVersionPrefix+"/polling-targets/no-such-id", http.NoBody)
	w := httptest.NewRecorder()
	s.handlePollingTargetByID(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandlePollingTargetByID_BadPathReturns400(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	// trailing-slash-only path = empty id.
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/polling-targets/", http.NoBody)
	w := httptest.NewRecorder()
	s.handlePollingTargetByID(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandlePollingTargetByID_PATCHRejected(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	target := seedTarget(t, s.db(), "r-1")
	req := httptest.NewRequest(http.MethodPatch, APIVersionPrefix+"/polling-targets/"+target.ID, http.NoBody)
	w := httptest.NewRecorder()
	s.handlePollingTargetByID(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}
