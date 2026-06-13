package topology

// read.go is the topology read use-case (ADR-0020, WS-A6): the GET endpoints'
// application service over a narrow Reader port, so the transport layer depends
// on a use-case instead of reaching into the database directly. The Reader is
// satisfied by an adapter in the composition root over the topology repository.

import (
	"context"
	"errors"
)

// ErrNodeNotFound is returned by Node when no node matches the id.
var ErrNodeNotFound = errors.New("topology: node not found")

// ErrUnavailable is returned when the underlying store is not wired (the handler
// maps it to 503, preserving the prior "Database not initialized" behavior).
var ErrUnavailable = errors.New("topology: store unavailable")

// Reader is the read surface the topology queries need. It mirrors the topology
// repository's read methods; the adapter satisfies it over database.TopologyRepository.
type Reader interface {
	List(ctx context.Context, opts ListOptions) ([]*Node, error)
	ListInterfaces(ctx context.Context, nodeID string) ([]*Interface, error)
	ListLinks(ctx context.Context, nodeID string) ([]*Link, error)
	ListARPBindings(ctx context.Context, opts ARPListOptions) ([]*ARPBinding, error)
}

// NodeDetail is the single-node read model: the node plus its interfaces and
// incident links.
type NodeDetail struct {
	Node       *Node
	Interfaces []*Interface
	Links      []*Link
}

// Queries is the topology read use-case.
type Queries struct {
	reader   Reader
	maxLimit int
}

// NewQueries builds the read use-case. maxLimit caps the node scan used by Node
// (there is no GetByID on the repository yet; Node lists then matches in-memory).
func NewQueries(reader Reader, maxLimit int) *Queries {
	return &Queries{reader: reader, maxLimit: maxLimit}
}

// Nodes lists topology nodes matching opts.
func (q *Queries) Nodes(ctx context.Context, opts ListOptions) ([]*Node, error) {
	return q.reader.List(ctx, opts)
}

// Node returns one node with its interfaces and incident links, or
// ErrNodeNotFound. Interface/link loads are best-effort: a partial failure
// yields an empty list rather than failing the whole detail view (the prior
// handler behavior).
func (q *Queries) Node(ctx context.Context, id string) (NodeDetail, error) {
	nodes, err := q.reader.List(ctx, ListOptions{Limit: q.maxLimit})
	if err != nil {
		return NodeDetail{}, err
	}
	var node *Node
	for _, n := range nodes {
		if n.ID == id {
			node = n
			break
		}
	}
	if node == nil {
		return NodeDetail{}, ErrNodeNotFound
	}
	interfaces, _ := q.reader.ListInterfaces(ctx, node.ID)
	links, _ := q.reader.ListLinks(ctx, node.ID)
	return NodeDetail{Node: node, Interfaces: interfaces, Links: links}, nil
}

// Links returns the links incident to nodeID.
func (q *Queries) Links(ctx context.Context, nodeID string) ([]*Link, error) {
	return q.reader.ListLinks(ctx, nodeID)
}

// ARP returns the ARP bindings matching opts.
func (q *Queries) ARP(
	ctx context.Context,
	opts ARPListOptions,
) ([]*ARPBinding, error) {
	return q.reader.ListARPBindings(ctx, opts)
}
