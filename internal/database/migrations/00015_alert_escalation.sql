-- 00015_alert_escalation.sql — how far up its rule's escalation ladder an
-- alert has been sent (P-B2, #3032).
--
-- The stage lives on the alert rather than in the escalator's memory, so a
-- restart neither re-sends a stage nor forgets one. 0 means the alert has
-- not been escalated, which is true of every alert raised before this
-- migration. `escalated_at` is when the last stage (or its repeat) went out;
-- the repeat period is measured from it.

-- +goose Up
ALTER TABLE alerts ADD COLUMN escalation_stage INTEGER NOT NULL DEFAULT 0;
ALTER TABLE alerts ADD COLUMN escalated_at TEXT;

-- The escalator's only read: open alerts raised by a rule that has a ladder.
CREATE INDEX IF NOT EXISTS idx_alerts_open_rule
    ON alerts(rule) WHERE acknowledged = 0 AND resolved = 0;

-- +goose Down
DROP INDEX IF EXISTS idx_alerts_open_rule;
ALTER TABLE alerts DROP COLUMN escalated_at;
ALTER TABLE alerts DROP COLUMN escalation_stage;
