//go:build !windows

package discovery

// Traceroute support enables path tracing to determine the network route (hop sequence)
// that packets take to reach a target host. Supports ICMP, UDP, and TCP-based traceroute
// for mapping network topology and identifying intermediate infrastructure.

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"syscall"
	"time"
)

// errTracerouteCanceled is the error a traceroute reports when its context is
// cancelled. Hop states live in hopstate.go, shared with the Windows tracer.
const errTracerouteCanceled = "traceroute canceled"

// Traceroute timing and buffer constants.
const (
	traceDNSResolveTimeoutS = 5    // Timeout in seconds for DNS resolution
	tracePTRResolveTimeoutS = 2    // Timeout in seconds for PTR lookup
	traceICMPBufferSize     = 1500 // Buffer size for ICMP reply packets

	// defaultMaxHops is used when the caller passes 0 (unset).
	defaultMaxHops = 30
	// maxAllowedHops bounds Tracer.maxHops so the hop-slice allocation in
	// initTracerouteResult can never grow unbounded from a caller-supplied
	// value (CWE-770). API handlers already clamp to this same ceiling, but
	// the clamp lives here too so NewTracer is safe for any caller.
	maxAllowedHops = 64
)

// TracerouteHop represents a single hop in a traceroute.
type TracerouteHop struct {
	TTL      int           `json:"ttl"`
	IP       string        `json:"ip,omitempty"`
	Hostname string        `json:"hostname,omitempty"`
	RTT      time.Duration `json:"rtt"`
	State    string        `json:"state"` // "reply", "timeout", "unreachable"
}

// TracerouteResult contains the complete traceroute result.
type TracerouteResult struct {
	Target    string          `json:"target"`
	TargetIP  string          `json:"targetIp"`
	Protocol  string          `json:"protocol"` // "icmp", "udp", "tcp"
	Port      int             `json:"port,omitempty"`
	Hops      []TracerouteHop `json:"hops"`
	Completed bool            `json:"completed"`
	Error     string          `json:"error,omitempty"`
}

// Tracer provides traceroute functionality.
type Tracer struct {
	timeout    time.Duration
	maxHops    int
	retries    int
	resolvePtr bool
	flowID     int
}

// HopCallback is called for each hop discovered during streaming traceroute.
// The callback receives the hop info and the current result state.
// Return false to stop the traceroute.
type HopCallback func(hop TracerouteHop, result *TracerouteResult) bool

// NewTracer creates a new Tracer instance.
func NewTracer(timeout time.Duration, maxHops int) *Tracer {
	if timeout == 0 {
		timeout = 1 * time.Second // Reduced from 3s for faster UI response
	}
	if maxHops <= 0 || maxHops > maxAllowedHops {
		maxHops = defaultMaxHops
	}
	return &Tracer{
		timeout:    timeout,
		maxHops:    maxHops,
		retries:    1,     // Reduced from 2 - one retry is usually enough
		resolvePtr: false, // Disabled by default - PTR lookups can be slow
		flowID:     newFlowID(),
	}
}

// newFlowID returns an ICMP echo identifier for one tracer.
//
// Two tracers running at once share the host's raw ICMP socket, so the
// identifier is what lets each recognise its own replies. Unpredictability is
// not the point, but the identifier is drawn from crypto/rand anyway: it costs
// one 16-bit read per tracer, and it keeps a security linter from having an
// opinion about a field an off-host observer can read off the wire regardless.
func newFlowID() int {
	var buf [2]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// crypto/rand does not fail on any supported platform; if it ever
		// does, a fixed identifier still traces correctly, it only loses the
		// ability to tell two concurrent tracers apart.
		return 1
	}
	return 1 + int(binary.BigEndian.Uint16(buf[:]))%math.MaxUint16
}

// NewTracerWithPTR creates a Tracer with reverse DNS lookups enabled.
func NewTracerWithPTR(timeout time.Duration, maxHops int) *Tracer {
	t := NewTracer(timeout, maxHops)
	t.resolvePtr = true
	return t
}

