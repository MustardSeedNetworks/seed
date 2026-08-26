package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDHCPLeaseRejectsAnUnknownInterface pins that a name this host does not
// have is a 400, not a 200 carrying a diagnostic failure. The two are different
// answers: "you asked for something that does not exist" versus "the thing you
// asked about is unhealthy".
func TestDHCPLeaseRejectsAnUnknownInterface(t *testing.T) {
	s := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/dhcp/lease",
		strings.NewReader(`{"interface":"definitely-not-an-interface"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleDHCPLease(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", rec.Code)
	}
}

// TestDHCPLeaseReportsARealInterface drives the handler against an interface
// this machine actually has, so the wiring is exercised end to end rather than
// only on its error paths.
func TestDHCPLeaseReportsARealInterface(t *testing.T) {
	names, err := hostInterfaceNames()
	if err != nil || len(names) == 0 {
		t.Skip("no interfaces available on this host")
	}

	s := &Server{}
	body := `{"interface":"` + names[0] + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/dhcp/lease",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleDHCPLease(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var result map[string]any
	if decodeErr := json.Unmarshal(rec.Body.Bytes(), &result); decodeErr != nil {
		t.Fatalf("response is not JSON: %v", decodeErr)
	}
	// The interface is echoed back whatever the lease state, so a caller can
	// tell which interface a result belongs to.
	if got, _ := result["interface"].(string); got != names[0] {
		t.Errorf("interface = %q, want %q", got, names[0])
	}
	if _, ok := result["status"]; !ok {
		t.Error("response carries no status")
	}
}

// TestDHCPLeaseRouteIsRegistered pins the route and its policy, since the whole
// point of this change is that the capability is reachable.
func TestDHCPLeaseRouteIsRegistered(t *testing.T) {
	s := newRoutePolicyServerForDHCP(t)

	for _, rt := range s.manifest {
		if rt.path != APIVersionPrefix+"/telemetry/dhcp/lease" {
			continue
		}
		if !rt.rateLimited {
			t.Error("the lease route is not rate limited; it shells out to a " +
				"platform command")
		}
		return
	}
	t.Fatal("the DHCP lease route is not registered")
}

func newRoutePolicyServerForDHCP(t *testing.T) *Server {
	t.Helper()
	s := &Server{mux: http.NewServeMux()}
	s.setupRoutes()
	return s
}
