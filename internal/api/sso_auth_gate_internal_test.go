// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// ssoGateServer builds a routed Server holding one viewer, one admin and a
// provider for handleSSOUpdate to find, so a refusal can be told apart from a
// 404 "provider not found".
func ssoGateServer(t *testing.T) *Server {
	t.Helper()

	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "sso.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, u := range []struct{ name, role string }{
		{"viewer", database.RoleViewer},
		{"admin", database.RoleAdmin},
	} {
		if _, createErr := db.CreateUser(t.Context(), u.name, "$2a$10$x", u.role); createErr != nil {
			t.Fatalf("seed %s: %v", u.name, createErr)
		}
	}

	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	cfg.Auth.SSO.Providers = []config.SSOProviderConfig{
		{Name: "google", Enabled: false, ClientID: "original-client-id"},
	}
	netMgr := netif.NewMockManager(netif.DefaultMockConfig())
	s := NewServer(cfg, filepath.Join(dir, "seed.json"), "", netMgr, false, nil, db, nil)
	t.Cleanup(s.Close)

	// /sso/update is Pro-gated as well as operator-gated, and ADR-0002 composes
	// requireFeature(writeGated(h)): on Free the feature gate answers 402 before
	// the role gate is reached, so a Free fixture cannot tell a working role gate
	// from a broken one. A trial licenses the feature and leaves the role gate as
	// the only thing under test.
	mgr, mgrErr := license.NewManagerWithDir(t.TempDir())
	if mgrErr != nil {
		t.Fatalf("license manager: %v", mgrErr)
	}
	s.licenseMgr = mgr
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}

	return s
}

// authedRequest builds a request carrying a real access token for username and
// the CSRF token that token's session is keyed on, so the request reaches the
// role gate instead of stopping at the CSRF wall.
func authedRequest(t *testing.T, s *Server, method, path, username, body string) *http.Request {
	t.Helper()

	token, err := s.authManager().GenerateAccessToken(t.Context(), username)
	if err != nil {
		t.Fatalf("mint access token for %s: %v", username, err)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	csrfToken, err := s.csrfManager().TokenForSession(auth.GetSessionIDFromRequest(req))
	if err != nil {
		t.Fatalf("mint CSRF token: %v", err)
	}
	req.Header.Set(auth.CSRFHeaderName, csrfToken)

	return req
}

// TestSSOUpdateRefusesSpoofedUsernameHeader pins #2632: /api/v1/sso/update and
// /sso/settings sat under the `/api/v1/sso/` auth-bypass prefix, so the JWT
// middleware never overwrote X-Username and callerRole resolved whatever the
// client sent. A viewer could rewrite the OAuth provider config by naming an
// admin in a header.
func TestSSOUpdateRefusesSpoofedUsernameHeader(t *testing.T) {
	s := ssoGateServer(t)

	body := `{"provider":"google","enabled":true,"clientId":"attacker-client-id",` +
		`"clientSecret":"attacker-secret","redirectUrl":"https://attacker.example/cb"}`
	req := authedRequest(t, s, http.MethodPut, "/api/v1/sso/update", "viewer", body)
	req.Header.Set("X-Username", "admin")

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("PUT /api/v1/sso/update as viewer spoofing X-Username: admin = %d, want %d",
			w.Code, http.StatusForbidden)
	}
	if got := s.config.Auth.SSO.Providers[0].ClientID; got != "original-client-id" {
		t.Errorf("provider client ID = %q, want it unchanged: the refused request rewrote the config", got)
	}
}

// TestSSOSettingsRequiresAuthentication is the read half of #2632. A viewer
// reading it is deliberate (#1254: minRole gates writes, every authenticated
// caller may read), so the thing the bypass cost was authentication itself:
// under the old `/api/v1/sso/` prefix the JWT middleware never ran, and only a
// hand-rolled token check inside the handler stood between an anonymous caller
// and the provider list. That check is gone now, so the middleware must be the
// one refusing.
func TestSSOSettingsRequiresAuthentication(t *testing.T) {
	s := ssoGateServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sso/settings", nil)
	req.Header.Set("X-Username", "admin")

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated GET /api/v1/sso/settings naming an admin in a header = %d, want %d",
			w.Code, http.StatusUnauthorized)
	}
}

// TestSSOHandshakeStaysAnonymous guards the other side of the narrowing: the
// three OAuth handshake routes are reached by a browser that has no session
// yet, so they must keep bypassing the JWT middleware. A 401 here would make
// signing in with SSO impossible.
func TestSSOHandshakeStaysAnonymous(t *testing.T) {
	s := ssoGateServer(t)

	for _, path := range []string{
		"/api/v1/sso/providers",
		"/api/v1/sso/login",
		"/api/v1/sso/callback",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, req)
		if w.Code == http.StatusUnauthorized {
			t.Errorf("GET %s answered 401: the handshake needs to run before a session exists", path)
		}
	}
}

// TestNoGatedRouteBypassesAuth is the general form of #2632: a route whose
// policy is a role gate or a licence-feature gate must pass through the JWT
// middleware, because both gates resolve the caller from an identity only that
// middleware establishes. A gated route sitting on an auth-bypass path reads
// whatever the anonymous caller supplied.
//
// This is the invariant the scripts/check-route-policy.sh gate runs; keeping it
// as a Go test over the real route table beats grepping route literals, which
// cannot see a path built from a constant.
func TestNoGatedRouteBypassesAuth(t *testing.T) {
	t.Parallel()

	s := ssoGateServer(t)

	checked := 0
	for _, rt := range s.manifest {
		if rt.minRole == "" && rt.feature == "" {
			continue
		}
		checked++
		path := rt.path
		if method, ok := methodFromPath(path); ok {
			path = strings.TrimPrefix(path, method+" ")
		}
		if auth.ShouldBypassAuth(path) {
			t.Errorf("%s carries minRole=%q feature=%q but bypasses the auth middleware, "+
				"so its gate resolves an identity the caller supplied (#2632)",
				rt.path, rt.minRole, rt.feature)
		}
	}

	if checked == 0 {
		t.Fatal("no gated routes were examined; the route-table lookup is wrong")
	}
}
