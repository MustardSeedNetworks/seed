-- 00008_anomaly_rollups_daily.sql — daily census rollup of the anomaly store
-- (ADR-0028). Realizes the deferred "daily rollups" clause of ADR-0021's locked
-- retention model (TTL + daily rollups). The `anomalies` table (00006) is
-- coalesced — one mutable row per (def, subject) carrying a cumulative count and
-- lifecycle — so it has NO per-occurrence event log to GROUP BY. The only
-- faithful daily artifact is therefore a CENSUS: each UTC day, one row per
-- (def, subject) whose lifecycle intersects that day, snapshotting the facts the
-- live row still holds. This long-term trend survives the 90-day TTL purge of
-- resolved anomalies (ADR-0021 / migration 00006), with bounded appliance growth
-- via the DailyDays tier horizon (purged in the same RunCleanup pass that writes
-- the census — census first, purge second).
--
-- NOT a timeseries RollupSource: there is no immutable raw stream, no hourly
-- tier, and the Phase-2 TTL cleanup already owns deletion of resolved rows
-- (ADR-0028 "Alternatives considered"). One daily table, one INSERT OR REPLACE.
--
-- Per-day occurrence counting (ADR-0028 §5, V1.0): count_cumulative stores the
-- live row's cumulative count at census time; exact per-day deltas are derived by
-- differencing consecutive day rows for the same (def, subject) at query time.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE anomaly_rollups_daily (
	day_bucket       TEXT    NOT NULL,                      -- YYYY-MM-DD UTC (retention.dayFormat)
	def_key          TEXT    NOT NULL CHECK (def_key <> ''),
	source           TEXT    NOT NULL CHECK (source <> ''), -- wifi|wired|snmp|bluetooth|health|security|autotest
	category         TEXT    NOT NULL CHECK (category <> ''),
	subject_kind     TEXT    NOT NULL CHECK (subject_kind <> ''),
	subject_id       TEXT    NOT NULL,
	max_severity     TEXT    NOT NULL CHECK (max_severity <> ''), -- highest severity held as of the census
	count_cumulative INTEGER NOT NULL CHECK (count_cumulative >= 0),
	first_seen       TEXT    NOT NULL,                       -- RFC3339, carried from the live row
	last_seen        TEXT    NOT NULL,                       -- RFC3339, carried from the live row
	is_resolved      INTEGER NOT NULL DEFAULT 0 CHECK (is_resolved IN (0, 1)),
	resolved_at      TEXT,                                   -- RFC3339, NULL while active
	PRIMARY KEY (day_bucket, def_key, subject_kind, subject_id) -- idempotent re-census
) STRICT;

-- Purge + day-range queries scan by bucket; cross-source correlation (the point
-- of the subject taxonomy) scans by subject.
CREATE INDEX idx_anomaly_rollups_daily_bucket ON anomaly_rollups_daily(day_bucket);
CREATE INDEX idx_anomaly_rollups_daily_subject ON anomaly_rollups_daily(subject_kind, subject_id);

-- +goose Down
DROP TABLE IF EXISTS anomaly_rollups_daily;
