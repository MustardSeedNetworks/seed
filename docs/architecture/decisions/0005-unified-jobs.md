# ADR-0005: Unified async job runner

**Status:** Accepted — 2026-05-31

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
