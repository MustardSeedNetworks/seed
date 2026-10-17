-- 00037_device_config_push.sql — configuration push plans (P-D3, #3499,
-- ADR-0035).
--
-- Two changes:
--
--   * device_config_targets learns push_credentials_id, the vault credential
--     a push logs in with. NULL means pushing is off, the default. It must be
--     another credential than the backup's, so the read-only backup login
--     never gains write access. SQLite cannot add a composite foreign key to
--     an existing table, so the table is rebuilt; NO TRANSACTION +
--     foreign_keys=OFF for the reason 00011 gives (device_config_backups
--     references the targets with ON DELETE CASCADE, and dropping the old
--     table with enforcement on would delete every backup).
--
--   * device_config_push_plans keeps each plan: the dry-run diff of a
--     candidate against a fresh backup, which an admin may apply until it
--     expires. Only a plan with a change is stored; a plan that would change
--     nothing, or that the target refused, is reported by its job and leaves
--     no row. The candidate and the diff are vault ciphertext: both can carry
--     secrets. backup_id has no foreign key because backups are pruned and a
--     plan outlives nothing it needs: running_sha256 is what apply compares.
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
				push_credentials_id TEXT CHECK (push_credentials_id IS NULL OR push_credentials_id <> credentials_id),
				UNIQUE (client_id, id),
				UNIQUE (client_id, host, port),
				FOREIGN KEY (client_id, credentials_id)
					REFERENCES device_credentials(client_id, id) ON DELETE RESTRICT,
				FOREIGN KEY (client_id, push_credentials_id)
					REFERENCES device_credentials(client_id, id) ON DELETE RESTRICT
			) STRICT;
INSERT INTO device_config_targets_new
	(id, client_id, name, host, port, platform, credentials_id, host_key_sha256,
	 pending_host_key_sha256, created_at, updated_at)
	SELECT id, client_id, name, host, port, platform, credentials_id, host_key_sha256,
	       pending_host_key_sha256, created_at, updated_at
	FROM device_config_targets;
DROP TABLE device_config_targets;
ALTER TABLE device_config_targets_new RENAME TO device_config_targets;
CREATE INDEX idx_device_config_targets_credentials ON device_config_targets(client_id, credentials_id);
CREATE INDEX idx_device_config_targets_push_credentials ON device_config_targets(client_id, push_credentials_id);

CREATE TABLE device_config_push_plans (
				id             TEXT NOT NULL PRIMARY KEY,
				client_id      TEXT NOT NULL,
				target_id      TEXT NOT NULL,
				created_by     TEXT NOT NULL CHECK (created_by <> ''),
				created_at     TEXT NOT NULL,
				expires_at     TEXT NOT NULL CHECK (expires_at > created_at),
				mode           TEXT NOT NULL CHECK (mode IN ('merge','replace')),
				diff_source    TEXT NOT NULL CHECK (diff_source IN ('device','predicted')),
				backup_id      TEXT NOT NULL,
				running_sha256 TEXT NOT NULL,
				candidate_enc  BLOB NOT NULL CHECK (CAST(candidate_enc AS TEXT) GLOB 'enc:v[0-9]*:*'),
				diff_enc       BLOB NOT NULL CHECK (CAST(diff_enc AS TEXT) GLOB 'enc:v[0-9]*:*'),
				diff_sha256    TEXT NOT NULL,
				added          INTEGER NOT NULL CHECK (added >= 0),
				removed        INTEGER NOT NULL CHECK (removed >= 0),
				CHECK (added + removed > 0),
				FOREIGN KEY (client_id, target_id)
					REFERENCES device_config_targets(client_id, id) ON DELETE CASCADE
			) STRICT;
CREATE INDEX idx_device_config_push_plans_target ON device_config_push_plans(client_id, target_id, created_at);

COMMIT;
PRAGMA foreign_keys=ON;

-- +goose Down
PRAGMA foreign_keys=OFF;
BEGIN;

DROP TABLE device_config_push_plans;

CREATE TABLE device_config_targets_old (
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
INSERT INTO device_config_targets_old
	SELECT id, client_id, name, host, port, platform, credentials_id, host_key_sha256,
	       pending_host_key_sha256, created_at, updated_at
	FROM device_config_targets;
DROP TABLE device_config_targets;
ALTER TABLE device_config_targets_old RENAME TO device_config_targets;
CREATE INDEX idx_device_config_targets_credentials ON device_config_targets(client_id, credentials_id);

COMMIT;
PRAGMA foreign_keys=ON;
