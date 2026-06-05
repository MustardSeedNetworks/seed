# ADR-0008: Pure-data discovery types may be reflected into the published schema

**Status:** Accepted — 2026-06-04

## Context

Phase 2 established a policy for the code-first schema registry
(`cmd/seed-schema`): a DTO is registered only if it is flat or nests only
local, purpose-built transport sub-structs in `internal/api`. DTOs that put an
internal **domain** type on the wire (`discovery.*`, `dhcp.*`, `netif.*`,
`logging.*`, `survey.*`, `config`) were given hand-written flat `internal/api`
mirrors so the published schema never reaches into a domain package. The
intent: keep the wire contract decoupled from internal struct layout.

`EngineDiscoveryResponse` (`internal/api/handlers_engine.go`) was the one DTO
left unregistered under that policy. It embeds the discovery result cluster
directly:

```go
type EngineDiscoveryResponse struct {
    Devices      []*discovery.DiscoveredDevice `json:"devices"`
    Stats        *discovery.EngineStats        `json:"stats"`
    ScanResult   *discovery.ScanResult         `json:"scanResult,omitempty"`
    Capabilities map[string]bool               `json:"capabilities"`
}
```

`discovery.DiscoveredDevice` reaches ~20–30 nested structs (LLDP/CDP/EDP/NDP
info, `DeviceProfile`, `SNMPFullData`, `DeviceVulnerabilities`, `WiFiPresence`,
`BluetoothPresence`, `EngineStats` → `RegistryStats`/`EventBusStats`,
`ScanResult` → `ScanStats`) — roughly 150 fields. ADR-0007 deferred registering
it to Phase 7 and contemplated extracting a CGO-free `internal/discovery/types`
package to serve as a clean shared contract.

Three facts, established while scoping Phase 7 S2, change the calculus:

1. **`internal/discovery` is already CGO-free.** Phase 6 (ADR-aligned with
   `CGO_BUILD_STRATEGY.md`) confined libpcap to `internal/capture/pcap`;
   `CGO_ENABLED=0 go build ./internal/discovery/` now succeeds. The original
   driver for the types-package extraction — getting these types out of a
   CGO-coupled package so the schema tool can reflect them — is moot.

2. **The contract is already coupled at runtime.** The live
   `GET /api/v1/discovery/engine` endpoint already serializes
   `discovery.DiscoveredDevice` (and the whole cluster) to JSON on the wire.
   Registering a schema does not *create* coupling between the wire shape and
   `discovery`'s struct layout — that coupling already ships. A schema only
   *documents* it and lets us generate the TypeScript types the Phase 7
   frontend needs.

3. **The cluster is pure data.** Every reachable struct is plain data
   (strings, ints, bools, `time.Time`, named-string enums, slices, and nested
   pure-data pointers). The only method in the entire tree is
   `DiscoveredDevice.ComputeDisplayName()`. The stats types
   (`RegistryStats`/`EventBusStats`/`ScanStats`) are counter snapshots, not the
   live registry/event-bus objects — no channels, mutexes, or funcs.

Given (1)–(3), the heavyweight `internal/discovery/types` extraction (a large,
careful refactor of the hot discovery package) would buy mainly policy purity,
not CGO-freeness or genuine decoupling — the contract is already what it is.
Hand-mirroring ~150 fields into `internal/api` would create a perpetual
sync-burden parallel tree (the explicit anti-goal of the Phase 2 policy).

## Decision

- **Genuinely pure-data discovery types may be reflected directly into the
  published schema.** `EngineDiscoveryResponse` is registered in
  `cmd/seed-schema` as-is, reflecting `discovery.DiscoveredDevice`,
  `discovery.EngineStats`, and `discovery.ScanResult` and their pure-data
  closure. This carves a narrow exception into the Phase 2 "no domain type in
  the schema" policy, limited to **pure-data** types whose JSON shape the
  endpoint already emits at runtime.
- **The exception does not extend to behavior-bearing or I/O-coupled domain
  types.** A domain type that carries channels, mutexes, funcs, live handles,
  or non-trivial behavior still gets a flat `internal/api` mirror (the Phase 2
  rule). "Pure data the endpoint already serializes" is the test.
- **The `internal/discovery/types` extraction contemplated by ADR-0007 is
  withdrawn as unnecessary.** Should a future need arise to decouple the wire
  contract from `discovery`'s internals (e.g. the struct layout starts churning
  for reasons unrelated to the API), revisit the extraction then.
- The round-trip and acyclicity guardrails in `cmd/seed-schema` remain the
  enforcement: a registered DTO must round-trip against its generated schema
  and produce an acyclic `$defs` graph. Those tests validate this registration.

## Consequences

- `EngineDiscoveryResponse` gains a generated JSON schema and TypeScript type,
  unblocking the Phase 7 S3 migration of the discovery UX onto the Engine +
  jobs spine (the frontend can consume typed device results instead of an
  opaque `unknown`).
- The published wire contract for the engine endpoint is now explicit and
  drift-gated (`check-schema-drift.sh` / `check-types-drift.sh`), where before
  it was an undocumented runtime serialization. This is a net increase in
  contract discipline, not a decrease.
- The wire shape is coupled to `discovery`'s pure-data struct layout. Because
  the schema is regenerated from the Go structs and drift-gated, a field change
  surfaces in review as a schema/type diff rather than silently — the coupling
  is visible and owned.
- Supersedes the portion of ADR-0007 that deferred the `EngineDiscoveryResponse`
  schema and contemplated an `internal/discovery/types` extraction as its
  prerequisite.
