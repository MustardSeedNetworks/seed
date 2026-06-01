# ADR-0004: In-process domain event bus

**Status:** Accepted — 2026-05-31

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
