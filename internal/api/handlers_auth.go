// Package api provides the HTTP/WebSocket server.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/auth"
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client IP for rate limiting
	clientIP := GetClientIP(r)

	// Check if IP is rate limited
	if s.loginRateLimiter.IsBlocked(clientIP) {
		w.Header().Set("Retry-After", "900") // 15 minutes
		remaining := s.loginRateLimiter.RemainingAttempts(clientIP)
		sendJSONResponse(w, http.StatusTooManyRequests, map[string]interface{}{
			"error":              "Too many failed login attempts",
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
		slog.Warn("Login decode error", "client_ip", clientIP, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Authenticate user (validates credentials)
	_, err := s.authManager.Authenticate(req.Username, req.Password)
	if err != nil {
		// Record failed attempt
		blocked := s.loginRateLimiter.RecordAttempt(clientIP, false)
		remaining := s.loginRateLimiter.RemainingAttempts(clientIP)

		if blocked {
			w.Header().Set("Retry-After", "900")
			sendJSONResponse(w, http.StatusTooManyRequests, map[string]interface{}{
				"error":              "Too many failed login attempts. Account temporarily locked.",
				"retry_after":        900,
				"remaining_attempts": 0,
			})
			return
		}

		sendJSONResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"error":              "Invalid credentials",
			"remaining_attempts": remaining,
		})
		return
	}

	// Record successful attempt (clears previous failures)
	s.loginRateLimiter.RecordAttempt(clientIP, true)

	// Generate access and refresh tokens (fixes #478)
	accessToken, err := s.authManager.GenerateAccessToken(req.Username)
	if err != nil {
		slog.Error("Failed to generate access token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	refreshToken, err := s.authManager.GenerateRefreshToken(req.Username)
	if err != nil {
		slog.Error("Failed to generate refresh token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set httpOnly cookies (fixes #478)
	cookieConfig := auth.DefaultCookieConfig()
	// Allow insecure cookies in development (LUMINETIQ_DEV=1)
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

	sendJSONResponse(w, http.StatusOK, resp)
}

// handleLogout handles user logout (fixes #544 - split from handlers.go).
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clear authentication cookies (fixes #478)
	cookieConfig := auth.DefaultCookieConfig()
	if !s.config.Server.HTTPS {
		cookieConfig.Secure = false
	}
	auth.ClearAuthCookies(w, cookieConfig)

	sendJSONResponse(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// handleRefreshToken handles token refresh using refresh token (fixes #478).
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get refresh token from cookie
	refreshToken, err := auth.GetRefreshTokenFromCookie(r)
	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Refresh token not found",
		})
		return
	}

	// Generate new access token
	newAccessToken, err := s.authManager.RefreshAccessToken(refreshToken)
	if err != nil {
		sendJSONResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Invalid or expired refresh token",
		})
		return
	}

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

	sendJSONResponse(w, http.StatusOK, resp)
}

// SetupStatusResponse represents the setup status response.
type SetupStatusResponse struct {
	NeedsSetup        bool   `json:"needsSetup"`
	SuggestedPassword string `json:"suggestedPassword,omitempty"`
}

// handleSetupStatus checks if initial setup is required (fixes #544 - split from handlers.go).
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	sendJSONResponse(w, http.StatusOK, resp)
}

// SetupCompleteRequest represents the setup completion request.
type SetupCompleteRequest struct {
	Password string `json:"password"`
}

// handleSetupComplete completes initial setup by setting admin password (fixes #544 - split from handlers.go).
func (s *Server) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, 1024)

	var req SetupCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Setup decode error", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, map[string]string{
			"error": "Password does not meet strength requirements",
		})
		return
	}

	// Hash the new password
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to hash password",
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
		slog.Error("Failed to save config after setup", "error", err)
		sendJSONResponse(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to save configuration",
		})
		return
	}

	slog.Info("Initial setup completed successfully")
	sendJSONResponse(w, http.StatusOK, map[string]string{
		"status": "success",
	})
}
