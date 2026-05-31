// Package listener defines the passive ingress contract — syslog,
// SNMP traps, NetFlow, etc. Every Listener is shaped like
// [engine.Engine] (Name + Start + Stop) so it registers in the same
// lifecycle registry that owns the probe, retention, and snmp-poller
// engines.
//
// Concrete implementations live in subpackages:
//
//   - internal/listener/syslog      — RFC 3164 / 5424 over UDP
//   - internal/listener/snmptrap    — SNMPv2c traps over UDP/162
//
// Listeners do not own persistence. Each one calls [Sink.Publish]
// with a typed [Event]; the sink decides where the event lands
// (default: the listener_events table via the database sink).
package listener

import (
	"context"
	"encoding/json"
	"time"
)

// Event is one passive-ingress observation. ClientID is filled by
// the enrichment step that resolves SourceAddr against the
// polling_targets and discovered_devices tables (Stage A4 work);
// unresolved sources land in the default client with TargetKind
// "unknown_ip".
type Event struct {
	// Kind identifies the listener that emitted this event:
	// "syslog", "snmp_trap", etc.
	Kind string `json:"kind"`

	// ClientID is "default" until enrichment resolves it.
	ClientID string `json:"clientId"`

	// SourceAddr is the host:port the packet arrived from.
	SourceAddr string `json:"sourceAddr"`

	// TargetKind / TargetID are populated by enrichment when the
	// source matches a known target. Empty when unresolved.
	TargetKind string `json:"targetKind,omitempty"`
	TargetID   string `json:"targetId,omitempty"`

	// Severity is the listener-native severity string when
	// applicable (syslog "warning", trap "informational", etc.).
	Severity string `json:"severity,omitempty"`

	// Timestamp is when the listener observed the event (wall clock).
	Timestamp time.Time `json:"timestamp"`

	// Payload is the parsed, listener-specific event body. Each
	// listener subpackage documents its payload schema.
	Payload json.RawMessage `json:"payload"`
}

// Listener is the lifecycle contract for one passive endpoint.
// Identical shape to [engine.Engine] so listeners register
// directly with the engine registry — no listener-specific
// supervisor needed.
type Listener interface {
	// Name is the stable identifier, e.g. "syslog-udp",
	// "snmp-trap-v2c". One word preferred.
	Name() string

	// Start binds the configured socket and begins streaming
	// events to the sink. Must be idempotent — a second Start
	// without an intervening Stop is a no-op.
	Start(ctx context.Context) error

	// Stop closes the socket and waits up to ctx deadline for
	// in-flight handlers to drain. Safe to call repeatedly.
	Stop(ctx context.Context) error
}

// Sink is the consumer-defined seam every listener calls to
// publish events. The default implementation (internal/listener/
// sink) persists into the listener_events table; tests inject
// a recording fake.
type Sink interface {
	Publish(ctx context.Context, evt Event) error
}
