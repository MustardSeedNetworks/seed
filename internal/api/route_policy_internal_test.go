package api

// route_policy_test.go enforces that method + body-limit are declared in the
// capability registry for every /api/v1 route (ADR-0002), and that the method
// gate actually rejects undeclared methods. This is the "no route bypasses the
// method/body-limit policy" guard for finding #6.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// apiPath strips a Go 1.22 method prefix ("GET /api/v1/x" -> "/api/v1/x").
func apiPath(path string) string {
	if _, ok := methodFromPath(path); ok {
		if i := strings.IndexByte(path, ' '); i > 0 {
			return path[i+1:]
		}
	}
	return path
}

// TestEveryAPIRouteDeclaresMethodAndBodyLimit fails if any /api/v1 route is
// registered without a declared method or body limit — i.e. bypasses the
// per-route policy the registry is meant to make authoritative.
func TestEveryAPIRouteDeclaresMethodAndBodyLimit(t *testing.T) {
	s := NewTestServer()
	defer s.Close()

	if len(s.manifest) == 0 {
		t.Fatal("route manifest is empty; setupRoutes did not run")
	}

	for _, rt := range s.manifest {
		if !strings.HasPrefix(apiPath(rt.path), APIVersionPrefix) {
			continue // infra introspection / static — not an API surface
		}
		if len(rt.methods) == 0 {
			t.Errorf("route %q bypasses the method policy: no method declared "+
				"(set route.methods, or use a Go 1.22 method-prefixed path)", rt.path)
		}
		if rt.maxBodyBytes <= 0 {
			t.Errorf("route %q has no body limit (register() should default it)", rt.path)
		}
	}
}

// TestMethodGateRejectsUndeclaredMethod proves the registry's method gate
// enforces the declared set: an undeclared method gets 405 + Allow, and the
// declared method reaches the handler.
func TestMethodGateRejectsUndeclaredMethod(t *testing.T) {
	s := NewTestServer()
	defer s.Close()

	// /api/v1/status is public and GET-only.
	const path = APIVersionPrefix + "/status"

	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST %s: got status %d, want 405", path, rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("405 Allow header: got %q, want %q", allow, http.MethodGet)
	}

	rec = httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET %s: got status %d, want 200", path, rec.Code)
	}
}

// TestPersistentWriteRoutesRequireOperator pins the operator-role gate on routes
// whose mutating methods persist state (config-to-disk or a live kernel change).
// These bypassed writeGated until the role gate was added; this guards against a
// future edit silently dropping minRole and re-opening the write to any viewer.
func TestPersistentWriteRoutesRequireOperator(t *testing.T) {
	s := NewTestServer()
	defer s.Close()

	// Persistent-write routes that MUST require operator+ on their mutating
	// methods. (Diagnostic-action POSTs — scans/probes/speedtest — are
	// deliberately ungated and intentionally absent from this list.)
	wantOperator := map[string]bool{
		APIVersionPrefix + "/interface":                  true, // PUT persists active NIC
		APIVersionPrefix + "/telemetry/vlan/interface":   true, // POST creates kernel VLAN
		APIVersionPrefix + "/security/discovery/options": true, // PUT saves discovery config
		APIVersionPrefix + "/security/devices/subnets":   true, // CRUD on persisted subnets
	}

	seen := make(map[string]bool, len(wantOperator))
	for _, rt := range s.manifest {
		if !wantOperator[apiPath(rt.path)] {
			continue
		}
		seen[apiPath(rt.path)] = true
		if rt.minRole != database.RoleOperator {
			t.Errorf("route %q: minRole = %q, want %q (persistent write must be operator-gated)",
				rt.path, rt.minRole, database.RoleOperator)
		}
	}
	for path := range wantOperator {
		if !seen[path] {
			t.Errorf("expected persistent-write route %q not found in manifest", path)
		}
	}
}
