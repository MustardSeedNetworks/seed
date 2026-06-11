package checkers_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// fakeLTIDoer captures the outbound request and returns a canned response.
type fakeLTIDoer struct {
	resp *http.Response
	err  error
	got  *http.Request
}

func (f *fakeLTIDoer) Do(req *http.Request) (*http.Response, error) {
	f.got = req
	return f.resp, f.err
}

// ltiMockResponse builds a minimal *[http.Response] with the given status.
func ltiMockResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

// ltiMockTLSResponse builds a response with a populated TLS state so
// that ssl_valid is captured as true.
func ltiMockTLSResponse(status int) *http.Response {
	resp := ltiMockResponse(status)
	resp.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{{}},
	}
	return resp
}

func TestLTIChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewLTIChecker().Kind() != "lti" {
		t.Errorf("LTIChecker.Kind = %q; want %q", checkers.NewLTIChecker().Kind(), "lti")
	}
}

func TestLTIChecker_Run_200_Success(t *testing.T) {
	t.Parallel()
	doer := &fakeLTIDoer{resp: ltiMockResponse(200)}
	c := checkers.NewLTIChecker().WithLTIDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindLTI,
		Target: "https://lms.example.com/lti/launch",
	})
	if !r.Success {
		t.Errorf("Success = false; want true: %s", r.Error)
	}
	if doer.got == nil || doer.got.Method != http.MethodHead {
		t.Errorf("method = %q; want HEAD", doer.got.Method)
	}
}

func TestLTIChecker_Run_302_Success(t *testing.T) {
	t.Parallel()
	// 3xx is intentionally a success for LTI (endpoints often redirect to
	// a login page before the actual launch).
	doer := &fakeLTIDoer{resp: ltiMockResponse(302)}
	c := checkers.NewLTIChecker().WithLTIDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindLTI,
		Target: "https://lms.example.com/lti/launch",
	})
	if !r.Success {
		t.Errorf("Success = false for 302; want true: %s", r.Error)
	}
}

func TestLTIChecker_Run_500_Failure(t *testing.T) {
	t.Parallel()
	doer := &fakeLTIDoer{resp: ltiMockResponse(500)}
	c := checkers.NewLTIChecker().WithLTIDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindLTI,
		Target: "https://lms.example.com/lti/launch",
	})
	if r.Success {
		t.Error("Success = true for 500; want false")
	}
	if !strings.Contains(r.Error, "500") {
		t.Errorf("Error = %q; want it to contain status code 500", r.Error)
	}
}

func TestLTIChecker_Run_TransportError_Failure(t *testing.T) {
	t.Parallel()
	doer := &fakeLTIDoer{err: errors.New("connection refused")}
	c := checkers.NewLTIChecker().WithLTIDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindLTI,
		Target: "https://lms.example.com/lti/launch",
	})
	if r.Success {
		t.Error("Success = true on transport error; want false")
	}
	if r.Error == "" {
		t.Error("Error is empty; want a descriptive message")
	}
}

func TestLTIChecker_Run_SSLValid_Captured(t *testing.T) {
	t.Parallel()
	doer := &fakeLTIDoer{resp: ltiMockTLSResponse(200)}
	c := checkers.NewLTIChecker().WithLTIDoer(doer)
	r := c.Run(context.Background(), probe.Probe{
		Kind:   probe.KindLTI,
		Target: "https://lms.example.com/lti/launch",
	})
	if !r.Success {
		t.Fatalf("Success = false; want true: %s", r.Error)
	}
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata unmarshal: %v", err)
	}
	if meta["ssl_valid"] != true {
		t.Errorf("ssl_valid = %v; want true", meta["ssl_valid"])
	}
}
