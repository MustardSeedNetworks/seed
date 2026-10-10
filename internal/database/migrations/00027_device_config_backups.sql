-- 00026_device_config_backups.sql — device configuration backup over SSH (P-D1,
-- #348).
--
-- Two changes:
--
--   * The credential vault learns a third kind, 'ssh': a user and a password,
--     both stored the way the SNMP secrets are (versioned ciphertext, never
--     plaintext). The kinds stay mutually exclusive, so an SSH row carries no
--     SNMP column and an SNMP row no SSH column. SQLite cannot alter a CHECK in
--     place, so the table is rebuilt; NO TRANSACTION + foreign_keys=OFF for the
--     reason 00011 gives (polling_targets references the vault, and dropping
--     the old table with enforcement on would fire that reference).
--
--   * device_config_targets names a device to back up and the vault credential
--     it logs in with; device_config_backups records every attempt, successful or not.
--     A failed attempt is a row with a status and an error and no config, so
--     "this switch stopped accepting our credential last Tuesday" is history
--     rather than a log line. The target's pinned host key is what turns a
--     changed key into a refusal instead of a silent re-trust.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose NO TRANSACTION

-- +goose Up
PRAGMA foreign_keys=OFF;
BEGIN;

CREATE TABLE device_credentials_new (
				id                 TEXT NOT NULL,
				client_id          TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				name               TEXT NOT NULL,
				kind               TEXT NOT NULL CHECK (kind IN ('v2c','v3','ssh')),
				security_level     TEXT CHECK (security_level IS NULL OR security_level IN ('noAuthNoPriv','authNoPriv','authPriv')),
				snmp_community_enc BLOB,
				snmp_v3_user       TEXT,
				snmp_v3_auth_enc   BLOB,
				snmp_v3_priv_enc   BLOB,
				snmp_v3_auth_proto TEXT CHECK (snmp_v3_auth_proto IS NULL OR snmp_v3_auth_proto IN ('SHA','SHA224','SHA256','SHA384','SHA512')),
				snmp_v3_priv_proto TEXT CHECK (snmp_v3_priv_proto IS NULL OR snmp_v3_priv_proto IN ('DES','AES','AES192','AES256')),
				ssh_user           TEXT,
				ssh_password_enc   BLOB,
				created_at         TEXT NOT NULL,
				updated_at         TEXT NOT NULL,
				PRIMARY KEY (id),
				UNIQUE (client_id, id),

				CHECK (snmp_community_enc IS NULL OR CAST(snmp_community_enc AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_auth_enc   IS NULL OR CAST(snmp_v3_auth_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_priv_enc   IS NULL OR CAST(snmp_v3_priv_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (ssh_password_enc   IS NULL OR CAST(ssh_password_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),

				-- SNMP kinds carry nothing of SSH's.
				CHECK (kind = 'ssh' OR (ssh_user IS NULL AND ssh_password_enc IS NULL)),

				CHECK (kind <> 'v2c' OR (
					snmp_community_enc IS NOT NULL
					AND security_level IS NULL
					AND snmp_v3_user     IS NULL
					AND snmp_v3_auth_enc IS NULL
					AND snmp_v3_priv_enc IS NULL
					AND snmp_v3_auth_proto IS NULL
					AND snmp_v3_priv_proto IS NULL
				)),

				CHECK (kind <> 'v3' OR (
					snmp_community_enc IS NULL
					AND snmp_v3_user IS NOT NULL AND snmp_v3_user <> ''
					AND security_level IS NOT NULL
				)),

				-- SSH is a user and a password, and nothing of SNMP's.
				CHECK (kind <> 'ssh' OR (
					ssh_user IS NOT NULL AND ssh_user <> ''
					AND ssh_password_enc IS NOT NULL
					AND security_level IS NULL
					AND snmp_community_enc IS NULL
					AND snmp_v3_user     IS NULL
					AND snmp_v3_auth_enc IS NULL
					AND snmp_v3_priv_enc IS NULL
					AND snmp_v3_auth_proto IS NULL
					AND snmp_v3_priv_proto IS NULL
				)),

				CHECK (security_level <> 'noAuthNoPriv' OR (
					snmp_v3_auth_enc IS NULL AND snmp_v3_priv_enc IS NULL
					AND snmp_v3_auth_proto IS NULL AND snmp_v3_priv_proto IS NULL
				)),
				CHECK (security_level <> 'authNoPriv' OR (
					snmp_v3_auth_enc IS NOT NULL AND snmp_v3_auth_proto IS NOT NULL
					AND snmp_v3_priv_enc IS NULL AND snmp_v3_priv_proto IS NULL
				)),
				CHECK (security_level <> 'authPriv' OR (
					snmp_v3_auth_enc IS NOT NULL AND snmp_v3_auth_proto IS NOT NULL
					AND snmp_v3_priv_enc IS NOT NULL AND snmp_v3_priv_proto IS NOT NULL
				))
			) STRICT;

INSERT INTO device_credentials_new (
				id, client_id, name, kind, security_level,
				snmp_community_enc, snmp_v3_user,
				snmp_v3_auth_enc, snmp_v3_priv_enc,
				snmp_v3_auth_proto, snmp_v3_priv_proto,
				created_at, updated_at
			)
			SELECT
				id, client_id, name, kind, security_level,
				snmp_community_enc, snmp_v3_user,
				snmp_v3_auth_enc, snmp_v3_priv_enc,
				snmp_v3_auth_proto, snmp_v3_priv_proto,
				created_at, updated_at
			FROM device_credentials;

DROP TABLE device_credentials;
ALTER TABLE device_credentials_new RENAME TO device_credentials;
CREATE INDEX idx_device_credentials_client ON device_credentials(client_id);
CREATE INDEX idx_device_credentials_name   ON device_credentials(name);

-- The credential reference is the (client, id) pair with RESTRICT, as on
-- polling_targets: a target cannot borrow another client's credential, and a
-- credential in use cannot be deleted out from under it. host_key_sha256 is
-- the SHA256 fingerprint pinned on first contact (OpenSSH's "SHA256:…" form);
-- NULL means nothing is pinned yet.
CREATE TABLE device_config_targets (
				id              TEXT NOT NULL PRIMARY KEY,
				client_id       TEXT NOT NULL REFERENCES clients(id),
				name            TEXT NOT NULL CHECK (name <> ''),
				host            TEXT NOT NULL CHECK (host <> ''),
				port            INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
				platform        TEXT NOT NULL CHECK (platform IN ('cisco-ios','cisco-nxos','arista-eos','juniper-junos')),
				credentials_id  TEXT NOT NULL,
				host_key_sha256 TEXT CHECK (host_key_sha256 IS NULL OR host_key_sha256 GLOB 'SHA256:*'),
				created_at      TEXT NOT NULL,
				updated_at      TEXT NOT NULL,
				UNIQUE (client_id, id),
				UNIQUE (client_id, host, port),
				FOREIGN KEY (client_id, credentials_id)
					REFERENCES device_credentials(client_id, id) ON DELETE RESTRICT
			) STRICT;
CREATE INDEX idx_device_config_targets_credentials ON device_config_targets(client_id, credentials_id);

-- A successful attempt has the configuration and its digest and no error; any
-- other status has an error and no configuration.
CREATE TABLE device_config_backups (
				id         TEXT NOT NULL PRIMARY KEY,
				client_id  TEXT NOT NULL,
				target_id  TEXT NOT NULL,
				taken_at   TEXT NOT NULL,
				status     TEXT NOT NULL CHECK (status IN ('ok','auth_failed','host_key_mismatch','unreachable','command_failed','error')),
				error      TEXT,
				config     TEXT,
				sha256     TEXT,
				CHECK (status <> 'ok' OR (config IS NOT NULL AND sha256 IS NOT NULL AND error IS NULL)),
				CHECK (status = 'ok' OR (config IS NULL AND sha256 IS NULL AND error IS NOT NULL)),
				FOREIGN KEY (client_id, target_id)
					REFERENCES device_config_targets(client_id, id) ON DELETE CASCADE
			) STRICT;
CREATE INDEX idx_device_device_config_backups_target ON device_config_backups(client_id, target_id, taken_at);

COMMIT;
PRAGMA foreign_keys=ON;

-- +goose Down
PRAGMA foreign_keys=OFF;
BEGIN;

DROP TABLE device_config_backups;
DROP TABLE device_config_targets;

-- The previous schema has no SSH kind, so SSH credentials cannot survive the
-- rollback.
DELETE FROM device_credentials WHERE kind = 'ssh';

CREATE TABLE device_credentials_old (
				id                 TEXT NOT NULL,
				client_id          TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				name               TEXT NOT NULL,
				kind               TEXT NOT NULL CHECK (kind IN ('v2c','v3')),
				security_level     TEXT CHECK (security_level IS NULL OR security_level IN ('noAuthNoPriv','authNoPriv','authPriv')),
				snmp_community_enc BLOB,
				snmp_v3_user       TEXT,
				snmp_v3_auth_enc   BLOB,
				snmp_v3_priv_enc   BLOB,
				snmp_v3_auth_proto TEXT CHECK (snmp_v3_auth_proto IS NULL OR snmp_v3_auth_proto IN ('SHA','SHA224','SHA256','SHA384','SHA512')),
				snmp_v3_priv_proto TEXT CHECK (snmp_v3_priv_proto IS NULL OR snmp_v3_priv_proto IN ('DES','AES','AES192','AES256')),
				created_at         TEXT NOT NULL,
				updated_at         TEXT NOT NULL,
				PRIMARY KEY (id),
				UNIQUE (client_id, id),
				CHECK (snmp_community_enc IS NULL OR CAST(snmp_community_enc AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_auth_enc   IS NULL OR CAST(snmp_v3_auth_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_priv_enc   IS NULL OR CAST(snmp_v3_priv_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (kind <> 'v2c' OR (
					snmp_community_enc IS NOT NULL
					AND security_level IS NULL
					AND snmp_v3_user     IS NULL
					AND snmp_v3_auth_enc IS NULL
					AND snmp_v3_priv_enc IS NULL
					AND snmp_v3_auth_proto IS NULL
					AND snmp_v3_priv_proto IS NULL
				)),
				CHECK (kind <> 'v3' OR (
					snmp_community_enc IS NULL
					AND snmp_v3_user IS NOT NULL AND snmp_v3_user <> ''
					AND security_level IS NOT NULL
				)),
				CHECK (security_level <> 'noAuthNoPriv' OR (
					snmp_v3_auth_enc IS NULL AND snmp_v3_priv_enc IS NULL
					AND snmp_v3_auth_proto IS NULL AND snmp_v3_priv_proto IS NULL
				)),
				CHECK (security_level <> 'authNoPriv' OR (
					snmp_v3_auth_enc IS NOT NULL AND snmp_v3_auth_proto IS NOT NULL
					AND snmp_v3_priv_enc IS NULL AND snmp_v3_priv_proto IS NULL
				)),
				CHECK (security_level <> 'authPriv' OR (
					snmp_v3_auth_enc IS NOT NULL AND snmp_v3_auth_proto IS NOT NULL
					AND snmp_v3_priv_enc IS NOT NULL AND snmp_v3_priv_proto IS NOT NULL
				))
			) STRICT;
INSERT INTO device_credentials_old (
				id, client_id, name, kind, security_level,
				snmp_community_enc, snmp_v3_user,
				snmp_v3_auth_enc, snmp_v3_priv_enc,
				snmp_v3_auth_proto, snmp_v3_priv_proto,
				created_at, updated_at
			)
			SELECT
				id, client_id, name, kind, security_level,
				snmp_community_enc, snmp_v3_user,
				snmp_v3_auth_enc, snmp_v3_priv_enc,
				snmp_v3_auth_proto, snmp_v3_priv_proto,
				created_at, updated_at
			FROM device_credentials;
DROP TABLE device_credentials;
ALTER TABLE device_credentials_old RENAME TO device_credentials;
CREATE INDEX idx_device_credentials_client ON device_credentials(client_id);
CREATE INDEX idx_device_credentials_name   ON device_credentials(name);

COMMIT;
PRAGMA foreign_keys=ON;
