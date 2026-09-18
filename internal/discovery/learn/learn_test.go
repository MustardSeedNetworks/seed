package learn_test

import (
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery/learn"
)

// hospitalRouter is the shape #2695 was reported against: an edge router on the
// transit network whose routing table names the six site /24s behind it, mixed
// in with the entries a learner must never turn into a sweep target.
func hospitalRouter() learn.Device {
	return learn.Device{
		IP: "10.44.40.1",
		Routes: []learn.Route{
			{Destination: "0.0.0.0", Prefix: 0, Type: "remote", Protocol: "netmgmt"},
			{Destination: "10.44.10.0", Prefix: 24, Type: "remote", Protocol: "ospf"},
			{Destination: "10.44.20.0", Prefix: 24, Type: "remote", Protocol: "ospf"},
			{Destination: "10.44.30.0", Prefix: 24, Type: "remote", Protocol: "netmgmt"},
			{Destination: "10.44.50.0", Prefix: 24, Type: "remote", Protocol: "ospf"},
			{Destination: "10.44.60.0", Prefix: 24, Type: "remote", Protocol: "ospf"},
			{Destination: "10.44.70.0", Prefix: 24, Type: "local", Protocol: "local"},
			{Destination: "10.44.40.0", Prefix: 24, Type: "local", Protocol: "local"},
			{Destination: "10.44.40.9", Prefix: 32, Type: "local", Protocol: "local"},
			{Destination: "203.0.113.0", Prefix: 24, Type: "remote", Protocol: "bgp"},
			{Destination: "10.99.0.0", Prefix: 16, Type: "remote", Protocol: "other"},
			{Destination: "10.98.0.0", Prefix: 16, Type: "reject", Protocol: "other"},
			{Destination: "10.97.0.0", Prefix: 16, Type: "blackhole", Protocol: "other"},
		},
	}
}

func cidrs(cands []learn.Candidate) []string {
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		out = append(out, c.CIDR)
	}
	return out
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// transitNetwork is the network the sweep already covers in every case below:
// the link Seed itself sits on, which needs no learned toggle.
func transitNetwork(t *testing.T) netip.Prefix {
	t.Helper()
	p, err := netip.ParsePrefix("10.44.40.0/24")
	if err != nil {
		t.Fatalf("ParsePrefix: %v", err)
	}
	return p
}

func TestCandidatesLearnsTheNetworksBehindARouter(t *testing.T) {
	local := []netip.Prefix{transitNetwork(t)}

	got := cidrs(learn.Candidates([]learn.Device{hospitalRouter()}, local))

	want := []string{
		"10.44.10.0/24", "10.44.20.0/24", "10.44.30.0/24",
		"10.44.50.0/24", "10.44.60.0/24", "10.44.70.0/24",
		"10.99.0.0/16",
	}
	if !equalStrings(got, want) {
		t.Errorf("Candidates() = %v, want %v", got, want)
	}
}

func TestCandidatesRejectsWhatMustNeverBeSwept(t *testing.T) {
	cases := []struct {
		name  string
		route learn.Route
	}{
		{"default route", learn.Route{Destination: "0.0.0.0", Prefix: 0, Type: "remote"}},
		{"host route", learn.Route{Destination: "10.10.10.7", Prefix: 32, Type: "remote"}},
		{"point-to-point", learn.Route{Destination: "10.10.10.6", Prefix: 31, Type: "remote"}},
		{"public", learn.Route{Destination: "203.0.113.0", Prefix: 24, Type: "remote"}},
		{"carrier-grade NAT", learn.Route{Destination: "100.64.0.0", Prefix: 24, Type: "remote"}},
		{"loopback", learn.Route{Destination: "127.0.0.0", Prefix: 8, Type: "remote"}},
		{"link-local", learn.Route{Destination: "169.254.0.0", Prefix: 16, Type: "remote"}},
		{"multicast", learn.Route{Destination: "224.0.0.0", Prefix: 4, Type: "remote"}},
		{"wider than the floor", learn.Route{Destination: "10.0.0.0", Prefix: 8, Type: "remote"}},
		{"rejected by the router", learn.Route{Destination: "10.5.0.0", Prefix: 16, Type: "reject"}},
		{"blackholed", learn.Route{Destination: "10.6.0.0", Prefix: 16, Type: "blackhole"}},
		{"unusable mask", learn.Route{Destination: "10.7.0.0", Prefix: 0, Type: "remote"}},
		{"not an address", learn.Route{Destination: "router-1", Prefix: 24, Type: "remote"}},
		{"IPv6", learn.Route{Destination: "2001:db8::", Prefix: 64, Type: "remote"}},
		// A unique-local IPv6 network answers "private" and sits inside the
		// prefix bounds, so only the address family keeps it out of a sweep
		// the scanner could never run.
		{"IPv6 unique-local in range", learn.Route{Destination: "fd00::", Prefix: 24, Type: "remote"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dev := learn.Device{IP: "10.44.40.1", Routes: []learn.Route{tc.route}}
			if got := learn.Candidates([]learn.Device{dev}, nil); len(got) != 0 {
				t.Errorf("Candidates() = %v, want none", cidrs(got))
			}
		})
	}
}

