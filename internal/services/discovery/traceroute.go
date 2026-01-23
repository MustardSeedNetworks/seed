//go:build !windows

package discovery

// Traceroute support enables path tracing to determine the network route (hop sequence)
// that packets take to reach a target host. Supports ICMP, UDP, and TCP-based traceroute
// for mapping network topology and identifying intermediate infrastructure.

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
	hopStateUnreachable   = "unreachable"
	errTracerouteCanceled = "traceroute canceled"
)

// Traceroute timing and buffer constants.
const (
	traceDNSResolveTimeoutS = 5    // Timeout in seconds for DNS resolution
	tracePTRResolveTimeoutS = 2    // Timeout in seconds for PTR lookup
	traceICMPBufferSize     = 1500 // Buffer size for ICMP reply packets
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

// icmpProbeResult represents the outcome of a single ICMP probe attempt.
type icmpProbeResult struct {
	success     bool
	rtt         time.Duration
	peer        net.Addr
	messageType ipv4.ICMPType
}

// hopOutcome represents the final outcome after processing an ICMP response.
type hopOutcome int

const (
	hopOutcomeContinue hopOutcome = iota // Continue to next TTL
	hopOutcomeComplete                   // Traceroute completed (destination reached or unreachable)
)

// NewTracer creates a new Tracer instance.
func NewTracer(timeout time.Duration, maxHops int) *Tracer {
	if timeout == 0 {
		timeout = 1 * time.Second // Reduced from 3s for faster UI response
	}
	if maxHops == 0 {
		maxHops = 30
	}
	return &Tracer{
		timeout:    timeout,
		maxHops:    maxHops,
		retries:    1,     // Reduced from 2 - one retry is usually enough
		resolvePtr: false, // Disabled by default - PTR lookups can be slow
	}
}

// NewTracerWithPTR creates a Tracer with reverse DNS lookups enabled.
func NewTracerWithPTR(timeout time.Duration, maxHops int) *Tracer {
	t := NewTracer(timeout, maxHops)
	t.resolvePtr = true
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
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr *syscall.Errno
		if errors.As(opErr.Err, &sysErr) {
			return *sysErr == syscall.ECONNREFUSED
		}
	}
	return false
}

// buildICMPEchoRequest creates an ICMP echo request message.
func buildICMPEchoRequest(seq int) ([]byte, error) {
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   1,
			Seq:  seq,
			Data: []byte("SEED"),
		},
	}
	return msg.Marshal(nil)
}

// sendICMPProbe sends an ICMP probe and waits for a response.
func (t *Tracer) sendICMPProbe(
	conn *icmp.PacketConn,
	dst *net.IPAddr,
	msgBytes []byte,
) icmpProbeResult {
	start := time.Now()

	if _, err := conn.WriteTo(msgBytes, dst); err != nil {
		return icmpProbeResult{}
	}

	if err := conn.SetReadDeadline(time.Now().Add(t.timeout)); err != nil {
		return icmpProbeResult{}
	}

	reply := make([]byte, traceICMPBufferSize)
	n, peer, err := conn.ReadFrom(reply)
	rtt := time.Since(start)

	if err != nil {
		return icmpProbeResult{}
	}

	rm, err := icmp.ParseMessage(1, reply[:n])
	if err != nil {
		return icmpProbeResult{}
	}

	msgType, ok := rm.Type.(ipv4.ICMPType)
	if !ok {
		return icmpProbeResult{}
	}

	return icmpProbeResult{
		success:     true,
		rtt:         rtt,
		peer:        peer,
		messageType: msgType,
	}
}

