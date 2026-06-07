package checkers

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultPingDialTimeout is the per-attempt dial timeout. Probes
// that need a tighter or looser bound override via Params.
const defaultPingDialTimeout = 3 * time.Second

// pingPrimaryPort is tried first; on failure pingFallbackPort
// is tried before declaring the host unreachable. This mirrors
// the existing internal/api/health_checks_ping.go behavior of
// using TCP reachability (port 80 → 443) as the ICMP substitute
// because seed deployments often lack CAP_NET_RAW.
const (
	pingPrimaryPort  = "80"
	pingFallbackPort = "443"
)

// PingParams is the kind-specific params shape. Empty values fall
// back to defaults.
type PingParams struct {
	// Port overrides the primary TCP port to dial. When non-empty,
	// the fallback port is NOT tried — the operator chose this
	// port deliberately.
	Port string `json:"port,omitempty"`

	// TimeoutMs overrides the per-attempt dial timeout. Default
	// 3000.
	TimeoutMs int `json:"timeout_ms,omitempty"`
}

// PingDialer is the test seam for PingChecker — production wires
// it to [net.Dialer].
type PingDialer interface {
	Dial(ctx context.Context, network, addr string) (net.Conn, error)
}

// PingChecker implements probe.Checker for Kind="ping". Reports
// reachability + dial latency via TCP fallback (the same ICMP
// substitute the legacy internal/api/health_checks_ping.go uses).
type PingChecker struct {
	dialer PingDialer
}

// NewPingChecker returns a PingChecker wired to a real [net.Dialer]
// via a small adapter that surfaces DialContext as the Dial method.
func NewPingChecker() *PingChecker {
	return &PingChecker{dialer: realPingDialer{}}
}

// realPingDialer wraps [net.Dialer] to expose DialContext as a Dial
// method matching the PingDialer interface.
type realPingDialer struct{}

// Dial honors ctx-based deadlines via [net.Dialer.DialContext].
func (realPingDialer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{}
	return d.DialContext(ctx, network, addr)
}

// WithPingDialer swaps the dialer (for tests).
func (c *PingChecker) WithPingDialer(d PingDialer) *PingChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindPing.
func (c *PingChecker) Kind() string { return probe.KindPing }

// RequiredCapabilities returns nil — TCP fallback needs no special
// capability. A future ICMP-only variant would require "raw_sockets".
func (c *PingChecker) RequiredCapabilities() []string { return nil }

// Run dials Probe.Target on the configured (or default) TCP port,
// times the dial, and returns a Result. Falls back to port 443
// only when the operator did NOT pin a port via Params.Port.
func (c *PingChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parsePingParams(p.Params)

	timeout := defaultPingDialTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	primaryPort := params.Port
	if primaryPort == "" {
		primaryPort = pingPrimaryPort
	}

	addr := net.JoinHostPort(p.Target, primaryPort)
	start := time.Now()
	conn, err := c.dialer.Dial(dialCtx, "tcp", addr)
	latencyMs := float64(time.Since(start).Milliseconds())

	fellBack := false
	// Only attempt fallback when the operator did NOT pin a port.
	if err != nil && params.Port == "" {
		fellBack = true
		addr = net.JoinHostPort(p.Target, pingFallbackPort)
		start = time.Now()
		conn, err = c.dialer.Dial(dialCtx, "tcp", addr)
		latencyMs = float64(time.Since(start).Milliseconds())
	}

	if err != nil {
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: time.Now().UTC(),
			Success:   false,
			LatencyMs: latencyMs,
			Error:     err.Error(),
		}
	}
	_ = conn.Close()

	meta, _ := json.Marshal(map[string]any{
		"dialed_addr": addr,
		"fell_back":   fellBack,
	})

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  meta,
	}
}

// parsePingParams decodes the params JSON; returns zero on empty
// or unparseable input.
func parsePingParams(raw json.RawMessage) PingParams {
	if len(raw) == 0 {
		return PingParams{}
	}
	var p PingParams
	_ = json.Unmarshal(raw, &p)
	return p
}
