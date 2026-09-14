-- 00013_alert_delivery_status.sql — record, per alert, what happened when Seed
-- tried to send it to the operator's webhook receiver (#368, S6-2 slice 3).
--
-- Until now the outcome of a delivery lived only in counters on the in-memory
-- Notifier, and nothing served those counters on any route. A receiver that
-- had been refusing every POST for a week looked exactly like a receiver that
-- was working: the inbox showed the alert either way. #368's acceptance calls
-- for the opposite — "endpoint down -> bounded retry, then a visible failed
-- status" — and the alert row is where an operator already looks.
--
-- `delivery_status` is deliberately empty by default and empty means *not
-- applicable*, never "failed". Most installs configure no receiver at all
-- (unset is silent, by design), and alerts raised before this migration were
-- never offered to one; painting either as a failure would put a warning on
-- every row of every air-gapped inbox. The states that mean something are
-- pending, delivered, failed and dropped.
--
-- `delivery_error` carries the last attempt's error text so the operator can
-- tell a DNS failure from a 401 without reading the daemon log, and
-- `delivery_attempted_at` says when that was — a stale timestamp is itself the
-- evidence that deliveries stopped.

-- +goose Up
ALTER TABLE alerts ADD COLUMN delivery_status TEXT NOT NULL DEFAULT '';
ALTER TABLE alerts ADD COLUMN delivery_attempted_at TEXT;
ALTER TABLE alerts ADD COLUMN delivery_error TEXT NOT NULL DEFAULT '';

-- The question the inbox asks is "is anything failing to leave the box", which
-- is a scan over a column that is empty on most rows; the partial index keeps
-- it off the ones delivery never touched.
CREATE INDEX IF NOT EXISTS idx_alerts_delivery_status
    ON alerts(delivery_status) WHERE delivery_status != '';

-- +goose Down
DROP INDEX IF EXISTS idx_alerts_delivery_status;
ALTER TABLE alerts DROP COLUMN delivery_error;
ALTER TABLE alerts DROP COLUMN delivery_attempted_at;
ALTER TABLE alerts DROP COLUMN delivery_status;
