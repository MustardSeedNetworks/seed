package retention

import "context"

// Engine runs the periodic rollup + tier-aware purge over all
// registered RollupSources. Implementation lands in Stage A2.
type Engine interface {
	// Register adds a RollupSource. Must be called before Start.
	Register(src RollupSource)

	// Start begins the periodic rollup + purge loop. Honors ctx
	// cancellation for clean shutdown.
	Start(ctx context.Context) error

	// Stop signals the loop to exit and waits up to ctx deadline for
	// in-flight rollups to complete.
	Stop(ctx context.Context) error
}
