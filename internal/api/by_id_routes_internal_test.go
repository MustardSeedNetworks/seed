package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
)

// routedCRUDServer serves the by-id routes through the daemon's own route
// table, with an admin for Mux's session.
func routedCRUDServer(t *testing.T) *Server {
	t.Helper()
	s := newDeviceCredentialsTestServer(t)
	s.pollingTargets = app.NewPollingTargets(s.db, s.pollingTargetLimit)
	s.alertRules = app.NewAlertRules(s.db)
	s.historyQueries = app.NewHistory(s.db)
	s.topologyQueries = app.NewTopologyQueries(s.db, topologyMaxLimit)
	s.discoveryDevices = app.NewDiscoveryDevices(s.discoveryEngine)
	s.initIdentityUseCases()
	if _, err := s.db().CreateUser(t.Context(), "admin", "$2a$10$x", roles.Admin); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	s.withRouteDeps(t)
	s.setupRoutes()
	return s
}

// Each by-id route reaches its handler through its {id} pattern, and a path
// with a segment past the id is not a resource.
func TestByIDRoutesThroughTheRouter(t *testing.T) {
	t.Parallel()
	s := routedCRUDServer(t)

	target := pollingTargetsPath + "/" + seedTarget(t, s.db(), "router-1").ID
	rule := alertRulesPath + "/" + strconv.FormatInt(seedAlertRule(t, s.db(), "cpu", true).ID, 10)
	created := postCredential(t, s, `{"name":"core","community":"public"}`)
	var cred struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &cred); err != nil || cred.ID == "" {
		t.Fatalf("create credential: %d %s", created.Code, created.Body.String())
	}
	credential := deviceCredentialsPath + "/" + cred.ID

	for _, tc := range []struct {
		method string
		path   string
		body   string
		want   int
	}{
		{http.MethodGet, target, "", http.StatusOK},
		{http.MethodPut, target, `{"name":"router-2","ipAddress":"10.0.0.2"}`, http.StatusOK},
		{http.MethodGet, target + "/extra", "", http.StatusNotFound},
		{http.MethodDelete, target, "", http.StatusNoContent},
		{http.MethodGet, target, "", http.StatusNotFound},

		{http.MethodGet, rule, "", http.StatusOK},
		{http.MethodGet, rule + "/extra", "", http.StatusNotFound},
		{http.MethodDelete, rule, "", http.StatusNoContent},
		{http.MethodGet, rule, "", http.StatusNotFound},

		{http.MethodGet, credential, "", http.StatusOK},
		{http.MethodGet, credential + "/extra", "", http.StatusNotFound},
		{http.MethodDelete, credential, "", http.StatusNoContent},
		{http.MethodGet, credential, "", http.StatusNotFound},

		{http.MethodGet, jobsPath + "/a/b", "", http.StatusNotFound},

		{http.MethodDelete, APIVersionPrefix + "/tokens/unknown", "", http.StatusNotFound},
		{http.MethodDelete, APIVersionPrefix + "/tokens/unknown/extra", "", http.StatusNotFound},

		{http.MethodGet, APIVersionPrefix + "/users/admin", "", http.StatusOK},
		{http.MethodGet, APIVersionPrefix + "/users/admin/extra", "", http.StatusNotFound},

		{http.MethodGet, APIVersionPrefix + "/history/probes/never-run", "", http.StatusOK},
		{http.MethodGet, APIVersionPrefix + "/history/probes/never-run/extra", "", http.StatusNotFound},

		{http.MethodGet, APIVersionPrefix + "/topology/nodes/no-such-node", "", http.StatusNotFound},
		{http.MethodGet, APIVersionPrefix + "/topology/nodes/a/b", "", http.StatusNotFound},

		{http.MethodGet, APIVersionPrefix + "/discovery/engine/device/AA:BB:CC:DD:EE:FF", "", http.StatusServiceUnavailable},
		{http.MethodGet, APIVersionPrefix + "/discovery/engine/device/AA:BB/extra", "", http.StatusNotFound},
	} {
		rec := httptest.NewRecorder()
		s.Mux().ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if rec.Code != tc.want {
			t.Errorf("%s %s = %d, want %d: %s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}
