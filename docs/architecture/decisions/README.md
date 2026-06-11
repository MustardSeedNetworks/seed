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
| [0004](0004-event-bus.md) | In-process domain event bus | Amended |
| [0005](0005-unified-jobs.md) | Unified async job runner | Accepted |
| [0006](0006-migrations-sql-goose-strict.md) | Schema as embedded `.sql` files, goose, STRICT tables | Accepted |
| [0007](0007-discovery-orchestrator-convergence.md) | Discovery orchestrator convergence — engine vs pipeline, deferred to Phase 7 | Amended |
| [0008](0008-pure-data-discovery-types-in-schema.md) | Pure-data discovery types may be reflected into the published schema | Accepted |
| [0009](0009-profile-ui-types-are-a-curated-view.md) | profile.ts/settings.ts are a curated UI view, not a config.Config mirror | Accepted |
| [0010](0010-identifier-casing-conventions.md) | Identifier casing — camelCase JSON wire, snake_case files/SQL | Accepted |
| [0011](0011-network-anomaly-engine.md) | Network-wide anomaly engine — one typed stream, data-driven catalog | Accepted |
| [0012](0012-wifi-monitor-capture-decode.md) | Wi-Fi monitor-mode capture port + 802.11 decode pipeline | Accepted |
| [0013](0013-bluetooth-live-capture.md) | Bluetooth live-scan capture port | Accepted |
| [0014](0014-config-validation-schema-is-a-constraints-validator.md) | config validation schema is a curated constraints validator, not a duplicate | Accepted |
| [0015](0015-credential-encryption-key-separation.md) | Separate the credential data-encryption key from `Auth.JWTSecret` | Amended |
| [0016](0016-strangle-internal-api-into-use-cases.md) | Strangle `internal/api` into per-domain use-case services | Accepted |
| [0017](0017-transactional-outbox-relay.md) | Transactional outbox relay — durable post-commit event delivery | Accepted |
| [0018](0018-discovery-pipeline-stage-split.md) | Discovery pipeline stage split — capabilities as staged ports | Accepted |
| [0019](0019-ed25519-signed-license-tokens.md) | Replace forgeable rotor-cipher license key with Ed25519-signed tokens | Accepted |
| [0020](0020-clean-hexagonal-api-foundation.md) | Clean-hexagonal `internal/api` foundation — use-cases + composition root | Accepted |
| [0021](0021-persist-and-converge-anomaly-engine.md) | Persist the anomaly engine in SQL and converge every source on it | Accepted |
| [0022](0022-passive-ingress-listeners.md) | Passive-ingress listeners share the engine lifecycle and a sink seam | Accepted |
| [0023](0023-snmp-polling-orchestrator.md) | SNMP polling as one engine driving per-target collector chains | Accepted |
| [0024](0024-identity-use-cases-and-repository-ports.md) | Identity decomposition — users / oauth / tokens use-cases over repository ports | Accepted |
| [0025](0025-probe-is-the-active-monitoring-anomaly-source.md) | Probe is the recurring-observation engine and the anomaly producer (probe vs jobs) | Accepted |
| [0026](0026-delete-dead-health-check-read-path.md) | Delete the dead health-check read path (results/scoring/SLA); keep run/settings/anomalies | Accepted |
| [0027](0027-migrate-health-checks-onto-probe.md) | Migrate on-demand health-checks onto the probe engine, then rename the transport | Proposed |
