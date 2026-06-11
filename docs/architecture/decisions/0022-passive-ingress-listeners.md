# ADR-0022: Passive-ingress listeners share the engine lifecycle and a sink seam

**Status:** Accepted — 2026-06-10 · as-built (documents `internal/listener`, shipped during the V1.0 NMS expansion)

## Context

The active diagnostics path *initiates* network conversations: the SNMP poller
(ADR-0023) reaches out to targets, discovery probes hosts. But an NMS also has to
receive **unsolicited** signals — syslog messages, SNMP traps, later NetFlow/IPFIX
— that devices push at us over UDP with no request on our side. These arrive
continuously, from many sources, on privileged low ports, and must be persisted
without back-pressuring the receive loop.

These passive endpoints share a shape: bind a socket, parse a wire format, emit a
normalized record, and run for the lifetime of the daemon. The risk was that each
one grows its own ad-hoc goroutine, its own start/stop handling, and its own
database writes — the same "per-subsystem bucket" sprawl the re-architecture is
removing elsewhere. We already have a uniform lifecycle owner: the `engine.Engine`
registry that supervises the probe, retention, and snmp-poller engines (Name +
Start + Stop, with `Status` reporting). Passive listeners fit that shape exactly.

## Decision

A single small contract package, **`internal/listener`**, defines the passive
ingress seam; concrete listeners live in subpackages and persistence lives behind
a port.

- **`Listener` is `engine.Engine`-shaped** — `Name() string`, `Start(ctx) error`,
  `Stop(ctx) error`, both idempotent. Listeners register directly with the engine
  registry; there is **no listener-specific supervisor**. Start binds the socket
  and streams; Stop closes it and drains in-flight handlers within the context
  deadline.

- **Listeners own no persistence — they publish to a `Sink` port.** `Sink.Publish(
  ctx, Event) error` is the only outbound seam. The default implementation
  (`internal/listener/sink`) inserts into the `listener_events` table; tests inject
  a recording fake. A listener never imports the database.

- **`Event` is the normalized record** — `Kind` (which listener), `SourceAddr`,
  `Severity` (listener-native string), `Timestamp`, and an opaque listener-specific
  `Payload json.RawMessage` whose schema each subpackage documents. `ClientID`,
  `TargetKind`, and `TargetID` are **enrichment outputs**, not listener concerns:
  the listener emits the raw `SourceAddr` and a later step resolves it.

- **Concrete listeners (V1.0):** `internal/listener/syslog` (RFC 3164 / RFC 5424
  over UDP) and `internal/listener/snmptrap` (SNMPv2c traps over UDP/162).

- **Binding is opt-in and fail-soft.** The composition root wires listeners in
  `initListeners` (`internal/api/server.go`); a bind failure logs a warning and the
  listener is simply not registered, rather than aborting daemon startup — a busy
  `:514`/`:162` or an unprivileged deployment must not take the server down.

## Consequences

- A new passive source (NetFlow, IPFIX, a TLS syslog variant) is a new subpackage
  implementing `Listener` plus a registry registration — no new supervisor, no new
  persistence path, no change to `internal/listener` itself.
- Because `Sink` is the only write seam, the storage decision (currently one row
  per event in `listener_events`) can change in one place without touching any
  listener, and the alerts pipeline (`internal/alerts/pipeline/listener_pipeline.go`)
  consumes the same persisted stream.
- Publishing is decoupled from receiving: a slow sink must not block the UDP read
  loop, so sink errors are logged rather than propagated back into the hot path.
- Listeners inherit the engine registry's uniform `Status`/lifecycle, so the same
  health surface that covers the poller covers ingress — no bespoke monitoring.

## Implementation status (2026-06-10)

- **`internal/listener`** — `Listener`, `Event`, `Sink` contracts (no I/O).
- **`internal/listener/syslog`** — UDP listener, RFC 3164 + RFC 5424 parse,
  default bind `:514`, remappable for unprivileged hosts. **TLS/TCP syslog
  (RFC 5425) is deferred** — it is a separate `Listener` type, not a flag on this
  one.
- **`internal/listener/snmptrap`** — SNMPv2c traps over UDP/162. **SNMPv3 (auth/priv)
  traps are out of scope for V1.0.**
- **`internal/listener/sink`** — default `Sink` over `database.ListenerEvents()`
  (`listener_events`, migration `00001_init.sql`).
- **Wiring** — `initListeners` in `internal/api/server.go` constructs the persist
  sink and both listeners and registers them with the engine registry.
- **Enrichment is NOT wired yet (Stage A4).** `ClientID` is `"default"` and
  `TargetKind`/`TargetID` are empty/`"unknown_ip"` on every event until the step
  that resolves `SourceAddr` against `polling_targets` / `discovered_devices`
  lands. Events persist and flow to the alerts pipeline; they are simply not yet
  attributed to a known device.
