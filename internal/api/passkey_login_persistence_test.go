package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

func TestPasskeyLoginRequiresPersistedCounter(t *testing.T) {
	for _, state := range []string{"missing credential", "closed database", "persisted credential"} {
		t.Run(state, func(t *testing.T) { checkPasskeyLoginPersistence(t, state) })
	}
}

func checkPasskeyLoginPersistence(t *testing.T, state string) {
	t.Helper()
	fixture := newMFAFixture(t)
	credential := &webauthn.Credential{ID: []byte("verified-passkey")}
	credential.Authenticator.SignCount = 7
	preparePasskeyPersistence(t, fixture, state, credential.ID)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/webauthn/login/finish", nil)
	response := httptest.NewRecorder()
	api.CompleteWebAuthnLoginForTest(fixture.server, response, request, "admin", credential)
	if state == "persisted credential" {
		assertPersistedPasskeyLogin(t, fixture, response)
	} else {
		assertRejectedPasskeyLogin(t, response)
	}
}

func preparePasskeyPersistence(t *testing.T, fixture *mfaTestFixture, state string, id []byte) {
	t.Helper()
	if state == "missing credential" {
		return
	}
	user, err := fixture.db.GetUser(t.Context(), "admin")
	require.NoError(t, err)
	_, err = fixture.db.AddWebAuthnCredential(t.Context(), user.ID, database.WebAuthnCredential{
		CredentialID: id, PublicKey: []byte("public-key"),
	})
	require.NoError(t, err)
	if state == "closed database" {
		require.NoError(t, fixture.db.Close())
	}
}

func assertRejectedPasskeyLogin(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Empty(t, response.Header().Values("Set-Cookie"))
	var login api.LoginResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &login))
	require.Empty(t, login.Token)
}

func assertPersistedPasskeyLogin(t *testing.T, fixture *mfaTestFixture, response *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusOK, response.Code)
	var login api.LoginResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &login))
	require.NotEmpty(t, login.Token)
	require.NotEmpty(t, response.Header().Values("Set-Cookie"))
	claims, err := fixture.server.AuthManager().ValidateToken(t.Context(), login.Token)
	require.NoError(t, err)
	require.Equal(t, "admin", claims.Username)
	user, err := fixture.db.GetUser(t.Context(), "admin")
	require.NoError(t, err)
	credentials, err := fixture.db.ListWebAuthnCredentials(t.Context(), user.ID)
	require.NoError(t, err)
	require.Len(t, credentials, 1)
	require.Equal(t, uint32(7), credentials[0].SignCount)
	require.NotNil(t, credentials[0].LastUsedAt)
}
