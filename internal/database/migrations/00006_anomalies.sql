-- 00006_anomalies.sql — persistent anomaly store (ADR-0021, Anomaly Platform
-- phase 1). One source-neutral table is the system of record for every detected
-- anomaly across all producers (Wi-Fi, wired, SNMP, Bluetooth, health, security,
-- AutoTest), completing ADR-0011's deferred persistence clause. The engine's
-- persistence Coordinator (internal/anomaly) writes rows through here on a
-- material change (new instance / severity escalation) and in batches from its
-- periodic Flush; resolution is written by MarkResolved on prune.
--
-- One row per live instance, keyed by the stable id `defKey|subjectKind|subjectId`
-- (the engine's in-memory coalescing key) so a re-detection updates the same row
-- and a restart re-loads it idempotently. Catalog-static fields (impact,
-- follow-ups) are NOT stored — they are re-derived from the embedded catalog by
-- def_key on read, so the store never duplicates static copy. evidence/standards
-- are per-instance/JSON. STRICT + explicit CHECKs, consistent with the baseline.
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE anomalies (
	id              TEXT    NOT NULL PRIMARY KEY,          -- defKey|subjectKind|subjectId
	def_key         TEXT    NOT NULL CHECK (def_key <> ''),
	source          TEXT    NOT NULL CHECK (source <> ''), -- wifi|wired|snmp|bluetooth|health|security|autotest
	category        TEXT    NOT NULL CHECK (category <> ''),
	severity        TEXT    NOT NULL CHECK (severity <> ''),
	subject_kind    TEXT    NOT NULL CHECK (subject_kind <> ''),
	subject_id      TEXT    NOT NULL,
	title           TEXT    NOT NULL,
	description     TEXT    NOT NULL,
	recommendation  TEXT    NOT NULL,
	evidence        TEXT,                                  -- JSON object (map[string]string)
	standards       TEXT,                                  -- JSON array (IEEE/RFC cites)
	count           INTEGER NOT NULL CHECK (count >= 0),
	first_seen      TEXT    NOT NULL,                       -- RFC3339
	last_seen       TEXT    NOT NULL,                       -- RFC3339
	resolved_at     TEXT,                                   -- RFC3339, NULL while active
	is_resolved     INTEGER NOT NULL DEFAULT 0 CHECK (is_resolved IN (0, 1)),
	acknowledged_by TEXT,
	acknowledged_at TEXT
) STRICT;

-- Query patterns: recency, by producer, by subject (cross-source correlation),
-- by severity, and the active/resolved split.
CREATE INDEX idx_anomalies_last_seen ON anomalies(last_seen);
CREATE INDEX idx_anomalies_source ON anomalies(source);
CREATE INDEX idx_anomalies_subject ON anomalies(subject_kind, subject_id);
CREATE INDEX idx_anomalies_severity ON anomalies(severity);
-- Active-instance scan (LoadActive on start) stays tiny — spans only unresolved rows.
CREATE INDEX idx_anomalies_active ON anomalies(id) WHERE is_resolved = 0;

-- +goose Down
DROP TABLE IF EXISTS anomalies;
