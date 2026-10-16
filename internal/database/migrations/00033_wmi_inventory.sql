-- 00033_wmi_inventory.sql — Windows inventory over WS-Management (P-D11,
-- #3506, ADR-0034).
--
--   * The credential vault learns a fourth kind, 'winrm': a user and a password
--     stored as versioned ciphertext, like SSH's. The kinds stay mutually
--     exclusive. SQLite cannot alter a CHECK in place, so the table is rebuilt;
--     NO TRANSACTION + foreign_keys=OFF for the reason 00011 gives
--     (polling_targets and device_config_targets reference the vault, and
--     dropping the old table with enforcement on would fire those references).
--
--   * wmi_targets names a Windows host, the vault credential it signs in with,
--     and the outcome of its last attempt. cert_sha256 is the listener
--     certificate's SHA-256 an operator confirmed, in the uppercase hex
--     PowerShell prints for $cert.GetCertHashString('SHA256'); NULL means
--     nothing is pinned. pending_cert_sha256 is the certificate an unpinned
--     host last presented, waiting for that confirmation.
--
--   * wmi_inventories holds each target's last successful inventory as JSON.
--     A successful collection replaces it; there is no history (ADR-0034 §5).
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
				kind               TEXT NOT NULL CHECK (kind IN ('v2c','v3','ssh','winrm')),
				security_level     TEXT CHECK (security_level IS NULL OR security_level IN ('noAuthNoPriv','authNoPriv','authPriv')),
				snmp_community_enc BLOB,
				snmp_v3_user       TEXT,
				snmp_v3_auth_enc   BLOB,
				snmp_v3_priv_enc   BLOB,
				snmp_v3_auth_proto TEXT CHECK (snmp_v3_auth_proto IS NULL OR snmp_v3_auth_proto IN ('SHA','SHA224','SHA256','SHA384','SHA512')),
				snmp_v3_priv_proto TEXT CHECK (snmp_v3_priv_proto IS NULL OR snmp_v3_priv_proto IN ('DES','AES','AES192','AES256')),
				ssh_user           TEXT,
				ssh_password_enc   BLOB,
				winrm_user         TEXT,
				winrm_password_enc BLOB,
				created_at         TEXT NOT NULL,
				updated_at         TEXT NOT NULL,
				PRIMARY KEY (id),
				UNIQUE (client_id, id),

				CHECK (snmp_community_enc IS NULL OR CAST(snmp_community_enc AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_auth_enc   IS NULL OR CAST(snmp_v3_auth_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_priv_enc   IS NULL OR CAST(snmp_v3_priv_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (ssh_password_enc   IS NULL OR CAST(ssh_password_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (winrm_password_enc IS NULL OR CAST(winrm_password_enc AS TEXT) GLOB 'enc:v[0-9]*:*'),

				-- Only an SSH credential carries SSH's columns, and only a WinRM
				-- credential WinRM's.
				CHECK (kind = 'ssh' OR (ssh_user IS NULL AND ssh_password_enc IS NULL)),
				CHECK (kind = 'winrm' OR (winrm_user IS NULL AND winrm_password_enc IS NULL)),

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

				-- WinRM is a user and a password, and nothing of SNMP's.
				CHECK (kind <> 'winrm' OR (
					winrm_user IS NOT NULL AND winrm_user <> ''
					AND winrm_password_enc IS NOT NULL
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
				ssh_user, ssh_password_enc,
				created_at, updated_at
			)
			SELECT
				id, client_id, name, kind, security_level,
				snmp_community_enc, snmp_v3_user,
				snmp_v3_auth_enc, snmp_v3_priv_enc,
				snmp_v3_auth_proto, snmp_v3_priv_proto,
				ssh_user, ssh_password_enc,
				created_at, updated_at
			FROM device_credentials;
DROP TABLE device_credentials;
ALTER TABLE device_credentials_new RENAME TO device_credentials;
CREATE INDEX idx_device_credentials_client ON device_credentials(client_id);
CREATE INDEX idx_device_credentials_name   ON device_credentials(name);

CREATE TABLE wmi_targets (
				id                  TEXT NOT NULL PRIMARY KEY,
				client_id           TEXT NOT NULL REFERENCES clients(id),
				name                TEXT NOT NULL CHECK (name <> ''),
				host                TEXT NOT NULL CHECK (host <> ''),
				port                INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
				credentials_id      TEXT NOT NULL,
				cert_sha256         TEXT CHECK (cert_sha256 IS NULL
					OR (length(cert_sha256) = 64 AND cert_sha256 NOT GLOB '*[^0-9A-F]*')),
				pending_cert_sha256 TEXT CHECK (pending_cert_sha256 IS NULL
					OR (length(pending_cert_sha256) = 64 AND pending_cert_sha256 NOT GLOB '*[^0-9A-F]*')),
				last_attempt_at     TEXT,
				last_status         TEXT CHECK (last_status IS NULL OR last_status IN
					('ok','auth_failed','cert_mismatch','cert_unconfirmed','unreachable','query_failed','error')),
				last_error          TEXT,
				created_at          TEXT NOT NULL,
				updated_at          TEXT NOT NULL,
				UNIQUE (client_id, id),
				UNIQUE (client_id, host, port),
				CHECK ((last_attempt_at IS NULL) = (last_status IS NULL)),
				CHECK (last_status IS NULL OR (last_status = 'ok') = (last_error IS NULL)),
				FOREIGN KEY (client_id, credentials_id)
					REFERENCES device_credentials(client_id, id) ON DELETE RESTRICT
			) STRICT;
CREATE INDEX idx_wmi_targets_credentials ON wmi_targets(client_id, credentials_id);

CREATE TABLE wmi_inventories (
				client_id    TEXT NOT NULL,
				target_id    TEXT NOT NULL,
				collected_at TEXT NOT NULL,
				inventory    TEXT NOT NULL CHECK (json_valid(inventory)),
				PRIMARY KEY (client_id, target_id),
				FOREIGN KEY (client_id, target_id)
					REFERENCES wmi_targets(client_id, id) ON DELETE CASCADE
			) STRICT;

COMMIT;
PRAGMA foreign_keys=ON;

-- +goose Down
PRAGMA foreign_keys=OFF;
BEGIN;

DROP TABLE wmi_inventories;
DROP TABLE wmi_targets;

-- The previous schema has no WinRM kind, so WinRM credentials cannot survive
-- the rollback.
DELETE FROM device_credentials WHERE kind = 'winrm';

CREATE TABLE device_credentials_old (
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
INSERT INTO device_credentials_old (
				id, client_id, name, kind, security_level,
				snmp_community_enc, snmp_v3_user,
				snmp_v3_auth_enc, snmp_v3_priv_enc,
				snmp_v3_auth_proto, snmp_v3_priv_proto,
				ssh_user, ssh_password_enc,
				created_at, updated_at
			)
			SELECT
				id, client_id, name, kind, security_level,
				snmp_community_enc, snmp_v3_user,
				snmp_v3_auth_enc, snmp_v3_priv_enc,
				snmp_v3_auth_proto, snmp_v3_priv_proto,
				ssh_user, ssh_password_enc,
				created_at, updated_at
			FROM device_credentials;
DROP TABLE device_credentials;
ALTER TABLE device_credentials_old RENAME TO device_credentials;
CREATE INDEX idx_device_credentials_client ON device_credentials(client_id);
CREATE INDEX idx_device_credentials_name   ON device_credentials(name);

COMMIT;
PRAGMA foreign_keys=ON;
