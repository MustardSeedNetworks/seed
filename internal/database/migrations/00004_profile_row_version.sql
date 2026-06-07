-- 00004_profile_row_version.sql — dedicated optimistic-concurrency token for
-- profiles (ADR re-arch Phase 5 hardening). #1559 used updated_at (RFC3339,
-- second precision) as the profile ETag, so two writes inside the same second
-- shared a token and the conflict went undetected — degrading to last-write-wins
-- in that sub-second window. A monotonic row_version, bumped on every write,
-- closes the window: the token is exact, not time-derived. The NOT NULL DEFAULT
-- backfills the single shipped/seeded profile to 1. STRICT-consistent with the
-- 0001 baseline. Regenerate the gate golden:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
ALTER TABLE profiles ADD COLUMN row_version INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE profiles DROP COLUMN row_version;
