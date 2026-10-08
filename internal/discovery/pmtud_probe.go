//go:build !windows

package discovery

// The send-and-classify half of a sized probe.
//
// The socket is opened with [net.ListenConfig.ListenPacket] rather than
// icmp.ListenPacket because setting the don't-fragment bit needs the file
// descriptor, and
// [icmp.PacketConn] keeps its inner connection unexported. x/net/icmp is still
// used to build and parse the messages -- only the socket is ours.

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// codeFragmentationNeeded is the destination-unreachable code a router sends
// when a packet exceeds the next hop's MTU and may not be fragmented.
const codeFragmentationNeeded = 4

// errNotARawIPSocket guards the type assertion below. ListenPacket on
// "ip4:icmp" always yields a [net.IPConn], so this is unreachable in practice
// and exists so a future change to that call fails loudly rather than silently
// probing without the don't-fragment bit -- which would measure nothing and
// look like a path with no bottleneck.
var errNotARawIPSocket = errors.New("icmp socket is not a raw IP socket")

// discardPort is the port the egress lookup dials. Dialling UDP only asks the
// kernel for a route; no datagram is sent, so the port is never reached.
const discardPort = 9

// errNoEgressInterface is returned when the source address the kernel picked
// for a destination belongs to no interface, which leaves no MTU to start from.
var errNoEgressInterface = errors.New("no interface holds the source address for the destination")

// MeasurePathMTU resolves a destination, starts the search from the MTU of
// the interface the kernel would send through, and runs it over a
// don't-fragment ICMP probe.
//
// An error means the search could not begin -- the name did not resolve, the
// route has no interface, the raw socket was refused. Everything the search
// itself learned, including a black hole or a cancellation, is in the result.
func MeasurePathMTU(ctx context.Context, target string, timeout time.Duration) (*PMTUDResult, error) {
	dst, err := resolveIPv4(target)
	if err != nil {
		return nil, err
	}
	localMTU, err := egressMTU(ctx, dst)
	if err != nil {
		return nil, err
	}

	probe, closeProbe, err := newICMPProbe(ctx, dst, newFlowID(), timeout)
	if err != nil {
		return nil, err
	}
	defer func() { _ = closeProbe() }()

	result := DiscoverPathMTU(ctx, target, localMTU, probe)
	result.TargetIP = dst.String()
	return result, nil
}

// egressMTU returns the MTU of the interface the kernel routes a destination
// through, found by the source address it picks for a connected UDP socket.
func egressMTU(ctx context.Context, dst net.IP) (int, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "udp4", net.JoinHostPort(dst.String(), strconv.Itoa(discardPort)))
	if err != nil {
		return 0, err
	}
	defer func() { _ = conn.Close() }()

	local, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return 0, errNoEgressInterface
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return 0, err
	}
	for _, iface := range ifaces {
		addrs, addrErr := iface.Addrs()
		if addrErr != nil {
			continue
		}
		for _, addr := range addrs {
			if ipNet, isNet := addr.(*net.IPNet); isNet && ipNet.IP.Equal(local.IP) {
				return iface.MTU, nil
			}
		}
	}
	return 0, errNoEgressInterface
}

// newICMPProbe returns a ProbeFunc that sends don't-fragment echo requests to a
// destination, for DiscoverPathMTU to drive, along with the closer for its
// socket.
func newICMPProbe(
	ctx context.Context,
	dst net.IP,
	flowID int,
	timeout time.Duration,
) (ProbeFunc, func() error, error) {
	var config net.ListenConfig
	packetConn, err := config.ListenPacket(ctx, "ip4:icmp", "0.0.0.0")
	if err != nil {
		return nil, nil, err
	}

	conn, ok := packetConn.(*net.IPConn)
	if !ok {
		_ = packetConn.Close()
		return nil, nil, errNotARawIPSocket
	}
	if dfErr := setDontFragment(conn); dfErr != nil {
		_ = conn.Close()
		return nil, nil, dfErr
	}

	seq := 0
	probe := func(ctx context.Context, size int) (ProbeOutcome, int, error) {
		seq++
		return sendSizedProbe(ctx, conn, dst, flowID, seq, size, timeout)
	}

	return probe, conn.Close, nil
}

