package database_test

// 00027 rebuilds device_credentials to admit the SSH kind and adds the
// configuration backup tables. Existing SNMP credentials, and the polling
// targets bound to them, survive the rebuild in both directions.

import (
	"context"
	"testing"
)

const credentialRows = `SELECT group_concat(row, ';') FROM (
	SELECT id || ',' || client_id || ',' || kind || ',' || coalesce(security_level, '-') || ',' ||
	       coalesce(CAST(snmp_community_enc AS TEXT), '-') || ',' || coalesce(snmp_v3_user, '-') AS row
	FROM device_credentials ORDER BY id)`

func TestMigration00027AddsSSHCredentialsAndKeepsSNMPOnes(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 26)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO device_credentials (id, client_id, name, kind, security_level,
		  snmp_community_enc, snmp_v3_user, created_at, updated_at)
		VALUES ('cred-a', 'default', 'public', 'v2c', NULL, CAST('enc:v1:x' AS BLOB), NULL, 't', 't'),
		       ('cred-b', 'default', 'ops', 'v3', 'noAuthNoPriv', NULL, 'ops', 't', 't');
		INSERT INTO polling_targets (id, name, ip_address, credentials_id, created_at, updated_at)
		VALUES ('pt-1', 'core', '10.0.0.1', 'cred-a', 't', 't')`); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}
	want := "cred-a,default,v2c,-,enc:v1:x,-;cred-b,default,v3,noAuthNoPriv,-,ops"

	upTo(t, db, 27)

	if got := scanString(t, db, credentialRows); got != want {
		t.Errorf("credentials after up = %q, want %q", got, want)
	}
	if got := scanString(t, db, `SELECT credentials_id FROM polling_targets WHERE id = 'pt-1'`); got != "cred-a" {
		t.Errorf("polling target binding after up = %q, want cred-a", got)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO device_credentials (id, client_id, name, kind, ssh_user, ssh_password_enc, created_at, updated_at)
		VALUES ('cred-s', 'default', 'backup', 'ssh', 'backup', CAST('enc:v1:pw' AS BLOB), 't', 't')`); err != nil {
		t.Fatalf("insert ssh credential: %v", err)
	}
	if got := scanString(t, db, `SELECT count(*) FROM pragma_foreign_key_check`); got != "0" {
		t.Errorf("foreign key violations after up: %s", got)
	}

	downTo(t, db, 26)

	if got := scanString(t, db, credentialRows); got != want {
		t.Errorf("credentials after down = %q, want %q (the ssh row cannot survive)", got, want)
	}
	for _, table := range []string{"device_config_targets", "device_config_backups"} {
		if tableExists(t, db, table) {
			t.Errorf("down left %s behind", table)
		}
	}
	if got := scanString(t, db, `SELECT count(*) FROM pragma_foreign_key_check`); got != "0" {
		t.Errorf("foreign key violations after down: %s", got)
	}
}
