# ADR-0029: Converge the per-producer anomaly engines into one server-owned engine

**Status:** Accepted — 2026-06-11 · completes the end-state of
[ADR-0021](0021-persist-and-converge-anomaly-engine.md) ("ONE anomaly engine, ONE SQL database, ONE
structure, used everywhere"). Builds on [ADR-0025](0025-probe-is-the-active-monitoring-anomaly-source.md)
(probe is the active-monitoring producer) and the Phase 1–4 persistence/producer work. Sequenced
after ADR-0028 (daily-rollup census), which is independent of this convergence.

## Context

ADR-0021 locked a single source-neutral `anomaly.Engine` + SQL store, with source as a first-class
correlation dimension. The Phase 1–4 work shipped the persistence `Coordinator`, the SQL store, and
**two** long-lived producers — but each owns its *own* engine and Coordinator:

- `internal/wifi/visibility.Service` — engine over the Wi-Fi catalog, `Coordinator(source=wifi)`, a
  5 s eval loop that observes + flushes + prunes (idle retention 5 m), load-on-start. Owned by
  `internal/api.BackgroundComponents` (manual Start/Stop).
- `internal/probe/anomaly.Producer` — engine over the probe catalog, `Coordinator(source=probe)`,
  event-driven consume + a 30 s maintain loop that flushes + prunes (silence window 15 m),
  load-on-start. Owned by the engine lifecycle registry (`registerEngineIfLicensed`).

Both write to the **same** `anomalies` table, differentiated by the `source` column, and their
catalogs are disjoint (Wi-Fi def IDs vs probe def IDs). So the store half of ADR-0021 is converged;
the **engine half is still scattered into two instances**. This is the remaining de-scatter.

Three facts in the current code shape the design:

1. **Source is a `Coordinator` field, not per-detection.** `NewCoordinator(engine, store, source)`
   tags every write with one source. A single shared Coordinator therefore cannot serve two sources —
   source must move onto the `Detection`.
2. **Resolution windows are genuinely per-source.** Wi-Fi resolves on 5 m of airspace silence; probe
   on 15 m (must exceed the probe interval, ADR-0025). A single engine pruned with one cutoff would
   wrongly resolve the other source's instances. `Prune` must become source-scoped.
3. **Consumers are asymmetric.** `/wifi/anomalies` reads the in-memory `engine.Snapshot`;
   `/telemetry/probes/anomalies` reads the store via `ActiveBySource(probe)`. With one shared engine
   holding all sources, the Wi-Fi handler can no longer treat "the engine" as Wi-Fi-only.

## Decision

### 1. One server-owned `anomaly.Coordinator` (single engine, merged catalog, single store)

The server constructs exactly one `Engine` over a **merged catalog** (Wi-Fi defs ∪ probe defs ∪
future), wrapped in one `Coordinator` over `db.Anomalies()`. The producers no longer own engines or
Coordinators — they are injected with the shared one. `NewCatalog`'s duplicate-ID rejection becomes
the fail-fast guard that two domains never ship a colliding def ID.

### 2. `Source` moves onto `Detection`; the Coordinator drops its single-source field

`Detection` gains a `Source`. The **producer stamps it once** at the hand-off to the Coordinator (the
domain rule code — `wifianomaly.Detector.Detect`, `probeanomaly.Detections` — stays source-agnostic;
only the one observe site stamps). The engine carries source on the tracked instance and projects it
into the persisted `Record`; `RecordID` stays `defKey|subjectKind|subjectId` (source-free), which is
unique because catalog IDs are globally unique across the merged catalog. `instanceKey` is unchanged
— coalescing is still by (def, subject).

### 3. `Prune` is source-scoped; `ResolveSubject` stays subject-scoped

`Coordinator.Prune(ctx, source, cutoff)` resolves only that source's idle instances, so each producer
drives its own window (Wi-Fi 5 m, probe 15 m) against the shared engine. `ResolveSubject` (the probe
clean-result fast-path, ADR-0025 §3) is unchanged — it is keyed by subject, and probe subjects are a
distinct `SubjectKind`, so there is no cross-source clearing.

### 4. Consumers read the store by source; the in-memory snapshot is internal-only

`/wifi/anomalies` switches to `db.Anomalies().ActiveBySource(SourceWiFi)`, mirroring the probe
endpoint. The in-memory `Snapshot` remains for diagnostics/status counts but is no longer a consumer
read path — so a shared engine holding every source never leaks the wrong source to an endpoint.

### 5. Load-on-start happens once, server-owned

The server calls `Coordinator.Load()` once during init (before any producer starts observing), over
the merged engine. No per-source cross-contamination, because one catalog holds every def — the
Phase-1 `Restore` "skip orphan defs" guard no longer silently drops the *other* producer's rows. The
two duplicate load-on-start paths in the producers are removed. A final `Flush` on shutdown stays.

### 6. `troubleshooting.AnalyzeBSSes` stays a transient one-shot — out of scope

The survey analysis path builds a single-use engine, returns a snapshot, and discards it (no
lifecycle, no persistence). It is intentionally ephemeral and is **not** converged.

## Consequences

- **The ADR-0021 end-state is reached** — one engine, one catalog, one store, one load, one place for
  cross-source correlation and the ADR-0028 census to build on.
- **Source becomes a true per-detection fact**, not a coordinator-construction accident — the natural
  home for it and a prerequisite for any third producer (wired/SNMP/Bluetooth) to share the engine.
- **Per-source resolution stays correct** — convergence does not flatten the genuinely different
  silence windows; it scopes them.
- **The read path is uniform** — both endpoints read the store by source; no in-memory/SQL split.
- **Cost:** a core type change (`Detection.Source`, source-scoped prune) + producer surgery (strip
  owned engines) + one consumer repoint. Phased so each PR is independently green.

## Phasing (each its own golden-gated PR)

- **P1 — core plumbing (this PR):** add `Detection.Source`; engine projects source into records and
  gains source-scoped prune; `Coordinator` drops the source constructor arg, reads source from the
  detection, and takes `Prune(ctx, source, cutoff)`. The two existing producers stamp their source at
  the observe site and pass it to Prune. Behavior-preserving — each still owns its Coordinator; this
  is the type refactor that makes a single shared Coordinator possible.
- **P2 — single server-owned Coordinator:** merged catalog + one Coordinator built in the server;
  inject it into both producers (they stop constructing their own engine/Coordinator/load); one
  server-owned load-on-start + shutdown flush.
- **P3 — consumers read the store:** `/wifi/anomalies` → `ActiveBySource(wifi)`; remove the dead
  in-memory read path; goldens/tests updated.

## Alternatives considered

- **Keep two Coordinators sharing one store (status quo).** Rejected: leaves two engines, two catalogs,
  two loads (with the orphan-def cross-contamination footgun), and no single home for cross-source
  correlation — i.e. ADR-0021's engine half stays unconverged.
- **A multi-source "wrapper" Coordinator holding N sub-engines.** Rejected: re-introduces N engines
  behind one type; the whole point is one coalescing map so a subject seen by two sources can
  correlate. The merged single catalog is simpler and is what ADR-0021 specified.
- **Unify the resolution window (one prune cutoff).** Rejected: 5 m vs 15 m are correctness-bearing
  (probe's must exceed the probe interval); flattening them would mis-resolve one source.
