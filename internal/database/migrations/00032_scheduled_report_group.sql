-- 00032_scheduled_report_group.sql — a device group on a scheduled report (P-D10).
--
-- A scheduled capacity or summary report may name a device group, and then
-- covers only the group's members as they stand when it runs. The group cannot
-- be deleted while a schedule names it (RESTRICT), for the same reason as
-- alert_rules.group_id: a report meant for these devices must not silently
-- become a report on every device.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
ALTER TABLE scheduled_reports ADD COLUMN group_id TEXT REFERENCES device_groups(id) ON DELETE RESTRICT;

-- +goose Down
-- SQLite cannot drop a column that is part of a foreign key, so
-- scheduled_reports is rebuilt in its 00001 shape. A schedule scoped to a
-- group is dropped rather than widened to every device.
CREATE TABLE scheduled_reports_old (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				template TEXT NOT NULL,
				format TEXT NOT NULL,
				schedule_json TEXT NOT NULL,
				parameters_json TEXT,
				recipients_json TEXT,
				enabled INTEGER DEFAULT 1 CHECK (enabled IN (0,1)),
				last_run TEXT,
				next_run TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			) STRICT;
INSERT INTO scheduled_reports_old (id, name, template, format, schedule_json, parameters_json,
	recipients_json, enabled, last_run, next_run, created_at, updated_at)
SELECT id, name, template, format, schedule_json, parameters_json,
	recipients_json, enabled, last_run, next_run, created_at, updated_at
FROM scheduled_reports WHERE group_id IS NULL;
DROP TABLE scheduled_reports;
ALTER TABLE scheduled_reports_old RENAME TO scheduled_reports;
CREATE INDEX idx_scheduled_reports_enabled ON scheduled_reports(enabled);
CREATE INDEX idx_scheduled_reports_next_run ON scheduled_reports(next_run);
