-- 00014_alert_deliveries.sql — one delivery outcome per alert per channel
-- (#2997).
--
-- 00013 put the outcome on the alert row because the webhook was the only
-- transport. Email is the second, and outbound syslog and SNMP traps follow;
-- with one slot, a working webhook would overwrite a mail relay that refuses
-- every message, which is exactly the silent failure 00013 existed to end.
--
-- No row for a channel still means delivery never applied there, never
-- "failed": most installs configure no receiver at all. The rows 00013 wrote
-- are carried over as the webhook's, the only channel that could have written
-- them.

-- +goose Up
CREATE TABLE alert_deliveries (
    alert_id     INTEGER NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    channel      TEXT NOT NULL,
    status       TEXT NOT NULL,
    attempted_at TEXT,
    error        TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (alert_id, channel)
) STRICT, WITHOUT ROWID;

-- The question the inbox asks is "is anything failing to leave the box".
CREATE INDEX idx_alert_deliveries_status ON alert_deliveries(status);

INSERT INTO alert_deliveries (alert_id, channel, status, attempted_at, error)
SELECT id, 'webhook', delivery_status, delivery_attempted_at, delivery_error
FROM alerts WHERE delivery_status != '';

DROP INDEX IF EXISTS idx_alerts_delivery_status;
ALTER TABLE alerts DROP COLUMN delivery_error;
ALTER TABLE alerts DROP COLUMN delivery_attempted_at;
ALTER TABLE alerts DROP COLUMN delivery_status;

-- +goose Down
ALTER TABLE alerts ADD COLUMN delivery_status TEXT NOT NULL DEFAULT '';
ALTER TABLE alerts ADD COLUMN delivery_attempted_at TEXT;
ALTER TABLE alerts ADD COLUMN delivery_error TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_alerts_delivery_status
    ON alerts(delivery_status) WHERE delivery_status != '';

UPDATE alerts SET
    delivery_status = d.status,
    delivery_attempted_at = d.attempted_at,
    delivery_error = d.error
FROM alert_deliveries d
WHERE d.alert_id = alerts.id AND d.channel = 'webhook';

DROP TABLE alert_deliveries;
