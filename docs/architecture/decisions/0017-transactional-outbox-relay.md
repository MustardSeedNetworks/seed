# ADR-0017: Transactional outbox relay

**Status:** Accepted — 2026-06-07 · Closes the Phase-5 persistence track

## Context

ADR-0004 (in-process domain event bus) mandated a **transactional outbox** so an
event is only ever delivered after the database write that produced it commits — a
rolled-back operation must never fire an "it happened" fact. ADR-0004's
2026-06-03 amendment then **deferred** the outbox on YAGNI grounds: the only
producer was the jobs runner, the only subscriber was the ephemeral SSE bridge,
and clients re-fetch durable job state on reconnect, so the dual-write gap had no
observable victim. The amendment recorded three explicit triggers for building it
(a cross-process consumer, a durable subscriber that must not miss events across a
restart, or an event carrying information not reconstructable from durable state).

The decision now is to **build the outbox** to close out the Phase-5 persistence
track and complete the re-architecture — providing the durable-delivery seam
*ahead of* the first triggering consumer rather than scrambling to retrofit it
when one lands. Two constraints shape the design:

1. **No regression to the live path.** The jobs runner already survives restart
   (durable store + `Recover`, ADR-0005) and re-derives state on reconnect. The
   amendment correctly warned that rewriting the runner to publish *through* the
   outbox would add a relay, poll cadence, and ordering concerns "for no present
   benefit." So the outbox must be **additive**, not a rewrite of the direct
   `bus.Publish` path the runner uses today.

2. **The bus stays the single delivery seam.** Subscribers already register by
   topic against `platform/events`. A relayed event must arrive on the same bus,
   on its original topic, so a future durable subscriber is an ordinary
   `bus.Subscribe` — it does not learn about the outbox.

## Decision

A **transactional outbox with a post-commit relay**, layered under
`internal/platform/outbox`, additive to the existing bus.

**Write side (atomic with the domain change).** A producer that needs durable
delivery enqueues the event *in the same transaction* as its domain write:

```go
err := db.WithTx(ctx, func(tx *sql.Tx) error {
    if err := domainWrite(tx, …); err != nil {
        return err            // rolls back; no outbox row, no event — ever
    }
    return db.Outbox().Enqueue(ctx, tx, topic, payload)
})
```

The `outbox` table row and the domain row commit or roll back **together** — that
is the entire correctness guarantee. `Enqueue` takes the caller's `*sql.Tx`; it
never opens its own transaction.

**Read side (the relay).** `outbox.Relay` polls the store for unpublished rows,
republishes each onto the bus as an `outbox.Message` (which implements
`events.Event`, `Topic()` returning the row's stored topic), and marks the batch
published — **publish first, mark second**. A crash between the two leaves the row
unpublished, so it replays on the next drain. The relay drains once on `Start`
(this is the across-restart republish), then on a short ticker.

**Delivery semantics: at-least-once.** Each enqueued row is published to the bus
at least once across restarts. Because publish-then-mark can double-publish (and
because the bus is fire-and-forget to async subscribers), **consumers must be
idempotent**, keyed on `Message.ID` — the outbox row's stable id. `outbox.Dedupe`
wraps a handler with a bounded most-recently-seen set to make the common case a
one-liner.

**Retention.** Published rows are pruned by `DeletePublishedBefore` on the
existing hourly maintenance loop (the same place jobs retention runs), so the
table does not grow without bound.

**Ordering.** Rows are fetched and published in insert order (`id` is a
monotonic, gap-tolerant key); per-topic order is preserved by the bus. The relay
does not promise cross-topic global order — events are independent facts.

## Consequences

- The durability seam ADR-0004 mandated now exists. The three deferral triggers
  are satisfied in advance: a cross-process consumer, a durable subscriber, or a
  non-reconstructable event can be added as a plain `bus.Subscribe` whose producer
  enqueues in-tx — no further infrastructure work.
- **The jobs runner is unchanged.** It keeps publishing `job.*` facts directly to
  the bus; its durability is its store + `Recover`, not the outbox. Migrating it
  to publish through the outbox would be a rewrite with no behavioural gain and is
  explicitly out of scope (the ADR-0004 amendment's reasoning stands for that
  path). The outbox is for *new* producers that need cross-restart durable
  delivery.
- At-least-once + idempotent consumers is a weaker contract than exactly-once, but
  exactly-once is unavailable to an in-process async bus without a transactional
  consumer; at-least-once with `Message.ID` dedup is the honest, standard choice.
- A relay goroutine polls the table on a short cadence. With no producers the cost
  is one indexed `WHERE published_at IS NULL` query per interval over an empty
  table — negligible, and the price of having the seam ready.
- This ADR **supersedes the "defer" disposition** of the ADR-0004 amendment (its
  reasoning is preserved there for the record); ADR-0005's "events emit
  post-commit via outbox" line is now satisfied by this relay for producers that
  opt in.

## Implementation status (2026-06-07)

- **Migration `00005_outbox.sql`** — `outbox` table (STRICT): `id` INTEGER PK
  AUTOINCREMENT (monotonic order key), `topic` TEXT, `payload` BLOB, `created_at`
  TEXT, `published_at` TEXT NULL. Partial index on unpublished rows; index on
  `published_at` for retention.
- **`database.OutboxRepository`** (`repository_outbox.go`) — `Enqueue(ctx, tx,
  topic, payload)` (runs on the caller's `*sql.Tx`), `FetchUnpublished(ctx,
  limit)`, `MarkPublished(ctx, ids)`, `DeletePublishedBefore(ctx, cutoff)`.
- **`internal/platform/outbox`** — `Store` port, `Record`, `Message`
  (`events.Event`), `Relay` (`Drain`/`Start`/`Stop`/`Cleanup`), `Dedupe` +
  `Deduper`. Pure; no `database/sql` import.
- **`dbOutboxStore`** (`internal/api/outbox_store.go`) — composition-root adapter
  bridging `outbox.Store` to `database.OutboxRepository`.
- **Wiring** — `BackgroundComponents.Outbox` (drains on Start, short ticker);
  retention on the maintenance loop. Dormant until a producer enqueues.
