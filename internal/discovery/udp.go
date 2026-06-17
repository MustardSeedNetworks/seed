//go:build !windows

package discovery

import (
	"context"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// udpProbeResult represents the outcome of a UDP probe attempt.
type udpProbeResult struct {
	success     bool
	rtt         time.Duration
	peer        net.Addr
	messageType ipv4.ICMPType
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
