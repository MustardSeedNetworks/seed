package checkers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/probe"
)

// defaultRTSPTimeout is the per-attempt RTSP probe timeout.
const defaultRTSPTimeout = 5 * time.Second

// defaultRTSPPort is the default port for rtsp:// when the URL
// omits one.
const defaultRTSPPort = "554"

// RTSPParams is the kind-specific params shape.
type RTSPParams struct {
	TimeoutMs int `json:"timeout_ms,omitempty"`
}

// RTSPChecker implements probe.Checker for Kind="rtsp". Sends an
// OPTIONS request over TCP and checks for a 200 OK response. The
// existing internal/api/health_checks_rtsp.go uses the same
// approach.
type RTSPChecker struct {
	dialer PingDialer
}

// NewRTSPChecker returns an RTSPChecker wired to a real dialer.
func NewRTSPChecker() *RTSPChecker {
	return &RTSPChecker{dialer: realPingDialer{}}
}

// WithRTSPDialer swaps the dialer for tests.
func (c *RTSPChecker) WithRTSPDialer(d PingDialer) *RTSPChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindRTSP.
func (c *RTSPChecker) Kind() string { return probe.KindRTSP }

// RequiredCapabilities returns nil.
func (c *RTSPChecker) RequiredCapabilities() []string { return nil }

// Run probes Probe.Target with an RTSP OPTIONS request. Target
// may be "host[:port]" or a full "rtsp://host[:port]/path" URL;
// path defaults to "*" if not provided.
func (c *RTSPChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseRTSPParams(p.Params)

	timeout := defaultRTSPTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr, path := parseRTSPTarget(p.Target)

	start := time.Now()
	conn, err := c.dialer.Dial(dialCtx, "tcp", addr)
	if err != nil {
		return probe.Result{
			ProbeID:   p.ID,
			ClientID:  p.ClientID,
			Kind:      p.Kind,
			Timestamp: time.Now().UTC(),
			Success:   false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
			Error:     err.Error(),
		}
	}
	defer func() { _ = conn.Close() }()

	deadline := time.Now().Add(timeout)
	if setErr := conn.SetDeadline(deadline); setErr != nil {
		return probe.Result{
			ProbeID: p.ID, ClientID: p.ClientID, Kind: p.Kind,
			Timestamp: time.Now().UTC(), Success: false,
			Error: "set deadline: " + setErr.Error(),
		}
	}

	req := fmt.Sprintf("OPTIONS %s RTSP/1.0\r\nCSeq: 1\r\nUser-Agent: seed-probe\r\n\r\n", path)
	if _, writeErr := conn.Write([]byte(req)); writeErr != nil {
		return probe.Result{
			ProbeID: p.ID, ClientID: p.ClientID, Kind: p.Kind,
			Timestamp: time.Now().UTC(), Success: false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
			Error:     "write OPTIONS: " + writeErr.Error(),
		}
	}

	reader := bufio.NewReader(conn)
	statusLine, readErr := reader.ReadString('\n')
	latencyMs := float64(time.Since(start).Milliseconds())
	if readErr != nil {
		return probe.Result{
			ProbeID: p.ID, ClientID: p.ClientID, Kind: p.Kind,
			Timestamp: time.Now().UTC(), Success: false,
			LatencyMs: latencyMs,
			Error:     "read response: " + readErr.Error(),
		}
	}

	statusLine = strings.TrimSpace(statusLine)
	const statusLineFields = 3
	parts := strings.SplitN(statusLine, " ", statusLineFields)
	statusOK := len(parts) >= 2 && parts[0] == "RTSP/1.0" && strings.HasPrefix(parts[1], "2")

	meta, _ := json.Marshal(map[string]any{
		"status_line": statusLine,
		"addr":        addr,
	})

	if !statusOK {
		return probe.Result{
			ProbeID: p.ID, ClientID: p.ClientID, Kind: p.Kind,
			Timestamp: time.Now().UTC(), Success: false,
			LatencyMs: latencyMs,
			Error:     "unexpected RTSP status: " + statusLine,
			Metadata:  meta,
		}
	}

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

// parseRTSPTarget returns (host:port, path) for the OPTIONS request.
// Accepts "host", "host:port", or a full rtsp:// URL.
func parseRTSPTarget(target string) (string, string) {
	if strings.HasPrefix(target, "rtsp://") {
		if host, path, ok := parseRTSPURL(target); ok {
			return host, path
		}
	}
	host := target
	if !strings.Contains(host, ":") {
		host += ":" + defaultRTSPPort
	}
	return host, "*"
}

// parseRTSPURL decomposes a full rtsp:// URL. Returns ok=false on
// parse failure.
func parseRTSPURL(target string) (string, string, bool) {
	u, err := url.Parse(target)
	if err != nil {
		return "", "", false
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":" + defaultRTSPPort
	}
	path := u.RequestURI()
	if path == "" || path == "/" {
		path = "*"
	}
	return host, path, true
}

func parseRTSPParams(raw json.RawMessage) RTSPParams {
	if len(raw) == 0 {
		return RTSPParams{}
	}
	var p RTSPParams
	_ = json.Unmarshal(raw, &p)
	return p
}
