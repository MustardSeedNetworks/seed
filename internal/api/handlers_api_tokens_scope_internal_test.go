// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// TestCallerRole_ClampsOnTokenScope proves the #1255 auth-time clamp:
// X-Token-Scope below the owner's role caps the effective role; at or
// above the owner's role is a no-op (no escalation possible); invalid
// scope values are ignored rather than locking the token out.
func TestCallerRole_ClampsOnTokenScope(t *testing.T) {
	t.Parallel()
	s, _ := usersTestSetup(t) // seeds "admin"
	seedRoledUser(t, s, "operator1", database.RoleOperator)

	cases := []struct {
		owner     string
		scope     string
		wantRole  string
		wantOK    bool
		wantClamp bool
	}{
		// Admin owner gets clamped down by viewer-scoped token.
		{"admin", "viewer", database.RoleViewer, true, true},
		{"admin", "operator", database.RoleOperator, true, true},
		// No scope = inherit owner role; admin stays admin.
		{"admin", "", database.RoleAdmin, true, false},
		// Scope at owner's level is a no-op.
		{"admin", "admin", database.RoleAdmin, true, false},
		// Operator owner can be clamped down to viewer.
		{"operator1", "viewer", database.RoleViewer, true, true},
		// Scope above owner's role can't escalate — owner's role wins.
		{"operator1", "admin", database.RoleOperator, true, false},
		// Invalid scope is ignored, owner's role applies.
		{"admin", "superuser", database.RoleAdmin, true, false},
	}
	for _, c := range cases {
		req := newAuthedRequest(http.MethodGet, APIVersionPrefix+"/x", nil, c.owner)
		if c.scope != "" {
			req.Header.Set("X-Token-Scope", c.scope)
		}
		role, ok := s.callerRole(req)
		if ok != c.wantOK || role != c.wantRole {
			t.Errorf("callerRole(owner=%s,scope=%q) = (%q,%v), want (%q,%v)",
				c.owner, c.scope, role, ok, c.wantRole, c.wantOK)
		}
	}
}

// TestWriteGate_ViewerScopedAdminTokenBlocksWrites proves end-to-end
// that an admin-owned PAT with scope=viewer is rejected by the write
// gate the same way a real viewer would be — the central #1226 gate
// reads the clamped role from callerRole, no special case needed.
func TestWriteGate_ViewerScopedAdminTokenBlocksWrites(t *testing.T) {
	t.Parallel()
	s, _ := usersTestSetup(t)

	// As "admin", the write gate would let the request through.
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/probe", nil, "admin")
	w := httptest.NewRecorder()
	if !s.requireWriteAccess(w, req) {
		t.Fatalf("admin without scope must clear gate, got %d", w.Code)
	}

	// Same admin caller, but the PAT clamped them to viewer.
	req = newAuthedRequest(http.MethodPost, APIVersionPrefix+"/probe", nil, "admin")
	req.Header.Set("X-Token-Scope", database.RoleViewer)
	w = httptest.NewRecorder()
	if s.requireWriteAccess(w, req) {
		t.Errorf("admin-owned viewer-scoped token must be blocked by write gate, status=%d", w.Code)
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// TestAPITokenRepo_ScopeRoundtrips proves the schema + Insert/Find/List
// path persists and reads back the per-token scope.
func TestAPITokenRepo_ScopeRoundtrips(t *testing.T) {
	t.Parallel()
	s, _ := usersTestSetup(t)
	db := s.services.Database.DB
	repo := database.NewAPITokenRepository(db)

	// Seed two tokens for "admin": one viewer-scoped, one inherits.
	rec1 := database.APITokenRecord{
		ID: "t1", OwnerUsername: "admin", Name: "ci-readonly",
		TokenHash: "hash1", Prefix: "sd_pat_abcde", Scope: database.RoleViewer,
	}
	rec2 := database.APITokenRecord{
		ID: "t2", OwnerUsername: "admin", Name: "all-access",
		TokenHash: "hash2", Prefix: "sd_pat_fghij", Scope: "",
	}
	if err := repo.Insert(t.Context(), rec1); err != nil {
		t.Fatalf("insert t1: %v", err)
	}
	if err := repo.Insert(t.Context(), rec2); err != nil {
		t.Fatalf("insert t2: %v", err)
	}

	got1, err := repo.FindActiveByHash(t.Context(), "hash1")
	if err != nil {
		t.Fatalf("find t1: %v", err)
	}
	if got1.Scope != database.RoleViewer {
		t.Errorf("scope round-trip: got %q, want %q", got1.Scope, database.RoleViewer)
	}

	got2, err := repo.FindActiveByHash(t.Context(), "hash2")
	if err != nil {
		t.Fatalf("find t2: %v", err)
	}
	if got2.Scope != "" {
		t.Errorf("empty-scope round-trip: got %q, want empty (NULL inherits)", got2.Scope)
	}

	// List preserves Scope on every row.
	list, err := repo.ListByOwner(t.Context(), "admin")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	scopes := map[string]string{}
	for _, r := range list {
		scopes[r.ID] = r.Scope
	}
	if scopes["t1"] != database.RoleViewer || scopes["t2"] != "" {
		t.Errorf("list scopes: got %v, want t1=viewer t2=''", scopes)
	}
}

// attachAPITokenRepo wires an APITokenRepository onto the test server's
// service container so the mint/list/revoke handlers can run.
func attachAPITokenRepo(t *testing.T, s *Server) {
	t.Helper()
	s.services.Auth.APITokens = database.NewAPITokenRepository(s.services.Database.DB)
}

// TestMintAPIToken_RejectsScopeAboveOwner proves the mint handler
// refuses to issue a token with a scope higher than the minter's role
// — that would be a one-line escalation if accepted.
func TestMintAPIToken_RejectsScopeAboveOwner(t *testing.T) {
	t.Parallel()
	s, mgr := usersTestSetup(t)
	attachAPITokenRepo(t, s)
	if r := mgr.StartTrial(); !r.Success { // mint requires Pro
		t.Fatalf("StartTrial: %s", r.Message)
	}
	seedRoledUser(t, s, "operator1", database.RoleOperator)
	// Operator tries to mint an admin-scoped token.
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/tokens",
		[]byte(`{"name":"escalate","scope":"admin"}`), "operator1")
	w := httptest.NewRecorder()
	s.handleAPITokens(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

// TestMintAPIToken_RejectsUnknownScope keeps the error from the schema
// CHECK constraint from being the user's first signal — the handler
// returns a clearer 400 up front.
func TestMintAPIToken_RejectsUnknownScope(t *testing.T) {
	t.Parallel()
	s, mgr := usersTestSetup(t)
	attachAPITokenRepo(t, s)
	if r := mgr.StartTrial(); !r.Success {
		t.Fatalf("StartTrial: %s", r.Message)
	}
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/tokens",
		[]byte(`{"name":"weird","scope":"superuser"}`), "admin")
	w := httptest.NewRecorder()
	s.handleAPITokens(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}
