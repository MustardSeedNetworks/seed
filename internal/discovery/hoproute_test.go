package discovery_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// coreRouter is a discovered router found at its loopback, answering
// traceroute from 10.0.12.1, with a default, a summary and a more specific
// route toward the server network.
func coreRouter() *discovery.DiscoveredDevice {
	return &discovery.DiscoveredDevice{
		IP: "10.255.0.1",
		SNMPData: &discovery.SNMPFullData{
			Interfaces: []discovery.SNMPInterface{
				{Index: 1, Name: "Gi0/1"},
				{Index: 2, Description: "GigabitEthernet0/2"},
			},
			IPAddresses: []discovery.SNMPIPAddress{
				{Address: "10.255.0.1", Prefix: 32},
				{Address: "10.0.12.1", Prefix: 30, IfIndex: 1},
			},
			Routing: []discovery.SNMPRoute{
				{
					Destination: "0.0.0.0",
					Prefix:      0,
					NextHop:     "10.0.12.2",
					IfIndex:     1,
					Type:        "remote",
					Protocol:    "static",
				},
				{
					Destination: "10.20.0.0",
					Prefix:      16,
					NextHop:     "10.0.23.3",
					IfIndex:     2,
					Type:        "remote",
					Protocol:    "ospf",
				},
				{
					Destination: "10.20.5.0",
					Prefix:      24,
					NextHop:     "10.0.23.4",
					IfIndex:     2,
					Type:        "remote",
					Protocol:    "bgp",
				},
				{
					Destination: "2001:db8::",
					Prefix:      32,
					NextHop:     "fe80::1",
					IfIndex:     1,
					Type:        "remote",
					Protocol:    "ospf",
				},
				{Destination: "10.30.0.0", Prefix: 16, IfIndex: 9, Type: "reject", Protocol: "static"},
			},
		},
	}
}

func TestRouteForHop(t *testing.T) {
	t.Parallel()

	truncated := coreRouter()
	truncated.SNMPData.RoutingTruncated = true
	noTable := coreRouter()
	noTable.SNMPData.Routing = nil
	noSNMP := &discovery.DiscoveredDevice{IP: "10.0.12.1"}

	tests := []struct {
		name    string
		devices []*discovery.DiscoveredDevice
		hop     string
		target  string
		want    *discovery.HopRoute
	}{
		{
			name:    "most specific route wins over the summary and the default",
			devices: []*discovery.DiscoveredDevice{coreRouter()},
			hop:     "10.255.0.1",
			target:  "10.20.5.9",
			want: &discovery.HopRoute{
				Device: "10.255.0.1", Destination: "10.20.5.0", Prefix: 24, NextHop: "10.0.23.4",
				IfIndex: 2, Interface: "GigabitEthernet0/2", Type: "remote", Protocol: "bgp",
			},
		},
		{
			name:    "hop answers from an interface address the walk listed",
			devices: []*discovery.DiscoveredDevice{noSNMP, coreRouter()},
			hop:     "10.0.12.1",
			target:  "10.20.9.9",
			want: &discovery.HopRoute{
				Device: "10.255.0.1", Destination: "10.20.0.0", Prefix: 16, NextHop: "10.0.23.3",
				IfIndex: 2, Interface: "GigabitEthernet0/2", Type: "remote", Protocol: "ospf",
			},
		},
		{
			name:    "default route covers an off-table target",
			devices: []*discovery.DiscoveredDevice{coreRouter()},
			hop:     "10.255.0.1",
			target:  "198.51.100.7",
			want: &discovery.HopRoute{
				Device: "10.255.0.1", Destination: "0.0.0.0", Prefix: 0, NextHop: "10.0.12.2",
				IfIndex: 1, Interface: "Gi0/1", Type: "remote", Protocol: "static",
			},
		},
		{
			name:    "a reject route is reported, and an unwalked ifIndex has no name",
			devices: []*discovery.DiscoveredDevice{coreRouter()},
			hop:     "10.255.0.1",
			target:  "10.30.1.1",
			want: &discovery.HopRoute{
				Device: "10.255.0.1", Destination: "10.30.0.0", Prefix: 16,
				IfIndex: 9, Type: "reject", Protocol: "static",
			},
		},
		{
			name:    "IPv6 target matches only the IPv6 route",
			devices: []*discovery.DiscoveredDevice{coreRouter()},
			hop:     "10.255.0.1",
			target:  "2001:db8:1::5",
			want: &discovery.HopRoute{
				Device: "10.255.0.1", Destination: "2001:db8::", Prefix: 32, NextHop: "fe80::1",
				IfIndex: 1, Interface: "Gi0/1", Type: "remote", Protocol: "ospf",
			},
		},
		{
			name:    "a truncated table says so",
			devices: []*discovery.DiscoveredDevice{truncated},
			hop:     "10.255.0.1",
			target:  "10.20.5.9",
			want: &discovery.HopRoute{
				Device: "10.255.0.1", Destination: "10.20.5.0", Prefix: 24, NextHop: "10.0.23.4",
				IfIndex: 2, Interface: "GigabitEthernet0/2", Type: "remote", Protocol: "bgp",
				TableTruncated: true,
			},
		},
		{
			name:    "a device whose table was not read does not shadow one whose was",
			devices: []*discovery.DiscoveredDevice{noSNMP, noTable, coreRouter()},
			hop:     "10.0.12.1",
			target:  "10.20.5.9",
			want: &discovery.HopRoute{
				Device: "10.255.0.1", Destination: "10.20.5.0", Prefix: 24, NextHop: "10.0.23.4",
				IfIndex: 2, Interface: "GigabitEthernet0/2", Type: "remote", Protocol: "bgp",
			},
		},
		{
			name:    "no discovered device owns the hop",
			devices: []*discovery.DiscoveredDevice{coreRouter()},
			hop:     "10.9.9.9",
			target:  "10.20.5.9",
		},
		{
			name:    "the owning device's table was not read",
			devices: []*discovery.DiscoveredDevice{noTable, noSNMP},
			hop:     "10.0.12.1",
			target:  "10.20.5.9",
		},
		{
			name:    "a hop that did not answer has no address",
			devices: []*discovery.DiscoveredDevice{coreRouter()},
			target:  "10.20.5.9",
		},
		{
			name:    "an unresolved target matches nothing",
			devices: []*discovery.DiscoveredDevice{coreRouter()},
			hop:     "10.255.0.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := discovery.RouteForHop(tt.devices, tt.hop, tt.target)
			switch {
			case tt.want == nil && got != nil:
				t.Fatalf("RouteForHop = %+v, want nil", *got)
			case tt.want != nil && got == nil:
				t.Fatalf("RouteForHop = nil, want %+v", *tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Fatalf("RouteForHop = %+v, want %+v", *got, *tt.want)
			}
		})
	}
}
