-- 00017_drop_dns_gateway_results.sql — delete the DNS and gateway result
-- tables (seed#2623). Nothing ever wrote either one: the telemetry engine
-- records gateway and DNS health as series in metrics, and reporting reads
-- gateway latency, loss and uptime from there. The indexes drop with the
-- tables in SQLite.
-- Regenerate the schema golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
DROP TABLE IF EXISTS dns_results;
DROP TABLE IF EXISTS gateway_results;

-- +goose Down
CREATE TABLE dns_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				server TEXT NOT NULL,
				hostname TEXT NOT NULL,
				response_time_ms REAL,
				resolved_ip TEXT,
				status TEXT NOT NULL,
				error_message TEXT,
				timestamp TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_dns_interface ON dns_results(interface_name);
CREATE INDEX idx_dns_results_client ON dns_results(client_id);
CREATE INDEX idx_dns_server ON dns_results(server);
CREATE INDEX idx_dns_timestamp ON dns_results(timestamp);
CREATE TABLE gateway_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				interface_name TEXT NOT NULL,
				gateway TEXT NOT NULL,
				latency_ms REAL,
				packet_loss REAL,
				reachable INTEGER CHECK (reachable IN (0,1)),
				timestamp TEXT NOT NULL
			, client_id TEXT NOT NULL DEFAULT 'default' REFERENCES clients(id)) STRICT;
CREATE INDEX idx_gateway_interface ON gateway_results(interface_name);
CREATE INDEX idx_gateway_results_client ON gateway_results(client_id);
CREATE INDEX idx_gateway_timestamp ON gateway_results(timestamp);
