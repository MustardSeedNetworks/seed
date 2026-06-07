// handlers_mfa_test.go — integration tests for the Wave 3 (#85) MFA
// endpoints: TOTP setup/verify/disable, MFA-gated login, and the
// per-account TOTP rate limiter.
//
// WebAuthn is exercised only at the unit level — full ceremonies need
// a real browser, so the API tests stop at "begin returns options"
// plus the in-memory session store invariants.

package api_test

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// testUserPassword is the bcrypt/argon2 input used to seed the test
// user. It satisfies the Wave 2 password policy (>12 chars, mixed
// classes) so EnforcePasswordPolicy doesn't fire on disable tests.
const testUserPassword = "MFA-Test-Pass-1234!"

// mfaTestFixture wires up a test server with a temp SQLite database
// and a single "admin" user we can drive MFA flows through.
type mfaTestFixture struct {
	server  *api.Server
	handler http.Handler
	db      *database.DB
	token   string // access token for the admin user
}

func newMFAFixture(t *testing.T) *mfaTestFixture {
	t.Helper()

	// MFA rate-limit store is package-global; reset between tests so a
	// previous test's failed verifications don't leak in.
	api.ResetMFAAttempts()
	t.Cleanup(api.ResetMFAAttempts)

	dbPath := filepath.Join(t.TempDir(), "mfa-test.db")
	db, err := database.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	hash, err := auth.HashPassword(testUserPassword)
	require.NoError(t, err)
	_, err = db.CreateUser(context.Background(), "admin", hash, "admin")
	require.NoError(t, err)

	server := api.NewTestServer()
	t.Cleanup(server.Close)

	// Wire the DB and a UserStore so VerifyPasswordOnly and the MFA
	// helpers find the user we just created.
	api.SetTestDB(server, db)
	server.AuthManager().SetUserStore(database.NewUserStoreAdapter(db))

	// Pre-issue an access token so the authenticated MFA endpoints
	// accept the request. The auth middleware reads the token from a
	// cookie or Authorization header.
	token, err := server.AuthManager().GenerateAccessToken(context.Background(), "admin")
	require.NoError(t, err)

	return &mfaTestFixture{
		server:  server,
		handler: server.GetAuthenticatedHandler(),
		db:      db,
		token:   token,
	}
}

// post executes an authenticated POST against the test handler and
// returns the response. Pass an empty token to drive unauthenticated
// requests (login endpoints).
func (f *mfaTestFixture) post(
	t *testing.T, path string, body any, token string,
) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(http.MethodPost, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	return w
}

