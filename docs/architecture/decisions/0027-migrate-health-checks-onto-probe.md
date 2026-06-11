# ADR-0027: Migrate the on-demand health-check stack onto the probe engine, then rename the transport

**Status:** Accepted — 2026-06-11 (scoping ADR; P1–P3 + P4 implemented; only the P5 transport rename remains)

> **P3+P4 cutover as-built (2026-06-11).** `/telemetry/health-checks/run` now dispatches the
> operator's configured probes through `Engine.RunNow` (load from the `probes` table → dispatch
> → persist → same breach/anomaly path a scheduled run uses, ADR-0025 §1) and maps the probe
> `Result`s into the card's shape. The ~3,600-LOC legacy parallel stack is **deleted**: the seven
> `health_checks_*.go` protocol files, the three `handlers_{medical,enterprise,industry}_checks.go`
> vertical files, the `run*Tests()`/`Run*Checks()` methods, the `CustomTestResult`/`CustomTestsResult`
> DTOs, and the P2 `hydrateHealthCheckTargets` shim. `handlers_health_checks.go` shrank to the shared
> status vocabulary + `getTestStatus` the rest of the API package still uses.
>
> **P4 came nearly free** — a deliberate scope choice. The card fetches `/run` and casts the JSON
> directly to its hand-written `HealthCheckData` type, so rather than reshape the wire + rework the
> card (the ADR's original P3/P4 split, which would have left the card broken between PRs), the new
> Go DTO (`HealthCheckRunResponse` in `healthcheckrun.go`) mirrors `HealthCheckData` field-for-field
> and the card is untouched. A card-consumption audit drove the DTO to emit **only the fields the card
> actually renders** (host/port/url, min/maxLatency, certCommonName, and the never-populated vertical
> detail fields were dropped). This also **fixed a pre-existing drift**: the old backend emitted
> `industryResults` + top-level `dicomResults`/`rtspResults`, but the card reads `industrialResults`
> + nested `medicalResults.dicomResults`/`videoResults.rtspResults` — those sections were silently
> empty before and now populate.
>
> No regression: an audit confirmed the vertical "rich" fields the probe checkers don't emit
> (rtsp codec/resolution, dicom serverAeTitle, opcua productName/serverState, fileshare IO, etc.)
> were **never populated by the legacy backend either** (they were declared-but-empty card fields, or
> driver-dead/heuristic strings). The HTTP per-phase timings + cert summary the card genuinely
> rendered are covered by P3a. Per-phase/cert/overall **status derivation** (the `getTestStatus`
> threshold logic) was ported into the `/run` mapping. The metadata read-types live in
> `internal/probe/checkers/metadata.go` (the package that owns the snake_case metadata format), not
> the API layer, so the camelCase wire-tag gate stays clean.
>
> **Deferred (noted, not silent):** the `config.HealthChecks` endpoint *lists* are now loaded-from-file
> but unread (settings + /run both read the `probes` table); removing them needs splitting the
> `HealthChecksConfig` type (still the settings transport shape) and is a separate cleanup. P5 (the
> `/telemetry/health-checks/*` → `/telemetry/probes/*` transport rename) is the only remaining phase.

> **P3a checker-enrichment as-built (2026-06-11).** A read-only audit before the P3 cutover
> found that the probe checkers emit thinner `Result.Metadata` than the legacy `/run` surfaced
> on the health-check card — so rewiring `/run` onto the engine as-is would have **regressed**
> the rendered diagnostics. (This also corrects the P1 note below: the HTTP checker was *not* a
> faithful port — it was a status/body-match reachability check.) P3a closes the gap for the
> dominant case: the HTTP/HTTPS checker now publishes a **per-phase timing breakdown**
> (`timings_ms`: dns/tcp/tls/ttfb via `httptrace`) and, for HTTPS, a **leaf-cert summary**
> (`tls`: CN/issuer/not-after/days-remaining/version, reusing the TLS checker's
> `extractCertInfo` so one cert path serves both kinds). The dedicated `tls` checker already
> emitted the cert block, and the P1 vertical checkers already emit their key protocol fields
> (HL7 `ack_code`, FHIR `fhir_version`/`resource_count`, Modbus `register_value`, etc.), so
> after P3a the probe metadata carries the rich data the card renders. **Deliberate non-ports**
> (documented, not silent regressions): (a) extended-ping loss/jitter/min/max — a multi-sample
> *on-demand snapshot* concept that does not belong in a continuously-scheduled probe (the
> engine samples once per interval; jitter/loss emerge from the time series, or from a future
> `/run` multi-sample mode, not from N rapid dials baked into the checker); (b) deep vertical
> fields (SQL driver-version/query timing, LDAP bind/search/entries, OPC-UA session state,
> FileShare read/write IO) — these need a real driver/library or a stateful protocol session
> and were already dead/stub in the legacy stack (see the P1 note), so omitting them is honest.
> Cert *status* (`success`/`warning`/`error`) stays a consumer concern (it needs the config
> expiry thresholds), derived in the P3/P4 mapping rather than baked into the checker.

> **P2 as-built (2026-06-11).** The `/telemetry/health-checks/settings` endpoint is now
> store-of-record backed by the `probes` table for all fourteen health-check kinds.
> `internal/api/healthcheckmapping.go` maps each `config.*Endpoint` ⇄ a `database.Probe`
> (identity in the `display_name`/`target`/`enabled` columns, the full endpoint JSON in
> `params_json`). GET reads via `loadHealthCheckEndpoints`; PUT flattens the request via
> `requestEndpointTargets` → `healthCheckProbesFromConfig` and persists with the transactional
> `ProbeRepository.ReplaceProbesByKinds` (delete-then-insert scoped to the fourteen kinds, so
> DNS/TLS/HTTPS probes are untouched), then calls `Engine.Reschedule`. Targets are now
> continuously monitored rather than on-demand only. The pre-P2 save path silently dropped the
> eight vertical kinds — they now persist. The unread `criticality` field (dead since ADR-0026
> deleted health scoring) was **removed outright** rather than given a home. `/run` is fed from
> the probes table via a thin best-effort hydrate snapshot; P3 deletes that. No schema
> migration was needed. (Landed in #1642.)

> **P1 as-built (2026-06-11).** The eight vertical checkers landed as `probe.Checker`
> implementations under `internal/probe/checkers/` (`hl7.go`, `fhir.go`, `sql.go`,
> `fileshare.go`, `ldap.go`, `lti.go`, `opcua.go`, `modbus.go`), each registered in the
> engine in `server.go`. HL7 (MLLP ACK), FHIR (CapabilityStatement), LTI (HEAD), OPC-UA
> (Hello probe), and Modbus (read-register ADU) port the legacy protocol behavior faithfully.
> SQL, FileShare, and LDAP port the **honest working** behavior only — TCP/TLS reachability —
> because the legacy `sql.Open`/driver, SMB/NFS local-mount perf test, and LDAP bind/search
> paths were dead or library-dependent (no driver/library is imported in the module). Those
> deeper capabilities are deferred and noted in each checker's doc comment; adding a real
> SQL driver or LDAP library is a separate, security-scanned dependency decision.
> No new threshold field or catalog `Def` was needed: `success`→`probe-unreachable` and
> `latency_ms`→`probe-high-latency` already cover the breach side (ADR-0025), and protocol
> detail (ACK code, FHIR version, register value) is surfaced as informational `Result.Metadata`.
> P2–P5 (settings storage, `/run` rewire + legacy deletion, frontend, transport rename) follow.

## Context

ADR-0025 made `internal/probe` the recurring-observation engine and the active-monitoring
anomaly producer; ADR-0026 deleted the dead health-check *read* path (scoring/SLA/results).
Both deliberately **kept three live routes** under `/telemetry/health-checks/*` and deferred
renaming the transport family until the survivors moved onto probe:

- `GET|POST /telemetry/health-checks/run` — run all configured checks **now** and return the batch.
- `GET|PUT  /telemetry/health-checks/settings` — the check **target** configuration.
- `GET      /telemetry/health-checks/anomalies` — already probe-backed (`source=probe`), since ADR-0026.

A read-only audit (2026-06-11) established what the two unmigrated routes actually are:

- **`/run` is a ~4,067-LOC legacy parallel stack** (13 `internal/api/health_checks_*.go` +
  `handlers_*_checks.go` files), entirely independent of `internal/probe`. It **re-implements** six
  protocols the probe checkers already cover (ping/tcp/udp/http/rtsp/dicom — `checkers/ping.go` even
  documents itself as "mirrors `health_checks_ping.go`") **and adds eight verticals with no probe
  checker at all**: HL7, FHIR, SQL, FileShare, LDAP, LTI, OPC-UA, Modbus.
- **`/settings` persists to the config JSON file** (`config.Config.HealthChecks`), not the `probes`
  SQLite table the probe engine uses. The two storage systems do not overlap. The per-endpoint
  `criticality` field ADR-0026 flagged as unread lives here.
- There is **no `/telemetry/probes/*` transport** — `/telemetry/health-checks/*` is the only HTTP
  surface for both the legacy on-demand checks and the probe-backed anomalies.

This is not the "small rewire" the roadmap assumed. It needs a decision and a phased plan before code,
which is what this ADR provides. The direction is already constrained by two prior decisions:

- **The probe package charter** (`internal/probe/doc.go`) already declares **one engine, one config
  table, one results table for ALL probe-style observations** and names exactly these kinds — DNS, TLS,
  PING, TCP, UDP, HTTP, HTTPS, RTSP, DICOM, **HL7, FHIR, LTI, LDAP, OPCUA, MODBUS**, NTP, SIP, 802.1X,
  cable, multi-step transactions — plus `internal/api/health_checks_*.go` as the **parallel stack to
  absorb** (SEED_ARCHITECTURE §3.1, Stage A1).
- **ADR-0025 §1** already drew the probe-vs-jobs boundary: a *recurring monitor is a probe*; a
  *one-shot diagnostic is a job*; and the one bridge is the engine's run-now primitive — "evaluate this
  *configured monitor* immediately," which shares the probe's threshold/breach/anomaly path. Both
  `Engine.RunDefinition` (ad-hoc) and `Engine.RunNow` (load a stored definition, dispatch, persist)
  exist today.

## Decision

Migrate the on-demand health-check stack onto the probe engine, delete the legacy parallel stack, and
rename the transport family **last**. Concretely:

### 1. The health-check verticals are probes — make the charter concrete

The eight verticals (HL7, FHIR, SQL, FileShare, LDAP, LTI, OPC-UA, Modbus) are **recurring
reachability/health observations of a target** — probes by ADR-0025 §1, and already claimed by the
probe charter. They become `probe.Checker` implementations under `internal/probe/checkers/`, registered
in the engine alongside the existing nine. The six already-duplicated protocols (ping/tcp/udp/http/
rtsp/dicom) keep the probe checker and **delete the `health_checks_*.go` copy** — one implementation per
protocol.

### 2. On-demand `/run` is a probe run-now, not a job

Per ADR-0025 §1, on-demand "run all my checks now" is **not** a `platform/jobs` job (jobs are one-shot
operations that produce a `Job{Result}` and *never* anomalies). It is the engine's on-demand evaluation
of the operator's **configured probe definitions** — `RunNow` per definition — sharing the same
checker → `Breach` → anomaly path an interval run uses. So an on-demand check **raises and clears the
same anomaly** a scheduled run would. `/run` is rewired to fan `RunNow` over the configured probes and
return their `Result`s; it dispatches through the engine, not the deleted `run*Tests()` methods.

### 3. `/settings` target config migrates from the config file to the `probes` table

The check targets (`config.Config.HealthChecks.*` lists) become **probe definitions** in the `probes`
table — the single source the engine schedules and `RunNow` loads. The migration maps each target list
to `Probe{Kind, Target, Params, IntervalSeconds, Enabled, Warning, Critical}`, and resolves the
`criticality` field to either a threshold input or a catalog/anomaly severity (decided per kind in the
implementing PR; cert-expiry in ADR-0025's as-built shows the kind-specific-threshold pattern). Once
migrated, these targets are **continuously monitored and feed the unified anomaly store** — the point of
the whole platform — instead of only being checked on demand.

### 4. Delete the ~4,067-LOC legacy stack — no compat

As each kind is covered by a probe checker, its `health_checks_*.go` / `handlers_*_checks.go` files,
the `run*Tests()`/`Run*Checks()` `Server` methods, and the config-file `HealthChecks` plumbing are
**deleted** (pre-alpha, no compat). The end state has one protocol implementation, one storage, one
transport.

### 5. Rename the transport family **last**, in one coherent change

Only after `/run` + `/settings` are probe-backed does `/telemetry/health-checks/*` →
`/telemetry/probes/*` become honest. Renaming earlier would leave a path that says "probes" while the
data still came from the legacy config-file stack — the half-rename ADR-0025 §5 and ADR-0026 explicitly
refused. The rename is mechanical: three `path:` literals in `server_routes.go`, regenerate the
`capabilities.txt` golden, update the auth/endpoints integration tests, and the four frontend fetch
sites (`HealthCheckCard`, `SlaDashboardCard`, the two settings-drawer hooks) plus the `HealthCheck*`
component/type/i18n identifiers. The `/anomalies` path rename rides along here.

## Phasing

Each phase is its own PR; the behavioral migration (1–4) **must precede** the rename (5).

| Phase | Scope | Notes |
|---|---|---|
| **P1** | Vertical checkers — HL7, FHIR, SQL, FileShare, LDAP, LTI, OPC-UA, Modbus as `probe.Checker`s, registered in the engine. | The bulk of the work; can be split per-kind or batched. Each needs its kind-specific threshold shape (à la cert-expiry). New files are single-word lowercase per the existing `internal/probe/checkers/` convention — `hl7.go`, `fhir.go`, `sql.go`, `fileshare.go`, `ldap.go`, `lti.go`, `opcua.go`, `modbus.go` — **no underscores** (the repo filename policy allows `_` only in `_test.go`). |
| **P2** ✅ | `/settings` storage migration: `config.Config.HealthChecks` target lists → `probes` table; a goose migration if the `probes` schema needs new columns. **As-built: `criticality` was removed outright** (unread since ADR-0026), not mapped; no migration was needed. | Settling the config→DB move; the scheduler now monitors these targets continuously. |
| **P3** ✅ | Rewire `/run` to fan `Engine.RunNow` over the configured probes; **delete** the legacy protocol files + `run*Tests()`/`Run*Checks()` methods. **As-built: ~3,600 LOC deleted** (combined with P4 — see the as-built note). Config-file-list plumbing left as a noted follow-up. | The deletion landed here once P3a covered the rendered metadata. |
| **P4** ✅ | Frontend rework. **As-built: nearly free** — the new `/run` Go DTO mirrors the card's existing `HealthCheckData` shape field-for-field (and fixes a pre-existing FE/backend drift), so the card was untouched and no shape reshuffle/broken-intermediate was needed. Component/token renames are cosmetic and not required for correctness; deferred. | Folded into the P3 PR to avoid a broken intermediate. |
| **P5** | Transport rename `/telemetry/health-checks/*` → `/telemetry/probes/*` (3 backend literals + `capabilities.txt` golden + integration tests + 4 FE fetch sites + i18n namespace). | Pure rename; rides last so it is never a half-rename. |

## Alternatives considered

- **On-demand `/run` becomes a `platform/jobs` job kind.** Rejected — ADR-0025 §1: jobs are one-shot
  operations producing a `Job{Result}` and never anomalies; `/run` is an on-demand evaluation of
  *configured monitors* that must raise/clear the same anomalies an interval run does. It is a probe
  run-now, not a job. (A genuinely long batch could later gain progress via SSE, but that does not make
  it a job — the engine already bounds concurrency.)
- **Rename the transport now, migrate behavior later.** Rejected — a `/telemetry/probes/*` path serving
  the legacy config-file stack is the misleading half-rename ADR-0025 §5 / ADR-0026 refused. The rename
  must follow the behavior.
- **Keep the legacy stack; only add new checkers to probe.** Rejected — two parallel protocol
  implementations is exactly the duplication the NMS re-architecture (probe charter, SEED_ARCHITECTURE
  §3.1) sets out to delete, and it leaves the verticals un-monitored (on-demand only, no anomalies).
- **Drop the eight verticals entirely.** Rejected — they are real diagnostic capabilities (medical
  HL7/FHIR/DICOM, enterprise SQL/LDAP/FileShare, industrial OPC-UA/Modbus, education LTI) that the
  probe charter already commits to owning.

## Consequences

- One protocol implementation, one config store (`probes` table), one transport family. ~4,067 LOC of
  duplicated legacy code deleted.
- The eight verticals gain **continuous monitoring and anomalies** — today they are on-demand only.
- Real cost: **eight new probe checkers** (P1) is the bulk of the effort, each needing a kind-specific
  threshold shape and tests; plus a storage migration and a non-trivial frontend rework.
- `criticality` finally gets a home (a threshold input or anomaly severity per kind) instead of being
  stored-but-unread.
- Until P1–P4 land, the transport keeps the `health-checks` name — accepted, because an honest name is
  worth more than an early one.

## Open questions (resolved in the implementing PRs, not here)

- **`/run` batch latency.** Fanning `RunNow` over many targets (DICOM/HTTP handshakes) may be slow.
  Lean: keep it synchronous and rely on the engine's bounded concurrency; revisit a progress-bearing
  variant only if real batch sizes demand it (still a probe run-now, not a job).
- **Per-kind threshold shapes.** Each vertical defines its own breach fields (à la
  `cert_days_remaining`); the exact fields and catalog `Def`s are authored with each checker in P1.
- **`criticality` mapping.** ~~Whether it becomes a threshold input or an anomaly severity is a per-kind
  call made in P2.~~ **Resolved in P2: removed.** Unread since ADR-0026 deleted health scoring; dropped
  everywhere (no-compat, pre-alpha) rather than invent a new behavior. A future per-kind severity input
  can be reintroduced deliberately if a real consumer needs it.
