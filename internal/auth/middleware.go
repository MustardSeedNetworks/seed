// middleware.go carries the HTTP authentication middleware for package auth,
// its token-extraction helpers, and the JSON error shape shared with the CSRF
// middleware in csrf.go.

package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Common error codes for auth middleware JSON responses (matches api.ErrorResponse).
const (
	errCodeUnauthorized = "UNAUTHORIZED"
	errCodeForbidden    = "FORBIDDEN"
)

// authErrorResponse represents a standardized error response for auth middleware.
type authErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// sendAuthError sends a JSON error response from auth/CSRF middleware.
// This ensures consistent error formats matching the API error schema.
func sendAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := authErrorResponse{
		Error: message,
		Code:  code,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.GetLogger().Error("Failed to encode auth error response", "error", err)
	}
}

// extractTokenFromSubprotocol extracts the JWT token from WebSocket subprotocol header.
// Supports formats:
//   - "access_token, <token>"
//   - "bearer, <token>"
//   - Just "<token>" (fallback)
func extractTokenFromSubprotocol(protocols string) string {
	// Split by comma to handle multiple protocols
	parts := strings.Split(protocols, ",")

	for i, part := range parts {
		part = strings.TrimSpace(part)

		// Check if this part is the auth protocol indicator
		if part == "access_token" || part == "bearer" {
			// Next part should be the token
			if i+1 < len(parts) {
				return strings.TrimSpace(parts[i+1])
			}
		}
	}

	// Fallback: if no recognized protocol, treat the whole string as token
	// This handles cases where client sends just the token
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0])
	}

	return ""
}

// preSessionPaths are the API paths reachable before the caller holds an access
// token: the login exchange and its second factor, first-run setup, and the
// recovery that rescues a locked-out account (#478, and #85 for the second
// factor).
//
// Every one of these that accepts a mutating method must also be on
// isCSRFExemptPath. They have no session, so the CSRF middleware would find no
// session ID and answer 401 — not protection, a wall. That divergence made
// TOTP enrolment a permanent lockout (#2391), which is why
// TestPreSessionPathsAreCSRFExempt now derives one list from the other.
func preSessionPaths() []string {
	return []string{
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/setup/status",
		"/api/v1/setup/complete",
		"/api/v1/recovery/status",
		"/api/v1/recovery/complete",
		"/api/v1/recovery/instructions",
		"/api/v1/auth/login/totp",
		"/api/v1/auth/webauthn/login/begin",
		"/api/v1/auth/webauthn/login/finish",
	}
}

// shouldBypassAuth checks if the given path should skip authentication.
// Returns true for login, refresh, setup, recovery, SSO endpoints and static files.
func shouldBypassAuth(path string) bool {
	if slices.Contains(preSessionPaths(), path) {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/sso/") {
		return true
	}
	// Skip auth for static files (non-API, non-WebSocket paths)
	if !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/ws") {
		return true
	}
	return false
}

// extractTokenFromRequest extracts the JWT token from the request.
// For WebSocket connections, checks Sec-WebSocket-Protocol header first.
// Falls back to cookie/Authorization header via GetTokenFromRequest.
func extractTokenFromRequest(r *http.Request) string {
	// For WebSocket connections, check Sec-WebSocket-Protocol first (fixes #478)
	if strings.HasPrefix(r.URL.Path, "/ws") {
		// Method 1 (Preferred): Check Sec-WebSocket-Protocol header
		// Format: "access_token, <token>" or "bearer, <token>"
		if protocols := r.Header.Get("Sec-WebSocket-Protocol"); protocols != "" {
			if token := extractTokenFromSubprotocol(protocols); token != "" {
				return token
			}
		}
	}

	// Use unified token extraction with cookie priority (fixes #478)
	token, _ := GetTokenFromRequest(r)
	return token
}

// handleTokenValidationError sends the appropriate error response for token validation failures.
func handleTokenValidationError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrTokenExpired) {
		sendAuthError(w, http.StatusUnauthorized, errCodeUnauthorized, "Token expired")
		return
	}
	sendAuthError(w, http.StatusUnauthorized, errCodeUnauthorized, "Invalid token")
}

// Middleware returns an HTTP middleware that validates JWT tokens.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for bypassed paths (login, refresh, setup, SSO, static files)
		if shouldBypassAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from request (WebSocket subprotocol or cookie/header)
		tokenString := extractTokenFromRequest(r)
		if tokenString == "" {
			sendAuthError(w, http.StatusUnauthorized, errCodeUnauthorized, "Unauthorized")
			return
		}

		// Validate the token
		claims, err := m.ValidateToken(r.Context(), tokenString)
		if err != nil {
			handleTokenValidationError(w, err)
			return
		}

		// Validate username claim exists and is not empty (fixes #711)
		if claims.Username == "" {
			sendAuthError(w, http.StatusUnauthorized, errCodeUnauthorized, "Invalid token: missing username claim")
			return
		}

		// Add claims to request context
		ctx := logging.WithUserID(r.Context(), claims.Username)
		ctx = WithClientID(ctx, claims.ClientID)
		r.Header.Set("X-Username", claims.Username) // Keep this for other potential uses
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
