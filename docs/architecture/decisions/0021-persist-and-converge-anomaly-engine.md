# ADR-0021: Persist the anomaly engine in SQL and converge every source on it

**Status:** Accepted — 2026-06-11 (owner-confirmed the cadence/retention/phase
decisions below on 2026-06-10) · completes the deferred persistence clause of
[ADR-0011](0011-network-anomaly-engine.md); builds on [ADR-0006](0006-migrations-sql-goose-strict.md)
(goose/STRICT migrations), [ADR-0004](0004-event-bus.md)/[ADR-0017](0017-transactional-outbox-relay.md)
(events), [ADR-0002](0002-capability-registry.md) (Pro gating), [ADR-0010](0010-identifier-casing-conventions.md) (casing).

## Context

ADR-0011 decided the right thing — **one network-wide `internal/anomaly` engine**, a
data-driven catalog, pluggable sources, cross-source correlation — and explicitly said the
engine "dedups, correlates … applies TTL/clear, **and persists**." In practice only part of
that shipped:

- The engine exists and is **in-memory only** (`Observe`/`Snapshot`/`Prune` over maps). The
  persistence clause was deferred with the Wi-Fi feature work and never built. Detected
  anomalies **evaporate on restart**; there is no history, no acknowledgement, no query surface.
- Only **one source** feeds it (the Wi-Fi stack, `internal/wifi/anomaly`).
- Worse, the health subsystem grew a **separate bespoke `internal/health.AnomalyDetector`**
  (in-memory, and currently unfed — `RecordLatency` has no production callers). That is exactly
  the "per-subsystem hard-coding / sibling buckets" failure ADR-0011 set out to avoid, re-emerging.

