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

// GenerateToken mints a fresh CSRF token for the given session key (derived
// from the caller's JWT via GetSessionIDFromRequest), replacing any existing
// one.
func (m *CSRFManager) GenerateToken(sessionID string) (string, error) {
	return m.mgr.Generate(sessionID)
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
// pre-session auth/setup endpoints, the client-log sink, and the SSO handshake.
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
		"/api/v1/reporting/logs/client": // logger runs before CSRF tokens exist
		return true
	}
	// SSO handshake endpoints are pre-session.
	return strings.HasPrefix(path, "/api/v1/sso/")
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
	if token == "" {
		return ""
	}
	return csrf.SessionKey(token)
}