// processICMPResponse updates hop state based on ICMP response type.
// Returns the outcome indicating whether to continue or complete the traceroute.
func (t *Tracer) processICMPResponse(
	hop *TracerouteHop,
	probe icmpProbeResult,
	result *TracerouteResult,
) hopOutcome {
	hop.RTT = probe.rtt
	t.setHopFromPeer(hop, probe.peer)

	// Using if-else instead of switch to avoid exhaustive enum check on external type
	if probe.messageType == ipv4.ICMPTypeEchoReply {
		hop.State = hopStateReply
		result.Hops = append(result.Hops, *hop)
		result.Completed = true
		return hopOutcomeComplete
	}

	if probe.messageType == ipv4.ICMPTypeDestinationUnreachable {
		hop.State = hopStateUnreachable
		result.Hops = append(result.Hops, *hop)
		result.Completed = true
		return hopOutcomeComplete
	}

	if probe.messageType == ipv4.ICMPTypeTimeExceeded {
		hop.State = hopStateReply
	}

	return hopOutcomeContinue
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

// createICMPConnection creates an ICMP connection for traceroute.
func createICMPConnection(result *TracerouteResult) *icmp.PacketConn {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		result.Error = fmt.Sprintf("failed to create ICMP socket: %v", err)
		return nil
	}
	return conn
}

// probeWithRetries attempts to probe a hop with retries.
// Returns true if a valid response was received.
func (t *Tracer) probeWithRetries(
	conn *icmp.PacketConn,
	dst *net.IPAddr,
	seq *int,
) icmpProbeResult {
	for range t.retries {
		*seq++
		msgBytes, err := buildICMPEchoRequest(*seq)
		if err != nil {
			continue
		}
		probe := t.sendICMPProbe(conn, dst, msgBytes)
		if probe.success {
			return probe
		}
	}
	return icmpProbeResult{}
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

// processStreamingICMPResponse processes an ICMP response for streaming traceroute.
// Returns (completed, shouldStop) - completed indicates destination reached,
// shouldStop indicates whether the callback requested to stop.
func (t *Tracer) processStreamingICMPResponse(
	hop *TracerouteHop,
	probe icmpProbeResult,
	result *TracerouteResult,
	onHop HopCallback,
) (bool, bool) {
	hop.RTT = probe.rtt
	t.setHopFromPeer(hop, probe.peer)

	// Using if-else instead of switch to avoid exhaustive enum check on external type
	if probe.messageType == ipv4.ICMPTypeEchoReply {
		hop.State = hopStateReply
		result.Hops = append(result.Hops, *hop)
		result.Completed = true
		invokeCallback(onHop, *hop, result)
		return true, true
	}

	if probe.messageType == ipv4.ICMPTypeDestinationUnreachable {
		hop.State = hopStateUnreachable
		result.Hops = append(result.Hops, *hop)
		result.Completed = true
		invokeCallback(onHop, *hop, result)
		return true, true
	}

	if probe.messageType == ipv4.ICMPTypeTimeExceeded {
		hop.State = hopStateReply
	}

	return false, false
}

// finalizeStreamingHop appends hop, invokes callback, and checks destination.
// Returns true if traceroute should stop (destination reached or callback requested stop).
func finalizeStreamingHop(
	hop TracerouteHop,
	result *TracerouteResult,
	targetIP string,
	onHop HopCallback,
) bool {
	result.Hops = append(result.Hops, hop)
	if !invokeCallback(onHop, hop, result) {
		return true
	}
	if hop.IP == targetIP {
		result.Completed = true
		return true
	}
	return false
}

// TraceICMP performs an ICMP-based traceroute.
func (t *Tracer) TraceICMP(ctx context.Context, target string) *TracerouteResult {
	result := t.initTracerouteResult(target, "icmp", 0)

	targetIP := resolveTracerouteTarget(target, result)
	if targetIP == nil {
		return result
	}

	conn := createICMPConnection(result)
	if conn == nil {
		return result
	}
	defer func() { _ = conn.Close() }()

	pconn := conn.IPv4PacketConn()
	dst := &net.IPAddr{IP: targetIP}
	seq := 0

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		if checkContextCanceled(ctx, result) {
			return result
		}

		hop := TracerouteHop{TTL: ttl, State: hopStateTimeout}

		if err := pconn.SetTTL(ttl); err != nil {
			hop.State = hopStateError
			result.Hops = append(result.Hops, hop)
			continue
		}

		probe := t.probeWithRetries(conn, dst, &seq)
		if probe.success {
			outcome := t.processICMPResponse(&hop, probe, result)
			if outcome == hopOutcomeComplete {
				return result
			}
		}

		if finalizeHop(hop, result, targetIP.String()) {
			return result
		}
	}

	return result
}

