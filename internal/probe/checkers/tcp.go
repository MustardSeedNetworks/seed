package checkers

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultTCPDialTimeout is the default per-attempt dial timeout.
const defaultTCPDialTimeout = 5 * time.Second

// metaKeyAddr is the shared Result.Metadata key for the dialed
// host:port address. Checkers across this package surface it so the
// probe result shows exactly which endpoint was contacted.
const metaKeyAddr = "addr"

// TCPParams is the kind-specific params shape. Port is required;
// TimeoutMs overrides the default dial timeout.
type TCPParams struct {
	Port      int `json:"port"`
	TimeoutMs int `json:"timeout_ms,omitempty"`
}

// TCPChecker implements probe.Checker for Kind="tcp" — dials a TCP
// port on Probe.Target and reports reachability + latency.
type TCPChecker struct {
	dialer PingDialer // shares the TCP dialer interface with PingChecker
}

// NewTCPChecker returns a TCPChecker wired to a real [net.Dialer].
func NewTCPChecker() *TCPChecker {
	return &TCPChecker{dialer: realPingDialer{}}
}

// WithTCPDialer swaps the dialer; used by tests.
func (c *TCPChecker) WithTCPDialer(d PingDialer) *TCPChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindTCP.
func (c *TCPChecker) Kind() string { return probe.KindTCP }

// RequiredCapabilities returns nil; TCP dial needs no special
// capability.
func (c *TCPChecker) RequiredCapabilities() []string { return nil }

// Run dials Probe.Target on Params.Port and returns Success=true
// with latency on connect, false with Error on failure.
func (c *TCPChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseTCPParams(p.Params)
	if params.Port <= 0 || params.Port > 65535 {
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Error:     "tcp probe requires params.port in range 1-65535",
		}
	}

	timeout := defaultTCPDialTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(p.Target, strconv.Itoa(params.Port))
	start := time.Now()
	conn, err := c.dialer.Dial(dialCtx, "tcp", addr)
	latencyMs := float64(time.Since(start).Milliseconds())

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

	meta, _ := json.Marshal(map[string]any{metaKeyAddr: addr})
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

func parseTCPParams(raw json.RawMessage) TCPParams {
	if len(raw) == 0 {
		return TCPParams{}
	}
	var p TCPParams
	_ = json.Unmarshal(raw, &p)
	return p
}