func TestCandidatesLearnsFromTheAddressTable(t *testing.T) {
	dev := learn.Device{
		IP: "10.44.40.1",
		Addresses: []learn.Address{
			{Address: "10.44.40.1", Prefix: 24}, // the transit network — already local
			{Address: "10.44.80.1", Prefix: 24}, // a connected site network
			{Address: "10.44.90.1", Prefix: 0},  // mask not readable: not a /0
			{Address: "127.0.0.1", Prefix: 8},   // loopback
		},
	}

	got := cidrs(learn.Candidates([]learn.Device{dev}, []netip.Prefix{transitNetwork(t)}))

	if want := []string{"10.44.80.0/24"}; !equalStrings(got, want) {
		t.Errorf("Candidates() = %v, want %v", got, want)
	}
}

func TestCandidatesReportsWhichDeviceAndTableNamedTheNetwork(t *testing.T) {
	devices := []learn.Device{
		{IP: "10.44.40.1", Routes: []learn.Route{{Destination: "10.44.10.0", Prefix: 24, Type: "remote"}}},
		{IP: "10.44.40.2", Addresses: []learn.Address{{Address: "10.44.20.1", Prefix: 24}}},
	}

	got := learn.Candidates(devices, nil)

	if len(got) != 2 {
		t.Fatalf("Candidates() returned %d, want 2: %v", len(got), cidrs(got))
	}
	if got[0].Source != learn.SourceRouteTable || got[0].Router != "10.44.40.1" {
		t.Errorf("first = %+v, want source %q from 10.44.40.1", got[0], learn.SourceRouteTable)
	}
	if got[1].Source != learn.SourceAddressTable || got[1].Router != "10.44.40.2" {
		t.Errorf("second = %+v, want source %q from 10.44.40.2", got[1], learn.SourceAddressTable)
	}
}

// A network two routers both know about is one candidate, and the first device
// to name it owns the attribution — otherwise a sweep target would appear once
// per router that has a route to it.
func TestCandidatesDeduplicates(t *testing.T) {
	devices := []learn.Device{
		{IP: "10.44.40.1", Routes: []learn.Route{{Destination: "10.44.10.0", Prefix: 24, Type: "remote"}}},
		{IP: "10.44.40.2", Routes: []learn.Route{{Destination: "10.44.10.0", Prefix: 24, Type: "remote"}}},
		{IP: "10.44.40.3", Addresses: []learn.Address{{Address: "10.44.10.5", Prefix: 24}}},
	}

	got := learn.Candidates(devices, nil)

	if len(got) != 1 || got[0].CIDR != "10.44.10.0/24" || got[0].Router != "10.44.40.1" {
		t.Errorf("Candidates() = %+v, want one 10.44.10.0/24 from 10.44.40.1", got)
	}
}

// A route's destination is not always already masked; 10.44.10.7/24 and
// 10.44.10.0/24 are the same sweep target and must not both be learned.
func TestCandidatesCanonicalisesTheNetworkAddress(t *testing.T) {
	dev := learn.Device{
		IP: "10.44.40.1",
		Routes: []learn.Route{
			{Destination: "10.44.10.7", Prefix: 24, Type: "remote"},
			{Destination: "10.44.10.0", Prefix: 24, Type: "remote"},
		},
	}

	got := cidrs(learn.Candidates([]learn.Device{dev}, nil))

	if want := []string{"10.44.10.0/24"}; !equalStrings(got, want) {
		t.Errorf("Candidates() = %v, want %v", got, want)
	}
}

// A network inside an already-swept one adds nothing.
func TestCandidatesSkipsWhatIsAlreadyCovered(t *testing.T) {
	dev := learn.Device{
		IP: "10.44.40.1",
		Routes: []learn.Route{
			{Destination: "10.44.40.128", Prefix: 25, Type: "remote"},
			{Destination: "10.44.50.0", Prefix: 24, Type: "remote"},
		},
	}

	got := cidrs(learn.Candidates([]learn.Device{dev}, []netip.Prefix{transitNetwork(t)}))

	if want := []string{"10.44.50.0/24"}; !equalStrings(got, want) {
		t.Errorf("Candidates() = %v, want %v", got, want)
	}
}

// A network wider than the one Seed already sweeps carries addresses nothing
// reaches today, so it is a candidate rather than something already covered.
func TestCandidatesLearnsASupernetOfTheSweptNetwork(t *testing.T) {
	dev := learn.Device{
		IP:     "10.44.40.1",
		Routes: []learn.Route{{Destination: "10.44.0.0", Prefix: 16, Type: "remote"}},
	}

	got := cidrs(learn.Candidates([]learn.Device{dev}, []netip.Prefix{transitNetwork(t)}))

	if want := []string{"10.44.0.0/16"}; !equalStrings(got, want) {
		t.Errorf("Candidates() = %v, want %v", got, want)
	}
}
