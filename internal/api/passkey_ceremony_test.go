package api_test

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/foundation/pkg/passkey"

	"github.com/MustardSeedNetworks/seed/internal/api"
)

const (
	passkeyRPID   = "localhost"
	passkeyOrigin = "https://localhost:8443"
)

func newPasskeyFixture(t *testing.T) *mfaTestFixture {
	t.Helper()
	f := newMFAFixture(t)
	service, err := passkey.New(passkey.Config{RPID: passkeyRPID, Origin: passkeyOrigin, DisplayName: "Seed"})
	require.NoError(t, err)
	f.server.SetPasskeyServiceForTest(service)
	return f
}

// postCeremony posts a finish body with the ceremony cookie the begin route
// set, as the browser would.
func (f *mfaTestFixture) postCeremony(
	t *testing.T, path string, body []byte, token string, cookie *http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Csrf-Token", f.csrfToken(t, token))
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	return w
}

func ceremonyCookie(t *testing.T, w *httptest.ResponseRecorder, name, path string) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			assert.Equal(t, path, c.Path)
			assert.True(t, c.HttpOnly && c.Secure, "ceremony cookie must be HttpOnly and Secure")
			assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
			return c
		}
	}
	t.Fatalf("no %s cookie in %v", name, w.Result().Cookies())
	return nil
}

// enrolPasskey runs registration through the production routes and returns
// the authenticator holding the new credential.
func enrolPasskey(t *testing.T, f *mfaTestFixture) *softAuthenticator {
	t.Helper()
	w := f.post(t, "/api/v1/auth/webauthn/register/begin", struct{}{}, f.token)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	cookie := ceremonyCookie(t, w, "seed_passkey_register", "/api/v1/auth/webauthn/register/finish")
	var creation api.WebAuthnCreationOptionsResponse
	decodeStrict(t, w, &creation)
	assert.Equal(t, "admin", creation.PublicKey.User.Name)
	assert.Equal(t, "required", creation.PublicKey.AuthenticatorSelection.ResidentKey)
	assert.Equal(t, "required", creation.PublicKey.AuthenticatorSelection.UserVerification)
	handle, err := base64.RawURLEncoding.DecodeString(creation.PublicKey.User.ID)
	require.NoError(t, err)

	authenticator := newSoftAuthenticator(t, passkeyRPID, passkeyOrigin)
	body := authenticator.register(t, creation.PublicKey.Challenge, handle)
	w = f.postCeremony(t, "/api/v1/auth/webauthn/register/finish", body, f.token, cookie)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// The challenge is single-use: replaying the same response fails.
	w = f.postCeremony(t, "/api/v1/auth/webauthn/register/finish", body, f.token, cookie)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	return authenticator
}

func beginPasskeyLogin(t *testing.T, f *mfaTestFixture) (string, *http.Cookie) {
	t.Helper()
	w := f.post(t, "/api/v1/auth/webauthn/login/begin", nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	cookie := ceremonyCookie(t, w, "seed_passkey_login", "/api/v1/auth/webauthn/login/finish")
	var request api.WebAuthnRequestOptionsResponse
	decodeStrict(t, w, &request)
	assert.Equal(t, "required", request.PublicKey.UserVerification)
	return request.PublicKey.Challenge, cookie
}

func TestPasskeyRegisterThenSignIn(t *testing.T) {
	f := newPasskeyFixture(t)
	authenticator := enrolPasskey(t, f)
	user, err := f.db.GetUser(t.Context(), "admin")
	require.NoError(t, err)

	challenge, cookie := beginPasskeyLogin(t, f)
	body := authenticator.assert(t, challenge, authenticator.handle)
	w := f.postCeremony(t, "/api/v1/auth/webauthn/login/finish", body, "", cookie)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var login api.LoginResponse
	decodeStrict(t, w, &login)
	claims, err := f.server.AuthManager().ValidateToken(t.Context(), login.Token)
	require.NoError(t, err)
	assert.Equal(t, "admin", claims.Username)

	stored, err := f.db.ListWebAuthnCredentials(t.Context(), user.ID)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	credential, err := passkey.UnmarshalCredential(stored[0].Record)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), credential.Authenticator.SignCount, "the updated record is persisted")
	assert.NotNil(t, stored[0].LastUsedAt)

	// Replaying the signed assertion against the spent ceremony fails.
	w = f.postCeremony(t, "/api/v1/auth/webauthn/login/finish", body, "", cookie)
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestPasskeySignInRejectsForeignCeremonyAndHandle(t *testing.T) {
	f := newPasskeyFixture(t)
	authenticator := enrolPasskey(t, f)

	t.Run("browser secret from another browser", func(t *testing.T) {
		challenge, cookie := beginPasskeyLogin(t, f)
		forged := *cookie
		forged.Value = cookie.Value[:len(cookie.Value)-1] + "x"
		body := authenticator.assert(t, challenge, authenticator.handle)
		w := f.postCeremony(t, "/api/v1/auth/webauthn/login/finish", body, "", &forged)
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	t.Run("no ceremony cookie", func(t *testing.T) {
		challenge, _ := beginPasskeyLogin(t, f)
		body := authenticator.assert(t, challenge, authenticator.handle)
		w := f.postCeremony(t, "/api/v1/auth/webauthn/login/finish", body, "", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})

	t.Run("user handle of another account", func(t *testing.T) {
		challenge, cookie := beginPasskeyLogin(t, f)
		body := authenticator.assert(t, challenge, []byte{0, 0, 0, 0, 0, 0, 0, 99})
		w := f.postCeremony(t, "/api/v1/auth/webauthn/login/finish", body, "", cookie)
		assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	})

	t.Run("registration ceremony cannot finish a sign-in", func(t *testing.T) {
		w := f.post(t, "/api/v1/auth/webauthn/register/begin", struct{}{}, f.token)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		cookie := ceremonyCookie(t, w, "seed_passkey_register", "/api/v1/auth/webauthn/register/finish")
		var creation api.WebAuthnCreationOptionsResponse
		decodeStrict(t, w, &creation)
		login := *cookie
		login.Name = "seed_passkey_login"
		body := authenticator.assert(t, creation.PublicKey.Challenge, authenticator.handle)
		w = f.postCeremony(t, "/api/v1/auth/webauthn/login/finish", body, "", &login)
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})
}

func TestPasskeyRoutesDisabledWithoutRelyingParty(t *testing.T) {
	f := newMFAFixture(t)
	w := f.post(t, "/api/v1/auth/webauthn/login/begin", nil, "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	w = f.post(t, "/api/v1/auth/webauthn/register/begin", struct{}{}, f.token)
	assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}
