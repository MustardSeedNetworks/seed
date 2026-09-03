package topology

import "testing"

// A device that reports both a routable address and a link-local one for the
// same MAC must not end up reachable only by the link-local. fe80:: needs a
// zone index to be usable at all, so storing it as a node's primary address
// produces something no caller can dial (#1371).
func TestRoutableAddressBeatsLinkLocal(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		current   string
		candidate string
		want      bool
	}{
		{"first address is taken", "", "192.0.2.10", true},
		{"link-local does not replace IPv4", "192.0.2.10", "fe80::1", false},
		{"link-local does not replace global IPv6", "2001:db8::1", "fe80::1", false},
		{"global IPv6 replaces link-local", "fe80::1", "2001:db8::1", true},
		{"IPv4 replaces link-local", "fe80::1", "192.0.2.10", true},
		{"global replaces unique-local", "fd00::1", "2001:db8::1", true},
		{"unique-local does not replace global", "2001:db8::1", "fd00::1", false},
		{"unique-local replaces link-local", "fe80::1", "fd00::1", true},
		{"loopback is never chosen", "", "127.0.0.1", false},
		{"unparseable is never chosen", "", "not-an-address", false},
		{"equal rank keeps the first seen", "192.0.2.10", "198.51.100.7", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := preferAddress(tc.current, tc.candidate); got != tc.want {
				t.Errorf("preferAddress(%q, %q) = %v, want %v", tc.current, tc.candidate, got, tc.want)
			}
		})
	}
}
