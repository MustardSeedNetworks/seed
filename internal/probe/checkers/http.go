package checkers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"slices"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultHTTPTimeout is the default request timeout.
const defaultHTTPTimeout = 10 * time.Second

// maxResponseBodyBytes caps how much of the response body the
// checker reads when matching body patterns. 1 MiB is generous.
const maxResponseBodyBytes = 1 << 20

// microsPerMilli converts microseconds to milliseconds as a float so the
// per-phase timings keep sub-millisecond precision (localhost handshakes
// round to zero at whole-millisecond resolution).
const microsPerMilli = 1000.0

// bodySnippetMaxBytes caps the on-failure body snippet included
// in the Result.Error field.
const bodySnippetMaxBytes = 200

// HTTPParams is the kind-specific params shape. Probe.Target is
// the URL (host[:port]/path); Params.Method defaults to GET.
// BodyMatch enables a substring check on the response body.
type HTTPParams struct {
	Method             string            `json:"method,omitempty"`               // default GET
	Headers            map[string]string `json:"headers,omitempty"`              // optional request headers
	ExpectStatus       []int             `json:"expect_status,omitempty"`        // empty = "any 2xx"
	BodyMatch          string            `json:"body_match,omitempty"`           // optional substring assertion
	FollowRedirects    bool              `json:"follow_redirects,omitempty"`     // default false
	TimeoutMs          int               `json:"timeout_ms,omitempty"`           // default 10000
	InsecureSkipVerify bool              `json:"insecure_skip_verify,omitempty"` // tls cert verification (https only)
}

// HTTPDoer is the test seam — [http.Client] implements it.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPChecker implements probe.Checker for Kind="http" (and "https"
// — same checker registered under both kinds, scheme comes from
// Probe.Target).
type HTTPChecker struct {
	kind  string
	doer  HTTPDoer
	clock func() time.Time
}

// NewHTTPChecker returns an HTTPChecker for kind="http". The
// production doer is created per-Run from the params (timeout,
// follow-redirects, TLS) so each probe gets isolated client state.
func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{kind: probe.KindHTTP, clock: time.Now}
}

// NewHTTPSChecker returns an HTTPChecker for kind="https" — same
// implementation, different Kind reported.
func NewHTTPSChecker() *HTTPChecker {
	return &HTTPChecker{kind: probe.KindHTTPS, clock: time.Now}
}

// WithHTTPDoer overrides the default per-request client construction;
// used by tests to inject a fake.
func (c *HTTPChecker) WithHTTPDoer(d HTTPDoer) *HTTPChecker {
	c.doer = d
	return c
}

// Kind returns the configured kind ("http" or "https").
func (c *HTTPChecker) Kind() string { return c.kind }

// RequiredCapabilities returns nil.
func (c *HTTPChecker) RequiredCapabilities() []string { return nil }

// Run executes the HTTP probe. Returns Success=true when the
// response status matches ExpectStatus (or 2xx if not specified)
// AND, when BodyMatch is set, the body contains that substring.
func (c *HTTPChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseHTTPParams(p.Params)

	timeout := defaultHTTPTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	method := strings.ToUpper(params.Method)
	if method == "" {
		method = http.MethodGet
	}

	tr := &httpTrace{}
	traceCtx := httptrace.WithClientTrace(reqCtx, tr.clientTrace())
	req, err := http.NewRequestWithContext(traceCtx, method, p.Target, http.NoBody)
	if err != nil {
		return c.failResult(p, 0, fmt.Sprintf("invalid request: %v", err))
	}
	for k, v := range params.Headers {
		req.Header.Set(k, v)
	}

	doer := c.doer
	if doer == nil {
		doer = buildHTTPClient(timeout, params)
	}

	start := c.clock()
	resp, err := doer.Do(req)
	latencyMs := float64(c.clock().Sub(start).Milliseconds())
	if err != nil {
		return c.failResult(p, latencyMs, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	statusOK := matchExpectedStatus(resp.StatusCode, params.ExpectStatus)
	bodyOK := true
	var bodySnippet string
	if params.BodyMatch != "" {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
		if readErr != nil {
			return c.failResult(p, latencyMs, "read body: "+readErr.Error())
		}
		bodyOK = strings.Contains(string(body), params.BodyMatch)
		if !bodyOK {
			bodySnippet = truncate(string(body), bodySnippetMaxBytes)
		}
	}

	metaMap := map[string]any{
		"status_code":  resp.StatusCode,
		"content_type": resp.Header.Get("Content-Type"),
		"body_match":   params.BodyMatch != "" && bodyOK,
	}
	if timings := tr.timings(start); len(timings) > 0 {
		metaMap["timings_ms"] = timings
	}
	if cert, ok := certInfoFromResponse(resp, req.URL.Hostname()); ok {
		metaMap["tls"] = cert
	}
	meta, _ := json.Marshal(metaMap)

	if !statusOK || !bodyOK {
		errMsg := fmt.Sprintf("status %d", resp.StatusCode)
		if !bodyOK {
			errMsg += "; body_match=false; snippet=" + bodySnippet
		}
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: c.clock().UTC(),
			Success:   false,
			LatencyMs: latencyMs,
			Error:     errMsg,
			Metadata:  meta,
		}
	}

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: c.clock().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  meta,
	}
}

