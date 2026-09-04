package api_test

// auth_lifecycle_tls_test.go drives one session end to end against the
// production TLS listener and a file-backed SQLite database, rather than a
// recorder and an in-memory store.
//
// Everything else in this package exercises the chain through
// httptest.ResponseRecorder or a plain-HTTP httptest server, which cannot see a
// defect that lives in the listener, the cookie attributes, or the way a real
// client stores and replays them. Secure cookies are the sharpest example: a
// cookie marked Secure is silently dropped by net/http's own jar over plain
// HTTP, so a plain-HTTP harness proves nothing about whether the session
// survives the round trip.
//
// The sequence is the one an operator performs: reject anonymous, log in, read,
// be refused a write without a CSRF token, fetch the token, write, log out, and
// have the captured session cookie refused on replay.

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

const (
	lifecycleUser     = "lifecycle-admin"
	lifecyclePassword = "Str0ng-Lifecycle-Passphrase!"
)

// tlsSession is a live server plus a client that keeps cookies the way a
// browser does.
type tlsSession struct {
	baseURL    string
	client     *http.Client
	configPath string
}

func TestAuthLifecycleOverProductionTLSListener(t *testing.T) {
	s := newTLSSession(t)

	t.Run("anonymous read is refused", func(t *testing.T) { s.stepAnonymousRead(t) })
	t.Run("login sets a session the client can carry", func(t *testing.T) { s.stepLogin(t) })
	t.Run("authenticated read succeeds", func(t *testing.T) { s.stepAuthenticatedRead(t) })
	t.Run("a write without a CSRF token is refused", func(t *testing.T) { s.stepWriteWithoutCSRF(t) })

	var csrfToken string
	t.Run("the session can fetch a CSRF token", func(t *testing.T) { csrfToken = s.stepFetchCSRF(t) })
	t.Run("the same write succeeds with the token and reaches disk", func(t *testing.T) {
		s.stepWriteWithCSRF(t, csrfToken)
	})

	// Captured before logout: this is what an attacker who lifted the cookie
	// would replay afterwards.
	stolen := s.cookies(t)

	t.Run("logout ends the session", func(t *testing.T) { s.stepLogout(t, csrfToken) })
	t.Run("the pre-logout cookie is refused on replay", func(t *testing.T) { s.stepReplay(t, stolen) })
}

func (s *tlsSession) stepAnonymousRead(t *testing.T) {
	t.Helper()
	status, _ := s.do(t, http.MethodGet, "/api/v1/settings", nil, "")
	if status != http.StatusUnauthorized {
		t.Errorf("GET /settings while anonymous = %d, want %d", status, http.StatusUnauthorized)
	}
}

func (s *tlsSession) stepLogin(t *testing.T) {
	t.Helper()
	status, body := s.do(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": lifecycleUser,
		"password": lifecyclePassword,
	}, "")
	if status != http.StatusOK {
		t.Fatalf("POST /auth/login = %d, want 200: %s", status, body)
	}

	var resp api.LoginResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode login response: %v (%s)", err, body)
	}
	if resp.MFARequired {
		t.Fatal("login reported mfaRequired for an account with no second factor")
	}
	if resp.Token == "" {
		t.Fatal("login returned no access token")
	}
	if len(s.cookies(t)) == 0 {
		t.Fatal("login set no cookie the client kept — a Secure cookie is dropped over plain HTTP")
	}
}

func (s *tlsSession) stepAuthenticatedRead(t *testing.T) {
	t.Helper()
	status, body := s.do(t, http.MethodGet, "/api/v1/settings", nil, "")
	if status != http.StatusOK {
		t.Fatalf("GET /settings while logged in = %d, want 200: %s", status, body)
	}
}

func (s *tlsSession) stepWriteWithoutCSRF(t *testing.T) {
	t.Helper()
	status, body := s.do(t, http.MethodPut, "/api/v1/settings", settingsUpdate("seed-lifecycle-a"), "")
	if status != http.StatusForbidden {
		t.Fatalf("PUT /settings with no CSRF token = %d, want %d: %s",
			status, http.StatusForbidden, body)
	}
}

func (s *tlsSession) stepFetchCSRF(t *testing.T) string {
	t.Helper()
	status, body := s.do(t, http.MethodGet, "/api/v1/auth/csrf", nil, "")
	if status != http.StatusOK {
		t.Fatalf("GET /auth/csrf = %d, want 200: %s", status, body)
	}
	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode CSRF response: %v (%s)", err, body)
	}
	token := resp["csrfToken"]
	if token == "" {
		token = resp["token"]
	}
	if token == "" {
		t.Fatalf("CSRF response carried no token: %s", body)
	}
	return token
}

func (s *tlsSession) stepWriteWithCSRF(t *testing.T, csrfToken string) {
	t.Helper()
	if csrfToken == "" {
		t.Skip("no CSRF token to present")
	}
	status, body := s.do(t, http.MethodPut, "/api/v1/settings",
		settingsUpdate("seed-lifecycle-b"), csrfToken)
	if status != http.StatusOK {
		t.Fatalf("PUT /settings with a CSRF token = %d, want 200: %s", status, body)
	}

	// The route is writeGated because it persists. Read the file the server
	// wrote rather than trusting the 200.
	saved, err := config.Load(s.configPath)
	if err != nil {
		t.Fatalf("reload the config the server saved: %v", err)
	}
	if saved.Speedtest.ServerID != "seed-lifecycle-b" {
		t.Errorf("persisted speedtest.serverId = %q, want %q",
			saved.Speedtest.ServerID, "seed-lifecycle-b")
	}
}

