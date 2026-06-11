# ADR-0026: Delete the dead health-check read-path (the `health_check_results` stack)

**Status:** Accepted — 2026-06-11 · executes the health-check deletion track flagged by
[ADR-0025](0025-probe-is-the-active-monitoring-anomaly-source.md) §4 ("the health-check stack is
legacy to delete, not revive") and completes the read-side convergence begun in
[ADR-0021](0021-persist-and-converge-anomaly-engine.md). Builds on
[ADR-0020](0020-clean-hexagonal-api-foundation.md) (the `internal/health/monitoring` use-case seam
this prunes).

## Context

ADR-0021 converged anomaly persistence onto one SQL store and deleted the never-fed bespoke
`internal/health.AnomalyDetector`. ADR-0025 then named `internal/probe` the active-monitoring
anomaly producer and declared the rest of the health-check stack — `health_check_results`, the
`HealthCheckRepository` write path, and the `internal/health` scoring/SLA/dependency code that reads
that table — **legacy to delete, not revive**. This ADR carries out that deletion.

A read-only audit (2026-06-11) confirmed the stack is entirely dead:

| Surface | State | Evidence |
|---|---|---|
| `health_check_results`, `health_check_rollups_hourly`, `health_check_rollups_daily` | **written by nothing** | `HealthCheckRepository.Record`/`RecordBatch` have zero production callers |
| `GET /telemetry/health-checks/results`, `/history` | **dead** | zero frontend callers **and** an empty backing table |
| `/scores`, `/sla` | **dead** | `health.ScoringService` / `health.SLATracker` read the empty table → always zero; the only consumer (`SlaDashboardCard`) renders all-zeros |
| `/alerts` | **dead** | `alerts.AlertManager` is never constructed (`s.healthAlerts` is never assigned; its ingest methods have zero callers) → the collaborator is nil → the route always returns 503 |
| `internal/health` (`scoring.go`, `sla.go`, `dependencies.go`) | **dead** | imported only by the seam being pruned; `DependencyManager` is constructed but has no reader at all |
| `internal/alerts` | **dead** | same importers; unfed |

The live pieces that **stay** are a different feature and out of scope here:

- `GET /telemetry/health-checks/run` — the ad-hoc, in-memory "run all configured checks now"
  handler. Computes fresh and returns inline; never touches the dead table. Consumed by
  `HealthCheckCard`.
- `GET|PUT /telemetry/health-checks/settings` — the check-target configuration CRUD. Drives `/run`.
- `GET /telemetry/health-checks/anomalies` — already repointed (ADR-0021 phase 4 / ADR-0025) at the
  unified store's `source=probe` slice; the probe producer feeds it. The only live stat on
  `SlaDashboardCard`.

## Decision

**Delete the dead health-check read-path in full — backend, the two now-callerless routes
(`/results`, `/history`) plus the three always-empty/always-503 routes (`/scores`, `/sla`,
`/alerts`), and the dead frontend (the three all-zero `SlaDashboardCard` sections and the
`slaConfigs`/`alertConfig` settings fields that feed only this stack). No stub is preserved.**

SLA reporting, endpoint scoring, and alerting **as features** are not ported — they are not
preserved as a dead shell either. If they are wanted, they are a fresh feature built directly on the
live `probe_results` time series + breach stream, with its own ADR, not a revival of this stack
(ADR-0025 §4, §"Alternatives"). Keeping a card that renders zeros and settings that configure
nothing is the staleness this codebase's Definition of Done forbids.

### What is deleted

- **DB:** migration `00007` drops `health_check_results`, `health_check_rollups_hourly`,
  `health_check_rollups_daily` and their indexes; `internal/database/repository_health_checks.go` and
  its models (`HealthCheckResult`, `HealthCheckHourlyRollup`, `HealthCheckDailyRollup`,
  `HealthCheckQueryOptions`); the `DB.HealthChecks()` accessor and `healthChecks` field; the
  `retention.go` `cleanupHealthCheck{Raw,Hourly,Daily}` tasks + their `RetentionPolicy`/`CleanupResult`
  fields.
- **Domain:** `internal/health` (`scoring.go`, `sla.go`, `dependencies.go`) and `internal/alerts`
  (whole packages).
- **Use-case (`internal/health/monitoring`):** the `ResultStore`, `Scorer`, `SLAReporter`, and
  `AlertReader` ports and the `Results`/`History`/`Scores`/`SLAReport`/`SLASummary`/`Alerts`/
  `AcknowledgeAlert` methods + their read models. The `AnomalyReader` port and `Anomalies` method
  **survive** — the package narrows to the one live concern.
- **Composition root (`internal/app/health.go`):** the four dead adapters; `NewHealthMonitoring`
  collapses to the single anomaly collaborator.
- **API:** `server_routes.go` registrations for `/results`, `/history`, `/scores`, `/sla`,
  `/alerts`; the matching handlers in `internal/api/health.go` (`handleHealthCheckAnomalies` stays);
  the `healthRepo`/`healthScore`/`healthSLA`/`healthDeps`/`healthAlerts` server fields, their
  accessors, and their construction in `initHealthRepositories`/`initHealthUseCases`.
- **Frontend:** `SlaDashboardCard` reduced to the live anomalies view (its scores/SLA/alerts
  sections + their fetches removed); the `slaConfigs`/`alertConfig` fields and their
  `HealthChecksSettingsAdvanced` UI; the now-orphaned response types.

### Endpoint path: still no rename

`/telemetry/health-checks/anomalies` keeps its path. The health-checks transport family does **not**
fully strangle here — `run` and `settings` remain live and legitimately health-check-named. Renaming
the family to `/telemetry/probes/*` rides the future migration of `run`/`settings` onto probe
(ADR-0025 §5), so the surface never goes half-renamed.

## Consequences

- ~Two thousand lines of unreachable code, two dead packages, three empty tables, and a misleading
  all-zeros dashboard are gone. The `monitoring` use-case now has a single honest concern.
- `SlaDashboardCard` shows only live data (probe-sourced active anomalies). Its name becomes a slight
  misnomer; a later rename/relocation rides the SLA-on-probe feature if it lands.
- A debt is made explicit and *not* silently carried as a stub: SLA/scoring/alerting on `probe_results`
  is a future feature, owned by a future ADR, only if prioritized.
- The live ad-hoc `/run` + `/settings` health-check feature is untouched; its own eventual migration
  onto probe (and the family path rename) is separate, larger work.

## Alternatives considered

- **Keep the frontend/settings as a stub until a probe-backed rebuild lands.** Rejected — leaves dead
  config and an all-zero dashboard, exactly the staleness the DoD forbids; pre-alpha has no
  compatibility reason to keep it.
- **Rebuild SLA/scoring on `probe_results` in this change.** Rejected — that is a new feature build
  (design + catalog + UI), not a deletion; folding it in would bloat the blast radius and keep the
  dead code alive until the new code ships. Clean strangles delete first; the rebuild, if wanted, is
  its own scoped ADR.
