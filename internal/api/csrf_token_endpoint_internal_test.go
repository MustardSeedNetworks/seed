// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// csrfEndpointServer builds a routed Server holding one admin, so a mutating
// write reaches the CSRF wall rather than stopping at the role gate.
func csrfEndpointServer(t *testing.T) *Server {
	t.Helper()

	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "csrf.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, createErr := db.CreateUser(t.Context(), "admin", "$2a$10$x", database.RoleAdmin); createErr != nil {
		t.Fatalf("seed admin: %v", createErr)
	}

	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	netMgr := netif.NewMockManager(netif.DefaultMockConfig())
	s := NewServer(cfg, filepath.Join(dir, "seed.json"), "", netMgr, false, nil, db, nil)
	t.Cleanup(s.Close)

	return s
}

// fetchCSRFToken drives GET /auth/csrf through the real chain with the given
// bearer and returns the token the endpoint answered with.
func fetchCSRFToken(t *testing.T, s *Server, bearer string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/auth/csrf", nil)
	req.Header.Set("Authorization", "Bearer "+bearer)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /auth/csrf = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode CSRF response %q: %v", w.Body.String(), err)
	}
	token := resp["csrfToken"]
	if token == "" {
		token = resp["token"]
	}
	if token == "" {
		t.Fatalf("CSRF response carried no token: %s", w.Body.String())
	}

	return token
}

// TestCSRFTokenEndpointDoesNotInvalidateTheFirstTab pins #2660. The endpoint
// minted with foundation's Generate, which *replaces* the session's stored
// token. Both tabs of one session share a bearer, so they share a session key:
// the second tab's fetch silently invalidated the first tab's token and the
// operator's next write there answered 403. foundation exposes GetOrCreate for
// exactly this — a token endpoint hands back the session's live token instead
// of rotating it on every read.
func TestCSRFTokenEndpointDoesNotInvalidateTheFirstTab(t *testing.T) {
	s := csrfEndpointServer(t)

	bearer, err := s.authManager().GenerateAccessToken(t.Context(), "admin")
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}

	// Two tabs of one session, in the order a browser opens them.
	firstTab := fetchCSRFToken(t, s, bearer)
	secondTab := fetchCSRFToken(t, s, bearer)

	if firstTab != secondTab {
		t.Errorf("the two tabs of one session hold different CSRF tokens:\n first  = %q\n second = %q\n"+
			"the endpoint rotated the session's token instead of returning the live one", firstTab, secondTab)
	}

	// The operator now writes from the FIRST tab.
	body := `{"speedtest":{"serverId":"seed-csrf-2660"}}`
	req := httptest.NewRequest(http.MethodPut, APIVersionPrefix+"/settings", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(auth.CSRFHeaderName, firstTab)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code == http.StatusForbidden {
		t.Fatalf("PUT /settings from the first tab = 403 after a second tab fetched a token: %s", w.Body.String())
	}
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /settings from the first tab = %d, want 200: %s", w.Code, w.Body.String())
	}
}

// TestCSRFTokenEndpointStillMintsForAFreshSession pins the other half: a
// session that has never asked still gets a token, and two different sessions
// never share one.
func TestCSRFTokenEndpointStillMintsForAFreshSession(t *testing.T) {
	s := csrfEndpointServer(t)

	first, err := s.authManager().GenerateAccessToken(t.Context(), "admin")
	if err != nil {
		t.Fatalf("mint first access token: %v", err)
	}
	// A refresh yields a new bearer, which is a new session key. Every minted
	// token carries a random jti (newTokenID), so the two differ even for the
	// same user in the same second.
	second, err := s.authManager().GenerateAccessToken(t.Context(), "admin")
	if err != nil {
		t.Fatalf("mint second access token: %v", err)
	}
	if first == second {
		t.Fatalf("two mints returned the same bearer; the jti is not unique")
	}

	a := fetchCSRFToken(t, s, first)
	b := fetchCSRFToken(t, s, second)
	if a == b {
		t.Errorf("two sessions were handed the same CSRF token %q", a)
	}
}