The owner's requirement (2026-06-10): a **complete, comprehensive anomaly database** — one
SQL-backed engine, one structure, fed by **every** detectable source (security, wired/link,
Wi-Fi, Bluetooth, SNMP, IEEE/standards, health/latency, AutoTest) and read **everywhere**
(monitoring, Survey, discovery, dashboards). Forces: anomalies must survive restart, be
queryable and historical, support acknowledge/resolve lifecycle, and be retained/aged — all in
one schema so cross-source correlation (ADR-0011's rogue-AP-on-LAN case) actually persists.

## Decision

Make `internal/anomaly` **SQL-backed** and **converge all sources on it**. This does not change
ADR-0011's architecture; it implements its persistence clause and enforces its single-engine rule.

1. **One SQL store behind a port.** The engine gains an optional `anomaly.Store` (consumer-defined
   port) it writes through on state change and loads active instances from on start. SQL only;
   migrations are embedded `.sql` via goose with STRICT tables (ADR-0006). No second store, no
   per-source tables.
2. **One `anomalies` table = the `Anomaly` projection + persistence lifecycle.** Columns the
   in-memory model lacks are added at the persistence boundary: a stable instance `id`
   (`defKey|subjectKind|subjectId`), `source` (producer), and lifecycle
   (`resolved_at`, `is_resolved`, `acknowledged_by`, `acknowledged_at`). Rich fields
   (`evidence`, `standards`, `follow_ups`) persist as JSON; scalars as columns. Indexed for the
   real query patterns: `(last_seen)`, `(source)`, `(subject_kind, subject_id)`, `(severity)`,
   `(is_resolved)`. Wire DTOs stay camelCase (ADR-0010).

   ```sql
   CREATE TABLE anomalies (
     id              TEXT    NOT NULL PRIMARY KEY,   -- defKey|subjectKind|subjectId
     def_key         TEXT    NOT NULL,
     source          TEXT    NOT NULL,               -- wifi|wired|snmp|bluetooth|health|security|autotest
     category        TEXT    NOT NULL,
     severity        TEXT    NOT NULL,
     subject_kind    TEXT    NOT NULL,
     subject_id      TEXT    NOT NULL,
     title           TEXT    NOT NULL,
     description     TEXT    NOT NULL,
     recommendation  TEXT    NOT NULL,
     evidence        TEXT,                            -- JSON
     standards       TEXT,                            -- JSON
     count           INTEGER NOT NULL,
     first_seen      TEXT    NOT NULL,                -- RFC3339
     last_seen       TEXT    NOT NULL,
     resolved_at     TEXT,
     is_resolved     INTEGER NOT NULL DEFAULT 0,
     acknowledged_by TEXT,
     acknowledged_at TEXT
   ) STRICT;
   ```
3. **Converge sources.** Delete the bespoke `internal/health.AnomalyDetector` (pre-alpha, no
   compat) and route health/latency detections through the general engine as `anomaly.Detection`s,
   exactly as Wi-Fi does. Wired/link, SNMP, Bluetooth, security, and AutoTest sources register the
   same way as they come online (ADR-0011's pluggable-source model). The engine stays
   source-neutral and unit-tested against synthetic detections.
4. **Reads go through the store.** The C2 `internal/health/monitoring` anomaly seam, the
   `/wifi/anomalies` endpoint, and Survey analysis all read persisted anomalies (with history +
   lifecycle), not per-detector in-memory snapshots. Pro sources stay gated via `requireFeature`
   (ADR-0002); the engine and store are tier-neutral.
5. **Lifecycle + retention (owner-confirmed 2026-06-10).** **Write cadence: write-through on
   material state change** (a new anomaly or a severity escalation) for durability, with the
   high-frequency recurrence updates (`last_seen`/`count`) **batched via a periodic flush** — so a
   scan burst is one write, not one per observation. Resolution (Prune) is written through. **Retention:
   keep active anomalies indefinitely; TTL-age resolved ones (default 90d) with daily per-(def,subject)
   rollups** mirroring `health_check_rollups_*`, for bounded growth on appliances.

6. **Catalog model & severity (per ADR-0011's `AnomalyDef`).** Each anomaly *type* is a catalog
   entry that gives the operator a guided answer, not just a flag: a one-line **title** (the
   tooltip), an optional longer **description**, **impact** (what it affects and how), and a
   **recommendation** (how to resolve), plus **standards** (IEEE/RFC citations). Citations are a
   **structured field** *and* are surfaced in the displayed detail (not metadata-only), so the
   cite shows in the copy while anomalies stay filterable/linkable by standard. **Follow-ups**
   are capability-gated (ADR-0002): where seed has a narrowing tool/test for the diagnosis it
   runs it **automatically**; where it does not, it **prompts** the user. Copy is authored
   originally — accurate, never pasted from copyrighted competitor analysis (ADR-0011). The
   catalog must be **extensive** across wired/Network, Security, Wi-Fi, SNMP, Bluetooth, and
   health — far beyond today's 19 Wi-Fi rules — grown per source as each lands; this is a
   first-class authoring workstream, not an afterthought.

   **Catalog storage: embedded YAML, loaded once into the in-memory `Catalog` — not a DB.** The
   catalog is never a runtime query path. It ships as embedded YAML (`//go:embed`), parsed once
   at startup into the validated, id-keyed `Catalog` (ADR-0011, fails-fast); every lookup is then
   an in-memory `map[defKey]Def` (O(1), no SQL/disk) — faster than any database, which buys
   nothing for a small, static, read-only-after-load dataset. This is the inverse of detected
   *instances* (many/mutable/filtered/persistent → SQL): **access pattern decides storage.** YAML
   over Go literals so copy + citations are editable and diff-reviewed without touching detection
   logic; YAML over a DB because no operator/runtime catalog editing is required, so no
   catalog-management UI or seed/override layer is built. CI validates that every detector
   `defKey` resolves to a catalog entry (no orphans, no dangling rules).

   **Severity is an ordered ladder: `Info → Warning → Error → Critical`** (refines ADR-0011's
   `info/warning/critical`). Definitions, so the wording is consistent and honest:
   - **Info** — advisory; a noteworthy condition that is not itself a problem (no action needed).
   - **Warning** — degraded / sub-optimal / best-practice deviation; works now but should be
     addressed before it bites.
   - **Error** — an active fault or confirmed security exposure impairing function; needs
     remediation.
   - **Critical** — outage-level impact or severe security compromise; urgent.

   "Failure" is **phrasing in a test-sourced `title`** ("X test failed"), not a separate severity
   level — the severity of such an anomaly is still Error or Critical. The `severity` column
   stores the ladder value.

## Alternatives considered

- **Per-subsystem anomaly tables** (health table, wifi table, …). Rejected — it is ADR-0011's
  "sibling buckets" failure with a schema attached: no cross-source correlation, duplicated
  lifecycle logic, UI sprawl.
- **Keep the engine in-memory, add only a health table.** Rejected — leaves two engines and two
  scatter points; anomalies still don't survive restart for the other sources.
- **Event-log / outbox only, no queryable table.** Rejected — operators need ad-hoc queries
  (by endpoint, severity, time, unresolved); the outbox (ADR-0017) remains the delivery path for
  live events, not the system of record.

## Consequences

- Anomalies survive restart and gain history, acknowledgement, and query surfaces; cross-source
  correlation becomes durable, not just live.
- One schema and one engine to evolve; the bespoke health detector is deleted (the de-scatter).
- New responsibilities: the migration, the write cadence/debounce, retention/rollup, and a
  start-up load of active instances. The instance `id` derivation must match the engine's
  in-memory coalescing key so restart is idempotent.
- Ties anomaly persistence to ADR-0006 (goose/STRICT) and reuses Alerts/severity/events rather
  than a parallel path.
- Phases: (1) store port + migration + engine write-through; (2) load-on-start + retention;
  (3) health source convergence + delete bespoke detector; (4) repoint readers (monitoring/wifi/
  Survey); (5) catalog growth (wired/SNMP/etc.) as those sources land.

## Confirmed decisions (owner, 2026-06-10)

1. **Write cadence** — write-through on material state change, recurrence batched via periodic
   flush. (Resolved over "write every Observe" — durability for events that matter without a
   per-scan write storm.)
2. **Retention** — keep active forever + TTL-age resolved (90d default) with daily rollups.
3. **Phase label** — tracked as the "Anomaly Platform" workstream in the master roadmap; the
   bespoke `internal/health.AnomalyDetector` is deleted outright in the producer slice (pre-alpha,
   no compat).

## Implementation phasing (as-built status)

Phase 1 (this slice) lands the **persistence foundation** the rest builds on: the `anomaly.Store`
port + `Record`/`Source` model, the `anomalies` migration, the `database.AnomalyRepository`
implementation, and the persistence-aware `anomaly.Coordinator` (write-through on material change,
batched `Flush`, resolve-on-`Prune`). The four-level severity ladder (`Info → Warning → Error →
Critical`) refines ADR-0011's three levels and is a **separate, additive change** (engine
`rank`/escalation + catalog defaults); the persistence layer stores the severity string verbatim, so
adding `Error` later needs no schema change. Subsequent phases (per the Consequences list): load-on-
start + TTL/rollup retention; health-source convergence + delete the bespoke detector; repoint the
monitoring/Wi-Fi/Survey readers; catalog growth as each source lands.

**Phase 2 (as-built) — resolved-anomaly TTL purge, live.** `AnomalyRepository.DeleteResolvedOlderThan`
deletes `is_resolved = 1 AND resolved_at < cutoff` and is wired as a `RetentionPolicy` task
(`AnomalyResolvedDays`, default 90) into the existing periodic `DB.RunCleanup` maintenance loop —
so the "keep active forever, age out resolved at 90d" decision is enforced in production now, bounding
table growth on appliances. Active rows are structurally safe (their `resolved_at` is NULL, so the
predicate never matches them). Two items the original phase-2 sketch bundled were **re-sequenced**,
deliberately:
- **Daily rollups → deferred (own design pass).** The `timeseries/retention.RollupSource` framework
  is a poor fit for mutable anomaly *instances* (its `PurgeRaw(cutoff)` would age out active rows, and
  its 7-day raw floor / hourly tier don't apply). The legacy health-style rollup is the right model,
  but health's own `CreateDailyRollup` has no scheduled caller today — rollups need a dedicated
  design that doesn't copy that latent gap. The TTL purge already delivers the growth-bounding win;
  rollups are long-term history, separable.
- **Load-on-start → folded into the producer phase.** Repopulating the in-memory engine from
  `LoadActive` is only meaningful once a long-lived, server-owned `Coordinator` exists (the per-request
  Wi-Fi engines are transient). Persistence already satisfies the "anomalies survive restart" *query*
  promise; engine count/escalation *continuity* is secondary, so load-on-start lands with the server
  Coordinator + producers rather than in isolation. **Note for that phase:** persisted `severity` is
  the effective (post-escalation) value; on reload the engine seeds `baseSeverity` from it, which can
  let an already-escalated instance bump one further level on continued recurrence after a restart — an
  accepted, bounded artifact (a persistent cross-restart problem reading as more urgent), not a second
  stored column.

**Phase 3 (as-built) — Wi-Fi producer persists + load-on-start.** The long-lived Wi-Fi
`visibility.Service` became the first real producer: `anomaly.Engine.Restore` + `Coordinator.Load`
repopulate the live set from `LoadActive` on boot, and the service write-throughs material changes,
batches recurrence into one `Flush` per evaluation tick, and resolves on `Prune`. A nil store keeps the
pure in-memory engine. This delivered the deferred load-on-start (folded here, as the phase-2 note
predicted) for the Wi-Fi source.

**Phase 4 (as-built) — delete the bespoke health detector + repoint the reader to the store.** The
never-fed `internal/health.AnomalyDetector` (and its `Anomaly`/`EndpointStats` types) is **deleted
outright** (pre-alpha, no compat) — the de-scatter the Decision mandates. The C2
`internal/health/monitoring` `AnomalyReader` seam and the `/telemetry/health-checks/anomalies` endpoint
now read the unified store's source=health slice through a new
`AnomalyRepository.ActiveBySource(ctx, source)` read (canonically ordered via the shared
`anomaly.SortAnomalies`). The endpoint DTO changes from the bespoke `health.Anomaly` shape to
`anomaly.Anomaly`; the per-endpoint rolling-stddev `AllStats`/`EndpointStats` surface is dropped (the
unified model carries evidence + count + lifecycle instead). The frontend consumes only `activeCount`,
so no frontend type migration was required. No migration → no schema-golden change.

- **Health *producer* deliberately NOT built in this slice — blocked on architecture.** Phase 4 wires
  the *read* path; it does not invent a health *write* path, because there is no live health data source
  to feed one. The health-check subsystem is **dormant**: `HealthCheckRepository.Record`/`RecordBatch`
  have zero production callers, so scoring/SLA/alerts/anomalies all read a table nothing writes. The only
  live latency-evaluating subsystem is `internal/probe` (its engine evaluates `latency_ms` thresholds),
  which is exactly the scope of the **deferred probe-vs-jobs ADR**. Authoring a synthetic health producer
  now would recreate the unfed-stub anti-pattern in a new place. The health-source slice of the store is
  therefore empty until either the dormant health-check executor is stood up or `probe` is bridged into
  the engine — a decision the probe-vs-jobs ADR must make first. The endpoint is correctly plumbed and
  lights up the moment a real source=health producer persists. The **health catalog `Def`s**
  (latency_spike/availability_dip/error_spike/…) land with that producer, since they are only exercised
  once detections flow.
