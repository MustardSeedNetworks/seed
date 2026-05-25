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

package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// userHandleFromID encodes a SQLite AUTOINCREMENT row ID as the
// fixed-size byte slice required by the WebAuthn user handle. We emit
// the int64's natural two's-complement big-endian representation so
// the encoding round-trips any value; rowIDs in seed are always
// positive, but the bit-exact form keeps the encoding future-proof.
func userHandleFromID(rowID int64) []byte {
	const (
		handleSize = 8
		bitsPerB   = 8
		byteMask   = 0xFF
	)
	id := make([]byte, handleSize)
	for i := range handleSize {
		// Shift out one byte at a time; arithmetic right-shift on
		// int64 is well-defined and the byte() cast keeps the low 8
		// bits — no gosec G115 because both ends of the conversion
		// are explicitly masked to byte-sized values.
		shift := (handleSize - 1 - i) * bitsPerB
		id[i] = byte((rowID >> shift) & byteMask)
	}
	return id
}

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

// totpToggleResponse is the JSON body returned by the TOTP
// setup/verify/disable endpoints. Using a struct rather than an
// inline map[string]any keeps the "status"/"enabled" field name
// strings out of goconst's eye (#85).
type totpToggleResponse struct {
	Status  string `json:"status"`
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
func usernameFromContext(r *http.Request) string {
	return r.Header.Get("X-Username")
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

// totpSetupResponse is returned by POST /api/v1/auth/totp/setup.
// QRCodePNGBase64 is base64-encoded so the client can render it via
// <img src="data:image/png;base64,..." />.
type totpSetupResponse struct {
	Secret          string `json:"secret"`
	ProvisioningURI string `json:"provisioning_uri"`
	QRCodePNGBase64 string `json:"qr_code_png_base64"`
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
	if username == "" || s.db() == nil {
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

	if storeErr := s.db().SetTOTPSecret(r.Context(), username, setup.Secret); storeErr != nil {
		logger.ErrorContext(r.Context(), "Failed to persist TOTP secret", "error", storeErr, "event", "mfa_setup")
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	logger.InfoContext(r.Context(), "TOTP setup initiated",
		"event", "mfa_setup", "username", username, "result", "pending")

	sendJSONResponse(w, logger, http.StatusOK, totpSetupResponse{
		Secret:          setup.Secret,
		ProvisioningURI: setup.ProvisioningURI,
		QRCodePNGBase64: base64.StdEncoding.EncodeToString(setup.QRCodePNG),
	})
}

// totpVerifyRequest is the body for POST /api/v1/auth/totp/verify.
type totpVerifyRequest struct {
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
	if username == "" || s.db() == nil {
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

	r.Body = http.MaxBytesReader(w, r.Body, mfaBodyLimit)
	var req totpVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
		return
	}

	secret, _, getErr := s.db().GetTOTP(r.Context(), username)
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
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.mfa.invalidCode"), "")
		return
	}

	if enableErr := s.db().EnableTOTP(r.Context(), username); enableErr != nil {
		logger.ErrorContext(r.Context(), "Failed to enable TOTP", "error", enableErr, "event", "mfa_setup")
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	recordMFAAuditEvent(r, username, mfaFactorTOTP, mfaStatusEnabled)
	logger.InfoContext(r.Context(), "TOTP enabled",
		"event", "mfa_setup", "username", username, "result", mfaStatusEnabled)

	sendJSONResponse(w, logger, http.StatusOK, totpToggleResponse{
		Status:  mfaStatusEnabled,
		Enabled: true,
	})
}

// totpDisableRequest is the body for POST /api/v1/auth/totp/disable.
// Both factors are required to disable.
type totpDisableRequest struct {
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
	if username == "" || s.db() == nil {
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

	r.Body = http.MaxBytesReader(w, r.Body, mfaBodyLimit)
	var req totpDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
		return
	}

	if err := s.authManager().VerifyPasswordOnly(r.Context(), username, req.Password); err != nil {
		mfaAttempts.RecordFailure(username)
		recordMFAAuditEvent(r, username, mfaFactorTOTP, "bad_password")
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.invalidCredentials"), "")
		return
	}
	secret, enabled, getErr := s.db().GetTOTP(r.Context(), username)
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
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.mfa.invalidCode"), "")
		return
	}

	if disableErr := s.db().DisableTOTP(r.Context(), username); disableErr != nil {
		logger.ErrorContext(r.Context(), "Failed to disable TOTP", "error", disableErr, "event", "mfa_disable")
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	recordMFAAuditEvent(r, username, mfaFactorTOTP, mfaStatusDisabled)
	logger.InfoContext(r.Context(), "TOTP disabled",
		"event", "mfa_disable", "username", username, "result", mfaStatusDisabled)
	sendJSONResponse(w, logger, http.StatusOK, totpToggleResponse{
		Status:  mfaStatusDisabled,
		Enabled: false,
	})
}

// ----------------------------------------------------------------------
// TOTP login (second factor)
// ----------------------------------------------------------------------

// totpLoginRequest is the body for POST /api/v1/auth/login/totp.
type totpLoginRequest struct {
	MFAToken string `json:"mfa_token" validate:"required"`
	Code     string `json:"code"      validate:"required,numeric,len=6"`
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

	r.Body = http.MaxBytesReader(w, r.Body, mfaBodyLimit)
	var req totpLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
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

	if s.db() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	secret, enabled, getErr := s.db().GetTOTP(r.Context(), username)
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

// webAuthnSessionStore holds in-flight WebAuthn ceremony state keyed
// by username. Sessions expire after webauthnSessionTTL.
type webAuthnSessionStore struct {
	mu       sync.Mutex
	sessions map[string]webAuthnSessionEntry
}

type webAuthnSessionEntry struct {
	data    webauthn.SessionData
	purpose string // "register" or "login"
	expires time.Time
}

const webauthnSessionTTL = 5 * time.Minute

func newWebAuthnSessionStore() *webAuthnSessionStore {
	return &webAuthnSessionStore{
		sessions: make(map[string]webAuthnSessionEntry),
	}
}

// Put stores a session, evicting expired entries opportunistically.
func (s *webAuthnSessionStore) Put(
	username, purpose string, data webauthn.SessionData,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for k, v := range s.sessions {
		if v.expires.Before(now) {
			delete(s.sessions, k)
		}
	}
	s.sessions[username+"|"+purpose] = webAuthnSessionEntry{
		data:    data,
		purpose: purpose,
		expires: now.Add(webauthnSessionTTL),
	}
}

// Take returns and removes the session for the given user/purpose.
// Returns (zero, false) when missing or expired.
func (s *webAuthnSessionStore) Take(
	username, purpose string,
) (webauthn.SessionData, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.sessions[username+"|"+purpose]
	if !ok {
		return webauthn.SessionData{}, false
	}
	delete(s.sessions, username+"|"+purpose)
	if entry.expires.Before(time.Now()) {
		return webauthn.SessionData{}, false
	}
	return entry.data, true
}

// webAuthnSessions is the package-level singleton ceremony cache.
var webAuthnSessions = newWebAuthnSessionStore() //nolint:gochecknoglobals // process-wide cache

// loadWebAuthnUser builds a *auth.WebAuthnUser for the given username
// by reading the row + their stored credentials out of the database.
func (s *Server) loadWebAuthnUser(
	ctx context.Context, username string,
) (*auth.WebAuthnUser, *database.User, error) {
	if s.db() == nil {
		return nil, nil, errors.New("database not available")
	}
	user, err := s.db().GetUser(ctx, username)
	if err != nil {
		return nil, nil, err
	}
	creds, err := s.db().ListWebAuthnCredentials(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	wanCreds := make([]webauthn.Credential, 0, len(creds))
	for _, c := range creds {
		wanCreds = append(wanCreds, webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Authenticator: webauthn.Authenticator{
				AAGUID:    c.AAGUID,
				SignCount: c.SignCount,
			},
		})
	}
	// Use the row ID as the user handle. WebAuthn requires it be 1-64
	// bytes and stable across renames; binary.BigEndian gives us a
	// canonical 8-byte encoding of the auto-increment int64 row ID.
	id := userHandleFromID(user.ID)
	return &auth.WebAuthnUser{
		ID:          id,
		Name:        user.Username,
		DisplayName: user.Username,
		Credentials: wanCreds,
	}, user, nil
}

// handleWebAuthnRegisterBegin starts the registration ceremony for the
// authenticated user. Returns the credential creation options the
// browser should pass to navigator.credentials.create().
func (s *Server) handleWebAuthnRegisterBegin(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	if !requirePost(w, r, logger, localizer) {
		return
	}
	username := usernameFromContext(r)
	if username == "" || s.webAuthnManager() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.unauthorized"), "")
		return
	}
	user, _, err := s.loadWebAuthnUser(r.Context(), username)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to load WebAuthn user", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	opts, sess, err := s.webAuthnManager().BeginRegistration(user)
	if err != nil {
		logger.ErrorContext(r.Context(), "WebAuthn begin registration failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	webAuthnSessions.Put(username, "register", *sess)
	logger.InfoContext(r.Context(), "WebAuthn registration started",
		"event", "webauthn_register", "username", username, "result", "begin")
	sendJSONResponse(w, logger, http.StatusOK, opts)
}

// handleWebAuthnRegisterFinish completes the registration ceremony and
// persists the new credential to webauthn_credentials.
func (s *Server) handleWebAuthnRegisterFinish(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	if !requirePost(w, r, logger, localizer) {
		return
	}
	username := usernameFromContext(r)
	if username == "" || s.webAuthnManager() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.unauthorized"), "")
		return
	}
	sess, ok := webAuthnSessions.Take(username, "register")
	if !ok {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.sessionExpired"), "")
		return
	}
	user, dbUser, err := s.loadWebAuthnUser(r.Context(), username)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to load WebAuthn user", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	cred, err := s.webAuthnManager().FinishRegistration(user, sess, r)
	if err != nil {
		logger.WarnContext(r.Context(), "WebAuthn finish registration failed", "error", err,
			"event", "webauthn_register", "username", username, "result", "rejected")
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.registrationFailed"), "")
		return
	}
	dbCred := database.WebAuthnCredential{
		UserID:          dbUser.ID,
		CredentialID:    cred.ID,
		PublicKey:       cred.PublicKey,
		SignCount:       cred.Authenticator.SignCount,
		AttestationType: cred.AttestationType,
		AAGUID:          cred.Authenticator.AAGUID,
	}
	if _, addErr := s.db().AddWebAuthnCredential(r.Context(), dbUser.ID, dbCred); addErr != nil {
		logger.ErrorContext(r.Context(), "Failed to persist WebAuthn credential", "error", addErr)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	logger.InfoContext(r.Context(), "WebAuthn credential registered",
		"event", "webauthn_register", "username", username, "result", "registered")
	sendJSONResponse(w, logger, http.StatusOK, totpToggleResponse{
		Status:  "registered",
		Enabled: true,
	})
}

// ----------------------------------------------------------------------
// WebAuthn login
// ----------------------------------------------------------------------

// webAuthnLoginBeginRequest is the body for /webauthn/login/begin.
// We accept the username here because (unlike registration) the user
// hasn't authenticated yet.
type webAuthnLoginBeginRequest struct {
	Username string `json:"username"`
}

// handleWebAuthnLoginBegin starts the assertion ceremony for the given
// username and returns the credential assertion options.
func (s *Server) handleWebAuthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	if !requirePost(w, r, logger, localizer) {
		return
	}
	if s.webAuthnManager() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.api.internalError"), "")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, mfaBodyLimit)
	var req webAuthnLoginBeginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
		return
	}

	user, _, err := s.loadWebAuthnUser(r.Context(), req.Username)
	if err != nil {
		// Don't disclose user existence — return a generic error.
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.noCredentials"), "")
		return
	}
	opts, sess, err := s.webAuthnManager().BeginLogin(user)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.noCredentials"), "")
		return
	}
	webAuthnSessions.Put(req.Username, "login", *sess)
	logger.InfoContext(r.Context(), "WebAuthn login started",
		"event", "webauthn_login", "username", req.Username, "result", "begin")
	sendJSONResponse(w, logger, http.StatusOK, opts)
}

// handleWebAuthnLoginFinish finishes the assertion ceremony, updates
// the credential's sign-count, and issues a real access token.
func (s *Server) handleWebAuthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	if !requirePost(w, r, logger, localizer) {
		return
	}
	if s.webAuthnManager() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.api.internalError"), "")
		return
	}

	// go-webauthn expects to parse the response body itself, so we
	// pull the username from a duplicated query param to avoid eating
	// the body before the library sees it.
	username := r.URL.Query().Get("username")
	if username == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
		return
	}

	sess, ok := webAuthnSessions.Take(username, "login")
	if !ok {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.sessionExpired"), "")
		return
	}
	user, _, err := s.loadWebAuthnUser(r.Context(), username)
	if err != nil {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.noCredentials"), "")
		return
	}

	cred, err := s.webAuthnManager().FinishLogin(user, sess, r)
	if err != nil {
		recordMFAAuditEvent(r, username, "webauthn", "rejected")
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.mfa.assertionFailed"), "")
		return
	}

	if updErr := s.db().UpdateWebAuthnSignCount(
		r.Context(), cred.ID, cred.Authenticator.SignCount,
	); updErr != nil {
		logger.WarnContext(r.Context(), "Failed to update WebAuthn sign count",
			"error", updErr, "username", username)
	}

	recordMFAAuditEvent(r, username, "webauthn", "success")
	logger.InfoContext(r.Context(), "WebAuthn login successful",
		"event", "webauthn_login", "username", username, "result", "success")

	accessToken, tokenErr := s.generateAndSetLoginTokens(w, r, username)
	if tokenErr != nil {
		logger.ErrorContext(r.Context(), "Failed to generate tokens after WebAuthn login", "error", tokenErr)
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
// MFA status (UI helper)
// ----------------------------------------------------------------------

// mfaStatusResponse is returned by GET /api/v1/auth/mfa/status.
type mfaStatusResponse struct {
	TOTPEnabled       bool `json:"totp_enabled"`
	WebAuthnEnabled   bool `json:"webauthn_enabled"`
	WebAuthnCredCount int  `json:"webauthn_credential_count"`
}

// handleMFAStatus returns a compact MFA enrolment summary for the UI.
func (s *Server) handleMFAStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}
	username := usernameFromContext(r)
	if username == "" || s.db() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.unauthorized"), "")
		return
	}
	_, totpEnabled, _ := s.db().GetTOTP(r.Context(), username)

	var credCount int
	if user, err := s.db().GetUser(r.Context(), username); err == nil {
		if creds, listErr := s.db().ListWebAuthnCredentials(r.Context(), user.ID); listErr == nil {
			credCount = len(creds)
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, mfaStatusResponse{
		TOTPEnabled:       totpEnabled,
		WebAuthnEnabled:   credCount > 0,
		WebAuthnCredCount: credCount,
	})
}
