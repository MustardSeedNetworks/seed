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
	PortUtilization(ctx context.Context) ([]PortUtilization, error)
}

// PortUtilization is one interface of a polled node with the utilization of
// its latest rated poll: the larger direction, in percent of line rate
// (ADR-0033 §5). Pct is nil when that poll could not rate the octets or the
// interface has no rated poll yet.
type PortUtilization struct {
	NodeID  string
	IfIndex uint32
	IfName  string
	IfDescr string
	Pct     *float64
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
	_ = q.attachUtilization(ctx, links)
	return NodeDetail{Node: node, Interfaces: interfaces, Links: links}, nil
}

// Links returns the links incident to nodeID, or every link when nodeID
// is empty, each with the utilization of its port.
func (q *Queries) Links(ctx context.Context, nodeID string) ([]*Link, error) {
	links, err := q.reader.ListLinks(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if err = q.attachUtilization(ctx, links); err != nil {
		return nil, err
	}
	return links, nil
}

// attachUtilization sets each link's UtilizationPct from its source port, or
// from its target port when the source end is not polled. The edge
// reconcilers name a port by its ifDescr or ifName, so a link's interface is
// matched against both; a label that two ports of one node share names
// neither, as in the reconcilers' own port lookup.
func (q *Queries) attachUtilization(ctx context.Context, links []*Link) error {
	if len(links) == 0 {
		return nil
	}
	ports, err := q.reader.PortUtilization(ctx)
	if err != nil {
		return err
	}
	type portKey struct{ nodeID, label string }
	type port struct {
		ifIndex   uint32
		pct       *float64
		ambiguous bool
	}
	byLabel := make(map[portKey]*port, len(ports))
	for _, p := range ports {
		for _, label := range []string{p.IfDescr, p.IfName} {
			if label == "" {
				continue
			}
			key := portKey{p.NodeID, label}
			prev, seen := byLabel[key]
			switch {
			case !seen:
				byLabel[key] = &port{ifIndex: p.IfIndex, pct: p.Pct}
			case prev.ifIndex != p.IfIndex:
				prev.ambiguous = true
			case p.Pct != nil && (prev.pct == nil || *p.Pct > *prev.pct):
				// The same port reached through a second polling target.
				prev.pct = p.Pct
			}
		}
	}
	lookup := func(nodeID, label string) *float64 {
		if p, ok := byLabel[portKey{nodeID, label}]; ok && !p.ambiguous {
			return p.pct
		}
		return nil
	}
	for _, l := range links {
		l.UtilizationPct = lookup(l.SourceNodeID, l.SourceInterface)
		if l.UtilizationPct == nil {
			l.UtilizationPct = lookup(l.TargetNodeID, l.TargetInterface)
		}
	}
	return nil
}

// ARP returns the ARP bindings matching opts.
func (q *Queries) ARP(
	ctx context.Context,
	opts ARPListOptions,
) ([]*ARPBinding, error) {
	return q.reader.ListARPBindings(ctx, opts)
}
