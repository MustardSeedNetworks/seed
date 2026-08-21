-- 00010_users_client_id.sql — give the user row an owning client, so a session
-- can carry tenancy as a claim instead of taking the caller's word for it
-- (#1797 U1.3 precondition). Every tenant-scoped table already carries
-- `client_id ... REFERENCES clients(id)`; `users` did not, so the only place a
-- request's client could come from was the request itself (see the query-param
-- reads in internal/api). This column is the source of truth the access token
-- is minted from.
--
-- Why a table rebuild rather than ALTER TABLE ADD COLUMN: SQLite rejects
-- "Cannot add a REFERENCES column with non-NULL default value", so the FK and
-- the NOT NULL DEFAULT cannot both be added in place.
--
-- Why NO TRANSACTION + foreign_keys=OFF: `users` is the parent of two
-- ON DELETE CASCADE children (api_tokens.owner_username, and the user_id FK at
-- 00001_init.sql:733). With foreign_keys=ON, the DROP in the rebuild fires
-- those cascades and silently empties both tables — verified on a scratch DB,
-- api_tokens went to zero rows. With foreign_keys=OFF the rebuild is inert and
-- `PRAGMA foreign_key_check` is clean afterwards. PRAGMA foreign_keys is a
-- no-op inside a transaction, which is why the transaction is explicit here.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose NO TRANSACTION

-- +goose Up
PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE users_new (
				id              INTEGER PRIMARY KEY AUTOINCREMENT,
				username        TEXT    NOT NULL UNIQUE CHECK (LENGTH(username) >= 3 AND LENGTH(username) <= 64),
				password_hash   TEXT    NOT NULL,
				role            TEXT    NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin','operator','viewer')),
				is_active       INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0,1)),
				last_login      TEXT,
				failed_attempts INTEGER NOT NULL DEFAULT 0,
				locked_until    TEXT,
				token_version   INTEGER NOT NULL DEFAULT 1,
				totp_secret     TEXT,
				totp_enabled    INTEGER NOT NULL DEFAULT 0 CHECK (totp_enabled IN (0,1)),
				auth_provider   TEXT    NOT NULL DEFAULT 'local' CHECK (auth_provider IN ('local','google','microsoft','github')),
				external_id     TEXT,
				email           TEXT,
				display_name    TEXT,
				client_id       TEXT    NOT NULL DEFAULT 'default' REFERENCES clients(id),
				created_at      TEXT    NOT NULL,
				updated_at      TEXT    NOT NULL,
				UNIQUE (auth_provider, external_id)
			) STRICT;
INSERT INTO users_new (
				id, username, password_hash, role, is_active, last_login,
				failed_attempts, locked_until, token_version, totp_secret,
				totp_enabled, auth_provider, external_id, email, display_name,
				created_at, updated_at
			)
			SELECT
				id, username, password_hash, role, is_active, last_login,
				failed_attempts, locked_until, token_version, totp_secret,
				totp_enabled, auth_provider, external_id, email, display_name,
				created_at, updated_at
			FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
CREATE INDEX idx_users_active               ON users(is_active);
CREATE INDEX idx_users_email                ON users(email);
CREATE INDEX idx_users_provider_external_id ON users(auth_provider, external_id);
CREATE INDEX idx_users_client               ON users(client_id);
CREATE INDEX idx_users_username             ON users(username);
COMMIT;
PRAGMA foreign_keys=ON;

-- +goose Down
PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE users_old (
				id              INTEGER PRIMARY KEY AUTOINCREMENT,
				username        TEXT    NOT NULL UNIQUE CHECK (LENGTH(username) >= 3 AND LENGTH(username) <= 64),
				password_hash   TEXT    NOT NULL,
				role            TEXT    NOT NULL DEFAULT 'viewer' CHECK (role IN ('admin','operator','viewer')),
				is_active       INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0,1)),
				last_login      TEXT,
				failed_attempts INTEGER NOT NULL DEFAULT 0,
				locked_until    TEXT,
				token_version   INTEGER NOT NULL DEFAULT 1,
				totp_secret     TEXT,
				totp_enabled    INTEGER NOT NULL DEFAULT 0 CHECK (totp_enabled IN (0,1)),
				auth_provider   TEXT    NOT NULL DEFAULT 'local' CHECK (auth_provider IN ('local','google','microsoft','github')),
				external_id     TEXT,
				email           TEXT,
				display_name    TEXT,
				created_at      TEXT    NOT NULL,
				updated_at      TEXT    NOT NULL,
				UNIQUE (auth_provider, external_id)
			) STRICT;
INSERT INTO users_old SELECT
				id, username, password_hash, role, is_active, last_login,
				failed_attempts, locked_until, token_version, totp_secret,
				totp_enabled, auth_provider, external_id, email, display_name,
				created_at, updated_at
			FROM users;
DROP TABLE users;
ALTER TABLE users_old RENAME TO users;
CREATE INDEX idx_users_active               ON users(is_active);
CREATE INDEX idx_users_email                ON users(email);
CREATE INDEX idx_users_provider_external_id ON users(auth_provider, external_id);
CREATE INDEX idx_users_username             ON users(username);
COMMIT;
PRAGMA foreign_keys=ON;
