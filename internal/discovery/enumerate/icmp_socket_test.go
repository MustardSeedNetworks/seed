package enumerate_test

import (
	"context"
	"net"
	"testing"
	"time"

	"golang.org/x/net/icmp"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// seed#2629: the sweep used to open a raw ICMP socket and nothing else, so an
// unprivileged daemon — every macOS install, and any Linux host whose package
// did not get CAP_NET_RAW — probed no target network at all and still reported
// "icmp" as an active discovery method.
//
// These tests assert the fallback wherever the kernel permits a datagram ICMP
// socket, and stand down where it does not: Linux gates that socket on
// net.ipv4.ping_group_range, which is `1 0` (nobody) on a stock Ubuntu and is
// read-only inside an unprivileged container, so on such a host there is no
// unprivileged sweep to prove and the guarantee this covers does not exist.

// requireAnUnprivilegedSocketIsPossible skips unless this process can obtain the
// datagram ICMP socket the fallback depends on. Probing it directly, rather than
// branching on GOOS or euid, is what keeps the skip honest: it names the exact
// capability the production path needs.
func requireAnUnprivilegedSocketIsPossible(t *testing.T) {
	t.Helper()

	if raw, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0"); err == nil {
		_ = raw.Close()
		return // privileged here; NewICMPPinger takes the raw path
	}
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		t.Skipf("this host allows neither ICMP socket to an unprivileged process: %v", err)
	}
	_ = conn.Close()
}

func TestNewICMPPingerOpensASocketWithoutPrivilege(t *testing.T) {
	requireAnUnprivilegedSocketIsPossible(t)

	p, err := enumerate.NewICMPPinger(time.Second)
	if err != nil {
		t.Fatalf("NewICMPPinger: %v (the datagram fallback is what this test exists to prove)", err)
	}
	if closeErr := p.Close(); closeErr != nil {
		t.Errorf("Close: %v", closeErr)
	}
}

func TestPingerReachesLoopbackInWhicheverModeItOpened(t *testing.T) {
	requireAnUnprivilegedSocketIsPossible(t)

	p, err := enumerate.NewICMPPinger(2 * time.Second)
	if err != nil {
		t.Fatalf("NewICMPPinger: %v", err)
	}
	defer func() { _ = p.Close() }()

	// The reply match and the destination address type both differ between the
	// two modes, so a round trip is the only assertion that catches a mode the
	// pinger opened but cannot actually use.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res := p.Ping(ctx, "127.0.0.1")
	if res.Error != nil {
		t.Fatalf("Ping(127.0.0.1): %v (privileged=%v)", res.Error, p.Privileged())
	}
	if !res.Reachable {
		testutil.SkipIfHostDropsLoopbackICMP(t)
		t.Fatalf("loopback unreachable (privileged=%v): the echo was sent but the reply never matched", p.Privileged())
	}
}

// seed#3251: the pinger opened only ICMPv4 sockets, so the IPv6 gateway card
// read 100 % loss for every router. An ICMPv6 pinger must make the round trip
// in whichever mode it opened, as the IPv4 one does above.
func TestICMPv6PingerReachesLoopback(t *testing.T) {
	p, err := enumerate.NewICMPv6Pinger(2 * time.Second)
	if err != nil {
		t.Skipf("this host allows no ICMPv6 socket to this process: %v", err)
	}
	defer func() { _ = p.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res := p.Ping(ctx, "::1")
	if res.Error != nil {
		t.Fatalf("Ping(::1): %v (privileged=%v)", res.Error, p.Privileged())
	}
	if !res.Reachable {
		t.Fatalf("::1 unreachable (privileged=%v): the echo was sent but the reply never matched", p.Privileged())
	}
}

// P-E1: AutoTest's internet stage pings through the interface under test. A
// pinger pinned to an interface must still make the round trip over that
// interface, in whichever mode it opened.
func TestPingerOnAnInterfaceReachesItsOwnAddress(t *testing.T) {
	requireAnUnprivilegedSocketIsPossible(t)

	lo := loopback(t)
	p, err := enumerate.NewICMPPingerOn(2*time.Second, false, lo)
	if err != nil {
		t.Fatalf("NewICMPPingerOn(%s): %v", lo.Name, err)
	}
	defer func() { _ = p.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res := p.Ping(ctx, "127.0.0.1")
	if res.Error != nil {
		t.Fatalf("Ping(127.0.0.1) on %s: %v (privileged=%v)", lo.Name, res.Error, p.Privileged())
	}
	if !res.Reachable {
		testutil.SkipIfHostDropsLoopbackICMP(t)
		t.Fatalf("loopback unreachable on %s (privileged=%v)", lo.Name, p.Privileged())
	}
}

func loopback(t *testing.T) *net.Interface {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces: %v", err)
	}
	for i := range ifaces {
		if ifaces[i].Flags&net.FlagLoopback != 0 {
			return &ifaces[i]
		}
	}
	t.Skip("no loopback interface")
	return nil
}
