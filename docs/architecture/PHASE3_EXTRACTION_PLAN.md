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

- [ ] `go test ./...` green; **harvest report/export golden snapshots unchanged**.
- [ ] `internal/modules/harvest` imports no `net/http`, no `database/sql`, no
      `internal/adapters/**`, no sibling `internal/modules/*` — `depguard` at `deny`.
- [ ] harvest's logic has table-driven unit tests that run with **fake ports**
      (no DB, no filesystem) — the payoff the phase exists for.
- [ ] `internal/app/harvest.go` builds the module from `Deps`; `internal/api/modules.go`
      consumes the module through the same surface (no behavior change).
- [ ] The harvest→health coupling is a `HealthSource` port, not a direct import.
- [ ] Docs synced (§7): `THE_SEED_ARCHITECTURE` harvest section + folder-tree + ring diagram.

### 4.6 PR slicing (each green, admin-merged)

1. **Scaffold + ports (no behavior change):** create `modules/harvest/` with
   `service.go`+`ports.go`, move the pure logic, define the ports; `adapters/store`
   + `adapters/http` shims implement/consume them; `depguard` rule at `warn`.
2. **Cut the cross-module health import** to `HealthSource`.
3. **Fake-port unit tests** for aggregator/generator/scheduler/service.
4. **Flip `depguard` to `deny`**; retire harvest's slice of `ServiceContainer` into
   `app/harvest.go`; doc sync.

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
