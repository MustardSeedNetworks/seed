//go:build linux

package gateway_test

import (
	"net"
	"testing"

	"github.com/vishvananda/netlink"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
)

// Netlink carries the mask with the destination; a RouteInfo that kept only
// the address could not tell a /24 from the host route to its own network
// address (seed#2765). A nil Dst is the default route, which reads as
// 0.0.0.0/0 like every other row rather than as the word "default".
func TestAppendRouteInfoCarriesTheMask(t *testing.T) {
	routes := []netlink.Route{
		{Dst: nil, Gw: net.IPv4(10, 44, 20, 1)},
		{Dst: &net.IPNet{IP: net.IPv4(10, 44, 20, 0), Mask: net.CIDRMask(24, 32)}},
		{Dst: &net.IPNet{IP: net.IPv4(10, 44, 20, 0), Mask: net.CIDRMask(32, 32)}},
		{Dst: &net.IPNet{IP: net.IPv4(10, 51, 0, 0), Mask: net.CIDRMask(16, 32)}, Gw: net.IPv4(10, 254, 200, 1)},
	}

	got := gateway.AppendRouteInfo(nil, routes, "inet")

	want := []gateway.RouteInfo{
		{Destination: "0.0.0.0", Prefix: 0, Gateway: "10.44.20.1", Family: "inet"},
		{Destination: "10.44.20.0", Prefix: 24, Family: "inet"},
		{Destination: "10.44.20.0", Prefix: 32, Family: "inet"},
		{Destination: "10.51.0.0", Prefix: 16, Gateway: "10.254.200.1", Family: "inet"},
	}
	if len(got) != len(want) {
		t.Fatalf("appendRouteInfo() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("route %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// The IPv6 default route is ::/0, not 0.0.0.0/0.
func TestAppendRouteInfoIPv6DefaultRoute(t *testing.T) {
	got := gateway.AppendRouteInfo(nil, []netlink.Route{{Dst: nil}}, "inet6")

	if len(got) != 1 || got[0].Destination != "::" || got[0].Prefix != 0 {
		t.Errorf("appendRouteInfo() = %+v, want ::/0", got)
	}
}
