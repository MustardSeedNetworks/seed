package database_test

// goose_baseline_test.go proves the Phase-5 goose baseline (ADR-0006): applying
// migrations/0001_init.sql via goose to a fresh empty DB yields a schema
// byte-identical to the snapshot golden (#1476) — i.e. the collapse of the
// legacy migration history is faithful. (The one-time generator that derived the
// baseline from the legacy runner was removed with that runner in Phase 5b-2;
// the baseline is now a hand-maintained .sql file like any goose migration.)

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

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
			"reproduce the legacy schema. Review migrations/0001_init.sql against the golden.")
	}
}
