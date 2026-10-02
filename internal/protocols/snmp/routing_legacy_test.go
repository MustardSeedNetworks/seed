package snmp_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// tableAgent answers a BulkWalk with the PDUs stored under the walked column.
type tableAgent map[string][]gosnmp.SnmpPDU

func (a tableAgent) BulkWalk(root string, walkFn gosnmp.WalkFunc) error {
	for _, pdu := range a[root] {
		if err := walkFn(pdu); err != nil {
			return err
		}
	}
	return nil
}

// ipRouteRow is one ipRouteTable row; an empty mask or next hop leaves that
// column out, as an agent that does not serve it would.
type ipRouteRow struct {
	dest, mask, nextHop string
	routeType           int
}

func ipRouteAgent(rows ...ipRouteRow) tableAgent {
	agent := tableAgent{}
	put := func(column, dest string, typ gosnmp.Asn1BER, value any) {
		agent[column] = append(agent[column], gosnmp.SnmpPDU{Name: "." + column + "." + dest, Type: typ, Value: value})
	}
	for _, row := range rows {
		put(snmp.OIDIpRouteDest, row.dest, gosnmp.IPAddress, row.dest)
		if row.mask != "" {
			put(snmp.OIDIpRouteMask, row.dest, gosnmp.IPAddress, row.mask)
		}
		if row.nextHop != "" {
			put(snmp.OIDIpRouteNextHop, row.dest, gosnmp.IPAddress, row.nextHop)
		}
		put(snmp.OIDIpRouteIfIndex, row.dest, gosnmp.Integer, 7)
		put(snmp.OIDIpRouteType, row.dest, gosnmp.Integer, row.routeType)
		put(snmp.OIDIpRouteProto, row.dest, gosnmp.Integer, 13)
		put(snmp.OIDIpRouteMetric1, row.dest, gosnmp.Integer, 110)
	}
	return agent
}

func TestIPRouteTableBuildsRoutesFromItsColumns(t *testing.T) {
	agent := ipRouteAgent(
		ipRouteRow{"10.20.0.0", "255.255.0.0", "10.0.0.1", 4},
		ipRouteRow{"0.0.0.0", "0.0.0.0", "10.0.0.254", 4},
		ipRouteRow{"10.0.0.0", "255.255.255.0", "10.0.0.2", 3},
	)

	table, err := snmp.ExportWalkIPRouteTable(agent, snmp.MaxRouteRows)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	got := table.Routes
	slices.SortFunc(got, func(a, b snmp.RouteEntry) int { return strings.Compare(a.Destination, b.Destination) })
	want := []snmp.RouteEntry{
		{
			Destination: "0.0.0.0",
			Prefix:      0,
			NextHop:     "10.0.0.254",
			IfIndex:     7,
			Type:        "remote",
			Protocol:    "ospf",
			Metric:      110,
		},
		{
			Destination: "10.0.0.0",
			Prefix:      24,
			NextHop:     "10.0.0.2",
			IfIndex:     7,
			Type:        "local",
			Protocol:    "ospf",
			Metric:      110,
		},
		{
			Destination: "10.20.0.0",
			Prefix:      16,
			NextHop:     "10.0.0.1",
			IfIndex:     7,
			Type:        "remote",
			Protocol:    "ospf",
			Metric:      110,
		},
	}
	if !slices.Equal(got, want) || table.Truncated {
		t.Errorf("routes = %+v (truncated=%v)\nwant %+v", got, table.Truncated, want)
	}
}

func TestIPRouteTableDropsRowsWithoutAUsableRoute(t *testing.T) {
	agent := ipRouteAgent(
		ipRouteRow{"10.1.0.0", "255.255.0.0", "10.0.0.1", 2}, // invalid(2): deleted by the agent
		ipRouteRow{"10.2.0.0", "255.0.255.0", "10.0.0.1", 4}, // not a contiguous mask
		ipRouteRow{"10.3.0.0", "", "10.0.0.1", 4},            // no mask: would read as a default route
		ipRouteRow{"10.4.0.0", "255.255.0.0", "", 4},         // no next hop
		ipRouteRow{"10.5.0.0", "255.255.0.0", "10.0.0.1", 4},
	)

	table, err := snmp.ExportWalkIPRouteTable(agent, snmp.MaxRouteRows)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(table.Routes) != 1 || table.Routes[0].Destination != "10.5.0.0" {
		t.Errorf("routes = %+v, want only 10.5.0.0", table.Routes)
	}
}

func TestGetRoutesReadsTheFirstTableThatHoldsRows(t *testing.T) {
	routes := func(dest string) func() (snmp.RouteTable, error) {
		return func() (snmp.RouteTable, error) {
			return snmp.RouteTable{Routes: []snmp.RouteEntry{{Destination: dest}}}, nil
		}
	}
	empty := func() (snmp.RouteTable, error) { return snmp.RouteTable{}, nil }
	failed := func() (snmp.RouteTable, error) { return snmp.RouteTable{}, errors.New("timeout") }

	tests := []struct {
		name     string
		readers  []func() (snmp.RouteTable, error)
		wantDest string
		wantErr  bool
	}{
		{
			"inetCidr wins",
			[]func() (snmp.RouteTable, error){routes("inet"), routes("cidr"), routes("legacy")},
			"inet",
			false,
		},
		{
			"cidr when inetCidr is empty",
			[]func() (snmp.RouteTable, error){empty, routes("cidr"), routes("legacy")},
			"cidr",
			false,
		},
		{
			"ipRouteTable when both CIDR tables are empty",
			[]func() (snmp.RouteTable, error){empty, empty, routes("legacy")},
			"legacy",
			false,
		},
		{
			"ipRouteTable when both CIDR tables fail",
			[]func() (snmp.RouteTable, error){failed, failed, routes("legacy")},
			"legacy",
			false,
		},
		{"no table holds rows", []func() (snmp.RouteTable, error){empty, empty, empty}, "", false},
		{"the last failure stands", []func() (snmp.RouteTable, error){empty, empty, failed}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, err := snmp.ExportFirstRouteTable(tt.readers...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			var gotDest string
			if len(table.Routes) > 0 {
				gotDest = table.Routes[0].Destination
			}
			if gotDest != tt.wantDest {
				t.Errorf("read %q, want %q", gotDest, tt.wantDest)
			}
		})
	}
}
