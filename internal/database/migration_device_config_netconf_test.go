package database_test

// 00030 rebuilds device_config_targets to admit the netconf platform. Existing
// targets and their backups survive the rebuild in both directions; netconf
// targets cannot survive the rollback, and neither can their backups.

import (
	"context"
	"testing"
)

const configTargetRows = `SELECT coalesce(group_concat(id || ',' || platform || ',' || port, ';'), '')
	FROM (SELECT id, platform, port FROM device_config_targets ORDER BY id)`

const configBackupRows = `SELECT coalesce(group_concat(id || ',' || target_id, ';'), '')
	FROM (SELECT id, target_id FROM device_config_backups ORDER BY id)`

func TestMigration00030AdmitsNETCONFTargetsAndKeepsTheRest(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 29)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO device_credentials (id, client_id, name, kind, ssh_user, ssh_password_enc, created_at, updated_at)
		VALUES ('cred-s', 'default', 'backup', 'ssh', 'backup', CAST('enc:v1:pw' AS BLOB), 't', 't');
		INSERT INTO device_config_targets (id, client_id, name, host, port, platform, credentials_id,
		  host_key_sha256, created_at, updated_at)
		VALUES ('dct-1', 'default', 'core', '10.0.0.1', 22, 'cisco-ios', 'cred-s', 'SHA256:k', 't', 't');
		INSERT INTO device_config_backups (id, client_id, target_id, taken_at, status, config_enc, sha256)
		VALUES ('dcb-1', 'default', 'dct-1', 't', 'ok', CAST('enc:v1:cfg' AS BLOB), 'abc')`); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	const netconfTarget = `
		INSERT INTO device_config_targets (id, client_id, name, host, port, platform, credentials_id,
		  created_at, updated_at)
		VALUES ('dct-2', 'default', 'edge', '10.0.0.2', 830, 'netconf', 'cred-s', 't', 't')`
	if _, err := db.ExecContext(ctx, netconfTarget); err == nil {
		t.Fatal("00029 accepted a netconf target")
	}

	upTo(t, db, 30)

	if got := scanString(t, db, configTargetRows); got != "dct-1,cisco-ios,22" {
		t.Errorf("targets after up = %q", got)
	}
	if got := scanString(t, db, configBackupRows); got != "dcb-1,dct-1" {
		t.Errorf("backups after up = %q, the rebuild must not cascade", got)
	}
	if _, err := db.ExecContext(ctx, netconfTarget+`;
		INSERT INTO device_config_backups (id, client_id, target_id, taken_at, status, error)
		VALUES ('dcb-2', 'default', 'dct-2', 't', 'unreachable', 'timeout')`); err != nil {
		t.Fatalf("insert netconf target: %v", err)
	}
	if got := scanString(t, db, `SELECT count(*) FROM pragma_foreign_key_check`); got != "0" {
		t.Errorf("foreign key violations after up: %s", got)
	}

	downTo(t, db, 29)

	if got := scanString(t, db, configTargetRows); got != "dct-1,cisco-ios,22" {
		t.Errorf("targets after down = %q, want only the cisco-ios one", got)
	}
	if got := scanString(t, db, configBackupRows); got != "dcb-1,dct-1" {
		t.Errorf("backups after down = %q, want the netconf target's gone", got)
	}
	if got := scanString(t, db, `SELECT count(*) FROM pragma_foreign_key_check`); got != "0" {
		t.Errorf("foreign key violations after down: %s", got)
	}
}
