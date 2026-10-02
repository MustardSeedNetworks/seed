package api_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/learn"
)

// Every device with an address is viewed: a device no SNMP walk reached still
// shows its own address and its passive LLDP/CDP neighbour, which are sweep
// evidence (seed#2832); one that answered carries its tables whole.
func TestDeviceViewsCarriesTablesAndSkipsDevicesWithoutAnAddress(t *testing.T) {
	devices := []*discovery.DiscoveredDevice{
		nil,
		{
			IP: "10.44.40.9", LLDPInfo: &discovery.LLDPDeviceInfo{ManagementAddress: "10.44.10.2"},
			CDPInfo: &discovery.CDPDeviceInfo{ManagementAddress: "10.44.20.2"},
		},
		{IP: "", SNMPData: &discovery.SNMPFullData{
			Routing: []discovery.SNMPRoute{{Destination: "10.44.10.0", Prefix: 24}},
		}}, // no address to attribute a candidate to
		{IP: "10.44.40.1", SNMPData: &discovery.SNMPFullData{
			Routing: []discovery.SNMPRoute{
				{Destination: "10.44.10.0", Prefix: 24, NextHop: "10.44.40.2", Type: "remote", Protocol: "ospf"},
			},
			IPAddresses:   []discovery.SNMPIPAddress{{Address: "10.44.40.1", Prefix: 24}},
			LLDPNeighbors: []discovery.SNMPLLDPNeighbor{{RemoteMgmtAddr: "10.44.30.2"}},
		}},
	}

	got := api.ExportDeviceViews(devices)

	if len(got) != 2 {
		t.Fatalf("deviceViews() returned %d views, want 2: %+v", len(got), got)
	}
	passive := got[0]
	if passive.IP != "10.44.40.9" || len(passive.Routes) != 0 || len(passive.Addresses) != 0 ||
		!slices.Equal(passive.Neighbours, []string{"10.44.10.2", "10.44.20.2"}) {
		t.Errorf("unprofiled view = %+v, want its address and both neighbours only", passive)
	}
	view := got[1]
	if view.IP != "10.44.40.1" {
		t.Errorf("IP = %q, want 10.44.40.1", view.IP)
	}
	if len(view.Routes) != 1 || view.Routes[0] != (learn.Route{
		Destination: "10.44.10.0", Prefix: 24, NextHop: "10.44.40.2", Type: "remote", Protocol: "ospf",
	}) {
		t.Errorf("Routes = %+v, want the route row carried whole", view.Routes)
	}
	if len(view.Addresses) != 1 || view.Addresses[0].Address != "10.44.40.1" ||
		view.Addresses[0].Prefix != 24 {
		t.Errorf("Addresses = %+v, want the address row carried whole", view.Addresses)
	}
	if !slices.Equal(view.Neighbours, []string{"10.44.30.2"}) {
		t.Errorf("Neighbours = %v, want the SNMP LLDP management address", view.Neighbours)
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
