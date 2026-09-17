// SPDX-License-Identifier: BUSL-1.1

package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

// selectableInterface asks the server which interfaces it offers and returns
// the first one, so the test names whatever this host actually has rather than
// assuming a topology. Selection rejects anything the manager does not list,
// loopback included.
func selectableInterface(t *testing.T, server *api.Server) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/interfaces", http.NoBody)
	w := httptest.NewRecorder()
	server.Mux().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list interfaces: status = %d: %s", w.Code, w.Body.String())
	}

	var ifaces []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(w.Body).Decode(&ifaces); err != nil {
		t.Fatalf("decode interfaces: %v", err)
	}
	for i := range ifaces {
		if ifaces[i].Name != "" {
			return ifaces[i].Name
		}
	}
	t.Skip("this host offers no selectable interface")
	return ""
}

// Selecting an interface must re-scope gateway detection with it. Before
// #2690 it did not, so the Network page kept reporting — and pinging — the
// gateway of whichever interface carried the default route.
func TestSelectingAnInterfaceRescopesGatewayDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	iface := selectableInterface(t, server)
	if server.GatewayTesterInterface() == iface {
		t.Fatalf("test is vacuous: the tester already names %q before the request", iface)
	}

	body, err := json.Marshal(map[string]string{"interface": iface})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/interface", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := server.GatewayTesterInterface(); got != iface {
		t.Errorf("gateway tester interface = %q, want %q", got, iface)
	}
}

// Selecting an interface must re-scope the resolvers too. The DNS card sat
// beside the gateway card showing the host's resolvers whatever interface was
// selected, and measured a lookup through whichever link carried the route
// (#2690).
func TestSelectingAnInterfaceRescopesDNSResolvers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	iface := selectableInterface(t, server)
	if server.DNSTesterInterface() == iface {
		t.Fatalf("test is vacuous: the tester already names %q before the request", iface)
	}

	body, err := json.Marshal(map[string]string{"interface": iface})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/interface", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Mux().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := server.DNSTesterInterface(); got != iface {
		t.Errorf("dns tester interface = %q, want %q", got, iface)
	}
}
