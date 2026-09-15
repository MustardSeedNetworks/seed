package enumerate_test

import (
	"context"
	"testing"
	"time"

	"golang.org/x/net/icmp"

	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
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
		t.Fatalf("loopback unreachable (privileged=%v): the echo was sent but the reply never matched", p.Privileged())
	}
}
