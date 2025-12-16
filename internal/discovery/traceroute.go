// Package discovery implements multi-protocol network device discovery.
// Traceroute support enables path tracing to determine the network route (hop sequence)
// that packets take to reach a target host. Supports ICMP, UDP, and TCP-based traceroute
// for mapping network topology and identifying intermediate infrastructure.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Traceroute hop state and status constants.
const (
	hopStateReply         = "reply"
	hopStateTimeout       = "timeout"
	hopStateError         = "error"
	errTracerouteCanceled = "traceroute canceled"
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

// NewTracer creates a new Tracer instance.
func NewTracer(timeout time.Duration, maxHops int) *Tracer {
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	if maxHops == 0 {
		maxHops = 30
	}
	return &Tracer{
		timeout:    timeout,
		maxHops:    maxHops,
		retries:    2,
		resolvePtr: true,
	}
}

// resolveIPv4 resolves a target hostname to its first IPv4 address.
func resolveIPv4(target string) (net.IP, error) {
	ips, err := net.LookupIP(target)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target: %w", err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("failed to resolve target: no addresses found")
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
	if names, err := net.LookupAddr(ip); err == nil && len(names) > 0 {
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
	if opErr, ok := err.(*net.OpError); ok {
		if sysErr, ok := opErr.Err.(*syscall.Errno); ok {
			return *sysErr == syscall.ECONNREFUSED
		}
	}
	return false
}

// TraceICMP performs an ICMP-based traceroute.
func (t *Tracer) TraceICMP(ctx context.Context, target string) *TracerouteResult {
	result := &TracerouteResult{
		Target:   target,
		Protocol: "icmp",
		Hops:     make([]TracerouteHop, 0, t.maxHops),
	}

	targetIP, err := resolveIPv4(target)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.TargetIP = targetIP.String()

	// Create ICMP connection for sending
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		result.Error = fmt.Sprintf("failed to create ICMP socket: %v", err)
		return result
	}
	defer conn.Close()

	// Get IPv4 packet conn for setting TTL
	pconn := conn.IPv4PacketConn()

	dst := &net.IPAddr{IP: targetIP}
	seq := 0

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		select {
		case <-ctx.Done():
			result.Error = errTracerouteCanceled
			return result
		default:
		}

		hop := TracerouteHop{
			TTL:   ttl,
			State: "timeout",
		}

		// Set TTL using ipv4 package
		if err := pconn.SetTTL(ttl); err != nil {
			hop.State = hopStateError
			result.Hops = append(result.Hops, hop)
			continue
		}

		// Try multiple times for this TTL
		for retry := range t.retries {
			_ = retry // Not used but required for intrange
			seq++
			start := time.Now()

			// Build ICMP echo request
			msg := &icmp.Message{
				Type: ipv4.ICMPTypeEcho,
				Code: 0,
				Body: &icmp.Echo{
					ID:   1,
					Seq:  seq,
					Data: []byte("SEED"),
				},
			}
			msgBytes, err := msg.Marshal(nil)
			if err != nil {
				continue
			}

			// Send packet
			if _, err := conn.WriteTo(msgBytes, dst); err != nil {
				continue
			}

			// Set read deadline
			//nolint:errcheck // Best-effort deadline setting
			conn.SetReadDeadline(time.Now().Add(t.timeout))

			// Read response
			reply := make([]byte, 1500)
			n, peer, err := conn.ReadFrom(reply)
			rtt := time.Since(start)

			if err != nil {
				// Timeout or error
				continue
			}

			// Parse the response
			rm, err := icmp.ParseMessage(1, reply[:n]) // 1 = ICMP for IPv4
			if err != nil {
				continue
			}

			hop.RTT = rtt
			t.setHopFromPeer(&hop, peer)

			switch rm.Type {
			case ipv4.ICMPTypeEchoReply:
				// Reached the destination
				hop.State = hopStateReply
				result.Hops = append(result.Hops, hop)
				result.Completed = true
				return result

			case ipv4.ICMPTypeTimeExceeded:
				// TTL exceeded - intermediate hop
				hop.State = hopStateReply

			case ipv4.ICMPTypeDestinationUnreachable:
				hop.State = "unreachable"
				result.Hops = append(result.Hops, hop)
				result.Completed = true
				return result
			}

			// Got a response, no need to retry
			break
		}

		result.Hops = append(result.Hops, hop)

		// Check if we reached the destination
		if hop.IP == targetIP.String() {
			result.Completed = true
			return result
		}
	}

	return result
}

// createUDPWithTTL creates a UDP connection with the specified TTL.
func createUDPWithTTL(targetIP net.IP, port, ttl int) (*net.UDPConn, error) {
	udpConn, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		IP:   targetIP,
		Port: port + ttl - 1,
	})
	if err != nil {
		return nil, err
	}
	rawConn, err := udpConn.SyscallConn()
	if err != nil {
		udpConn.Close()
		return nil, err
	}
	var setErr error
	//nolint:errcheck // Control callback handles its own error
	rawConn.Control(func(fd uintptr) {
		setErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
	})
	if setErr != nil {
		udpConn.Close()
		return nil, setErr
	}
	return udpConn, nil
}

