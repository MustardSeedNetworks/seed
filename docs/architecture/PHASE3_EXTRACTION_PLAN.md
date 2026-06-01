# Phase 3 — Domain Core / Hexagon Extraction Plan

**Status:** Proposed — 2026-06-01
**Owner:** re-architecture workstream
**Scope:** Implements Phase 3 of `RE_ARCHITECTURE_BLUEPRINT.md` (§16) under ADR-0001
(modulith hexagon). Phases 0–2 are complete (registry + code-first contract,
97 DTOs, round-trip + non-recursive CI gates).

This is a plan of record to review **before** any package moves. It refines the
blueprint's Phase 3 row with the current tree, concurrent-session reality, and a
concrete pilot (harvest) so the pattern is proven on the lowest-risk module
before it is copied to the rest.

---

## 1. Goal

Move domain logic out of fat HTTP handlers and concrete-dependency packages into
pure `internal/modules/<m>/` cores that are unit-testable without I/O. Each
module declares the interfaces it needs (`ports.go`); infrastructure
(`adapters/{http,store,net}`) implements them; `internal/app` wires the graph.
Direction is enforced by `depguard`: `adapters → modules → platform → stdlib`,
and **modules never import each other**.

Non-goal for Phase 3: the full platform/adapters *spine rehome* (moving
`auth`/`config`/`logging`/`api`/`database` wholesale). See §6 — deferred.

---

## 2. Constraints that shape the plan

- **Greenfield:** no customers, no back-compat. Bigger PRs are fine; slice for
  reviewability, not compatibility. Move-and-preserve debugged logic — never
  rewrite from scratch.
- **Behavior-preserving, golden-test-gated.** Every extraction rides the golden
  HTTP harness (Phase 0). The "failing test first" for a mechanical move is the
  golden snapshot that must still pass; new behavior gets a new unit test.
- **`main` stays green for the team** — there are concurrent sessions.
- **`depguard` warn → deny, per directory.** A module's rule lands as `warn`
  while logic is still being lifted, flips to `deny` once the package is clean.

### 2.1 Concurrent-session map (collision risk, measured 2026-06-01)

| Area | Commits/14d | Phase-3 disposition |
|---|---|---|
| `polling` | 16 | **HOT** — do not touch (active NMS workstream) |
| `alerts` | 6 | **HOT** — do not touch |
| `services/discovery` | 4 | HOT-ish — defer to Phase 6 anyway |
| `auth` | 3 | WARM — part of deferred spine rehome; leave |
| `canopy` | 2 | WARM — extract after pattern is proven |
| `harvest` | 1 | **COLD** — pilot |
| `pipeline` (roots) | 0 | COLD |
| `oauth`, `services/shell` | 0 | COLD (shell = Phase 6) |

The blueprint's Phase-0 "skeleton + spine rehome" was deferred precisely because
moving the spine packages collides with these hot zones. Phase 3 therefore
creates only the structure the pilot needs and grows it module-by-module.

---

## 3. Strategy: strangler, skeleton-light

1. **Create only `internal/modules/` and `internal/app/`** with the pilot. Do
   **not** pre-create or pre-fill `internal/platform/` and `internal/adapters/`
   by moving existing spine packages — wire the pilot against the packages where
   they live today (`internal/config`, `internal/database`, `internal/api`).
   The hexagon ring names are a *destination*, reached package-by-package; the
   pilot proves the dependency *direction* (inward) without the disruptive moves.
2. **Pilot one cold, self-contained module end-to-end** (harvest) — §4.
3. **Roll the rest cold→hot** — §5.
4. **Rehome the spine last, coordinated** — §6.

> Pragmatic note on `app/`: the pilot introduces a *partial* composition root that
> builds the harvest module from its `Deps`, used alongside the existing
> `ServiceContainer` wiring. `app` absorbs the rest of the graph as later modules
> extract; the 60-field `ServiceContainer` is retired only when the last module
> lands. No big-bang swap.

---

## 4. Pilot — `harvest` (reporting / export / logs)

### 4.1 Why harvest

- Cold (1 commit/14d) and **only one external importer** (`internal/api/modules.go`)
  → minimal blast radius and near-zero rebase risk against concurrent work.
- Already service-structured (`services_*.go` + a `Module` facade) and **I/O-light**:
  no direct `net/http` or `database/sql`. It is the closest existing package to
  the target shape, so it best isolates "did the *pattern* work" from "did a big
  messy move work".

