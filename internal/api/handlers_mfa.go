// handlers_mfa.go — TOTP + WebAuthn endpoints for the Wave 3 (#85)
// multi-factor authentication flow.
//
// The MFA endpoints are split across two phases:
//
//  1. Enrolment (authenticated user, after initial login)
//     POST /api/v1/auth/totp/setup    → returns secret + provisioning URI + QR PNG
//     POST /api/v1/auth/totp/verify   → confirms a code from the candidate secret, enables TOTP
//     POST /api/v1/auth/totp/disable  → password + code required, clears the secret
//     POST /api/v1/auth/webauthn/register/begin
//     POST /api/v1/auth/webauthn/register/finish
//
//  2. Login second factor (no JWT yet, only an mfa_pending token)
//     POST /api/v1/auth/login/totp    → trades mfa_pending + code for real JWT
//     POST /api/v1/auth/webauthn/login/begin
//     POST /api/v1/auth/webauthn/login/finish
//
// Rate-limit policy: TOTP verify and login/totp share a per-account
// limiter (5 attempts / 60s) to defeat brute-force of the code space.
//
// The WebAuthn handlers are in handlers_webauthn.go.

package api

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// TOTP/WebAuthn enrolment + login constants.
const (
	// mfaCodeAttemptLimit is the maximum TOTP verifications allowed
	// per account inside mfaCodeAttemptWindow before we block. 5 / 60s
	// gives us roughly the same protection as the password limiter but
	// scoped per-user instead of per-IP (so an attacker can't spread
	// the attempts across IPs).
	mfaCodeAttemptLimit  = 5
	mfaCodeAttemptWindow = time.Minute

	// mfaBodyLimit caps the request body size for MFA endpoints. The
	// JSON payloads are small (well under 32 KiB even for WebAuthn
	// assertions) so 64 KiB is a generous ceiling.
	mfaBodyLimit int64 = 64 * 1024

	// mfaFactorTOTP is the factor name used in mfa_attempt audit logs.
	mfaFactorTOTP = "totp"

	// mfaStatusEnabled is the JSON status string returned by TOTP
	// setup/verify on success.
	mfaStatusEnabled = "enabled"

	// mfaStatusDisabled is the JSON status string returned by TOTP
	// disable on success.
	mfaStatusDisabled = "disabled"
)

// MFAToggleResponse is the JSON body returned when a second factor is turned
// on or off: TOTP verify/disable and WebAuthn register/finish.
type MFAToggleResponse struct {
	Status  string `json:"status"  jsonschema:"enum=enabled,enum=disabled,enum=registered"`
	Enabled bool   `json:"enabled"`
}

// mfaAttemptStore is a per-account sliding-window rate limiter for
// TOTP code verifications. It deliberately lives in-memory: TOTP brute
// force protection is best-effort and the limiter resets across
// restarts, which is acceptable.
//
// Only failed attempts count toward the budget so legitimate users
// aren't locked out by intermittent network retries; the brute-force
// threat model is that an attacker can submit > mfaCodeAttemptLimit
// guesses per minute, which is precisely what we want to bound.
type mfaAttemptStore struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

func newMFAAttemptStore() *mfaAttemptStore {
	return &mfaAttemptStore{failures: make(map[string][]time.Time)}
}

// Allow returns true iff the account has fewer than mfaCodeAttemptLimit
// recorded failures in the current window. It does NOT record a new
// attempt — call RecordFailure after a failed verification.
func (s *mfaAttemptStore) Allow(username string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-mfaCodeAttemptWindow)
	pruned := s.failures[username][:0]
	for _, ts := range s.failures[username] {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}
	s.failures[username] = pruned
	return len(pruned) < mfaCodeAttemptLimit
}

// RecordFailure marks one more failed verification for the account.
func (s *mfaAttemptStore) RecordFailure(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures[username] = append(s.failures[username], time.Now())
}

// Reset clears all recorded failures (test helper).
func (s *mfaAttemptStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures = make(map[string][]time.Time)
}

