package api

// handlers_setup.go holds the first-run setup endpoints: setup status and
// the operator-initiated creation of the bootstrap admin.

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// SetupStatusResponse represents the setup status response.
type SetupStatusResponse struct {
	NeedsSetup        bool   `json:"needsSetup"`
	SuggestedPassword string `json:"suggestedPassword,omitempty"`
	Username          string `json:"username"` // Fixes #768 - provide username from config
	// SetupTokenFile is where the operator reads the token setup/complete
	// requires. The token itself is never returned here.
	SetupTokenFile string `json:"setupTokenFile,omitempty"`
}

// handleSetupStatus checks if initial setup is required (fixes #544 - split from handlers.go).
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	// Check if using default password
	needsSetup := auth.IsDefaultPasswordHash(s.config.Auth.DefaultPasswordHash)

	// Security fix #891: Check if setup mode has timed out
	if needsSetup && !s.setupModeStartTime.IsZero() {
		setupDuration := time.Since(s.setupModeStartTime)
		if setupDuration > setupModeTimeoutMin*time.Minute {
			logger.WarnContext(r.Context(), "Setup mode has expired", "event", "auth.setup.expired",
				"elapsed_minutes", setupDuration.Minutes())
			sendErrorResponseWithDetails(w, logger, http.StatusForbidden, ErrCodeSetupExpired,
				"Setup mode has expired. Please restart the server to try again.", "")
			return
		}
	}

	resp := SetupStatusResponse{
		NeedsSetup: needsSetup,
		Username:   s.config.Auth.DefaultUsername, // Fixes #768 - provide username from config
	}

	if needsSetup {
		suggestedPassword, err := auth.GenerateSecurePassword(suggestedPasswordLength)
		if err == nil {
			resp.SuggestedPassword = suggestedPassword
		}
		resp.SetupTokenFile = s.setupTokenManager().FilePath()
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// SetupCompleteRequest represents the setup completion request.
type SetupCompleteRequest struct {
	Password   string `json:"password"   validate:"required,min=8"`
	SetupToken string `json:"setupToken" validate:"required"` // Security fix #724, #758
}

// SetupCompleteResponse acknowledges that the admin password is set.
type SetupCompleteResponse struct {
	Status string `json:"status" jsonschema:"enum=success"`
}

func ensurePostMethod(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) bool {
	if r.Method == http.MethodPost {
		return true
	}

	sendErrorResponseWithDetails(
		w,
		logger,
		http.StatusMethodNotAllowed,
		ErrCodeMethodNotAllowed,
		localizer.T("errors.api.methodNotAllowed"),
		"",
	) // fixes #694, #699
	return false
}

func (s *Server) ensureSetupCompletionAllowed(
	w http.ResponseWriter,
	logger *slog.Logger,
	_ *i18n.Localizer,
	clientIP string,
) bool {
	// Security: Only allow setup when password is still the default (fixes #758, #724)
	// This prevents unauthenticated password reset after initial setup
	if !auth.IsDefaultPasswordHash(s.config.Auth.DefaultPasswordHash) {
		logger.Warn("Setup complete attempted after initial setup already done",
			"client_ip", clientIP,
			"event", "auth.setup.blocked")
		sendErrorResponseWithDetails(w, logger, http.StatusForbidden, ErrCodeForbidden,
			"Setup has already been completed. Use authenticated password change instead.", "")
		return false
	}

	// Security fix #891: Check if setup mode has timed out
	if !s.setupModeStartTime.IsZero() {
		setupDuration := time.Since(s.setupModeStartTime)
		if setupDuration > setupModeTimeoutMin*time.Minute {
			logger.Warn("Setup completion attempted after timeout",
				"client_ip", clientIP,
				"elapsed_minutes", setupDuration.Minutes(),
				"event", "auth.setup.timeout_complete_blocked")
			sendErrorResponseWithDetails(w, logger, http.StatusForbidden, ErrCodeSetupExpired,
				"Setup mode has expired. Please restart the server to try again.", "")
			return false
		}
	}

	return true
}

func decodeSetupCompleteRequest(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	localizer *i18n.Localizer,
) (SetupCompleteRequest, bool) {
	var req SetupCompleteRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeAuth, logger, localizer) {
		return SetupCompleteRequest{}, false
	}

	return req, true
}

func (s *Server) ensureValidSetupToken(
	w http.ResponseWriter,
	logger *slog.Logger,
	clientIP string,
	token string,
) bool {
	// The token proves the caller can read the host's data directory, so a
	// network peer or a cross-site form cannot complete setup (#724, #758).
	if s.setupTokenManager().Validate(strings.TrimSpace(token)) {
		return true
	}

	logger.Warn("Setup complete attempted with invalid token",
		"client_ip", clientIP,
		"event", "auth.setup.invalid_token")
	sendErrorResponseWithDetails(w, logger, http.StatusForbidden, ErrCodeForbidden,
		"Invalid setup token. Copy it from "+s.setupTokenManager().FilePath()+" and try again.", "")
	return false
}

