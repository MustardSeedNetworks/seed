# ADR-0005: Unified async job runner

**Status:** Accepted — 2026-05-31 · Core implemented (Phase 4) — 2026-06-02

## Context

Every long-running operation reinvents run/status/cancel/progress: `speedtest` +
`speedtest/status`, `iperf/client` + `client/status`, `discovery/engine/scan` +
`stats` + `events`, `vulnerabilities/scan` + `status`, `pipeline/start` + `status` +
`cancel`, survey `start`/`pause`/`complete`. ~15 endpoints with duplicated,
inconsistent cancellation, progress, and timeout logic.

## Decision

One **`platform/jobs` task runner.** A `Job{ID, Kind, State, Progress, Result}` with a
uniform surface: `POST /jobs` (kind + params), `GET /jobs/{id}`, `DELETE /jobs/{id}`
(cancel), and a single SSE stream. Speedtest, iperf, discovery, vuln, survey,
traceroute, pipeline become **job kinds**.

Semantics:
- Persisted (survive restart or fail cleanly); retention + cleanup.
- Bounded concurrency; reject with `503` under overload (appliance backpressure).
- Cancellation via ctx; progress reporting; result stored in `store`.
- Job authz is role-level, consistent with the capability registry.
- Jobs emit events on state change → the frontend subscribes to one job/event stream.

## Consequences

- ~15 bespoke endpoints collapse into one model; consistent cancel/progress/timeout.
- The frontend's "one SSE manager → React Query cache" becomes concrete (one stream).
- Requires upfront design of the job abstraction and persistence.
- Each long-op must be refactored into a job kind (Phase 4).

## Implementation status (Phase 4 — 2026-06-02)

Shipped:

- **Runner core** `internal/platform/jobs` (#1467): `Job{ID,Kind,State,Progress,Result,Err}`,
  `Runner.Register/Submit/Get/Cancel/Cleanup/Close`. Bounded concurrency with a hard
  cap (`ErrAtCapacity` → 503); cancel via context; clamped progress; panic-isolated
  handlers; state-change facts published on the `platform/events` bus (`job.*` topics);
  in-memory store with retention.
- **HTTP surface** (#1468): `POST /api/v1/jobs`, `GET /api/v1/jobs/{id}`,
  `DELETE /api/v1/jobs/{id}`, and one SSE stream `GET /api/v1/jobs/events`. Wired through
  the capability registry (POST/DELETE operator-gated); Idempotency-Key on create.
- **7 long-op kinds**: speedtest (#1470), iperf (#1471), vuln-scan (#1472),
  engine-scan (#1473), bluetooth-scan / wifi-discovery-scan / device-scan (#1474). Each is
  an additive thin wrapper over the existing ctx-aware service behind an interface seam.

Deviations / deferred from the original decision:

- **Persistence is in-memory v1** (fail-cleanly-on-restart), not durable. The transactional
  outbox + durable job store land in **Phase 5**; the runner already exposes the seam.
- **Legacy endpoints retained.** Each migrated op keeps its old `/telemetry/*` or
  `/security/*` run/status endpoint (additive strangler); the "~15 endpoints collapse"
  completes when the frontend consumes `/jobs` in **Phase 7**, at which point the legacy
  pairs retire.
- **survey is NOT a job kind.** `create → add-floors → add-samples → generate-report` is a
  stateful interactive *session*, not a one-shot long-op; it stays a session. (Its
  `generate-report` step may later become a job.)
- **pipeline deferred to Phase 6.** The discovery `pipeline` duplicates the discovery
  `engine` (both orchestrate discover→enrich→assess). Rather than enshrine the redundancy
  as a job kind, engine↔pipeline consolidation is folded into the **Phase-6** discovery
  split; the engine (DeviceRegistry-as-SSoT + event distribution) is the canonical
  orchestrator and is already exposed as the `engine-scan` kind.
