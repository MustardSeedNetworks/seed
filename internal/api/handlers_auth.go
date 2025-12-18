// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a successful login response.
type LoginResponse struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

// handleLogin handles user login (fixes #544 - split from handlers.go).
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "") // fixes #694, #699
		return
	}

	// Get client IP for rate limiting (fixes #716)
	// Each IP is tracked independently, providing protection against distributed attacks
	clientIP := GetClientIP(r)

	// Check if IP is rate limited
	if s.loginRateLimiter.IsBlocked(clientIP) {
		// Security audit log: blocked login attempt (fixes #697)
		logger.Warn("Login blocked due to rate limiting",
			"client_ip", clientIP,
			"event", "auth.login.blocked")
		w.Header().Set("Retry-After", "900") // 15 minutes
		remaining := s.loginRateLimiter.RemainingAttempts(clientIP)
		sendJSONResponse(w, logger, http.StatusTooManyRequests, map[string]interface{}{
			"error":              localizer.T("errors.auth.tooManyAttempts"),
			"retry_after":        900,
			"remaining_attempts": remaining,
		})
		return
	}

	// Limit request body size to prevent memory exhaustion (1KB is plenty for login)
	r.Body = http.MaxBytesReader(w, r.Body, 1024)

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Log error without exposing credentials
		logger.Warn("Login decode error", "client_ip", clientIP, "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "") // fixes #694, #699
		return
	}

	// Authenticate user (validates credentials)
	_, err := s.authManager.Authenticate(req.Username, req.Password)
	if err != nil {
		// Security audit log: failed login attempt (fixes #697)
		logger.Warn("Login failed",
			"username", req.Username,
			"client_ip", clientIP,
			"event", "auth.login.failed",
			"error", "invalid credentials")

		// Record failed attempt
		blocked := s.loginRateLimiter.RecordAttempt(clientIP, false)
		remaining := s.loginRateLimiter.RemainingAttempts(clientIP)

		if blocked {
			// Security audit log: account locked (fixes #697)
			logger.Warn("Account locked due to too many failed attempts",
				"username", req.Username,
				"client_ip", clientIP,
				"event", "auth.account.locked")
			w.Header().Set("Retry-After", "900")
			sendJSONResponse(w, logger, http.StatusTooManyRequests, map[string]interface{}{
				"error":              localizer.T("errors.auth.accountLocked"),
				"retry_after":        900,
				"remaining_attempts": 0,
			})
			return
		}

		sendJSONResponse(w, logger, http.StatusUnauthorized, map[string]interface{}{
			"error":              localizer.T("errors.auth.invalidCredentials"),
			"remaining_attempts": remaining,
		})
		return
	}

	// Security audit log: successful login (fixes #697)
	logger.Info("Login successful",
		"username", req.Username,
		"client_ip", clientIP,
		"event", "auth.login.success")

	// Record successful attempt (clears previous failures)
	s.loginRateLimiter.RecordAttempt(clientIP, true)

	// Generate access and refresh tokens (fixes #478)
	accessToken, err := s.authManager.GenerateAccessToken(req.Username)
	if err != nil {
		logger.Error("Failed to generate access token", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, localizer.T("errors.api.internalError"), "") // fixes #694, #699
		return
	}

	refreshToken, err := s.authManager.GenerateRefreshToken(req.Username)
	if err != nil {
		logger.Error("Failed to generate refresh token", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal, localizer.T("errors.api.internalError"), "") // fixes #694, #699
		return
	}

	// Set httpOnly cookies (fixes #478)
	cookieConfig := auth.DefaultCookieConfig()
	// Allow insecure cookies in development mode (HTTPS disabled)
	if !s.config.Server.HTTPS {
		cookieConfig.Secure = false
	}
	auth.SetAccessTokenCookie(w, accessToken, cookieConfig)
	auth.SetRefreshTokenCookie(w, refreshToken, cookieConfig)

	// Also return access token in response for backwards compatibility (API clients)
	resp := LoginResponse{
		Token:   accessToken,
		Expires: time.Now().Add(auth.AccessTokenDuration).Unix(),
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleLogout handles user logout (fixes #544 - split from handlers.go).
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "") // fixes #694, #699
		return
	}

	// Security audit log: user logout (fixes #697)
	clientIP := GetClientIP(r)
	logger.Info("User logout",
		"client_ip", clientIP,
		"event", "auth.logout")

	// Clear authentication cookies (fixes #478)
	cookieConfig := auth.DefaultCookieConfig()
	if !s.config.Server.HTTPS {
		cookieConfig.Secure = false
	}
	auth.ClearAuthCookies(w, cookieConfig)

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{"status": "logged out"})
}

