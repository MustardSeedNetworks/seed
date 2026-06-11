# ADR-0025: Probe is the recurring-observation engine and the anomaly producer for active monitoring (probe vs jobs)

**Status:** Accepted — 2026-06-11 · resolves the "probe-vs-jobs" loose end flagged with
[ADR-0021](0021-persist-and-converge-anomaly-engine.md) (anomaly convergence) and names the
active-monitoring anomaly source. Builds on [ADR-0005](0005-unified-jobs.md) (jobs runner),
[ADR-0011](0011-network-anomaly-engine.md) (anomaly engine), and
[ADR-0023](0023-snmp-polling-orchestrator.md) (the sibling active poller).

## Context

ADR-0021 converged anomaly *persistence* and deleted the never-fed bespoke
`internal/health.AnomalyDetector`, repointing the `/telemetry/health-checks/anomalies` read at the
unified SQL store. It deliberately stopped short of building a health **producer**, because doing so
surfaced an unresolved architecture question and a dormant subsystem:

1. **The health-check subsystem is dead.** `HealthCheckRepository.Record`/`RecordBatch` have **zero
   production callers** — scoring, SLA, alerting, and anomalies all read a `health_check_results` table
   that nothing writes. It is the "parallel stack" the V1.0 NMS re-architecture is strangling
   (`internal/probe/doc.go`; SEED_ARCHITECTURE.md §3.1, which lists `internal/api/health_checks_*.go`,
   `dnsmon`, and `sslmon` as the duplicated stacks `internal/probe` **replaces**). Reviving it would
   re-grow the exact duplication the refactor removes.

2. **The live active-monitoring engine is `internal/probe`.** It is fully wired (`initProbeEngine`:
   `WithStorage(db.Probes(), scheduler)`, nine registered checkers, registered as a licensed lifecycle
   engine), schedules probes, persists `probe_results`, evaluates `Warning`/`Critical` thresholds, and
   already emits a `ResultEvent{Result, []Breach}` to subscribers via `Engine.Subscribe()` — a fan-out
   channel with **no production consumer yet**. A `Breach{ProbeID, Severity, Field, Threshold, Actual}`
   is structurally an `anomaly.Detection` waiting to happen.

3. **Probe vs jobs was never written down.** Both are async, but they are different shapes of work, and
   the boundary needs to be explicit so neither absorbs the other by drift.

4. **The store's source name was a placeholder.** ADR-0021 added `anomaly.SourceHealth = "health"`,
   anticipating a health producer. With health dead and probe canonical, that name is a misnomer.

## Decision

**`internal/probe` is the single recurring-observation engine, distinct from the `internal/platform/jobs`
runner, and it is the producer of active-monitoring anomalies. Its threshold breaches feed the unified
anomaly store under `source=probe`. The dormant health-check stack is legacy to be deleted, not revived.**

### 1. Probe vs jobs — the boundary

| | **Probe** (`internal/probe`, ADR-0011/0021/this) | **Jobs** (`internal/platform/jobs`, ADR-0005) |
|---|---|---|
| Trigger | Declarative + **recurring** on a fixed interval (the scheduler) | Imperative + **one-shot**, user/system requested |
| Examples | DNS/TLS/ping/HTTP/RTSP/DICOM monitors of a target | speedtest, iperf, a discovery scan, a vuln scan, a survey |
| Output | `probe_results` time series + threshold `Breach`es → **anomalies** | a `Job{State, Progress, Result}` record + state events |
| Lifecycle | runs forever at interval; result is "the latest observation" | runs to terminal state (done/failed/cancelled) then stops |

A **recurring monitor is a probe**; a **one-shot diagnostic is a job**. They are not merged: a probe is
not "a job kind," and a job is not "a probe run." The one bridge is `Engine.RunNow(probeID)` — the probe
engine's primitive for "evaluate this *configured monitor* immediately" (used by AutoTest sequences and
manual UI refresh). That is an on-demand evaluation of an existing probe definition, not a general job;
it shares the probe's threshold/breach/anomaly path, so an on-demand check raises and clears the same
anomaly an interval run would. Jobs never produce anomalies; they produce job results.

### 2. Probe breaches feed the unified anomaly store (`source=probe`)

A long-lived, server-owned anomaly **producer** subscribes to `Engine.Subscribe()` and maps each
`ResultEvent`'s breaches onto `anomaly.Detection`s, persisting through an `anomaly.Coordinator`
(write-through on material change, batched `Flush`, resolve-on-`Prune`; ADR-0021). This is the same
pattern the Wi-Fi visibility producer uses, applied to a push channel rather than a pull snapshot.

- **Source name.** Rename `anomaly.SourceHealth` → **`anomaly.SourceProbe = "probe"`** (pre-alpha, no
  compat). It is accurate: probe spans DNS/TLS/ping/… not just "health". The ADR-0021 phase-4 reader
  (`AnomalyRepository.ActiveBySource`, the `/telemetry/health-checks/anomalies` handler) reads
  `source=probe`.
