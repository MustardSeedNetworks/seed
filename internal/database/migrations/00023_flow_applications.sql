-- 00023_flow_applications.sql — application identification from flow (P-C4).
--
-- The collector names each flow's application as it stores it, from the
-- signature table in effect at that moment (internal/appid). Editing the
-- table changes how later flows are named, not earlier ones: a stored row
-- records what the table said when the flow arrived. Rows stored before
-- this migration are unknown.
--
-- The rollups keep per-application totals past the raw horizon, on the same
-- tier horizons as the conversation rollups beside them.

-- +goose Up
ALTER TABLE flow_records ADD COLUMN application TEXT NOT NULL DEFAULT 'unknown';

CREATE TABLE flow_applications_hourly (
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	application TEXT NOT NULL,
	hour_bucket TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	PRIMARY KEY (client_id, application, hour_bucket)
) STRICT;
CREATE INDEX idx_flow_applications_hourly_bucket ON flow_applications_hourly(hour_bucket);

CREATE TABLE flow_applications_daily (
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	application TEXT NOT NULL,
	day_bucket TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	PRIMARY KEY (client_id, application, day_bucket)
) STRICT;
CREATE INDEX idx_flow_applications_daily_bucket ON flow_applications_daily(day_bucket);

-- +goose Down
DROP TABLE flow_applications_daily;
DROP TABLE flow_applications_hourly;
ALTER TABLE flow_records DROP COLUMN application;
