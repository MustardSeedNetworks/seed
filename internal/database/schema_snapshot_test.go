package database_test

// schema_snapshot_test.go captures the full domain schema the migration runner
// produces as a checked-in golden artifact (testdata/schema.sql). It is the
// regression gate ADR-0006 mandates ("a migrate-from-empty → assert schema test
// gates drift") and the ground-truth the Phase-5 collapse to a goose-managed
// STRICT 0001_init.sql baseline must reproduce.
//
// Regenerate:  UPDATE_SCHEMA_GOLDEN=1 go test ./internal/database/ -run TestSchemaSnapshot
// Verify:      go test ./internal/database/ -run TestSchemaSnapshot

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// schemaBookkeepingTables are migration-bookkeeping tables: runner-internal and
// not part of the domain schema. Excluding both keeps the snapshot stable across
// the Phase-5 runner swap (homegrown → goose).
func schemaBookkeepingTables() map[string]bool {
	return map[string]bool{
		"schema_migrations": true, // homegrown index+1 runner
		"goose_db_version":  true, // goose runner (post Phase-5 collapse)
	}
}

func TestSchemaSnapshot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "schema.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	got := dumpSchema(t, db.WriteConn())

	goldenPath := filepath.Join("testdata", "schema.sql")
	if os.Getenv("UPDATE_SCHEMA_GOLDEN") == "1" {
		if mkErr := os.MkdirAll(filepath.Dir(goldenPath), 0o750); mkErr != nil {
			t.Fatalf("mkdir testdata: %v", mkErr)
		}
		if wErr := os.WriteFile(goldenPath, []byte(got), 0o600); wErr != nil {
			t.Fatalf("write golden: %v", wErr)
		}
		t.Logf("updated %s (%d bytes)", goldenPath, len(got))
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (regenerate with UPDATE_SCHEMA_GOLDEN=1): %v", err)
	}
	if got != string(want) {
		t.Errorf("schema drift vs %s: the migration runner's schema changed.\n"+
			"If intended, regenerate with UPDATE_SCHEMA_GOLDEN=1 and review the diff.", goldenPath)
	}
}

// schemaObject is one DDL object from sqlite_master.
type schemaObject struct {
	typ  string
	name string
	ddl  string
}

// dumpSchema returns a normalized, deterministic dump of the domain schema
// (tables, indexes, triggers, views) the migration runner produced, excluding
// migration-bookkeeping tables and SQLite internals. Objects are ordered by
// (type, name) so the snapshot is independent of creation order.
func dumpSchema(t *testing.T, conn *sql.DB) string {
	t.Helper()
	rows, err := conn.Query(`
		SELECT type, name, tbl_name, sql FROM sqlite_master
		WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%'
	`)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer func() { _ = rows.Close() }()

	bookkeeping := schemaBookkeepingTables()
	var objects []schemaObject
	for rows.Next() {
		var typ, name, tblName, ddl string
		if scanErr := rows.Scan(&typ, &name, &tblName, &ddl); scanErr != nil {
			t.Fatalf("scan sqlite_master: %v", scanErr)
		}
		if bookkeeping[name] || bookkeeping[tblName] {
			continue
		}
		objects = append(objects, schemaObject{typ: typ, name: name, ddl: strings.TrimSpace(ddl)})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("iterate sqlite_master: %v", rowsErr)
	}

	sort.Slice(objects, func(i, j int) bool {
		if objects[i].typ != objects[j].typ {
			return objects[i].typ < objects[j].typ
		}
		return objects[i].name < objects[j].name
	})

	var b strings.Builder
	for _, o := range objects {
		b.WriteString("-- ")
		b.WriteString(o.typ)
		b.WriteString(": ")
		b.WriteString(o.name)
		b.WriteString("\n")
		b.WriteString(o.ddl)
		b.WriteString(";\n\n")
	}
	return b.String()
}

// The database every test gets from testDB is a copy of a file the
// migration runner produced once, not a fresh migration run. That is
// only sound while the copy is indistinguishable from the run — so
// this compares the two directly, against the same golden the snapshot
// above gates. It fails if the template is ever built from something
// other than a full migration, or if copying one file were to lose
// part of the database.
func TestTemplateDatabaseMatchesAFreshMigration(t *testing.T) {
	t.Parallel()

	fresh, err := database.Open(filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatalf("open fresh db: %v", err)
	}
	defer func() { _ = fresh.Close() }()

	fromTemplate, cleanup := testDB(t)
	defer cleanup()

	if got, want := dumpSchema(t, fromTemplate.WriteConn()), dumpSchema(t, fresh.WriteConn()); got != want {
		t.Errorf("schema from the copied template differs from a fresh migration run")
	}
	if gooseVersion(t, fromTemplate.WriteConn()) != gooseVersion(t, fresh.WriteConn()) {
		t.Error("migration version from the copied template differs from a fresh migration run")
	}
}

// gooseVersion is the highest applied migration, which says the copy
// carries the runner's bookkeeping too — a schema that matches while
// the version does not would migrate again on the next open.
func gooseVersion(t *testing.T, conn *sql.DB) int64 {
	t.Helper()
	var version int64
	if err := conn.QueryRow(
		`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1`,
	).Scan(&version); err != nil {
		t.Fatalf("read goose version: %v", err)
	}
	return version
}
