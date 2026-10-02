package multicast

import (
	"context"
	"net"
	"net/netip"
	"runtime"
	"testing"
	"time"

	"golang.org/x/net/ipv4"
)

// multicastInterface is an up interface that can carry an IPv4 group: lo0 on
// macOS, the first non-loopback one on Linux, whose lo has no MULTICAST flag.
func multicastInterface(t *testing.T) *net.Interface {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces: %v", err)
	}
	var fallback *net.Interface
	for i := range ifaces {
		ifi := &ifaces[i]
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagMulticast == 0 || !hasIPv4(ifi) {
			continue
		}
		if ifi.Flags&net.FlagLoopback != 0 {
			return ifi
		}
		if fallback == nil {
			fallback = ifi
		}
	}
	if fallback == nil {
		t.Skip("no up interface with IPv4 and MULTICAST on this host")
	}
	return fallback
}

func hasIPv4(ifi *net.Interface) bool {
	addrs, err := ifi.Addrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil {
			return true
		}
	}
	return false
}

// freeUDPPort asks the kernel for a port nothing on this host is bound to.
func freeUDPPort(t *testing.T) int {
	t.Helper()
	c, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	defer func() { _ = c.Close() }()
	addr, ok := c.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("LocalAddr is %T", c.LocalAddr())
	}
	return addr.Port
}

// sendUntil multicasts payloads to group out of ifi until done closes, so the
// assertion never depends on guessing when the listener joined.
func sendUntil(t *testing.T, ifi *net.Interface, group string, port int, done <-chan struct{}) {
	t.Helper()
	c, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		t.Errorf("sender ListenUDP: %v", err)
		return
	}
	defer func() { _ = c.Close() }()
	p := ipv4.NewPacketConn(c)
	if err = p.SetMulticastInterface(ifi); err != nil {
		t.Errorf("SetMulticastInterface(%s): %v", ifi.Name, err)
		return
	}
	_ = p.SetMulticastLoopback(true)
	dst := &net.UDPAddr{IP: net.ParseIP(group), Port: port}
	payload := make([]byte, 188)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if _, err = c.WriteTo(payload, dst); err != nil {
				t.Errorf("WriteTo %s: %v", dst, err)
				return
			}
		}
	}
}

// End to end over a real interface: the join makes the kernel deliver the
// group, the count is the group's alone, and every sender named is this host.
// A second socket joined to a sibling group on the same port is what makes the
// destination check load-bearing on Linux, where a wildcard-bound socket is
// handed every group any socket on the host has joined for its port.
func TestListenHearsTheGroupAndOnlyTheGroup(t *testing.T) {
	ifi := multicastInterface(t)
	port := freeUDPPort(t)
	const group, sibling = "239.255.42.99", "239.255.42.100"

	other, err := net.ListenMulticastUDP("udp4", ifi, &net.UDPAddr{IP: net.ParseIP(sibling), Port: port})
	if err != nil {
		t.Fatalf("join sibling group: %v", err)
	}
	defer func() { _ = other.Close() }()

	done := make(chan struct{})
	senders := make(chan struct{}, 2)
	for _, g := range []string{group, sibling} {
		go func() {
			defer func() { senders <- struct{}{} }()
			sendUntil(t, ifi, g, port, done)
		}()
	}

	res, err := Listen(t.Context(), ListenRequest{Group: group, Port: port, Interface: ifi.Name, DurationSeconds: 1})
	close(done)
	<-senders
	<-senders
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	if res.Packets == 0 {
		t.Fatalf("heard nothing on %s:%d via %s", group, port, ifi.Name)
	}
	// x/net cannot read a datagram's destination on Windows, so there the
	// result must admit it counted the whole port.
	if wantFiltered := runtime.GOOS != "windows"; res.GroupFiltered != wantFiltered {
		t.Fatalf("GroupFiltered = %v on %s, want %v", res.GroupFiltered, runtime.GOOS, wantFiltered)
	}
	if res.Bytes != res.Packets*188 {
		t.Fatalf("bytes %d for %d packets of 188", res.Bytes, res.Packets)
	}
	var fromSources uint64
	for _, s := range res.Sources {
		if !isLocal(t, s.Address) {
			t.Fatalf("source %s is not an address of this host", s.Address)
		}
		fromSources += s.Packets
	}
	if fromSources != res.Packets {
		t.Fatalf("sources account for %d of %d packets", fromSources, res.Packets)
	}
	// At one packet per 20ms per group over one second, counting the sibling
	// too would roughly double this; a generous ceiling keeps it robust.
	if res.GroupFiltered && res.Packets > 75 {
		t.Fatalf("heard %d packets in 1s from a 50/s sender: the sibling group was counted", res.Packets)
	}
}

func isLocal(t *testing.T, s string) bool {
	t.Helper()
	want, err := netip.ParseAddr(s)
	if err != nil {
		return false
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatalf("InterfaceAddrs: %v", err)
	}
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok {
			if got, valid := netip.AddrFromSlice(n.IP); valid && got.Unmap() == want {
				return true
			}
		}
	}
	return false
}

func TestListenRefusesBeforeOpeningASocket(t *testing.T) {
	t.Parallel()

	if _, err := Listen(
		t.Context(),
		ListenRequest{Group: "239.1.1.1", Port: 5004, Interface: "does-not-exist0"},
	); err == nil {
		t.Fatal("Listen on a missing interface succeeded")
	}
}

// A cancel must end a blocked read at once, not at the end of the window: the
// operator pressed stop, and the group membership should go with it.
func TestCancelEndsABlockedListenPromptly(t *testing.T) {
	t.Parallel()

	ifi := multicastInterface(t)
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	started := time.Now()
	res, err := Listen(
		ctx,
		ListenRequest{Group: "239.255.42.101", Port: freeUDPPort(t), Interface: ifi.Name, DurationSeconds: 30},
	)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("cancelled listen took %s", elapsed)
	}
	if res.ListenedMs >= 5000 || res.Packets != 0 {
		t.Fatalf("listened %dms, heard %d packets on a silent group", res.ListenedMs, res.Packets)
	}
}