// sendReauthRejected answers a wrong password or code on a route that
// already holds a valid session (TOTP verify and disable). It is a 403, not a
// 401: the client reads 401 as an expired session, refreshes, replays the
// request (spending a second attempt against mfaAttempts) and then signs the
// user out over a typo (#2731). The code tells the UI which factor was wrong.
func sendReauthRejected(w http.ResponseWriter, logger *slog.Logger, code, message string) {
	sendErrorResponseWithDetails(w, logger, http.StatusForbidden, code, message, "")
}

// mfaAttempts is a package-level singleton used by all MFA handlers.
// It's safe for concurrent use.
var mfaAttempts = newMFAAttemptStore() //nolint:gochecknoglobals // process-wide rate limit

// requirePost rejects anything that isn't POST and returns false; the
// response has already been sent on rejection.
func requirePost(
	w http.ResponseWriter, r *http.Request,
	logger *slog.Logger, localizer *i18n.Localizer,
) bool {
	if r.Method == http.MethodPost {
		return true
	}
	sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
		ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
	return false
}

// usernameFromContext returns the authenticated username from the
// request, or empty if no auth middleware ran (e.g. login endpoints
// that haven't completed yet).
//
// #2632: this read X-Username, a header the caller could set. Any route that
// reached a handler without passing the JWT middleware — /sso/settings and
// /sso/update did, under the old bypass prefix — handed the role gate an
// identity the client had chosen.
func usernameFromContext(r *http.Request) string {
	return auth.UsernameFromContext(r.Context())
}

// recordMFAAuditEvent emits a structured audit log entry for an MFA
// attempt. We log all attempts (success + failure) to give operators
// a clean signal of brute-force / lockout events.
func recordMFAAuditEvent(
	r *http.Request, username, factor, result string,
) {
	logging.FromContext(r.Context()).InfoContext(r.Context(), "MFA attempt",
		"event", "mfa_attempt",
		"username", username,
		"factor", factor,
		"result", result,
	)
}

// ----------------------------------------------------------------------
// TOTP enrolment
// ----------------------------------------------------------------------

// TOTPSetupResponse is returned by POST /api/v1/auth/totp/setup.
// QRCodePNGBase64 is base64-encoded so the client can render it via
// <img src="data:image/png;base64,..." />.
type TOTPSetupResponse struct {
	Secret          string `json:"secret"`
	ProvisioningURI string `json:"provisioningUri"`
	QRCodePNGBase64 string `json:"qrCodePngBase64"`
}

// handleTOTPSetup generates a fresh TOTP secret for the authenticated
// user and persists it (with totp_enabled = 0). The client is then
// expected to scan the QR code in an authenticator app and POST a code
// to /verify to complete enrolment.
func (s *Server) handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if !requirePost(w, r, logger, localizer) {
		return
	}

	username := usernameFromContext(r)
	if username == "" || !s.identityMFA.Available() {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.unauthorized"), "")
		return
	}

	setup, err := auth.GenerateTOTPSecret(username, "Seed")
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to generate TOTP secret", "error", err, "event", "mfa_setup")
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	if storeErr := s.identityMFA.SetTOTPSecret(r.Context(), username, setup.Secret); storeErr != nil {
		logger.ErrorContext(r.Context(), "Failed to persist TOTP secret", "error", storeErr, "event", "mfa_setup")
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	logger.InfoContext(r.Context(), "TOTP setup initiated",
		"event", "mfa_setup", "username", username, "result", "pending")

	sendJSONResponse(w, logger, http.StatusOK, TOTPSetupResponse{
		Secret:          setup.Secret,
		ProvisioningURI: setup.ProvisioningURI,
		QRCodePNGBase64: base64.StdEncoding.EncodeToString(setup.QRCodePNG),
	})
}

// TOTPVerifyRequest is the body for POST /api/v1/auth/totp/verify.
type TOTPVerifyRequest struct {
	Code string `json:"code" validate:"required,numeric,len=6"`
}

