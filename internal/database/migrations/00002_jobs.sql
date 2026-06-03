-- 00002_jobs.sql — durable job store (ADR-0005, Phase 5c).
-- The platform/jobs Runner shipped (Phase 4) with an in-memory store that loses
-- jobs on restart — the correct fail-cleanly v1. This table is the durable
-- backing the Runner writes lifecycle transitions through, so GET /jobs/{id}
-- survives a restart. The Runner wiring + the transactional outbox land in
-- follow-up 5c slices; this migration + its repository (repository_jobs.go) are
-- additive. STRICT + explicit domain CHECKs, consistent with the 0001 baseline
-- (Phase 5b-3 / 5b-4). Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE jobs (
	id           TEXT NOT NULL PRIMARY KEY,
	kind         TEXT NOT NULL,
	state        TEXT NOT NULL DEFAULT 'queued'
	             CHECK (state IN ('queued','running','succeeded','failed','cancelled')),
	progress     REAL NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 1),
	result_json  TEXT,
	error        TEXT,
	created_at   TEXT NOT NULL,
	updated_at   TEXT NOT NULL,
	completed_at TEXT
) STRICT;

CREATE INDEX idx_jobs_completed ON jobs(completed_at);
CREATE INDEX idx_jobs_created ON jobs(created_at);
CREATE INDEX idx_jobs_kind ON jobs(kind);
CREATE INDEX idx_jobs_state ON jobs(state);

-- +goose Down
DROP TABLE IF EXISTS jobs;
