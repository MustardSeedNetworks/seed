package database_test

// goose_baseline_test.go bootstraps and proves the Phase-5 goose baseline
// (ADR-0006). TestGenerateGooseBaseline (run once, guarded) writes
// migrations/00001_init.sql faithfully from the legacy runner's schema;
// TestGooseBaselineReproducesSchema then applies that baseline via goose to a
// fresh empty DB and asserts the resulting schema matches the snapshot golden —
// i.e. goose + the baseline produce byte-identical schema to the homegrown
// runner. That proof gates the runner swap.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/krisarmstrong/seed/internal/database"
)

// schemaDDLByType returns the (name, ddl) pairs of the given sqlite_master type
// (e.g. "table", "index"), sorted by name, excluding bookkeeping + internals.
func schemaDDLByType(t *testing.T, conn *sql.DB, objType string) [][2]string {
	t.Helper()
	rows, err := conn.Query(
		`SELECT name, sql FROM sqlite_master WHERE type=? AND sql IS NOT NULL AND name NOT LIKE 'sqlite_%'`,
		objType,
	)
	if err != nil {
		t.Fatalf("query %s: %v", objType, err)
	}
	defer func() { _ = rows.Close() }()

	bookkeeping := schemaBookkeepingTables()
	var out [][2]string
	for rows.Next() {
		var name, ddl string
		if scanErr := rows.Scan(&name, &ddl); scanErr != nil {
			t.Fatalf("scan: %v", scanErr)
		}
		if bookkeeping[name] {
			continue
		}
		out = append(out, [2]string{name, strings.TrimSpace(ddl)})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("rows: %v", rowsErr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

func TestGenerateGooseBaseline(t *testing.T) {
	if os.Getenv("GEN_BASELINE") != "1" {
		t.Skip("set GEN_BASELINE=1 to (re)generate migrations/00001_init.sql")
	}

	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "gen.db"))
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	defer func() { _ = db.Close() }()

	tables := schemaDDLByType(t, db.Conn(), "table")
	indexes := schemaDDLByType(t, db.Conn(), "index")

	var b strings.Builder
	b.WriteString("-- 00001_init.sql — Phase-5 schema baseline (ADR-0006).\n")
	b.WriteString("-- Faithful collapse of the legacy migration history; generated from the\n")
	b.WriteString("-- schema the old runner produced. NOT yet STRICT (that is a follow-up).\n")
	b.WriteString("-- Regenerate: GEN_BASELINE=1 go test ./internal/database/ -run TestGenerateGooseBaseline\n\n")
	b.WriteString("-- +goose Up\n")
	for _, tb := range tables {
		b.WriteString(tb[1])
		b.WriteString(";\n")
	}
	b.WriteString("\n")
	for _, ix := range indexes {
		b.WriteString(ix[1])
		b.WriteString(";\n")
	}
	b.WriteString("\n-- +goose Down\n")
	for _, tb := range slices.Backward(tables) {
		b.WriteString("DROP TABLE IF EXISTS ")
		b.WriteString(tb[0])
		b.WriteString(";\n")
	}

	if mkErr := os.MkdirAll("migrations", 0o750); mkErr != nil {
		t.Fatalf("mkdir migrations: %v", mkErr)
	}
	if wErr := os.WriteFile(filepath.Join("migrations", "00001_init.sql"), []byte(b.String()), 0o600); wErr != nil {
		t.Fatalf("write baseline: %v", wErr)
	}
	t.Logf("wrote migrations/00001_init.sql (%d tables, %d indexes)", len(tables), len(indexes))
}

func TestGooseBaselineReproducesSchema(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rawDB, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "goose.db")+"?_txlock=immediate")
	if err != nil {
		t.Fatalf("open raw sqlite: %v", err)
	}
	defer func() { _ = rawDB.Close() }()
	if _, pragmaErr := rawDB.Exec("PRAGMA foreign_keys = ON"); pragmaErr != nil {
		t.Fatalf("pragma: %v", pragmaErr)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, rawDB, os.DirFS("migrations"))
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	if _, upErr := provider.Up(context.Background()); upErr != nil {
		t.Fatalf("goose up: %v", upErr)
	}

	got := dumpSchema(t, rawDB)
	want, err := os.ReadFile(filepath.Join("testdata", "schema.sql"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		t.Errorf("goose baseline schema != snapshot golden: the baseline does not faithfully\n" +
			"reproduce the legacy schema. Regenerate with GEN_BASELINE=1 and review.")
	}
}
