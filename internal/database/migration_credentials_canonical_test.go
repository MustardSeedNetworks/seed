package database_test

// migration_credentials_canonical_test.go covers 00011: which credential
// states the vault now accepts, which it refuses, that the rebuild does not
// destroy the bindings hanging off it, and that a populated database survives
// Up/Down/Up.
//
// The invalid cases are table-driven and each names the real-world mistake it
// stands for, because "constraint failed" is not a useful record of what was
// being prevented.

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/MustardSeedNetworks/seed/internal/polling"
)

const credMigrationVersion int64 = 11

// insertCanonicalCred inserts a credential row with explicit columns and
// returns the error, so tests can assert on rejection as well as acceptance.
func insertCanonicalCred(db *sql.DB, cols string, args ...any) error {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	_, err := db.ExecContext(context.Background(),
		"INSERT INTO device_credentials ("+cols+") VALUES ("+placeholders+")", args...)
	return err
}

func TestCredentialsVaultAcceptsEveryCanonicalState(t *testing.T) {
	t.Parallel()

	const cols = `id, client_id, name, kind, security_level, snmp_community_enc,
		snmp_v3_user, snmp_v3_auth_enc, snmp_v3_priv_enc,
		snmp_v3_auth_proto, snmp_v3_priv_proto, created_at, updated_at`
	const now = "2026-08-21T00:00:00Z"

	tests := []struct {
		name string
		args []any
	}{
		{
			"v2c community",
			[]any{
				"c-v2c", "default", "v2c cred", "v2c", nil, []byte("enc:v1:community"),
				nil, nil, nil, nil, nil, now, now,
			},
		},
		{
			"v3 noAuthNoPriv",
			[]any{
				"c-noauth", "default", "v3 noauth", "v3", "noAuthNoPriv", nil,
				"operator", nil, nil, nil, nil, now, now,
			},
		},
		{
			"v3 authNoPriv",
			[]any{
				"c-auth", "default", "v3 auth", "v3", "authNoPriv", nil,
				"operator", []byte("enc:v1:auth"), nil, "SHA256", nil, now, now,
			},
		},
		{
			"v3 authPriv",
			[]any{
				"c-authpriv", "default", "v3 authpriv", "v3", "authPriv", nil,
				"operator", []byte("enc:v1:auth"), []byte("enc:v1:priv"), "SHA512", "AES256", now, now,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A database per case: they run in parallel, and SQLite serialises
			// writers, so sharing one would trade a real assertion for
			// SQLITE_BUSY.
			t.Parallel()
			if err := insertCanonicalCred(migrateTo(t, credMigrationVersion), cols, tt.args...); err != nil {
				t.Errorf("canonical state rejected: %v", err)
			}
		})
	}
}

