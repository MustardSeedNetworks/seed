package autotest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// pingTimeout bounds one echo to an internet target.
const pingTimeout = 2 * time.Second

// errNoAnswer is a ping that timed out without an error of its own.
var errNoAnswer = errors.New("no echo reply")

// PingHost resolves host when it is a name and sends it one ICMP echo through
// iface, over an unprivileged datagram socket where the host allows one. The
// echo leaves through iface whatever the host's routes say, so a target
// reached over another link does not count for this one. An empty iface
// leaves the echo on host routing.
func PingHost(ctx context.Context, iface, host string) (time.Duration, error) {
	addr, err := resolve(ctx, host)
	if err != nil {
		return 0, err
	}
	pinger, err := openPinger(iface, addr.Is6())
	if err != nil {
		return 0, err
	}
	defer func() { _ = pinger.Close() }()

	res := pinger.Ping(ctx, addr.String())
	if !res.Reachable {
		if res.Error != nil {
			return 0, res.Error
		}
		return 0, errNoAnswer
	}
	return res.RTT, nil
}

func openPinger(iface string, ipv6 bool) (*enumerate.ICMPPinger, error) {
	if iface == "" {
		if ipv6 {
			return enumerate.NewICMPv6Pinger(pingTimeout)
		}
		return enumerate.NewICMPPinger(pingTimeout)
	}
	link, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("open ICMP socket on %s: %w", iface, err)
	}
	pinger, err := enumerate.NewICMPPingerOn(pingTimeout, ipv6, link)
	if err != nil {
		return nil, fmt.Errorf("open ICMP socket on %s: %w", iface, err)
	}
	return pinger, nil
}

func resolve(ctx context.Context, host string) (netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Unmap(), nil
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("resolve %s: %w", host, err)
	}
	return addrs[0].Unmap(), nil
}
