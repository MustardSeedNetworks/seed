package routing_test

import (
	"context"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/routing"
)

// The NIAC replay contract, pinned against OIDs NIAC actually emits.
//
// NIAC is the second consumer's test fixture: seed is pointed at a replayed
// network and its findings compared against the authored truth. That only
// works if the two agree on the wire, and for a while they did not — NIAC
// served ipRouteTable (1.3.6.1.2.1.4.21) and RFC 4292's inetCidrRouteTable
// (…4.24.7), while this collector walks RFC 2096's ipCidrRouteTable
// (…4.24.4). A replayed router therefore reported no routes at all, and
// nothing here could see it, because every test in this file builds its OIDs
// with the same helper the parser is checked against.
//
// So these strings are copied verbatim from a NIAC agent's MIB rather than
// generated. If either side changes its index, this fails — which is the
// whole point.
//
// The fixture is one authored router: a static default via 10.254.200.30 and
// the connected 10.254.200.0/27, which has no next hop.
const (
	niacDefaultRoute   = ".0.0.0.0.0.0.0.0.0.10.254.200.30"
	niacConnectedRoute = ".10.254.200.0.255.255.255.224.0.0.0.0.0"
)

func niacOID(column, index string) string {
	return "1.3.6.1.2.1.4.24.4.1." + column + index
}

func TestCollectParsesNIACEmittedRouteOIDs(t *testing.T) {
	t.Parallel()

	fc := &fakeClient{vbs: []snmp.Varbind{
		{OID: niacOID("5", niacDefaultRoute), Value: uint32(10048)},
		{OID: niacOID("6", niacDefaultRoute), Value: routing.TypeRemote},
		{OID: niacOID("7", niacDefaultRoute), Value: routing.ProtoNetmgmt},
		{OID: niacOID("8", niacDefaultRoute), Value: uint32(0)},
		{OID: niacOID("11", niacDefaultRoute), Value: 1},

		{OID: niacOID("5", niacConnectedRoute), Value: uint32(10048)},
		{OID: niacOID("6", niacConnectedRoute), Value: routing.TypeLocal},
		{OID: niacOID("7", niacConnectedRoute), Value: routing.ProtoLocal},
		{OID: niacOID("8", niacConnectedRoute), Value: uint32(0)},
		{OID: niacOID("11", niacConnectedRoute), Value: 1},
	}}
	pub := &fakePublisher{}
	c := routing.New(factoryFor(fc), pub, at)
	if err := c.Collect(context.Background(), snmp.Target{ID: "t-niac"}, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 {
		t.Fatalf("published %d observations, want 1", len(pub.got))
	}
	routes := pub.got[0].Routes
	if len(routes) != 2 {
		t.Fatalf("parsed %d routes from NIAC's OIDs, want 2: %+v", len(routes), routes)
	}

	// The next hop is the field path analysis needs, and the one an index the
	// parser mis-splits would silently drop.
	byDest := map[string]routing.Route{}
	for _, route := range routes {
		byDest[route.Destination] = route
	}
	for dest, want := range map[string]struct{ mask, nextHop string }{
		"0.0.0.0":      {mask: "0.0.0.0", nextHop: "10.254.200.30"},
		"10.254.200.0": {mask: "255.255.255.224", nextHop: "0.0.0.0"},
	} {
		got, ok := byDest[dest]
		if !ok {
			t.Errorf("no route for %s; got %+v", dest, byDest)
			continue
		}
		if got.Mask != want.mask {
			t.Errorf("%s mask = %q, want %q", dest, got.Mask, want.mask)
		}
		if got.NextHop != want.nextHop {
			t.Errorf("%s next hop = %q, want %q", dest, got.NextHop, want.nextHop)
		}
	}
}