// sendSizedProbe sends one don't-fragment echo request of a given total packet
// size and classifies what comes back.
func sendSizedProbe(
	ctx context.Context,
	conn *net.IPConn,
	dst net.IP,
	flowID, seq, size int,
	timeout time.Duration,
) (ProbeOutcome, int, error) {
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Body: &icmp.Echo{ID: flowID, Seq: seq, Data: make([]byte, PayloadForMTU(size))},
	}
	packet, err := msg.Marshal(nil)
	if err != nil {
		return ProbeLost, 0, err
	}

	if _, writeErr := conn.WriteTo(packet, &net.IPAddr{IP: dst}); writeErr != nil {
		if oversizedLocally(writeErr) {
			// The kernel refused it, so it never reached the wire. That is a
			// fact about this host rather than the path, but for the search it
			// means the same thing: this size does not work.
			return ProbeTooBig, 0, nil
		}
		return ProbeLost, 0, writeErr
	}

	return awaitProbeReply(ctx, conn, flowID, seq, timeout)
}

// awaitProbeReply reads until it finds the reply to this probe or the deadline
// passes, discarding ICMP belonging to anything else on the host.
func awaitProbeReply(
	ctx context.Context,
	conn *net.IPConn,
	flowID, seq int,
	timeout time.Duration,
) (ProbeOutcome, int, error) {
	deadline := time.Now().Add(timeout)
	buf := make([]byte, traceICMPBufferSize)

	for {
		// Cancellation is reported rather than swallowed: the search stops on
		// it either way, but a run abandoned by the operator and one that
		// found nothing are different outcomes to explain.
		if err := ctx.Err(); err != nil {
			return ProbeLost, 0, err
		}
		if err := conn.SetReadDeadline(deadline); err != nil {
			return ProbeLost, 0, err
		}

		// A read deadline elapsing means nothing came back, which is a probe
		// outcome rather than a failure -- the search is built to interpret
		// silence, so it is returned as ProbeLost with no error.
		n, ok := readPacket(conn, buf)
		if !ok {
			return ProbeLost, 0, nil
		}
		raw := buf[:n]

		reply, parseErr := icmp.ParseMessage(protocolICMP, raw)
		if parseErr != nil {
			continue
		}
		id, replySeq, ok := echoIDSeq(reply)
		if !ok || id != flowID || replySeq != seq {
			continue
		}

		return classifyProbeReply(reply, raw)
	}
}

// readPacket reads one packet, reporting failure as "nothing arrived" rather
// than as an error: at this point in the search a silent socket and a silent
// path are the same observation.
func readPacket(conn *net.IPConn, buf []byte) (int, bool) {
	n, _, err := conn.ReadFrom(buf)
	if err != nil {
		return 0, false
	}
	return n, true
}

// classifyProbeReply turns a matched reply into a probe outcome.
func classifyProbeReply(reply *icmp.Message, raw []byte) (ProbeOutcome, int, error) {
	switch reply.Body.(type) {
	case *icmp.Echo:
		return ProbeArrived, 0, nil
	case *icmp.DstUnreach:
		if reply.Code == codeFragmentationNeeded {
			return ProbeTooBig, advertisedNextHopMTU(raw), nil
		}
		// Any other unreachable is a routing or policy answer, not a size one.
		return ProbeLost, 0, nil
	default:
		return ProbeLost, 0, nil
	}
}

// advertisedNextHopMTU reads the next-hop MTU a router puts in the second half
// of the unused word of its fragmentation-needed error (RFC 1191 section 4).
// Zero means it advertised nothing -- the field postdates the original ICMP
// spec and remains a SHOULD -- and the search bisects instead.
//
// It reads the raw datagram rather than the parsed body because x/net/icmp
// strips those four bytes and does not expose them. That field is what turns a
// multi-probe search into a single round trip, so it is worth parsing by hand.
func advertisedNextHopMTU(raw []byte) int {
	const (
		nextHopMTUOffset = 6
		nextHopMTUEnd    = 8
	)
	if len(raw) < nextHopMTUEnd {
		return 0
	}
	return int(binary.BigEndian.Uint16(raw[nextHopMTUOffset:nextHopMTUEnd]))
}