func TestCredentialsVaultRejectsInvalidStates(t *testing.T) {
	t.Parallel()

	const cols = `id, client_id, name, kind, security_level, snmp_community_enc,
		snmp_v3_user, snmp_v3_auth_enc, snmp_v3_priv_enc,
		snmp_v3_auth_proto, snmp_v3_priv_proto, created_at, updated_at`
	const now = "2026-08-21T00:00:00Z"

	tests := []struct {
		name string
		why  string
		args []any
	}{
		{
			"plaintext community", "a secret that was never encrypted",
			[]any{
				"x1", "default", "n", "v2c", nil, []byte("public"),
				nil, nil, nil, nil, nil, now, now,
			},
		},
		{
			"legacy unversioned ciphertext", "the key that produced it is unknown, so it cannot be rotated",
			[]any{
				"x2", "default", "n", "v2c", nil, []byte("enc:opaque"),
				nil, nil, nil, nil, nil, now, now,
			},
		},
		{
			"MD5 auth", "the one this build accepts that no deployment should use",
			[]any{
				"x3", "default", "n", "v3", "authNoPriv", nil,
				"operator", []byte("enc:v1:auth"), nil, "MD5", nil, now, now,
			},
		},
		{
			"privacy without auth", "an SNMPv3 state that cannot authenticate what it decrypts",
			[]any{
				"x4", "default", "n", "v3", "authPriv", nil,
				"operator", nil, []byte("enc:v1:priv"), nil, "AES", now, now,
			},
		},
		{
			"v2c carrying v3 fields", "a credential that is both kinds at once",
			[]any{
				"x5", "default", "n", "v2c", nil, []byte("enc:v1:community"),
				"operator", nil, nil, nil, nil, now, now,
			},
		},
		{
			"v3 carrying a community", "the same ambiguity from the other side",
			[]any{
				"x6", "default", "n", "v3", "authNoPriv", []byte("enc:v1:community"),
				"operator", []byte("enc:v1:auth"), nil, "SHA", nil, now, now,
			},
		},
		{
			"v3 with no security level", "a v3 credential whose level nobody chose",
			[]any{
				"x7", "default", "n", "v3", nil, nil,
				"operator", []byte("enc:v1:auth"), nil, "SHA", nil, now, now,
			},
		},
		{
			"authPriv missing the priv secret", "a level that disagrees with what is stored",
			[]any{
				"x8", "default", "n", "v3", "authPriv", nil,
				"operator", []byte("enc:v1:auth"), nil, "SHA", nil, now, now,
			},
		},
		{
			"unknown kind", "not a credential this schema can name",
			[]any{
				"x9", "default", "n", "v1", nil, []byte("enc:v1:community"),
				nil, nil, nil, nil, nil, now, now,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := insertCanonicalCred(migrateTo(t, credMigrationVersion), cols, tt.args...); err == nil {
				t.Errorf("accepted %s — %s", tt.name, tt.why)
			}
		})
	}
}

func TestCredentialsVaultIsSameClientAndRestrictive(t *testing.T) {
	t.Parallel()
	db := migrateTo(t, credMigrationVersion)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx,
		`INSERT INTO clients (id, name, slug, created_at, updated_at)
		 VALUES ('tenant-b','Tenant B','tenant-b','2026-08-21T00:00:00Z','2026-08-21T00:00:00Z')`,
	); err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO device_credentials (id, client_id, name, kind, snmp_community_enc, created_at, updated_at)
		 VALUES ('cred-b','tenant-b','theirs','v2c',CAST('enc:v1:community' AS BLOB),
		         '2026-08-21T00:00:00Z','2026-08-21T00:00:00Z')`,
	); err != nil {
		t.Fatalf("create credential: %v", err)
	}

	const insertTarget = `
		INSERT INTO polling_targets
			(id, client_id, name, ip_address, credentials_id, created_at, updated_at)
		VALUES (?, ?, 'target', '10.0.0.1', ?, '2026-08-21T00:00:00Z','2026-08-21T00:00:00Z')`

	// These are steps, not cases: the delete can only be attempted once the
	// binding exists, so they run in order rather than as parallel subtests.
	if _, err := db.ExecContext(ctx, insertTarget, "t-cross", "default", "cred-b"); err == nil {
		t.Error("accepted a target referencing another client's credential")
	}
	if _, err := db.ExecContext(ctx, insertTarget, "t-same", "tenant-b", "cred-b"); err != nil {
		t.Fatalf("rejected a same-client credential reference: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM device_credentials WHERE id = 'cred-b'`); err == nil {
		t.Fatal("deleted a credential a target still references")
	}

	var bound string
	if err := db.QueryRowContext(ctx,
		`SELECT credentials_id FROM polling_targets WHERE id = 't-same'`).Scan(&bound); err != nil {
		t.Fatalf("re-read target: %v", err)
	}
	if bound != "cred-b" {
		t.Errorf("target binding = %q, want cred-b — the refused delete still unbound it", bound)
	}
}