// handleTOTPVerify confirms the user can derive a valid code from the
// candidate secret stored by /setup. On success it flips totp_enabled
// to 1; subsequent logins will then require a code.
func (s *Server) handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if !requirePost(w, r, logger, localizer) {
		return
	}
	username := usernameFromContext(r)
	if username == "" || !s.identityMFA.Available() {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.unauthorized"), "")
		return
	}

	if !mfaAttempts.Allow(username) {
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "rate_limited")
		sendErrorResponseWithDetails(w, logger, http.StatusTooManyRequests,
			ErrCodeRateLimit, localizer.T("errors.api.rateLimitExceeded"), "")
		return
	}

	var req TOTPVerifyRequest
	if !decodeJSONStrictLocalizedWith(w, r, &req, mfaBodyLimit,
		logger, localizer, "username", username, "factor", mfaFactorTOTP) {
		return
	}

	secret, _, getErr := s.identityMFA.GetTOTP(r.Context(), username)
	if getErr != nil || secret == "" {
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "no_secret")
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.notEnrolled"), "")
		return
	}

	valid, err := auth.VerifyTOTP(secret, req.Code)
	if err != nil || !valid {
		mfaAttempts.RecordFailure(username)
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "rejected")
		sendReauthRejected(w, logger, ErrCodeInvalidMFACode, localizer.T("errors.mfa.invalidCode"))
		return
	}

	if enableErr := s.identityMFA.EnableTOTP(r.Context(), username); enableErr != nil {
		logger.ErrorContext(r.Context(), "Failed to enable TOTP", "error", enableErr, "event", "mfa_setup")
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	recordMFAAuditEvent(r, username, mfaFactorTOTP, mfaStatusEnabled)
	logger.InfoContext(r.Context(), "TOTP enabled",
		"event", "mfa_setup", "username", username, "result", mfaStatusEnabled)

	sendJSONResponse(w, logger, http.StatusOK, MFAToggleResponse{
		Status:  mfaStatusEnabled,
		Enabled: true,
	})
}

// TOTPDisableRequest is the body for POST /api/v1/auth/totp/disable.
// Both factors are required to disable.
type TOTPDisableRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

// handleTOTPDisable removes the TOTP enrolment for the authenticated
// user. Requires the current password AND a current code as both
// factors to defeat session hijacking attacks.
func (s *Server) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if !requirePost(w, r, logger, localizer) {
		return
	}
	username := usernameFromContext(r)
	if username == "" || !s.identityMFA.Available() {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.unauthorized"), "")
		return
	}

	if !mfaAttempts.Allow(username) {
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "rate_limited")
		sendErrorResponseWithDetails(w, logger, http.StatusTooManyRequests,
			ErrCodeRateLimit, localizer.T("errors.api.rateLimitExceeded"), "")
		return
	}

	var req TOTPDisableRequest
	if !decodeJSONStrictLocalizedWith(w, r, &req, mfaBodyLimit,
		logger, localizer, "username", username, "factor", mfaFactorTOTP) {
		return
	}

	if err := s.authManager().VerifyPasswordOnly(r.Context(), username, req.Password); err != nil {
		mfaAttempts.RecordFailure(username)
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "bad_password")
		sendReauthRejected(w, logger, ErrCodePasswordInvalid, localizer.T("errors.auth.invalidCredentials"))
		return
	}
	secret, enabled, getErr := s.identityMFA.GetTOTP(r.Context(), username)
	if getErr != nil || !enabled || secret == "" {
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "not_enrolled")
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.notEnrolled"), "")
		return
	}
	valid, err := auth.VerifyTOTP(secret, req.Code)
	if err != nil || !valid {
		mfaAttempts.RecordFailure(username)
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "bad_code")
		sendReauthRejected(w, logger, ErrCodeInvalidMFACode, localizer.T("errors.mfa.invalidCode"))
		return
	}

	if disableErr := s.identityMFA.DisableTOTP(r.Context(), username); disableErr != nil {
		logger.ErrorContext(r.Context(), "Failed to disable TOTP", "error", disableErr, "event", "mfa_disable")
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	recordMFAAuditEvent(r, username, mfaFactorTOTP, mfaStatusDisabled)
	logger.InfoContext(r.Context(), "TOTP disabled",
		"event", "mfa_disable", "username", username, "result", mfaStatusDisabled)
	sendJSONResponse(w, logger, http.StatusOK, MFAToggleResponse{
		Status:  mfaStatusDisabled,
		Enabled: false,
	})
}