func (s *tlsSession) stepLogout(t *testing.T, csrfToken string) {
	t.Helper()
	status, body := s.do(t, http.MethodPost, "/api/v1/auth/logout", nil, csrfToken)
	if status != http.StatusOK {
		t.Fatalf("POST /auth/logout = %d, want 200: %s", status, body)
	}
	status, _ = s.do(t, http.MethodGet, "/api/v1/settings", nil, "")
	if status != http.StatusUnauthorized {
		t.Errorf("GET /settings after logout = %d, want %d", status, http.StatusUnauthorized)
	}
}

func (s *tlsSession) stepReplay(t *testing.T, stolen []*http.Cookie) {
	t.Helper()
	if len(stolen) == 0 {
		t.Fatal("no session cookie was captured before logout")
	}
	s.setCookies(t, stolen)

	status, body := s.do(t, http.MethodGet, "/api/v1/settings", nil, "")
	if status != http.StatusUnauthorized {
		t.Errorf("replaying the pre-logout session cookie = %d, want %d: %s",
			status, http.StatusUnauthorized, body)
	}
}

// settingsUpdate is a small persistent write on one of the six groups the
// settings route actually applies, with no side effect beyond the file.
func settingsUpdate(serverID string) map[string]any {
	return map[string]any{"speedtest": map[string]any{"serverId": serverID}}
}

func newTLSSession(t *testing.T) *tlsSession {
	t.Helper()

	// ensureSelfSignedCert writes certs/ relative to the working directory.
	t.Chdir(t.TempDir())

	held := holdFallbackBasePort(t)
	port := held.Addr().(*net.TCPAddr).Port
	_ = held.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "seed.json")

	cfg := testutil.NewConfigBuilder().WithPort(port).Build()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	db, err := database.Open(filepath.Join(dir, "seed.db"))
	if err != nil {
		t.Fatalf("open the file-backed database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	hash, hashErr := auth.HashPassword(lifecyclePassword)
	if hashErr != nil {
		t.Fatalf("hash the test password: %v", hashErr)
	}
	if _, createErr := db.CreateUser(context.Background(), lifecycleUser, hash, "admin"); createErr != nil {
		t.Fatalf("create the test user: %v", createErr)
	}

	// NewServer, not the test constructor: it is the path that binds configPath
	// into the settings use-case and wires the database-backed user store, both
	// of which this test relies on.
	netMgr := netif.NewMockManager(netif.DefaultMockConfig())
	server := api.NewServer(cfg, configPath, "", netMgr, false, nil, db, nil)
	t.Cleanup(server.Close)

	if _, _, certErr := server.EnsureSelfSignedCert(); certErr != nil {
		t.Fatalf("generate the listener certificate: %v", certErr)
	}

	serveErr := make(chan error, 1)
	initialized := make(chan struct{})
	go func() {
		serveErr <- server.StartHTTPSForTest(server.Handler(), initialized)
	}()
	<-initialized

	var once sync.Once
	t.Cleanup(func() {
		once.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.ShutdownHTTPSForTest(ctx)
			<-serveErr
		})
	})

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	transport := trustedTransport(t)
	t.Cleanup(transport.CloseIdleConnections)

	client := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	s := &tlsSession{client: client, configPath: configPath}
	s.baseURL = waitForLifecycleListener(t, client, port)
	return s
}

// waitForLifecycleListener finds the port the listener actually bound; #69's
// fallback walks +1..+9 when the requested one is taken.
func waitForLifecycleListener(t *testing.T, client *http.Client, requestedPort int) string {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		for candidate := requestedPort; candidate <= requestedPort+9; candidate++ {
			// The generated certificate carries DNSNames localhost and
			// seed.local and no IP SAN, so the IP literal will not verify.
			base := "https://localhost:" + strconv.Itoa(candidate)
			resp, err := client.Get(base + "/__version")
			if err != nil {
				lastErr = err
				continue
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("TLS listener never answered on %d..%d: %v", requestedPort, requestedPort+9, lastErr)
	return ""
}

func (s *tlsSession) do(t *testing.T, method, path string, body any, csrfToken string) (int, []byte) {
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
	if csrfToken != "" {
		req.Header.Set("X-Csrf-Token", csrfToken)
	}

	resp, err := s.client.Do(req)
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

func (s *tlsSession) jarURL(t *testing.T) *url.URL {
	t.Helper()
	u, err := url.Parse(s.baseURL)
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}
	return u
}

func (s *tlsSession) cookies(t *testing.T) []*http.Cookie {
	t.Helper()
	return s.client.Jar.Cookies(s.jarURL(t))
}

func (s *tlsSession) setCookies(t *testing.T, cookies []*http.Cookie) {
	t.Helper()
	s.client.Jar.SetCookies(s.jarURL(t), cookies)
}
