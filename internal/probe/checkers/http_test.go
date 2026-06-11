package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

type fakeHTTPDoer struct {
	resp *http.Response
	err  error
	got  *http.Request
}

func (f *fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	f.got = req
	return f.resp, f.err
}

func mockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestHTTPChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewHTTPChecker().Kind() != "http" {
		t.Errorf("HTTPChecker.Kind != http")
	}
	if checkers.NewHTTPSChecker().Kind() != "https" {
		t.Errorf("HTTPSChecker.Kind != https")
	}
}

func TestHTTPChecker_Run_200_Default(t *testing.T) {
	t.Parallel()
	doer := &fakeHTTPDoer{resp: mockResponse(200, "ok")}
	c := checkers.NewHTTPChecker().WithHTTPDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "http",
		Target: "http://example.com/",
	})
	if !r.Success {
		t.Errorf("Success = false; want true: %s", r.Error)
	}
	if doer.got.Method != http.MethodGet {
		t.Errorf("method = %q, want GET", doer.got.Method)
	}
}

func TestHTTPChecker_Run_404_NotInExpect(t *testing.T) {
	t.Parallel()
	doer := &fakeHTTPDoer{resp: mockResponse(404, "not found")}
	c := checkers.NewHTTPChecker().WithHTTPDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "http",
		Target: "http://example.com/missing",
	})
	if r.Success {
		t.Error("Success = true for 404 with default 2xx expect")
	}
}

func TestHTTPChecker_Run_ExpectStatusMatches(t *testing.T) {
	t.Parallel()
	doer := &fakeHTTPDoer{resp: mockResponse(404, "")}
	c := checkers.NewHTTPChecker().WithHTTPDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "http",
		Target: "http://example.com/x",
		Params: json.RawMessage(`{"expect_status":[404]}`),
	})
	if !r.Success {
		t.Errorf("Success = false; 404 explicitly expected: %s", r.Error)
	}
}

func TestHTTPChecker_Run_BodyMatchPasses(t *testing.T) {
	t.Parallel()
	doer := &fakeHTTPDoer{resp: mockResponse(200, "OK seed is healthy")}
	c := checkers.NewHTTPChecker().WithHTTPDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "http",
		Target: "http://example.com/health",
		Params: json.RawMessage(`{"body_match":"healthy"}`),
	})
	if !r.Success {
		t.Errorf("Success = false; body should match: %s", r.Error)
	}
}

func TestHTTPChecker_Run_BodyMatchFails(t *testing.T) {
	t.Parallel()
	doer := &fakeHTTPDoer{resp: mockResponse(200, "unrelated response")}
	c := checkers.NewHTTPChecker().WithHTTPDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "http",
		Target: "http://example.com/",
		Params: json.RawMessage(`{"body_match":"healthy"}`),
	})
	if r.Success {
		t.Error("Success = true; body_match should fail")
	}
}

func TestHTTPChecker_Run_TransportError(t *testing.T) {
	t.Parallel()
	doer := &fakeHTTPDoer{err: errors.New("connection refused")}
	c := checkers.NewHTTPChecker().WithHTTPDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "http",
		Target: "http://down.example.com/",
	})
	if r.Success {
		t.Error("Success = true; transport error should fail")
	}
}

func TestHTTPChecker_Run_CustomMethodAndHeaders(t *testing.T) {
	t.Parallel()
	doer := &fakeHTTPDoer{resp: mockResponse(200, "")}
	c := checkers.NewHTTPChecker().WithHTTPDoer(doer)
	_ = c.Run(context.Background(), probe.Probe{
		Kind:   "http",
		Target: "http://example.com/",
		Params: json.RawMessage(`{"method":"HEAD","headers":{"X-Probe":"seed"}}`),
	})
	if doer.got.Method != http.MethodHead {
		t.Errorf("method = %q, want HEAD", doer.got.Method)
	}
	if doer.got.Header.Get("X-Probe") != "seed" {
		t.Errorf("X-Probe header = %q, want seed", doer.got.Header.Get("X-Probe"))
	}
}

// httpMeta unmarshals a Result's Metadata into a generic map for assertion.
func httpMeta(t *testing.T, r probe.Result) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Metadata, &m); err != nil {
		t.Fatalf("unmarshal metadata %q: %v", r.Metadata, err)
	}
	return m
}

// TestHTTPChecker_Run_CapturesPhaseTimings runs against a real local
// server so the httptrace hooks fire, and asserts the per-phase timing
// breakdown lands in Result.Metadata. These are the timings the legacy
// /run surfaced on the health-check card (ADR-0027 P3a parity).
func TestHTTPChecker_Run_CapturesPhaseTimings(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := checkers.NewHTTPChecker().Run(context.Background(), probe.Probe{
		Kind:   "http",
		Target: srv.URL,
	})
	if !r.Success {
		t.Fatalf("Success = false; want true: %s", r.Error)
	}
	meta := httpMeta(t, r)
	timings, ok := meta["timings_ms"].(map[string]any)
	if !ok {
		t.Fatalf("metadata has no timings_ms object: %v", meta)
	}
	// TTFB always elapses (the server processes the request); TCP connect
	// elapses for a fresh connection. DNS/TLS may be ~0 for a plain-HTTP
	// IP-literal target, so only assert the always-present phases.
	if ttfb, _ := timings["ttfb"].(float64); ttfb <= 0 {
		t.Errorf("ttfb timing = %v, want > 0", timings["ttfb"])
	}
	if _, present := timings["tcp"]; !present {
		t.Errorf("timings missing tcp phase: %v", timings)
	}
}

// TestHTTPSChecker_Run_CapturesCertInfo runs against a real TLS server
// and asserts the leaf-cert summary the card renders for HTTPS endpoints
// (CN, issuer, days-remaining, negotiated version) lands in Metadata.
func TestHTTPSChecker_Run_CapturesCertInfo(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := checkers.NewHTTPSChecker().Run(context.Background(), probe.Probe{
		Kind:   "https",
		Target: srv.URL,
		// httptest TLS server presents a self-signed cert; let the probe
		// inspect it without verification (matches the TLS checker policy).
		Params: json.RawMessage(`{"insecure_skip_verify":true}`),
	})
	if !r.Success {
		t.Fatalf("Success = false; want true: %s", r.Error)
	}
	meta := httpMeta(t, r)
	cert, ok := meta["tls"].(map[string]any)
	if !ok {
		t.Fatalf("metadata has no tls cert object: %v", meta)
	}
	if cert["tls_version"] == "" || cert["tls_version"] == nil {
		t.Errorf("tls_version missing: %v", cert)
	}
	if _, present := cert["days_remaining"]; !present {
		t.Errorf("days_remaining missing: %v", cert)
	}
}

// TestHTTPChecker_Run_PlainHTTPHasNoCert confirms a plain-HTTP probe
// emits timings but no tls cert object.
func TestHTTPChecker_Run_PlainHTTPHasNoCert(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	r := checkers.NewHTTPChecker().Run(context.Background(), probe.Probe{Kind: "http", Target: srv.URL})
	if _, hasTLS := httpMeta(t, r)["tls"]; hasTLS {
		t.Error("plain-HTTP probe should not carry a tls cert object")
	}
}
