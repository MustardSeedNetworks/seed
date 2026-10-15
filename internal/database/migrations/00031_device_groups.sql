-- 00031_device_groups.sql — device groups (P-D10).
--
-- A group is a client's named set of polling targets. An alert rule may name
-- a group, and then fires only for events from the group's members.
--
-- A member is a polling target; deleting the target removes it from every
-- group. A group that an alert rule names cannot be deleted (RESTRICT): an
-- operator who meant "alert on these devices" must not silently get "alert on
-- every device" because the group went away.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE device_groups (
				id          TEXT NOT NULL PRIMARY KEY,
				client_id   TEXT NOT NULL REFERENCES clients(id),
				name        TEXT NOT NULL CHECK (name <> ''),
				description TEXT NOT NULL DEFAULT '',
				created_at  TEXT NOT NULL,
				updated_at  TEXT NOT NULL,
				UNIQUE (client_id, name)
			) STRICT;

CREATE TABLE device_group_members (
				group_id  TEXT NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
				target_id TEXT NOT NULL REFERENCES polling_targets(id) ON DELETE CASCADE,
				PRIMARY KEY (group_id, target_id)
			) STRICT;
CREATE INDEX idx_device_group_members_target ON device_group_members(target_id);

ALTER TABLE alert_rules ADD COLUMN group_id TEXT REFERENCES device_groups(id) ON DELETE RESTRICT;

-- +goose Down
-- SQLite cannot drop a column that is part of a foreign key, so alert_rules is
-- rebuilt in its 00030 shape. Nothing references alert_rules. A rule scoped to
-- a group is dropped rather than widened to every device.
CREATE TABLE alert_rules_old (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),
				match_kind TEXT,
				match_severity TEXT,
				match_payload_contains TEXT,
				alert_type TEXT NOT NULL,
				alert_severity TEXT NOT NULL,
				alert_title TEXT NOT NULL,
				alert_message TEXT NOT NULL,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			, window_seconds INTEGER NOT NULL DEFAULT 0, threshold_count INTEGER NOT NULL DEFAULT 1) STRICT;
INSERT INTO alert_rules_old (id, name, enabled, match_kind, match_severity, match_payload_contains,
	alert_type, alert_severity, alert_title, alert_message, created_at, updated_at,
	window_seconds, threshold_count)
SELECT id, name, enabled, match_kind, match_severity, match_payload_contains,
	alert_type, alert_severity, alert_title, alert_message, created_at, updated_at,
	window_seconds, threshold_count
FROM alert_rules WHERE group_id IS NULL;
DROP TABLE alert_rules;
ALTER TABLE alert_rules_old RENAME TO alert_rules;
DROP TABLE device_group_members;
DROP TABLE device_groups;
