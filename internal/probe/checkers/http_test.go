package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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
