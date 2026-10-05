package database_test

// 00024 drops the tables nothing writes and rebuilds microburst_events without
// its never-set device_id. Stored bursts survive the rebuild in both
// directions, and Down restores every dropped table.

import (
	"context"
	"database/sql"
	"testing"
)

const burstRows = `SELECT group_concat(row, ';') FROM (
	SELECT id || ',' || timestamp || ',' || interface_name || ',' || direction || ',' ||
	       peak_utilization_pct || ',' || duration_ms || ',' || sampling_mode || ',' ||
	       link_speed_mbps || ',' || client_id AS row
	FROM microburst_events ORDER BY id)`

func TestMigration00024DropsWriterlessTablesAndKeepsBursts(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 23)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO microburst_events (timestamp, interface_name, direction,
		  peak_utilization_pct, duration_ms, sampling_mode, link_speed_mbps)
		VALUES ('2026-10-05T00:00:00.001Z', 'eth0', 'rx', 97.5, 3, 'capture_1ms', 1000),
		       ('2026-10-05T00:00:01.000Z', 'eth0', 'tx', 91.0, 2, 'capture_1ms', 1000)`); err != nil {
		t.Fatalf("seed bursts: %v", err)
	}
	want := "1,2026-10-05T00:00:00.001Z,eth0,rx,97.5,3,capture_1ms,1000,default;" +
		"2,2026-10-05T00:00:01.000Z,eth0,tx,91.0,2,capture_1ms,1000,default"

	upTo(t, db, 24)

	if got := scanString(t, db, burstRows); got != want {
		t.Errorf("bursts after up = %q, want %q", got, want)
	}
	if got := scanString(t, db,
		`SELECT count(*) FROM pragma_table_info('microburst_events') WHERE name = 'device_id'`); got != "0" {
		t.Error("microburst_events still has device_id")
	}
	dropped := []string{"discovered_devices", "voip_calls", "wifi_access_points", "network_problems"}
	for _, table := range dropped {
		if tableExists(t, db, table) {
			t.Errorf("%s survived the migration", table)
		}
	}

	downTo(t, db, 23)

	if got := scanString(t, db, burstRows); got != want {
		t.Errorf("bursts after down = %q, want %q", got, want)
	}
	for _, table := range dropped {
		if !tableExists(t, db, table) {
			t.Errorf("down did not restore %s", table)
		}
	}
	if got := scanString(t, db, `SELECT count(*) FROM pragma_foreign_key_check`); got != "0" {
		t.Errorf("foreign key violations after down: %s", got)
	}
}

func scanString(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	var s string
	if err := db.QueryRowContext(context.Background(), query).Scan(&s); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return s
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n); err != nil {
		t.Fatalf("look up %s: %v", table, err)
	}
	return n == 1
}
