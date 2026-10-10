package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
)

// TestCSRFExemptionsArePinned names every authenticated route that takes a
// state-changing method without CSRF (#1223). A new exemption must be added
// here deliberately: it is a pre-session or non-state-changing endpoint, never
// a data-mutating route. Unauthenticated routes are the pre-session steps,
// which cannot carry a token (#2391).
func TestCSRFExemptionsArePinned(t *testing.T) {
	t.Parallel()
	s := newRoutePolicyServer(t)
	want := []string{
		APIVersionPrefix + "/auth/logout",           // idempotent session teardown
		APIVersionPrefix + "/reporting/logs/client", // the logger runs before a token exists
	}
	var got []string
	for _, p := range s.routes.Policies() {
		if p.Auth && !p.CSRF && takesStateChangingMethod(p) {
			got = append(got, p.Path)
		}
		if !p.Auth && p.CSRF {
			t.Errorf("%s declares CSRF without Auth; a pre-session route cannot carry a token", p.Path)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("authenticated state-changing routes without CSRF = %v, want %v", got, want)
	}
}

func takesStateChangingMethod(p route.Policy) bool {
	if len(p.Methods) == 0 {
		return true
	}
	return slices.ContainsFunc(p.Methods, func(m string) bool {
		return m != http.MethodGet && m != http.MethodHead && m != http.MethodOptions
	})
}

// TestRegistrarRefusalsUseSeedEnvelope pins the Registrar's own refusals to
// the codes and messages seed answered before it adopted the shared registrar.
func TestRegistrarRefusalsUseSeedEnvelope(t *testing.T) {
	t.Parallel()
	s := &Server{}
	s.withRouteDeps(t)
	reg := s.newRegistrar()
	reg.RegisterAll([]route.Route{
		{Path: "/panic", Handler: func(http.ResponseWriter, *http.Request) { panic("boom") }},
		{Path: "/get-only", Handler: func(http.ResponseWriter, *http.Request) {}, Methods: []string{http.MethodGet}},
		{Path: "/csrf", Handler: func(http.ResponseWriter, *http.Request) {}, CSRF: true},
	})
	handler := i18n.Middleware()(reg.Handler())

	jwt, mintErr := s.authManager().GenerateToken(t.Context(), "admin")
	if mintErr != nil {
		t.Fatal(mintErr)
	}
	tests := []struct {
		name, method, path string
		session            bool
		wantStatus         int
		wantCode, wantMsg  string
	}{
		{
			"recovered panic", http.MethodGet, "/panic", false, http.StatusInternalServerError,
			ErrCodeInternal, "Internal server error",
		},
		{
			"undeclared method", http.MethodPost, "/get-only", false, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, "Method not allowed",
		},
		{
			"missing CSRF token", http.MethodPost, "/csrf", true, http.StatusForbidden,
			ErrCodeForbidden, "CSRF token required",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			if tc.session {
				req.Header.Set("Authorization", "Bearer "+jwt)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			var body ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode %q: %v", rec.Body.String(), err)
			}
			if rec.Code != tc.wantStatus || body.Code != tc.wantCode || !strings.HasPrefix(body.Error, tc.wantMsg) {
				t.Errorf("%s %s = %d %s %q, want %d %s %q…", tc.method, tc.path,
					rec.Code, body.Code, body.Error, tc.wantStatus, tc.wantCode, tc.wantMsg)
			}
		})
	}
}

// TestPreSessionRoutesAreUnauthenticated pins the /api routes that run before
// the caller holds a session — sign-in and its second factor, refresh,
// first-run setup, recovery and the OAuth handshake (#478, #2391). Every other
// /api route declares Auth.
func TestPreSessionRoutesAreUnauthenticated(t *testing.T) {
	t.Parallel()
	s := newRoutePolicyServer(t)

	want := []string{
		APIVersionPrefix + "/auth/login",
		APIVersionPrefix + "/auth/refresh",
		APIVersionPrefix + "/auth/login/totp",
		APIVersionPrefix + "/auth/webauthn/login/begin",
		APIVersionPrefix + "/auth/webauthn/login/finish",
		APIVersionPrefix + "/setup/status",
		APIVersionPrefix + "/setup/complete",
		APIVersionPrefix + "/recovery/status",
		APIVersionPrefix + "/recovery/complete",
		APIVersionPrefix + "/recovery/instructions",
		APIVersionPrefix + "/sso/providers",
		APIVersionPrefix + "/sso/login",
		APIVersionPrefix + "/sso/callback",
	}
	var got []string
	for _, p := range s.routes.Policies() {
		if p.Auth || !strings.HasPrefix(p.Path, "/api/") {
			continue
		}
		got = append(got, p.Path)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("unauthenticated /api routes = %v, want %v", got, want)
	}
}

// TestCSRFSessionKey: a resolved PAT and a request without a session pass
// CSRF (nothing ambient to forge); a session JWT is keyed by its hash.
func TestCSRFSessionKey(t *testing.T) {
	t.Parallel()
	jwtBearer := httptest.NewRequest(http.MethodPost, "/api/v1/x", http.NoBody)
	jwtBearer.Header.Set("Authorization", "Bearer header.payload.signature")
	pat := httptest.NewRequest(http.MethodPost, "/api/v1/x", http.NoBody)
	pat.Header.Set("Authorization", "Bearer header.payload.signature")
	pat = pat.WithContext(auth.WithAPITokenAuth(pat.Context()))

	if key, ok := csrfSessionKey(jwtBearer); !ok || key == "" {
		t.Errorf("session JWT: key=%q ok=%t, want a key", key, ok)
	}
	if key, ok := csrfSessionKey(pat); ok || key != "" {
		t.Errorf("resolved PAT: key=%q ok=%t, want no session", key, ok)
	}
	if key, ok := csrfSessionKey(httptest.NewRequest(http.MethodPost, "/api/v1/x", http.NoBody)); ok || key != "" {
		t.Errorf("no credential: key=%q ok=%t, want no session", key, ok)
	}
}

// TestRegistrarErrorCodes covers the refusals the routed table above cannot
// reach without an expired or forged token.
func TestRegistrarErrorCodes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		code       string
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{"csrf_token_expired", http.StatusForbidden, ErrCodeForbidden, "CSRF token expired"},
		{"csrf_token_invalid", http.StatusForbidden, ErrCodeForbidden, "Invalid CSRF token"},
		{"csrf_unavailable", http.StatusInternalServerError, ErrCodeInternal, "Internal server error"},
	}
	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			handler := i18n.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				registrarError(w, r, http.StatusForbidden, tc.code, "foundation's message")
			}))
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/csrf", http.NoBody))
			var body ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tc.wantStatus || body.Code != tc.wantCode || body.Error != tc.wantMsg {
				t.Errorf("%s = %d %s %q, want %d %s %q", tc.code,
					rec.Code, body.Code, body.Error, tc.wantStatus, tc.wantCode, tc.wantMsg)
			}
		})
	}
}

// TestScopeGateRefusesAnUnknownScope: a route declaring a scope the registry
// cannot enforce stops the daemon at registration instead of serving
// unprotected.
func TestScopeGateRefusesAnUnknownScope(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Error("scopeGate(\"admin\") did not panic")
		}
	}()
	(&Server{}).scopeGate("admin")
}
