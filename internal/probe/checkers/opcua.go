package checkers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultOPCUATimeout is the per-attempt OPC-UA probe timeout.
const defaultOPCUATimeout = 10 * time.Second

// defaultOPCUAPort is the default OPC-UA TCP port (IANA 4840).
const defaultOPCUAPort = 4840

// opcuaReadDeadlineWindow is the short read deadline applied after the
// Hello write. A timeout here is NOT a failure — TCP already succeeded.
const opcuaReadDeadlineWindow = 2 * time.Second

// opcuaResponseBufferSize bounds a single read of the OPC-UA response.
const opcuaResponseBufferSize = 256

// opcuaMinResponseLen is the minimum byte count to inspect the message type.
const opcuaMinResponseLen = 4

// opcuaHello is the 4-byte minimal OPC-UA Hello prefix (OPC-UA Part 6
// §7.1.2, "HEL" message type + "F" final-chunk flag).
const opcuaHello = "HELF"

// OPCUAParams is the kind-specific params shape. All fields are optional.
type OPCUAParams struct {
	// SecurityMode is informational only (None / Sign / SignAndEncrypt).
	// V1.0 does not negotiate a secure channel; it is stored in metadata.
	SecurityMode string `json:"security_mode,omitempty"`

	// TimeoutMs overrides the per-attempt dial timeout. Default 10000.
	TimeoutMs int `json:"timeout_ms,omitempty"`
}

// OPCUAChecker implements probe.Checker for Kind="opcua". It opens a
// TCP connection to the OPC-UA server, writes a 4-byte Hello prefix,
// and reads whatever the server sends back.
//
// Scoping (V1.0): this is a shallow TCP + minimal Hello probe. It does
// NOT perform a full OPC-UA OpenSecureChannel / GetEndpoints handshake.
// TCP connect success is the pass signal; ACK/ERR classification is
// informational. Operators needing the full handshake should use an
// OPC-UA client library outside this probe.
type OPCUAChecker struct {
	dialer PingDialer
}

// NewOPCUAChecker returns an OPCUAChecker wired to a real dialer.
func NewOPCUAChecker() *OPCUAChecker {
	return &OPCUAChecker{dialer: realPingDialer{}}
}

// WithOPCUADialer swaps the dialer (for tests).
func (c *OPCUAChecker) WithOPCUADialer(d PingDialer) *OPCUAChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindOPCUA.
func (c *OPCUAChecker) Kind() string { return probe.KindOPCUA }

// RequiredCapabilities returns nil; OPC-UA TCP needs no special hardware.
func (c *OPCUAChecker) RequiredCapabilities() []string { return nil }

// Run connects to the OPC-UA TCP port derived from Probe.Target
// (expected form: opc.tcp://host:port/path), writes a minimal Hello
// prefix, and classifies the server's response. TCP connect success
// means Success=true; only a dial error, URL parse error, or write
// error is a hard failure.
func (c *OPCUAChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseOPCUAParams(p.Params)

	timeout := defaultOPCUATimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}

	// Parse opc.tcp://host:port/path — net/url handles the scheme.
	parsedURL, parseErr := url.Parse(p.Target)
	if parseErr != nil {
		return opcuaFailure(p, 0, "parse target URL: "+parseErr.Error())
	}
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = strconv.Itoa(defaultOPCUAPort)
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(host, port)
	start := time.Now()
	conn, err := c.dialer.Dial(dialCtx, "tcp", addr)
	if err != nil {
		return opcuaFailure(p, time.Since(start), "connection failed: "+err.Error())
	}
	defer func() { _ = conn.Close() }()

	// Write the Hello prefix. A write error means the connection dropped
	// immediately — treat as a failure.
	if _, writeErr := conn.Write([]byte(opcuaHello)); writeErr != nil {
		return opcuaFailure(p, time.Since(start), "send Hello: "+writeErr.Error())
	}

	// Apply a short read deadline. A timeout or closed connection here is
	// NOT a failure: the TCP connection already succeeded, meaning the
	// server is reachable. Some servers reject the malformed Hello and
	// close cleanly; that is still a connectivity success.
	_ = conn.SetReadDeadline(time.Now().Add(opcuaReadDeadlineWindow))
	respBuf := make([]byte, opcuaResponseBufferSize)
	n, readErr := conn.Read(respBuf)

	latencyMs := float64(time.Since(start).Milliseconds())

	serverInfo, msgType := opcuaClassifyResponse(n, readErr, respBuf)

	meta := map[string]any{
		metaKeyAddr:   addr,
		"server_info": serverInfo,
	}
	if msgType != "" {
		meta["msg_type"] = msgType
	}
	if params.SecurityMode != "" {
		meta["security_mode"] = params.SecurityMode
	}
	metaJSON, _ := json.Marshal(meta)

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  metaJSON,
	}
}

// opcuaClassifyResponse interprets the first bytes of the server
// response and returns a human-readable server_info string plus the
// raw 3-byte message type (empty when there was no readable response).
func opcuaClassifyResponse(n int, readErr error, buf []byte) (string, string) {
	const tcpOnly = "TCP connection successful, server may require full handshake"
	if readErr != nil {
		// Timeout or connection close after Hello — TCP is fine.
		return tcpOnly, ""
	}
	if n < opcuaMinResponseLen {
		// Too short to classify; still a TCP success.
		return tcpOnly, ""
	}
	msgType := string(buf[:3])
	switch msgType {
	case "ACK":
		return "OPC-UA server acknowledged connection", msgType
	case "ERR":
		return "OPC-UA server returned error (authentication may be required)", msgType
	default:
		return fmt.Sprintf("server responded with: %s", msgType), msgType
	}
}

// opcuaFailure builds a failed Result with the measured latency so far.
func opcuaFailure(p probe.Probe, elapsed time.Duration, msg string) probe.Result {
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   false,
		LatencyMs: float64(elapsed.Milliseconds()),
		Error:     msg,
	}
}

func parseOPCUAParams(raw json.RawMessage) OPCUAParams {
	if len(raw) == 0 {
		return OPCUAParams{}
	}
	var p OPCUAParams
	_ = json.Unmarshal(raw, &p)
	return p
}