// TraceUDP performs a UDP-based traceroute.
func (t *Tracer) TraceUDP(ctx context.Context, target string, port int) *TracerouteResult {
	result := &TracerouteResult{
		Target:   target,
		Protocol: "udp",
		Port:     port,
		Hops:     make([]TracerouteHop, 0, t.maxHops),
	}

	if port == 0 {
		port = 33434 // Traditional traceroute start port
	}

	targetIP, err := resolveIPv4(target)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.TargetIP = targetIP.String()

	icmpConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		result.Error = fmt.Sprintf("failed to create ICMP socket: %v", err)
		return result
	}
	defer icmpConn.Close()

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		select {
		case <-ctx.Done():
			result.Error = errTracerouteCanceled
			return result
		default:
		}

		hop := TracerouteHop{TTL: ttl, State: hopStateTimeout}

		udpConn, err := createUDPWithTTL(targetIP, port, ttl)
		if err != nil {
			hop.State = hopStateError
			result.Hops = append(result.Hops, hop)
			continue
		}

		for range t.retries {
			start := time.Now()
			if _, err = udpConn.Write([]byte("SEED")); err != nil {
				continue
			}
			//nolint:errcheck // Best-effort deadline setting
			icmpConn.SetReadDeadline(time.Now().Add(t.timeout))
			reply := make([]byte, 1500)
			n, peer, err := icmpConn.ReadFrom(reply)
			rtt := time.Since(start)
			if err != nil {
				continue
			}
			rm, err := icmp.ParseMessage(1, reply[:n])
			if err != nil {
				continue
			}
			hop.RTT = rtt
			t.setHopFromPeer(&hop, peer)

			if rm.Type == ipv4.ICMPTypeDestinationUnreachable {
				hop.State = hopStateReply
				result.Hops = append(result.Hops, hop)
				result.Completed = true
				udpConn.Close()
				return result
			}
			if rm.Type == ipv4.ICMPTypeTimeExceeded {
				hop.State = hopStateReply
			}
			break
		}

		udpConn.Close()
		result.Hops = append(result.Hops, hop)
		if hop.IP == targetIP.String() {
			result.Completed = true
			return result
		}
	}
	return result
}

// TraceTCP performs a TCP-based traceroute using SYN packets.
func (t *Tracer) TraceTCP(ctx context.Context, target string, port int) *TracerouteResult {
	result := &TracerouteResult{
		Target:   target,
		Protocol: "tcp",
		Port:     port,
		Hops:     make([]TracerouteHop, 0, t.maxHops),
	}

	if port == 0 {
		port = 80 // Default to HTTP
	}

	targetIP, err := resolveIPv4(target)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.TargetIP = targetIP.String()

	// Create ICMP listener for receiving TTL exceeded messages
	icmpConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		result.Error = fmt.Sprintf("failed to create ICMP socket: %v", err)
		return result
	}
	defer icmpConn.Close()

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		select {
		case <-ctx.Done():
			result.Error = errTracerouteCanceled
			return result
		default:
		}

		hop := TracerouteHop{
			TTL:   ttl,
			State: "timeout",
		}

		for range t.retries {
			start := time.Now()

			// Use TCP connect with timeout - simpler than raw SYN
			dialer := net.Dialer{
				Timeout: t.timeout,
				Control: func(_, _ string, c syscall.RawConn) error {
					var setErr error
					//nolint:errcheck // Control callback handles its own error
					c.Control(func(fd uintptr) {
						setErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
					})
					return setErr
				},
			}

			// Start connection attempt in goroutine
			connCh := make(chan net.Conn, 1)
			errCh := make(chan error, 1)
			go func() {
				conn, err := dialer.DialContext(ctx, "tcp4", fmt.Sprintf("%s:%d", targetIP, port))
				if err != nil {
					errCh <- err
				} else {
					connCh <- conn
				}
			}()

			// Also listen for ICMP TTL exceeded
			//nolint:errcheck // Best-effort deadline setting
			icmpConn.SetReadDeadline(time.Now().Add(t.timeout))
			icmpReply := make([]byte, 1500)

			// Wait for either TCP response, ICMP response, or timeout
			select {
			case conn := <-connCh:
				// TCP connection succeeded - reached destination
				conn.Close()
				hop.RTT = time.Since(start)
				hop.IP = targetIP.String()
				hop.Hostname = t.resolveHostname(hop.IP)
				hop.State = hopStateReply
				result.Hops = append(result.Hops, hop)
				result.Completed = true
				return result

			case tcpErr := <-errCh:
				rtt := time.Since(start)
				// Check if it's a refused connection (RST) - still means we reached target
				if t.isConnectionRefused(tcpErr) {
					hop.RTT = rtt
					hop.IP = targetIP.String()
					hop.Hostname = t.resolveHostname(hop.IP)
					hop.State = hopStateReply
					result.Hops = append(result.Hops, hop)
					result.Completed = true
					return result
				}

				// Check for ICMP response
				n, peer, err := icmpConn.ReadFrom(icmpReply)
				if err == nil {
					rm, err := icmp.ParseMessage(1, icmpReply[:n])
					if err == nil && rm.Type == ipv4.ICMPTypeTimeExceeded {
						hop.RTT = rtt
						t.setHopFromPeer(&hop, peer)
						hop.State = hopStateReply
						break
					}
				}
				continue

			case <-time.After(t.timeout):
				continue
			}
		}

		result.Hops = append(result.Hops, hop)
	}

	return result
}
