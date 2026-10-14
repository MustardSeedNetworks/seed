-- 00028_drop_link_utilization.sql — drop topology_links.utilization_pct
-- (seed#3024).
--
-- Nothing ever wrote the column: the edge reconcilers upserted every link at
-- 0 %, whatever its load. Link utilization is now read from the interface
-- rates the SNMP pipeline stores in metrics (P-A4), at the time of the read,
-- so a stored copy would only go stale.
--
-- Regenerate the gate golden after edits:
--   UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot

-- +goose Up
ALTER TABLE topology_links DROP COLUMN utilization_pct;

-- +goose Down
ALTER TABLE topology_links ADD COLUMN utilization_pct REAL;
