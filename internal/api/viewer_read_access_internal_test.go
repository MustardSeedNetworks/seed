// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// routedServer builds a Server with the real route table and one seeded user.
//
// usersTestSetup builds a bare Server with no routes registered, so it cannot
// answer a question about the table itself; NewServer installs it.
func routedServer(t *testing.T, username, role string) *Server {
	t.Helper()

	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "routes.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, createErr := db.CreateUser(t.Context(), username, "$2a$10$x", role); createErr != nil {
		t.Fatalf("seed %s: %v", username, createErr)
	}

	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	netMgr := netif.NewMockManager(netif.DefaultMockConfig())
	s := NewServer(cfg, filepath.Join(dir, "seed.json"), "", netMgr, false, nil, db, nil)
	t.Cleanup(s.Close)

	return s
}

// probeBudget bounds one route probe. A streaming route never returns on its
// own: an SSE handler writes its first frame and then parks until the client
// goes away, and a [httptest.ResponseRecorder] is neither a client that can go
// away nor a writer whose Write ever fails. Without a deadline the sweep hangs
// on /jobs/events and /discovery/engine/events forever.
//
// It is only ever a floor. The 405 this helper looks for comes from methodGate,
// before any handler work, so a short budget cannot turn a real 405 into a
// miss; it can only cut short a route that was already answering something
// else, which is the answer the caller wanted anyway.
const probeBudget = 2 * time.Second

// answersOnlyMethodNotAllowed reports whether every method the route declares
// comes back 405 — the signature of a route and its handler disagreeing.
func answersOnlyMethodNotAllowed(t *testing.T, s *Server, path string, methods []string) bool {
	t.Helper()

	for _, method := range methods {
		// The deadline is what lets a streaming route be swept at all. Until
		// #2553 every SSE route answered 500 before it streamed — the logging
		// middleware's wrapper was not an http.Flusher — so this sweep passed
		// over them in microseconds and never noticed it could not handle one.
		// Fixing the wrapper made them stream, and the sweep hung.
		ctx, cancel := context.WithTimeout(t.Context(), probeBudget)
		req := bearerRequest(t, s, method, path, "admin").WithContext(ctx)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, req)
		cancel()
		if w.Code == http.StatusUnauthorized {
			t.Fatalf("%s %s answered 401, so the request never reached the method "+
				"gate and this assertion proves nothing", method, path)
		}
		if w.Code != http.StatusMethodNotAllowed {
			return false
		}
	}

	return true
}

// bearerRequest builds a request carrying a real access token for username.
//
// newAuthedRequest sets X-Username, which is the contract between the auth
// middleware and the handlers below it — fine when a test calls a handler
// directly, useless through s.Handler(), where the real middleware runs, finds
// no token and answers 401. A test that drives the full chain with X-Username
// alone asserts nothing: every response is 401, and 401 is neither the 403 nor
// the 405 such a test is looking for.
func bearerRequest(t *testing.T, s *Server, method, path, username string) *http.Request {
	t.Helper()

	token, err := s.authManager().GenerateAccessToken(t.Context(), username)
	if err != nil {
		t.Fatalf("mint access token for %s: %v", username, err)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return req
}

// TestViewerCanReadEveryRoleGatedRoute walks the real route table and drives a
// GET as a viewer against every route that carries minRole, asserting none of
// them is refused on the read.
//
// #1254 was filed on the reading that "42 routes carry minRole: op, 29 of them
// gate GET, so a viewer's Settings page mostly fails to load". That inference
// does not hold: minRole is applied through writeGated, which passes GET, HEAD
// and OPTIONS for every role and gates only the mutating methods. Counting
// routes that both carry minRole and serve GET therefore counts routes whose
// reads were never gated.
//
// This is the check that inference needed. If a future route ever gates a read
// behind operator, this fails and names it, rather than the symptom surfacing as
// an unexplained blank page.
func TestViewerCanReadEveryRoleGatedRoute(t *testing.T) {
	t.Parallel()

	s := routedServer(t, "viewer1", database.RoleViewer)

	// Counted before the subtests run, since t.Parallel defers them.
	checked := 0
	for _, rt := range s.manifest {
		if rt.minRole == "" {
			continue
		}
		// Only routes that actually serve a read.
		if len(rt.methods) > 0 && !slices.Contains(rt.methods, http.MethodGet) {
			continue
		}

		path := rt.path
		// Collection routes ending in "/" take an id; any value exercises the gate.
		if strings.HasSuffix(path, "/") {
			path += "probe"
		}

		checked++
		t.Run(rt.path, func(t *testing.T) {
			t.Parallel()
			req := bearerRequest(t, s, http.MethodGet, path, "viewer1")
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)

			if w.Code == http.StatusUnauthorized {
				t.Fatalf("GET %s answered 401, so the request never reached the role "+
					"gate and this assertion proves nothing", path)
			}
			if w.Code == http.StatusForbidden {
				t.Errorf("GET %s as a viewer = 403; a read behind minRole=%q is not "+
					"reachable, so the page that needs it renders empty: %s",
					path, rt.minRole, strings.TrimSpace(w.Body.String()))
			}
		})
	}

	if checked == 0 {
		t.Fatal("no role-gated read routes were exercised; the route table lookup is wrong")
	}
}

// isPreSessionPath reports whether a path is reachable before the caller holds
// an access token, where a 401 is a legitimate handler answer.
//
// #2632: the OAuth entries are the three handshake paths, not a `/api/v1/sso/`
// prefix. The prefix made this helper tolerate a 401 on /sso/settings and
// /sso/update — the same over-broad reading that let their role gate trust a
// client header, surviving in the suite's own definition of "pre-session".
func isPreSessionPath(path string) bool {
	if slices.Contains([]string{
		"/api/v1/sso/providers", "/api/v1/sso/login", "/api/v1/sso/callback",
	}, path) {
		return true
	}
	for _, prefix := range []string{
		"/api/v1/auth/", "/api/v1/setup/", "/api/v1/recovery/",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// TestRouteMethodsMatchTheirHandlers catches the mismatch that made
// /telemetry/iperf/server dead on arrival: the route declared `get`, the handler
// answered 405 to anything but POST, so the methodGate and the handler refused
// opposite halves and no method worked at all, for any role.
//
// The route-consumer gate could not see this — it matches paths, not methods.
// This drives every declared method of every route that names one and fails on a
// route where they all 405, which is the signature of the two disagreeing.
func TestRouteMethodsMatchTheirHandlers(t *testing.T) {
	t.Parallel()

	s := routedServer(t, "admin", database.RoleAdmin)

	for _, rt := range s.manifest {
		if len(rt.methods) == 0 {
			continue
		}
		// Pre-session routes answer 401 on their own terms — an empty body is
		// not a credential — so a 401 there is the handler working, not the
		// chain refusing. They are covered by the auth tests instead.
		if isPreSessionPath(rt.path) {
			continue
		}
		path := rt.path
		if strings.HasSuffix(path, "/") {
			path += "probe"
		}

		t.Run(rt.path, func(t *testing.T) {
			t.Parallel()
			if answersOnlyMethodNotAllowed(t, s, path, rt.methods) {
				t.Errorf("%s answers 405 to every method it declares (%v) — the route "+
					"and its handler disagree, so the endpoint is unreachable",
					rt.path, rt.methods)
			}
		})
	}
}
