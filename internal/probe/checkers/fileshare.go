package checkers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultFileShareTimeout is the per-attempt TCP dial timeout.
const defaultFileShareTimeout = 10 * time.Second

// File share protocol constants (SMB RFC 1001/1002, NFS RFC 7530).
const (
	fileShareProtocolSMB = "smb"
	fileShareProtocolNFS = "nfs"

	fileShareDefaultSMBPort = 445  // SMB over TCP (direct hosting, IANA 445)
	fileShareDefaultNFSPort = 2049 // NFS (IANA 2049)
)

// FileShareParams is the kind-specific params shape. Protocol selects the
// port; Share is surfaced in metadata only (no mount or filesystem I/O).
// TimeoutMs overrides the default dial timeout.
//
// V1.0 scope: TCP reachability only. Protocol-level mount, read, and write
// performance tests are out of scope — those require the share to be
// OS-mounted at a local path, which is fragile in a network-probe context.
type FileShareParams struct {
	Protocol  string `json:"protocol,omitempty"`   // "smb"|"nfs", default "smb"
	Share     string `json:"share,omitempty"`      // e.g. "//host/sharename" (metadata only)
	TimeoutMs int    `json:"timeout_ms,omitempty"` // default 10000
}

// FileShareChecker implements probe.Checker for Kind="fileshare". It dials
// the SMB (445) or NFS (2049) TCP port on Probe.Target and reports
// reachability + latency. Protocol-level session negotiation and filesystem
// I/O are intentionally out of scope for V1.0.
type FileShareChecker struct {
	dialer PingDialer
}

// NewFileShareChecker returns a FileShareChecker wired to a real dialer.
func NewFileShareChecker() *FileShareChecker {
	return &FileShareChecker{dialer: realPingDialer{}}
}

// WithFileShareDialer swaps the dialer (for tests).
func (c *FileShareChecker) WithFileShareDialer(d PingDialer) *FileShareChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindFileShare.
func (c *FileShareChecker) Kind() string { return probe.KindFileShare }

// RequiredCapabilities returns nil; TCP dial needs no special capability.
func (c *FileShareChecker) RequiredCapabilities() []string { return nil }

// Run dials the protocol-appropriate TCP port on Probe.Target and returns
// Success=true with latency on connect, false with Error on failure.
func (c *FileShareChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseFileShareParams(p.Params)

	// Resolve port by protocol; empty protocol defaults to SMB.
	protocol := strings.ToLower(params.Protocol)
	if protocol == "" {
		protocol = fileShareProtocolSMB
	}

	var port int
	switch protocol {
	case fileShareProtocolSMB:
		port = fileShareDefaultSMBPort
	case fileShareProtocolNFS:
		port = fileShareDefaultNFSPort
	default:
		return fileShareFailure(p, 0, fmt.Sprintf("Unsupported protocol: %s", params.Protocol))
	}

	timeout := defaultFileShareTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(p.Target, strconv.Itoa(port))
	start := time.Now()
	conn, err := c.dialer.Dial(dialCtx, "tcp", addr)
	latencyMs := float64(time.Since(start).Milliseconds())

	if err != nil {
		return fileShareFailure(p, time.Duration(latencyMs)*time.Millisecond, err.Error())
	}
	_ = conn.Close()

	meta, _ := json.Marshal(map[string]any{
		"protocol":  protocol,
		metaKeyAddr: addr,
		"share":     params.Share,
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

// fileShareFailure builds a failed Result with the measured latency so far.
func fileShareFailure(p probe.Probe, elapsed time.Duration, msg string) probe.Result {
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

func parseFileShareParams(raw json.RawMessage) FileShareParams {
	if len(raw) == 0 {
		return FileShareParams{}
	}
	var p FileShareParams
	_ = json.Unmarshal(raw, &p)
	return p
}
