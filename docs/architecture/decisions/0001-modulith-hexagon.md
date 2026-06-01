# ADR-0001: Modulith hexagon structure

**Status:** AMENDED — 2026-06-01 (see `PHASE3_RECONCILE_PROPOSAL.md`)

> **Amendment (2026-06-01):** the `internal/modules/<botanical>` + `internal/
> adapters/` rings this ADR proposed were found to be dead parallel wiring (the
> api consumes feature packages directly, not the module facades). The structural
> intent — *dependencies point inward, infra behind ports, depguard-enforced
> direction* — is **retained**, but realized as a **capability-first modular
> monolith**: flat `internal/<feature>` packages (`wifi`, `diagnostics`,
> `security`, `reporting`), one composition root, **ports as a technique at real
> I/O seams** (not a dedicated `adapters/` folder ring). The botanical module
> facades + the `modules/`/`adapters/` rings are being removed. depguard now
> enforces *direction*, not folder layout.

**Status (original):** Accepted — 2026-05-31

## Context

Seed grew organically. The product's five-module mental model
(Roots/Canopy/Shell/Sap/Harvest) is not represented consistently in the tree:
Canopy and Harvest are top-level domain packages, but Sap is `internal/services/*`
(grouped by layer), Shell is split across `services/shell` + `services/discovery`,
and Roots is `internal/pipeline/*` (named after a mechanism). Business logic lives
inside 800–1,100-LOC HTTP handlers that also call `db.Devices()` directly, so there
is no I/O-free place to put logic or to unit-test it. `internal/api` imports 24
internal packages (god package); `ServiceContainer` has 60+ leaf fields.

Three structures were considered: layer-first hexagon (`core/`+`adapters/`),
per-module full hexagon (`modules/<m>/{domain,app,ports,adapters}`), and a
modulith hybrid.

## Decision

Adopt the **modulith hybrid**:

- `internal/modules/<m>/` — pure domain logic + `ports.go`, no `net/http`/`database/sql`.
- `internal/platform/` — cross-cutting spine (config, logging, lifecycle, events, jobs, …).
- `internal/adapters/{http,cli,store,net}/` — infra ring, grouped by technology, organized by module *inside*.
- `internal/app/` — single composition root.

Dependencies point inward, enforced by `depguard`. Modules never import each other.
Seed stays a single binary (modulith), not microservices.

## Consequences

- High cohesion for domain logic; the registry + unified contract get one HTTP view.
- The five modules become first-class and consistent across seed/stem/niac.
- Requires a one-time, mechanical package-move (Phase 0 skeleton + Phase 3 extraction).
- Rejected: layer-first (splits a module's logic from its adapters → lower cohesion);
  per-module full hexagon (4-layer nesting × 5 modules = ceremony without the
  microservice isolation payoff).
