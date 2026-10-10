// middleware.go carries the HTTP authentication middleware for package auth,
// its token-extraction helpers, and the JSON error shape shared with the CSRF
// middleware in csrf.go.

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// errCodeUnauthorized is the code of the auth middleware's refusals (matches
// api.ErrCodeUnauthorized).
const errCodeUnauthorized = "UNAUTHORIZED"

// authErrorResponse represents a standardized error response for auth middleware.
type authErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// sendUnauthorized sends the auth middleware's 401 in the API error schema.
func sendUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	resp := authErrorResponse{
		Error: message,
		Code:  errCodeUnauthorized,
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
		sendUnauthorized(w, "Token expired")
		return
	}
	sendUnauthorized(w, "Invalid token")
}

// patAuthKey marks a request already authenticated by the personal-access-token
// middleware that runs in front of this one. Named for the abbreviation rather
// than "apiToken" because gosec's G101 heuristic reads any string constant
// whose identifier contains "token" as a hardcoded credential.
const patAuthKey contextKey = "pat_auth"

// WithAPITokenAuth marks the request as authenticated by a personal access
// token. The marker lives in the context, not a header: a header is writable by
// the caller, and this one decides whether the JWT and CSRF middlewares are
// skipped. It is set only after the token store has resolved the token.
func WithAPITokenAuth(ctx context.Context) context.Context {
	return context.WithValue(ctx, patAuthKey, true)
}

// IsAPITokenAuth reports whether the PAT middleware already authenticated this
// request.
func IsAPITokenAuth(ctx context.Context) bool {
	authenticated, ok := ctx.Value(patAuthKey).(bool)
	return ok && authenticated
}

// Middleware returns an HTTP middleware that validates JWT tokens.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// #2450: the PAT middleware runs in front of this one and forwards the
		// request with the owner's identity already set. Re-reading the bearer
		// here would validate an `sd_pat_…` string as a JWT and reject every
		// PAT request as "Invalid token".
		if IsAPITokenAuth(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from request (WebSocket subprotocol or cookie/header)
		tokenString := extractTokenFromRequest(r)
		if tokenString == "" {
			sendUnauthorized(w, "Unauthorized")
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
			sendUnauthorized(w, "Invalid token: missing username claim")
			return
		}

		// Add claims to request context
		ctx := logging.WithUserID(r.Context(), claims.Username)
		ctx = WithClientID(ctx, claims.ClientID)
		// #2632: the identity travels on the context, not on a header the
		// caller could have written. Nothing sets X-Username any more, so
		// there is nothing to spoof.
		ctx = WithUsername(ctx, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
