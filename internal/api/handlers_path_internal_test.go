package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// TestTracerouteResultCarriesHopRoutes pins the wire shape seed#2587 adds: a
// hop a discovered router answers on carries that router's route toward the
// target, and a hop no device owns carries no route key at all.
func TestTracerouteResultCarriesHopRoutes(t *testing.T) {
	t.Parallel()

	router := &discovery.DiscoveredDevice{
		IP: "10.255.0.1",
		SNMPData: &discovery.SNMPFullData{
			Interfaces:  []discovery.SNMPInterface{{Index: 2, Name: "Gi0/2"}},
			IPAddresses: []discovery.SNMPIPAddress{{Address: "10.0.12.1", Prefix: 30, IfIndex: 2}},
			Routing: []discovery.SNMPRoute{
				{
					Destination: "10.20.0.0",
					Prefix:      16,
					NextHop:     "10.0.23.3",
					IfIndex:     2,
					Type:        "remote",
					Protocol:    "ospf",
				},
			},
		},
	}
	trace := &discovery.TracerouteResult{
		Target:   "server.example.test",
		TargetIP: "10.20.5.9",
		Protocol: "icmp",
		Hops: []discovery.TracerouteHop{
			{TTL: 1, IP: "192.168.1.1", State: "reply"},
			{TTL: 2, IP: "10.0.12.1", State: "reply"},
			{TTL: 3, State: "timeout"},
		},
	}

	got := toTracerouteResult(trace, []*discovery.DiscoveredDevice{router})

	if got.Hops[0].Route != nil || got.Hops[2].Route != nil {
		t.Fatalf("hops no device owns carry a route: %+v, %+v", got.Hops[0].Route, got.Hops[2].Route)
	}
	want := HopRoute{
		Device: "10.255.0.1", Destination: "10.20.0.0", Prefix: 16, NextHop: "10.0.23.3",
		IfIndex: 2, Interface: "Gi0/2", Type: "remote", Protocol: "ospf",
	}
	if got.Hops[1].Route == nil || *got.Hops[1].Route != want {
		t.Fatalf("hop 2 route = %+v, want %+v", got.Hops[1].Route, want)
	}

	body, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	wire := string(body)
	const wantRoute = `"route":{"device":"10.255.0.1","destination":"10.20.0.0","prefix":16,` +
		`"nextHop":"10.0.23.3","ifIndex":2,"interface":"Gi0/2","type":"remote","protocol":"ospf"}`
	if !strings.Contains(wire, wantRoute) {
		t.Fatalf("wire shape missing %s in %s", wantRoute, wire)
	}
	if n := strings.Count(wire, `"route"`); n != 1 {
		t.Fatalf("%d route keys on the wire, want 1: %s", n, wire)
	}
}

// TestTracerouteResultWithoutDevices keeps the path view working before any
// discovery has run: no devices, no routes, every hop still mapped.
func TestTracerouteResultWithoutDevices(t *testing.T) {
	t.Parallel()

	trace := &discovery.TracerouteResult{
		TargetIP: "10.20.5.9",
		Hops:     []discovery.TracerouteHop{{TTL: 1, IP: "10.0.12.1", State: "reply"}},
	}
	got := toTracerouteResult(trace, nil)
	if len(got.Hops) != 1 || got.Hops[0].Route != nil {
		t.Fatalf("hops = %+v, want one hop without a route", got.Hops)
	}
	if toTracerouteResult(nil, nil) != nil {
		t.Fatal("nil trace mapped to a non-nil result")
	}
}
