package snmp_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// fakeRouteAgent serves a route table of rows routes, every column in index
// order the way an agent answers a BulkWalk, and counts how many rows of each
// column it had to hand over before the walker stopped it.
type fakeRouteAgent struct {
	rows      int
	index     func(row int) string
	delivered map[string]int
}

func (a *fakeRouteAgent) BulkWalk(root string, walkFn gosnmp.WalkFunc) error {
	for row := range a.rows {
		a.delivered[root]++
		pdu := gosnmp.SnmpPDU{Name: "." + root + "." + a.index(row), Type: gosnmp.Integer, Value: 4}
		if err := walkFn(pdu); err != nil {
			return err
		}
	}
	return nil
}

// Each row is a distinct 10.x.y.0/24, so no two rows collapse onto one key.
func inetCidrIndex(row int) string {
	return fmt.Sprintf("1.4.10.%d.%d.0.24.0.0.1.4.10.254.200.1", row/256, row%256)
}

func ipCidrIndex(row int) string {
	return fmt.Sprintf("10.%d.%d.0.255.255.255.0.0.10.254.200.1", row/256, row%256)
}

type routeTableWalk struct {
	name    string
	index   func(int) string
	columns []string
	walk    func(snmp.BulkWalker, int) (snmp.RouteTable, error)
}

type routeTableSize struct {
	name          string
	rows          int
	wantRoutes    int
	wantTruncated bool
}

func TestRouteTableWalksStopAtTheRowCap(t *testing.T) {
	walks := []routeTableWalk{
		{
			name:  "inetCidrRouteTable",
			index: inetCidrIndex,
			columns: []string{
				snmp.OIDInetCidrRouteIfIndex, snmp.OIDInetCidrRouteType,
				snmp.OIDInetCidrRouteProto, snmp.OIDInetCidrRouteMetric1,
			},
			walk: snmp.ExportWalkInetCidrRouteTable,
		},
		{
			name:  "ipCidrRouteTable",
			index: ipCidrIndex,
			columns: []string{
				snmp.OIDIpCidrRouteDest, snmp.OIDIpCidrRouteIfIndex, snmp.OIDIpCidrRouteType,
				snmp.OIDIpCidrRouteProto, snmp.OIDIpCidrRouteMetric1,
			},
			walk: snmp.ExportWalkIPCidrRouteTable,
		},
	}
	sizes := []routeTableSize{
		{"past the cap", snmp.MaxRouteRows + 500, snmp.MaxRouteRows, true},
		{"exactly the cap", snmp.MaxRouteRows, snmp.MaxRouteRows, false},
		{"under the cap", 3, 3, false},
	}

	for _, walk := range walks {
		for _, size := range sizes {
			t.Run(walk.name+"/"+size.name, func(t *testing.T) {
				assertCappedWalk(t, walk, size)
			})
		}
	}
}

func assertCappedWalk(t *testing.T, walk routeTableWalk, size routeTableSize) {
	t.Helper()
	agent := &fakeRouteAgent{rows: size.rows, index: walk.index, delivered: map[string]int{}}

	table, err := walk.walk(agent, snmp.MaxRouteRows)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(table.Routes) != size.wantRoutes || table.Truncated != size.wantTruncated {
		t.Fatalf("got %d routes, truncated=%v; want %d, truncated=%v",
			len(table.Routes), table.Truncated, size.wantRoutes, size.wantTruncated)
	}
	// Stopping means the agent was asked for no row past the one that
	// proved the table is longer, in every column.
	for _, column := range walk.columns {
		if got, most := agent.delivered[column], min(size.rows, snmp.MaxRouteRows+1); got != most {
			t.Errorf("column %s read %d rows, want %d", column, got, most)
		}
	}
	for _, route := range table.Routes {
		if route.Type != "remote" {
			t.Fatalf("route %s/%d kept without its type column: %+v",
				route.Destination, route.Prefix, route)
		}
	}
}

func TestRouteTableWalkReportsAnAgentFailure(t *testing.T) {
	failing := failingWalker{err: errors.New("request timeout")}
	for name, walk := range map[string]func(snmp.BulkWalker, int) (snmp.RouteTable, error){
		"inetCidrRouteTable": snmp.ExportWalkInetCidrRouteTable,
		"ipCidrRouteTable":   snmp.ExportWalkIPCidrRouteTable,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := walk(failing, snmp.MaxRouteRows)
			if err == nil || !strings.Contains(err.Error(), "request timeout") {
				t.Fatalf("err = %v, want the agent's failure", err)
			}
		})
	}
}

type failingWalker struct{ err error }

func (w failingWalker) BulkWalk(string, gosnmp.WalkFunc) error { return w.err }
