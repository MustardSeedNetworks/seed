-- 00021_flow_records_format.sql — flow_records names its export format
-- (P-C2, #3090).
--
-- The version column held the datagram's version number: 5, 9 or 10. sFlow
-- v5 is also version 5, so the number no longer says which protocol a row
-- came from. The column becomes format, a closed set of names. SQLite cannot
-- change a STRICT column's type in place, so the table is rebuilt.

-- +goose Up
CREATE TABLE flow_records_new (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	exporter TEXT NOT NULL,
	format TEXT NOT NULL CHECK (format IN ('netflow5', 'netflow9', 'ipfix', 'sflow5')),
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
INSERT INTO flow_records_new
	(id, client_id, exporter, format, observation_domain, flow_start, flow_end,
	 src_addr, dst_addr, src_port, dst_port, protocol, tcp_flags, bytes, packets,
	 input_if, output_if, received_at)
SELECT id, client_id, exporter,
	CASE version WHEN 5 THEN 'netflow5' WHEN 9 THEN 'netflow9' ELSE 'ipfix' END,
	observation_domain, flow_start, flow_end, src_addr, dst_addr, src_port,
	dst_port, protocol, tcp_flags, bytes, packets, input_if, output_if, received_at
FROM flow_records;
DROP TABLE flow_records;
ALTER TABLE flow_records_new RENAME TO flow_records;
CREATE INDEX idx_flow_records_end ON flow_records(flow_end);
CREATE INDEX idx_flow_records_exporter_end ON flow_records(exporter, flow_end);

-- +goose Down
CREATE TABLE flow_records_old (
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
-- sFlow rows have no version-number form and are dropped.
INSERT INTO flow_records_old
	(id, client_id, exporter, version, observation_domain, flow_start, flow_end,
	 src_addr, dst_addr, src_port, dst_port, protocol, tcp_flags, bytes, packets,
	 input_if, output_if, received_at)
SELECT id, client_id, exporter,
	CASE format WHEN 'netflow5' THEN 5 WHEN 'netflow9' THEN 9 ELSE 10 END,
	observation_domain, flow_start, flow_end, src_addr, dst_addr, src_port,
	dst_port, protocol, tcp_flags, bytes, packets, input_if, output_if, received_at
FROM flow_records WHERE format <> 'sflow5';
DROP TABLE flow_records;
ALTER TABLE flow_records_old RENAME TO flow_records;
CREATE INDEX idx_flow_records_end ON flow_records(flow_end);
CREATE INDEX idx_flow_records_exporter_end ON flow_records(exporter, flow_end);
