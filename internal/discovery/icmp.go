//go:build !windows

package discovery

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// protocolICMP is the IANA protocol number for ICMP, which icmp.ParseMessage
// needs to pick a decoder.
const protocolICMP = 1

// icmpProbeResult represents the outcome of a single ICMP probe attempt.
type icmpProbeResult struct {
	success     bool
	rtt         time.Duration
	peer        net.Addr
	messageType ipv4.ICMPType
}

// buildICMPEchoRequest creates an ICMP echo request carrying the tracer's flow
// identifier. ID and Seq together are what a reply must echo back for us to
// claim it, and ID is also the only field a load-balancing router can hash on
// that we control -- see Tracer.flowID.
func buildICMPEchoRequest(flowID, seq int) ([]byte, error) {
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   flowID,
			Seq:  seq,
			Data: []byte("SEED"),
		},
	}
	return msg.Marshal(nil)
}

// echoIDSeq extracts the echo identifier and sequence a reply refers to, and
// reports whether the message carries one at all.
//
// An echo reply carries them directly. A time-exceeded or destination-
// unreachable carries the *original* datagram we sent -- an IP header plus at
// least its first 8 bytes, which for ICMP is the echo header -- so the probe it
// answers is recoverable from the quotation.
func echoIDSeq(msg *icmp.Message) (int, int, bool) {
	switch body := msg.Body.(type) {
	case *icmp.Echo:
		return body.ID, body.Seq, true
	case *icmp.TimeExceeded:
		return quotedEchoIDSeq(body.Data)
	case *icmp.DstUnreach:
		return quotedEchoIDSeq(body.Data)
	default:
		return 0, 0, false
	}
}

// quotedEchoIDSeq reads the echo ID and sequence out of the original datagram
// quoted back inside an ICMP error.
func quotedEchoIDSeq(quoted []byte) (int, int, bool) {
	header, err := icmp.ParseIPv4Header(quoted)
	if err != nil {
		return 0, 0, false
	}
	inner := quoted[header.Len:]
	// Type, code, checksum, ID, sequence -- the 8 bytes an ICMP error is
	// obliged to quote. Anything shorter cannot identify a probe.
	const icmpEchoHeaderLen = 8
	if len(inner) < icmpEchoHeaderLen {
		return 0, 0, false
	}
	return int(binary.BigEndian.Uint16(inner[4:6])), int(binary.BigEndian.Uint16(inner[6:8])), true
}

// sendICMPProbe sends one ICMP probe and waits for the reply to *that* probe.
//
// A raw ICMP listener receives every ICMP packet the host sees -- including
// replies to a ping running in another process. Reading one packet and calling
// it the answer misattributes those: a foreign echo reply would end a
// traceroute early, and a foreign error would name the wrong hop. So
// non-matching packets are discarded and the read is retried until the deadline
// the timeout sets.
func (t *Tracer) sendICMPProbe(
	conn *icmp.PacketConn,
	dst net.Addr,
	msgBytes []byte,
	seq int,
	privileged bool,
) icmpProbeResult {
	start := time.Now()

	if _, err := conn.WriteTo(msgBytes, dst); err != nil {
		return icmpProbeResult{}
	}

	deadline := start.Add(t.timeout)
	reply := make([]byte, traceICMPBufferSize)

	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			return icmpProbeResult{}
		}

		n, peer, err := conn.ReadFrom(reply)
		if err != nil {
			return icmpProbeResult{}
		}
		rtt := time.Since(start)

		rm, err := icmp.ParseMessage(protocolICMP, reply[:n])
		if err != nil {
			continue
		}

		msgType, ok := rm.Type.(ipv4.ICMPType)
		if !ok {
			continue
		}

		// A raw socket on loopback is handed the echo *request* we just sent as
		// well as the reply to it, and the two carry the same ID and sequence.
		// Matching on those alone accepted our own probe as its own answer, so
		// the hop kept its initial "timeout" state while still naming the peer.
		if msgType == ipv4.ICMPTypeEcho {
			continue
		}

		id, replySeq, ok := echoIDSeq(rm)
		if !ok || replySeq != seq {
			continue
		}
		// Only a raw socket carries our identifier end to end. Linux rewrites
		// the echo ID of a datagram ICMP socket to the socket's port, so the ID
		// that comes back is the kernel's, not t.flowID -- the sweep drops the
		// same check for the same reason (enumerate/icmp.go). The kernel also
		// demultiplexes that socket per process, which is the isolation the ID
		// check provides on a raw one; the sequence still pins the probe.
		if privileged && id != t.flowID {
			continue
		}

		return icmpProbeResult{
			success:     true,
			rtt:         rtt,
			peer:        peer,
			messageType: msgType,
		}
	}
}

// createICMPConnection opens the socket a traceroute probes with, preferring a
// raw ICMP listener and falling back to the unprivileged datagram socket where
// raw is refused (seed#2689). It reports which one it got: the destination
// address type and the reply match both differ between the two modes.
//
// A raw socket needs root or CAP_NET_RAW. Without the fallback an unprivileged
// daemon -- every macOS install -- traced nothing at all and Path Analysis
// showed an empty result, which is the whole of what the operator saw.
func createICMPConnection(result *TracerouteResult) (*icmp.PacketConn, bool) {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err == nil {
		return conn, true
	}

	rawErr := err
	conn, err = icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		result.Error = fmt.Sprintf("failed to create ICMP socket (raw: %v; datagram: %v)", rawErr, err)
		return nil, false
	}
	return conn, false
}

// createRawICMPConnection opens the raw listener the UDP and TCP tracers read
// their errors from. They have no datagram fallback: a datagram ICMP socket
// only receives the replies and errors belonging to the echo requests it sent
// itself, so it would deliver nothing for a UDP or TCP probe and the trace
// would come back silently empty rather than saying why (seed#2689).
func createRawICMPConnection(result *TracerouteResult) *icmp.PacketConn {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		result.Error = fmt.Sprintf(
			"%s traceroute needs a raw ICMP socket, which this process may not open (%v); "+
				"use the ICMP protocol, which works unprivileged", result.Protocol, err)
		return nil
	}
	return conn
}

// probeWithRetries attempts to probe a hop with retries.
// Returns true if a valid response was received.
func (t *Tracer) probeWithRetries(
	conn *icmp.PacketConn,
	dst net.Addr,
	seq *int,
	privileged bool,
) icmpProbeResult {
	for range t.retries {
		*seq++
		msgBytes, err := buildICMPEchoRequest(t.flowID, *seq)
		if err != nil {
			continue
		}
		probe := t.sendICMPProbe(conn, dst, msgBytes, *seq, privileged)
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
//
// It is TraceICMPStreaming without a listener: the two loops were duplicates
// that had already drifted apart, and invokeCallback treats a nil callback as
// "keep going".
func (t *Tracer) TraceICMP(ctx context.Context, target string) *TracerouteResult {
	return t.TraceICMPStreaming(ctx, target, nil)
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

	conn, privileged := createICMPConnection(result)
	if conn == nil {
		return result
	}
	defer func() { _ = conn.Close() }()

	pconn := conn.IPv4PacketConn()
	dst := icmpDestination(targetIP, privileged)
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

		probe := t.probeWithRetries(conn, dst, &seq, privileged)
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
