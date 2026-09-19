package gateway

import "testing"

// fakeNamer resolves the interface columns the way Windows would, without a
// Windows host: the IPv4 table names an interface by its local address and
// the IPv6 table by its index.
type fakeNamer struct {
	byAddress map[string]string
	byIndex   map[int]string
}

func (f fakeNamer) nameByAddress(addr string) string { return f.byAddress[addr] }
func (f fakeNamer) nameByIndex(index int) string     { return f.byIndex[index] }

func testNamer() fakeNamer {
	return fakeNamer{
		byAddress: map[string]string{"10.44.20.157": "Ethernet 2"},
		byIndex:   map[int]string{12: "Ethernet 2"},
	}
}

// TestParseRouteFieldsReadsTheNetmaskColumn is #2765's first half: route
// print's netmask column was dropped, so a /24 and the host route to its
// network address parsed to the same RouteInfo.
func TestParseRouteFieldsReadsTheNetmaskColumn(t *testing.T) {
	network := []string{"10.44.20.0", "255.255.255.0", "10.44.20.1", "10.44.20.157", "25"}
	host := []string{"10.44.20.0", "255.255.255.255", "10.44.20.1", "10.44.20.157", "25"}

	gotNetwork, ok := parseRouteFields(network, "inet", testNamer())
	if !ok {
		t.Fatal("network route not parsed")
	}
	gotHost, ok := parseRouteFields(host, "inet", testNamer())
	if !ok {
		t.Fatal("host route not parsed")
	}

	if gotNetwork.Prefix != 24 {
		t.Errorf("network prefix = %d, want 24", gotNetwork.Prefix)
	}
	if gotHost.Prefix != 32 {
		t.Errorf("host prefix = %d, want 32", gotHost.Prefix)
	}
	if gotNetwork == gotHost {
		t.Errorf("a /24 and a /32 to the same address parsed identically: %+v", gotNetwork)
	}
}

// TestParseRouteFieldsNamesTheInterface is #2765's second half: the IPv4
// Interface column is the interface's local ADDRESS, and netif.rollback
// matches it against a name, so defaultGatewayFor could never match on
// Windows.
func TestParseRouteFieldsNamesTheInterface(t *testing.T) {
	fields := []string{"0.0.0.0", "0.0.0.0", "10.44.20.1", "10.44.20.157", "25"}

	got, ok := parseRouteFields(fields, "inet", testNamer())
	if !ok {
		t.Fatal("default route not parsed")
	}
	if got.Interface != "Ethernet 2" {
		t.Errorf("Interface = %q, want the name %q", got.Interface, "Ethernet 2")
	}
	if got.Destination != "0.0.0.0" || got.Prefix != 0 {
		t.Errorf("default route = %s/%d, want 0.0.0.0/0", got.Destination, got.Prefix)
	}
}

// TestParseRouteFieldsDropsOnLink: route print writes "On-link" in the gateway
// column of a connected route. It is not an address, and once Interface
// carries a name defaultGatewayFor would otherwise hand it back as a gateway
// to restore.
func TestParseRouteFieldsDropsOnLink(t *testing.T) {
	fields := []string{"10.44.20.0", "255.255.255.0", "On-link", "10.44.20.157", "281"}

	got, ok := parseRouteFields(fields, "inet", testNamer())
	if !ok {
		t.Fatal("connected route not parsed")
	}
	if got.Gateway != "" {
		t.Errorf("Gateway = %q, want empty for an on-link route", got.Gateway)
	}
}

// TestParseRouteFieldsIPv6Columns: `route print -6` writes
// "If Metric Network Destination Gateway", so the destination is the third
// column, not the first — the first is the interface index.
func TestParseRouteFieldsIPv6Columns(t *testing.T) {
	fields := []string{"12", "281", "2001:db8:44:20::/64", "fe80::1"}

	got, ok := parseRouteFields(fields, "inet6", testNamer())
	if !ok {
		t.Fatal("IPv6 route not parsed")
	}
	if got.Destination != "2001:db8:44:20::" || got.Prefix != 64 {
		t.Errorf("destination = %s/%d, want 2001:db8:44:20::/64", got.Destination, got.Prefix)
	}
	if got.Interface != "Ethernet 2" {
		t.Errorf("Interface = %q, want %q resolved from index 12", got.Interface, "Ethernet 2")
	}
	if got.Gateway != "fe80::1" {
		t.Errorf("Gateway = %q, want fe80::1", got.Gateway)
	}
}

// TestParseRouteOutputWholeTable drives the section handling over real
// `route print` shapes: the header rows, the separator rules and the
// Persistent Routes section must all stay out of the result.
func TestParseRouteOutputWholeTable(t *testing.T) {
	const v4 = `===========================================================================
Active Routes:
Network Destination        Netmask          Gateway       Interface  Metric
          0.0.0.0          0.0.0.0      10.44.20.1    10.44.20.157     25
       10.44.20.0    255.255.255.0         On-link     10.44.20.157    281
     10.44.20.157  255.255.255.255         On-link     10.44.20.157    281
        10.51.0.0      255.255.0.0   10.254.200.1    10.254.200.250     26
===========================================================================
Persistent Routes:
  None
`

	got := parseRouteOutput(v4, "inet", testNamer())

	want := []RouteInfo{
		{Destination: "0.0.0.0", Prefix: 0, Gateway: "10.44.20.1", Interface: "Ethernet 2", Family: "inet"},
		{Destination: "10.44.20.0", Prefix: 24, Interface: "Ethernet 2", Family: "inet"},
		{Destination: "10.44.20.157", Prefix: 32, Interface: "Ethernet 2", Family: "inet"},
		{Destination: "10.51.0.0", Prefix: 16, Gateway: "10.254.200.1", Family: "inet"},
	}
	if len(got) != len(want) {
		t.Fatalf("parsed %d routes, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("route %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestParseRouteOutputIPv6Table: the -6 table's own layout, including the
// "If Metric" header that the -4 reader would have taken for a route.
func TestParseRouteOutputIPv6Table(t *testing.T) {
	const v6 = `===========================================================================
Active Routes:
 If Metric Network Destination      Gateway
 12    281 ::/0                     fe80::1
  1    331 ::1/128                  On-link
 12    281 2001:db8:44:20::/64      On-link
===========================================================================
`

	got := parseRouteOutput(v6, "inet6", testNamer())

	want := []RouteInfo{
		{Destination: "::", Prefix: 0, Gateway: "fe80::1", Interface: "Ethernet 2", Family: "inet6"},
		{Destination: "::1", Prefix: 128, Family: "inet6"},
		{Destination: "2001:db8:44:20::", Prefix: 64, Interface: "Ethernet 2", Family: "inet6"},
	}
	if len(got) != len(want) {
		t.Fatalf("parsed %d routes, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("route %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// A netmask whose ones are not contiguous is not a prefix length at all.
// Counting its bits anyway would turn 255.0.255.0 into a /0, i.e. into a
// default route, so the row is dropped instead.
func TestParseRouteFieldsRejectsANonContiguousMask(t *testing.T) {
	fields := []string{"10.44.20.0", "255.0.255.0", "10.44.20.1", "10.44.20.157", "25"}

	if got, ok := parseRouteFields(fields, "inet", testNamer()); ok {
		t.Errorf("parsed %+v, want the row dropped", got)
	}
}
