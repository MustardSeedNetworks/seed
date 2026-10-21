package api

// handlers_webauthn.go holds the passkey endpoints: enrolment for a signed-in
// user and passwordless sign-in. foundation/pkg/passkey owns the ceremonies and
// the fleet's verification policy (discoverable credentials, user
// verification, single-use browser-bound challenges); this file owns seed's
// routes, the cookie that binds a ceremony to its browser, and storage. The
// TOTP endpoints and the shared MFA helpers are in handlers_mfa.go.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/MustardSeedNetworks/foundation/pkg/passkey"

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

// passkeyCeremonyTTL matches foundation's ceremony timeout: the cookie lives
// no longer than the challenge it names.
const passkeyCeremonyTTL = 5 * time.Minute

// passkeyCookieScope returns a ceremony cookie's name and path. Each purpose
// has its own cookie scoped to its own finish route, so it travels with no
// other request.
func passkeyCookieScope(purpose passkey.Purpose) (string, string) {
	if purpose == passkey.Registration {
		return "seed_passkey_register", APIVersionPrefix + "/auth/webauthn/register/finish"
	}
	return "seed_passkey_login", APIVersionPrefix + "/auth/webauthn/login/finish"
}

func setPasskeyCookie(w http.ResponseWriter, purpose passkey.Purpose, value string, maxAge int) {
	name, path := passkeyCookieScope(purpose)
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// startPasskeyCeremony keeps the session server-side under a random browser
// secret and hands the browser only "<ceremony id>.<secret>" in an HttpOnly
// cookie. account is the user handle for enrolment and empty for sign-in.
func (s *Server) startPasskeyCeremony(
	w http.ResponseWriter, purpose passkey.Purpose, session *passkey.Session, account string,
) error {
	const secretBytes = 32
	secret := make([]byte, secretBytes)
	_, _ = rand.Read(secret) // crypto/rand.Read never returns an error
	browser := base64.RawURLEncoding.EncodeToString(secret)
	id, err := s.passkeyCeremonies.Add(*session, passkey.Binding{
		Browser: browser, AccountID: account, Purpose: purpose,
	})
	if err != nil {
		return err
	}
	setPasskeyCookie(w, purpose, id+"."+browser, int(passkeyCeremonyTTL.Seconds()))
	return nil
}

// takePasskeyCeremony consumes the ceremony the request's cookie names. The
// cookie is spent either way: a failed finish needs a fresh challenge.
func (s *Server) takePasskeyCeremony(
	w http.ResponseWriter, r *http.Request, purpose passkey.Purpose, account string,
) (passkey.Session, bool) {
	setPasskeyCookie(w, purpose, "", -1)
	name, _ := passkeyCookieScope(purpose)
	cookie, err := r.Cookie(name)
	if err != nil {
		return passkey.Session{}, false
	}
	id, browser, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		return passkey.Session{}, false
	}
	return s.passkeyCeremonies.Take(id, passkey.Binding{
		Browser: browser, AccountID: account, Purpose: purpose,
	})
}

