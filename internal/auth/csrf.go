package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// CSRF token configuration.
const (
	// CSRFHeaderName is the HTTP header name for CSRF tokens.
	CSRFHeaderName = "X-Csrf-Token"
	// CSRFCookieName is the cookie name for CSRF tokens.
	CSRFCookieName = "csrf_token"
)

// CSRFManager backs Seed's CSRF protection with the fleet-shared foundation
// per-session token manager (github.com/MustardSeedNetworks/foundation/pkg/csrf).
// Token storage, generation, constant-time validation, expiry sweep and the
// sha256(session-key) model all live in foundation now; Seed keeps only its
// product-specific policy — the curated exempt-list, the response format, and
// the JWT-derived session key — in CSRFMiddleware and GetSessionIDFromRequest
// below.
type CSRFManager struct {
	mgr *csrf.Manager
}

// NewCSRFManager creates a CSRF manager backed by foundation's per-session
// token manager, whose cleanup goroutine is stopped via Stop() on shutdown.
func NewCSRFManager() *CSRFManager {
	return &CSRFManager{mgr: csrf.NewManager()}
}

// TokenForSession returns the session's live CSRF token, minting one only when
// the session has none or its token has expired. The session key is derived
// from the caller's JWT via GetSessionIDFromRequest, so every tab of one login
// shares it: minting a *fresh* token here (foundation's Generate) replaced the
// stored one, so a second tab's fetch silently invalidated the first tab's and
// the operator's next write there answered 403 (#2660). foundation exposes
// GetOrCreate for exactly this hazard.
//
// This is not a weakening. The token is still per-session, unguessable and
// expiring; returning the live one is what makes the double-submit cookie
// pattern work across tabs. Ending a session still drops its token via
// RevokeToken, and a new session is a new key.
func (m *CSRFManager) TokenForSession(sessionID string) (string, error) {
	return m.mgr.GetOrCreate(sessionID)
}

// ValidateToken checks the token against the one stored for sessionID. It
// returns one of csrf.ErrTokenMissing / ErrTokenInvalid / ErrTokenExpired so
// CSRFMiddleware can render a distinct response per cause.
func (m *CSRFManager) ValidateToken(sessionID, token string) error {
	return m.mgr.Validate(sessionID, token)
}

// RevokeToken drops a session's CSRF token, e.g. on logout or session rotation.
func (m *CSRFManager) RevokeToken(sessionID string) {
	m.mgr.Revoke(sessionID)
}

// Stop shuts down the manager's background cleanup goroutine.
func (m *CSRFManager) Stop() {
	m.mgr.Stop()
}

// isCSRFExemptPath reports whether path is on the curated CSRF exempt-list:
// pre-session auth/setup endpoints, the client-log sink, and the OAuth handshake.
// Every entry MUST be safe to serve on a state-changing method without a CSRF
// token — i.e. pre-session or non-security-state-changing — NEVER a data-mutating
// route. csrf_coverage_test.go pins this set (#1223): a change here must be
// reflected and justified there. Kept as a switch (no package globals) per the
// repo's gochecknoglobals convention; the switch is also allocation-free on the
// per-request hot path.
func isCSRFExemptPath(path string) bool {
	switch path {
	case "/api/v1/auth/login", // pre-session: no CSRF token issued yet
		"/api/v1/auth/refresh",          // access token may be expired → no valid token yet
		"/api/v1/auth/logout",           // safe/idempotent session teardown
		"/api/v1/setup/status",          // first-run, pre-auth
		"/api/v1/setup/complete",        // first-run, pre-auth
		"/api/v1/reporting/logs/client", // logger runs before CSRF tokens exist

		// The second half of a login, and the recovery that rescues one. All
		// three already bypass the JWT middleware (ShouldBypassAuth) because
		// they run before the user holds an access token — so requiring a CSRF
		// token here is not protection, it is a wall: the middleware finds no
		// session, answers 401, and the login can never be completed. Enrolling
		// TOTP locked the account permanently, and recovery failed the same way
		// (#2391).
		//
		// Exempting them is not a loosening. CSRF defends against riding an
		// authenticated browser's cookie, and there is no session to ride; the
		// mfaToken or recovery token in the body IS the credential, and an
		// attacker who has it does not need the victim's browser. That is the
		// same reasoning that already exempts login, refresh and setup.
		// The recovery reads are GETs, which CSRF skips by method anyway.
		// They are listed so the rule stays "everything pre-session is exempt":
		// if one of them ever accepts a POST, it works instead of silently
		// answering 401.
		"/api/v1/auth/login/totp",
		"/api/v1/auth/webauthn/login/begin",
		"/api/v1/auth/webauthn/login/finish",
		"/api/v1/recovery/status",
		"/api/v1/recovery/complete",
		"/api/v1/recovery/instructions",

		// The OAuth handshake, and only the handshake: the browser arrives at
		// these with no bearer, so they bypass the JWT middleware and must be
		// exempt here for the same reason the login half is. They are GETs,
		// which CSRF skips by method anyway; listing them keeps the rule
		// "everything pre-session is exempt" derivable (TestPreSessionPathsAreCSRFExempt).
		// #2632: this was a `/api/v1/sso/` prefix, which also exempted the
		// operator-gated /sso/settings and /sso/update.
		"/api/v1/sso/providers",
		"/api/v1/sso/login",
		"/api/v1/sso/callback":
		return true
	}
	return false
}

