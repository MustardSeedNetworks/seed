//go:build !windows

package discovery

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

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

// tcpDialChannels holds channels for async TCP dial results.
type tcpDialChannels struct {
	connCh chan net.Conn
	errCh  chan error
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
