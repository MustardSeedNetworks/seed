# ADR-0007: Discovery orchestrator convergence — engine vs pipeline, deferred to Phase 7

**Status:** Accepted — 2026-06-03 · partially superseded by [ADR-0008](0008-pure-data-discovery-types-in-schema.md) (2026-06-04) on the schema/types-extraction deferral; the Pipeline→Engine fold still stands

## Context

The discovery subsystem (`internal/discovery`, formerly `internal/services/discovery`)
carries **four** overlapping orchestrators, discovered while scoping the Phase 6
split:

| Type | File | Role | Owned by |
|---|---|---|---|
| **Engine** | `engine.go` | "primary" unified orchestrator; `DeviceRegistry` is the single source of truth; distributes events | composition root |
| **Service** | `service.go` | older direct-settings orchestrator; holds the Pipeline via `SetPipeline` | composition root |
| **Pipeline** | `pipeline.go` + `pipeline_*.go` | sequential phased model (enumeration → resolution → scanning → assessment); own `currentRun` state | Service |
| **Manager** | `manager.go` | L2 capture coordinator (LLDP/CDP/EDP) | `DeviceDiscovery` |

Engine and Pipeline do the **same job** — discover → enrich (SNMP/portscan/profile)
→ assess vulnerabilities — two different ways. Engine is the better-architected of
the two: registry-as-SSoT plus event distribution that dovetails with the new
events bus (ADR-0004) and jobs spine (ADR-0005). Pipeline is the older phased model
with a parallel `currentRun` state machine. Service exists mainly to drive Pipeline.

**The decisive finding:** the **frontend discovery UX runs on Pipeline, not Engine.**
`ui/src/hooks/usePipelineStatus.ts` + `NetworkDiscoveryCard` + `PipelineProgress`
call seven live `/api/v1/security/pipeline/*` endpoints (status/config/start/cancel/
port-intensity/timing-profiles); the eight `/api/v1/discovery/engine/*` endpoints
have **zero** frontend consumers. So the architecturally-preferable orchestrator has
no users, and the shipping UX rides the one we would otherwise retire.

A related deferral rides on the same fact: registering a code-first schema for
`EngineDiscoveryResponse` / `discovery.DiscoveredDevice` (the ~150-field cluster across
~20–30 nested structs) only delivers value once a TypeScript client consumes the
Engine endpoint — which is the same Phase 7 frontend migration.

## Decision

- **Phase 6 is dedup-only for orchestration.** The capture-port extraction (S1,
  ADR-aligned with `CGO_BUILD_STRATEGY.md`) and the package relocation (S3) ship; the
  **orchestrators are not restructured** in Phase 6. No Pipeline/Service behavior
  changes, no endpoint changes.
- **Engine is the canonical target;** Pipeline (and the Service-as-Pipeline-driver
  layer) is the legacy duplicate to retire.
- **The Pipeline → Engine fold is deferred to Phase 7,** executed together with the
  frontend migration that moves the discovery UI off `/api/v1/security/pipeline/*`
  onto Engine + the `/jobs` spine. Folding earlier would mean rewriting Pipeline's
  phased state machine to delegate to Engine while preserving the exact
  `/security/pipeline/*` wire shape `PipelineProgress` expects — work that Phase 7
  then discards when the UI stops calling those endpoints.
- **The `internal/discovery/types` extraction and the `EngineDiscoveryResponse` /
  `DiscoveredDevice` schema registration are deferred to Phase 7** (gated on the
  frontend consuming Engine). Until a consumer exists, the ~150-field cluster stays
  an opaque `any` job result / unregistered DTO rather than a speculative generated
  schema maintained against no client.

> **Partial supersession — [ADR-0008](0008-pure-data-discovery-types-in-schema.md)
> (2026-06-04):** the schema/types bullet above was revisited sooner than Phase 7.
> ADR-0008 registers `EngineDiscoveryResponse` directly by reflecting `discovery`'s
> pure-data types (drift-gated by `cmd/seed-schema`) and **withdraws the
> `internal/discovery/types` extraction as unnecessary** — the wire contract is
> generated from the domain structs and visible in review, not decoupled behind a
> new package. The rest of this ADR — deferring the Pipeline→Engine orchestrator
> fold to Phase 7 — is unaffected and still pending.

## Consequences

- Phase 6 closes as **capture port + relocation + this decision** — no speculative
  restructuring in the hot discovery zone, consistent with the "build it when a
  consumer exists" stance taken for the transactional outbox (ADR-0004 amendment).
- The duplication is documented and owned, not silently tolerated: Phase 7 has a
  named task (Pipeline → Engine fold + endpoint retirement + types/schema) with a
  clear trigger (the UI migration).
- Survey stays a stateful interactive session, not an orchestrator (unchanged).
- Risk: the two orchestrators coexist through Phase 6, so a discovery change may need
  touching both. Accepted — the alternative (a fold now) is rewritten in Phase 7.

Supersedes the Phase-6 portions of the blueprint that implied an in-phase engine↔
pipeline consolidation and an in-phase `EngineDiscoveryResponse` schema registration.