// handleSetupComplete completes initial setup by setting admin password (fixes #544 - split from handlers.go).
// Security fix #758, #724: Requires valid setup token and only allows when setup is actually needed.
func (s *Server) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	clientIP := s.getClientIP(r)

	if !ensurePostMethod(w, r, logger, localizer) {
		return
	}

	if !s.ensureSetupCompletionAllowed(w, logger, localizer, clientIP) {
		return
	}

	req, ok := decodeSetupCompleteRequest(w, r, logger, localizer)
	if !ok {
		return
	}

	if !s.ensureValidSetupToken(w, logger, clientIP, req.SetupToken) {
		return
	}

	// Enforce full password policy: length+class, zxcvbn strength,
	// HIBP breach corpus. (Wave 2 / task #86.)
	username := s.config.Auth.DefaultUsername
	policyResult, policyErr := auth.EnforcePasswordPolicy(
		r.Context(),
		req.Password,
		[]string{username},
	)
	if policyErr != nil {
		s.respondPolicyRejection(w, logger, localizer, clientIP, "setup", policyResult)
		return
	}

	// Capture the algorithm of the OLD hash for the audit log before
	// we overwrite it (Wave 2 / task #84).
	previousAlg := string(auth.DetectHashAlgorithm(s.config.Auth.DefaultPasswordHash))

	// Hash the new password (Argon2id).
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T("errors.api.internalError"), "")
		return
	}

	// Update config with new password hash
	// NOTE: Must unlock before Save() - Save() acquires RLock internally (fixes #783, #815)
	s.config.Lock()
	s.config.Auth.DefaultPasswordHash = hash
	s.config.Unlock() // Explicit unlock before Save() to prevent deadlock

	// Create the admin user, or update its password when it already exists.
	// With no user store the config hash below stays the only copy.
	if _, createErr := s.identityUsers.Create(r.Context(), username, hash, "admin"); createErr != nil &&
		!errors.Is(createErr, users.ErrUnavailable) {
		if updateErr := s.identityUsers.UpdatePassword(r.Context(), username, hash); updateErr != nil {
			logger.ErrorContext(r.Context(), "Failed to update user in database", "error", updateErr)
		}
	}

	// Update auth manager (also updates database via UserStore if set)
	s.authManager().UpdatePasswordHash(r.Context(), hash)

	// Save config to disk
	if saveErr := s.config.Save(s.configPath); saveErr != nil {
		logger.ErrorContext(r.Context(), "Failed to save config after setup", "error", saveErr)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T("errors.config.failedToSave"), "")
		return
	}

	if consumeErr := s.setupTokenManager().Consume(); consumeErr != nil {
		logger.WarnContext(r.Context(), "Failed to remove setup token file", "error", consumeErr)
	}

	// Security audit log: initial setup completed (fixes #697)
	// Wave 2 / task #84: include previous_algorithm so password rotations
	// are auditable across the bcrypt → argon2id migration.
	logger.InfoContext(r.Context(), "Initial setup completed - admin password changed",
		"client_ip", clientIP,
		"event", "password_change",
		"result", "accepted",
		"previous_algorithm", previousAlg,
		"setup", true)

	sendJSONResponse(w, logger, http.StatusOK, SetupCompleteResponse{Status: statusSuccess})
}

// respondPolicyRejection translates a PasswordPolicyResult into a
// 400 response and emits a structured password_change audit log.
// Used by both setup completion and recovery completion.
func (s *Server) respondPolicyRejection(
	w http.ResponseWriter,
	logger *slog.Logger,
	localizer *i18n.Localizer,
	clientIP, source string,
	result auth.PasswordPolicyResult,
) {
	logger.Warn("Password change rejected by policy",
		"client_ip", clientIP,
		"event", "password_change",
		"result", "rejected",
		"reject_reason", string(result.Reason),
		"source", source,
		"score", result.Strength.Score,
		"breach_count", result.BreachCount,
		"detail", result.Message)

	i18nKey := "errors.password.weak"
	if result.Reason == auth.PasswordRejectBreached {
		i18nKey = "errors.password.breached"
	}

	sendJSONResponse(w, logger, http.StatusBadRequest, map[string]any{
		"error":        localizer.T(i18nKey),
		"reason":       string(result.Reason),
		"detail":       result.Message,
		"score":        result.Strength.Score,
		statusWarning:  result.Strength.Warning,
		"suggestions":  result.Strength.Suggestions,
		"breach_count": result.BreachCount,
	})
}
