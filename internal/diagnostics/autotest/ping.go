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

// PingHost resolves host when it is a name and sends it one ICMP echo, over
// an unprivileged datagram socket where the host allows one.
func PingHost(ctx context.Context, host string) (time.Duration, error) {
	addr, err := resolve(ctx, host)
	if err != nil {
		return 0, err
	}
	open := enumerate.NewICMPPinger
	if addr.Is6() {
		open = enumerate.NewICMPv6Pinger
	}
	pinger, err := open(pingTimeout)
	if err != nil {
		return 0, fmt.Errorf("open ICMP socket: %w", err)
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
