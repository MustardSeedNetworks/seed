-- 00022_flow_conversations.sql — hourly and daily flow rollups (P-C3, #3095).
--
-- flow_records keeps the raw horizon; these keep what top talkers and top
-- conversations need past it, on the same tier horizons as the metric and
-- probe rollups. One row per direction of a conversation: (src, dst,
-- protocol) per bucket, the bucket being the hour or day the flow ended in.
-- Ports are not kept, so a bucket holds one row per host pair and protocol
-- however many sessions it carried.

-- +goose Up
CREATE TABLE flow_conversations_hourly (
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	src_addr TEXT NOT NULL,
	dst_addr TEXT NOT NULL,
	protocol INTEGER NOT NULL,
	hour_bucket TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	PRIMARY KEY (client_id, src_addr, dst_addr, protocol, hour_bucket)
) STRICT;
CREATE INDEX idx_flow_conversations_hourly_bucket ON flow_conversations_hourly(hour_bucket);

CREATE TABLE flow_conversations_daily (
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	src_addr TEXT NOT NULL,
	dst_addr TEXT NOT NULL,
	protocol INTEGER NOT NULL,
	day_bucket TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	PRIMARY KEY (client_id, src_addr, dst_addr, protocol, day_bucket)
) STRICT;
CREATE INDEX idx_flow_conversations_daily_bucket ON flow_conversations_daily(day_bucket);

-- +goose Down
DROP TABLE flow_conversations_daily;
DROP TABLE flow_conversations_hourly;
