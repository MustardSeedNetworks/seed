package api_test

// pat_lifecycle_tls_test.go drives a personal access token end to end against
// the production TLS listener and a file-backed SQLite database, through the
// same chain Server.Handler builds.
//
// #2450: every PAT request answered 401. The PAT middleware resolved the token
// and forwarded, then the JWT middleware pulled the same bearer out of the
// header and failed ValidateToken; a mutating request hit a second wall in the
// CSRF middleware, which derives its session key from a JWT-shaped bearer and
// so found none. The package's other PAT tests pass a stub `next` and were
// green throughout: they prove apiTokenMiddleware resolves a token, not that a
// request survives the chain.

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/license"
)

const bogusPAT = "sd_pat_00000000000000000000000000000000"

func TestPersonalAccessTokenSurvivesTheMiddlewareChain(t *testing.T) {
	s := newTLSSession(t)
	s.startProTrial(t)
	s.stepLogin(t)
	csrfToken := s.stepFetchCSRF(t)

	pat := s.mintPAT(t, csrfToken)

	t.Run("a read authenticates", func(t *testing.T) {
		status, body := s.doWithBearer(t, http.MethodGet, "/api/v1/settings", nil, pat)
		if status != http.StatusOK {
			t.Fatalf("GET /settings with a PAT = %d, want 200: %s", status, body)
		}
	})

	t.Run("a write needs no CSRF token and reaches disk", func(t *testing.T) {
		status, body := s.doWithBearer(t, http.MethodPut, "/api/v1/settings",
			settingsUpdate("seed-pat-write"), pat)
		if status != http.StatusOK {
			t.Fatalf("PUT /settings with a PAT = %d, want 200: %s", status, body)
		}
		saved, err := config.Load(s.configPath)
		if err != nil {
			t.Fatalf("reload the config the server saved: %v", err)
		}
		if saved.Speedtest.ServerID != "seed-pat-write" {
			t.Errorf("persisted speedtest.serverId = %q, want %q",
				saved.Speedtest.ServerID, "seed-pat-write")
		}
	})

	t.Run("an unknown PAT is refused", func(t *testing.T) {
		status, _ := s.doWithBearer(t, http.MethodGet, "/api/v1/settings", nil, bogusPAT)
		if status != http.StatusUnauthorized {
			t.Errorf("GET /settings with an unminted PAT = %d, want 401", status)
		}
	})

	// The CSRF exemption is keyed on the context marker the PAT middleware
	// sets after the store lookup, not on the shape of the bearer. An unknown
	// PAT never reaches CSRF, so this asserts the other half: a browser
	// session still hits the wall.
	t.Run("a cookie session still needs a CSRF token", func(t *testing.T) {
		status, body := s.do(t, http.MethodPut, "/api/v1/settings",
			settingsUpdate("seed-pat-should-not-persist"), "")
		if status != http.StatusForbidden {
			t.Fatalf("PUT /settings on a cookie session with no CSRF token = %d, want 403: %s",
				status, body)
		}
	})
}

// startProTrial puts the running server on Pro, which PAT minting requires.
// The manager is rooted in a temp directory: NewServer's own manager persists
// activation state in the developer's real config directory.
func (s *tlsSession) startProTrial(t *testing.T) {
	t.Helper()
	mgr, err := license.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("license manager: %v", err)
	}
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}
	s.server.SetLicenseManagerForTest(mgr)
}

// mintPAT mints an operator-scoped token through the API the operator uses.
func (s *tlsSession) mintPAT(t *testing.T, csrfToken string) string {
	t.Helper()
	status, body := s.do(t, http.MethodPost, "/api/v1/tokens",
		map[string]string{"name": "pat-lifecycle", "scope": "operator"}, csrfToken)
	if status != http.StatusCreated {
		t.Fatalf("POST /tokens = %d, want 201: %s", status, body)
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode mint response: %v (%s)", err, body)
	}
	if !strings.HasPrefix(resp.Token, "sd_pat_") {
		t.Fatalf("minted token %q is not a PAT", resp.Token)
	}
	return resp.Token
}

// doWithBearer sends a request the way automation does: an Authorization
// header, no cookie jar, no CSRF token.
func (s *tlsSession) doWithBearer(
	t *testing.T, method, path string, body any, bearer string,
) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		reader = strings.NewReader(string(encoded))
	}

	req, err := http.NewRequest(method, s.baseURL+path, reader)
	if err != nil {
		t.Fatalf("build %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+bearer)

	// A fresh client with no jar: the session cookie must play no part.
	client := &http.Client{Transport: trustedTransport(t), Timeout: s.client.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s %s body: %v", method, path, err)
	}
	return resp.StatusCode, payload
}
