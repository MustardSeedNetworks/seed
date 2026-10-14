package api

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
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
		// Not a #3261 mismatch: these served, as aliases of /profiles/switch.
		{APIVersionPrefix + "/profiles/active", "GET", []string{http.MethodPost, http.MethodPut}},
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

// TestProfileSwitchAnswersTheActivatedProfile pins the one remaining way to
// change the active profile and the body it answers, now that /profiles/active
// is read only.
func TestProfileSwitchAnswersTheActivatedProfile(t *testing.T) {
	s := ssoGateServer(t)
	const profiles = APIVersionPrefix + "/profiles"
	serve := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, authedRequest(t, s, method, path, "admin", body))
		return w
	}

	w := serve(http.MethodPost, profiles, `{"name":"lab"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created ProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	w = serve(http.MethodPost, profiles+"/switch", `{"profileId":"`+created.ID+`"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var switched map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &switched))
	require.ElementsMatch(t, []string{"message", "profile"}, slices.Collect(maps.Keys(switched)))
	var profile ProfileResponse
	require.NoError(t, json.Unmarshal(switched["profile"], &profile))
	require.Equal(t, created.ID, profile.ID)

	w = serve(http.MethodGet, profiles+"/active", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), created.ID)
}
