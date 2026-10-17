-- 00039_device_images.sql — operating system images for staging (P-D3,
-- #3499, ADR-0035 §3).
--
-- One row per uploaded image whose bytes matched the hash its vendor
-- publishes. The file itself lives in the image directory beside the
-- database, named by id; the row is the record of what it is. Both digests
-- are kept because staging asks the device to verify its copy with the one
-- the platform computes (SHA-512 on IOS and EOS, SHA-256 on Junos). A name is
-- the file name the device gets, so it is unique per client.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE device_images (
				id          TEXT NOT NULL PRIMARY KEY,
				client_id   TEXT NOT NULL REFERENCES clients(id),
				name        TEXT NOT NULL CHECK (name <> ''),
				platform    TEXT NOT NULL CHECK (platform IN ('cisco-ios','arista-eos','juniper-junos')),
				size        INTEGER NOT NULL CHECK (size > 0),
				sha256      TEXT NOT NULL CHECK (length(sha256) = 64),
				sha512      TEXT NOT NULL CHECK (length(sha512) = 128),
				uploaded_by TEXT NOT NULL CHECK (uploaded_by <> ''),
				uploaded_at TEXT NOT NULL,
				UNIQUE (client_id, name)
			) STRICT;

-- +goose Down
DROP TABLE device_images;
