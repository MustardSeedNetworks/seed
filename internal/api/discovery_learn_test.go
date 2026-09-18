package api_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
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
