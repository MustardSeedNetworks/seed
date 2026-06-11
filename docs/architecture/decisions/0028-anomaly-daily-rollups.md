# ADR-0028: Daily rollups for the anomaly store (census of mutable instances, not a RollupSource)

**Status:** Proposed — 2026-06-11 · design-only (no code in this ADR) · realizes the deferred
"daily rollups" clause of [ADR-0021](0021-persist-and-converge-anomaly-engine.md) §"Owner decisions"
and its "Phase 2 as-built" re-sequencing note. Builds on
[ADR-0006](0006-migrations-sql-goose-strict.md) (goose/STRICT migrations),
[ADR-0010](0010-identifier-casing-conventions.md) (casing). Owner sign-off needed on the two
decision points flagged **[OWNER]** below before implementation.

## Context

ADR-0021 converged every anomaly source onto one SQL `anomalies` table and locked two retention
mechanisms:

> **Retention: TTL + daily rollups**, mirroring `health_check_rollups_*` — resolved anomalies
> pruned after a window (default 90d), daily per-(def,subject) rollups kept long-term.

Phase 2 (#1630) shipped the **TTL purge** half (`DeleteResolvedOlderThan`, default 90d, wired into
the data-retention goroutine). It **re-sequenced the daily rollups into their own design pass** —
this ADR — for a concrete reason recorded in the Phase-2 as-built note:

> `timeseries/retention.RollupSource` fits time-bucketed raw rows, not mutable anomaly instances;
> health's `CreateDailyRollup` has no scheduled caller (latent gap — don't copy).

Two facts make the rollup non-trivial, and they are the whole reason this is an ADR and not a
one-line `Register(src)`:

1. **The `anomalies` table is coalesced, not an event log.** The schema (migration
   `00006_anomalies.sql`) keys on `id = defKey|subjectKind|subjectId` — exactly **one mutable row
   per (def, subject)** carrying a running `count`, `first_seen`, `last_seen`, and lifecycle
   (`is_resolved`, `resolved_at`). There is **no per-occurrence history** to `GROUP BY`. You cannot
   reconstruct "what happened on 2026-03-14" after the fact — the row only remembers its cumulative
   count and the most recent `last_seen`.

2. **`RollupSource` models the opposite shape.** The retention engine's `RollupSource`
   (`internal/timeseries/retention/types.go`) is `RollupHour → RollupDay → PurgeRaw/Hourly/Daily`
   over **immutable, timestamped raw rows** (`probe_results`, `metrics`): each raw row belongs to
   exactly one bucket forever, so `INSERT OR REPLACE ... GROUP BY` is faithful and idempotent. An
   anomaly instance belongs to *every* day between `first_seen` and `last_seen` and **keeps
   mutating** — it has no hourly tier and no raw stream the engine could purge (the Phase-2 TTL
   cleanup already owns deletion of resolved rows). Forcing anomalies through `RollupSource` would
   mean a no-op `RollupHour`, a `PurgeRaw` that double-owns the TTL purge, and a `RollupDay` that
   lies about its inputs.

## Decision

### 1. Anomaly daily rollups are a **daily census of the live table**, not a re-aggregation of an event stream

Because the table is coalesced, the only faithful daily artifact is a **census**: each UTC day, write
one row per (def, subject) whose lifecycle **intersects that day**, snapshotting the facts the live
row still holds. The purpose (per ADR-0021) is long-term trend that survives the 90-day TTL purge of
resolved rows — so the census must capture enough that, after the live row is gone, history still
answers *"on day D, def=X subject=Y was active at severity=S, cumulative count C, resolved=?"*.

A day "intersects" an anomaly when `day_bucket` falls in `[trunc(first_seen), trunc(last_seen)]`
(active anomaly) or equals `trunc(resolved_at)` (the day it cleared). In practice the scheduled pass
only needs to (re)write rows for **the current day and any day touched since the last successful
pass** — see §3.

### 2. Anomalies do **not** implement `RollupSource`; the census rides the existing maintenance tick

- **No `RollupSource` implementation, no hourly tier.** One daily table, one daily INSERT-OR-REPLACE.
- **The census step is folded into the same maintenance pass that already runs the Phase-2 TTL
  `RunCleanup`** (the data-retention goroutine in `internal/api/server_shutdown.go`), so there is
  **one owner** of anomaly-table maintenance and one ordering guarantee: **census first, purge
  second** — the rollup must read resolved rows *before* TTL deletes them, or a resolved anomaly
  purged at 90d would never be censused on its resolution day.
- **Purge of the census table itself** is governed by the `DailyDays` tier horizon
  (`TierHorizons`), matching how `probe_rollups_daily` is bounded. Free/Starter keep zero; Pro keeps
  long-term (2y). This keeps appliance growth bounded.

