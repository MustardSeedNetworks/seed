//go:build !windows

package discovery

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

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
