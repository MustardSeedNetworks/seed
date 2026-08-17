-- 00009_drop_survey_samples.sql — delete the Wi-Fi survey sample table
-- (seed#1789/#1790). Wi-Fi site survey and planning moved out of Seed to
-- Trellis; Seed keeps Wi-Fi troubleshooting only. survey_samples backed the
-- removed internal/wifi/survey package and its handlers_survey* API surface,
-- both deleted in the same change. The index drops with the table in SQLite.
-- Regenerate the schema golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
DROP TABLE IF EXISTS survey_samples;

-- +goose Down
CREATE TABLE survey_samples (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				survey_id TEXT NOT NULL,
				x REAL NOT NULL,
				y REAL NOT NULL,
				signal_dbm INTEGER,
				noise_dbm INTEGER,
				snr_db INTEGER,
				channel INTEGER,
				frequency_mhz INTEGER,
				bssid TEXT,
				ssid TEXT,
				timestamp TEXT NOT NULL,
				networks_json TEXT,
				metadata_json TEXT
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_survey_samples_client ON survey_samples(client_id);
CREATE INDEX idx_survey_samples_coords ON survey_samples(x, y);
CREATE INDEX idx_survey_samples_survey ON survey_samples(survey_id);
