package topology_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

type fakeReader struct {
	nodes      []*database.TopologyNode
	interfaces []*database.TopologyInterface
	links      []*database.TopologyLink
	listErr    error
}

func (f fakeReader) List(context.Context, database.TopologyListOptions) ([]*database.TopologyNode, error) {
	return f.nodes, f.listErr
}

func (f fakeReader) ListInterfaces(context.Context, string) ([]*database.TopologyInterface, error) {
	return f.interfaces, nil
}

func (f fakeReader) ListLinks(context.Context, string) ([]*database.TopologyLink, error) {
	return f.links, nil
}

func (f fakeReader) ListARPBindings(
	context.Context,
	database.TopologyARPListOptions,
) ([]*database.TopologyARPBinding, error) {
	return nil, nil
}

func TestNodeReturnsDetailForMatch(t *testing.T) {
	r := fakeReader{
		nodes:      []*database.TopologyNode{{ID: "a"}, {ID: "b"}},
		interfaces: []*database.TopologyInterface{{IfName: "eth0"}},
		links:      []*database.TopologyLink{{LinkType: "lldp"}},
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
	q := topology.NewQueries(fakeReader{nodes: []*database.TopologyNode{{ID: "a"}}}, 1000)
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
