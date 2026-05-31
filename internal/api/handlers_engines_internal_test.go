package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
	// 9 engines from initDatabaseDependentServices (verified by
	// the wire-up test in server_a4_engines_internal_test.go).
	const wantCount = 9
	if resp.Count != wantCount {
		t.Errorf("count = %d, want %d", resp.Count, wantCount)
	}
	names := map[string]bool{}
	for _, e := range resp.Engines {
		if n, ok := e["name"].(string); ok {
			names[n] = true
		}
	}
	for _, expected := range []string{
		"probe", "retention", "snmp-poller",
		"topology-sysinfo-reconciler", "alert-listener-pipeline",
	} {
		if !names[expected] {
			t.Errorf("missing engine %q in response", expected)
		}
	}
}

func TestHandleEngines_RejectsNonGET(t *testing.T) {
	s := &Server{services: NewServiceContainer()}
	req := httptest.NewRequest(http.MethodPost, APIVersionPrefix+"/engines", http.NoBody)
	w := httptest.NewRecorder()
	s.handleEngines(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
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
