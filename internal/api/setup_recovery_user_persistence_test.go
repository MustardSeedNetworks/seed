package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

const persistedPassword = "MySecure123!Pass"

// TestSetupAndRecoveryPersistTheAdminUser pins the user-store seam of the
// first-run setup and password-recovery handlers: setup creates the admin row
// or updates an existing one, recovery rewrites its hash, and both still
// succeed with no database. The auth manager carries no UserStore here, so any
// row written comes from the handler itself.
func TestSetupAndRecoveryPersistTheAdminUser(t *testing.T) {
	for _, state := range []string{"no database", "no user row", "existing user row"} {
		t.Run("setup/"+state, func(t *testing.T) { checkSetupPersistence(t, state) })
		t.Run("recovery/"+state, func(t *testing.T) { checkRecoveryPersistence(t, state) })
	}
}

func newPersistenceServer(t *testing.T, state string) (*api.Server, *database.DB, string) {
	t.Helper()
	username := testutil.GetTestDefaults().Auth.Username
	cfg := testutil.NewConfigBuilder().WithAuth(username, auth.SetupModePlaceholder).Build()
	server := api.NewTestServerWithConfig(cfg)
	t.Cleanup(server.Close)
	if state == "no database" {
		return server, nil, username
	}
	db, err := database.Open(dbtest.Path(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	if state == "existing user row" {
		_, err = db.CreateUser(t.Context(), username, "$2a$10$stale", "operator")
		require.NoError(t, err)
	}
	api.SetTestDB(server, db)
	return server, db, username
}

func checkSetupPersistence(t *testing.T, state string) {
	t.Helper()
	server, db, username := newPersistenceServer(t, state)
	setupToken, err := server.SetupTokenManager().GenerateToken()
	require.NoError(t, err)
	body, err := json.Marshal(api.SetupCompleteRequest{Password: persistedPassword, SetupToken: setupToken})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/setup/complete", bytes.NewReader(body))
	server.HandleSetupComplete(w, request)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	if db == nil {
		return
	}
	wantRole := "admin"
	if state == "existing user row" {
		wantRole = "operator"
	}
	assertPersistedPassword(t, db, username, wantRole)
}

func checkRecoveryPersistence(t *testing.T, state string) {
	t.Helper()
	server, db, username := newPersistenceServer(t, state)
	dir := t.TempDir()
	recovery := auth.NewRecoveryTokenManager(dir)
	server.SetRecoveryManager(recovery)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".recovery"), nil, 0o600))
	require.True(t, recovery.CheckRecoveryMode())
	token, err := os.ReadFile(recovery.TokenFilePath())
	require.NoError(t, err)
	body, err := json.Marshal(map[string]string{
		"token": strings.TrimSpace(string(token)), "password": persistedPassword,
	})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/recovery/complete", bytes.NewReader(body))
	server.HandleRecoveryComplete(w, request)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	if db == nil {
		return
	}
	if state == "no user row" {
		_, err = db.GetUser(t.Context(), username)
		require.ErrorIs(t, err, users.ErrUserNotFound)
		return
	}
	assertPersistedPassword(t, db, username, "operator")
}

func assertPersistedPassword(t *testing.T, db *database.DB, username, wantRole string) {
	t.Helper()
	user, err := db.GetUser(t.Context(), username)
	require.NoError(t, err)
	require.Equal(t, wantRole, user.Role)
	ok, _, err := auth.VerifyPassword(user.PasswordHash, persistedPassword)
	require.NoError(t, err)
	require.True(t, ok)
}
