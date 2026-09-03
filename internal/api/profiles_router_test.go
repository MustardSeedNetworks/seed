package api_test

// profiles_router_test.go drives every profile path through server.Handler(),
// the full chain the daemon actually serves — recover, logging, CORS, auth,
// CSRF, then the mux.
//
// Driving the handler directly would have passed throughout the entire period
// the API was broken. handleProfiles trimmed "/api/profiles" from a path the
// registry mounts at "/api/v1/profiles", so nothing was trimmed and every
// request fell through to the by-ID cases carrying "api/v1/profiles" as the
// profile id. A test calling s.handleProfiles(w, r) constructs the request
// itself and can hand it whatever path it likes; only the mux exposes the
// mismatch between where a route is mounted and what its handler expects.
//
// The route-consumer gate cannot catch this either, by its own admission: the
// UI's paths all match the registered "/profiles/" prefix route, so the gate
// sees a served path. What is inside a prefix route is only observable by
// making the request.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// profileRouterFixture is a DB-backed server on the full Handler() chain, with
// an operator session and a CSRF token — everything a mutating profile request
// needs to reach its handler.
type profileRouterFixture struct {
	ts    *httptest.Server
	token string
	csrf  string
}

func newProfileRouterFixture(t *testing.T) *profileRouterFixture {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")
	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save test config: %v", err)
	}

	db, err := database.Open(filepath.Join(tmpDir, "seed.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	hash, err := auth.HashPassword("test-operator-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err = db.CreateUser(t.Context(), "operator", hash, "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	netMgr, err := netif.NewManager("")
	if err != nil {
		t.Logf("warning: network manager: %v", err)
	}

	server := api.NewServer(cfg, configPath, "", netMgr, false, nil, db, nil)
	t.Cleanup(server.Close)
	server.AuthManager().SetUserStore(database.NewUserStoreAdapter(db))

	token, err := server.AuthManager().GenerateAccessToken(t.Context(), "operator")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ts := httptest.NewUnstartedServer(server.Handler())
	ts.Listener = ln
	ts.Start()
	t.Cleanup(ts.Close)

	fixture := &profileRouterFixture{ts: ts, token: token}
	fixture.csrf = fixture.fetchCSRF(t)
	return fixture
}

// fetchCSRF mints a token the way the UI does, so mutating requests survive
// the CSRF middleware rather than being rejected before they route.
func (f *profileRouterFixture) fetchCSRF(t *testing.T) string {
	t.Helper()

	status, body := f.do(t, http.MethodGet, "/api/v1/auth/csrf", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /api/v1/auth/csrf = %d, want 200: %s", status, body)
	}

	var payload struct {
		CSRFToken string `json:"csrfToken"`
		Token     string `json:"token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode csrf response %s: %v", body, err)
	}
	if payload.CSRFToken != "" {
		return payload.CSRFToken
	}
	return payload.Token
}

// do issues one authenticated request and returns the status and body.
func (f *profileRouterFixture) do(t *testing.T, method, path string, body any) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(t.Context(), method, f.ts.URL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Content-Type", "application/json")
	if f.csrf != "" {
		req.Header.Set("X-Csrf-Token", f.csrf)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, payload
}

// workingProfileID returns a profile to operate on.
//
// A second profile is a Pro feature -- the free tier allows exactly one -- so
// the fixture adopts the profile the server creates on first run rather than
// making its own. That is also the shape a free-tier install is actually in,
// which is the one worth testing.
func (f *profileRouterFixture) workingProfileID(t *testing.T) string {
	t.Helper()

	status, body := f.do(t, http.MethodGet, "/api/v1/profiles", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /api/v1/profiles = %d, want 200: %s", status, body)
	}

	var listed struct {
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(body, &listed); err != nil {
		t.Fatalf("decode profile list %s: %v", body, err)
	}
	if len(listed.Profiles) > 0 {
		return listed.Profiles[0].ID
	}
	return f.createFirstProfile(t)
}

// createFirstProfile creates a profile on an installation that has none. The
// first is free at every tier.
func (f *profileRouterFixture) createFirstProfile(t *testing.T) string {
	t.Helper()

	status, body := f.do(t, http.MethodPost, "/api/v1/profiles", map[string]any{
		"name":        "router-probe",
		"description": profileTestDescription,
		"config":      json.RawMessage(`{"notes":"original"}`),
	})
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("POST /api/v1/profiles = %d, want 2xx: %s", status, body)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode created profile %s: %v", body, err)
	}
	if created.ID == "" {
		t.Fatalf("created profile has no id: %s", body)
	}
	return created.ID
}

// profileTestDescription is written by the test and asserted on afterwards, so
// a settings save that blanks it is visible.
const profileTestDescription = "written by the router test"

// routed reports whether a status means the request reached its handler. A 404
// or a 405 is what the prefix bug produced; anything else -- including a 402
// from the licence gate -- means the router did its job.
func routed(status int) bool {
	return status != http.StatusNotFound && status != http.StatusMethodNotAllowed
}

// TestProfileAPIRoutesThroughTheMux is the probe table from the 2026-09-02
// audit, executed. Every one of these returned 404 or 405 before the prefix
// fix (#2331 / S1-9).
//
// The subtests run in sequence rather than in parallel: switching must happen
// before there is an active profile to read, and each case operates on state
// the previous one left. Ordering is the point, so the parent does not
// parallelise.
func TestProfileAPIRoutesThroughTheMux(t *testing.T) {
	fixture := newProfileRouterFixture(t)
	id := fixture.workingProfileID(t)

	// Give the profile a description and a config to notice being clobbered.
	if status, body := fixture.do(t, http.MethodPut, "/api/v1/profiles/"+id, map[string]any{
		"name":        "router-probe",
		"description": profileTestDescription,
		"config":      json.RawMessage(`{"notes":"original"}`),
	}); status != http.StatusOK {
		t.Fatalf("seed the profile: PUT = %d: %s", status, body)
	}

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"list", http.MethodGet, "/api/v1/profiles", nil},
		{"get by id", http.MethodGet, "/api/v1/profiles/" + id, nil},
		{
			"update", http.MethodPut, "/api/v1/profiles/" + id,
			map[string]any{"name": "router-probe-renamed", "description": "updated"},
		},
		{
			"patch settings", http.MethodPatch, "/api/v1/profiles/" + id + "/settings",
			map[string]any{"theme": "dark"},
		},
		// switch runs before active so there is an active profile to read.
		// A fresh installation has neither an active nor a default profile,
		// and "no profile is active" is a correct 404 rather than a routing
		// failure -- so the order here is the order an operator works in.
		{"switch", http.MethodPost, "/api/v1/profiles/switch", map[string]any{"profileId": id}},
		{"active", http.MethodGet, "/api/v1/profiles/active", nil},
		{"export", http.MethodGet, "/api/v1/profiles/export", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := fixture.do(t, tc.method, tc.path, tc.body)
			if status < 200 || status > 299 {
				t.Errorf("%s %s = %d, want 2xx: %s", tc.method, tc.path, status, body)
			}
		})
	}
}

// create, duplicate and delete all reach a policy answer on a free-tier
// installation with one profile -- a second profile needs Pro, and the only
// profile is the active one and cannot be deleted. Those are decisions, not
// routing failures, so what is asserted here is that each one reached the
// handler that made them. Before the prefix fix, create and duplicate were 405
// and delete was 404: no policy was ever consulted.
// Sequential for the same reason as above: delete consumes the profile that
// duplicate reads.
func TestWriteOperationsReachTheirHandlers(t *testing.T) {
	fixture := newProfileRouterFixture(t)
	id := fixture.workingProfileID(t)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{
			"create", http.MethodPost, "/api/v1/profiles",
			map[string]any{"name": "second-profile", "description": "a second one"},
		},
		{"duplicate", http.MethodPost, "/api/v1/profiles/" + id + "/duplicate", nil},
		{"delete", http.MethodDelete, "/api/v1/profiles/" + id, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := fixture.do(t, tc.method, tc.path, tc.body)
			if !routed(status) {
				t.Errorf("%s %s = %d -- the router never reached a handler: %s",
					tc.method, tc.path, status, body)
			}
		})
	}
}

// The prefix bug's signature: a request for the collection was dispatched to
// the by-id handler with "api/v1/profiles" as the id. Asking for a profile
// that genuinely does not exist must 404, or the fix would be indistinguishable
// from the handler having stopped checking.
func TestUnknownProfileStill404s(t *testing.T) {
	t.Parallel()

	fixture := newProfileRouterFixture(t)

	status, _ := fixture.do(t, http.MethodGet, "/api/v1/profiles/no-such-profile", nil)
	if status != http.StatusNotFound {
		t.Errorf("GET unknown profile = %d, want 404", status)
	}
}

// PATCH must not blank the fields it was not given. catalog.Update assigns
// Description and IsDefault unconditionally, so a settings save that passed a
// bare update through would clear a profile's description every time an
// operator changed a setting.
func TestPatchSettingsPreservesTheRestOfTheProfile(t *testing.T) {
	t.Parallel()

	fixture := newProfileRouterFixture(t)
	id := fixture.workingProfileID(t)

	if status, body := fixture.do(t, http.MethodPut, "/api/v1/profiles/"+id, map[string]any{
		"name":        "settings-preserve",
		"description": profileTestDescription,
		"config":      json.RawMessage(`{"notes":"original"}`),
	}); status != http.StatusOK {
		t.Fatalf("seed the profile: PUT = %d: %s", status, body)
	}

	status, body := fixture.do(t, http.MethodPatch, "/api/v1/profiles/"+id+"/settings",
		map[string]any{"theme": "dark"})
	if status != http.StatusOK {
		t.Fatalf("PATCH settings = %d, want 200: %s", status, body)
	}

	var updated struct {
		Description string `json:"description"`
		Config      struct {
			Notes    string         `json:"notes"`
			Settings map[string]any `json:"settings"`
		} `json:"config"`
	}
	if err := json.Unmarshal(body, &updated); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}

	if updated.Description != profileTestDescription {
		t.Errorf("description = %q; a settings save cleared it", updated.Description)
	}
	if updated.Config.Notes != "original" {
		t.Errorf("config.notes = %q; a settings save overwrote the rest of the config", updated.Config.Notes)
	}
	if fmt.Sprint(updated.Config.Settings["theme"]) != "dark" {
		t.Errorf("config.settings = %v, want the patched theme", updated.Config.Settings)
	}
}
