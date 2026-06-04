# Architecture Decision Records

ADRs capture *why* a structural decision was made, so the reasoning survives the
re-architecture. Format: Context → Decision → Consequences. Status values:
Proposed · Accepted · Superseded.

See the [Re-Architecture Blueprint](../RE_ARCHITECTURE_BLUEPRINT.md) for the full picture.

| ADR | Title | Status |
|---|---|---|
| [0001](0001-modulith-hexagon.md) | Modulith hexagon structure | Accepted |
| [0002](0002-capability-registry.md) | Capability registry for route policy | Accepted |
| [0003](0003-contract-first-boundary.md) | Contract boundary — code-first; OpenAPI deferred | Amended |
| [0004](0004-event-bus.md) | In-process domain event bus | Accepted |
| [0005](0005-unified-jobs.md) | Unified async job runner | Accepted |
| [0006](0006-migrations-sql-goose-strict.md) | Schema as embedded `.sql` files, goose, STRICT tables | Accepted |
| [0007](0007-discovery-orchestrator-convergence.md) | Discovery orchestrator convergence — engine vs pipeline, deferred to Phase 7 | Accepted |
