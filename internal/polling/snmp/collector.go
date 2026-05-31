package snmp

import "context"

// Collector executes one OID-tree gathering pass against a Target.
// One Collector per logical OID surface (sys_info, if_table, lldp,
// arp, fdb, routing, host_resources, bgp4_mib, microburst_counters).
// Implementations live in internal/polling/snmp/collectors/.
//
// Collectors may emit any combination of:
//   - metrics writes (via the metrics repository)
//   - event writes (via kind-specific event repositories)
//   - topology observations (via the topology reconciler)
//
// The collector chain is per-target, ordered, and serially executed.
// One slow collector can delay later collectors in the same chain
// but does not affect other targets' chains.
type Collector interface {
	// Name is a stable identifier matching the strings used in
	// polling_targets.collector_chain JSON.
	Name() string

	// Collect runs one pass against the target. The Session is the
	// gosnmp session wrapped by internal/protocols/snmp; Creds
	// carries the decrypted auth material. Return errors are logged
	// but do not abort the chain — later collectors still run.
	Collect(ctx context.Context, target Target, creds ResolvedCredentials) error
}