func TestMigration00011PreservesPopulatedRowsThroughUpDownUp(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, credMigrationVersion-1)
	ctx := context.Background()

	// A v2c and a v3 credential in the pre-canonical shape, plus a target bound
	// to the v2c one. The binding is what an unguarded rebuild destroys.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO device_credentials
			(id, client_id, name, snmp_community_enc, created_at, updated_at)
		VALUES ('legacy-v2c','default','legacy v2c',CAST('enc:v1:community' AS BLOB),
		        '2026-08-21T00:00:00Z','2026-08-21T00:00:00Z')`); err != nil {
		t.Fatalf("seed v2c: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO device_credentials
			(id, client_id, name, snmp_v3_user, snmp_v3_auth_enc, snmp_v3_priv_enc,
			 snmp_v3_auth_proto, snmp_v3_priv_proto, created_at, updated_at)
		VALUES ('legacy-v3','default','legacy v3','operator',
		        CAST('enc:v1:auth' AS BLOB), CAST('enc:v1:priv' AS BLOB),
		        'sha256','aes256','2026-08-21T00:00:00Z','2026-08-21T00:00:00Z')`); err != nil {
		t.Fatalf("seed v3: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO polling_targets
			(id, client_id, name, ip_address, credentials_id, created_at, updated_at)
		VALUES ('t1','default','target','10.0.0.1','legacy-v2c',
		        '2026-08-21T00:00:00Z','2026-08-21T00:00:00Z')`); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	upTo(t, db, credMigrationVersion)

	var kind, level string
	if err := db.QueryRowContext(ctx,
		`SELECT kind, COALESCE(security_level,'') FROM device_credentials WHERE id='legacy-v3'`,
	).Scan(&kind, &level); err != nil {
		t.Fatalf("read migrated v3: %v", err)
	}
	if kind != polling.CredentialKindV3 || level != polling.SecurityLevelAuthPriv {
		t.Errorf("v3 migrated as kind=%q level=%q, want v3/authPriv", kind, level)
	}

	var proto string
	if err := db.QueryRowContext(ctx,
		`SELECT snmp_v3_auth_proto FROM device_credentials WHERE id='legacy-v3'`).Scan(&proto); err != nil {
		t.Fatalf("read proto: %v", err)
	}
	if proto != "SHA256" {
		t.Errorf("auth proto = %q, want SHA256 — lowercase names must be folded, not rejected", proto)
	}

	assertTargetBinding(t, db, "legacy-v2c")

	// Down and back up: the binding must survive both directions.
	downTo(t, db, credMigrationVersion-1)
	assertTargetBinding(t, db, "legacy-v2c")
	upTo(t, db, credMigrationVersion)
	assertTargetBinding(t, db, "legacy-v2c")
}

// assertTargetBinding fails when the target lost its credential reference,
// which is what an ON DELETE SET NULL firing during a rebuild looks like.
func assertTargetBinding(t *testing.T, db *sql.DB, want string) {
	t.Helper()
	var bound sql.NullString
	if err := db.QueryRowContext(context.Background(),
		`SELECT credentials_id FROM polling_targets WHERE id='t1'`).Scan(&bound); err != nil {
		t.Fatalf("read target binding: %v", err)
	}
	if bound.String != want {
		t.Fatalf("target binding = %q, want %q — the rebuild unbound it", bound.String, want)
	}
}

func TestMigration00011RefusesAnAmbiguousLegacyRow(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, credMigrationVersion-1)

	// A community string AND a v3 user. The canonical form has no name for it,
	// and guessing which half was meant is how a device gets polled with the
	// wrong secret — so the migration must abort rather than pick one.
	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO device_credentials
			(id, client_id, name, snmp_community_enc, snmp_v3_user, created_at, updated_at)
		VALUES ('ambiguous','default','both',CAST('enc:v1:community' AS BLOB),'operator',
		        '2026-08-21T00:00:00Z','2026-08-21T00:00:00Z')`); err != nil {
		t.Fatalf("seed ambiguous row: %v", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	if _, upErr := provider.UpTo(context.Background(), credMigrationVersion); upErr == nil {
		t.Error("migration accepted a row that is both a v2c and a v3 credential")
	}
}

// downTo rolls the schema back to version.
func downTo(t *testing.T, db *sql.DB, version int64) {
	t.Helper()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	if _, downErr := provider.DownTo(context.Background(), version); downErr != nil {
		t.Fatalf("goose down to %d: %v", version, downErr)
	}
}
