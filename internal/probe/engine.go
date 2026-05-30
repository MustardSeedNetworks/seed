package probe

import "context"

// Engine schedules and dispatches probes, evaluates thresholds, and
// emits ResultEvents to subscribers. Implementation lands in Stage
// A1; this is the contract.
//
// Lifecycle: callers construct an Engine, register Checkers, then
// call Start. Start spawns the scheduler loop. Stop drains in-flight
// probes and closes subscriber channels. On-demand invocations
// (AutoTest, manual triggers) go through RunNow/RunDefinition.
type Engine interface {
	// Start begins the scheduler loop, reading enabled probes from
	// the probes table and dispatching them at their configured
	// intervals.
	Start(ctx context.Context) error

	// Stop drains in-flight probes (bounded by ctx) and closes all
	// subscriber channels.
	Stop(ctx context.Context) error

	// RunNow dispatches one probe immediately, bypassing the
	// schedule. The Result is persisted exactly as a scheduled run
	// would be. Used by AutoTest sequences and manual UI triggers.
	RunNow(ctx context.Context, probeID string) (Result, error)

	// RunDefinition dispatches an ad-hoc probe that is not persisted
	// in the probes table. The Result is NOT persisted either.
	// Used for one-shot diagnostics and transaction-step execution.
	RunDefinition(ctx context.Context, p Probe) Result

	// RegisterChecker adds a Checker to the engine's registry.
	// Replaces any previously-registered Checker for the same Kind.
	RegisterChecker(c Checker)

	// Subscribe returns a channel that receives ResultEvents for
	// every completed probe. Subscribers (alerts pipeline) must
	// drain the channel; the engine drops events on a full buffer
	// and increments a counter.
	Subscribe() <-chan ResultEvent
}
