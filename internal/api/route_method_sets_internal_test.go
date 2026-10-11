package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRouteMethodSetsMatchTheirHandlers pins #3261: these routes declared
// methods their handler never serves, so the request passed the method gate,
// auth and CSRF only to be refused by the handler — and the OpenAPI document,
// generated from the registry, listed operations that always answer 405. The
// gate now refuses them itself and names the methods that do work.
func TestRouteMethodSetsMatchTheirHandlers(t *testing.T) {
	s := ssoGateServer(t)

	tests := []struct {
		path    string
		allow   string
		refused []string
	}{
		{APIVersionPrefix + "/telemetry/dns", "GET", []string{http.MethodPost}},
		{APIVersionPrefix + "/wifi/wifi", "GET", []string{http.MethodPost, http.MethodPut}},
		{APIVersionPrefix + "/profiles", "GET, POST", []string{http.MethodPut, http.MethodDelete}},
		{APIVersionPrefix + "/telemetry/dhcp/rogue/servers", "GET", []string{http.MethodPost, http.MethodDelete}},
		{APIVersionPrefix + "/telemetry/probes/run", "GET", []string{http.MethodPost}},
		{APIVersionPrefix + "/telemetry/publicip", "GET, POST", []string{http.MethodPut}},
		{APIVersionPrefix + "/security/devices", "GET", []string{http.MethodPost}},
	}
	for _, tt := range tests {
		for _, method := range tt.refused {
			t.Run(method+" "+tt.path, func(t *testing.T) {
				w := httptest.NewRecorder()
				s.Handler().ServeHTTP(w, authedRequest(t, s, method, tt.path, "admin", "{}"))

				require.Equal(t, http.StatusMethodNotAllowed, w.Code, w.Body.String())
				require.Equal(t, tt.allow, w.Header().Get("Allow"))
			})
		}
	}
}

// TestProfilesCollectionStillListsAndCreates guards the narrowing above: the
// exact /profiles route keeps the two operations its handler serves.
func TestProfilesCollectionStillListsAndCreates(t *testing.T) {
	s := ssoGateServer(t)
	const path = APIVersionPrefix + "/profiles"

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authedRequest(t, s, http.MethodPost, path, "admin", `{"name":"lab"}`))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authedRequest(t, s, http.MethodGet, path, "admin", ""))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"lab"`)
}
