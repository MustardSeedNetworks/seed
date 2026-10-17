-- 00038_device_config_push_apply.sql — applying a configuration push plan
-- (P-D3, #3499, ADR-0035 §1-2).
--
-- A plan is applied at most once. applied_at and applied_by are set together
-- when an apply claims the plan, by an UPDATE that only matches an unapplied,
-- unexpired plan, so two applies of one plan cannot both start.
-- apply_outcome is set when that apply ends; it is NULL while the apply runs.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
ALTER TABLE device_config_push_plans ADD COLUMN applied_at TEXT;
ALTER TABLE device_config_push_plans ADD COLUMN applied_by TEXT
	CHECK (applied_by IS NULL OR applied_by <> '');
ALTER TABLE device_config_push_plans ADD COLUMN apply_outcome TEXT
	CHECK (apply_outcome IS NULL OR apply_outcome IN
		('confirmed','rolled_back','reverting','config_drifted','failed'));

-- +goose Down
ALTER TABLE device_config_push_plans DROP COLUMN apply_outcome;
ALTER TABLE device_config_push_plans DROP COLUMN applied_by;
ALTER TABLE device_config_push_plans DROP COLUMN applied_at;
