package listener

import "context"

// Registry manages the set of Listeners enabled on this Seed
// instance. It owns the bind/unbind lifecycle, the fan-in of Events
// from all Listeners, and the enrichment step that resolves
// SourceAddr to (ClientID, TargetKind, TargetID).
type Registry interface {
	// Register adds a Listener. Must be called before StartAll.
	// Listeners are not bound until StartAll runs.
	Register(l Listener, cfg Config)

	// StartAll binds all registered Listeners (gated by the
	// passive_listeners license flag). Returns the first bind error;
	// listeners that bind successfully before the error remain bound.
	StartAll(ctx context.Context) error

	// StopAll closes all bound Listeners. Honors ctx deadline.
	StopAll(ctx context.Context) error

	// Events returns the fan-in channel carrying enriched Events
	// from all bound Listeners.
	Events() <-chan Event
}
