package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// Note: time package not needed - we get expiry duration from recoveryManager

// RecoveryStatusResponse represents the recovery mode status.
type RecoveryStatusResponse struct {
	Active        bool   `json:"active"`
	RemainingTime int    `json:"remainingTime,omitempty"` // Seconds remaining until token expires
	Instructions  string `json:"instructions,omitempty"`
}

// RecoveryCompleteRequest represents a password recovery request.
type RecoveryCompleteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// RecoveryCompleteResponse represents a recovery completion response.
type RecoveryCompleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// handleRecoveryStatus checks if password recovery mode is active.
// This endpoint is public (no auth required) so the login page can check status.
func (s *Server) handleRecoveryStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	// Check if recovery manager is configured
	if s.recoveryManager == nil {
		sendJSONResponse(w, logger, http.StatusOK, RecoveryStatusResponse{
			Active: false,
		})
		return
	}

	// Check if recovery mode is active (this also generates token if trigger file exists)
	active := s.recoveryManager.CheckRecoveryMode()

	resp := RecoveryStatusResponse{
		Active: active,
	}

	if active {
		remaining := s.recoveryManager.RemainingTime()
		resp.RemainingTime = int(remaining.Seconds())
		resp.Instructions = "Enter the recovery token from " + s.recoveryManager.TokenFilePath()
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// updatePasswordHash updates the password hash in config, database, and auth manager.
// Returns the username and any error encountered.
func (s *Server) updatePasswordHash(ctx context.Context, hash string) (string, error) {
	// Update config with new password hash
	s.config.Lock()
	s.config.Auth.DefaultPasswordHash = hash
	username := s.config.Auth.DefaultUsername
	s.config.Unlock()

	// Update user in database if available
	if s.db != nil {
		userStore := database.NewUserStoreAdapter(s.db)
		if updateErr := userStore.UpdatePassword(ctx, username, hash); updateErr != nil {
			logging.FromContext(ctx).
				WarnContext(ctx, "Failed to update user in database during recovery", "error", updateErr)

			// Continue anyway - config update is the primary storage
		}
	}

	// Update auth manager (invalidates all existing tokens)
	s.authManager.UpdatePasswordHash(ctx, hash)

	// Save config to disk
	if saveErr := s.config.Save(s.configPath); saveErr != nil {
		return username, saveErr
	}

	return username, nil
}

// handleRecoveryComplete processes password recovery with a valid token.
// Requires a valid recovery token that was written to the filesystem.
func (s *Server) handleRecoveryComplete(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	clientIP := s.getClientIP(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	// Check if recovery manager is configured
	if s.recoveryManager == nil {
		logger.Warn("Recovery attempt but recovery manager not configured",
			"client_ip", clientIP,
			"event", "auth.recovery.not_configured")
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable, ErrCodeInternal,
			"Password recovery is not available", "")
		return
	}

	// Check rate limiting for recovery attempts
	if s.loginRateLimiter.IsBlocked(clientIP) {
		logger.Warn("Recovery blocked due to rate limiting",
			"client_ip", clientIP,
			"event", "auth.recovery.blocked")
		w.Header().Set("Retry-After", "900")
		sendErrorResponseWithDetails(w, logger, http.StatusTooManyRequests, ErrCodeRateLimit,
			"Too many attempts. Please try again later.", "")
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeAuth)

	var req RecoveryCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Recovery decode error", "client_ip", clientIP, "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"), "")
		return
	}

	// Validate the recovery token
	if !s.recoveryManager.ValidateAndConsume(req.Token) {
		logger.Warn("Recovery failed - invalid or expired token",
			"client_ip", clientIP,
			"event", "auth.recovery.invalid_token")

		// Record failed attempt for rate limiting
		s.loginRateLimiter.RecordAttempt(clientIP, false)

		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized, ErrCodeUnauthorized,
			"Invalid or expired recovery token", "")
		return
	}

	// Token is valid - proceed with password reset

	// Validate password strength
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		logger.Warn("Recovery failed - weak password",
			"client_ip", clientIP,
			"event", "auth.recovery.weak_password")
		sendJSONResponse(w, logger, http.StatusBadRequest, map[string]string{
			"error": localizer.T("errors.password.weak"),
		})
		return
	}

	// Hash the new password
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		logger.Error("Failed to hash password during recovery", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T("errors.api.internalError"), "")
		return
	}

	// Update password in config, database, and auth manager
	username, err := s.updatePasswordHash(r.Context(), hash)
	if err != nil {
		logger.Error("Failed to save config after recovery", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T("errors.config.failedToSave"), "")
		return
	}

	// Cleanup recovery files
	s.recoveryManager.Cleanup()

	// Reset rate limiter for this IP after successful recovery
	s.loginRateLimiter.RecordAttempt(clientIP, true)

	// Security audit log
	logger.Info("Password recovery completed successfully",
		"client_ip", clientIP,
		"username", username,
		"event", "auth.recovery.success")

	sendJSONResponse(w, logger, http.StatusOK, RecoveryCompleteResponse{
		Success: true,
		Message: "Password has been reset. All existing sessions have been invalidated.",
	})
}

// RecoveryInstructionsResponse provides instructions for starting recovery.
type RecoveryInstructionsResponse struct {
	TriggerFile string   `json:"triggerFile"`
	TokenFile   string   `json:"tokenFile"`
	ExpiryTime  string   `json:"expiryTime"`
	Steps       []string `json:"steps"`
}

// handleRecoveryInstructions returns instructions for password recovery.
// This is a public endpoint that provides information without exposing secrets.
func (s *Server) handleRecoveryInstructions(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	if s.recoveryManager == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable, ErrCodeInternal,
			"Password recovery is not configured", "")
		return
	}

	expiryDuration := s.recoveryManager.TokenExpiryDuration()
	resp := RecoveryInstructionsResponse{
		TriggerFile: s.recoveryManager.TriggerFilePath(),
		TokenFile:   s.recoveryManager.TokenFilePath(),
		ExpiryTime:  expiryDuration.String(),
		Steps: []string{
			"1. SSH into the server",
			"2. Create the trigger file: touch " + s.recoveryManager.TriggerFilePath(),
			"3. Wait a moment, then read the token: cat " + s.recoveryManager.TokenFilePath(),
			"4. Return to this page and enter the token with your new password",
			"5. The token expires after " + expiryDuration.String(),
		},
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}
