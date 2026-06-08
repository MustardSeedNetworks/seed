-- 00005_outbox.sql — transactional outbox (ADR-0017, closes Phase 5).
-- A producer that needs durable, cross-restart event delivery writes one row
-- here in the SAME transaction as its domain change (database.OutboxRepository
-- .Enqueue runs on the caller's *sql.Tx). The row and the domain row commit or
-- roll back together — that atomicity is the entire correctness guarantee. A
-- post-commit relay (internal/platform/outbox.Relay) drains unpublished rows,
-- republishes them onto the in-process bus, and stamps published_at; published
-- rows are pruned on the maintenance loop's retention tick. At-least-once
-- delivery — consumers dedupe on the row id (see outbox.Dedupe). Additive: no
-- existing producer is rewired; the jobs runner keeps publishing directly.
-- STRICT + explicit CHECKs, consistent with the 0001 baseline. Regenerate the
-- gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE outbox (
	-- AUTOINCREMENT so ids are monotonic and never reused after retention
	-- deletes — the dedup key (Message.ID) must stay stable and collision-free.
	id           INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	topic        TEXT NOT NULL CHECK (topic <> ''),
	payload      BLOB NOT NULL,
	created_at   TEXT NOT NULL,
	published_at TEXT
) STRICT;

-- The relay's hot path: fetch unpublished rows in insert order. Partial index
-- keeps it tiny — it only spans the backlog, not the published history.
CREATE INDEX idx_outbox_unpublished ON outbox(id) WHERE published_at IS NULL;
-- Retention sweep: delete published rows older than the cutoff.
CREATE INDEX idx_outbox_published ON outbox(published_at);

-- +goose Down
DROP TABLE IF EXISTS outbox;