// passkeyUser is a seed account as go-webauthn sees it: the row-ID user
// handle, the username, and the complete stored credentials.
type passkeyUser struct {
	handle      []byte
	name        string
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte                         { return u.handle }
func (u *passkeyUser) WebAuthnName() string                       { return u.name }
func (u *passkeyUser) WebAuthnDisplayName() string                { return u.name }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// loadPasskeyUser builds the account's ceremony view through the users
// use-case and the MFA store.
func (s *Server) loadPasskeyUser(ctx context.Context, username string) (*passkeyUser, *users.User, error) {
	if !s.identityMFA.Available() {
		return nil, nil, errors.New("database not available")
	}
	user, err := s.identityUsers.Get(ctx, username)
	if err != nil {
		return nil, nil, err
	}
	stored, err := s.identityMFA.ListWebAuthnCredentials(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	credentials := make([]webauthn.Credential, 0, len(stored))
	for _, c := range stored {
		credential, decodeErr := passkey.UnmarshalCredential(c.Record)
		if decodeErr != nil {
			return nil, nil, decodeErr
		}
		credentials = append(credentials, credential)
	}
	return &passkeyUser{
		handle: userHandleFromID(user.ID), name: user.Username, credentials: credentials,
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
	if username == "" || s.passkeys == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.unauthorized"), "")
		return
	}
	user, _, err := s.loadPasskeyUser(r.Context(), username)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to load passkey user", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	opts, session, err := s.passkeys.BeginRegistration(user)
	if err == nil {
		err = s.startPasskeyCeremony(w, passkey.Registration, session, string(user.handle))
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "WebAuthn begin registration failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	logger.InfoContext(r.Context(), "WebAuthn registration started",
		"event", "webauthn_register", "username", username, "result", "begin")
	sendJSONResponse(w, logger, http.StatusOK, newWebAuthnCreationOptions(opts, user.handle))
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
	if username == "" || s.passkeys == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.auth.unauthorized"), "")
		return
	}
	user, dbUser, err := s.loadPasskeyUser(r.Context(), username)
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to load passkey user", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}
	session, ok := s.takePasskeyCeremony(w, r, passkey.Registration, string(user.handle))
	if !ok {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.sessionExpired"), "")
		return
	}
	credential, err := s.passkeys.FinishRegistration(user, session, r)
	if err != nil {
		logger.WarnContext(r.Context(), "WebAuthn finish registration failed", "error", err,
			"event", "webauthn_register", "username", username, "result", "rejected")
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.registrationFailed"), "")
		return
	}
	record, err := passkey.MarshalCredential(*credential)
	if err == nil {
		_, err = s.identityMFA.AddWebAuthnCredential(r.Context(), dbUser.ID, mfa.WebAuthnCredential{
			CredentialID: credential.ID, Record: record,
		})
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to persist WebAuthn credential", "error", err)
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

// handleWebAuthnLoginBegin starts a username-free sign-in: the browser offers
// whichever discoverable passkey it holds for this relying party.
func (s *Server) handleWebAuthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	if !requirePost(w, r, logger, localizer) {
		return
	}
	if s.handleLoginRateLimited(w, logger, localizer, s.getClientIP(r)) {
		return
	}
	if s.passkeys == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.api.internalError"), "")
		return
	}
	opts, session, err := s.passkeys.BeginLogin()
	if err == nil {
		err = s.startPasskeyCeremony(w, passkey.Login, session, "")
	}
	if err != nil {
		// The ceremony store is bounded; a full one is a transient refusal.
		logger.WarnContext(r.Context(), "WebAuthn begin login failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.api.internalError"), "")
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, newWebAuthnRequestOptions(opts))
}

// handleWebAuthnLoginFinish verifies the assertion, persists the updated
// credential record, and issues a session.
func (s *Server) handleWebAuthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	if !requirePost(w, r, logger, localizer) {
		return
	}
	clientIP := s.getClientIP(r)
	if s.handleLoginRateLimited(w, logger, localizer, clientIP) {
		return
	}
	if s.passkeys == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.api.internalError"), "")
		return
	}
	session, ok := s.takePasskeyCeremony(w, r, passkey.Login, "")
	if !ok {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeBadRequest, localizer.T("errors.mfa.sessionExpired"), "")
		return
	}

	var username string
	resolve := func(credentialID, userHandle []byte) (webauthn.User, error) {
		owner, err := s.identityMFA.WebAuthnCredentialOwner(r.Context(), credentialID)
		if err != nil {
			return nil, err
		}
		user, _, err := s.loadPasskeyUser(r.Context(), owner)
		if err != nil {
			return nil, err
		}
		if string(user.handle) != string(userHandle) {
			return nil, mfa.ErrCredentialNotFound
		}
		username = owner
		return user, nil
	}
	_, credential, err := s.passkeys.FinishLogin(resolve, session, r)
	if err != nil {
		recordMFAAuditEvent(r, username, "webauthn", "rejected")
		s.loginRateLimiter().RecordAttempt(clientIP, false)
		sendErrorResponseWithDetails(w, logger, http.StatusUnauthorized,
			ErrCodeUnauthorized, localizer.T("errors.mfa.assertionFailed"), "")
		return
	}

	s.completeWebAuthnLogin(w, r, username, credential)
}

func (s *Server) completeWebAuthnLogin(
	w http.ResponseWriter, r *http.Request, username string, credential *webauthn.Credential,
) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	record, err := passkey.MarshalCredential(*credential)
	if err == nil {
		err = s.identityMFA.UpdateWebAuthnCredential(r.Context(), credential.ID, record)
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to update WebAuthn credential",
			"error", err, "username", username)
		recordMFAAuditEvent(r, username, "webauthn", "rejected")
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.api.internalError"), "")
		return
	}

	recordMFAAuditEvent(r, username, "webauthn", "success")
	s.loginRateLimiter().RecordAttempt(s.getClientIP(r), true)
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
