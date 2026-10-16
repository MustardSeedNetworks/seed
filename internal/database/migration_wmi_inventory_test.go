package database_test

// 00033 rebuilds device_credentials to admit the winrm kind and adds the
// Windows inventory tables. Existing credentials and the targets that
// reference them survive the rebuild in both directions; WinRM credentials,
// their targets and inventories cannot survive the rollback.

import (
	"context"
	"testing"
)

const credentialKindRows = `SELECT coalesce(group_concat(id || ',' || kind, ';'), '')
	FROM (SELECT id, kind FROM device_credentials ORDER BY id)`

func TestMigration00033AdmitsWinRMCredentialsAndKeepsTheRest(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 32)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO device_credentials (id, client_id, name, kind, ssh_user, ssh_password_enc, created_at, updated_at)
		VALUES ('cred-s', 'default', 'backup', 'ssh', 'backup', CAST('enc:v1:pw' AS BLOB), 't', 't');
		INSERT INTO device_credentials (id, client_id, name, kind, snmp_community_enc, created_at, updated_at)
		VALUES ('cred-v', 'default', 'poll', 'v2c', CAST('enc:v1:c' AS BLOB), 't', 't');
		INSERT INTO device_config_targets (id, client_id, name, host, port, platform, credentials_id,
		  created_at, updated_at)
		VALUES ('dct-1', 'default', 'core', '10.0.0.1', 22, 'cisco-ios', 'cred-s', 't', 't')`); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}

	upTo(t, db, 33)

	if got := scanString(t, db, credentialKindRows); got != "cred-s,ssh;cred-v,v2c" {
		t.Errorf("credentials after up = %q", got)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO device_credentials (id, client_id, name, kind, winrm_user, winrm_password_enc, created_at, updated_at)
		VALUES ('cred-w', 'default', 'inventory', 'winrm', 'LAB\inv', CAST('enc:v1:pw' AS BLOB), 't', 't');
		INSERT INTO wmi_targets (id, client_id, name, host, port, credentials_id, created_at, updated_at)
		VALUES ('wmt-1', 'default', 'dc01', '10.0.0.5', 5986, 'cred-w', 't', 't');
		INSERT INTO wmi_inventories (client_id, target_id, collected_at, inventory)
		VALUES ('default', 'wmt-1', 't', '{}')`); err != nil {
		t.Fatalf("insert WinRM credential and target: %v", err)
	}
	if got := scanString(t, db, `SELECT count(*) FROM pragma_foreign_key_check`); got != "0" {
		t.Errorf("foreign key violations after up: %s", got)
	}

	downTo(t, db, 32)

	if got := scanString(t, db, credentialKindRows); got != "cred-s,ssh;cred-v,v2c" {
		t.Errorf("credentials after down = %q, want the WinRM one gone", got)
	}
	if got := scanString(t, db, `SELECT count(*) FROM device_config_targets`); got != "1" {
		t.Errorf("backup targets after down = %s, the rebuild must not cascade", got)
	}
	if got := scanString(t, db, `SELECT count(*) FROM pragma_foreign_key_check`); got != "0" {
		t.Errorf("foreign key violations after down: %s", got)
	}
}
