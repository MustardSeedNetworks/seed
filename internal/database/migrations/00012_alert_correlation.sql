-- 00012_alert_correlation.sql — give an alert the two facts needed to say what
-- probably caused it (#409, S6-2 slice 2).
--
-- `rule` records which pipeline rule raised the alert. The pipelines have
-- always known this (`fire(ctx, "bgp.flap", ...)`) and always dropped it on the
-- floor, so the only thing left on the row that hinted at the rule was the
-- title — prose written for a human, which changes whenever the copy is
-- improved. Anything reasoning about why an alert fired had to pattern-match
-- English. Now it does not.
--
-- `root_cause_id` points at an earlier alert that explains this one. It is a
-- self-reference into the same table: a BGP session that drops because the
-- interface carrying it went down names that interface's alert. Correlation
-- annotates, it never replaces — both alerts are still stored and still
-- delivered, so the inbox remains the complete record.
--
-- ON DELETE SET NULL, because alert retention prunes by age: the cause is
-- older than the effect and is deleted first, and losing the explanation must
-- never take the alert with it.
--
-- Both columns are nullable/defaulted, so existing rows stay valid: alerts
-- raised before this migration carry no rule and no cause, which is honest —
-- nothing recorded either at the time.

-- +goose Up
ALTER TABLE alerts ADD COLUMN rule TEXT NOT NULL DEFAULT '';
ALTER TABLE alerts ADD COLUMN root_cause_id INTEGER REFERENCES alerts(id) ON DELETE SET NULL;

-- Correlation reads "the most recent alert for this rule on this device", and
-- the inbox will want "everything this alert explains".
CREATE INDEX IF NOT EXISTS idx_alerts_rule_source ON alerts(rule, source, created_at);
CREATE INDEX IF NOT EXISTS idx_alerts_root_cause ON alerts(root_cause_id);

-- +goose Down
DROP INDEX IF EXISTS idx_alerts_root_cause;
DROP INDEX IF EXISTS idx_alerts_rule_source;
ALTER TABLE alerts DROP COLUMN root_cause_id;
ALTER TABLE alerts DROP COLUMN rule;
