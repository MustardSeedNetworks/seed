package database_test

// 00016 renames the trap listener's event kind. Stored events and operator
// rules still naming the old kind would stop matching anything the listener
// now emits, so both move; other kinds stay put.

import (
	"context"
	"testing"
)

func TestMigration00016RenamesTrapKind(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 15)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO listener_events (kind, source_addr, severity, observed_at, payload_json, ingested_at)
		VALUES ('snmp-trap-v2c', '10.0.0.1:162', 'warning', '2026-10-04T12:00:00Z', '{}', '2026-10-04T12:00:00Z'),
		       ('syslog-udp', '10.0.0.2:514', 'notice', '2026-10-04T12:00:00Z', '{}', '2026-10-04T12:00:00Z');
		INSERT INTO alert_rules (name, match_kind, alert_type, alert_severity, alert_title, alert_message, created_at, updated_at)
		VALUES ('traps', 'snmp-trap-v2c', 'system', 'warning', 't', 'm', '2026-10-04T12:00:00Z', '2026-10-04T12:00:00Z'),
		       ('syslog', 'syslog-udp', 'system', 'warning', 't', 'm', '2026-10-04T12:00:00Z', '2026-10-04T12:00:00Z');
	`); err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	upTo(t, db, 16)

	for _, q := range []struct{ query, want string }{
		{`SELECT group_concat(kind, ',') FROM (SELECT kind FROM listener_events ORDER BY id)`, "snmp-trap,syslog-udp"},
		{`SELECT group_concat(match_kind, ',') FROM (SELECT match_kind FROM alert_rules ORDER BY id)`, "snmp-trap,syslog-udp"},
	} {
		var got string
		if err := db.QueryRowContext(ctx, q.query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", q.query, err)
		}
		if got != q.want {
			t.Errorf("%s = %q, want %q", q.query, got, q.want)
		}
	}
}