// CSRFMiddleware returns HTTP middleware that validates CSRF tokens on state-changing requests.
// It exempts GET, HEAD, OPTIONS, and TRACE methods as they should be safe/idempotent.
func (m *CSRFManager) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for safe methods (RFC 7231)
		if r.Method == http.MethodGet ||
			r.Method == http.MethodHead ||
			r.Method == http.MethodOptions ||
			r.Method == http.MethodTrace {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF for non-API routes (static files, etc.)
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF for the curated exempt-list (pre-session auth/setup + client
		// logging + SSO handshake). See isCSRFExemptPath for the per-entry
		// justification; the set is pinned by csrf_coverage_test.go (#1223) so a
		// new exemption can't slip in unreviewed.
		if isCSRFExemptPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// #2450: a personal access token is not an ambient credential. A
		// cross-site page cannot set an Authorization header, so a
		// PAT-authenticated request carries no CSRF risk — and it has no
		// session to key a token on, so requiring one made every mutating
		// PAT request 401. The marker is set by the PAT middleware after the
		// store resolved the token, never derived from the bearer's shape.
		if IsAPITokenAuth(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}

		// Derive the per-session key from the request's authenticated JWT.
		sessionID := GetSessionIDFromRequest(r)

		if sessionID == "" {
			logging.GetLogger().WarnContext(r.Context(), "CSRF validation failed: no session ID",
				"path", r.URL.Path,
				"method", r.Method)
			sendAuthError(w, http.StatusUnauthorized, errCodeUnauthorized, "Unauthorized")
			return
		}

		// Get CSRF token from request header
		token := r.Header.Get(CSRFHeaderName)

		// Validate the token
		if err := m.ValidateToken(sessionID, token); err != nil {
			logging.GetLogger().WarnContext(r.Context(), "CSRF validation failed",
				"path", r.URL.Path,
				"method", r.Method,
				"error", err)

			switch {
			case errors.Is(err, csrf.ErrTokenMissing):
				sendAuthError(w, http.StatusForbidden, errCodeForbidden, "CSRF token required")
			case errors.Is(err, csrf.ErrTokenExpired):
				sendAuthError(w, http.StatusForbidden, errCodeForbidden, "CSRF token expired")
			default:
				sendAuthError(w, http.StatusForbidden, errCodeForbidden, "Invalid CSRF token")
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetSessionIDFromRequest derives the CSRF session key from the request's
// authenticated JWT. The JWT is extracted the same way the auth middleware
// reads it (cookie, then Authorization header, then WebSocket subprotocol),
// then hashed via foundation's SessionKey so the bearer plaintext is never
// stored in the manager. Returns "" when the request carries no token, so the
// middleware can reject with 401. Exported for the CSRF token endpoint handler,
// which must derive the same key it later validates against.
func GetSessionIDFromRequest(r *http.Request) string {
	token, _ := GetTokenFromRequest(r)
	// Only a JWT-shaped bearer (a browser session token) is CSRF-relevant.
	// A malformed value or a non-JWT bearer gets no session key. That is a
	// rejection, not a pass: an empty session ID makes the middleware answer
	// 401. A resolved API token skips this middleware entirely on the
	// IsAPITokenAuth marker (#2450); anything else with a non-JWT bearer has
	// not authenticated at all. This mirrors the pre-migration gate (the old
	// payload-segment extraction returned "" for a non-JWT), changing only the
	// key derivation to sha256(bearer).
	const jwtMinParts = 2
	if len(strings.Split(token, ".")) < jwtMinParts {
		return ""
	}
	return csrf.SessionKey(token)
}
