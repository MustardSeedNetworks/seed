package api_test

// golden_http_authchain_test.go extends the characterization harness to the
// FULL middleware chain (server.Handler()), not just the bare mux. It boots a
// DB-backed server and snapshots (status, stable security headers, normalized
// body) for a representative slice of routes: public pass-through, protected
// GET/POST (unauthenticated → the auth boundary), and a feature-gated route.
//
// These goldens pin the behavior the Phase-1 capability registry must preserve:
// the auth/CSRF/security-header chain and the per-route gating outcomes.
//
// Regenerate:  UPDATE_GOLDEN=1 go test ./internal/api/ -run TestGoldenHTTPAuthChain
// Verify:      go test ./internal/api/ -run TestGoldenHTTPAuthChain

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// stableSecurityHeaderNames is the deterministic subset of response headers worth
// snapshotting: set globally by securityHeadersMiddleware on every response.
// Volatile headers (Date, Content-Length, etc.) are intentionally excluded.
func stableSecurityHeaderNames() []string {
	return []string{
		"Content-Security-Policy",
		"Referrer-Policy",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-Xss-Protection",
	}
}

// authChainRoutes is the representative slice through the full chain: public
// pass-through (version, status), protected GET (settings, profiles), writeGated
// mutating POST (settings/link, wifi/survey/create), and a feature-gated route
// (path/path). The harness records whatever the current behavior is
// (characterization); it asserts nothing — a refactor that changes any outcome
// produces a reviewable golden diff.
func authChainRoutes() []goldenRoute {
	return []goldenRoute{
		{name: "version", method: http.MethodGet, path: "/__version"},
		{name: "status", method: http.MethodGet, path: "/api/v1/status"},
		{name: "settings-get", method: http.MethodGet, path: "/api/v1/settings"},
		{name: "profiles-get", method: http.MethodGet, path: "/api/v1/profiles"},
		{name: "settings-link-post", method: http.MethodPost, path: "/api/v1/settings/link"},
		{name: "survey-create-post", method: http.MethodPost, path: "/api/v1/wifi/survey/create"},
		{name: "path-get", method: http.MethodGet, path: "/api/v1/path/path"},
		// Unified job runner (ADR-0005): create is operator-gated/mutating,
		// inspect is a safe read — both behind the global auth chain.
		{name: "jobs-create-post", method: http.MethodPost, path: "/api/v1/jobs"},
		{name: "jobs-get", method: http.MethodGet, path: "/api/v1/jobs/some-id"},
	}
}

func TestGoldenHTTPAuthChain(t *testing.T) {
	srv := newFullChainServer(t)
	defer srv.close()

	for _, rt := range authChainRoutes() {
		t.Run(rt.name, func(t *testing.T) {
			status, headers, body := doFullChainRequest(t, srv.ts.URL+rt.path, rt.method)
			snapshot := formatChainSnapshot(status, headers, normalizeJSON(t, body))
			goldenPath := filepath.Join("testdata", "golden", "authchain", rt.name+".txt")
			compareGolden(t, goldenPath, snapshot)
		})
	}
}

// newFullChainServer boots a DB-backed server and serves the full Handler()
// chain (recover → … → auth → CSRF → mux), unlike newTestEndpointServer which
// serves the bare mux.
func newFullChainServer(t *testing.T) *testEndpointServer {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save test config: %v", err)
	}

	db, err := database.Open(filepath.Join(tmpDir, "seed.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	netMgr, err := netif.NewManager("")
	if err != nil {
		t.Logf("warning: network manager: %v", err)
	}

	server := api.NewServer(cfg, configPath, "", netMgr, false, nil, db, nil)

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ts := httptest.NewUnstartedServer(server.Handler())
	ts.Listener = ln
	ts.Start()
	return &testEndpointServer{ts: ts, server: server}
}

// doFullChainRequest issues the request and returns status, headers, and body.
func doFullChainRequest(t *testing.T, url, method string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, resp.Header, body
}

// formatChainSnapshot renders status + the stable security headers + body.
func formatChainSnapshot(status int, headers http.Header, normalizedBody string) string {
	var b strings.Builder
	b.WriteString("status: ")
	b.WriteString(strconv.Itoa(status))
	b.WriteString("\nheaders:\n")
	for _, name := range stableSecurityHeaderNames() {
		val := headers.Get(name)
		if val == "" {
			val = "(absent)"
		}
		b.WriteString("  ")
		b.WriteString(name)
		b.WriteString(": ")
		b.WriteString(val)
		b.WriteString("\n")
	}
	if normalizedBody != "" {
		b.WriteString("body:\n")
		b.WriteString(normalizedBody)
		if !strings.HasSuffix(normalizedBody, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}
