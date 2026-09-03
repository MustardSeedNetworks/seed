//go:build darwin

package enumerate

import (
	"net"
	"syscall"
	"testing"

	"golang.org/x/net/route"
)

// linkRoute builds the shape the kernel hands back for one neighbour, so the
// conversion can be exercised without a neighbour table in front of it.
func linkRoute(ip [4]byte, mac []byte, name string) *route.RouteMessage {
	return &route.RouteMessage{
		Index: 1,
		Addrs: []route.Addr{
			syscall.RTAX_DST:     &route.Inet4Addr{IP: ip},
			syscall.RTAX_GATEWAY: &route.LinkAddr{Index: 1, Name: name, Addr: mac},
		},
	}
}

func TestARPEntryFromRoute(t *testing.T) {
	// RFC 7042 documentation address; the values only have to round-trip.
	mac := []byte{0x00, 0x00, 0x5e, 0x00, 0x53, 0x01}

	tests := []struct {
		name string
		msg  *route.RouteMessage
		want *ARPEntry // nil means the row must be skipped
	}{
		{
			name: "resolved neighbour",
			msg:  linkRoute([4]byte{192, 0, 2, 1}, mac, "en0"),
			want: &ARPEntry{IP: "192.0.2.1", MAC: "00:00:5E:00:53:01", Interface: "en0", State: "REACHABLE"},
		},
		{
			// `arp -an` prints this as "(incomplete)": an unanswered probe, not
			// a neighbour. The Linux reader drops the same row.
			name: "unresolved probe has no link address",
			msg:  linkRoute([4]byte{192, 0, 2, 234}, nil, "en0"),
		},
		{
			// macOS keeps multicast groups in this table; a group is not a host.
			name: "multicast group",
			msg:  linkRoute([4]byte{224, 0, 0, 251}, []byte{0x01, 0x00, 0x5e, 0x00, 0x00, 0xfb}, "en0"),
		},
		{
			name: "gateway is not a link address",
			msg: &route.RouteMessage{Addrs: []route.Addr{
				syscall.RTAX_DST:     &route.Inet4Addr{IP: [4]byte{192, 0, 2, 1}},
				syscall.RTAX_GATEWAY: &route.Inet4Addr{IP: [4]byte{192, 0, 2, 254}},
			}},
		},
		{
			name: "destination is not IPv4",
			msg: &route.RouteMessage{Addrs: []route.Addr{
				syscall.RTAX_DST:     &route.Inet6Addr{},
				syscall.RTAX_GATEWAY: &route.LinkAddr{Addr: mac},
			}},
		},
		{
			name: "message carries no gateway",
			msg: &route.RouteMessage{Addrs: []route.Addr{
				syscall.RTAX_DST: &route.Inet4Addr{IP: [4]byte{192, 0, 2, 1}},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := arpEntryFromRoute(tt.msg)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("arpEntryFromRoute() = %+v, want nil", got)
				}

				return
			}
			if got == nil {
				t.Fatal("arpEntryFromRoute() = nil, want an entry")
			}
			if got.IP != tt.want.IP || got.MAC != tt.want.MAC ||
				got.Interface != tt.want.Interface || got.State != tt.want.State {
				t.Errorf("arpEntryFromRoute() = %+v, want %+v", got, tt.want)
			}
			if got.LastSeen.IsZero() {
				t.Error("LastSeen is zero; the caller ages entries by it")
			}
		})
	}
}

// TestReadARPTablePlatformSeesTheGateway drives the real sysctl.
//
// The regression it guards is not "the parser is wrong" but "the read comes
// back empty and says nothing is wrong" (#2272), which only a live table can
// show. The default gateway is the one neighbour a host with any network at
// all must have resolved, so it is the oracle.
//
// It skips under `go test`, and that is not the test being lazy. macOS gates
// the IPv4 neighbour cache behind Local Network privacy, attributed to the
// responsible process: a binary the `go` tool spawns is denied and the kernel
// answers with zero bytes and no error, while the same binary run from the
// shell reads the table. Measured on Darwin 27.0.0, 2026-09-03:
//
//	go run .        AF_INET NET_RT_FLAGS RTF_LLINFO -> 0 bytes
//	./probe         AF_INET NET_RT_FLAGS RTF_LLINFO -> 1292 bytes, 9 entries
//	sh -c ./probe   AF_INET NET_RT_FLAGS RTF_LLINFO -> 1292 bytes, 9 entries
//
// So run it as its own binary to get the assertion:
//
//	go test -c -o /tmp/enumerate.test ./internal/discovery/enumerate && /tmp/enumerate.test -test.run SeesTheGateway
func TestReadARPTablePlatformSeesTheGateway(t *testing.T) {
	gateway := defaultGatewayForTest(t)
	if gateway == "" {
		t.Skip("no IPv4 default route on this host; nothing to assert against")
	}

	// The skip is decided by an independent read, not by the reader under
	// test: otherwise a reader that asks the kernel the wrong question skips
	// instead of failing, which is the defect rather than a reason to pass.
	control, err := route.FetchRIB(syscall.AF_INET, ribTypeFlags, syscall.RTF_LLINFO)
	if err != nil {
		t.Fatalf("FetchRIB: %v", err)
	}
	if len(control) == 0 {
		t.Skip("this process cannot see the IPv4 neighbour cache; run the test binary directly, see the doc comment")
	}

	entries, err := (&ARPScanner{}).readARPTablePlatform()
	if err != nil {
		t.Fatalf("readARPTablePlatform() error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("the kernel holds %d bytes of link-layer routes and the reader found no entries", len(control))
	}

	for _, entry := range entries {
		if entry.IP != gateway {
			continue
		}
		if _, parseErr := net.ParseMAC(entry.MAC); parseErr != nil {
			t.Fatalf("gateway %s has MAC %q, which does not parse: %v", gateway, entry.MAC, parseErr)
		}

		return
	}

	t.Fatalf("the default gateway %s is not in the %d entries read from the neighbour cache", gateway, len(entries))
}

// defaultGatewayForTest returns this host's IPv4 default gateway, or "".
func defaultGatewayForTest(t *testing.T) string {
	t.Helper()

	rib, err := route.FetchRIB(syscall.AF_INET, route.RIBTypeRoute, 0)
	if err != nil {
		t.Fatalf("FetchRIB: %v", err)
	}
	msgs, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		t.Fatalf("ParseRIB: %v", err)
	}

	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok || len(rm.Addrs) <= syscall.RTAX_GATEWAY {
			continue
		}
		dst, ok := rm.Addrs[syscall.RTAX_DST].(*route.Inet4Addr)
		if !ok || !net.IP(dst.IP[:]).IsUnspecified() {
			continue
		}
		if gw, isIPv4 := rm.Addrs[syscall.RTAX_GATEWAY].(*route.Inet4Addr); isIPv4 {
			return net.IP(gw.IP[:]).String()
		}
	}

	return ""
}