- **Subject taxonomy.** Add **`anomaly.SubjectProbe = "probe"`**, keyed by `ProbeID`, as the correlation
  subject for active-monitoring anomalies (per ADR-0011's open `SubjectKind` set). One anomaly instance
  coalesces per `(defKey, probeID)`.
- **Catalog.** Probe failure modes get original, data-driven `Def`s in the unified catalog
  (`probe.unreachable`, `probe.high_latency`, …; cert-expiry, dns-failure, etc. as checkers grow their
  threshold shapes). `Breach.Field` selects the `defKey`; `Breach.Severity` overrides the catalog
  default; `Field`/`Threshold`/`Actual`/`Kind`/`Target` become the detection evidence.

### 3. Resolution is push-model TTL-on-silence

Unlike the Wi-Fi producer, which re-snapshots the full live set each evaluation and prunes anything
**absent** from the snapshot, probe is **event-driven**: a recovered probe simply stops emitting
breaches. The producer therefore resolves an instance by **`Prune(cutoff)` on a periodic maintenance
tick**, where `cutoff = now − resolveWindow`. `resolveWindow` must exceed a probe's interval so a
still-failing probe (which re-breaches every interval, refreshing `lastSeen`) stays active; a recovered
probe's anomaly resolves once it has been silent for the window. Interval-aware (per-probe) resolution
and an explicit "clean result clears this probe's anomalies now" fast-path are noted refinements, not
required for the first producer.

### 4. The health-check stack is legacy to delete, not revive

`health_check_results`, the `HealthCheckRepository` write path, and the `internal/health`
scoring/SLA/dependency code that reads that empty table are the strangled parallel stack. They are
**deleted or rebuilt on probe** in a separate cleanup track (their own slice), per `internal/probe`'s
charter — not resurrected to satisfy the anomaly producer.

### 5. Endpoint path: deferred rename

The read endpoint stays **`/telemetry/health-checks/anomalies`** for now even though it reads
`source=probe`, because the whole `/telemetry/health-checks/{results,history,scores,sla,alerts,anomalies}`
transport family is strangled together (those siblings still front the legacy tables). Renaming only the
anomalies path to `/telemetry/probes/anomalies` would leave a half-renamed, inconsistent surface. The
path rename rides the health-checks→probes transport migration as one coherent change (pre-alpha, no
compat, so the rename is free when it happens).

## Alternatives considered

- **Revive the dormant health-check executor as the producer.** Rejected — it re-grows the parallel
  stack the NMS re-architecture is explicitly deleting (`internal/probe/doc.go`, SEED_ARCHITECTURE §3.1).
  Two engines doing scheduled-probe→threshold→record is the duplication ADR-0011/0021 set out to remove.
- **Make probe "a job kind."** Rejected — jobs (ADR-0005) are one-shot, terminal, progress-bearing
  operations; probes are infinite recurring monitors with a results time series and a breach→anomaly
  pipeline. Collapsing them would saddle the jobs runner with scheduling/interval/threshold concerns it
  was never meant to own, and saddle every probe with a terminal-state lifecycle it does not have.
- **Per-source anomaly tables / a second engine for probe.** Rejected — ADR-0021's single store and
  source-neutral engine already hold; probe is just another `Source` writing the one schema.
- **Resolve probe anomalies only by exact recovery signal (clean result).** Deferred, not rejected — a
  precise fast-path, but TTL-on-silence is simpler, matches the engine's existing `Prune(cutoff)` API,
  and is correct; the fast-path is an additive refinement.

## Consequences

- The idle `Engine.Subscribe()` channel gains its first consumer; active-monitoring problems become
  durable, queryable, lifecycle-tracked anomalies instead of evaporating. (The alerts pipeline the
  channel was originally built for can attach as a second subscriber later — the fan-out supports it.)
- `anomaly.SourceProbe` + `anomaly.SubjectProbe` enter the taxonomy; the phase-4 reader points at
  `source=probe`.
- A new long-lived producer component registers in the lifecycle registry (Start/Stop), owning the
  subscriber loop and the Flush/Prune maintenance tick — a concurrency surface that needs its own tests.
- A new authoring workstream: the probe anomaly catalog (`Def`s per breach/failure mode), grown as
  checkers add kind-specific thresholds.
- A debt is made explicit: the health-check transport family + the legacy tables are slated for deletion;
  the endpoint path rename rides that migration.

## Implementation phasing

1. **This ADR + taxonomy/naming (slice A).** Add the ADR; rename `SourceHealth`→`SourceProbe`; add
   `SubjectProbe`; repoint the ADR-0021 phase-4 reader at `source=probe`. No producer yet → the endpoint
   reads empty (as it does today), now under the correct source.
2. **The producer (slice B).** Probe anomaly catalog + `Breach`→`Detection` mapping + the server-owned
   subscriber/Coordinator component (load-on-start, Flush + Prune tick), wired in the composition root
   after the probe engine. Tests cover the mapping and the concurrency.
3. **Catalog growth + the health-check deletion track** follow as checkers and the transport strangle land.
