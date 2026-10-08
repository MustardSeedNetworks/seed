package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/identity/mfa"
)

// mfaStatusOf reads /api/v1/auth/mfa/status for the fixture's session.
func mfaStatusOf(t *testing.T, f *mfaTestFixture) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/mfa/status", nil)
	req.Header.Set("Authorization", "Bearer "+f.token)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	return w.Code, body
}

// TestMFAStatusCountsPasskeys pins that the status summary counts the
// user's registered WebAuthn credentials (seed#2750 MFA port).
func TestMFAStatusCountsPasskeys(t *testing.T) {
	f := newMFAFixture(t)
	user, err := f.db.GetUser(t.Context(), "admin")
	require.NoError(t, err)
	for _, id := range []string{"key-a", "key-b"} {
		_, err = f.db.AddWebAuthnCredential(t.Context(), user.ID, mfa.WebAuthnCredential{
			CredentialID: []byte(id), PublicKey: []byte("public-key"),
		})
		require.NoError(t, err)
	}

	code, body := mfaStatusOf(t, f)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, true, body["webauthnEnabled"])
	assert.InDelta(t, 2, body["webauthnCredentialCount"], 0)
	assert.Equal(t, false, body["totpEnabled"])
}

// TestMFAWithoutDatabase pins the no-database behavior: enrolment and status
// answer 401 rather than failing or panicking (seed#2750 MFA port).
func TestMFAWithoutDatabase(t *testing.T) {
	api.ResetMFAAttempts()
	t.Cleanup(api.ResetMFAAttempts)
	server := api.NewTestServer()
	t.Cleanup(server.Close)
	token, err := server.AuthManager().GenerateAccessToken(context.Background(), "admin")
	require.NoError(t, err)
	f := &mfaTestFixture{server: server, handler: server.Handler(), token: token}

	code, _ := mfaStatusOf(t, f)
	assert.Equal(t, http.StatusUnauthorized, code)
	for _, path := range []string{"/api/v1/auth/totp/setup", "/api/v1/auth/totp/verify", "/api/v1/auth/totp/disable"} {
		w := f.post(t, path, map[string]string{"code": "000000"}, f.token)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "%s: %s", path, w.Body.String())
	}
}