// WithFlowID fixes the tracer's ICMP echo identifier.
//
// Holding it constant keeps every probe of one trace on a single path through
// a load balancer that hashes on it; varying it between traces is how #395
// enumerates the alternatives. Routers differ in what they hash -- many use
// only the address pair, in which case this changes nothing -- so it is a
// lever, not a guarantee.
func (t *Tracer) WithFlowID(id int) *Tracer {
	t.flowID = id
	return t
}

// resolveIPv4 resolves a target hostname to its first IPv4 address.
func resolveIPv4(target string) (net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), traceDNSResolveTimeoutS*time.Second)
	defer cancel()
	resolver := &net.Resolver{}
	ips, err := resolver.LookupIP(ctx, "ip", target)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target: %w", err)
	}
	if len(ips) == 0 {
		return nil, errors.New("failed to resolve target: no addresses found")
	}
	for _, ip := range ips {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4, nil
		}
	}
	return nil, errors.New(errNoIPv4ForTarget)
}

// resolveHostname performs a reverse DNS lookup if PTR resolution is enabled.
func (t *Tracer) resolveHostname(ip string) string {
	if !t.resolvePtr {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), tracePTRResolveTimeoutS*time.Second)
	defer cancel()
	resolver := &net.Resolver{}
	if names, err := resolver.LookupAddr(ctx, ip); err == nil && len(names) > 0 {
		return names[0]
	}
	return ""
}

// setHopFromPeer sets hop IP and hostname from a peer address.
func (t *Tracer) setHopFromPeer(hop *TracerouteHop, peer net.Addr) {
	if peerIP, ok := peer.(*net.IPAddr); ok {
		hop.IP = peerIP.IP.String()
		hop.Hostname = t.resolveHostname(hop.IP)
	}
}

// isConnectionRefused checks if an error indicates a TCP connection was refused.
func (*Tracer) isConnectionRefused(err error) bool {
	if opErr, ok := errors.AsType[*net.OpError](err); ok {
		if sysErr, sysOK := errors.AsType[*syscall.Errno](opErr.Err); sysOK {
			return *sysErr == syscall.ECONNREFUSED
		}
	}
	return false
}

// checkContextCanceled checks if context is canceled and updates result accordingly.
func checkContextCanceled(ctx context.Context, result *TracerouteResult) bool {
	select {
	case <-ctx.Done():
		result.Error = errTracerouteCanceled
		return true
	default:
		return false
	}
}

// initTracerouteResult creates and initializes a TracerouteResult for ICMP protocol.
func (t *Tracer) initTracerouteResult(target, protocol string, port int) *TracerouteResult {
	result := &TracerouteResult{
		Target:   target,
		Protocol: protocol,
		Hops:     make([]TracerouteHop, 0, t.maxHops),
	}
	if port > 0 {
		result.Port = port
	}
	return result
}

// resolveTracerouteTarget resolves the target and updates the result.
// Returns the resolved IP or nil if resolution failed.
func resolveTracerouteTarget(target string, result *TracerouteResult) net.IP {
	targetIP, err := resolveIPv4(target)
	if err != nil {
		result.Error = err.Error()
		return nil
	}
	result.TargetIP = targetIP.String()
	return targetIP
}

// finalizeHop appends hop to result and checks if destination was reached.
func finalizeHop(hop TracerouteHop, result *TracerouteResult, targetIP string) bool {
	result.Hops = append(result.Hops, hop)
	if hop.IP == targetIP {
		result.Completed = true
		return true
	}
	return false
}

// invokeCallback safely invokes the hop callback if it's not nil.
// Returns true if traceroute should continue, false if it should stop.
func invokeCallback(onHop HopCallback, hop TracerouteHop, result *TracerouteResult) bool {
	if onHop == nil {
		return true
	}
	return onHop(hop, result)
}
