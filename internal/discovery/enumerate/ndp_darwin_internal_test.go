//go:build darwin

package enumerate

import (
	"net"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/net/route"
)

// neighbourRoute builds the shape the kernel hands back for one IPv6
// neighbour-cache entry, so the conversion can be exercised without a
// neighbour table in front of it.
func neighbourRoute(ip string, zone int, flags int, mac []byte) *route.RouteMessage {
	dst := &route.Inet6Addr{ZoneID: zone}
	copy(dst.IP[:], net.ParseIP(ip).To16())

	return &route.RouteMessage{
		Flags: flags,
		Index: 1,
		Addrs: []route.Addr{
			syscall.RTAX_DST:     dst,
			syscall.RTAX_GATEWAY: &route.LinkAddr{Index: 12, Name: "en0", Addr: mac},
		},
	}
}

func TestNDPRowFromRoute(t *testing.T) {
	// RFC 7042 documentation address; the values only have to round-trip.
	mac := []byte{0x00, 0x00, 0x5e, 0x00, 0x53, 0x01}

	tests := []struct {
		name    string
		msg     *route.RouteMessage
		want    ndpRow
		wantRow bool
	}{
		{
			name: "resolved link-local neighbour",
			msg:  neighbourRoute("fe80::1865:2d21:d7d8:673e", 12, 0, mac),
			want: ndpRow{
				ip:    net.ParseIP("fe80::1865:2d21:d7d8:673e"),
				zone:  12,
				iface: "en0",
				mac:   "00:00:5e:00:53:01",
			},
			wantRow: true,
		},
		{
			// ndp prints "(incomplete)": the kernel is still resolving it.
			name:    "unresolved neighbour keeps its row with no MAC",
			msg:     neighbourRoute("2001:db8::7", 0, 0, nil),
			want:    ndpRow{ip: net.ParseIP("2001:db8::7"), iface: "en0"},
			wantRow: true,
		},
		{
			// #2337: the placeholder a filtered reader is handed is not a MAC.
			name:    "placeholder link address",
			msg:     neighbourRoute("fe80::2", 12, 0, []byte{0x02, 0, 0, 0, 0, 0}),
			want:    ndpRow{ip: net.ParseIP("fe80::2"), zone: 12, iface: "en0"},
			wantRow: true,
		},
		{
			// ndp prints these "permanent"; they are this host, not a neighbour.
			name: "host's own address",
			msg:  neighbourRoute("fe80::1894:73c5:7543:b644", 12, syscall.RTF_LOCAL|syscall.RTF_LLINFO, mac),
		},
		{
			name: "multicast group",
			msg:  neighbourRoute("ff02::fb", 12, 0, []byte{0x33, 0x33, 0, 0, 0, 0xfb}),
		},
		{
			// ::1 is in the table with an address, not a link, as its gateway.
			name: "gateway is not a link address",
			msg: &route.RouteMessage{Addrs: []route.Addr{
				syscall.RTAX_DST:     &route.Inet6Addr{IP: [16]byte{15: 1}},
				syscall.RTAX_GATEWAY: &route.Inet6Addr{IP: [16]byte{15: 1}},
			}},
		},
		{
			name: "destination is not IPv6",
			msg: &route.RouteMessage{Addrs: []route.Addr{
				syscall.RTAX_DST:     &route.Inet4Addr{IP: [4]byte{192, 0, 2, 1}},
				syscall.RTAX_GATEWAY: &route.LinkAddr{Addr: mac},
			}},
		},
		{
			name: "truncated message",
			msg:  &route.RouteMessage{Addrs: []route.Addr{&route.Inet6Addr{}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ndpRowFromRoute(tt.msg)
			if ok != tt.wantRow {
				t.Fatalf("row = %v, want %v (%+v)", ok, tt.wantRow, got)
			}
			if !ok {
				return
			}
			if !got.ip.Equal(tt.want.ip) || got.zone != tt.want.zone || got.iface != tt.want.iface ||
				got.mac != tt.want.mac {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNDPLinkLayer(t *testing.T) {
	tests := []struct {
		name string
		addr []byte
		want string
	}{
		{"macOS unresolved placeholder", []byte{0x02, 0, 0, 0, 0, 0}, ""},
		{"all-zero address", []byte{0, 0, 0, 0, 0, 0}, ""},
		{"no address", nil, ""},
		{"not Ethernet length", []byte{0x00, 0xc0, 0x17}, ""},
		{"real address", []byte{0x00, 0xc0, 0x17, 0x53, 0x62, 0x06}, "00:c0:17:53:62:06"},
		{"real locally-administered address", []byte{0x02, 0, 0, 0, 0, 0x01}, "02:00:00:00:00:01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ndpLinkLayer(tt.addr); got != tt.want {
				t.Errorf("ndpLinkLayer(%v) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

func TestND6StateName(t *testing.T) {
	tests := []struct {
		state int32
		want  string
	}{
		{nd6LLInfoIncomplete, "INCOMPLETE"},
		{nd6LLInfoReachable, "REACHABLE"},
		{nd6LLInfoStale, "STALE"},
		{nd6LLInfoDelay, "DELAY"},
		{nd6LLInfoProbe, "PROBE"},
		{-2, "UNKNOWN"}, // ND6_LLINFO_NOSTATE
	}

	for _, tt := range tests {
		if got := nd6StateName(tt.state); got != tt.want {
			t.Errorf("nd6StateName(%d) = %q, want %q", tt.state, got, tt.want)
		}
	}
}

// The ioctl request encodes the struct size, so a layout that drifts from the
// C struct sends the kernel a request it does not recognise.
func TestIn6NbrInfoMatchesTheKernelLayout(t *testing.T) {
	if size := unsafe.Sizeof(in6NbrInfo{}); size != 56 {
		t.Errorf("sizeof(in6_nbrinfo) = %d, want 56 (LP64)", size)
	}
	if siocgnbrinfoIn6 != 0xc038694e {
		t.Errorf("SIOCGNBRINFO_IN6 = %#x, want 0xc038694e", siocgnbrinfoIn6)
	}
}

// The scanner must read the live table, not return an empty map as it did
// before #2089. Under `go test` macOS may hand the process a filtered table
// (ADR-0030), so this asserts the read works, not what it finds.
func TestReadNDPTable_ReadsTheLiveTable(t *testing.T) {
	live, err := readNDPTable("")
	if err != nil {
		t.Fatalf("readNDPTable: %v", err)
	}
	for _, n := range live {
		t.Logf("%-40s %-17s %-10s router=%v", n.IPv6, n.MAC, n.State, n.IsRouter)
		if n.IPv6 == "" || n.State == "" {
			t.Errorf("incomplete neighbour %+v", n)
		}
	}
}
