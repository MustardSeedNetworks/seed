package checkers

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultUDPProbeTimeout is the per-attempt UDP probe timeout.
const defaultUDPProbeTimeout = 3 * time.Second

// UDPParams is the kind-specific params shape. UDP probing is
// inherently unreliable (no connection establishment); the probe
// reports Success=true if no error occurred during write+read
// attempts, false otherwise. Use kind="dns" for true protocol-aware
// UDP probes (DNS).
type UDPParams struct {
	Port      int    `json:"port"`
	Payload   string `json:"payload,omitempty"`    // optional payload to send
	TimeoutMs int    `json:"timeout_ms,omitempty"` // default 3000
}

// UDPProber is the test seam for UDPChecker. Production wires it
// to [net.DialUDP] via realUDPProber.
type UDPProber interface {
	Probe(ctx context.Context, addr string, payload []byte, timeout time.Duration) error
}

// UDPChecker implements probe.Checker for Kind="udp".
type UDPChecker struct {
	prober UDPProber
}

// NewUDPChecker returns a UDPChecker wired to the production prober.
func NewUDPChecker() *UDPChecker {
	return &UDPChecker{prober: realUDPProber{}}
}

// WithUDPProber swaps the prober (for tests).
func (c *UDPChecker) WithUDPProber(p UDPProber) *UDPChecker {
	c.prober = p
	return c
}

// Kind returns probe.KindUDP.
func (c *UDPChecker) Kind() string { return probe.KindUDP }

// RequiredCapabilities returns nil.
func (c *UDPChecker) RequiredCapabilities() []string { return nil }

// Run sends an optional payload to Target:Port and reports latency.
// UDP is connectionless; "success" means the OS accepted the
// write + (if payload sent) we did not error reading a response
// before timeout.
func (c *UDPChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseUDPParams(p.Params)
	if params.Port <= 0 || params.Port > 65535 {
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: time.Now().UTC(),
			Success:   false,
			Error:     "udp probe requires params.port in range 1-65535",
		}
	}

	timeout := defaultUDPProbeTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}

	addr := net.JoinHostPort(p.Target, strconv.Itoa(params.Port))
	start := time.Now()
	err := c.prober.Probe(ctx, addr, []byte(params.Payload), timeout)
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

// realUDPProber dials a UDP socket and (if payload given) writes
// it. A UDP "success" in V1.0 means the OS routed the packet
// without immediate error — semantics matching the legacy
// internal/api/health_checks_port.go behavior.
type realUDPProber struct{}

// Probe sends payload to addr with deadline; returns nil on
// success, error on dial / write / read failure.
func (realUDPProber) Probe(ctx context.Context, addr string, payload []byte, timeout time.Duration) error {
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "udp", addr)
	if err != nil {
		return errors.Join(errors.New("udp dial"), err)
	}
	defer func() { _ = conn.Close() }()

	deadline := time.Now().Add(timeout)
	if setErr := conn.SetDeadline(deadline); setErr != nil {
		return errors.Join(errors.New("udp set deadline"), setErr)
	}

	// If no payload, the dial alone is the probe.
	if len(payload) == 0 {
		return nil
	}

	if _, writeErr := conn.Write(payload); writeErr != nil {
		return errors.Join(errors.New("udp write"), writeErr)
	}

	// Best-effort read: many UDP services don't echo, so we
	// accept "no response before deadline" as success when a
	// payload was sent. We only fail on hard errors (refused, etc).
	buf := make([]byte, 1)
	_, readErr := conn.Read(buf)
	if readErr != nil {
		var netErr net.Error
		if errors.As(readErr, &netErr) && netErr.Timeout() {
			// Timeout reading is OK — UDP service may be one-way.
			return nil
		}
		return errors.Join(errors.New("udp read"), readErr)
	}
	return nil
}

func parseUDPParams(raw json.RawMessage) UDPParams {
	if len(raw) == 0 {
		return UDPParams{}
	}
	var p UDPParams
	_ = json.Unmarshal(raw, &p)
	return p
}
