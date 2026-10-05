-- 00025_voip_streams.sql — RTP stream quality from the VoIP analyser (P-A7).
--
-- One row is one window of one RTP stream: a direction of a call, keyed by
-- its addresses and SSRC, over at most a minute. The analyser finds streams
-- without signalling, so there is no call identifier to join the two
-- directions on. delay_ms is what the E-model was given (packetization,
-- encoder lookahead and the jitter buffer); a passive probe cannot see the
-- network's one-way delay, so it is not included.

-- +goose Up
CREATE TABLE voip_streams (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
	interface_name TEXT NOT NULL,
	src_addr TEXT NOT NULL,
	dst_addr TEXT NOT NULL,
	ssrc INTEGER NOT NULL,
	codec TEXT NOT NULL,
	started_at TEXT NOT NULL,
	ended_at TEXT NOT NULL,
	packets_expected INTEGER NOT NULL,
	packets_received INTEGER NOT NULL,
	loss_pct REAL NOT NULL,
	jitter_ms REAL NOT NULL,
	max_jitter_ms REAL NOT NULL,
	delay_ms REAL NOT NULL,
	r_factor REAL NOT NULL,
	mos REAL NOT NULL
) STRICT;
CREATE INDEX idx_voip_streams_started ON voip_streams(started_at);
CREATE INDEX idx_voip_streams_client ON voip_streams(client_id);
CREATE INDEX idx_voip_streams_mos ON voip_streams(mos);

-- +goose Down
DROP TABLE voip_streams;