// ----------------------------------------------------------------------
// TOTP login (second factor)
// ----------------------------------------------------------------------

// TOTPLoginRequest is the body for POST /api/v1/auth/login/totp.
type TOTPLoginRequest struct {
	MFAToken string `json:"mfaToken" validate:"required"`
	Code     string `json:"code"     validate:"required,numeric,len=6"`
}

// handleLoginTOTP trades an mfa_pending token + a valid TOTP code for
// a real access token. The mfa_pending token is single-use in the
// sense that it can't be exchanged for the same access token twice
// (the new access token bumps token_version internally? no — but the
// MFA token expires in 5 min). We additionally apply the per-account
// rate limiter to defeat brute-force of the 6-digit code space.
func (s *Server) handleLoginTOTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if !requirePost(w, r, logger, localizer) {
		return
	}

	var req TOTPLoginRequest
	if !decodeJSONStrictLocalizedWith(w, r, &req, mfaBodyLimit,
		logger, localizer, "factor", mfaFactorTOTP) {
		return
	}

	username, err := s.authManager().ValidateMFAPendingToken(req.MFAToken)
	if err != nil {
		recordMFAAuditEvent(r, "", mfaFactorTOTP, "bad_mfa_token")
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.mfa.invalidToken"), "")
		return
	}

	if !mfaAttempts.Allow(username) {
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "rate_limited")
		sendErrorResponseWithDetails(w, logger, http.StatusTooManyRequests,
			ErrCodeRateLimit, localizer.T("errors.api.rateLimitExceeded"), "")
		return
	}

	if !s.identityMFA.Available() {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	secret, enabled, getErr := s.identityMFA.GetTOTP(r.Context(), username)
	if getErr != nil || !enabled || secret == "" {
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "not_enrolled")
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.mfa.notEnrolled"), "")
		return
	}
	valid, vErr := auth.VerifyTOTP(secret, req.Code)
	if vErr != nil || !valid {
		mfaAttempts.RecordFailure(username)
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "rejected")
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.mfa.invalidCode"), "")
		return
	}

	recordMFAAuditEvent(r, username, mfaFactorTOTP, "success")
	logger.InfoContext(r.Context(), "MFA login successful",
		"event", "auth.login.success", "username", username, "factor", mfaFactorTOTP)

	accessToken, tokenErr := s.generateAndSetLoginTokens(w, r, username)
	if tokenErr != nil {
		logger.ErrorContext(r.Context(), "Failed to generate tokens after MFA", "error", tokenErr)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, LoginResponse{
		Token:   accessToken,
		Expires: time.Now().Add(auth.AccessTokenDuration).Unix(),
	})
}

// ----------------------------------------------------------------------
// WebAuthn registration
// ----------------------------------------------------------------------

// MFAStatusResponse is returned by GET /api/v1/auth/mfa/status.
type MFAStatusResponse struct {
	TOTPEnabled       bool `json:"totpEnabled"`
	WebAuthnEnabled   bool `json:"webauthnEnabled"`
	WebAuthnCredCount int  `json:"webauthnCredentialCount"`
}

// handleMFAStatus returns a compact MFA enrolment summary for the UI.
func (s *Server) handleMFAStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	username := usernameFromContext(r)
	if username == "" || !s.identityMFA.Available() {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.unauthorized"), "")
		return
	}
	_, totpEnabled, _ := s.identityMFA.GetTOTP(r.Context(), username)

	var credCount int
	if user, err := s.identityUsers.Get(r.Context(), username); err == nil {
		if creds, listErr := s.identityMFA.ListWebAuthnCredentials(r.Context(), user.ID); listErr == nil {
			credCount = len(creds)
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, MFAStatusResponse{
		TOTPEnabled:       totpEnabled,
		WebAuthnEnabled:   credCount > 0,
		WebAuthnCredCount: credCount,
	})
}
