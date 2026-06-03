# ADR-0004: In-process domain event bus

**Status:** Accepted — 2026-05-31 · Core implemented (Phase 4) — 2026-06-02 ·
Outbox deferred (Phase 5c) — 2026-06-03

## Context

ADR-0001 forbids modules importing each other (the distributed-monolith guard,
`depguard`-enforced). But modules genuinely need to react to one another: Shell
discovers a device → Sap/health should monitor it, Harvest should record it,
Sap/alerts should evaluate thresholds. A seam is required. Options: an event bus,
or explicit app-layer orchestration via interfaces.

## Decision

A **`platform/events` in-process domain event bus.** Modules publish/subscribe typed
events; they never call each other directly.

Semantics:
- In-process, single binary — no broker.
- **Events are facts, past-tense** (`DeviceDiscovered`) — never commands. They
  notify/react; they do not request/respond.
- Async, at-least-once, ordered per-topic. A panicking subscriber must not fail the publisher.
- Ephemeral by default; **audit-class events are durable** (→ `platform/audit`).
- Subscribers registered before publishers start (supervisor ordering) so startup events aren't lost.
- **Transactional outbox:** events emit only after the DB commit (ADR via §7) — a
  rolled-back operation never fires an "it happened" event.

## Consequences

- Modules stay islands with zero import edges; fan-out is clean.
- Requires discipline: events must remain facts, or the bus becomes hidden control-flow.
- Needs the outbox to stay consistent with persistence.
- Rejected pure app-layer orchestration: verbose for publish-many, grows the app layer.
  (Hybrid still allowed for trivial 1:1 calls, but the bus is the default.)

## Implementation status (Phase 4 — 2026-06-02)

- **Bus core** `internal/platform/events` (#1466): typed `Event` facts, `Subscribe`/
  `Publish`/`Close`; async, ordered per-subscriber, at-least-once on `Close` drain,
  panic-isolated. The unified jobs runner (ADR-0005) publishes `job.*` state-change
  facts onto it; the `GET /api/v1/jobs/events` SSE stream is the first subscriber.

## Amendment (2026-06-03): transactional outbox **deferred**, with a trigger

The Decision above mandates a **transactional outbox** (events emit only after the
DB commit). After Phase 5c made the jobs store durable (#1481–#1485 — write-through
persistence, boot recovery, durable idempotency, retention), we deliberately
**defer** the outbox rather than build it now. This is a recorded decision, not an
omission.

**Why defer.** The transactional outbox exists to prevent a *dual-write
inconsistency* between the database and an **external or durable** consumer (a
message broker, another process, a persistent worker): "state committed but the
event to an outside party was lost." seed has no such consumer today:

- the bus is **in-process, single binary** (this ADR) — no broker;
- its only subscriber is the **ephemeral SSE bridge** (and a future in-process
  lifecycle supervisor), which is gone on restart anyway;
- on reconnect, clients **re-fetch state** from the now-durable `GET /jobs/{id}`
  (Phase 5c), so a state change that committed without its in-process event being
  observed is **self-healing** — there is no observable victim of the gap.

Building the outbox now would rewrite the live publish path (the runner would stop
publishing directly; `Save` would write job + outbox rows in one transaction; a
relay would poll/signal, publish post-commit, and mark rows published) — adding a
relay goroutine, poll cadence, and ordering concerns **for no present benefit**.
That is premature infrastructure; fitting the pattern to the actual problem is the
best-practice call.

**Trigger to build it.** Implement the transactional outbox when **any** of these
becomes true — at which point the dual-write gap gains an observable victim:

1. a **cross-process** or out-of-band event consumer is introduced (a broker, a
   sidecar, a separate worker, webhooks/push to an external system);
2. a **durable subscriber** that must not miss an event across a restart exists
   (e.g. a persisted audit/notification pipeline that does not re-derive from
   queryable state);
3. an event carries information **not reconstructable** from durable state via a
   re-fetch (so "re-sync on reconnect" no longer heals a lost event).

Until then the in-process bus + durable, queryable job state is the correct design.
ADR-0005's "events emit post-commit via outbox" line is governed by this amendment.
