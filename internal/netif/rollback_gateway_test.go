package netif_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/netif"
)

// A rollback restores the interface's DEFAULT gateway. The connected subnet is
// listed first in every platform's table, so selecting "the first route out of
// this interface that names a gateway" picks the wrong row — and on Windows,
// where the connected row's gateway column reads "On-link", it picks a string
// that is not an address at all (seed#2765).
func TestDefaultGatewayInPicksTheDefaultRouteNotTheFirstOne(t *testing.T) {
	routes := []gateway.RouteInfo{
		{Destination: "10.44.20.0", Prefix: 24, Gateway: "10.44.20.9", Interface: "en0", Family: "inet"},
		{Destination: "0.0.0.0", Prefix: 0, Gateway: "10.44.20.1", Interface: "en0", Family: "inet"},
		{Destination: "0.0.0.0", Prefix: 0, Gateway: "10.254.200.1", Interface: "feth0", Family: "inet"},
	}

	if got := netif.DefaultGatewayIn(routes, "en0"); got != "10.44.20.1" {
		t.Errorf("DefaultGatewayIn(en0) = %q, want 10.44.20.1", got)
	}
	if got := netif.DefaultGatewayIn(routes, "feth0"); got != "10.254.200.1" {
		t.Errorf("DefaultGatewayIn(feth0) = %q, want 10.254.200.1", got)
	}
}

// An interface can hold an address and no default route; that is not an error
// and must not borrow another interface's gateway.
func TestDefaultGatewayInIsEmptyWithoutADefaultRoute(t *testing.T) {
	routes := []gateway.RouteInfo{
		{Destination: "10.44.30.0", Prefix: 24, Interface: "en1", Family: "inet"},
		{Destination: "0.0.0.0", Prefix: 0, Gateway: "10.44.20.1", Interface: "en0", Family: "inet"},
	}

	if got := netif.DefaultGatewayIn(routes, "en1"); got != "" {
		t.Errorf("DefaultGatewayIn(en1) = %q, want empty", got)
	}
}
