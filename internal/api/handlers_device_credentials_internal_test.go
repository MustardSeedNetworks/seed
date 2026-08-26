package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
)

// testEncrypter stands in for the keyring. Deliberately not a no-op: a no-op
// would let the "never returns a secret" tests pass while plaintext round-
// tripped through storage.
type testEncrypter struct{}

func (testEncrypter) EncryptValue(plaintext string) (string, error) {
	return "enc:v1:" + plaintext, nil
}

func newDeviceCredentialsTestServer(t *testing.T) *Server {
	t.Helper()
	db := newTestDB(t)
	s := &Server{}
	s.dbConn = db
	svc, err := app.NewDeviceCredentials(s.db, testEncrypter{})
	if err != nil {
		t.Fatalf("build credentials use-case: %v", err)
	}
	s.deviceCredentials = svc
	return s
}

func postCredential(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(httptest.NewRequest(http.MethodPost,
		deviceCredentialsPath, bytes.NewBufferString(body)))
	rec := httptest.NewRecorder()
	s.handleDeviceCredentials(rec, req)
	return rec
}

// TestCredentialResponsesNeverCarrySecrets is the property #1799 exists for.
// Settings used to store v2c communities in plaintext config and return them
// to clients; no response on this surface may carry a secret, in plaintext or
// ciphertext.
func TestCredentialResponsesNeverCarrySecrets(t *testing.T) {
	s := newDeviceCredentialsTestServer(t)

	created := postCredential(t, s, `{"name":"core","community":"s3cret-community"}`)
	if created.Code != http.StatusOK {
		t.Fatalf("create: status %d, body %s", created.Code, created.Body.String())
	}

	listReq := withClaim(httptest.NewRequest(http.MethodGet, deviceCredentialsPath, nil))
	listRec := httptest.NewRecorder()
	s.handleDeviceCredentials(listRec, listReq)

	for _, body := range []string{created.Body.String(), listRec.Body.String()} {
		for _, secret := range []string{"s3cret-community", "enc:v1:"} {
			if strings.Contains(body, secret) {
				t.Errorf("response leaked %q: %s", secret, body)
			}
		}
		// The non-secret identity must still be present, or the UI cannot
		// render a list of credentials to choose between.
		if !strings.Contains(body, "core") {
			t.Errorf("response dropped the credential name: %s", body)
		}
	}
}

// TestCredentialIsScopedToTheCallersClient — a credential owned by another
// tenant must be indistinguishable from one that does not exist.
func TestCredentialIsScopedToTheCallersClient(t *testing.T) {
	s := newDeviceCredentialsTestServer(t)

	created := postCredential(t, s, `{"name":"mine","community":"c"}`)
	var body map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatal("created credential has no id")
	}

	// Same id, no client claim: unauthenticated, not another tenant's read.
	req := httptest.NewRequest(http.MethodGet, deviceCredentialsPathPrefix+id, nil)
	rec := httptest.NewRecorder()
	s.handleDeviceCredentialByID(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no claim: status %d, want 401", rec.Code)
	}
}

func TestCredentialValidationIsA400WithAReason(t *testing.T) {
	s := newDeviceCredentialsTestServer(t)

	for _, tc := range []struct{ name, body, want string }{
		{"both kinds", `{"name":"x","community":"c","snmpV3User":"u"}`, "not both"},
		{"neither", `{"name":"x"}`, "either a community"},
		{"no name", `{"community":"c"}`, "name is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := postCredential(t, s, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d, want 400: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(strings.ToLower(rec.Body.String()), tc.want) {
				t.Errorf("body %q does not explain the problem (want %q)",
					rec.Body.String(), tc.want)
			}
		})
	}
}

// TestCredentialStoreUnavailableIs503 — a server with no keyring has no
// credential use-case. Reporting 503 says "not configured"; a 500 would say
// "broken", and returning 200 with an empty list would say "you have none".
func TestCredentialStoreUnavailableIs503(t *testing.T) {
	s := &Server{}
	req := withClaim(httptest.NewRequest(http.MethodGet, deviceCredentialsPath, nil))
	rec := httptest.NewRecorder()
	s.handleDeviceCredentials(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status %d, want 503", rec.Code)
	}
}

// TestCredentialIDMustBeServerGenerated — the id arrives from the URL path, so
// it is caller-controlled. Validating its shape means a lookup cannot be
// steered, and it is what makes the id safe to log: an unvalidated value can
// carry newlines and forge log entries, which is what CodeQL flagged on the
// first revision of this handler.
func TestCredentialIDMustBeServerGenerated(t *testing.T) {
	s := newDeviceCredentialsTestServer(t)

	for _, id := range []string{
		"",
		"not-a-credential",
		"cred-XYZ",
		"cred-0123456789ab\ninjected log line",
		"cred-0123456789abcdef",
		"../../etc/passwd",
	} {
		// The path is set after construction: httptest.NewRequest refuses to
		// build a target containing a newline, but a raw client can send one,
		// and by the time the mux dispatches, it is just a string field.
		req := withClaim(httptest.NewRequest(http.MethodGet, deviceCredentialsPath, nil))
		req.URL.Path = deviceCredentialsPathPrefix + id
		rec := httptest.NewRecorder()
		s.handleDeviceCredentialByID(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("id %q: status %d, want 400", id, rec.Code)
		}
	}

	// The generated shape is accepted — 404 because it does not exist, which
	// is the point: it got past validation and was actually looked up.
	req := withClaim(httptest.NewRequest(http.MethodGet,
		deviceCredentialsPathPrefix+"cred-0123456789ab", nil))
	rec := httptest.NewRecorder()
	s.handleDeviceCredentialByID(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("well-formed id: status %d, want 404", rec.Code)
	}
}

func TestCredentialDeleteReturns204(t *testing.T) {
	s := newDeviceCredentialsTestServer(t)
	created := postCredential(t, s, `{"name":"gone","community":"c"}`)
	var body map[string]any
	_ = json.Unmarshal(created.Body.Bytes(), &body)
	id, _ := body["id"].(string)

	req := withClaim(httptest.NewRequest(http.MethodDelete, deviceCredentialsPathPrefix+id, nil))
	rec := httptest.NewRecorder()
	s.handleDeviceCredentialByID(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status %d, want 204: %s", rec.Code, rec.Body.String())
	}
}
