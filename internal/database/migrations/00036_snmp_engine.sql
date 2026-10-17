-- 00036_snmp_engine.sql — Seed's own SNMP engine identity (P-B4).
--
-- Seed is the authoritative engine for every SNMPv3 inform it receives, so
-- senders localize their keys against its snmpEngineID and synchronize with
-- its snmpEngineBoots (RFC 3414 2.2). Both must survive restarts: a new
-- engine ID invalidates every sender's localized keys, and a boots counter
-- that goes backwards reopens the replay window. One row, created on the
-- trap listener's first start; boots is incremented on every later start.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE snmp_engine (
				id        INTEGER NOT NULL PRIMARY KEY CHECK (id = 1),
				engine_id BLOB    NOT NULL CHECK (length(engine_id) BETWEEN 5 AND 32),
				boots     INTEGER NOT NULL CHECK (boots BETWEEN 1 AND 2147483647)
			) STRICT;

-- +goose Down
DROP TABLE snmp_engine;
