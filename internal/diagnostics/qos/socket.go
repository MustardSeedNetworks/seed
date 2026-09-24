package qos

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"net"
	"net/netip"
	"runtime"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	// ecnBits are the low two bits of the TOS byte and traffic class; the
	// DSCP sits above them.
	ecnBits = 2

	// oobSize holds the one control message a listen asks for, with room
	// for a platform that attaches more.
	oobSize = 128
)

// Send puts the requested probes on the wire to the target's listen.
func Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	s, err := parseSend(req)
	if err != nil {
		return nil, err
	}
	// Unconnected, so an ICMP port-unreachable for a probe that beat the
	// far side's listen is not handed back as a write error that ends the
	// burst: the listen's result is the verdict, not the sender's socket.
	network := "udp4"
	if s.target.Addr().Is6() {
		network = "udp6"
	}
	conn, err := net.ListenUDP(network, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	var run [8]byte
	_, _ = rand.Read(run[:])
	res, err := send(ctx, newUDPProbeConn(conn, s.target), s, binary.BigEndian.Uint64(run[:]), pace)
	if err != nil {
		return nil, err
	}
	res.Marked = runtime.GOOS != "windows"
	return res, nil
}

func pace(ctx context.Context) error {
	t := time.NewTimer(probeInterval)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// udpProbeConn marks each datagram through the socket's TOS or traffic class.
// The ECN bits stay zero.
type udpProbeConn struct {
	conn   *net.UDPConn
	target netip.AddrPort
	v4     *ipv4.Conn
	v6     *ipv6.Conn
}

func newUDPProbeConn(conn *net.UDPConn, target netip.AddrPort) *udpProbeConn {
	if target.Addr().Is6() {
		return &udpProbeConn{conn: conn, target: target, v6: ipv6.NewConn(conn)}
	}
	return &udpProbeConn{conn: conn, target: target, v4: ipv4.NewConn(conn)}
}

func (c *udpProbeConn) setDSCP(dscp uint8) error {
	if c.v6 != nil {
		return c.v6.SetTrafficClass(int(dscp) << ecnBits)
	}
	return c.v4.SetTOS(int(dscp) << ecnBits)
}

func (c *udpProbeConn) write(b []byte) error {
	_, err := c.conn.WriteToUDPAddrPort(b, c.target)
	return err
}

// Listen opens the requested port, reads the marking of every probe that
// reaches it, and closes the port when the window closes or ctx ends.
func Listen(ctx context.Context, req ListenRequest) (*ListenResult, error) {
	s, err := parseListen(req)
	if err != nil {
		return nil, err
	}
	src, err := openSocket(s)
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.conn.Close() }()
	return collect(ctx, src, s, time.Now)
}

// socketSource is a UDP socket that asks the kernel for each datagram's TOS
// or traffic class alongside it.
type socketSource struct {
	conn     *net.UDPConn
	v6       bool
	observes bool
	oob      []byte
}

func openSocket(s listenSpec) (*socketSource, error) {
	conn, err := net.ListenUDP(s.network(), &net.UDPAddr{Port: s.port})
	if err != nil {
		return nil, err
	}
	src := &socketSource{conn: conn, v6: s.v6, oob: make([]byte, oobSize)}
	src.observes = enableTOS(conn, s.v6) == nil
	return src, nil
}

func (s *socketSource) read(buf []byte) (int, netip.Addr, int, error) {
	if !s.observes {
		n, from, err := s.conn.ReadFromUDPAddrPort(buf)
		return n, from.Addr().WithZone(""), -1, err
	}
	n, oobn, _, from, err := s.conn.ReadMsgUDPAddrPort(buf, s.oob)
	if err != nil {
		return 0, netip.Addr{}, 0, err
	}
	dscp := -1
	if tos, ok := parseTOS(s.oob[:oobn], s.v6); ok {
		dscp = tos >> ecnBits
	}
	return n, from.Addr().WithZone(""), dscp, nil
}

func (s *socketSource) setDeadline(t time.Time) { _ = s.conn.SetReadDeadline(t) }

func (s *socketSource) interrupt() { _ = s.conn.SetReadDeadline(time.Unix(1, 0)) }

func (s *socketSource) observesDSCP() bool { return s.observes }
