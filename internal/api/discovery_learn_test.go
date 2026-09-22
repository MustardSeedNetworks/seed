package api_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/learn"
)

// The learner reads two SNMP tables off each discovered device. A device no
// SNMP walk reached carries neither, and contributing it would put an empty
// routing view in front of the rules for every host on the subnet.
func TestRoutingViewsCarriesBothTablesAndSkipsDevicesWithout(t *testing.T) {
	devices := []*discovery.DiscoveredDevice{
		nil,
		{IP: "10.44.40.9"}, // profiled, no SNMP
		{IP: "10.44.40.8", SNMPData: &discovery.SNMPFullData{ // answered, nothing routed
			System: nil,
		}},
		{IP: "", SNMPData: &discovery.SNMPFullData{
			Routing: []discovery.SNMPRoute{{Destination: "10.44.10.0", Prefix: 24}},
		}}, // no address to attribute a candidate to
		{IP: "10.44.40.1", SNMPData: &discovery.SNMPFullData{
			Routing: []discovery.SNMPRoute{
				{Destination: "10.44.10.0", Prefix: 24, Type: "remote", Protocol: "ospf"},
			},
			IPAddresses: []discovery.SNMPIPAddress{{Address: "10.44.40.1", Prefix: 24}},
		}},
	}

	got := api.ExportRoutingViews(devices)

	if len(got) != 1 {
		t.Fatalf("routingViews() returned %d views, want 1: %+v", len(got), got)
	}
	view := got[0]
	if view.IP != "10.44.40.1" {
		t.Errorf("IP = %q, want 10.44.40.1", view.IP)
	}
	if len(view.Routes) != 1 || view.Routes[0].Destination != "10.44.10.0" ||
		view.Routes[0].Prefix != 24 || view.Routes[0].Type != "remote" ||
		view.Routes[0].Protocol != "ospf" {
		t.Errorf("Routes = %+v, want the route row carried whole", view.Routes)
	}
	if len(view.Addresses) != 1 || view.Addresses[0].Address != "10.44.40.1" ||
		view.Addresses[0].Prefix != 24 {
		t.Errorf("Addresses = %+v, want the address row carried whole", view.Addresses)
	}
}

func TestLocalPrefixesMasksTheScannersSubnet(t *testing.T) {
	cases := []struct {
		name   string
		subnet string
		want   string
	}{
		{"masked already", "10.44.40.0/24", "10.44.40.0/24"},
		{"an address inside it", "10.44.40.17/24", "10.44.40.0/24"},
		{"no subnet yet", "", ""},
		{"not a prefix", "10.44.40.17", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := api.ExportLocalPrefixes(tc.subnet)
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("localPrefixes(%q) = %v, want none", tc.subnet, got)
				}
				return
			}
			if len(got) != 1 || got[0].String() != tc.want {
				t.Errorf("localPrefixes(%q) = %v, want [%s]", tc.subnet, got, tc.want)
			}
		})
	}
}

// hostRoutesVia narrows Seed's own forwarding table to what leaves through
// the interface discovery is bound to. The rows below are this Mac's real
// table (feth0 carrying two routed /16s over a transit link), which is the
// shape seed#2695 was written from.
func TestHostRoutesViaNarrowsToTheActiveInterface(t *testing.T) {
	read := func() ([]gateway.RouteInfo, error) {
		return []gateway.RouteInfo{
			{Destination: "0.0.0.0", Prefix: 0, Gateway: "10.44.20.1", Interface: "en0", Family: "inet"},
			{Destination: "10.51.0.0", Prefix: 16, Gateway: "10.254.200.1", Interface: "feth0", Family: "inet"},
			{Destination: "10.52.0.0", Prefix: 16, Gateway: "10.254.200.1", Interface: "feth0", Family: "inet"},
			{Destination: "10.254.200.0", Prefix: 24, Interface: "feth0", Family: "inet"},
			{Destination: "10.44.20.0", Prefix: 24, Interface: "en0", Family: "inet"},
			{Destination: "fd00::", Prefix: 64, Interface: "feth0", Family: "inet6"},
		}, nil
	}

	got := api.ExportHostRoutesVia("feth0", read)

	want := []learn.HostRoute{
		{Destination: "10.51.0.0", Prefix: 16, Gateway: "10.254.200.1"},
		{Destination: "10.52.0.0", Prefix: 16, Gateway: "10.254.200.1"},
		{Destination: "10.254.200.0", Prefix: 24},
	}
	if len(got) != len(want) {
		t.Fatalf("hostRoutesVia() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("route %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// Two ways the host's table says nothing, neither of which is an error: no
// interface bound yet, and a routing table that cannot be read.
func TestHostRoutesViaIsEmptyWithoutAnInterfaceOrATable(t *testing.T) {
	read := func() ([]gateway.RouteInfo, error) {
		return []gateway.RouteInfo{
			{Destination: "10.51.0.0", Prefix: 16, Interface: "feth0", Family: "inet"},
		}, nil
	}
	if got := api.ExportHostRoutesVia("", read); got != nil {
		t.Errorf("hostRoutesVia(\"\") = %+v, want none", got)
	}

	failing := func() ([]gateway.RouteInfo, error) { return nil, errNoRoutingTable }
	if got := api.ExportHostRoutesVia("feth0", failing); got != nil {
		t.Errorf("hostRoutesVia with an unreadable table = %+v, want none", got)
	}
}

var errNoRoutingTable = errors.New("routing table unavailable")
