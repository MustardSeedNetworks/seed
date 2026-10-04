-- 00019_flow_records.sql — decoded NetFlow v5/v9 and IPFIX flows (P-C1, #3082).
--
-- One row per flow record as the exporter sent it. Times are UTC with fixed
-- millisecond precision so they sort as text. Counters are the exporter's,
-- scaled by the v5 header's sampling interval. The retention engine purges
-- rows past the raw horizon.

-- +goose Up
CREATE TABLE flow_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	exporter TEXT NOT NULL,
	version INTEGER NOT NULL,
	observation_domain INTEGER NOT NULL,
	flow_start TEXT NOT NULL,
	flow_end TEXT NOT NULL,
	src_addr TEXT NOT NULL,
	dst_addr TEXT NOT NULL,
	src_port INTEGER NOT NULL,
	dst_port INTEGER NOT NULL,
	protocol INTEGER NOT NULL,
	tcp_flags INTEGER NOT NULL,
	bytes INTEGER NOT NULL,
	packets INTEGER NOT NULL,
	input_if INTEGER NOT NULL,
	output_if INTEGER NOT NULL,
	received_at TEXT NOT NULL
) STRICT;
CREATE INDEX idx_flow_records_end ON flow_records(flow_end);
CREATE INDEX idx_flow_records_exporter_end ON flow_records(exporter, flow_end);

-- +goose Down
DROP TABLE flow_records;
