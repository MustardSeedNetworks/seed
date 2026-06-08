// SPDX-License-Identifier: BUSL-1.1

package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

// TestCSRFExemptList_Golden pins the CSRF exempt-list (#1223). It is the
// regression gate: every currently-exempt path must stay exempt, and a
// representative set of data-mutating routes must stay PROTECTED. Changing
// isCSRFExemptPath without updating this table fails CI — forcing a reviewer to
// confirm a new exemption is a pre-session/non-state-changing endpoint, never a
// data-mutating route. Do NOT "fix" a failure by pasting the new behaviour in
// blind; confirm the exemption is safe first.
func TestCSRFExemptList_Golden(t *testing.T) {
	t.Parallel()
	exempt := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout",
		"/api/v1/setup/status",
		"/api/v1/setup/complete",
		"/api/v1/reporting/logs/client",
		"/api/v1/sso/callback", // prefix match
		"/api/v1/sso/",
	}
	for _, p := range exempt {
		if !auth.ExportIsCSRFExemptPath(p) {
			t.Errorf("expected %q to be CSRF-exempt, but it is NOT — did the exempt-list shrink?", p)
		}
	}

	// These mutating routes MUST require a CSRF token. If any becomes exempt,
	// that is a security regression — fix the route, not this test.
	protected := []string{
		"/api/v1/profiles",
		"/api/v1/config/import",
		"/api/v1/devices",
		"/api/v1/users",
		"/api/v1/auth/loginX",             // not an exact match
		"/api/v1/reporting/logs/client/x", // exact entry, not a prefix
		"/api/v1/sso",                     // prefix needs the trailing slash
	}
	for _, p := range protected {
		if auth.ExportIsCSRFExemptPath(p) {
			t.Errorf("SECURITY: %q is CSRF-exempt but must be protected (#1223)", p)
		}
	}
}

// TestCSRFMiddleware_BlocksProtectedMutation is the integration test (#1223): a
// state-changing request to a non-exempt /api route without a valid CSRF token
// is blocked (the wrapped handler never runs), while an exempt path and safe
// methods pass through. Without a session the middleware returns 401; with a
// session but a bad/absent token it returns 403 — both are "blocked", the
// property pinned here.
func TestCSRFMiddleware_BlocksProtectedMutation(t *testing.T) {
	t.Parallel()
	m := auth.NewCSRFManager()
	defer m.Stop()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := m.CSRFMiddleware(next)

	serve := func(method, path string) *httptest.ResponseRecorder {
		called = false
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec
	}

	// Protected mutating route, no session, no token → blocked.
	rec := serve(http.MethodPost, "/api/v1/profiles")
	if called {
		t.Error("CSRF middleware let an unprotected POST /api/v1/profiles reach the handler")
	}
	if rec.Code == http.StatusOK {
		t.Errorf("POST /api/v1/profiles without CSRF token: status = %d, want a block (401/403)", rec.Code)
	}

	// Exempt route passes through.
	if serve(http.MethodPost, "/api/v1/auth/login"); !called {
		t.Error("CSRF middleware blocked an exempt POST /api/v1/auth/login")
	}

	// Safe method on a protected route passes through (RFC 7231).
	if serve(http.MethodGet, "/api/v1/profiles"); !called {
		t.Error("CSRF middleware blocked a safe GET /api/v1/profiles")
	}
}
