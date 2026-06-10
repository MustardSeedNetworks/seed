# ADR-0021: Persist the anomaly engine in SQL and converge every source on it

**Status:** Proposed — 2026-06-10 · completes the deferred persistence clause of
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
5. **Lifecycle + retention** (defaults below, to confirm): persist on state change (new/escalated/
   resolved), debounced; keep active anomalies indefinitely; age resolved ones via TTL with
   optional daily rollups mirroring `health_check_rollups_*`.

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

## Decisions to confirm (owner)

1. **Write cadence** — write-through on state change, debounced (recommended) vs periodic flush
   vs write-through every Observe. Trade-off: durability vs write volume.
2. **Retention** — keep active forever + TTL-age resolved with daily rollups (recommended) vs
   raw-forever vs TTL-everything.
3. **Phase label** — standalone "Anomaly Platform" workstream (recommended) vs a phase appended
   to the strangle plan.
