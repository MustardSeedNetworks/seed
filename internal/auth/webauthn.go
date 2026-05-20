// webauthn.go — WebAuthn (passkey) primitives for package auth.
//
// Wraps github.com/go-webauthn/webauthn so the rest of the package
// doesn't need to depend on it directly. We expose:
//
//   - NewWebAuthnManager — construct once at startup with the RP ID,
//     display name, and origins.
//   - WebAuthnUser — the implementation of the library's User
//     interface that the API/database layer wires in.
//   - BeginRegistration / FinishRegistration — ceremony helpers for
//     enrolling a new authenticator.
//   - BeginLogin / FinishLogin — ceremony helpers for authenticating
//     with a previously enrolled authenticator.
//
// All session data is opaque to callers — they're expected to stash it
// in a server-side store keyed by the user's session.

package auth

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// Wave 3 (#85) defaults — see WebAuthnConfig for callers that need to
// override these.
const (
	// webauthnDefaultRPID is the relying-party identifier used when the
	// caller passes an empty value. Localhost is appropriate for the
	// "dev" profile only; production deployments must set RPID to the
	// public hostname.
	webauthnDefaultRPID = "localhost"

	// webauthnDefaultDisplayName is the human-readable RP name shown by
	// browsers/authenticators during the ceremony.
	webauthnDefaultDisplayName = "Seed"

	// webauthnDefaultOrigin is the default trusted origin URL used when
	// the caller passes an empty origin list.
	webauthnDefaultOrigin = "https://localhost:8443"
)

// ErrNoWebAuthnCredentials is returned by BeginLogin when the user has
// not enrolled any authenticators.
var ErrNoWebAuthnCredentials = errors.New("user has no webauthn credentials")

// WebAuthnConfig configures NewWebAuthnManager.
type WebAuthnConfig struct {
	// RPID is the relying-party ID — the public hostname (no scheme,
	// no port). Empty defaults to webauthnDefaultRPID.
	RPID string
	// RPDisplayName is the human-readable display name. Empty defaults
	// to webauthnDefaultDisplayName.
	RPDisplayName string
	// RPOrigins is the list of trusted origin URLs (scheme + host +
	// port). Empty defaults to [webauthnDefaultOrigin].
	RPOrigins []string
}

// WebAuthnManager wraps *webauthn.WebAuthn so the rest of the auth
// package doesn't need to import the library directly.
type WebAuthnManager struct {
	wan *webauthn.WebAuthn
}

// NewWebAuthnManager builds a *WebAuthnManager from the given config.
// It is safe to call once at startup and share the result across
// goroutines.
func NewWebAuthnManager(cfg WebAuthnConfig) (*WebAuthnManager, error) {
	rpID := cfg.RPID
	if rpID == "" {
		rpID = webauthnDefaultRPID
	}
	displayName := cfg.RPDisplayName
	if displayName == "" {
		displayName = webauthnDefaultDisplayName
	}
	origins := cfg.RPOrigins
	if len(origins) == 0 {
		origins = []string{webauthnDefaultOrigin}
	}

	wan, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: displayName,
		RPOrigins:     origins,
	})
	if err != nil {
		return nil, fmt.Errorf("new webauthn: %w", err)
	}
	return &WebAuthnManager{wan: wan}, nil
}

// WebAuthnUser implements the webauthn.User interface. Callers
// construct one of these from the database row + the user's existing
// credentials and pass it into Begin/Finish ceremonies.
//
// ID must be stable across renames — we use the SQLite auto-increment
// row ID encoded big-endian. Name is the username; DisplayName is
// either the username or a friendly label (Seed has no separate
// "display name" today so we duplicate Name).
type WebAuthnUser struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

// WebAuthnID implements webauthn.User.
func (u *WebAuthnUser) WebAuthnID() []byte { return u.ID }

// WebAuthnName implements webauthn.User.
func (u *WebAuthnUser) WebAuthnName() string { return u.Name }

// WebAuthnDisplayName implements webauthn.User.
func (u *WebAuthnUser) WebAuthnDisplayName() string {
	if u.DisplayName == "" {
		return u.Name
	}
	return u.DisplayName
}

// WebAuthnCredentials implements webauthn.User.
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// BeginRegistration starts the registration ceremony for a new
// authenticator. The returned *webauthn.SessionData must be stored
// server-side and passed back to FinishRegistration.
func (m *WebAuthnManager) BeginRegistration(
	user *WebAuthnUser,
) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	if m == nil || m.wan == nil {
		return nil, nil, errors.New("webauthn manager not initialised")
	}
	opts, sess, err := m.wan.BeginRegistration(user)
	if err != nil {
		return nil, nil, fmt.Errorf("begin registration: %w", err)
	}
	return opts, sess, nil
}

// FinishRegistration completes the registration ceremony and returns
// the credential to persist. Callers should then insert that credential
// into the webauthn_credentials table.
func (m *WebAuthnManager) FinishRegistration(
	user *WebAuthnUser,
	sessionData webauthn.SessionData,
	r *http.Request,
) (*webauthn.Credential, error) {
	if m == nil || m.wan == nil {
		return nil, errors.New("webauthn manager not initialised")
	}
	cred, err := m.wan.FinishRegistration(user, sessionData, r)
	if err != nil {
		return nil, fmt.Errorf("finish registration: %w", err)
	}
	return cred, nil
}

// BeginLogin starts the assertion (login) ceremony. The user must
// already have credentials; if not, ErrNoWebAuthnCredentials is
// returned so callers can produce a clean 400 instead of a 500.
func (m *WebAuthnManager) BeginLogin(
	user *WebAuthnUser,
) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	if m == nil || m.wan == nil {
		return nil, nil, errors.New("webauthn manager not initialised")
	}
	if len(user.Credentials) == 0 {
		return nil, nil, ErrNoWebAuthnCredentials
	}
	opts, sess, err := m.wan.BeginLogin(user)
	if err != nil {
		return nil, nil, fmt.Errorf("begin login: %w", err)
	}
	return opts, sess, nil
}

// FinishLogin completes the assertion ceremony and returns the
// matched credential (with the post-ceremony updated sign-count) for
// the caller to persist.
func (m *WebAuthnManager) FinishLogin(
	user *WebAuthnUser,
	sessionData webauthn.SessionData,
	r *http.Request,
) (*webauthn.Credential, error) {
	if m == nil || m.wan == nil {
		return nil, errors.New("webauthn manager not initialised")
	}
	cred, err := m.wan.FinishLogin(user, sessionData, r)
	if err != nil {
		return nil, fmt.Errorf("finish login: %w", err)
	}
	return cred, nil
}
