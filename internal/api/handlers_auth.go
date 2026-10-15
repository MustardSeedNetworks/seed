package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Authentication constants.
const (
	// suggestedPasswordLength is the length of auto-generated suggested passwords.
	suggestedPasswordLength = 16

	// rateLimitRetryAfterSec is the Retry-After header value in seconds for rate-limited requests.
	rateLimitRetryAfterSec = 900
)

// LoginRequest represents a login request.
type LoginRequest struct {
	// Length cap is intentionally large; the auth layer applies its own
	// runtime guards on credential length. Catch only the obvious: empty.
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a successful login response.
//
// MFARequired and MFAToken are populated instead of Token/Expires when
// the user has TOTP enabled — the client must then call
// /api/v1/auth/login/totp with the supplied mfa_token + a fresh code
// to obtain the real access token. Wave 3 (#85).
type LoginResponse struct {
	Token       string `json:"token,omitempty"`
	Expires     int64  `json:"expires,omitempty"`
	MFARequired bool   `json:"mfaRequired,omitempty"`
	MFAToken    string `json:"mfaToken,omitempty"`
}

// handleLoginRateLimited checks and handles rate limiting, returns true if blocked.
func (s *Server) handleLoginRateLimited(
	w http.ResponseWriter,
	logger *slog.Logger,
	localizer *i18n.Localizer,
	clientIP string,
) bool {
	if !s.loginRateLimiter().IsBlocked(clientIP) {
		return false
	}
	logger.Warn(
		"Login blocked due to rate limiting",
		"client_ip",
		clientIP,
		"event",
		"auth.login.blocked",
	)
	w.Header().Set("Retry-After", strconv.Itoa(rateLimitRetryAfterSec))
	sendJSONResponse(w, logger, http.StatusTooManyRequests, map[string]any{
		"error":              localizer.T("errors.auth.tooManyAttempts"),
		"retry_after":        rateLimitRetryAfterSec,
		"remaining_attempts": s.loginRateLimiter().RemainingAttempts(clientIP),
	})
	return true
}

// handleLoginFailure handles a failed login attempt including rate limiting.
func (s *Server) handleLoginFailure(
	w http.ResponseWriter,
	logger *slog.Logger,
	localizer *i18n.Localizer,
	username, clientIP string,
) {
	logger.Warn("Login failed", "username", username, "client_ip", clientIP,
		"event", "auth.login.failed", "error", "invalid credentials")

	blocked := s.loginRateLimiter().RecordAttempt(clientIP, false)
	remaining := s.loginRateLimiter().RemainingAttempts(clientIP)

	if blocked {
		logger.Warn("Account locked due to too many failed attempts",
			"username", username, "client_ip", clientIP, "event", "auth.account.locked")
		w.Header().Set("Retry-After", strconv.Itoa(rateLimitRetryAfterSec))
		sendJSONResponse(w, logger, http.StatusTooManyRequests, map[string]any{
			"error": localizer.T(
				"errors.auth.accountLocked",
			), "retry_after": rateLimitRetryAfterSec, "remaining_attempts": 0,
		})
		return
	}

	sendJSONResponse(w, logger, http.StatusUnauthorized, map[string]any{
		"error": localizer.T("errors.auth.invalidCredentials"), "remaining_attempts": remaining,
	})
}

// generateAndSetLoginTokens generates tokens and sets cookies, returns access token or error.
func (s *Server) generateAndSetLoginTokens(
	w http.ResponseWriter,
	r *http.Request,
	username string,
) (string, error) {
	accessToken, err := s.authManager().GenerateAccessToken(r.Context(), username)
	if err != nil {
		return "", fmt.Errorf("access token: %w", err)
	}

	refreshToken, err := s.authManager().GenerateRefreshToken(r.Context(), username)
	if err != nil {
		return "", fmt.Errorf("refresh token: %w", err)
	}

	cookieConfig := auth.DefaultCookieConfig()
	auth.SetAccessTokenCookie(w, accessToken, cookieConfig)
	auth.SetRefreshTokenCookie(w, refreshToken, cookieConfig)

	return accessToken, nil
}

// handleLogin handles user login (fixes #544 - split from handlers.go).
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	clientIP := s.getClientIP(r)
	if s.handleLoginRateLimited(w, logger, localizer, clientIP) {
		return
	}

	var req LoginRequest
	if !decodeJSONStrictLocalizedWith(w, r, &req, MaxBodySizeAuth,
		logger, localizer, "client_ip", clientIP) {
		return
	}

	// Wave 3 (#85): verify password first, then check whether TOTP is
	// enrolled. If it is, hand back an MFA-pending token instead of
	// the real access token — the client must then post the TOTP code
	// to /api/v1/auth/login/totp to complete the login.
	if err := s.authManager().VerifyPasswordOnly(r.Context(), req.Username, req.Password); err != nil {
		s.handleLoginFailure(w, logger, localizer, req.Username, clientIP)
		return
	}
	s.loginRateLimiter().RecordAttempt(clientIP, true)

	if s.identityMFA.Available() {
		_, totpEnabled, totpErr := s.identityMFA.GetTOTP(r.Context(), req.Username)
		if totpErr == nil && totpEnabled {
			mfaToken, err := s.authManager().
				GenerateMFAPendingToken(r.Context(), req.Username)
			if err != nil {
				logger.ErrorContext(r.Context(), "Failed to generate MFA pending token", "error", err)
				sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
					ErrCodeInternal, localizer.T("errors.api.internalError"), "")
				return
			}
			logger.InfoContext(r.Context(), "Login awaiting second factor",
				"username", req.Username,
				"client_ip", clientIP,
				"event", "auth.login.mfa_required",
				"factor", "totp")
			sendJSONResponse(w, logger, http.StatusOK, LoginResponse{
				MFARequired: true,
				MFAToken:    mfaToken,
			})
			return
		}
	}

	logger.InfoContext(r.Context(),
		"Login successful",
		"username", req.Username,
		"client_ip", clientIP,
		"event", "auth.login.success",
	)

	accessToken, err := s.generateAndSetLoginTokens(w, r, req.Username)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to generate tokens", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, LoginResponse{
		Token:   accessToken,
		Expires: time.Now().Add(auth.AccessTokenDuration).Unix(),
	})
}

