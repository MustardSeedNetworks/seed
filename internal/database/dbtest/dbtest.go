// Package dbtest hands tests a migrated database without running every
// migration for each test. modernc.org/sqlite is pure Go, so under -race a
// full migration run costs about 1.3 s on its own and ~10 s when a package's
// parallel tests contend for the CPU; opening a copy of a file migrated once
// costs about 80 ms (seed#2614, seed#2719).
//
// The copy is opened through database.Open like any other file, so every
// pragma, the owner-only file mode and the no-op migration check still run.
// Tests that exercise the migration run itself must keep opening an empty
// path.
package dbtest

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

const templateName = "template.db"

// template is the migrated database file's bytes, built once per test binary.
var template struct {
	once  sync.Once
	bytes []byte
	err   error
}

func build() ([]byte, error) {
	dir, err := os.MkdirTemp("", "seed-dbtest-*")
	if err != nil {
		return nil, fmt.Errorf("create template dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	db, err := database.Open(filepath.Join(dir, templateName))
	if err != nil {
		return nil, fmt.Errorf("migrate template: %w", err)
	}
	// Close checkpoints the WAL into the main file, so that one file is the
	// whole database.
	if closeErr := db.Close(); closeErr != nil {
		return nil, fmt.Errorf("close template: %w", closeErr)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open template dir: %w", err)
	}
	defer func() { _ = root.Close() }()
	return root.ReadFile(templateName)
}

// Path writes a fresh copy of the migrated template into the test's temp dir
// and returns its path, for code under test that opens the database itself.
func Path(tb testing.TB) string {
	tb.Helper()
	template.once.Do(func() { template.bytes, template.err = build() })
	if template.err != nil {
		tb.Fatalf("build template database: %v", template.err)
	}
	path := filepath.Join(tb.TempDir(), "seed.db")
	if err := os.WriteFile(path, template.bytes, 0o600); err != nil {
		tb.Fatalf("write template copy: %v", err)
	}
	return path
}

// Open returns a database on a fresh copy of the template, closed when the
// test ends. Closing it earlier is fine: DB.Close is idempotent.
func Open(tb testing.TB) *database.DB {
	tb.Helper()
	db, err := database.Open(Path(tb))
	if err != nil {
		tb.Fatalf("open template copy: %v", err)
	}
	tb.Cleanup(func() { _ = db.Close() })
	return db
}
