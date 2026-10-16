-- 00033_bluetooth_devices.sql — Bluetooth devices across scans (P-E3).
--
-- Every scan upserts each device it heard, keyed by address, so a device
-- keeps the id and the time it was first heard across scans and restarts.
-- Classification and signal are the latest scan's; a name or vendor a later
-- scan did not hear is kept rather than blanked.
--
-- 00024 dropped the 00001 bluetooth_devices table, which nothing wrote. This
-- one is written by the scan in the same change.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
CREATE TABLE bluetooth_devices (
				address         TEXT NOT NULL PRIMARY KEY,
				id              TEXT NOT NULL UNIQUE,
				address_type    TEXT NOT NULL DEFAULT '' CHECK (address_type IN ('', 'public', 'random')),
				name            TEXT NOT NULL DEFAULT '',
				vendor          TEXT NOT NULL DEFAULT '',
				type            TEXT NOT NULL CHECK (type IN ('classic', 'ble', 'dual')),
				device_class    TEXT NOT NULL DEFAULT '',
				appearance      INTEGER NOT NULL DEFAULT 0,
				class_of_device INTEGER NOT NULL DEFAULT 0,
				manufacturer_id INTEGER NOT NULL DEFAULT 0,
				rssi            INTEGER NOT NULL,
				tx_power        INTEGER NOT NULL DEFAULT 0,
				first_seen      TEXT NOT NULL,
				last_seen       TEXT NOT NULL
			) STRICT;
CREATE INDEX idx_bluetooth_devices_last_seen ON bluetooth_devices(last_seen);

-- +goose Down
DROP TABLE bluetooth_devices;
