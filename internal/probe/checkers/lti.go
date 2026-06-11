package checkers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultLTITimeout is the per-request timeout for LTI probes (mirrors
// the legacy LTITimeout constant in internal/api/handlers_industry_checks.go).
const defaultLTITimeout = 10 * time.Second

// ltiSuccessMin / ltiSuccessMax define the half-open status range that
// counts as success: [200, 400). LTI launch URLs frequently redirect, so
// all 3xx responses are intentionally included (legacy §185 comment).
const (
	ltiSuccessMin = 200
	ltiSuccessMax = 400
)

// LTIParams is the kind-specific params shape. Probe.Target is the LTI
// launch URL. LTIVersion is carried through to metadata only; it does
// not alter the probe logic. TimeoutMs overrides the default 10 s.
type LTIParams struct {
	LTIVersion string `json:"lti_version,omitempty"` // informational (1.1 / 1.3 / advantage)
	TimeoutMs  int    `json:"timeout_ms,omitempty"`  // default 10000
}

// LTIChecker implements probe.Checker for Kind="lti". It issues an HTTP
// HEAD request to Probe.Target and reports success for any 2xx/3xx
// response, mirroring the legacy testLTIEndpoint behaviour.
type LTIChecker struct {
	doer  HTTPDoer
	clock func() time.Time
}

// NewLTIChecker returns an LTIChecker backed by the default production
// [http.Client]. The client is constructed per Run call so each probe gets
// isolated state (timeout, redirect chain).
func NewLTIChecker() *LTIChecker {
	return &LTIChecker{clock: time.Now}
}

// WithLTIDoer replaces the default per-request client; used by tests to
// inject a fake without needing a real network.
func (c *LTIChecker) WithLTIDoer(d HTTPDoer) *LTIChecker {
	c.doer = d
	return c
}

// Kind returns probe.KindLTI.
func (c *LTIChecker) Kind() string { return probe.KindLTI }

// RequiredCapabilities returns nil; LTI probes need no special hardware.
func (c *LTIChecker) RequiredCapabilities() []string { return nil }

// Run issues a HEAD request to Probe.Target and reports success when the
// response status falls in [200, 400). SSL validity is captured for
// https targets. Transport errors always fail.
func (c *LTIChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := ltiParseParams(p.Params)

	timeout := defaultLTITimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, p.Target, http.NoBody)
	if err != nil {
		return ltiFailure(p, c.clock().UTC(), 0, fmt.Sprintf("invalid request: %v", err))
	}

	doer := c.doer
	if doer == nil {
		doer = ltiDefaultClient(timeout)
	}

	start := c.clock()
	resp, err := doer.Do(req)
	latencyMs := float64(c.clock().Sub(start).Milliseconds())
	if err != nil {
		return ltiFailure(p, c.clock().UTC(), latencyMs, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	// Capture SSL validity for https targets (non-https => sslValid stays false).
	sslValid := resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 &&
		strings.HasPrefix(p.Target, "https://")

	meta, _ := json.Marshal(map[string]any{
		"status_code": resp.StatusCode,
		"lti_version": params.LTIVersion,
		"ssl_valid":   sslValid,
	})

	if resp.StatusCode < ltiSuccessMin || resp.StatusCode >= ltiSuccessMax {
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: c.clock().UTC(),
			Success:   false,
			LatencyMs: latencyMs,
			Error:     fmt.Sprintf("Unexpected status: %d", resp.StatusCode),
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

// ltiFailure builds a failure Result with no metadata.
func ltiFailure(p probe.Probe, ts time.Time, latencyMs float64, msg string) probe.Result {
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: ts,
		Success:   false,
		LatencyMs: latencyMs,
		Error:     msg,
	}
}

// ltiDefaultClient returns a production [http.Client] that follows up to
// ten redirects (matching legacy ltiMaxRedirects) with the given timeout.
func ltiDefaultClient(timeout time.Duration) HTTPDoer {
	const maxRedirects = 10
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("lti: exceeded %d redirects", maxRedirects)
			}
			return nil
		},
	}
}

// ltiParseParams unmarshals the raw JSON params; returns zero-value on
// absent or malformed input.
func ltiParseParams(raw json.RawMessage) LTIParams {
	if len(raw) == 0 {
		return LTIParams{}
	}
	var p LTIParams
	_ = json.Unmarshal(raw, &p)
	return p
}