// TraceICMPStreaming performs an ICMP-based traceroute with per-hop callbacks.
// This enables real-time UI updates as each hop is discovered.
func (t *Tracer) TraceICMPStreaming(
	ctx context.Context,
	target string,
	onHop HopCallback,
) *TracerouteResult {
	result := t.initTracerouteResult(target, "icmp", 0)

	targetIP := resolveTracerouteTarget(target, result)
	if targetIP == nil {
		return result
	}

	conn := createICMPConnection(result)
	if conn == nil {
		return result
	}
	defer func() { _ = conn.Close() }()

	pconn := conn.IPv4PacketConn()
	dst := &net.IPAddr{IP: targetIP}
	seq := 0

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		if checkContextCanceled(ctx, result) {
			return result
		}

		hop := TracerouteHop{TTL: ttl, State: hopStateTimeout}

		if err := pconn.SetTTL(ttl); err != nil {
			hop.State = hopStateError
			result.Hops = append(result.Hops, hop)
			if !invokeCallback(onHop, hop, result) {
				return result
			}
			continue
		}

		probe := t.probeWithRetries(conn, dst, &seq)
		if probe.success {
			_, shouldStop := t.processStreamingICMPResponse(&hop, probe, result, onHop)
			if shouldStop {
				return result
			}
		}

		if finalizeStreamingHop(hop, result, targetIP.String(), onHop) {
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
		_ = udpConn.Close()
		return nil, err
	}
	var setErr error
	if ctrlErr := rawConn.Control(func(fd uintptr) {
		setErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
	}); ctrlErr != nil {
		_ = udpConn.Close()
		return nil, ctrlErr
	}
	if setErr != nil {
		_ = udpConn.Close()
		return nil, setErr
	}
	return udpConn, nil
}

// udpProbeResult represents the outcome of a UDP probe attempt.
type udpProbeResult struct {
	success     bool
	rtt         time.Duration
	peer        net.Addr
	messageType ipv4.ICMPType
}

// sendUDPProbe sends a UDP probe and waits for an ICMP response.
func (t *Tracer) sendUDPProbe(
	udpConn *net.UDPConn,
	icmpConn *icmp.PacketConn,
) udpProbeResult {
	start := time.Now()

	if _, err := udpConn.Write([]byte("SEED")); err != nil {
		return udpProbeResult{}
	}

	if err := icmpConn.SetReadDeadline(time.Now().Add(t.timeout)); err != nil {
		return udpProbeResult{}
	}

	reply := make([]byte, traceICMPBufferSize)
	n, peer, err := icmpConn.ReadFrom(reply)
	rtt := time.Since(start)

	if err != nil {
		return udpProbeResult{}
	}

	rm, err := icmp.ParseMessage(1, reply[:n])
	if err != nil {
		return udpProbeResult{}
	}

	msgType, ok := rm.Type.(ipv4.ICMPType)
	if !ok {
		return udpProbeResult{}
	}

	return udpProbeResult{
		success:     true,
		rtt:         rtt,
		peer:        peer,
		messageType: msgType,
	}
}

