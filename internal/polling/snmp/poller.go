package snmp

import "context"

// Poller schedules SNMP polling across all enabled Targets. One
// instance per Seed instance. Implementation lands in Stage A3.
//
// Per-target lifecycle on each tick:
//  1. Resolve the Target's credentials from device_credentials
//     (decrypted via license.Manager.DecryptSecret)
//  2. Walk the configured CollectorChain in order, invoking each
//     registered Collector
//  3. Persist results via the collectors' chosen surfaces (metrics,
//     events, topology)
//  4. Update the Target's last_polled_at / last_status / last_error
type Poller interface {
	// RegisterCollector adds a Collector. Re-registering the same
	// Name replaces the previous Collector. Must be called before
	// Start.
	RegisterCollector(c Collector)

	// Start begins the per-target scheduling loop. Honors ctx
	// cancellation for clean shutdown.
	Start(ctx context.Context) error

	// Stop signals the loop to exit and waits up to ctx deadline for
	// in-flight collector chains to complete.
	Stop(ctx context.Context) error
}