// handleRefreshToken handles token refresh using refresh token (fixes #478).
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "") // fixes #694, #699
		return
	}

	// Get refresh token from cookie
	refreshToken, err := auth.GetRefreshTokenFromCookie(r)
	if err != nil {
		// Security audit log: refresh token not found (fixes #697)
		clientIP := GetClientIP(r)
		logger.Warn("Token refresh failed - token not found",
			"client_ip", clientIP,
			"event", "auth.refresh.failed",
			"error", "token not found")
		sendJSONResponse(w, logger, http.StatusUnauthorized, map[string]string{
			"error": localizer.T("errors.auth.refreshNotFound"),
		})
		return
	}

	// Generate new access token
	newAccessToken, err := s.authManager.RefreshAccessToken(refreshToken)
	if err != nil {
		// Security audit log: invalid/expired refresh token (fixes #697)
		clientIP := GetClientIP(r)
		logger.Warn("Token refresh failed - invalid or expired token",
			"client_ip", clientIP,
			"event", "auth.refresh.failed",
			"error", err.Error())
		sendJSONResponse(w, logger, http.StatusUnauthorized, map[string]string{
			"error": localizer.T("errors.auth.expiredToken"),
		})
		return
	}

	// Security audit log: successful token refresh (fixes #697)
	clientIP := GetClientIP(r)
	logger.Info("Token refresh successful",
		"client_ip", clientIP,
		"event", "auth.refresh.success")

	// Set new access token cookie
	cookieConfig := auth.DefaultCookieConfig()
	if !s.config.Server.HTTPS {
		cookieConfig.Secure = false
	}
	auth.SetAccessTokenCookie(w, newAccessToken, cookieConfig)

	// Return new access token
	resp := LoginResponse{
		Token:   newAccessToken,
		Expires: time.Now().Add(auth.AccessTokenDuration).Unix(),
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// SetupStatusResponse represents the setup status response.
type SetupStatusResponse struct {
	NeedsSetup        bool   `json:"needsSetup"`
	SuggestedPassword string `json:"suggestedPassword,omitempty"`
}

// handleSetupStatus checks if initial setup is required (fixes #544 - split from handlers.go).
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "") // fixes #694, #699
		return
	}

	// Check if using default password
	needsSetup := auth.IsDefaultPasswordHash(s.config.Auth.DefaultPasswordHash)

	resp := SetupStatusResponse{
		NeedsSetup: needsSetup,
	}

	// If setup is needed, suggest a secure password
	if needsSetup {
		suggestedPassword, err := auth.GenerateSecurePassword(16)
		if err == nil {
			resp.SuggestedPassword = suggestedPassword
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// SetupCompleteRequest represents the setup completion request.
type SetupCompleteRequest struct {
	Password string `json:"password"`
}

// handleSetupComplete completes initial setup by setting admin password (fixes #544 - split from handlers.go).
// Security fix #758, #724: Only allow setup completion when setup is actually needed (default password).
func (s *Server) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "") // fixes #694, #699
		return
	}

	// Security: Only allow setup when password is still the default (fixes #758, #724)
	// This prevents unauthenticated password reset after initial setup
	if !auth.IsDefaultPasswordHash(s.config.Auth.DefaultPasswordHash) {
		clientIP := GetClientIP(r)
		logger.Warn("Setup complete attempted after initial setup already done",
			"client_ip", clientIP,
			"event", "auth.setup.blocked")
		sendErrorResponseWithDetails(w, logger, http.StatusForbidden, ErrCodeForbidden,
			"Setup has already been completed. Use authenticated password change instead.", "")
		return
	}

	// Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, 1024)

	var req SetupCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Setup decode error", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "") // fixes #694, #699
		return
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		sendJSONResponse(w, logger, http.StatusBadRequest, map[string]string{
			"error": localizer.T("errors.password.weak"),
		})
		return
	}

	// Hash the new password
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		sendJSONResponse(w, logger, http.StatusInternalServerError, map[string]string{
			"error": localizer.T("errors.api.internalError"),
		})
		return
	}

	// Update config with new password hash
	s.config.Lock()
	s.config.Auth.DefaultPasswordHash = hash
	s.config.Unlock()

	// Update auth manager
	s.authManager.UpdatePasswordHash(hash)

	// Save config to disk
	if err := s.config.Save(s.configPath); err != nil {
		logger.Error("Failed to save config after setup", "error", err)
		sendJSONResponse(w, logger, http.StatusInternalServerError, map[string]string{
			"error": localizer.T("errors.config.failedToSave"),
		})
		return
	}

	// Security audit log: initial setup completed (fixes #697)
	clientIP := GetClientIP(r)
	logger.Info("Initial setup completed - admin password changed",
		"client_ip", clientIP,
		"event", "auth.setup.complete")

	sendJSONResponse(w, logger, http.StatusOK, map[string]string{
		"status": "success",
	})
}
