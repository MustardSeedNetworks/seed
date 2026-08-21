-- 00011_credentials_canonical.sql — make the SNMP credential vault a real
-- schema instead of a bag of nullable columns (#1792, runbook S1.1).
--
-- Before this, `device_credentials` could hold states that are not credentials
-- at all: plaintext in the ciphertext columns, a v3 user with a v2c community,
-- privacy with no authentication, MD5. Nothing rejected them, so the failure
-- surfaced as a poll that authenticated wrongly rather than as a write that was
-- refused. And `polling_targets.credentials_id` referenced the vault globally,
-- so the foreign key proved the row existed but not that it belonged to the
-- target's client — the invariant the repository has to enforce by hand today.
--
-- What the canonical form asserts:
--   * kind is exactly one of v2c or v3, and the columns of the other kind are
--     NULL. A row cannot be both.
--   * every secret is versioned ciphertext, "enc:v<N>:…". Plaintext and the
--     legacy unversioned "enc:…" are rejected, not migrated.
--   * v3 carries an explicit security_level, and it agrees with the secrets
--     present: authPriv has both, authNoPriv has auth only, noAuthNoPriv has
--     neither. Privacy without authentication is unrepresentable.
--   * auth is SHA-family only. MD5 is out: it is the one this build accepts
--     that no deployment should be using.
--   * (client_id, id) is unique, and polling_targets references *that* pair
--     with ON DELETE RESTRICT — so a credential in use cannot be deleted, and
--     a target cannot reference another client's vault entry at all.
--
-- The CHECK constraints are the migration's validator. Existing rows are copied
-- through them, so an ambiguous or invalid row aborts the migration with the
-- constraint that rejected it, rather than being rewritten or given a default.
-- That is deliberate: guessing what a half-populated credential meant is how a
-- device ends up polled with the wrong secret.
--
-- Why NO TRANSACTION + foreign_keys=OFF: `polling_targets.credentials_id`
-- references device_credentials ON DELETE SET NULL, so with enforcement on, the
-- DROP inside the vault rebuild fires that rule and silently nulls every
-- target's credential binding — verified on a scratch DB, 'c1' became NULL with
-- no error. PRAGMA foreign_keys is a no-op inside a transaction, which is why
-- the transaction is explicit. `PRAGMA foreign_key_check` runs clean afterwards
-- and a test migrates a populated database across the boundary to prove the
-- bindings survived.
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

				-- Every stored secret is versioned ciphertext. The legacy
				-- unversioned "enc:…" fails this too, on purpose: the key that
				-- produced it is unknown, so it cannot be rotated.
				CHECK (snmp_community_enc IS NULL OR CAST(snmp_community_enc AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_auth_enc   IS NULL OR CAST(snmp_v3_auth_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),
				CHECK (snmp_v3_priv_enc   IS NULL OR CAST(snmp_v3_priv_enc   AS TEXT) GLOB 'enc:v[0-9]*:*'),

				-- v2c is a community string and nothing else.
				CHECK (kind <> 'v2c' OR (
					snmp_community_enc IS NOT NULL
					AND security_level IS NULL
					AND snmp_v3_user     IS NULL
					AND snmp_v3_auth_enc IS NULL
					AND snmp_v3_priv_enc IS NULL
					AND snmp_v3_auth_proto IS NULL
					AND snmp_v3_priv_proto IS NULL
				)),

				-- v3 is a user plus a security level, and no community.
				CHECK (kind <> 'v3' OR (
					snmp_community_enc IS NULL
					AND snmp_v3_user IS NOT NULL AND snmp_v3_user <> ''
					AND security_level IS NOT NULL
				)),

				-- The security level and the secrets present cannot disagree,
				-- which is what makes "privacy without authentication"
				-- unrepresentable rather than merely discouraged.
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

-- Existing rows are copied through the constraints above. kind and
-- security_level are derived from what the row actually holds; a row that
-- holds a combination the canonical form has no name for fails the copy and
-- aborts the migration.
INSERT INTO device_credentials_new (
				id, client_id, name, kind, security_level,
				snmp_community_enc, snmp_v3_user,
				snmp_v3_auth_enc, snmp_v3_priv_enc,
				snmp_v3_auth_proto, snmp_v3_priv_proto,
				created_at, updated_at
			)
			SELECT
				id, client_id, name,
				CASE
					WHEN snmp_community_enc IS NOT NULL AND (snmp_v3_user IS NULL OR snmp_v3_user = '') THEN 'v2c'
					WHEN snmp_v3_user IS NOT NULL AND snmp_v3_user <> '' AND snmp_community_enc IS NULL THEN 'v3'
					-- Both or neither: not a credential this schema can name.
					-- NULL fails the NOT NULL on kind and aborts.
					ELSE NULL
				END,
				CASE
					WHEN snmp_v3_user IS NULL OR snmp_v3_user = '' THEN NULL
					WHEN snmp_v3_priv_enc IS NOT NULL THEN 'authPriv'
					WHEN snmp_v3_auth_enc IS NOT NULL THEN 'authNoPriv'
					ELSE 'noAuthNoPriv'
				END,
				snmp_community_enc,
				NULLIF(snmp_v3_user, ''),
				snmp_v3_auth_enc, snmp_v3_priv_enc,
				NULLIF(UPPER(TRIM(snmp_v3_auth_proto)), ''),
				NULLIF(UPPER(TRIM(snmp_v3_priv_proto)), ''),
				created_at, updated_at
			FROM device_credentials;

DROP TABLE device_credentials;
ALTER TABLE device_credentials_new RENAME TO device_credentials;
CREATE INDEX idx_device_credentials_client ON device_credentials(client_id);
CREATE INDEX idx_device_credentials_name   ON device_credentials(name);

-- polling_targets is rebuilt only to change its reference: the pair, not the
-- id, and RESTRICT rather than SET NULL. Silently unbinding every target when
-- a credential is deleted is the behaviour that made the old FK feel safe while
-- doing nothing useful.
CREATE TABLE polling_targets_new (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				ip_address TEXT NOT NULL,
				snmp_version TEXT NOT NULL DEFAULT 'v2c',
				credentials_id TEXT,
				poll_interval_seconds INTEGER NOT NULL DEFAULT 300,
				enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),
				last_polled_at TEXT,
				last_status TEXT,
				last_error TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id),
				collector_chain TEXT NOT NULL DEFAULT '["sys_info","if_table","lldp","arp","fdb"]',
				FOREIGN KEY (client_id, credentials_id)
					REFERENCES device_credentials(client_id, id) ON DELETE RESTRICT
			) STRICT;

INSERT INTO polling_targets_new (
				id, name, ip_address, snmp_version, credentials_id,
				poll_interval_seconds, enabled, last_polled_at, last_status,
				last_error, created_at, updated_at, client_id, collector_chain
			)
			SELECT
				id, name, ip_address, snmp_version, credentials_id,
				poll_interval_seconds, enabled, last_polled_at, last_status,
				last_error, created_at, updated_at, client_id, collector_chain
			FROM polling_targets;

DROP TABLE polling_targets;
ALTER TABLE polling_targets_new RENAME TO polling_targets;
CREATE INDEX idx_polling_targets_client  ON polling_targets(client_id);
CREATE INDEX idx_polling_targets_enabled ON polling_targets(enabled);
CREATE INDEX idx_polling_targets_ip      ON polling_targets(ip_address);

COMMIT;
PRAGMA foreign_keys=ON;

-- +goose Down
PRAGMA foreign_keys=OFF;
BEGIN;

CREATE TABLE device_credentials_old (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				snmp_community_enc BLOB,
				snmp_v3_user TEXT,
				snmp_v3_auth_enc BLOB,
				snmp_v3_priv_enc BLOB,
				snmp_v3_auth_proto TEXT,
				snmp_v3_priv_proto TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
INSERT INTO device_credentials_old (
				id, name, snmp_community_enc, snmp_v3_user, snmp_v3_auth_enc,
				snmp_v3_priv_enc, snmp_v3_auth_proto, snmp_v3_priv_proto,
				created_at, updated_at, client_id
			)
			SELECT
				id, name, snmp_community_enc, snmp_v3_user, snmp_v3_auth_enc,
				snmp_v3_priv_enc, snmp_v3_auth_proto, snmp_v3_priv_proto,
				created_at, updated_at, client_id
			FROM device_credentials;
DROP TABLE device_credentials;
ALTER TABLE device_credentials_old RENAME TO device_credentials;
CREATE INDEX idx_device_credentials_client ON device_credentials(client_id);
CREATE INDEX idx_device_credentials_name   ON device_credentials(name);

CREATE TABLE polling_targets_old (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				ip_address TEXT NOT NULL,
				snmp_version TEXT NOT NULL DEFAULT 'v2c',
				credentials_id TEXT,
				poll_interval_seconds INTEGER NOT NULL DEFAULT 300,
				enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),
				last_polled_at TEXT,
				last_status TEXT,
				last_error TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id), collector_chain TEXT NOT NULL DEFAULT '["sys_info","if_table","lldp","arp","fdb"]',
				FOREIGN KEY (credentials_id) REFERENCES device_credentials(id) ON DELETE SET NULL) STRICT;
INSERT INTO polling_targets_old (
				id, name, ip_address, snmp_version, credentials_id,
				poll_interval_seconds, enabled, last_polled_at, last_status,
				last_error, created_at, updated_at, client_id, collector_chain
			)
			SELECT
				id, name, ip_address, snmp_version, credentials_id,
				poll_interval_seconds, enabled, last_polled_at, last_status,
				last_error, created_at, updated_at, client_id, collector_chain
			FROM polling_targets;
DROP TABLE polling_targets;
ALTER TABLE polling_targets_old RENAME TO polling_targets;
CREATE INDEX idx_polling_targets_client  ON polling_targets(client_id);
CREATE INDEX idx_polling_targets_enabled ON polling_targets(enabled);
CREATE INDEX idx_polling_targets_ip      ON polling_targets(ip_address);

COMMIT;
PRAGMA foreign_keys=ON;
