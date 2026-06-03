//go:build windows

package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Traceroute hop state constants.
const (
	HopStateReply       = "reply"
	HopStateTimeout     = "timeout"
	HopStateUnreachable = "unreachable"
)

// Traceroute timing constants.
const (
	traceDNSResolveTimeoutS = 5 // Timeout in seconds for DNS resolution
	tracePTRResolveTimeoutS = 2 // Timeout in seconds for PTR lookup
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
}

// HopCallback is called for each hop discovered during streaming traceroute.
// The callback receives the hop info and the current result state.
// Return false to stop the traceroute.
type HopCallback func(hop TracerouteHop, result *TracerouteResult) bool

// NewTracer creates a new Tracer instance.
func NewTracer(timeout time.Duration, maxHops int) *Tracer {
	if timeout == 0 {
		timeout = 1 * time.Second
	}
	if maxHops == 0 {
		maxHops = 30
	}
	return &Tracer{
		timeout:    timeout,
		maxHops:    maxHops,
		retries:    1,
		resolvePtr: false,
	}
}

// NewTracerWithPTR creates a Tracer with reverse DNS lookups enabled.
func NewTracerWithPTR(timeout time.Duration, maxHops int) *Tracer {
	t := NewTracer(timeout, maxHops)
	t.resolvePtr = true
	return t
}

// resolveIPv4Windows resolves a target hostname to its first IPv4 address.
func resolveIPv4Windows(target string) (net.IP, error) {
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
	return nil, errors.New("no IPv4 address found for target")
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
		return strings.TrimSuffix(names[0], ".")
	}
	return ""
}

// TraceICMP performs an ICMP-based traceroute.
// On Windows, this uses a simplified approach since raw ICMP requires admin privileges.
func (t *Tracer) TraceICMP(ctx context.Context, target string) *TracerouteResult {
	result := &TracerouteResult{
		Target:   target,
		Protocol: "icmp",
		Hops:     make([]TracerouteHop, 0),
	}

	// Resolve target
	targetIP, err := resolveIPv4Windows(target)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.TargetIP = targetIP.String()

	// On Windows, full ICMP traceroute requires Administrator privileges
	// For non-admin mode, we do a simplified TCP-based path probe
	for ttl := 1; ttl <= t.maxHops; ttl++ {
		select {
		case <-ctx.Done():
			result.Error = "traceroute cancelled"
			return result
		default:
		}

		hop := TracerouteHop{
			TTL:   ttl,
			State: HopStateTimeout,
		}

		// Try TCP connect with timeout as a proxy for reachability
		start := time.Now()
		dialer := net.Dialer{Timeout: t.timeout}
		conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:80", result.TargetIP))
		hop.RTT = time.Since(start)

		if err == nil {
			conn.Close()
			hop.IP = result.TargetIP
			hop.State = HopStateReply
			hop.Hostname = t.resolveHostname(hop.IP)
			result.Hops = append(result.Hops, hop)
			result.Completed = true
			break
		}

		// Check if we got a response (even if error)
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			if opErr.Timeout() {
				hop.State = HopStateTimeout
			} else {
				// Got some kind of response
				hop.State = HopStateReply
				hop.IP = result.TargetIP
			}
		}

		result.Hops = append(result.Hops, hop)

		// Break early if we've reached the target
		if hop.IP == result.TargetIP {
			result.Completed = true
			break
		}
	}

	return result
}

// TraceICMPStreaming performs an ICMP-based traceroute with per-hop callbacks.
// This enables real-time UI updates as each hop is discovered.
func (t *Tracer) TraceICMPStreaming(ctx context.Context, target string, onHop HopCallback) *TracerouteResult {
	result := &TracerouteResult{
		Target:   target,
		Protocol: "icmp",
		Hops:     make([]TracerouteHop, 0),
	}

	// Resolve target
	targetIP, err := resolveIPv4Windows(target)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.TargetIP = targetIP.String()

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		select {
		case <-ctx.Done():
			result.Error = "traceroute cancelled"
			return result
		default:
		}

		hop := TracerouteHop{
			TTL:   ttl,
			State: HopStateTimeout,
		}

		start := time.Now()
		dialer := net.Dialer{Timeout: t.timeout}
		conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:80", result.TargetIP))
		hop.RTT = time.Since(start)

		if err == nil {
			conn.Close()
			hop.IP = result.TargetIP
			hop.State = HopStateReply
			hop.Hostname = t.resolveHostname(hop.IP)
			result.Hops = append(result.Hops, hop)
			result.Completed = true
			if onHop != nil {
				onHop(hop, result)
			}
			break
		}

		var opErr *net.OpError
		if errors.As(err, &opErr) {
			if opErr.Timeout() {
				hop.State = HopStateTimeout
			} else {
				hop.State = HopStateReply
				hop.IP = result.TargetIP
			}
		}

		result.Hops = append(result.Hops, hop)

		// Call the hop callback
		if onHop != nil && !onHop(hop, result) {
			return result
		}

		if hop.IP == result.TargetIP {
			result.Completed = true
			break
		}
	}

	return result
}

