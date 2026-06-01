# ADR-0001: Modulith hexagon structure

**Status:** Accepted — 2026-05-31

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
