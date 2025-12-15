// Package auth provides authentication and authorization.
package auth

import (
	"fmt"
	"net/http"
	"time"
)

const (
	// CookieNameAccess is the name of the access token cookie
	CookieNameAccess = "luminetiq_access"

	// CookieNameRefresh is the name of the refresh token cookie
	CookieNameRefresh = "luminetiq_refresh"

	// AccessTokenDuration is how long access tokens are valid (short-lived)
	AccessTokenDuration = 15 * time.Minute

	// RefreshTokenDuration is how long refresh tokens are valid
	RefreshTokenDuration = 7 * 24 * time.Hour // 7 days
)

// CookieConfig holds cookie security settings.
type CookieConfig struct {
	// Secure sets the Secure flag (HTTPS only)
	Secure bool

	// SameSite sets the SameSite attribute
	SameSite http.SameSite

	// Domain sets the cookie domain
	Domain string

	// Path sets the cookie path
	Path string
}

// DefaultCookieConfig returns secure defaults.
func DefaultCookieConfig() CookieConfig {
	return CookieConfig{
		Secure:   true, // HTTPS only
		SameSite: http.SameSiteStrictMode,
		Domain:   "", // Current domain
		Path:     "/",
	}
}

// SetAccessTokenCookie sets the access token as an httpOnly cookie.
func SetAccessTokenCookie(w http.ResponseWriter, token string, config CookieConfig) {
	cookie := &http.Cookie{
		Name:     CookieNameAccess,
		Value:    token,
		Path:     config.Path,
		Domain:   config.Domain,
		Expires:  time.Now().Add(AccessTokenDuration),
		MaxAge:   int(AccessTokenDuration.Seconds()),
		Secure:   config.Secure,
		HttpOnly: true, // Prevent JavaScript access (XSS protection)
		SameSite: config.SameSite,
	}
	http.SetCookie(w, cookie)
}

// SetRefreshTokenCookie sets the refresh token as an httpOnly cookie.
func SetRefreshTokenCookie(w http.ResponseWriter, token string, config CookieConfig) {
	cookie := &http.Cookie{
		Name:     CookieNameRefresh,
		Value:    token,
		Path:     config.Path,
		Domain:   config.Domain,
		Expires:  time.Now().Add(RefreshTokenDuration),
		MaxAge:   int(RefreshTokenDuration.Seconds()),
		Secure:   config.Secure,
		HttpOnly: true,
		SameSite: config.SameSite,
	}
	http.SetCookie(w, cookie)
}

// ClearAuthCookies removes both access and refresh token cookies.
func ClearAuthCookies(w http.ResponseWriter, config CookieConfig) {
	// Set expired cookies to clear them
	clearCookie := func(name string) {
		cookie := &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     config.Path,
			Domain:   config.Domain,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			Secure:   config.Secure,
			HttpOnly: true,
			SameSite: config.SameSite,
		}
		http.SetCookie(w, cookie)
	}

	clearCookie(CookieNameAccess)
	clearCookie(CookieNameRefresh)
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
// 1. Cookie (most secure)
// 2. Authorization header (Bearer token - fallback for API clients)
// 3. Query parameter (least secure - WebSocket only, deprecated)
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

	// Try query parameter (WebSocket legacy support - deprecated)
	if token := r.URL.Query().Get("token"); token != "" {
		return token, "query"
	}

	return "", "none"
}