// processUDPResponse updates hop state based on the UDP probe response.
// Returns true if traceroute is complete (destination reached).
func (t *Tracer) processUDPResponse(
	hop *TracerouteHop,
	probe udpProbeResult,
	result *TracerouteResult,
) bool {
	hop.RTT = probe.rtt
	t.setHopFromPeer(hop, probe.peer)

	if probe.messageType == ipv4.ICMPTypeDestinationUnreachable {
		hop.State = hopStateReply
		result.Hops = append(result.Hops, *hop)
		result.Completed = true
		return true
	}

	if probe.messageType == ipv4.ICMPTypeTimeExceeded {
		hop.State = hopStateReply
	}

	return false
}

// probeUDPWithRetries attempts UDP probes with retries.
func (t *Tracer) probeUDPWithRetries(
	udpConn *net.UDPConn,
	icmpConn *icmp.PacketConn,
) udpProbeResult {
	for range t.retries {
		probe := t.sendUDPProbe(udpConn, icmpConn)
		if probe.success {
			return probe
		}
	}
	return udpProbeResult{}
}

// TraceUDP performs a UDP-based traceroute.
func (t *Tracer) TraceUDP(ctx context.Context, target string, port int) *TracerouteResult {
	if port == 0 {
		port = 33434 // Traditional traceroute start port
	}
	result := t.initTracerouteResult(target, "udp", port)

	targetIP := resolveTracerouteTarget(target, result)
	if targetIP == nil {
		return result
	}

	icmpConn := createICMPConnection(result)
	if icmpConn == nil {
		return result
	}
	defer func() { _ = icmpConn.Close() }()

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		if checkContextCanceled(ctx, result) {
			return result
		}

		hop := TracerouteHop{TTL: ttl, State: hopStateTimeout}

		udpConn, err := createUDPWithTTL(targetIP, port, ttl)
		if err != nil {
			hop.State = hopStateError
			result.Hops = append(result.Hops, hop)
			continue
		}

		probe := t.probeUDPWithRetries(udpConn, icmpConn)
		_ = udpConn.Close()

		if probe.success {
			if t.processUDPResponse(&hop, probe, result) {
				return result
			}
		}

		if finalizeHop(hop, result, targetIP.String()) {
			return result
		}
	}

	return result
}

// tcpProbeOutcome represents the result of a TCP probe attempt.
type tcpProbeOutcome int

const (
	tcpProbeTimeout   tcpProbeOutcome = iota // No response received
	tcpProbeConnected                        // TCP connection succeeded (destination reached)
	tcpProbeRefused                          // Connection refused (destination reached)
	tcpProbeICMP                             // ICMP TTL exceeded received (intermediate hop)
)

// tcpProbeResult holds the result of a TCP probe.
type tcpProbeResult struct {
	outcome tcpProbeOutcome
	rtt     time.Duration
	peer    net.Addr // For ICMP responses
}

// createTCPDialer creates a [net.Dialer] configured with the specified TTL.
func (t *Tracer) createTCPDialer(ttl int) net.Dialer {
	return net.Dialer{
		Timeout: t.timeout,
		Control: func(_, _ string, c syscall.RawConn) error {
			var setErr error
			if ctrlErr := c.Control(func(fd uintptr) {
				setErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
			}); ctrlErr != nil {
				return ctrlErr
			}
			return setErr
		},
	}
}

// tcpDialChannels holds channels for async TCP dial results.
type tcpDialChannels struct {
	connCh chan net.Conn
	errCh  chan error
}

// startTCPDial starts an asynchronous TCP connection attempt.
func startTCPDial(
	ctx context.Context,
	dialer net.Dialer,
	targetIP net.IP,
	port int,
) tcpDialChannels {
	ch := tcpDialChannels{
		connCh: make(chan net.Conn, 1),
		errCh:  make(chan error, 1),
	}

	go func() {
		conn, err := dialer.DialContext(ctx, "tcp4", fmt.Sprintf("%s:%d", targetIP, port))
		if err != nil {
			ch.errCh <- err
		} else {
			ch.connCh <- conn
		}
	}()

	return ch
}

