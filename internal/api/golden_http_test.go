package api_test

// golden_http_test.go is the characterization ("golden") harness for the HTTP
// surface. It boots the real server (in-memory-ish: temp config, nil DB) and
// snapshots (status, normalized JSON body) for a curated set of routes into
// testdata/golden/*.txt. Subsequent runs replay and diff against the snapshot.
//
// Purpose: a refactor-safety net for the re-architecture (capability registry,
// hexagon extraction, package rehome). A pure structural refactor must keep
// every golden byte-identical; a deliberate behavior change updates the golden
// with UPDATE_GOLDEN=1 and is reviewed as a golden diff.
//
// This first increment covers PUBLIC routes reachable on the bare mux
// (server.Mux(), no global auth middleware). Authenticated routes + the full
// middleware chain + the per-route gating matrix are the next increment and
// need a full-handler accessor (tracked in the blueprint Phase 0).
//
// Regenerate:  UPDATE_GOLDEN=1 go test ./internal/api/ -run TestGoldenHTTP
// Verify:      go test ./internal/api/ -run TestGoldenHTTP

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// bodyMode controls how a route's response body participates in the snapshot.
type bodyMode int

const (
	bodyGolden     bodyMode = iota // snapshot the normalized JSON body
	bodyStatusOnly                 // env-dependent body (e.g. host NICs): snapshot status only
)

// goldenRoute is one row of the characterization matrix.
type goldenRoute struct {
	name   string // golden file stem (stable, kebab-ish)
	method string // HTTP method
	path   string // request path
	body   bodyMode
}

// goldenRoutes are routes that return a stable, deterministic response on the
// bare mux without authentication. Host-dependent bodies use bodyStatusOnly.
func goldenRoutes() []goldenRoute {
	return []goldenRoute{
		{name: "version", method: http.MethodGet, path: "/__version", body: bodyGolden},
		// The capability manifest pins every route's policy (role/feature/rate) in
		// one snapshot — the strongest regression detector for the registry.
		{name: "capabilities", method: http.MethodGet, path: "/__capabilities", body: bodyGolden},
		{name: "status", method: http.MethodGet, path: "/api/v1/status", body: bodyGolden},
		{name: "settings", method: http.MethodGet, path: "/api/v1/settings", body: bodyGolden},
		{name: "interfaces", method: http.MethodGet, path: "/api/v1/interfaces", body: bodyStatusOnly},
	}
}

// isVolatileKey reports whether a JSON object key holds a value that varies
// run-to-run (build stamps, clocks, durations, tokens). Such values are replaced
// with a sentinel so the snapshot stays deterministic while still proving shape.
func isVolatileKey(k string) bool {
	switch strings.ToLower(k) {
	case "version", "commit", "buildtime", "build_time",
		"uibuildhash", "uptime", "uptimeseconds", "uptime_seconds",
		"timestamp", "time", "startedat", "started_at", "started",
		"requestid", "request_id", "csrftoken", "csrf_token", "token",
		"duration", "durationms", "latency", "latencyms",
		"now", "date", "generatedat", "generated_at":
		return true
	default:
		return false
	}
}

const volatileSentinel = "<volatile>"

func TestGoldenHTTP(t *testing.T) {
	srv := newTestEndpointServer(t)
	defer srv.close()

	for _, rt := range goldenRoutes() {
		t.Run(rt.name, func(t *testing.T) {
			status, body := doGoldenRequest(t, srv.ts.URL+rt.path, rt.method)

			var snapshot string
			if rt.body == bodyStatusOnly {
				snapshot = formatSnapshot(status, "")
			} else {
				snapshot = formatSnapshot(status, normalizeJSON(t, body))
			}

			goldenPath := filepath.Join("testdata", "golden", rt.name+".txt")
			compareGolden(t, goldenPath, snapshot)
		})
	}
}

// doGoldenRequest issues the request and returns status + raw body.
func doGoldenRequest(t *testing.T, url, method string) (int, []byte) {
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
	return resp.StatusCode, body
}

// formatSnapshot renders a stable, human-readable snapshot.
func formatSnapshot(status int, normalizedBody string) string {
	var b strings.Builder
	b.WriteString("status: ")
	b.WriteString(strconv.Itoa(status))
	b.WriteString("\n")
	if normalizedBody != "" {
		b.WriteString("body:\n")
		b.WriteString(normalizedBody)
		if !strings.HasSuffix(normalizedBody, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// normalizeJSON parses the body, scrubs volatile values, and re-marshals with
// deterministic key ordering (Go sorts map[string]any keys). Non-JSON bodies
// are returned trimmed and verbatim.
func normalizeJSON(t *testing.T, raw []byte) string {
	t.Helper()
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		// Not JSON — snapshot verbatim (still deterministic for static bodies).
		return trimmed
	}
	scrubbed := scrubVolatile(v)
	out, err := json.MarshalIndent(scrubbed, "", "  ")
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	return string(out)
}

// scrubVolatile walks the decoded JSON and replaces volatile-key values with a
// sentinel, recursively.
func scrubVolatile(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, child := range val {
			if isVolatileKey(k) {
				out[k] = volatileSentinel
				continue
			}
			out[k] = scrubVolatile(child)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, child := range val {
			out[i] = scrubVolatile(child)
		}
		return out
	default:
		return val
	}
}

// compareGolden writes the golden when UPDATE_GOLDEN is set, otherwise diffs.
func compareGolden(t *testing.T, path, got string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated golden: %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with UPDATE_GOLDEN=1 to create): %v", path, err)
	}
	if got != string(want) {
		t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", path, want, got)
	}
}
