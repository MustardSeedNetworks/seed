package database_test

// migration_users_client_id_test.go covers the one thing 00010 can silently get
// wrong. `users` is the parent of two ON DELETE CASCADE children, so rebuilding
// it with foreign_keys=ON fires those cascades on the DROP and empties them —
// an upgrade that quietly destroys every API token. The migration disables FK
// enforcement for the rebuild; this proves it, by migrating a populated DB
// across the boundary rather than inspecting the schema.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

// migrateTo brings a fresh DB up to version and returns the handle.
func migrateTo(t *testing.T, version int64) *sql.DB {
	t.Helper()

	dir := t.TempDir()
	rawDB, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "m.db")+"?_txlock=immediate")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })
	if _, pragmaErr := rawDB.Exec("PRAGMA foreign_keys = ON"); pragmaErr != nil {
		t.Fatalf("pragma: %v", pragmaErr)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, rawDB, os.DirFS("migrations"))
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	if _, upErr := provider.UpTo(context.Background(), version); upErr != nil {
		t.Fatalf("goose up to %d: %v", version, upErr)
	}
	return rawDB
}

func upTo(t *testing.T, db *sql.DB, version int64) {
	t.Helper()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS("migrations"))
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	if _, upErr := provider.UpTo(context.Background(), version); upErr != nil {
		t.Fatalf("goose up to %d: %v", version, upErr)
	}
}

func TestMigration00010PreservesUsersAndCascadeChildren(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 9)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, role, created_at, updated_at)
		VALUES ('operator1', 'hash', 'operator', '2026-08-20T00:00:00Z', '2026-08-20T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO api_tokens (id, owner_username, name, token_hash, prefix, scope, created_at)
		VALUES ('tok1', 'operator1', 'ci', 'deadbeef', 'seed_ci', 'operator', '2026-08-20T00:00:00Z')
	`); err != nil {
		t.Fatalf("insert api token: %v", err)
	}

	upTo(t, db, 10)

	var clientID string
	if err := db.QueryRowContext(ctx,
		`SELECT client_id FROM users WHERE username = 'operator1'`).Scan(&clientID); err != nil {
		t.Fatalf("user did not survive the rebuild: %v", err)
	}
	if clientID != "default" {
		t.Errorf("client_id = %q, want %q — existing rows must backfill to the default client", clientID, "default")
	}

	var tokens int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM api_tokens WHERE owner_username = 'operator1'`).Scan(&tokens); err != nil {
		t.Fatalf("count api tokens: %v", err)
	}
	if tokens != 1 {
		t.Errorf("api_tokens rows = %d, want 1 — the rebuild fired an ON DELETE CASCADE", tokens)
	}

	rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		t.Error("foreign_key_check reported violations after the rebuild")
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("foreign_key_check rows: %v", rowsErr)
	}

	var fkOn int
	if pragmaErr := db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fkOn); pragmaErr != nil {
		t.Fatalf("read foreign_keys pragma: %v", pragmaErr)
	}
	if fkOn != 1 {
		t.Error("the migration left foreign_keys disabled on the connection")
	}
}

func TestMigration00010RejectsUnknownClient(t *testing.T) {
	t.Parallel()

	db := migrateTo(t, 10)
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO users (username, password_hash, role, client_id, created_at, updated_at)
		VALUES ('ghost', 'hash', 'viewer', 'no-such-client', '2026-08-20T00:00:00Z', '2026-08-20T00:00:00Z')
	`)
	if err == nil {
		t.Fatal("expected the client_id foreign key to reject an unknown client")
	}
}
