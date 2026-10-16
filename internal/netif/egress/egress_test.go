package egress_test

import (
	"context"
	"net"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/netif/egress"
)

func TestControlWithoutAnInterfaceLeavesHostRouting(t *testing.T) {
	t.Parallel()
	if egress.Control("") != nil {
		t.Fatal("Control(\"\") returned a control function; want nil so the socket keeps host routing")
	}
}

// A socket that cannot be pinned must not open at all: opening it unpinned
// would quietly test whichever link carries the route.
func TestControlRefusesAnUnknownInterface(t *testing.T) {
	t.Parallel()
	lc := net.ListenConfig{Control: egress.Control("seed-no-such-if0")}
	conn, err := lc.ListenPacket(context.Background(), "udp4", "127.0.0.1:0")
	if err == nil {
		_ = conn.Close()
		t.Fatal("ListenPacket succeeded on an interface that does not exist")
	}
}

// Traffic over the pinned interface still flows.
func TestControlPinsADialThatStillReachesItsPeer(t *testing.T) {
	t.Parallel()
	lo := loopbackName(t)

	srv, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket: %v", err)
	}
	defer func() { _ = srv.Close() }()

	d := net.Dialer{Control: egress.Control(lo)}
	conn, err := d.DialContext(t.Context(), "udp4", srv.LocalAddr().String())
	if err != nil {
		t.Fatalf("dial pinned to %s: %v", lo, err)
	}
	defer func() { _ = conn.Close() }()
	if _, err = conn.Write([]byte("seed")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, 8)
	n, _, err := srv.ReadFrom(buf)
	if err != nil || string(buf[:n]) != "seed" {
		t.Fatalf("peer read %q, %v; want \"seed\"", buf[:n], err)
	}
}

func loopbackName(t *testing.T) string {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces: %v", err)
	}
	for _, i := range ifaces {
		if i.Flags&net.FlagLoopback != 0 {
			return i.Name
		}
	}
	t.Skip("no loopback interface")
	return ""
}
