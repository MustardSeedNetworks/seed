-- 00040_webauthn_credential_record.sql — store each passkey as the complete
-- record foundation/pkg/passkey serializes (D-SEED-114, #3644).
--
-- The old columns kept the public key, sign count, attestation type and
-- AAGUID and dropped the authenticator flags and attestation data a later
-- assertion is checked against. Those rows cannot be rebuilt into complete
-- records, so they are dropped and their owners enrol again (pre-1.0: no
-- carried-over formats).
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
DROP TABLE webauthn_credentials;
CREATE TABLE webauthn_credentials (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				credential_id BLOB NOT NULL UNIQUE,
				record BLOB NOT NULL,
				created_at TEXT NOT NULL,
				last_used_at TEXT,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			) STRICT;
CREATE INDEX idx_webauthn_user ON webauthn_credentials(user_id);

-- +goose Down
DROP TABLE webauthn_credentials;
CREATE TABLE webauthn_credentials (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				credential_id BLOB NOT NULL UNIQUE,
				public_key BLOB NOT NULL,
				sign_count INTEGER NOT NULL DEFAULT 0,
				attestation_type TEXT,
				transports TEXT,
				aaguid BLOB,
				created_at TEXT NOT NULL,
				last_used_at TEXT,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			) STRICT;
CREATE UNIQUE INDEX idx_webauthn_credential_id
				ON webauthn_credentials(credential_id);
CREATE INDEX idx_webauthn_user ON webauthn_credentials(user_id);
