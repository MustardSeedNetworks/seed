package topology_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/topology"
)

type fakeReader struct {
	nodes      []*topology.Node
	interfaces []*topology.Interface
	links      []*topology.Link
	ports      []topology.PortUtilization
	listErr    error
	portsErr   error
}

func (f fakeReader) List(context.Context, topology.ListOptions) ([]*topology.Node, error) {
	return f.nodes, f.listErr
}

func (f fakeReader) ListInterfaces(context.Context, string) ([]*topology.Interface, error) {
	return f.interfaces, nil
}

func (f fakeReader) ListLinks(context.Context, string) ([]*topology.Link, error) {
	return f.links, nil
}

func (f fakeReader) ListARPBindings(
	context.Context,
	topology.ARPListOptions,
) ([]*topology.ARPBinding, error) {
	return nil, nil
}

func (f fakeReader) PortUtilization(context.Context) ([]topology.PortUtilization, error) {
	return f.ports, f.portsErr
}

func TestNodeReturnsDetailForMatch(t *testing.T) {
	r := fakeReader{
		nodes:      []*topology.Node{{ID: "a"}, {ID: "b"}},
		interfaces: []*topology.Interface{{IfName: "eth0"}},
		links:      []*topology.Link{{LinkType: "lldp"}},
	}
	q := topology.NewQueries(r, 1000)

	detail, err := q.Node(context.Background(), "b")
	if err != nil {
		t.Fatalf("Node: %v", err)
	}
	if detail.Node == nil || detail.Node.ID != "b" {
		t.Errorf("wrong node: %+v", detail.Node)
	}
	if len(detail.Interfaces) != 1 || len(detail.Links) != 1 {
		t.Errorf("interfaces/links not loaded: %+v", detail)
	}
}

func TestNodeUnknownReturnsNotFound(t *testing.T) {
	q := topology.NewQueries(fakeReader{nodes: []*topology.Node{{ID: "a"}}}, 1000)
	if _, err := q.Node(context.Background(), "missing"); !errors.Is(err, topology.ErrNodeNotFound) {
		t.Errorf("want ErrNodeNotFound, got %v", err)
	}
}

func TestNodeSurfacesListError(t *testing.T) {
	wantErr := errors.New("db down")
	q := topology.NewQueries(fakeReader{listErr: wantErr}, 1000)
	if _, err := q.Node(context.Background(), "a"); !errors.Is(err, wantErr) {
		t.Errorf("list error not surfaced: %v", err)
	}
}

func TestLinksTakeUtilizationFromTheirPort(t *testing.T) {
	ports := []topology.PortUtilization{
		{NodeID: "sw1", IfIndex: 1, IfName: "Gi0/1", IfDescr: "GigabitEthernet0/1", Pct: new(42.0)},
		{NodeID: "sw1", IfIndex: 2, IfName: "Gi0/2", IfDescr: "GigabitEthernet0/2"},
		{NodeID: "sw2", IfIndex: 7, IfName: "Gi1/7", Pct: new(9.0)},
		// Two ports of sw3 share a description, so it names neither.
		{NodeID: "sw3", IfIndex: 1, IfName: "eth0", IfDescr: "uplink", Pct: new(5.0)},
		{NodeID: "sw3", IfIndex: 2, IfName: "eth1", IfDescr: "uplink", Pct: new(6.0)},
		// The same port reached through two polling targets.
		{NodeID: "sw4", IfIndex: 3, IfName: "p3", Pct: new(11.0)},
		{NodeID: "sw4", IfIndex: 3, IfName: "p3", Pct: new(12.0)},
	}
	tests := []struct {
		name string
		link topology.Link
		want *float64
	}{
		{"source port by ifName", topology.Link{SourceNodeID: "sw1", SourceInterface: "Gi0/1"}, new(42.0)},
		{
			"source port by ifDescr",
			topology.Link{SourceNodeID: "sw1", SourceInterface: "GigabitEthernet0/1"},
			new(42.0),
		},
		{
			"source unpolled, target port",
			topology.Link{SourceNodeID: "host", SourceInterface: "en0", TargetNodeID: "sw2", TargetInterface: "Gi1/7"},
			new(9.0),
		},
		{
			"source port wins over target",
			topology.Link{SourceNodeID: "sw1", SourceInterface: "Gi0/1", TargetNodeID: "sw2", TargetInterface: "Gi1/7"},
			new(42.0),
		},
		{
			"source port with no rated poll falls back to target",
			topology.Link{SourceNodeID: "sw1", SourceInterface: "Gi0/2", TargetNodeID: "sw2", TargetInterface: "Gi1/7"},
			new(9.0),
		},
		{"port on another node", topology.Link{SourceNodeID: "sw2", SourceInterface: "Gi0/1"}, nil},
		{"ambiguous label", topology.Link{SourceNodeID: "sw3", SourceInterface: "uplink"}, nil},
		{
			"unambiguous name beside an ambiguous label",
			topology.Link{SourceNodeID: "sw3", SourceInterface: "eth1"},
			new(6.0),
		},
		{"one port through two targets", topology.Link{SourceNodeID: "sw4", SourceInterface: "p3"}, new(12.0)},
		{"no port named", topology.Link{SourceNodeID: "sw1"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := tt.link
			q := topology.NewQueries(fakeReader{links: []*topology.Link{&link}, ports: ports}, 1000)
			got, err := q.Links(context.Background(), "")
			if err != nil {
				t.Fatalf("Links: %v", err)
			}
			switch gotPct := got[0].UtilizationPct; {
			case tt.want == nil && gotPct != nil:
				t.Errorf("utilization = %v, want none", *gotPct)
			case tt.want != nil && (gotPct == nil || *gotPct != *tt.want):
				t.Errorf("utilization = %v, want %v", gotPct, *tt.want)
			}
		})
	}
}

func TestLinksSurfacePortUtilizationError(t *testing.T) {
	wantErr := errors.New("metrics down")
	q := topology.NewQueries(fakeReader{
		links:    []*topology.Link{{SourceNodeID: "sw1"}},
		portsErr: wantErr,
	}, 1000)
	if _, err := q.Links(context.Background(), ""); !errors.Is(err, wantErr) {
		t.Errorf("port utilization error not surfaced: %v", err)
	}
}
