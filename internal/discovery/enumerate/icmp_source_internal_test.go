package enumerate

import (
	"net"
	"testing"
)

// A pinger pinned to an interface sends from that interface's own address.
// A global address wins over a link-local one, which cannot reach beyond the
// link, and an interface with no address of the family has nothing to send
// from.
func TestSourceAddress(t *testing.T) {
	t.Parallel()

	cidr := func(s string) net.Addr {
		ip, ipnet, err := net.ParseCIDR(s)
		if err != nil {
			t.Fatalf("ParseCIDR(%q): %v", s, err)
		}
		ipnet.IP = ip
		return ipnet
	}
	tests := []struct {
		name    string
		addrs   []net.Addr
		ipv6    bool
		want    string
		wantErr bool
	}{
		{"ipv4", []net.Addr{cidr("fe80::1/64"), cidr("192.0.2.10/24")}, false, "192.0.2.10", false},
		{
			"global ipv6 over link-local",
			[]net.Addr{cidr("fe80::1/64"), cidr("2001:db8::10/64")},
			true,
			"2001:db8::10",
			false,
		},
		{
			"link-local ipv6 carries the zone",
			[]net.Addr{cidr("192.0.2.10/24"), cidr("fe80::1/64")},
			true,
			"fe80::1%eth1",
			false,
		},
		{"no ipv4", []net.Addr{cidr("2001:db8::10/64")}, false, "", true},
		{"no ipv6", []net.Addr{cidr("192.0.2.10/24")}, true, "", true},
		{"no addresses", nil, false, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := sourceAddress("eth1", tc.addrs, tc.ipv6)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("sourceAddress = %q, %v; want %q, error %v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}