### 3. The census has a **live scheduled caller from day one** — do not repeat the `CreateDailyRollup` gap

The deleted health stack shipped a `CreateDailyRollup` with **zero scheduled callers** — a rollup
that never ran. The discipline this ADR mandates:

- The census is invoked from the live maintenance goroutine, guarded to run at most once per UTC day
  (idempotent `INSERT OR REPLACE` makes an accidental re-run harmless, so the guard is an
  optimization, not a correctness requirement).
- **Acceptance criterion (regression guard):** a test asserts the maintenance wiring actually calls
  the census — the same "0 callers" gap that bit `CreateDailyRollup` must be impossible to
  reintroduce silently. Mirrors the producer-callsite tests already used elsewhere in the anomaly
  work.

### 4. Schema — `anomaly_rollups_daily` (STRICT, goose migration)

One row per (day, def, subject). Proposed columns:

| Column | Type | Notes |
|---|---|---|
| `day_bucket` | TEXT | `YYYY-MM-DD` UTC, matches `retention.dayFormat` |
| `def_key` | TEXT | catalog key |
| `source` | TEXT | `wifi\|wired\|snmp\|…` — carried for source-scoped trend queries |
| `category` | TEXT | carried from the live row |
| `subject_kind` | TEXT | correlation key (ADR-0021 taxonomy) |
| `subject_id` | TEXT | correlation key |
| `max_severity` | TEXT | highest severity the (def,subject) held as of the census |
| `count_cumulative` | INTEGER | the live row's cumulative `count` at census time (see §5) |
| `first_seen` | TEXT | RFC3339, carried |
| `last_seen` | TEXT | RFC3339, carried |
| `is_resolved` | INTEGER | 0/1 as of the census |
| `resolved_at` | TEXT | nullable |

- **PK** `(day_bucket, def_key, subject_kind, subject_id)` → idempotent re-census.
- **Indexes:** `day_bucket` (purge + day-range queries); `(subject_kind, subject_id)` (cross-source
  correlation, the whole point of the subject taxonomy).
- STRICT + `CHECK` constraints consistent with `00006_anomalies.sql`; schema golden regenerated.

### 5. Per-day occurrences — store cumulative, difference at query time (V1.0)

The live row carries only a **cumulative** `count`, so a faithful "occurrences on day D" needs the
prior day's cumulative. V1.0 stores `count_cumulative` and derives per-day deltas by differencing
consecutive day rows for the same (def, subject) at query time. This is the same honesty trade-off
the probe daily rollup already documents (its `AVG(avg_latency_ms)` caveat) — exact intra-day
recurrence counting would require snapshotting at each day boundary and is deferred. **[OWNER]**
confirm census-with-cumulative is acceptable vs. carrying a true per-day delta column.

## [OWNER] decision points

1. **Census vs. true per-day delta** (§5): ship the simpler cumulative-snapshot census in V1.0, or
   pay for a per-day-delta column now?
2. **`DailyDays` horizon for anomaly rollups**: reuse the probe tier horizons verbatim (Pro 730d), or
   a distinct anomaly horizon? Default proposal: reuse, one policy to reason about.

## Consequences

- **Long-term anomaly trend survives the 90d TTL purge** — the locked ADR-0021 retention model is
  finally whole (TTL + rollups), with bounded appliance growth.
- **One owner of `anomalies`-table maintenance** (census + TTL in the same ordered pass) — no
  split-brain between the retention engine and the anomaly cleanup, and no double-owned purge.
- **`RollupSource` stays honest** — it keeps meaning "immutable raw → hourly → daily." We did not
  bend a clean abstraction to fit a different data shape; we named the difference and built the
  smaller right thing.
- **The `CreateDailyRollup` mistake is structurally prevented**, not just avoided — a callsite test
  gates it.
- **Cost:** one new table + migration + a census query + maintenance wiring + tests. No new
  goroutine, no hourly tier, no new package — the census rides the existing tick.

## Alternatives considered

- **Implement `RollupSource` for anomalies.** Rejected: no immutable raw stream, no hourly tier, and
  `PurgeRaw` would collide with the Phase-2 TTL ownership. Forces three of six interface methods to
  lie.
- **Append an immutable `anomaly_events` log and roll *that* up** (true event-sourced rollups).
  Faithful to per-day occurrence counts, and a natural `RollupSource`. Rejected for V1.0: it doubles
  the write path of the busiest producer (Wi-Fi scan bursts) precisely after ADR-0021 chose
  write-through-on-material-change specifically to avoid a per-observation write storm. Revisit if
  exact intra-day recurrence analytics become a requirement.
- **No rollups, rely on TTL only.** Rejected: violates the locked ADR-0021 decision and loses all
  history older than 90 days — the appliance could not answer "was this site flapping last quarter?"
