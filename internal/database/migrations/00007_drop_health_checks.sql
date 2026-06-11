-- 00007_drop_health_checks.sql — delete the dead health-check read-path stack
-- (ADR-0026). The health_check_results table and its hourly/daily rollups were
-- never written: HealthCheckRepository.Record/RecordBatch had zero production
-- callers, so scoring, SLA, and alerting all read an empty table. The active
-- monitoring engine is internal/probe (probe_results time series + breach→anomaly
-- pipeline, ADR-0025); anomalies persist to the unified `anomalies` table
-- (ADR-0021). These three tables and their indexes are removed as dead schema.
-- Indexes drop with their table in SQLite.
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
DROP TABLE IF EXISTS health_check_rollups_hourly;
DROP TABLE IF EXISTS health_check_rollups_daily;
DROP TABLE IF EXISTS health_check_results;

-- +goose Down
CREATE TABLE health_check_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				check_type TEXT NOT NULL,
				endpoint_name TEXT NOT NULL,
				endpoint_target TEXT NOT NULL,
				success INTEGER NOT NULL CHECK (success IN (0,1)),
				latency_ms REAL,
				status_code INTEGER,
				error_message TEXT,
				metadata_json TEXT,
				recorded_at TEXT NOT NULL
			) STRICT;
CREATE TABLE health_check_rollups_daily (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				check_type TEXT NOT NULL,
				endpoint_name TEXT NOT NULL,
				day_bucket TEXT NOT NULL,
				total_checks INTEGER NOT NULL,
				successful_checks INTEGER NOT NULL,
				avg_latency_ms REAL,
				min_latency_ms REAL,
				max_latency_ms REAL,
				p95_latency_ms REAL,
				availability_percent REAL
			) STRICT;
CREATE TABLE health_check_rollups_hourly (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				check_type TEXT NOT NULL,
				endpoint_name TEXT NOT NULL,
				hour_bucket TEXT NOT NULL,
				total_checks INTEGER NOT NULL,
				successful_checks INTEGER NOT NULL,
				avg_latency_ms REAL,
				min_latency_ms REAL,
				max_latency_ms REAL,
				p95_latency_ms REAL
			) STRICT;
CREATE INDEX idx_health_check_endpoint_time ON health_check_results(endpoint_name, recorded_at);
CREATE INDEX idx_health_check_recorded ON health_check_results(recorded_at);
CREATE INDEX idx_health_check_type_time ON health_check_results(check_type, recorded_at);
CREATE INDEX idx_health_daily_bucket ON health_check_rollups_daily(day_bucket);
CREATE UNIQUE INDEX idx_health_daily_unique
				ON health_check_rollups_daily(check_type, endpoint_name, day_bucket);
CREATE INDEX idx_health_hourly_bucket ON health_check_rollups_hourly(hour_bucket);
CREATE UNIQUE INDEX idx_health_hourly_unique
				ON health_check_rollups_hourly(check_type, endpoint_name, hour_bucket);
