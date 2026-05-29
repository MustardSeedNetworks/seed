// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/seed/internal/database"
)

// seedRoledUser adds a user with the given role to the test server's DB.
func seedRoledUser(t *testing.T, s *Server, username, role string) {
	t.Helper()
	if _, err := s.services.Database.DB.CreateUser(t.Context(), username, "$2a$10$x", role); err != nil {
		t.Fatalf("seed %s (%s): %v", username, role, err)
	}
}

// TestWriteGate_MethodAndRoleMatrix proves the central #1226 wrapper:
// reads pass through for every role; non-safe methods require operator+.
func TestWriteGate_MethodAndRoleMatrix(t *testing.T) {
	t.Parallel()
	s, _ := usersTestSetup(t) // seeds "admin" (RoleAdmin)
	seedRoledUser(t, s, "viewer1", database.RoleViewer)
	seedRoledUser(t, s, "operator1", database.RoleOperator)

	called := false
	probe := func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}
	gated := s.writeGated(probe)

	cases := []struct {
		user       string
		method     string
		wantStatus int
		wantCalled bool
	}{
		// Safe methods pass through for every role, including viewer.
		{"viewer1", http.MethodGet, http.StatusOK, true},
		{"viewer1", http.MethodHead, http.StatusOK, true},
		{"viewer1", http.MethodOptions, http.StatusOK, true},
		// Viewer is blocked on every write method.
		{"viewer1", http.MethodPost, http.StatusForbidden, false},
		{"viewer1", http.MethodPut, http.StatusForbidden, false},
		{"viewer1", http.MethodPatch, http.StatusForbidden, false},
		{"viewer1", http.MethodDelete, http.StatusForbidden, false},
		// Operator and admin clear the gate on writes.
		{"operator1", http.MethodPost, http.StatusOK, true},
		{"admin", http.MethodDelete, http.StatusOK, true},
	}
	for _, c := range cases {
		called = false
		req := newAuthedRequest(c.method, APIVersionPrefix+"/probe", nil, c.user)
		w := httptest.NewRecorder()
		gated(w, req)
		if w.Code != c.wantStatus {
			t.Errorf("%s %s: status = %d, want %d", c.user, c.method, w.Code, c.wantStatus)
		}
		if called != c.wantCalled {
			t.Errorf("%s %s: handler called = %v, want %v", c.user, c.method, called, c.wantCalled)
		}
	}

	// No caller -> 401, not 403.
	called = false
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/probe", nil, "")
	w := httptest.NewRecorder()
	gated(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no-caller POST: status = %d, want 401", w.Code)
	}
	if called {
		t.Error("no-caller POST: handler should not be called")
	}
}

// TestRequireRole_Hierarchy exercises the rank comparison directly so the
// viewer<operator<admin ordering can't silently regress.
func TestRequireRole_Hierarchy(t *testing.T) {
	t.Parallel()
	s, _ := usersTestSetup(t)
	seedRoledUser(t, s, "viewer1", database.RoleViewer)
	seedRoledUser(t, s, "operator1", database.RoleOperator)

	cases := []struct {
		user    string
		min     string
		allowed bool
	}{
		{"viewer1", database.RoleViewer, true},
		{"viewer1", database.RoleOperator, false},
		{"viewer1", database.RoleAdmin, false},
		{"operator1", database.RoleOperator, true},
		{"operator1", database.RoleAdmin, false},
		{"admin", database.RoleAdmin, true},
		{"admin", database.RoleOperator, true},
	}
	for _, c := range cases {
		req := newAuthedRequest(http.MethodGet, APIVersionPrefix+"/x", nil, c.user)
		w := httptest.NewRecorder()
		got := s.requireRole(w, req, c.min)
		if got != c.allowed {
			t.Errorf("requireRole(%s, min=%s) = %v, want %v (status %d)", c.user, c.min, got, c.allowed, w.Code)
		}
	}
}

// TestCallerRole_NoDBIsImplicitAdmin guards the single-user/env-mode path:
// with no user DB configured, the lone operator is treated as admin so the
// write gate never locks a single-user deployment out of its own tool.
func TestCallerRole_NoDBIsImplicitAdmin(t *testing.T) {
	t.Parallel()
	s := &Server{services: NewServiceContainer()} // no DB attached
	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/x", nil, "someone")
	role, ok := s.callerRole(req)
	if !ok || role != database.RoleAdmin {
		t.Fatalf("callerRole with no DB = (%q, %v), want (admin, true)", role, ok)
	}
	w := httptest.NewRecorder()
	if !s.requireWriteAccess(w, req) {
		t.Errorf("requireWriteAccess with no DB should allow; status=%d", w.Code)
	}
}

// TestWriteGate_WiredOnSettingsRoute proves the wrapper is actually
// attached at route registration: a viewer hitting POST /api/v1/settings
// is rejected with 403 by the gate, never reaching handleSettings. If
// somebody removes the s.writeGated(...) wrap at registration time, this
// test fails — guarding against a silent regression of the policy.
func TestWriteGate_WiredOnSettingsRoute(t *testing.T) {
	t.Parallel()
	s, _ := usersTestSetup(t)
	seedRoledUser(t, s, "viewer1", database.RoleViewer)
	s.setupRoutes()

	req := newAuthedRequest(http.MethodPost, APIVersionPrefix+"/settings", []byte(`{}`), "viewer1")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("viewer POST /settings via mux: status = %d, want 403 (gate must be wired at registration)", w.Code)
	}
}