// handleLogout handles user logout (fixes #544 - split from handlers.go).
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	// Revoke tokens to prevent reuse (ported from Stem)
	// Extract and blacklist both access and refresh tokens
	if accessToken, err := auth.GetAccessTokenFromCookie(r); err == nil && accessToken != "" {
		s.authManager().RevokeToken(accessToken)
	}
	if refreshToken, err := auth.GetRefreshTokenFromCookie(r); err == nil && refreshToken != "" {
		s.authManager().RevokeToken(refreshToken)
	}

	// Security audit log: user logout (fixes #697)
	clientIP := s.getClientIP(r)
	logger.InfoContext(r.Context(), "User logout",
		"client_ip", clientIP,
		"event", "auth.logout")

	// Clear authentication cookies (fixes #478)
	cookieConfig := auth.DefaultCookieConfig()
	auth.ClearAuthCookies(w, cookieConfig)

	sendJSONResponse(w, logger, http.StatusOK, LogoutResponse{Status: "logged out"})
}

// LogoutResponse is the POST /api/v1/auth/logout body.
type LogoutResponse struct {
	Status string `json:"status"`
}

// handleRefreshToken handles token refresh using refresh token (fixes #478).
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	// Rate-limit refresh on the same per-IP bucket as login (#1224).
	// Refresh tokens are signed JWTs so a valid one can't be brute-forced,
	// but limiting still blunts CPU-abuse and refresh-token-stuffing if a
	// token ever leaks. Bail early if this IP is already blocked.
	clientIP := s.getClientIP(r)
	if s.handleLoginRateLimited(w, logger, localizer, clientIP) {
		return
	}

	// Get refresh token from cookie
	refreshToken, err := auth.GetRefreshTokenFromCookie(r)
	if err != nil {
		// Deliberately NOT charged to the login limiter. Having no refresh
		// cookie is the normal state of any unauthenticated page load, not a
		// credential-guessing attempt, and refresh shares login's per-IP bucket
		// (#1224). Charging it meant ordinary browsing spent the login budget:
		// see #2178, where E2E workers all share 127.0.0.1 and roughly three
		// tests' worth of absent-cookie refreshes locked the next login out
		// with a 429.
		logger.WarnContext(r.Context(), "Token refresh failed - token not found",
			"client_ip", clientIP,
			"event", "auth.refresh.failed",
			"error", "token not found")
		sendJSONResponse(w, logger, http.StatusUnauthorized, map[string]string{
			"error": localizer.T("errors.auth.refreshNotFound"),
		})
		return
	}

	// Generate new access token
	newAccessToken, err := s.authManager().RefreshAccessToken(r.Context(), refreshToken)
	if err != nil {
		// Charge the limiter only for a token that is *wrong*, never for one
		// that merely aged out. A forged or malformed token is someone probing;
		// an expired one is a session that ended normally, which every idle tab
		// produces. Both used to count, so normal use drained the login budget
		// this bucket exists to protect (#2178).
		if !errors.Is(err, auth.ErrTokenExpired) {
			s.loginRateLimiter().RecordAttempt(clientIP, false)
		}
		logger.WarnContext(r.Context(), "Token refresh failed - invalid or expired token",
			"client_ip", clientIP,
			"event", "auth.refresh.failed",
			"error", err.Error())
		sendJSONResponse(w, logger, http.StatusUnauthorized, map[string]string{
			"error": localizer.T("errors.auth.expiredToken"),
		})
		return
	}

	// Successful refresh resets the per-IP attempt counter (#1224).
	s.loginRateLimiter().RecordAttempt(clientIP, true)

	// Security audit log: successful token refresh (fixes #697)
	logger.InfoContext(r.Context(), "Token refresh successful",
		"client_ip", clientIP,
		"event", "auth.refresh.success")

	// Set new access token cookie
	cookieConfig := auth.DefaultCookieConfig()
	auth.SetAccessTokenCookie(w, newAccessToken, cookieConfig)

	// Return new access token
	resp := LoginResponse{
		Token:   newAccessToken,
		Expires: time.Now().Add(auth.AccessTokenDuration).Unix(),
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// CSRFTokenResponse represents the CSRF token response.
type CSRFTokenResponse struct {
	Token string `json:"token"`
}

// handleCSRFToken returns the authenticated session's CSRF token, minting one
// only when the session has none (#2660: minting a fresh one on every read let a
// second tab invalidate the first).
// The token must be included in X-CSRF-Token header for all state-changing requests.
func (s *Server) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	// Get session ID from JWT (set by auth middleware)
	sessionID := auth.GetSessionIDFromRequest(r)
	if sessionID == "" {
		logger.WarnContext(r.Context(), "CSRF token request without valid session")
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusUnauthorized,
			ErrCodeUnauthorized,
			localizer.T("errors.auth.invalidCredentials"),
			"",
		)
		return
	}

	token, err := s.csrfManager().GetOrCreate(sessionID)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to resolve CSRF token", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.api.internalError"),
			"",
		)
		return
	}

	// CSRF cookie must be readable by JS (so axios reads it and echoes as
	// X-CSRF-Token header) — that's the entire point of the double-submit
	// CSRF pattern. Locked down otherwise (Secure + SameSiteStrict).
	//nolint:gosec // G124: HttpOnly=false is intentional per double-submit-cookie CSRF pattern (OWASP-approved); Secure+Strict + HTTPS-required daemon are the actual defense
	// nosemgrep: go.lang.security.audit.net.cookie-missing-httponly.cookie-missing-httponly -- intentional double-submit CSRF cookie, Secure+SameSiteStrict, same rationale as the #nolint
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	sendJSONResponse(w, logger, http.StatusOK, CSRFTokenResponse{Token: token})
}
