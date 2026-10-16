-- 00035_autotest_results.sql — AutoTest run records (P-E2).
--
-- Every completed AutoTest is kept, with the jack label and asset tag that tie
-- it to a physical port. stages is the run's per-stage JSON as the API returns
-- it. A record keeps its id through export and re-import, so importing an
-- export restores the same row.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE autotest_results (
				id          TEXT NOT NULL PRIMARY KEY,
				interface   TEXT NOT NULL,
				jack_label  TEXT NOT NULL DEFAULT '',
				asset_tag   TEXT NOT NULL DEFAULT '',
				verdict     TEXT NOT NULL CHECK (verdict IN ('pass', 'warn', 'fail')),
				started_at  TEXT NOT NULL,
				duration_ms INTEGER NOT NULL CHECK (duration_ms >= 0),
				stages      TEXT NOT NULL
			) STRICT;
CREATE INDEX idx_autotest_results_started_at ON autotest_results(started_at);

-- +goose Down
DROP TABLE autotest_results;
