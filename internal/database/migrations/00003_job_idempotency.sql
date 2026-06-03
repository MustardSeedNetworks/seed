-- 00003_job_idempotency.sql — durable Idempotency-Key store (ADR-0005, Phase 5c-4).
-- The POST /jobs Idempotency-Key dedup shipped (Phase 4) as a bounded in-memory
-- FIFO cache — lost on restart, so a client retry across a restart could create
-- a duplicate job. This table makes it durable: a key maps to the job it
-- created plus a hash of the request, so a replay is detected and a reused key
-- with a different body conflicts. ON DELETE CASCADE ties a key's lifetime to
-- its job — when retention deletes the job, its key goes too (a later replay
-- then misses and creates afresh, which the handler already tolerates).
-- STRICT, consistent with the 0001/0002 baselines. Regenerate the gate golden:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE job_idempotency (
	key          TEXT NOT NULL PRIMARY KEY,
	request_hash TEXT NOT NULL,
	job_id       TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
	created_at   TEXT NOT NULL
) STRICT;

CREATE INDEX idx_job_idempotency_job ON job_idempotency(job_id);

-- +goose Down
DROP TABLE IF EXISTS job_idempotency;
