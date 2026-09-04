package api_test

// mfa_login_chain_test.go drives the second factor through the FULL middleware
// chain, which is where it was broken (#2391).
//
// handlers_mfa_test.go covers the same flow through GetAuthenticatedHandler, so
// it passed while the product was unusable: /api/v1/auth/login/totp bypasses the
// JWT middleware but was not on the CSRF exempt list, so the CSRF middleware
// found no session and answered 401. Enrolling TOTP from the shipped Security
// page locked the account permanently, and /api/v1/recovery/complete — the way
// back in — failed identically.
//
// The lesson is the shape of the harness: a test that skips the chain cannot see
// a defect that lives in it.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

const mfaChainPassword = "Str0ng-Mfa-Chain-Passphrase!"

// postThroughChain issues a POST through server.Handler() — the recover →
// headers → auth → CSRF → mux stack a real request crosses.
func postThroughChain(t *testing.T, handler http.Handler, path string, body any) (int, []byte) {
	t.Helper()

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode %s body: %v", path, err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	return w.Code, w.Body.Bytes()
}

func TestSecondFactorLoginSurvivesTheMiddlewareChain(t *testing.T) {
	api.ResetMFAAttempts()
	t.Cleanup(api.ResetMFAAttempts)

	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "mfa-chain.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	hash, hashErr := auth.HashPassword(mfaChainPassword)
	if hashErr != nil {
		t.Fatalf("hash password: %v", hashErr)
	}
	if _, createErr := db.CreateUser(context.Background(), "admin", hash, "admin"); createErr != nil {
		t.Fatalf("create user: %v", createErr)
	}

	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	netMgr := netif.NewMockManager(netif.DefaultMockConfig())
	server := api.NewServer(cfg, filepath.Join(dir, "seed.json"), "", netMgr, false, nil, db, nil)
	t.Cleanup(server.Close)
	handler := server.Handler()

	// Enrol TOTP the way MfaCard does, then log in and complete the second
	// factor — the step that answered 401 for every user who enrolled.
	const totpSecret = "JBSWY3DPEHPK3PXP"
	if secretErr := db.SetTOTPSecret(context.Background(), "admin", totpSecret); secretErr != nil {
		t.Fatalf("store the TOTP secret: %v", secretErr)
	}
	if enableErr := db.EnableTOTP(context.Background(), "admin"); enableErr != nil {
		t.Fatalf("enable TOTP: %v", enableErr)
	}

	status, body := postThroughChain(t, handler, "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": mfaChainPassword,
	})
	if status != http.StatusOK {
		t.Fatalf("POST /auth/login = %d, want 200: %s", status, body)
	}

	var login api.LoginResponse
	if unmarshalErr := json.Unmarshal(body, &login); unmarshalErr != nil {
		t.Fatalf("decode login response: %v (%s)", unmarshalErr, body)
	}
	if !login.MFARequired || login.MFAToken == "" {
		t.Fatalf("login after enrolment did not ask for a second factor: %s", body)
	}

	// The whole point: a correct code must complete the login. Before the fix
	// the CSRF middleware answered 401 here, so this never reached the handler
	// and the account was locked for good.
	status, body = postThroughChain(t, handler, "/api/v1/auth/login/totp", map[string]string{
		"mfaToken": login.MFAToken,
		"code":     codeNow(t, totpSecret),
	})
	if status != http.StatusOK {
		t.Fatalf("POST /auth/login/totp with a correct code = %d, want 200 — "+
			"the second factor cannot be completed and the account is locked out: %s", status, body)
	}

	var final api.LoginResponse
	if unmarshalErr := json.Unmarshal(body, &final); unmarshalErr != nil {
		t.Fatalf("decode second-factor response: %v (%s)", unmarshalErr, body)
	}
	if final.Token == "" {
		t.Errorf("second factor returned no access token: %s", body)
	}
	if final.MFARequired {
		t.Errorf("second factor still reports mfaRequired: %s", body)
	}
}