### 4.2 Current dependencies (→ become ports)

| harvest uses today | Becomes | Notes |
|---|---|---|
| `internal/database` (8 files) | `ReportRepo` port | report/template/schedule CRUD + the metric aggregation queries; implemented in `adapters/store` (today: `internal/database`) |
| filesystem (`os`, `path/filepath`) + `gofpdf` | `ReportStore` + `Exporter` ports | save/read report files; render PDF/HTML/CSV/JSON |
| `time` / `uuid` | `Clock` + `IDGen` ports | makes async generate + scheduler deterministic in tests |
| `internal/config` (8 files) | `cfg.Harvest` slice | module gets only its config slice, per §5.5 of blueprint |
| **`internal/health`** (1 file: `health_report.go`) | **`HealthSource` port** | ⚠️ cross-module read (harvest ← sap/health). Forbidden as a direct import under module-independence. Inject as a query port now; convert to a `platform/events` subscription when Phase 4 lands. |

### 4.3 Target layout

```
internal/modules/harvest/
  service.go        # GeneratorService orchestration (async generate, save, fail) — I/O-free
  ports.go          # ReportRepo, ReportStore, Exporter, Clock, IDGen, HealthSource
  aggregator/       # metric aggregation (pure)
  generator/        # render orchestration (pure; format renderers call Exporter)
  scheduler/        # recurring-report scheduling (pure; uses Clock)
  templates/        # built-in + custom templates (pure)
internal/adapters/http/harvest_handlers.go   # thin: decode -> service -> encode (registry unchanged)
internal/adapters/store/harvest_repo.go      # implements ReportRepo over sqlite (lifted from internal/database)
internal/adapters/store/report_files.go      # implements ReportStore over the filesystem
internal/app/harvest.go                       # builds harvest.New(harvest.Deps{...})
```

PDF/CSV/HTML/JSON renderers move under `generator/` but the byte-producing parts
sit behind the `Exporter` port so `service.go` stays I/O-free.

### 4.4 depguard for the pilot

Add a `domain-purity` rule scoped to `**/internal/modules/harvest/**`:
deny `net/http`, `database/sql`, `internal/adapters/**`, and the other four
`internal/modules/*`. Land as `warn`; flip to `deny` once the lift is clean.
(General per-module rule shape is in blueprint §4.3.)

### 4.5 Acceptance criteria (pilot done when all true)

- [x] `go test ./...` green; **harvest report/export golden snapshots unchanged**.
- [x] `internal/modules/harvest` imports no `net/http`, no `database/sql`, no
      `internal/adapters/**`, no `internal/database`, no sibling
      `internal/modules/*` — `depguard` at `deny` (1b-v completed the
      `internal/database` ban, prod-only via `harvest-no-database`).
- [x] harvest's logic has table-driven unit tests that run with **fake ports**
      (no DB, no filesystem) — `report_repo_test.go` (fake `ReportRepo`) and
      `aggregator_repo_test.go` (fake `MetricsRepo`).
- [x] `internal/app/harvest.go` builds the module from `Deps`; `internal/api/modules.go`
      consumes the module through the same surface (no behavior change).
