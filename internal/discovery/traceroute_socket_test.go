//go:build !windows

package discovery_test

import (
	"context"
	"testing"
	"time"

	"golang.org/x/net/icmp"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// seed#2689: the tracer opened a raw ICMP socket and nothing else, so an
// unprivileged run — every macOS install, and any Linux host whose package did
// not get CAP_NET_RAW — got "failed to create ICMP socket: ... operation not
// permitted" and Path Analysis showed no hops at all. The sweep already falls
// back to the datagram ICMP socket (seed#2629); the tracer now does too.
//
// These tests assert the fallback wherever the kernel permits that socket, and
// stand down where it does not: Linux gates it on net.ipv4.ping_group_range,
// which is `1 0` (nobody) on a stock Ubuntu and read-only inside an
// unprivileged container, so on such a host there is no unprivileged traceroute
// to prove and the guarantee this covers does not exist.
func requireATracerouteSocketIsPossible(t *testing.T) {
	t.Helper()

	if raw, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0"); err == nil {
		_ = raw.Close()
		return // privileged here; the tracer takes the raw path
	}
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		t.Skipf("this host allows neither ICMP socket to an unprivileged process: %v", err)
	}
	_ = conn.Close()
}

func TestTraceICMPReachesLoopbackInWhicheverModeItOpened(t *testing.T) {
	requireATracerouteSocketIsPossible(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := discovery.NewTracer(2*time.Second, 5).TraceICMP(ctx, "127.0.0.1")

	if result.Error != "" {
		t.Fatalf(
			"traceroute to loopback reported %q; the datagram fallback is what this test exists to prove",
			result.Error,
		)
	}
	if len(result.Hops) == 0 {
		t.Fatal("traceroute to loopback returned no hops")
	}
	// The reply match and the destination address type both differ between the
	// two modes, so a round trip is the only assertion that catches a mode the
	// tracer opened but cannot actually use.
	first := result.Hops[0]
	if first.State != "reply" || first.IP != "127.0.0.1" {
		t.Errorf("first hop = %+v; want a reply from 127.0.0.1", first)
	}
	if !result.Completed {
		t.Error("traceroute to loopback did not complete")
	}
}
