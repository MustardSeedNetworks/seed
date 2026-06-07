# Seed Re-Architecture Blueprint

> ⚠️ **AMENDED 2026-06-01 — Phase 3 pivoted.** The "modulith hexagon" with one
> `internal/modules/<botanical>` package per product module (Roots/Canopy/Shell/
> Sap/Harvest) proved, during execution, to be *dead parallel wiring* the HTTP
> request path never consumed. Phase 3's live plan of record is now
> **`PHASE3_RECONCILE_PROPOSAL.md`**: a right-sized modular monolith — fat
> handlers + api `ServiceContainer` groupings → **capability-first** feature
> packages (`internal/wifi`, `internal/diagnostics`, `internal/security`,
> `internal/reporting`), one composition root, ports only at real I/O seams, and
> **descriptive code names** (botanical names retained for *marketing* only). The
> module facades have been deleted. Phases 0–2 (registry, code-first contract,
> golden harness) stand. Read the RECONCILE proposal for current direction.

**Product:** Seed (Network Diagnostics + Wi-Fi Troubleshooting + Security + Compliance)
**Owner:** Mustard Seed Networks
**Status:** AMENDED — Phase 3 superseded by PHASE3_RECONCILE_PROPOSAL.md (2026-06-01)
**Updated:** 2026-06-01
**Companion ADRs:** [`decisions/`](decisions/)
**Supersedes (structure):** the legacy seed/cross-repo structure plans in msn-docs — see [§17](#17-documentation-alignment-per-phase-gate)

> **Product boundary:** Seed = network discovery, monitoring, troubleshooting, security &
> compliance — **non-disruptive by default, IDS/IPS-friendly**. Seed does **NOT** implement
> RFC2544 / ITU-T Y.1564 active throughput testing (that is **Stem**). Where this blueprint
> mentions scans/jobs (discovery, vulnerability, port), they are **user-initiated diagnostics**,
> not passive-vs-active performance testing.

> This is the single source of truth for the Seed re-architecture. It supersedes
> ad-hoc structure decisions. The *shape* defined here is harmonized across
> seed / stem / niac (see [Harmonization](#13-harmonization-across-the-three-products));
> each repo owns its own implementation — there is **no master repo**.

---

## 1. Context & posture

Seed today: ~119K LOC Go backend (36 internal packages), ~79K LOC React/TS
frontend (339 files), ~120 HTTP routes, SQLite, SSE for live data, five product
modules (Roots / Canopy / Shell / Sap / Harvest). It works and is reasonably
disciplined (no `init()` side effects, explicit constructor wiring, pure-Go
SQLite, schema-drift + output-escaping CI gates, govulncheck hard gate).

It is **not** a rewrite-because-it's-broken situation. It grew organically and
the seams are now in the wrong places.

### Greenfield posture (decisive constraint)

**No customers, no clients yet → no backwards-compatibility burden.** This unlocks
clean breaks everywhere:

- Normalize the route table and gating order freely — no API to preserve.
- Collapse the 2,079-line migration history into one clean baseline schema.
- Delete dead code / deprecated aliases / no-op env vars on sight.
- Fix the crypto smell (JWT-secret-as-cipher-key) with no migration path.
- Design the API we *want*, not the one we have.

**Caveat that bounds the ambition:** "no customers" means *no back-compat*, **not**
*rewrite-from-scratch*. The 119K LOC is valuable because it is **debugged** — every
`// fixes #NNN` edge case is hard-won knowledge a clean-room rewrite would silently
lose. The play is **move-and-preserve the working logic, drop every compatibility
shim**. Strangler stays as a *safety* technique; it sheds its *compatibility* baggage.

**"main stays green" still holds** — not for customers, but for the **team**
(concurrent Claude/developer sessions run on this repo; a broken `main` blocks them).
Green now means "builds + tests pass," not "preserves the old API."

---

## 2. Goals & non-goals

### Goals
1. Move from "enforcement & contracts by *convention*" to "by *construction*."
2. Make the product's five-module mental model structural and consistent.
3. A pure, I/O-free domain core that unit-tests without HTTP or SQLite.
4. One frontend/backend contract, generated, never hand-maintained twice.
5. A harmonized structure shared across seed / stem / niac.

### Non-goals
- Microservices. Seed is and stays a **modulith** (one binary, one DB, one server).
- A magic DI framework. Wiring stays explicit, in one composition root.
- A rich product CLI for seed (see [§12](#12-cli-scope)).
- Distributed anything, phone-home anything (air-gapped market — see [§9](#9-observability-local-only)).

---

## 3. Principles (best practices we hold — and reject)

**Hold:**
- **Package by domain, not by layer.** No top-level `services/`, `models/`, `handlers/`.
- **`internal/`-only.** Seed is an application, not a library. Everything unimportable from outside.
- **Dependencies point inward.** Domain imports nothing from http/sql/infra. Enforced by `depguard` in CI ([§4.3](#43-dependency-rules-enforced)).
- **No `utils` / `common` / `shared` junk drawers.** Packages are named for what they provide.
- **No stutter.** `canopy.Service`, never `canopy.CanopyService`.
- **One concept per package.** The 24.5K-LOC discovery package and the `api` god-package both violated this; the new structure fixes both.
- **Events are facts, not commands.** ([§6](#6-cross-module-events))

**Reject (cargo-cult):**
- `golang-standards/project-layout`'s `pkg/` directory — folklore, not Go-team guidance. We stay `internal/`-only.
- Per-module `adapters/app/domain/ports` quadruple-nesting (microservice style) — ceremony without the isolation payoff for a single-binary modulith.

---

## 4. Target architecture — the modulith hexagon

### 4.1 The rings

```
                    ┌─────────────────────────────────────────────┐
                    │  contract/  (OpenAPI 3.1 — source of truth)  │
                    │  → generates Go transport DTOs + TS client   │
                    └───────────────┬─────────────────────────────┘
                                    │ codegen
   ┌────────────────────────────────┴───────────────────────────────┐
   │  adapters/  (infra ring)                                        │
   │  http (server, capability registry, middleware, handlers, ui)  │
   │  cli (thin/ops)   store (sqlite, repos, migrations, outbox)    │
   │  net (snmp, dhcp, pcap, netif, phy, mibdb)                      │
   └───────────────┬─────────────────────────────────────────────────┘
                   │ implements ports defined by ↓ ; calls services of ↓
   ┌───────────────┴───────────────────────────────────────────────┐
   │  modules/  (domain core — PURE: no net/http, no database/sql)  │
   │  sap · shell · canopy · roots · harvest                        │
   │  each: service.go (logic) + ports.go (interfaces it needs)     │
   │  modules NEVER import each other → talk via platform/events    │
   └───────────────┬───────────────────────────────────────────────┘
                   │ uses
   ┌───────────────┴───────────────────────────────────────────────┐
   │  platform/  (cross-cutting spine — same shape in all 3 repos)  │
   │  config logging i18n license auth version paths secret         │
   │  lifecycle (supervisor) · events (bus) · jobs (task runner)    │
   │  audit · validation                                            │
   └────────────────────────────────────────────────────────────────┘

   internal/app/  composition root: builds the whole graph, top → bottom
   cmd/seed/      thin entrypoint: app.New(); app.Run()
```

Domain logic lives in `modules/`. Infrastructure lives in `adapters/`. Both depend
on `platform/`. The `app` composition root wires them. `cmd/seed` is a shell.

### 4.2 Folder structure & migration map

```
seed/
├── cmd/seed/                 # thin cobra entrypoint
├── contract/                 # OpenAPI specs + codegen config (NEW)
├── internal/
│   ├── app/                  # composition root (NEW) — replaces ServiceContainer sprawl
│   ├── modules/              # the five product modules, first-class & consistent
│   │   ├── sap/              #   live telemetry (link, cable, dns, gateway, vlan,
│   │   │   ├── service.go    #   iperf, speedtest, perf, snmp) + health + jobs ingestion
│   │   │   ├── ports.go
│   │   │   └── {link,dns,health,...}/
│   │   ├── shell/            #   security posture + discovery pipeline + vuln
│   │   │   └── {enumerate,resolve,fingerprint,vuln}/
│   │   ├── canopy/           #   Wi-Fi visibility/troubleshooting (survey, wifi, channel)
│   │   ├── roots/            #   path analysis (traceroute, topology, enrichment)
│   │   └── harvest/          #   reporting / export / logs
│   ├── platform/             # cross-cutting spine (same shape in stem/niac)
│   │   ├── config/ logging/ i18n/ license/ auth/ oauth/ version/ paths/
│   │   ├── secret/           # typed Secret wrapper (NEW)
│   │   ├── lifecycle/        # Component supervisor (NEW) — collapses engine+orchestration+scheduler
│   │   ├── events/           # in-process domain event bus (NEW)
│   │   ├── jobs/             # unified async task runner (NEW)
│   │   ├── audit/            # append-only audit log (NEW)
│   │   └── validation/ constants/
│   ├── adapters/             # infra ring, grouped by technology
│   │   ├── http/             # server, route.go (registry), middleware, *_handlers.go, ui embed
│   │   ├── cli/              # thin ops adapter over core (license, export, setup, validate)
│   │   ├── store/            # sqlite, per-module repos, migrations/, outbox, timeseries
│   │   └── net/              # snmp, dhcp, pcap, netif, phy, mibdb
│   └── testutil/
├── ui/                       # frontend (unchanged top-level)
└── docs/architecture/        # this blueprint + ADRs
```

**Package migration map (current → target):**

> **SUPERSEDED (Phase 3 reconcile, 2026-06).** The `internal/modules/*` +
> `internal/adapters/*` hexagon rings below were **abandoned**. The backend is
> **capability-first**: flat `internal/<feature>` packages (e.g. `internal/harvest`,
> `internal/reporting`, `internal/discovery`), with ports applied as a technique at
> the I/O seam (e.g. `internal/reporting/store`) and `depguard` enforcing direction.
> This table is kept only as a record of the original plan. Concretely:
> `internal/services/discovery` relocated to **`internal/discovery`** (Phase 6 S3,
> #1489), *not* `internal/modules/shell/*`; `internal/harvest` stayed flat (not
> `internal/modules/harvest`). Treat the rows below as historical, not the target.

| Current | → Target |
|---|---|
| `internal/canopy/{,channel,survey,wifi,data}` | `internal/modules/canopy/…` |
| `internal/harvest/{aggregator,generator,scheduler,templates,data}` | `internal/modules/harvest/…` |
| `internal/services/{link,cable,dns,gateway,iperf,vlan,speedtest,performance,telemetry,snmp,dhcp}` | `internal/modules/sap/…` |
| `internal/services/shell` | `internal/modules/shell/` |
| `internal/services/discovery` | `internal/modules/shell/{enumerate,resolve,fingerprint}` + `…/vuln` |
| `internal/pipeline/{analysis,enrichment,publicip,topology,traceroute}` | `internal/modules/roots/…` |
| `internal/{health,probe,probe/checkers}` | `internal/modules/sap/health/…` |
| `internal/{listener,polling,alerts}` | `internal/modules/sap/…` (settled at extraction) |
| `internal/{config,logging,i18n,license,auth,oauth,version,paths,validation,constants,system,truststore,update}` | `internal/platform/…` |
| `internal/{engine,orchestration,scheduler}` | `internal/platform/lifecycle/` |
| `internal/api`, `internal/api/{data,ui}` | `internal/adapters/http/…` |
| `internal/database`, `internal/timeseries{,/retention}` | `internal/adapters/store/…` |
| `internal/{dhcp,protocols/snmp,netif,netif/detection,phy,mibdb}` | `internal/adapters/net/…` |
| `cmd/seed/cmd_*` (license/export/setup/validate logic) | thin cobra → `internal/adapters/cli/…` |

Debatable placements, settled at extraction time, not guessed now: `update`
(self-update — platform vs. its own concern), `alerts` (parked in `sap`),
`truststore` (TLS trust — platform vs. adapters/net).

### 4.3 Dependency rules (enforced)

Drop-in for the existing golangci-lint v2.12.1 config — makes "dependencies point
inward" a CI gate, not a hope:

```yaml
# .golangci.yml
linters-settings:
  depguard:
    rules:
      domain-purity:
        files: ["**/internal/modules/**"]
        deny:
          - pkg: "net/http"
            desc: "modules are pure: define a port, implement it in adapters/http"
          - pkg: "database/sql"
            desc: "modules are pure: define a Repo port, implement it in adapters/store"
          - pkg: "github.com/MustardSeedNetworks/seed/internal/adapters"
            desc: "inward-only: adapters depend on modules, never the reverse"
      module-independence:
        files: ["**/internal/modules/**"]
        deny:
          # each module is an island; cross-module talk goes through platform/events.
          # (enumerated per-module in practice: canopy may not import sap/shell/roots/harvest, etc.)
          - pkg: "github.com/MustardSeedNetworks/seed/internal/modules"
            desc: "modules must not import each other — publish/subscribe via platform/events"
      platform-isolation:
        files: ["**/internal/platform/**"]
        deny:
          - pkg: "github.com/MustardSeedNetworks/seed/internal/modules"
          - pkg: "github.com/MustardSeedNetworks/seed/internal/adapters"
```

> The `module-independence` rule needs per-module allow/deny tuning (a module
> imports its own sub-packages). Implement as one rule file per module directory.

Resulting direction: `adapters → modules ← (nothing infra)`, `adapters → platform`,
`modules → platform`, `platform → stdlib only`.

### 4.4 Composition root (`internal/app`)

The fix for the 60-field `ServiceContainer` and the
`initDatabaseDependentServices` / `initProbeEngine` / `initHealthServices` sprawl.
One readable file builds the graph top-to-bottom:

```go
// internal/app/app.go
func New(cfg config.Config) (*App, error) {
    log := logging.New(cfg.Logging)

    // 1. platform spine
    bus    := events.New()
    audit  := audit.New(...)
    jobs   := jobs.NewRunner(cfg.Jobs, bus)

    // 2. adapters (infra ring)
    db     := store.Open(cfg.DB)            // implements module Repo ports
    snmp   := net.NewSNMP(cfg.SNMP)         // implements module probe ports

    // 3. modules (domain), wired to ports — each gets ONLY its config slice
    sap    := sap.New(sap.Deps{Repo: db.Sap(), Probe: snmp, Bus: bus, Clock: clock.System, Cfg: cfg.Sap})
    shell  := shell.New(shell.Deps{Repo: db.Shell(), Bus: bus, Jobs: jobs, Cfg: cfg.Shell})
    // canopy, roots, harvest …

    // 4. transport adapters over the modules
    httpSrv := httpadapter.New(cfg.Server, registry(sap, shell, ...))

    // 5. lifecycle: register components, supervisor owns start/stop order
    sup := lifecycle.New(sap, shell, canopy, roots, harvest, jobs, httpSrv)
    return &App{sup: sup}, nil
}
```

Manual wiring (no magic), but in **one home** instead of ten.

---

## 5. Cross-cutting systems

### 5.1 Capability / route registry *(the keystone)*

**Problem killed:** today role-gating (`writeGated`), license-gating
(`requireFeature`), and rate-limiting are applied by *remembering* to wrap each
route, in inconsistent nesting order (Roots, Canopy, and Harvest each nest
differently). "Add a mutating route without the wrapper" is a documented
regression class. Auth + CSRF are already global (`server_lifecycle.go`) and stay so.

**Change:** routes become declarative data; one registrar composes middleware in
one fixed, audited order.

```go
// internal/adapters/http/route.go
type Route struct {
    Method    string          // "GET","POST"; "" = any
    Path      string
    Auth      AuthRequirement // Public | Authenticated (default Authenticated)
    MinRole   database.Role   // "" = none; RoleOperator = writeGated-equivalent
    Feature   string          // "" = none; "path_analysis" = requireFeature
    RateLimit RateClass       // None | Standard | Heavy
    Handler   http.HandlerFunc
    Module    string          // for the audit manifest
}

// register applies wrappers in ONE canonical order for EVERY route:
//   rateLimit → requireFeature → requireRole → handler
func (s *Server) register(rt Route) { /* … */ s.manifest = append(s.manifest, rt) }
```

**Deliverables:**
1. `Route` + `register` + canonical wrapper order (incidentally fixes the nesting bugs).
2. All ~120 routes converted to per-module tables.
3. `GET /__capabilities` (or build-time JSON) emitting the manifest — the fleet-audit surface.
4. **CI gate `check-route-policy.sh`:** fail if any handler is registered outside
   `register()`, or if a mutating route (`POST/PUT/DELETE/PATCH`) has neither
   `MinRole` nor an explicit `{Auth: Public}`. *This is where "forgot the wrapper"
   becomes impossible.*

Greenfield bonus: normalize the gating-order divergences directly — no preserve-and-flag ceremony.

### 5.2 Contract boundary — code-first (ADR-0003, amended)

**Problem killed:** most request/response types are hand-typed *twice* (the
~1,300 LOC Profile/Settings types worst of all) with no enforced link → silent drift.

**Change (corrected from the original OpenAPI-first plan — see ADR-0003):** there is
already a working, CI-gated **code-first** pipeline; the fix is *coverage*, not new
tooling.

```
Go DTO ──seed-schema──► docs/schemas/api/*.schema.json ──gen-types──► ui generated TS
   gated by check-schema-drift.sh + check-types-drift.sh
```

- **Go DTOs stay the single source of truth.** Widen `seed-schema`'s target list from
  6 → all request/response DTOs; TS regenerates; drift fails CI.
- **Delete hand-maintained TS twins** (`profile.ts`, `settings.ts`, hand-written
  `client.ts` call sites) as each DTO is generated.
- **OpenAPI is a deferred additive output**, not a rewrite: emit OpenAPI 3.1 from the
  capability manifest (§5.1) + the schemas only when there's a reader (Redoc docs or a
  third-party consumer). Non-breaking bolt-on, no rework. Mirrors the org's "API
  versioning deferred until 3rd-party needs arise."
- Rejected: hand-authored OpenAPI-first — discards working infra, adds a permanent
  spec-vs-code sync burden for an API that currently serves only our own frontend.

### 5.3 Component lifecycle (`platform/lifecycle`)

**Problem killed:** `engine` + `orchestration` + `scheduler` + per-module
`Start/Stop` + `api.Modules` overlap; startup order is correct by accident.

```go
type Component interface {
    Name() string
    DependsOn() []string
    Required() bool                 // false → degrade, don't crash (see §10)
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() HealthStatus
}
```

Supervisor: topological-sort start, reverse-order stop, aggregate health, single
scheduler. Subscribers (event bus) registered **before** publishers start, so
startup events aren't lost. Exposes `/livez` `/readyz`. Bounded, timeout-driven
graceful shutdown.

### 5.4 Error mapping discipline

Domain returns **typed** errors (`sap.ErrProbeUnreachable`) that know nothing about
HTTP. One table in `adapters/http` maps domain error → status + `ErrCode`. The
domain stays transport-ignorant; status codes are never chosen inside business logic.

### 5.5 Configuration — two tiers

Seed has **two** config sources that must not be conflated:

- **Bootstrap/infra config** — `seed.json` + env (flags > env > file > defaults),
  loaded once at boot, validated, secrets env-only. Immutable at runtime. Each module
  receives a small slice (`sap.Config`) with only its fields, built in the composition
  root — **never the god `*config.Config`**. Removes a fat coupling point; modules test
  with a 3-field literal.
- **Runtime settings** — user-editable preferences in the `settings` DB table, changed
  via the API at runtime (thresholds, display options, feature toggles). Owned by the
  relevant module behind a repo port; never read from the file.

Rule: anything an operator edits in the UI is a *runtime setting* (DB); anything needed
to start the process is *bootstrap config* (file/env).

### 5.6 Secrets (`platform/secret`)

A typed `Secret` whose `String()` / `MarshalJSON` return `[REDACTED]` — leaking a
secret becomes a compile-time-shaped mistake, not a dependency on log redaction.
Wraps JWT secret, SNMP creds, NVD key, OAuth secrets. Fixes the
JWT-secret-as-cipher-key smell with a dedicated, rotatable key (greenfield → no migration).

### 5.7 Background-work taxonomy (three kinds, three homes)

Seed runs work that isn't request/response. Today five subsystems (`scheduler`,
`polling/snmp`, `listener/{syslog,snmptrap,sink}`, `harvest/scheduler`,
`timeseries/retention`) are conflated under "the scheduler." They are **three distinct
kinds** and must not share a model:

| Kind | What | Home | Lifecycle |
|---|---|---|---|
| **One-shot async** | user-initiated scans/tests (speedtest, discovery, vuln, survey, traceroute) | `platform/jobs` (§8) | created per request, runs to completion, emits events |
| **Recurring scheduled** | retention cleanup, scheduled reports, health-check cadence | `platform/scheduler` (cron-like) | registered at boot, fires on schedule, each tick may enqueue a job |
| **Continuous ingestion** | SNMP polling, syslog + snmptrap listeners | supervised `Component`s (§5.3) in their module (`modules/sap`) | run for the whole process lifetime, ingest external data, emit events |

Rule: jobs are *finite and user-initiated*; the scheduler is *recurring and time-driven*;
ingestion daemons are *continuous and externally-driven*. All three are owned by the
component supervisor and publish to the event bus; none reach across modules directly.

### 5.8 Request lifecycle & validation

A handler in `adapters/http` is a thin pipeline with named stages — no business logic:

```
decode (generated DTO + OpenAPI request validation)   ← syntactic validation
   → map DTO → domain input
   → core call (modules/<m>.Service method, ctx-first)  ← semantic validation lives in the domain
   → map domain error → status + ErrCode (§5.4)
   → encode (generated DTO, consistent envelope)
```

- **Syntactic validation** (shape, types, required, ranges) is generated from the
  OpenAPI spec and runs at the boundary — build on the existing `internal/api/decode.go`.
- **Semantic validation** (business invariants) lives in the domain (`modules/*`), never
  in the handler.
- **Conventions** (consolidated, enforced by Spectral §5.2): one response envelope, RFC3339
  timestamps, cursor pagination + `limit`/`filter`/`sort` params, idempotency keys on
  job-creating POSTs, ETag/`If-Match` on mutable resources (§7).

---

## 6. Cross-module events (`platform/events`)

Decision: **in-process domain event bus** (ADR-0004). Modules never import each
other; they publish/subscribe typed events.

**Semantics (load-bearing — specify or it becomes hidden coupling):**
- **In-process, single binary** — no broker.
- **Events are facts, past-tense** (`DeviceDiscovered`, `SurveyCompleted`) — *never*
  commands. Rule: events notify/react; they do not request/respond. This is the
  guardrail against spaghetti control-flow.
- **Async, at-least-once, ordered per-topic.** A panicking subscriber must not fail
  the publisher.
- **Ephemeral by default**; **audit events are durable** (→ `platform/audit`).
- **Subscribers registered before publishers start** (supervisor ordering).
- **Transactional outbox** (→ [§7](#7-persistence)): events emit only *after* the DB
  commit, so a rolled-back op never fires an "it happened" event.

Example fan-out: `shell` emits `DeviceDiscovered` → `sap/health` starts monitoring,
`harvest` records it, `sap/alerts` evaluates thresholds — with zero import edges
between modules.

---

## 7. Persistence (`adapters/store`)

> **STATUS (2026-06-07): Phase 5b — schema modernization — is DONE; two items
> remain.** Done: `.sql` migrations embedded via `//go:embed` + the **goose**
> runner (`internal/database/goose.go`), the **collapsed `0001_init.sql`
> baseline** (61 STRICT tables), the migrate-from-empty drift gate
> (`goose_baseline_test.go` / `schema_snapshot_test.go`), and a `WithTx`
> transaction wrapper (`database.go`). **Remaining (feature-sized):**
> (1) the **transactional outbox relay** (the `00002_jobs.sql` + `platform/events`
> comments both note it is "layered on" later) and (2) **optimistic concurrency**
> (version/ETag + `If-Match`) on the mutable resources — not yet implemented.
> The repo-interfaces-as-domain-ports point below is piloted in
> `internal/reporting/store` (ReportRepo/ScheduleRepo/MetricsRepo/ExportRepo);
> generalizing it to all 21 repos is deliberately deferred — done bespoke per
> domain only where a use-case needs it (the ADR-0016 strangle), not wholesale
> (which just produces near-identical adapters, as the polling-targets attempt
> confirmed).

- Repo **interfaces defined in the domain** (`modules/*/ports.go`), implemented in
  `adapters/store`. Eager-constructed at boot (kills the lazy-init race). *(Piloted
  in `reporting/store`; generalize on demand, not wholesale.)*
- **UnitOfWork / Tx** abstraction for multi-table atomicity. *(`DB.WithTx` exists;
  a typed UnitOfWork over the domain ports is the open generalization.)*
- **Transactional outbox** table: domain writes + the event row commit together;
  a relay publishes to the bus post-commit. *(OPEN — the durability layer for
  `platform/events`.)*
- **Optimistic concurrency:** version/ETag + `If-Match` on mutable resources
  (config, profiles, settings) — multi-user is live; last-write-wins loses data.
  *(OPEN — the highest-value remaining correctness item.)*
- **Reference data** (OUI vendor DB, MIB defs, default config, NVD cache) is
  **embedded read-only**, never rows in the mutable DB.

### 7.1 Schema & migrations (ADR-0006)

> **✅ IMPLEMENTED (Phase 5b).** The whole target below shipped: `.sql` files under
> `internal/database/migrations/` embedded via `//go:embed`, the **goose** runner
> (`goose.go`), a collapsed **`0001_init.sql`** baseline with **STRICT** tables +
> explicit FK/CHECK/UNIQUE constraints, and the migrate-from-empty drift gate. The
> homegrown index+1 runner and the Go-string migrations are gone. The paragraph
> below is the original problem statement, kept for historical context.

Today the schema lives as raw SQL **inside Go string literals** (`migrations.go`,
2,190 lines, ~40 tables; up-only homegrown runner; version = slice index+1). The
*connection setup* (`database.go`) is already sound and is **ported verbatim**:
`foreign_keys=ON`, `journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout`,
`cache_size`, `temp_store=MEMORY`, pool limits, WAL-checkpoint on close, auto-rebuild.

Target (best practice):

- **`.sql` files, not Go strings.** One file per migration —
  `internal/adapters/store/migrations/NNNN_description.sql` — embedded via `//go:embed`
  so the binary stays self-contained (same mechanism as the UI embed). Gains SQL
  tooling, linting (`sqlfluff`/`sqlc`), diffability, DBA reviewability.
- **Runner: [`goose`](https://github.com/pressly/goose)** (pinned) over the embedded
  `embed.FS` — up + down, a versioning table, and a `status` command. Replaces the
  homegrown index+1 scheme.
- **Schema collapse (greenfield):** flatten the 2,190-line history into a single clean
  `0001_init.sql` baseline (no production data to preserve). The fragile hand-written
  table-rebuild migrations (`PRAGMA foreign_keys=OFF; CREATE …_new …`) disappear into it.
- **STRICT tables** (SQLite 3.37+) throughout the baseline — enforces declared column
  types instead of SQLite's permissive default. Free safety upgrade in a clean baseline.
- **Explicit constraints in the baseline** — FKs on every relationship, `CHECK` for
  enums/states, `NOT NULL` discipline, `UNIQUE` where implied; standardized timestamps
  (one of TEXT ISO-8601 *or* INTEGER epoch, uniformly).
- **Data-model review:** straighten the denormalized tables (the 1,157-line
  `repository_discovery.go` hints at it) in the baseline rather than carrying them forward.
- **Forward-only in production**, down-migrations for dev/rollback; no business logic in
  migrations; a "migrate-from-empty → assert schema" test gates drift.

### 7.2 Artifacts & file storage

Generated and uploaded binary artifacts — Harvest reports (PDF/CSV), Canopy floor-plan
images, AirMapper imports — are **files on disk under the data dir, not DB blobs**:

- Stored under `${data dir}/artifacts/<kind>/<id>`; the DB holds only metadata + path.
- Uploads size-limited (existing `bodyLimitMiddleware`) and **path-traversal-safe**
  (validated IDs, never user-supplied paths joined raw).
- Downloads **streamed** (no full-buffer), produced as job kinds (§8) when expensive.
- Temp files cleaned on a `platform/scheduler` (§5.7) retention tick.
- Owned by the producing module behind a `BlobStore` port → testable, swappable for an
  object store later without touching the domain.

---

## 8. Unified async jobs (`platform/jobs`)

Decision: **one task runner** (ADR-0005) replacing the ~15 bespoke run/status/cancel
endpoints (speedtest, iperf, discovery engine, vuln scan, survey, traceroute, pipeline).

```go
type Job struct {
    ID       string
    Kind     string          // "speedtest" | "discovery.scan" | "vuln.scan" | …
    State    State           // queued | running | succeeded | failed | cancelled
    Progress float64
    Result   any
}
```

**Semantics:**
- **Persisted** (survive restart, or fail cleanly); retention + cleanup.
- **Bounded concurrency** (max parallel scans); **reject with 503 under overload** —
  appliance backpressure, not an unbounded queue.
- **Cancellation via ctx**; progress reporting; result stored in `store`.
- **Job authz** is role-level, consistent with the registry.
- **Jobs emit events** on state change → the frontend subscribes to **one** job/event
  SSE stream (not per-card).

Uniform surface: `POST /jobs` (kind + params), `GET /jobs/{id}`, `DELETE /jobs/{id}`
(cancel), one SSE stream. Collapses ~15 endpoints into one model with consistent
cancellation, progress, and timeouts.

**Status — implemented in Phase 4 (2026-06-02).** Runner core (`internal/platform/jobs`),
the HTTP surface (`/api/v1/jobs` + `/api/v1/jobs/events` SSE), and **7 long-op kinds**
(speedtest, iperf, vuln-scan, engine-scan, bluetooth-scan, wifi-discovery-scan,
device-scan) are on `main`. Each kind is an additive thin wrapper over the existing
ctx-aware service behind an interface seam; the legacy run/status endpoints are retained
until the frontend consumes `/jobs` (Phase 7), at which point they retire. Carve-outs:
**persistence is in-memory v1** (durable store + transactional outbox → Phase 5);
**survey stays a session** (not a one-shot job); **pipeline is deferred to Phase 6** —
it duplicates the discovery `engine` (the canonical DeviceRegistry-as-SSoT orchestrator,
already the `engine-scan` kind), so engine↔pipeline consolidation folds into the Phase-6
discovery split rather than being enshrined as a kind. See ADR-0005 for detail.

---

## 9. Observability (local-only)

Constraint: **no phone-home** (air-gapped clinical / industrial / government market).
Bake it in so no one wires hosted APM later:

- **`/metrics`** OpenMetrics endpoint (local scrape only).
- **OTel spans** across the component graph, exported **only** to a local/opt-in
  collector — never an SaaS default.
- **`/livez` `/readyz`** from the supervisor's aggregate health.
- **Structured event taxonomy** documented as a catalog (`auth.*`, `scan.*`,
  `license.*`, `job.*`) so fleet SIEM rules are a spec, not tribal knowledge.

---

## 10. Multi-platform & degraded operation

Seed runs macOS / Linux / Windows with hardware access (Wi-Fi, pcap, raw sockets,
service install). OS-specific code lives behind ports in `adapters/net`, with
build-tagged implementations.

**Optional / degraded modules:** Canopy needs Wi-Fi hardware; the dev-srv boxes are
wired servers. At startup each module probes its capability behind a port; absent →
the supervisor marks it **degraded (not crashed)** via `Component.Required() == false`,
and its routes return a clean `503 capability_unavailable` instead of 500.

---

## 11. Audit log (`platform/audit`)

Distinct from operational `slog`. **Append-only, tamper-evident** record of
security-relevant actions (login, config change, user mgmt, license activation, scan
initiation). Table stakes for the compliance/air-gapped target market. Durable
sink for the bus's audit-class events. Queryable.

---

## 12. CLI scope

Seed is **web-UI-first**. There is **no product CLI** (`seed scan`/`seed wifi` would
be worse versions of the UI). But the CLI is not deletable — part of it is the
**operational/lifecycle surface** that cannot live in the web UI:

| Command | Why it can't be web-only |
|---|---|
| `seed serve` | the launcher — no server to serve a UI until it runs |
| `seed install` / `uninstall` / `service` | systemd / launchd / Windows service registration |
| `seed install-ca` | trust the self-signed cert *before* the browser loads the UI |
| `seed setup` / `reset` | first-run admin bootstrap / factory reset |
| `seed license activate` / `trial` | air-gapped boxes activate offline at the terminal |
| `seed validate` | config check without booting |

Rule: **the CLI stays thin and operational, not a domain mirror.** Where an ops
command touches domain logic (`license`, `export`, `setup`), it goes through core via
`adapters/cli` for parity — but it does *not* expose scans/surveys/telemetry.

Harmonization note: `adapters/cli` is **thin in seed**, **thick in stem/niac**
(genuinely CLI-first products — `stem test -t throughput`). Same slot, product-appropriate weight.

---

## 13. Harmonization across the three products

The spine and rings are the **same shape** in all three; only `modules/` differs.
Per the standing rule — *harmonization, not a master repo* — the *shape* is shared
and documented; each repo *implements* its own.

| Path | seed | stem | niac |
|---|---|---|---|
| `internal/modules/` | sap, shell, canopy, roots, harvest | reflector, benchmark, servicetest, trafficgen, measure, certify | protocols/{arp,dhcp,dns,bgp,ospf,snmp} |
| `internal/platform/` | config, logging, license, auth, lifecycle, events, jobs, audit, version | *(identical set)* | *(identical set)* |
| `internal/adapters/` | http, cli, store, net | http, cli, store, net (+ `src/dataplane` C stays separate) | http, cli, store, net |
| `contract/`, `cmd/<p>/`, `ui/` | ✓ | ✓ | ✓ |
| `adapters/cli` weight | thin (ops) | thick (product) | thick (product) |

Land each pattern in seed first as the reference implementation, then port the
*pattern* (not the file) to stem and niac.

---

## 14. Frontend architecture

Mirror the backend rigor:

- **Generated typed client** from the `contract/` OpenAPI — delete hand-written `client.ts` and hand-maintained types.
- **React Query as the single server-state layer.** Zustand for UI-only state; finish retiring the legacy `profileContext`.
- **SSE is the single realtime transport.** WebSocket is not used as a backend transport (only `/events` SSE exists); the stale `/ws` reference in `CLAUDE.md` is retired. **One SSE manager → React Query cache** (the unified job/event stream from §8). Cards just `useQuery`; no per-card `EventSource` dance.
- **Route-level auth/role guards** sourced from the **same capability manifest** as the backend registry — single source of truth for "who can see this."
- **Error boundaries + suspense per module** so one card's failure doesn't blank the dashboard.
- **Design-token SSoT** — lands the existing token-consolidation / brand-token-map work; the re-arch executes it, doesn't relitigate it.
- **Form/validation via Valibot** mirroring the contract schemas; **optimistic updates** through React Query.
- **Per-module route code-splitting** (lazy `import()`); enforce the existing `lighthouserc` budget in CI.

---

## 15. Testing strategy

- **Unit** — domain (`modules/*`), zero I/O, **hand-written fakes for ports** (not mock frameworks), table tests.
- **Integration** — adapters against a real in-memory SQLite; `net` adapters against fakes.
- **Contract** — validate handlers against the OpenAPI spec (request/response conform).
- **Golden / characterization** — snapshot `(status, headers, body)` for representative routes; **our** refactor-safety net (greenfield → update freely when current behavior is wrong).
- **Arch-fitness** — the route-policy CI gate (runtime side of depguard).
- **E2E** — Playwright, `getByTestId` selectors (per house rule).
- **Determinism** — `Clock` port injected everywhere; Go 1.26 `testing/synctest` for time-dependent logic (retention, SLA, health windows).
- **Hard CI gates** — `go test -race`; benchmark-regression on hot paths (packet/discovery) per `PERFORMANCE_GUIDELINES.md`; the existing schema/output-escaping/govulncheck gates retained.

---

## 16. Migration plan — phased, strangler, greenfield-bold

Phases ordered by risk/reward; each is independently valuable and ships behind the
protected-branch flow. Greenfield → bigger PRs are fine (slice for reviewability,
not back-compat). Each behavior-preserving phase rides on the golden tests.

| Phase | Outcome | Effort | Risk | Depends |
|---|---|---|---|---|
| **0** Prep & hygiene | purge tracked binaries (146MB `seed`, 157MB embed dir, `coverage.out`); golden-test harness; route inventory; **skeleton + spine rehome** (mechanical `git mv` of unambiguous leaf pkgs → `platform/`, `adapters/`) | 0.5 wk | none | — |
| **1** Capability registry | "forgot the wrapper" impossible; `/__capabilities` manifest; `check-route-policy.sh` | 1–1.5 wk | low | 0 |
| **2** Contract-first (per module) | one generated source of truth; Spectral lint; Redoc | 1.5 wk + roll | low-med | 0 |
| **3** Domain core / hexagon | logic testable without I/O; modules move to `modules/` as extracted | 2 wk + roll | med | 1,2 |
| **4** Lifecycle + events + jobs | one supervisor; event bus; unified job runner | 1.5 wk | med | 3 (partial) |
| **5** Persistence ports + schema collapse | UnitOfWork + outbox; single baseline schema; data-model review; optimistic concurrency | 1–1.5 wk | low-med | 3 |
| **6** Discovery pipeline split | 24.5K monolith → enumerate→resolve→fingerprint→vuln | 1 wk | med | 3,5 |
| **7** Frontend coherence | generated client + RQ + single SSE/job stream + code-split | 1.5 wk | low-med | 2,4 |

**Folder migration is not a big-bang.** Phase 0 establishes the skeleton and moves
only unambiguous leaf packages (mechanical, golden-test-guarded). Domain modules move
*with* their Phase-3 extraction (never twice). `depguard` rules turn on per-directory,
warn→deny, as each package is cleaned.

**Lowest-regret stop point:** after Phases 1–2 the two largest latent-defect surfaces
(enforcement + contract drift) are closed. If appetite is limited, stop there.

**Every phase is TDD-gated:** the "failing test" is the golden snapshot (or a new
unit test for new behavior) that must pass before the phase is done.

---

## 17. Documentation alignment (per-phase gate)

Documentation is a **phase gate, not an end-of-project pass**: each phase's PR updates the
msn-docs it invalidates and adds/updates diagrams, so docs never lag the code by more than
one phase. (Decisions, 2026-05-31: *blueprint supersedes prior plans*; *per-phase gate cadence*.)

**Why this blueprint supersedes the prior plans:** modern software should be built on current
best-practice design — ports-and-adapters, contract-first boundaries, policy-by-construction,
event-driven decoupling. The legacy plans normalized the *existing* layer-grouped layout
(`api/`, `services/`, `pipeline/`) or a flat by-module variant; neither carries the hexagon,
the registry, contract-first codegen, the event bus, or unified jobs. The blueprint does, so
it is canonical:

- `msn-docs-internal/05-Engineering/PROJECT_STRUCTURE_MIGRATION_PLAN.md` — **superseded for structure** (cross-repo; migration mechanics retained). Banner added 2026-05-31.
- `msn-docs-internal/05-Engineering/REFACTOR_PLAN.md` — **superseded** (flat-by-module target). Banner added 2026-05-31.
- `msn-docs-internal/02-The-Seed/THE_SEED_ARCHITECTURE.md` v2.0 — **rewritten to match at Phase 3** (forward-pointer added 2026-05-31).

**Per-phase msn-doc sync map:**

| Phase | msn docs to align |
|---|---|
| 0–1 registry | `API_DESIGN_GUIDELINES`, `SECURITY_ARCHITECTURE`, `CI_CD_CONFIGURATION`, `architecture/product-boundaries` |
| 2 contract | `API_DESIGN_GUIDELINES`, `API_REFERENCE`, `ERROR_HANDLING` |
| 3 hexagon | `THE_SEED_ARCHITECTURE` (+`_BACKEND_ARCHITECTURE`), `CODING_STANDARDS`, `architecture/platform-architecture`; finalize supersede of the two legacy plans |
| 4 lifecycle/events/jobs | `architecture/platform-architecture`, `MONITORING_ALERTING`, `LOGGING_STANDARDS` |
| 5 persistence | `DATA_MODEL` (schema v13 → single baseline), `DATABASE_MIGRATIONS` |
| 6 discovery | `DISCOVERY_ENGINE_ARCHITECTURE` + `discovery_reference/` |
| 7 frontend | `THE_SEED_UI_ARCHITECTURE`, `COMPONENT_CATALOG`, `E2E_CONVENTIONS` |

**Diagrams:** ASCII, to match the existing msn-docs convention (zero Mermaid in the repo
today). Standard set produced/refreshed per phase: dependency-direction ring, component +
lifecycle graph, request lifecycle through the registry, event/job flow, folder tree. The
harmonized **cross-repo shape** is canonicalized in `msn-docs-internal/05-Engineering/architecture/`;
each repo's `docs/architecture/` holds its own instance (no master repo).

---

## 18. Open items (implementation detail, decided *in* the work)

Deliberately **not** in this blueprint — these are detail, not architecture:
specific rate-limit numbers per `RateClass`, exact baseline-schema columns,
individual `ErrCode` values, component-test fixtures, the per-module `depguard`
allow-lists, the exact event catalog entries. Each is decided in its phase.

---

## 19. Decision log (ADRs)

| ADR | Decision |
|---|---|
| [0001](decisions/0001-modulith-hexagon.md) | Modulith hexagon (modules / platform / adapters), not microservices, not layer-first |
| [0002](decisions/0002-capability-registry.md) | Capability registry — authz/feature/rate-limit by construction |
| [0003](decisions/0003-contract-first-boundary.md) | Contract boundary — code-first (Go DTOs → schema → TS); OpenAPI deferred (amended) |
| [0004](decisions/0004-event-bus.md) | In-process domain event bus for cross-module comms |
| [0005](decisions/0005-unified-jobs.md) | Unified async job runner replacing per-feature run/status/cancel |
| [0006](decisions/0006-migrations-sql-goose-strict.md) | Schema as embedded `.sql` files, run by goose, STRICT tables, single baseline |