// codeNow computes the current TOTP code for the given base32 secret
// using the same parameters as auth.VerifyTOTP.
func codeNow(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCodeCustom(secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	require.NoError(t, err)
	return code
}

// TestTOTPEnrollmentFlow covers setup → verify → login-with-totp.
func TestTOTPEnrollmentFlow(t *testing.T) {
	f := newMFAFixture(t)

	// 1. setup
	w := f.post(t, "/api/v1/auth/totp/setup", nil, f.token)
	require.Equal(t, http.StatusOK, w.Code, "setup body: %s", w.Body.String())
	var setupResp struct {
		Secret          string `json:"secret"`
		ProvisioningURI string `json:"provisioning_uri"`
		QRCodePNGBase64 string `json:"qr_code_png_base64"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&setupResp))
	assert.NotEmpty(t, setupResp.Secret)
	assert.NotEmpty(t, setupResp.ProvisioningURI)
	assert.NotEmpty(t, setupResp.QRCodePNGBase64)

	// 2. verify with a correct code
	code := codeNow(t, setupResp.Secret)
	w = f.post(t, "/api/v1/auth/totp/verify",
		map[string]string{"code": code}, f.token)
	require.Equal(t, http.StatusOK, w.Code, "verify body: %s", w.Body.String())

	// 3. confirm totp_enabled is set in the database
	secret, enabled, err := f.db.GetTOTP(context.Background(), "admin")
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, setupResp.Secret, secret)

	// 4. password login now returns mfa_required = true
	w = f.post(t, "/api/v1/auth/login",
		map[string]string{"username": "admin", "password": testUserPassword}, "")
	require.Equal(t, http.StatusOK, w.Code)
	var loginResp api.LoginResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&loginResp))
	assert.True(t, loginResp.MFARequired)
	assert.NotEmpty(t, loginResp.MFAToken)
	assert.Empty(t, loginResp.Token, "no access token until second factor")

	// 5. POST the code to /login/totp and expect a real token
	w = f.post(t, "/api/v1/auth/login/totp", map[string]string{
		"mfa_token": loginResp.MFAToken,
		"code":      codeNow(t, setupResp.Secret),
	}, "")
	require.Equal(t, http.StatusOK, w.Code, "login/totp body: %s", w.Body.String())
	var final api.LoginResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&final))
	assert.NotEmpty(t, final.Token)
	assert.False(t, final.MFARequired)
}

// TestTOTPVerify_WrongCode returns 401 and does NOT flip totp_enabled.
func TestTOTPVerify_WrongCode(t *testing.T) {
	f := newMFAFixture(t)

	w := f.post(t, "/api/v1/auth/totp/setup", nil, f.token)
	require.Equal(t, http.StatusOK, w.Code)

	w = f.post(t, "/api/v1/auth/totp/verify",
		map[string]string{"code": "000000"}, f.token)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	_, enabled, err := f.db.GetTOTP(context.Background(), "admin")
	require.NoError(t, err)
	assert.False(t, enabled, "wrong code must not enable TOTP")
}

// TestLoginTOTP_BadMFAToken rejects an unparseable MFA token without
// touching the user record.
func TestLoginTOTP_BadMFAToken(t *testing.T) {
	f := newMFAFixture(t)

	w := f.post(t, "/api/v1/auth/login/totp", map[string]string{
		"mfa_token": "not-a-real-jwt",
		"code":      "123456",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestTOTPRateLimit is a smoke check that the per-account limiter
// trips after five rapid failures within the 60s window.
func TestTOTPRateLimit(t *testing.T) {
	f := newMFAFixture(t)

	w := f.post(t, "/api/v1/auth/totp/setup", nil, f.token)
	require.Equal(t, http.StatusOK, w.Code)

	// Five failed verifications use up the budget…
	for range 5 {
		_ = f.post(t, "/api/v1/auth/totp/verify",
			map[string]string{"code": "000000"}, f.token)
	}
	// …the sixth must be rate-limited (HTTP 429).
	w = f.post(t, "/api/v1/auth/totp/verify",
		map[string]string{"code": "000000"}, f.token)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

// TestTOTPDisableRequiresBothFactors ensures a stale session can't
// silently remove the MFA enrolment with a password alone.
func TestTOTPDisableRequiresBothFactors(t *testing.T) {
	f := newMFAFixture(t)

	// enrol first
	w := f.post(t, "/api/v1/auth/totp/setup", nil, f.token)
	require.Equal(t, http.StatusOK, w.Code)
	var setup struct {
		Secret string `json:"secret"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&setup))

	w = f.post(t, "/api/v1/auth/totp/verify",
		map[string]string{"code": codeNow(t, setup.Secret)}, f.token)
	require.Equal(t, http.StatusOK, w.Code)

	// Wrong password → 401, still enabled
	w = f.post(t, "/api/v1/auth/totp/disable", map[string]string{
		"password": "WRONG",
		"code":     codeNow(t, setup.Secret),
	}, f.token)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	_, enabled, _ := f.db.GetTOTP(context.Background(), "admin")
	assert.True(t, enabled)

	// Correct password + correct code → disabled
	w = f.post(t, "/api/v1/auth/totp/disable", map[string]string{
		"password": testUserPassword,
		"code":     codeNow(t, setup.Secret),
	}, f.token)
	assert.Equal(t, http.StatusOK, w.Code)
	_, enabled, _ = f.db.GetTOTP(context.Background(), "admin")
	assert.False(t, enabled)
}

// TestMFAStatus reflects current enrolment.
func TestMFAStatus(t *testing.T) {
	f := newMFAFixture(t)

	// no enrolment yet
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/mfa/status", nil)
	req.Header.Set("Authorization", "Bearer "+f.token)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var status struct {
		TOTPEnabled       bool `json:"totp_enabled"`
		WebAuthnEnabled   bool `json:"webauthn_enabled"`
		WebAuthnCredCount int  `json:"webauthn_credential_count"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&status))
	assert.False(t, status.TOTPEnabled)
	assert.False(t, status.WebAuthnEnabled)
	assert.Equal(t, 0, status.WebAuthnCredCount)

	// enroll TOTP
	w = f.post(t, "/api/v1/auth/totp/setup", nil, f.token)
	require.Equal(t, http.StatusOK, w.Code)
	var setup struct {
		Secret string `json:"secret"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&setup))
	w = f.post(t, "/api/v1/auth/totp/verify",
		map[string]string{"code": codeNow(t, setup.Secret)}, f.token)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/mfa/status", nil)
	req.Header.Set("Authorization", "Bearer "+f.token)
	w = httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	require.NoError(t, json.NewDecoder(w.Body).Decode(&status))
	assert.True(t, status.TOTPEnabled)
}

// TestSecretIsBase32 confirms the secret pquerna returns decodes
// cleanly under RFC 4648 base32 (no-padding) — i.e. it's what
// authenticator apps expect.
func TestSecretIsBase32(t *testing.T) {
	f := newMFAFixture(t)
	w := f.post(t, "/api/v1/auth/totp/setup", nil, f.token)
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Secret string `json:"secret"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	_, err := base32.StdEncoding.WithPadding(base32.NoPadding).
		DecodeString(resp.Secret)
	assert.NoError(t, err)
}
