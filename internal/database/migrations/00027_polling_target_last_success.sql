-- 00027_polling_target_last_success.sql — when a target last answered (seed#2523).
--
-- last_polled_at and last_status describe only the most recent attempt, so a
-- target that is failing now cannot say when it last worked. last_success_at
-- is stamped by every successful poll and left alone by a failed one. Rows
-- whose last poll succeeded start with that poll's time.

-- +goose Up
ALTER TABLE polling_targets ADD COLUMN last_success_at TEXT;
UPDATE polling_targets SET last_success_at = last_polled_at WHERE last_status = 'ok';

-- +goose Down
ALTER TABLE polling_targets DROP COLUMN last_success_at;
