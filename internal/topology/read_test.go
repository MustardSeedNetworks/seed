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
	listErr    error
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
