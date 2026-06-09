package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/engine"
)

// minimalEngine is an Engine that does not implement Reporter — used
// to exercise the type-assertion fallback in handleEngines.
type minimalEngine struct{ name string }

func (m *minimalEngine) Name() string                { return m.name }
func (*minimalEngine) Start(_ context.Context) error { return nil }
func (*minimalEngine) Stop(_ context.Context) error  { return nil }

// reportingEngine implements both Engine and Reporter — exercises
// the rich-status path.
type reportingEngine struct {
	name   string
	status engine.Status
}

func (r *reportingEngine) Name() string                { return r.name }
func (*reportingEngine) Start(_ context.Context) error { return nil }
func (*reportingEngine) Stop(_ context.Context) error  { return nil }
func (r *reportingEngine) Status() engine.Status       { return r.status }

func TestHandleEngines_ReturnsRegistryContents(t *testing.T) {
	s := &Server{services: NewServiceContainer()}
	initDatabaseDependentServices(s.services, newTestDB(t))

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/engines", http.NoBody)
	w := httptest.NewRecorder()
	s.handleEngines(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Count   int              `json:"count"`
		Engines []map[string]any `json:"engines"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// initDatabaseDependentServices wires a Free-tier
	// license.Manager (no key file in tests), so Stage A5.9
	// gating filters out everything above Free. The Free-tier
	// engines (probe + retention) land.
	names := map[string]bool{}
	for _, e := range resp.Engines {
		if n, ok := e["name"].(string); ok {
			names[n] = true
		}
	}
	for _, expected := range []string{"probe", "retention"} {
		if !names[expected] {
			t.Errorf("missing free-tier engine %q in response; got %v", expected, names)
		}
	}
}
func TestHandleEngines_NoServicesReturnsEmpty(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/engines", http.NoBody)
	w := httptest.NewRecorder()
	s.handleEngines(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count != 0 {
		t.Errorf("count = %d, want 0", resp.Count)
	}
}

func TestHandleEngines_EngineWithoutReporter_DefaultsToStateOK(t *testing.T) {
	s := &Server{services: NewServiceContainer()}
	if regErr := s.services.Engines.Register(&minimalEngine{name: "plain"}); regErr != nil {
		t.Fatalf("register: %v", regErr)
	}

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/engines", http.NoBody)
	w := httptest.NewRecorder()
	s.handleEngines(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Engines []map[string]any `json:"engines"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Engines) == 0 {
		t.Fatal("expected at least one engine entry")
	}
	got := resp.Engines[0]
	if got["state"] != "ok" {
		t.Errorf("state = %v, want \"ok\" (engine without Reporter)", got["state"])
	}
	if got["lastError"] != "" {
		t.Errorf("lastError = %v, want empty", got["lastError"])
	}
}

func TestHandleEngines_EngineWithReporter_SurfacesStatus(t *testing.T) {
	s := &Server{services: NewServiceContainer()}
	rep := &reportingEngine{
		name: "rich",
		status: engine.Status{
			State:      engine.StateDegraded,
			LastTickAt: time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
			LastError:  "scan timeout: ctx deadline exceeded",
			Inflight:   3,
		},
	}
	if regErr := s.services.Engines.Register(rep); regErr != nil {
		t.Fatalf("register: %v", regErr)
	}

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/engines", http.NoBody)
	w := httptest.NewRecorder()
	s.handleEngines(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp struct {
		Engines []map[string]any `json:"engines"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Engines) == 0 {
		t.Fatal("expected at least one engine entry")
	}
	got := resp.Engines[0]
	if got["state"] != "degraded" {
		t.Errorf("state = %v, want \"degraded\"", got["state"])
	}
	if got["lastError"] != "scan timeout: ctx deadline exceeded" {
		t.Errorf("lastError = %v", got["lastError"])
	}
	if got["lastTickAt"] != "2026-05-31T12:00:00Z" {
		t.Errorf("lastTickAt = %v, want RFC3339 string", got["lastTickAt"])
	}
	// JSON decode of integer field yields float64 — cast through that.
	if inflight, ok := got["inflight"].(float64); !ok || int(inflight) != 3 {
		t.Errorf("inflight = %v (%T), want 3", got["inflight"], got["inflight"])
	}
}

func TestHandleEngines_ReporterReturnsEmptyState_FillsOK(t *testing.T) {
	s := &Server{services: NewServiceContainer()}
	if regErr := s.services.Engines.Register(&reportingEngine{
		name:   "blank-state",
		status: engine.Status{}, // State left empty
	}); regErr != nil {
		t.Fatalf("register: %v", regErr)
	}
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/engines", http.NoBody)
	w := httptest.NewRecorder()
	s.handleEngines(w, req)
	var resp struct {
		Engines []map[string]any `json:"engines"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Engines[0]["state"] != "ok" {
		t.Errorf("blank Status.State should default to ok, got %v", resp.Engines[0]["state"])
	}
}
