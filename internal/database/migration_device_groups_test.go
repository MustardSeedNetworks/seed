package database_test

// 00031 adds device groups and alert_rules.group_id. Rolling back keeps every
// unscoped rule and drops the scoped ones rather than widening them.

import (
	"context"
	"testing"
)

func TestMigration00031DeviceGroupsRollBackWithoutWideningRules(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 30)
	ctx := context.Background()
	upTo(t, db, 31)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO polling_targets (id, client_id, name, ip_address, created_at, updated_at)
		VALUES ('tgt-1', 'default', 'core', '10.0.0.1', 't', 't');
		INSERT INTO device_groups (id, client_id, name, created_at, updated_at)
		VALUES ('grp-1', 'default', 'core', 't', 't');
		INSERT INTO device_group_members (group_id, target_id) VALUES ('grp-1', 'tgt-1');
		INSERT INTO alert_rules (id, name, alert_type, alert_severity, alert_title, alert_message,
		  created_at, updated_at, group_id)
		VALUES (1, 'everyone', 'syslog', 'warning', 't', 'm', 't', 't', NULL),
		       (2, 'core only', 'syslog', 'warning', 't', 'm', 't', 't', 'grp-1')`); err != nil {
		t.Fatalf("seed rows: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM device_groups WHERE id = 'grp-1'`); err == nil {
		t.Fatal("deleted a group an alert rule names")
	}

	downTo(t, db, 30)

	if got := scanString(t, db, `SELECT group_concat(name, ';') FROM alert_rules`); got != "everyone" {
		t.Errorf("rules after down = %q, want only the unscoped rule", got)
	}
	if got := scanString(t, db, `SELECT count(*) FROM pragma_foreign_key_check`); got != "0" {
		t.Errorf("foreign key violations after down: %s", got)
	}

	upTo(t, db, 31)
	if got := scanString(t, db, `SELECT count(*) FROM alert_rules WHERE group_id IS NULL`); got != "1" {
		t.Errorf("unscoped rules after re-up = %s, want 1", got)
	}
}
