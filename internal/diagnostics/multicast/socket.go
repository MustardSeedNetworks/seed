package multicast

import (
	"context"
	"net"
	"net/netip"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// readBufferSize absorbs a burst: a stream at line rate outruns a reader that
// wakes once per datagram.
const readBufferSize = 1 << 20

// Listen joins the requested group on the named interface, counts what
// arrives for it, and leaves the group when the window closes or ctx ends.
func Listen(ctx context.Context, req ListenRequest) (*ListenResult, error) {
	s, err := parse(req, lookupInterface)
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

func lookupInterface(name string) (ifaceInfo, error) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return ifaceInfo{}, err
	}
	return ifaceInfo{up: ifi.Flags&net.FlagUp != 0, multicast: ifi.Flags&net.FlagMulticast != 0}, nil
}

// socketSource is a UDP socket joined to one group.
//
// It is bound to the port on the wildcard address, which is what
// ListenMulticastUDP does and what lets it share the port with the
// application under test (the runtime sets SO_REUSEADDR/SO_REUSEPORT). The
// cost is that it also receives datagrams addressed to other groups other
// sockets joined on that port, so each datagram's destination is read from
// the control message and checked against the group.
type socketSource struct {
	conn    *net.UDPConn
	v4      *ipv4.PacketConn
	v6      *ipv6.PacketConn
	filters bool
}

func openSocket(s spec) (*socketSource, error) {
	ifi, err := net.InterfaceByName(s.iface)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenMulticastUDP(s.network(), ifi, &net.UDPAddr{IP: s.group.AsSlice(), Port: s.port})
	if err != nil {
		return nil, err
	}
	_ = conn.SetReadBuffer(readBufferSize)

	src := &socketSource{conn: conn}
	if s.group.Is4() {
		src.v4 = ipv4.NewPacketConn(conn)
		src.filters = src.v4.SetControlMessage(ipv4.FlagDst, true) == nil
	} else {
		src.v6 = ipv6.NewPacketConn(conn)
		src.filters = src.v6.SetControlMessage(ipv6.FlagDst, true) == nil
	}
	return src, nil
}

func (s *socketSource) read(buf []byte) (int, netip.Addr, netip.Addr, error) {
	var (
		n    int
		from net.Addr
		dst  net.IP
		err  error
	)
	if s.v4 != nil {
		var cm *ipv4.ControlMessage
		n, cm, from, err = s.v4.ReadFrom(buf)
		if cm != nil {
			dst = cm.Dst
		}
	} else {
		var cm *ipv6.ControlMessage
		n, cm, from, err = s.v6.ReadFrom(buf)
		if cm != nil {
			dst = cm.Dst
		}
	}
	if err != nil {
		return 0, netip.Addr{}, netip.Addr{}, err
	}

	var sender netip.Addr
	if udp, ok := from.(*net.UDPAddr); ok {
		sender = udp.AddrPort().Addr().WithZone("")
	}
	dstAddr, _ := netip.AddrFromSlice(dst)
	return n, sender, dstAddr, nil
}

func (s *socketSource) setDeadline(t time.Time) { _ = s.conn.SetReadDeadline(t) }

func (s *socketSource) interrupt() { _ = s.conn.SetReadDeadline(time.Unix(1, 0)) }

func (s *socketSource) filtersByDestination() bool { return s.filters }