// waitForTCPProbeResult waits for TCP or ICMP response and returns the probe result.
func (t *Tracer) waitForTCPProbeResult(
	ch tcpDialChannels,
	icmpConn *icmp.PacketConn,
	start time.Time,
) tcpProbeResult {
	icmpReply := make([]byte, traceICMPBufferSize)

	select {
	case conn := <-ch.connCh:
		_ = conn.Close()
		return tcpProbeResult{outcome: tcpProbeConnected, rtt: time.Since(start)}

	case tcpErr := <-ch.errCh:
		rtt := time.Since(start)
		if t.isConnectionRefused(tcpErr) {
			return tcpProbeResult{outcome: tcpProbeRefused, rtt: rtt}
		}

		n, peer, err := icmpConn.ReadFrom(icmpReply)
		if err == nil {
			rm, parseErr := icmp.ParseMessage(1, icmpReply[:n])
			if parseErr == nil && rm.Type == ipv4.ICMPTypeTimeExceeded {
				return tcpProbeResult{outcome: tcpProbeICMP, rtt: rtt, peer: peer}
			}
		}
		return tcpProbeResult{outcome: tcpProbeTimeout}

	case <-time.After(t.timeout):
		return tcpProbeResult{outcome: tcpProbeTimeout}
	}
}

// processTCPProbeResult updates hop based on probe result.
// Returns true if destination was reached and traceroute should complete.
func (t *Tracer) processTCPProbeResult(
	hop *TracerouteHop,
	probe tcpProbeResult,
	result *TracerouteResult,
	targetIP string,
) bool {
	hop.RTT = probe.rtt

	switch probe.outcome {
	case tcpProbeConnected, tcpProbeRefused:
		hop.IP = targetIP
		hop.Hostname = t.resolveHostname(hop.IP)
		hop.State = hopStateReply
		result.Hops = append(result.Hops, *hop)
		result.Completed = true
		return true

	case tcpProbeICMP:
		t.setHopFromPeer(hop, probe.peer)
		hop.State = hopStateReply
		return false

	case tcpProbeTimeout:
		return false
	}

	return false
}

// probeTCPWithRetries performs TCP probing with retries for a single TTL.
// Returns (gotResponse, destinationReached).
func (t *Tracer) probeTCPWithRetries(
	ctx context.Context,
	hop *TracerouteHop,
	result *TracerouteResult,
	targetIP net.IP,
	port int,
	ttl int,
	icmpConn *icmp.PacketConn,
) bool {
	dialer := t.createTCPDialer(ttl)

	for range t.retries {
		if err := icmpConn.SetReadDeadline(time.Now().Add(t.timeout)); err != nil {
			continue
		}

		start := time.Now()
		ch := startTCPDial(ctx, dialer, targetIP, port)
		probe := t.waitForTCPProbeResult(ch, icmpConn, start)

		if probe.outcome == tcpProbeTimeout {
			continue
		}

		if t.processTCPProbeResult(hop, probe, result, targetIP.String()) {
			return true
		}

		// Got ICMP response, stop retrying
		break
	}

	return false
}

// TraceTCP performs a TCP-based traceroute using SYN packets.
func (t *Tracer) TraceTCP(ctx context.Context, target string, port int) *TracerouteResult {
	if port == 0 {
		port = 80 // Default to HTTP
	}
	result := t.initTracerouteResult(target, "tcp", port)

	targetIP := resolveTracerouteTarget(target, result)
	if targetIP == nil {
		return result
	}

	icmpConn := createICMPConnection(result)
	if icmpConn == nil {
		return result
	}
	defer func() { _ = icmpConn.Close() }()

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		if checkContextCanceled(ctx, result) {
			return result
		}

		hop := TracerouteHop{TTL: ttl, State: hopStateTimeout}

		if t.probeTCPWithRetries(ctx, &hop, result, targetIP, port, ttl, icmpConn) {
			return result
		}

		result.Hops = append(result.Hops, hop)
	}

	return result
}