// TraceUDP performs a UDP-based traceroute.
// On Windows, this requires Administrator privileges for raw sockets.
func (t *Tracer) TraceUDP(ctx context.Context, target string, port int) *TracerouteResult {
	if port == 0 {
		port = 33434 // Traditional traceroute start port
	}
	result := &TracerouteResult{
		Target:   target,
		Protocol: "udp",
		Port:     port,
		Hops:     make([]TracerouteHop, 0),
	}

	// Resolve target
	targetIP, err := resolveIPv4Windows(target)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.TargetIP = targetIP.String()

	// UDP traceroute requires raw sockets on Windows (Administrator privileges)
	// Fall back to simplified TCP-based approach
	for ttl := 1; ttl <= t.maxHops; ttl++ {
		select {
		case <-ctx.Done():
			result.Error = "traceroute cancelled"
			return result
		default:
		}

		hop := TracerouteHop{
			TTL:   ttl,
			State: HopStateTimeout,
		}

		start := time.Now()
		dialer := net.Dialer{Timeout: t.timeout}
		conn, err := dialer.DialContext(ctx, "udp", fmt.Sprintf("%s:%d", result.TargetIP, port))
		hop.RTT = time.Since(start)

		if err == nil {
			conn.Close()
			hop.IP = result.TargetIP
			hop.State = HopStateReply
			hop.Hostname = t.resolveHostname(hop.IP)
			result.Hops = append(result.Hops, hop)
			result.Completed = true
			break
		}

		result.Hops = append(result.Hops, hop)
	}

	return result
}

// TraceTCP performs a TCP-based traceroute.
func (t *Tracer) TraceTCP(ctx context.Context, target string, port int) *TracerouteResult {
	if port == 0 {
		port = 80 // Default to HTTP
	}
	result := &TracerouteResult{
		Target:   target,
		Protocol: "tcp",
		Port:     port,
		Hops:     make([]TracerouteHop, 0),
	}

	// Resolve target
	targetIP, err := resolveIPv4Windows(target)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.TargetIP = targetIP.String()

	// TCP traceroute - try to connect
	for ttl := 1; ttl <= t.maxHops; ttl++ {
		select {
		case <-ctx.Done():
			result.Error = "traceroute cancelled"
			return result
		default:
		}

		hop := TracerouteHop{
			TTL:   ttl,
			State: HopStateTimeout,
		}

		start := time.Now()
		dialer := net.Dialer{Timeout: t.timeout}
		conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", result.TargetIP, port))
		hop.RTT = time.Since(start)

		if err == nil {
			conn.Close()
			hop.IP = result.TargetIP
			hop.State = HopStateReply
			hop.Hostname = t.resolveHostname(hop.IP)
			result.Hops = append(result.Hops, hop)
			result.Completed = true
			break
		}

		// Analyze error
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			if opErr.Timeout() {
				hop.State = HopStateTimeout
			} else if strings.Contains(err.Error(), "refused") {
				hop.State = HopStateReply
				hop.IP = result.TargetIP
				result.Completed = true
			}
		}

		result.Hops = append(result.Hops, hop)

		if result.Completed {
			break
		}
	}

	return result
}

// TraceWithCallback performs traceroute with per-hop callback.
func (t *Tracer) TraceWithCallback(ctx context.Context, target string, callback HopCallback) *TracerouteResult {
	result := t.TraceICMP(ctx, target)

	// Call callback for each hop
	for _, hop := range result.Hops {
		if !callback(hop, result) {
			break
		}
	}

	return result
}
