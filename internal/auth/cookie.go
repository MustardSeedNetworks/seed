package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// CookieNameAccess is the name of the access token cookie.
	CookieNameAccess = "seed_access"

	// CookieNameRefresh is the name of the refresh token cookie.
	CookieNameRefresh = "seed_refresh"

	// AccessTokenDuration is how long access tokens are valid (short-lived).
	AccessTokenDuration = 15 * time.Minute

	// RefreshTokenDuration is how long refresh tokens are valid.
	RefreshTokenDuration = 7 * 24 * time.Hour // 7 days

	// MaxSessionLifetime is the absolute maximum time a session can last (fixes #717).
	// Even with valid refresh tokens, sessions expire after this duration.
	MaxSessionLifetime = 24 * time.Hour // 24 hours
)

// CookieConfig holds cookie scope settings.
//
// Security attributes (Secure, HttpOnly, SameSite) are hardcoded on every
// auth cookie because the daemon serves HTTPS only — it binds no HTTP
// listener, so no auth flow ever runs over plain HTTP. They are
// intentionally NOT configurable.
type CookieConfig struct {
	// Domain sets the cookie domain
	Domain string

	// Path sets the cookie path
	Path string
}

// DefaultCookieConfig returns the standard cookie scope for seed.
func DefaultCookieConfig() CookieConfig {
	return CookieConfig{
		Domain: "", // Current domain
		Path:   "/",
	}
}

// newAuthCookie builds an [http.Cookie] with seed's hardcoded auth-cookie
// security baseline. Centralising the literals here keeps every auth
// cookie identical (and lets gosec G124 see Secure/HttpOnly/SameSite are
// all set unconditionally).
func newAuthCookie(name, value string, expires time.Time, maxAge int, config CookieConfig) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     config.Path,
		Domain:   config.Domain,
		Expires:  expires,
		MaxAge:   maxAge,
		Secure:   true,                    // HTTPS-only (no HTTP listener; auth never served over plain HTTP)
		HttpOnly: true,                    // Prevent JavaScript access (XSS protection)
		SameSite: http.SameSiteStrictMode, // Block cross-site contexts (CSRF)
	}
}

// SetAccessTokenCookie sets the access token as an httpOnly cookie.
func SetAccessTokenCookie(w http.ResponseWriter, token string, config CookieConfig) {
	http.SetCookie(
		w,
		newAuthCookie(
			CookieNameAccess,
			token,
			time.Now().Add(AccessTokenDuration),
			int(AccessTokenDuration.Seconds()),
			config,
		),
	)
}

// SetRefreshTokenCookie sets the refresh token as an httpOnly cookie.
func SetRefreshTokenCookie(w http.ResponseWriter, token string, config CookieConfig) {
	http.SetCookie(
		w,
		newAuthCookie(
			CookieNameRefresh,
			token,
			time.Now().Add(RefreshTokenDuration),
			int(RefreshTokenDuration.Seconds()),
			config,
		),
	)
}

// ClearAuthCookies removes both access and refresh token cookies.
func ClearAuthCookies(w http.ResponseWriter, config CookieConfig) {
	for _, name := range []string{CookieNameAccess, CookieNameRefresh} {
		http.SetCookie(w, newAuthCookie(name, "", time.Unix(0, 0), -1, config))
	}
}

// GetAccessTokenFromCookie extracts the access token from cookies.
func GetAccessTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieNameAccess)
	if err != nil {
		return "", fmt.Errorf("access token cookie not found: %w", err)
	}
	return cookie.Value, nil
}

// GetRefreshTokenFromCookie extracts the refresh token from cookies.
func GetRefreshTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieNameRefresh)
	if err != nil {
		return "", fmt.Errorf("refresh token cookie not found: %w", err)
	}
	return cookie.Value, nil
}

// GetTokenFromRequest tries to extract token from request in order of preference:
// 1. Cookie (most secure).
// 2. Authorization header (Bearer token - fallback for API clients).
// 3. Sec-WebSocket-Protocol header (for WebSocket connections).
// Query parameter authentication is disabled for security (fixes #706).
func GetTokenFromRequest(r *http.Request) (string, string) {
	// Try cookie first (most secure)
	if token, err := GetAccessTokenFromCookie(r); err == nil && token != "" {
		return token, "cookie"
	}

	// Try Authorization header (API client fallback)
	if auth := r.Header.Get("Authorization"); auth != "" {
		const bearerPrefix = "Bearer "
		if len(auth) > len(bearerPrefix) && auth[:len(bearerPrefix)] == bearerPrefix {
			return auth[len(bearerPrefix):], "header"
		}
	}

	// Try Sec-WebSocket-Protocol header for WebSocket auth (fixes #706 alternative)
	// Format: "access_token, <token>" where browser sends protocols as comma-separated
	if wsProtocol := r.Header.Get("Sec-WebSocket-Protocol"); wsProtocol != "" {
		protocols := strings.Split(wsProtocol, ",")
		for i, p := range protocols {
			p = strings.TrimSpace(p)
			// Look for token after "access_token" marker
			if p == "access_token" && i+1 < len(protocols) {
				return strings.TrimSpace(protocols[i+1]), "subprotocol"
			}
		}
	}

	return "", "none"
}