// failResult builds a failure Result with the given message.
func (c *HTTPChecker) failResult(p probe.Probe, latencyMs float64, msg string) probe.Result {
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: c.clock().UTC(),
		Success:   false,
		LatencyMs: latencyMs,
		Error:     msg,
	}
}

// minSuccessfulStatus is the lower bound for an implicit
// 2xx-default match.
const minSuccessfulStatus = 200

// maxSuccessfulStatus is the (exclusive) upper bound.
const maxSuccessfulStatus = 300

// matchExpectedStatus returns true when actual matches one of the
// expected codes. Empty expected means "any 2xx response".
func matchExpectedStatus(actual int, expected []int) bool {
	if len(expected) == 0 {
		return actual >= minSuccessfulStatus && actual < maxSuccessfulStatus
	}
	return slices.Contains(expected, actual)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func parseHTTPParams(raw json.RawMessage) HTTPParams {
	if len(raw) == 0 {
		return HTTPParams{}
	}
	var p HTTPParams
	_ = json.Unmarshal(raw, &p)
	return p
}

// buildHTTPClient constructs a per-probe [http.Client] honoring the
// params' timeout + follow-redirects + TLS-insecure flags. Returned
// as an HTTPDoer for the checker's purposes.
func buildHTTPClient(timeout time.Duration, params HTTPParams) HTTPDoer {
	transport, _ := http.DefaultTransport.(*http.Transport)
	transport = transport.Clone()
	if params.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // operator-opt-in
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	if !params.FollowRedirects {
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return client
}

// httpTrace records the connection-phase timestamps captured by an
// [httptrace.ClientTrace] during a single request, so the checker can
// publish a DNS/TCP/TLS/TTFB latency breakdown — the per-phase timing
// the legacy /run surfaced on the health-check card (ADR-0027 P3a).
type httpTrace struct {
	dnsStart, dnsDone   time.Time
	connStart, connDone time.Time
	tlsStart, tlsDone   time.Time
	firstByte           time.Time
}

// clientTrace returns the trace whose hooks stamp the phase timestamps.
// Connection reuse means DNS/connect/TLS hooks may not fire; the timings
// helper emits only the phases that were actually measured.
func (h *httpTrace) clientTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart:             func(httptrace.DNSStartInfo) { h.dnsStart = time.Now() },
		DNSDone:              func(httptrace.DNSDoneInfo) { h.dnsDone = time.Now() },
		ConnectStart:         func(_, _ string) { h.connStart = time.Now() },
		ConnectDone:          func(_, _ string, _ error) { h.connDone = time.Now() },
		TLSHandshakeStart:    func() { h.tlsStart = time.Now() },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { h.tlsDone = time.Now() },
		GotFirstResponseByte: func() { h.firstByte = time.Now() },
	}
}

// timings returns the measured per-phase durations in milliseconds. start
// is the wall-clock instant the request was dispatched (the TTFB baseline).
// Phases that did not fire (e.g. TLS on a plain-HTTP request, or any phase
// on a reused connection) are omitted rather than reported as zero.
func (h *httpTrace) timings(start time.Time) map[string]float64 {
	m := map[string]float64{}
	add := func(key string, from, to time.Time) {
		if !from.IsZero() && !to.IsZero() && to.After(from) {
			m[key] = float64(to.Sub(from).Microseconds()) / microsPerMilli
		}
	}
	add("dns", h.dnsStart, h.dnsDone)
	add("tcp", h.connStart, h.connDone)
	add("tls", h.tlsStart, h.tlsDone)
	if !h.firstByte.IsZero() && !start.IsZero() && h.firstByte.After(start) {
		m["ttfb"] = float64(h.firstByte.Sub(start).Microseconds()) / microsPerMilli
	}
	return m
}

// certInfoFromResponse extracts the leaf-certificate summary from a TLS
// response, reusing the same shape the TLS checker publishes so a single
// cert-extraction path serves both kinds. Returns ok=false for plain HTTP.
func certInfoFromResponse(resp *http.Response, sni string) (TLSCertInfo, bool) {
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return TLSCertInfo{}, false
	}
	return extractCertInfo(TLSConnState{
		PeerCertificates: resp.TLS.PeerCertificates,
		Version:          resp.TLS.Version,
	}, sni), true
}
