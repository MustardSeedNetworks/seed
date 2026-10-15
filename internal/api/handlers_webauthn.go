package api

// handlers_webauthn.go holds the WebAuthn passkey endpoints: registration
// for an authenticated user and the login second factor. The TOTP
// endpoints and the shared MFA helpers are in handlers_mfa.go.

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/mfa"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
	"github.com/MustardSeedNetworks/seed/internal/logging"
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
// through the users use-case and the MFA store.
func (s *Server) loadWebAuthnUser(
	ctx context.Context, username string,
) (*auth.WebAuthnUser, *users.User, error) {
	if !s.identityMFA.Available() {
		return nil, nil, errors.New("database not available")
	}
	user, err := s.identityUsers.Get(ctx, username)
	if err != nil {
		return nil, nil, err
	}
	creds, err := s.identityMFA.ListWebAuthnCredentials(ctx, user.ID)
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
	sendJSONResponse(w, logger, http.StatusOK, newWebAuthnCreationOptions(opts, user.ID))
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
	dbCred := mfa.WebAuthnCredential{
		UserID:          dbUser.ID,
		CredentialID:    cred.ID,
		PublicKey:       cred.PublicKey,
		SignCount:       cred.Authenticator.SignCount,
		AttestationType: cred.AttestationType,
		AAGUID:          cred.Authenticator.AAGUID,
	}
	if _, addErr := s.identityMFA.AddWebAuthnCredential(r.Context(), dbUser.ID, dbCred); addErr != nil {
		logger.ErrorContext(r.Context(), "Failed to persist WebAuthn credential", "error", addErr)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	logger.InfoContext(r.Context(), "WebAuthn credential registered",
		"event", "webauthn_register", "username", username, "result", "registered")
	sendJSONResponse(w, logger, http.StatusOK, MFAToggleResponse{
		Status:  "registered",
		Enabled: true,
	})
}

// ----------------------------------------------------------------------
// WebAuthn login
// ----------------------------------------------------------------------

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

	var req WebAuthnLoginBeginRequest
	if !decodeJSONStrictLocalized(w, r, &req, mfaBodyLimit, logger, localizer) {
		return
	}
	if req.Username == "" {
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
	sendJSONResponse(w, logger, http.StatusOK, newWebAuthnRequestOptions(opts))
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

	s.completeWebAuthnLogin(w, r, username, cred)
}

func (s *Server) completeWebAuthnLogin(
	w http.ResponseWriter, r *http.Request, username string, cred *webauthn.Credential,
) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	if updErr := s.identityMFA.UpdateWebAuthnSignCount(
		r.Context(), cred.ID, cred.Authenticator.SignCount,
	); updErr != nil {
		logger.ErrorContext(r.Context(), "Failed to update WebAuthn sign count",
			"error", updErr, "username", username)
		recordMFAAuditEvent(r, username, "webauthn", "rejected")
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
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
