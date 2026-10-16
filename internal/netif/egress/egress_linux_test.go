//go:build linux

package egress_test

import (
	"context"
	"net"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/MustardSeedNetworks/seed/internal/netif/egress"
)

func TestControlSetsBindToDevice(t *testing.T) {
	t.Parallel()
	lo := loopbackName(t)

	lc := net.ListenConfig{Control: egress.Control(lo)}
	conn, err := lc.ListenPacket(context.Background(), "udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket pinned to %s: %v", lo, err)
	}
	defer func() { _ = conn.Close() }()

	raw, err := conn.(*net.UDPConn).SyscallConn()
	if err != nil {
		t.Fatalf("SyscallConn: %v", err)
	}
	var dev string
	var optErr error
	if ctlErr := raw.Control(func(fd uintptr) {
		dev, optErr = unix.GetsockoptString(int(fd), unix.SOL_SOCKET, unix.SO_BINDTODEVICE)
	}); ctlErr != nil {
		t.Fatalf("Control: %v", ctlErr)
	}
	if optErr != nil || dev != lo {
		t.Fatalf("SO_BINDTODEVICE = %q, %v; want %q", dev, optErr, lo)
	}
}
