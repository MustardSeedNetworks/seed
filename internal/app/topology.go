package app

// topology.go wires the composition root to the topology read use-case
// (ADR-0020, WS-A6). The adapter implements the topology.Reader port over the
// topology repository, resolving the database lazily; a nil database yields
// topology.ErrUnavailable so the handler degrades to 503 rather than panicking.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// NewTopologyQueries builds the topology read use-case over a lazy database
// accessor. maxLimit caps the node scan used for single-node lookups.
func NewTopologyQueries(db func() *database.DB, maxLimit int) *topology.Queries {
	return topology.NewQueries(topologyReader{db: db}, maxLimit)
}

// topologyReader implements topology.Reader over the topology repository. A nil
// database makes every read return topology.ErrUnavailable.
type topologyReader struct {
	db func() *database.DB
}

func (a topologyReader) repo() (*database.TopologyRepository, error) {
	db := a.db()
	if db == nil {
		return nil, topology.ErrUnavailable
	}
	return db.Topology(), nil
}

func (a topologyReader) List(
	ctx context.Context, opts database.TopologyListOptions,
) ([]*database.TopologyNode, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.List(ctx, opts)
}

func (a topologyReader) ListInterfaces(
	ctx context.Context, nodeID string,
) ([]*database.TopologyInterface, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.ListInterfaces(ctx, nodeID)
}

func (a topologyReader) ListLinks(
	ctx context.Context, nodeID string,
) ([]*database.TopologyLink, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.ListLinks(ctx, nodeID)
}

func (a topologyReader) ListARPBindings(
	ctx context.Context, opts database.TopologyARPListOptions,
) ([]*database.TopologyARPBinding, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.ListARPBindings(ctx, opts)
}
