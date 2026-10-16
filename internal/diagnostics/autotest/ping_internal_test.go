package autotest

import (
	"net/netip"
	"testing"
)

// A literal address is pinged as given, with an IPv4-mapped IPv6 form
// unmapped so the IPv4 pinger takes it.
func TestResolveTakesLiteralsWithoutALookup(t *testing.T) {
	t.Parallel()

	tests := []struct{ host, want string }{
		{"192.0.2.1", "192.0.2.1"},
		{"::ffff:192.0.2.1", "192.0.2.1"},
		{"2001:db8::1", "2001:db8::1"},
	}
	for _, tc := range tests {
		got, err := resolve(t.Context(), tc.host)
		if err != nil || got != netip.MustParseAddr(tc.want) {
			t.Fatalf("resolve(%q) = %v, %v; want %s", tc.host, got, err, tc.want)
		}
	}
}