- [x] The harvest→health coupling is gone (it was dead code — deleted, #1428).
- [x] Docs synced (§7): `THE_SEED_ARCHITECTURE` *Hexagon Structure* section
      (ring diagram + harvest folder tree), `THE_SEED_BACKEND_ARCHITECTURE`
      *Repository ports* subsection, and a `platform-architecture.md` note —
      msn-docs #18.

**Pilot complete (2026-06-01).** All §4.5 criteria green. harvest is the proven
exemplar for the modulith hexagon; the pattern (relocate → cut cross-module
leaks → ports → adapters/store → app composition root → depguard bans) is ready
to copy to the next module (`roots`, §5).

### 4.6 PR slicing (each green, admin-merged) — STATUS

Resliced during execution into atomic PRs:

1. ✅ **1a relocate** (#1427): `git mv internal/harvest → internal/modules/harvest`,
   import-path rewrite. Pure move, golden-green.
2. ✅ **1b-ii cut health** (#1428): `health_report.go` was **dead code**
   (zero callers) → deleted, not ported. Removed harvest→health. Preserved
   `statusCritical` into `types.go`.
3. ✅ **1b-iii enforce purity** (#1429): `depguard` `modules-domain-purity`
   (deny `net/http`/`database/sql`/`internal/adapters` on `internal/modules/**`)
   + `harvest-module-independence` (deny sibling module roots). RED-proven.
4. ✅ **1b-iv ReportRepo** (see §4.7): report-record SQL (`GetReport`/
   `ListReports`/`scanReport`/`saveReport`/`DeleteReport` row) lifted verbatim
   into `internal/adapters/store/harvest_repo.go` behind the `harvest.ReportRepo`
   port (`ports.go`). `GeneratorService` delegates and keeps `db` for export
   (1b-v). Rewired `harvest.New`/`NewGeneratorService` + the 2 prod callers
   (`cmd_serve.go`, `cmd_service_windows.go`) + all test sites. `depguard`
   `modules-domain-purity` split: the transport/SQL-driver ban stays universal,
   the infra-ring ban is now production-only (`!$test`) so test files can wire
   the real store adapter (`modules-no-adapter-import`). Golden HTTP suite
   unchanged; lint 0; `go test ./...` green. Added a DB-free `GeneratorService`
   unit test via a fake `ReportRepo`.
5. ✅ **1b-v ScheduleRepo + MetricsRepo + ExportRepo** — all remaining harvest
   SQL lifted into `internal/adapters/store` (`harvest_schedule_repo.go`,
   `harvest_metrics_repo.go`, `harvest_export_repo.go`), behind three new ports
   in `ports.go`. The aggregator keeps severity-bucket / category semantics
   (the meaning of `statusCritical` stays in the domain; `MetricsRepo` returns a
   raw `severity → count` map); `sqliteDateFormat` moved to the adapter (a SQL
   concern). `GeneratorService`/`AggregatorService`/`SchedulerService` no longer
   hold `*database.DB`. `harvest.New` now takes a **`Deps`** struct; the new
   `internal/app/harvest.go` composition root wires the store adapters and is
   called by both prod callers (`app.NewHarvest(cfg, db)`). `depguard`
   `harvest-no-database` bans `internal/database` in harvest production code
   (`!$test`; tests still open real SQLite). Golden HTTP suite unchanged; lint 0;
   `go test ./...` green. Added `aggregator_repo_test.go` (DB-free aggregator via
   a fake `MetricsRepo`). **The harvest module is now fully persistence-free.**

(Clock/IDGen ports: optional/low-value — `time.Now()` sprawls 7 files, mostly
presentational PDF/CSV stamps. Skip unless determinism is needed.)

### 4.7 ReportRepo execution recipe (turnkey for the fresh pass)

Goal: move report-record SQL out of the module behind a port; harvest depends on
an interface, the SQL lives in `internal/adapters/store`.

1. **Port** — `internal/modules/harvest/ports.go` (new):
   ```go
   type ReportRepo interface {
       GetReport(ctx context.Context, id string) (*Report, error)
       ListReports(ctx context.Context) ([]Report, error)
       SaveReport(ctx context.Context, r *Report) error
       DeleteReport(ctx context.Context, id string) error // row only
   }
   ```
2. **Adapter** — `internal/adapters/store/harvest_repo.go` (new pkg `store`):
   `type ReportRepo struct { db *database.DB }` + `NewReportRepo(db)`. Move the
   SQL + scanning verbatim from `services_reports.go` (`GetReport`/`ListReports`/
   `scanReport`/`scanReportFromRows`) and `services.go:saveReport`, plus the
   `DELETE FROM reports` from `DeleteReport`. Returns `*harvest.Report` (adapter
   imports harvest — correct inward direction). **Move logic, don't rewrite.**
3. **Service** — `GeneratorService` gains a `reports ReportRepo` field; its
   `GetReport`/`ListReports`/`saveReport`/`DeleteReport` delegate to it.
   `DeleteReport` keeps its `os.Remove(file)` orchestration. `GeneratorService`
   **keeps `db`** for now (export `devices`/`device_vulnerabilities` queries in
   `services_export.go` are a separate `ExportRepo` concern — slice 1b-v).
4. **Wiring** — `NewGeneratorService(cfg, reports, db, ts, as)`; `harvest.New`
   takes the repo and passes it (aggregator/scheduler stay on `db`). Build the
   adapter in the 3 callers: `internal/api/modules.go`, `cmd/seed/cmd_serve.go`,
   `cmd/seed/cmd_service_windows.go` → `store.NewReportRepo(db)`.
5. **Tests** — ~10 `NewGeneratorService(cfg, db, ts, as)` sites in
   `internal/modules/harvest/internal_test.go` (+ `services_test.go`) → pass
   `store.NewReportRepo(db)`; add `adapters/store` import. (External test pkg
   `harvest_test` may import adapters.) Optionally add a fake `ReportRepo` for a
   DB-free unit test of `GeneratorService` (the §4.5 payoff).
6. **Gates** — `go build ./...`, harvest unit tests, **golden HTTP suite** (the
   behavior gate), `gofumpt -w` the rewired importers (import-path moves trip
   import ordering), `golangci-lint` 0 issues. depguard stays green (still using
   the `internal/database` wrapper, not `database/sql`; the `internal/database`
   ban arrives in 1b-v after the store ring is fully populated).

---

## 5. Rollout order (cold → hot)

| # | Module | From | Risk | Notes |
|---|---|---|---|---|
| 1 | **harvest** | `internal/harvest` | low | pilot — proves the pattern |
| 2 | **roots** | `internal/pipeline/{analysis,enrichment,publicip,topology,traceroute}` | low-med | cold; **designs the flat `PathResponse` transport DTO deferred in Phase 2** + the `GatewayResponse` non-recursive split |
| 3 | **canopy** | `internal/canopy/{,channel,survey,wifi,data}` | med | 14 importers; warm — extract after pattern is set |
| 4 | **sap** | `internal/services/{link,cable,dns,gateway,iperf,vlan,speedtest,performance,telemetry,snmp}` + `health`,`probe` | high | 120 files — slice per sub-domain, many PRs; `alerts`/`polling`/`listener` settle here but are HOT now, so they land **last within sap** once their workstream quiets |
| 5 | **shell** | `internal/services/shell` + `internal/services/discovery` | high | the 24.5K discovery monolith is **Phase 6** (enumerate→resolve→fingerprint→vuln); coordinate |

Cross-module reads discovered during extraction (like harvest→health) become
query ports in Phase 3 and migrate to `platform/events` subscriptions in Phase 4.

---

## 6. Deferred: platform / adapters spine rehome

Moving `config`/`logging`/`i18n`/`license`/`auth`/`oauth`/`version`/`paths`/…
→ `platform/`, and `api`/`database`/`net` packages → `adapters/`, is **not** in
Phase 3's critical path and collides with the auth/oauth and alerts/polling
workstreams. Do it **after** the module extractions, **one leaf package at a
time**, each a mechanical `git mv` guarded by the golden tests, and **only after
checking the concurrent-session map**. `depguard` `platform-isolation` /
inward-only rules turn on per package as it moves.

---

## 7. Documentation gate (per blueprint §17)

Phase 3 PRs update, in lockstep:
`msn-docs-internal/02-The-Seed/THE_SEED_ARCHITECTURE.md` (+ `_BACKEND_ARCHITECTURE`),
`05-Engineering/CODING_STANDARDS.md`, `05-Engineering/architecture/platform-architecture.md`,
and finalize the supersede banners on the two legacy structure plans. Diagrams
(ASCII): dependency-direction ring, folder tree, request lifecycle through the
registry — refreshed as modules land.

---

## 8. Risks & mitigations

| Risk | Mitigation |
|---|---|
| Collision with hot zones (polling/alerts/discovery/auth) | strategy is skeleton-light + cold-first; spine rehome deferred (§6); re-check the §2.1 map before each PR |
| `ServiceContainer` ↔ `app` coexistence confusion | `app` grows additively; `ServiceContainer` retired only after the last module; documented in `app/` package doc |
| Hidden cross-module imports surface mid-lift | expected (harvest→health is the first) — convert to a query port, log it for the Phase-4 event migration |
| Behavior drift during a move | golden HTTP snapshots are the gate; a move PR that changes a snapshot is rejected until explained |
| Scope creep into sap/shell early | order is fixed cold→hot; sap/shell are explicitly last / Phase 6 |

---

## 9. Decision log

- **2026-06-01:** Pilot = `harvest` (lowest risk: cold, 1 importer, I/O-light).
  Approach = strangler, skeleton-light, spine rehome deferred. Plan-doc-first
  before any code moves. (Owner directive: best-practice code *and* architecture
  throughout.)
