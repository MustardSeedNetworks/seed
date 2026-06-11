# ADR-0023: SNMP polling as one engine driving per-target collector chains

**Status:** Accepted — 2026-06-10 · as-built (documents `internal/polling/snmp`, the Stage A0 NMS scaffold)

## Context

The V1.0 NMS expansion replaces three legacy polling services —
`internal/services/estatepoll`, `internal/services/servermon`, and the
SNMP-polling half of `internal/services/microburst` — with one coherent active
poller (Stage A3 of the NMS plan). Those legacy services each owned their own
scheduling, their own SNMP session handling, and their own idea of what to read,
which is exactly the duplication the re-architecture removes.

Two cross-cutting constraints shaped the replacement:

1. **Different targets need different reads.** A switch wants LLDP/CDP/FDB/iftable;
   a server wants host-resources; a router wants routing/BGP. A single fixed poll
   loop either over-reads every device or special-cases device types in code.

2. **The protocol primitives already have an owner.** `internal/protocols/snmp`
   owns OID definitions, gosnmp session wrappers, and MIB parsing. The poller must
   *use* that, not re-implement wire-level SNMP, and must keep CGO/transport
   concerns out of the orchestration logic.

## Decision

**`internal/polling/snmp`** is one `engine.Engine`-shaped `Poller` per Seed
instance that schedules and runs an **ordered, per-target collector chain**.

- **`Poller` is an engine.** `Name() "snmp-poller"`, `Start`/`Stop`, and a
  `Status()` reporting last-chain time, last error, and in-flight count to the same
  registry that owns probes, retention, and listeners (ADR-0022). On `Start` it
  loads every enabled row from `polling_targets` and registers **one
  `scheduler.Job` per target** at that target's `PollIntervalSec`.

- **Parallel across targets, serial within a chain.** Each target is an independent
  scheduler job, so targets poll concurrently; within a single target the collector
  chain runs sequentially. One slow device never blocks another; one slow collector
  only delays later collectors for the *same* device.

- **`Collector` is a pluggable port.** `Name() string` + `Collect(ctx, Target,
  ResolvedCredentials) error`. A target's `collector_chain` is a JSON array of
  collector names in `polling_targets`; the poller looks each up by name. A chain
  entry with **no registered collector is skipped with a warning**
  (`ErrCollectorNotRegistered`) and the rest of the chain still runs — collectors
  can be rolled out or disabled via config without a code change.

- **Non-fatal collection.** A collector returning an error is logged; the chain
  continues. After the chain runs, `last_status` / `last_error` / `last_polled_at`
  are written back to the target row, so one broken OID surface (e.g. iftable)
  doesn't lose the data the other collectors gathered (arp, fdb).

- **`Client` / `ClientFactory` abstract the session.** Collectors talk to a
  `Client` (`Get` scalar OIDs in order, `Walk` a subtree); production builds a
  gosnmp-backed client via `internal/protocols/snmp`, tests inject a fake. The
  poller owns session lifecycle and credential resolution; collectors never open
  connections.

- **Observations land via a shared sink**, so every collector persists through one
  seam rather than each reaching into the database.

## Consequences

- Adding a metric surface is a new collector package implementing the port plus a
  registration in `orchestrator.Build` — no scheduler, session, or persistence code.
  V1.0 ships ten: `sysinfo`, `iftable`, `lldp`, `cdp`, `fdp`, `arp`, `fdb`,
  `routing`, `hostresources`, `bgp4`.
- Per-target chains make device-appropriate polling a **data** decision
  (`collector_chain` column), not a code branch on device type.
- The `Client` port keeps the orchestration package CGO-free and unit-testable; the
  gosnmp/transport detail stays in `internal/protocols/snmp`.
- The poller is visible in the engine registry's health surface like every other
  long-running component, with real in-flight/last-error telemetry.

## Implementation status (2026-06-10)

- **`internal/polling/snmp`** — `Poller` (engine + scheduler wiring), `Collector` /
  `Client` / `ClientFactory` ports, `Target` / `ResolvedCredentials` types.
- **`internal/polling/snmp/orchestrator`** — `Build` wires the `Poller` with the ten
  collectors above against a persisting sink.
- **`internal/polling/snmp/collectors/*`** — one package per collector.
- **Wiring** — registered with the engine registry from `internal/api/server.go`.
- **Credential decryption is a STUB.** `credentialsForTarget()` (`poller.go`)
  returns an empty `ResolvedCredentials{}`; nothing yet decrypts a target's
  `CredentialsID` against the `device_credentials` store. Until Stage A3.x wires the
  secret/license manager in, the poller schedules and runs chains but cannot
  authenticate against real SNMPv2c/v3 devices — this path is scaffolding, not a
  shipped feature. The poll loop, chaining, status write-back, and persistence are
  real and tested with fake clients; only the credential resolution is missing.
