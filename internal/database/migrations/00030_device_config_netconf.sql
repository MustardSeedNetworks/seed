-- 00030_device_config_netconf.sql — configuration backup over NETCONF (P-D1,
-- #348).
--
-- device_config_targets admits a fifth platform, 'netconf': any vendor's
-- NETCONF server, whose running datastore is read with <get-config> over SSH.
-- SQLite cannot alter a CHECK in place, so the table is rebuilt; NO
-- TRANSACTION + foreign_keys=OFF for the reason 00011 gives
-- (device_config_backups references the targets with ON DELETE CASCADE, and
-- dropping the old table with enforcement on would delete every backup).
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose NO TRANSACTION

-- +goose Up
PRAGMA foreign_keys=OFF;
BEGIN;

CREATE TABLE device_config_targets_new (
				id              TEXT NOT NULL PRIMARY KEY,
				client_id       TEXT NOT NULL REFERENCES clients(id),
				name            TEXT NOT NULL CHECK (name <> ''),
				host            TEXT NOT NULL CHECK (host <> ''),
				port            INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
				platform        TEXT NOT NULL CHECK (platform IN ('cisco-ios','cisco-nxos','arista-eos','juniper-junos','netconf')),
				credentials_id  TEXT NOT NULL,
				host_key_sha256 TEXT CHECK (host_key_sha256 IS NULL OR host_key_sha256 GLOB 'SHA256:*'),
				pending_host_key_sha256 TEXT CHECK (pending_host_key_sha256 IS NULL OR pending_host_key_sha256 GLOB 'SHA256:*'),
				created_at      TEXT NOT NULL,
				updated_at      TEXT NOT NULL,
				UNIQUE (client_id, id),
				UNIQUE (client_id, host, port),
				FOREIGN KEY (client_id, credentials_id)
					REFERENCES device_credentials(client_id, id) ON DELETE RESTRICT
			) STRICT;
INSERT INTO device_config_targets_new SELECT * FROM device_config_targets;
DROP TABLE device_config_targets;
ALTER TABLE device_config_targets_new RENAME TO device_config_targets;
CREATE INDEX idx_device_config_targets_credentials ON device_config_targets(client_id, credentials_id);

COMMIT;
PRAGMA foreign_keys=ON;

-- +goose Down
PRAGMA foreign_keys=OFF;
BEGIN;

-- The previous schema has no netconf platform, so those targets and their
-- backups cannot survive the rollback. Enforcement is off, so the cascade
-- that would remove the backups does not run.
DELETE FROM device_config_backups
	WHERE (client_id, target_id) IN (SELECT client_id, id FROM device_config_targets WHERE platform = 'netconf');
DELETE FROM device_config_targets WHERE platform = 'netconf';

CREATE TABLE device_config_targets_old (
				id              TEXT NOT NULL PRIMARY KEY,
				client_id       TEXT NOT NULL REFERENCES clients(id),
				name            TEXT NOT NULL CHECK (name <> ''),
				host            TEXT NOT NULL CHECK (host <> ''),
				port            INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
				platform        TEXT NOT NULL CHECK (platform IN ('cisco-ios','cisco-nxos','arista-eos','juniper-junos')),
				credentials_id  TEXT NOT NULL,
				host_key_sha256 TEXT CHECK (host_key_sha256 IS NULL OR host_key_sha256 GLOB 'SHA256:*'),
				pending_host_key_sha256 TEXT CHECK (pending_host_key_sha256 IS NULL OR pending_host_key_sha256 GLOB 'SHA256:*'),
				created_at      TEXT NOT NULL,
				updated_at      TEXT NOT NULL,
				UNIQUE (client_id, id),
				UNIQUE (client_id, host, port),
				FOREIGN KEY (client_id, credentials_id)
					REFERENCES device_credentials(client_id, id) ON DELETE RESTRICT
			) STRICT;
INSERT INTO device_config_targets_old SELECT * FROM device_config_targets;
DROP TABLE device_config_targets;
ALTER TABLE device_config_targets_old RENAME TO device_config_targets;
CREATE INDEX idx_device_config_targets_credentials ON device_config_targets(client_id, credentials_id);

COMMIT;
PRAGMA foreign_keys=ON;
