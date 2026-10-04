package database_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// TestOpenKeepsURIReservedPathsDistinct pins that the filesystem path reaches
// SQLite as a path, not as URI syntax. Interpolated raw into the file: URI, a
// '#' started a fragment and a '?' started the query, so shared#one.db and
// shared#two.db both opened a third file named "shared" (#2906), and '%' was
// percent-decoded into a different name.
func TestOpenKeepsURIReservedPathsDistinct(t *testing.T) {
	tests := []struct {
		name         string
		first, other string
		windowsValid bool
	}{
		{"fragment", "shared#one.db", "shared#two.db", true},
		{"query", "shared?one.db", "shared?two.db", false},
		{"percent escape", "shared%23.db", "shared#.db", true},
		{"space", "shared one.db", "shared two.db", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS == "windows" && !tt.windowsValid {
				t.Skip("character is not valid in a Windows file name")
			}
			assertDistinctDatabases(t, t.TempDir(), tt.first, tt.other)
		})
	}
}

// assertDistinctDatabases opens first and other in dir and fails if they share
// one database, if other lost its DSN pragmas, or if any third file appears.
func assertDistinctDatabases(t *testing.T, dir, first, other string) {
	t.Helper()
	ctx := context.Background()

	db := openReserved(t, filepath.Join(dir, first))
	if _, err := db.Exec(ctx, "CREATE TABLE marker (id INTEGER)"); err != nil {
		t.Fatalf("create marker in %q: %v", first, err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close %q: %v", first, err)
	}

	db = openReserved(t, filepath.Join(dir, other))
	var n int
	if err := db.QueryRow(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE name = 'marker'").Scan(&n); err != nil {
		t.Fatalf("query %q: %v", other, err)
	}
	if n != 0 {
		t.Fatalf("%q sees the table created in %q: the paths alias one database", other, first)
	}

	// The pragmas ride the DSN's query, so a path that swallowed it would open
	// with SQLite's default rollback journal.
	var mode string
	if err := db.QueryRow(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("journal_mode on %q: %v", other, err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode on %q = %q, want wal", other, mode)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		switch e.Name() {
		case first, other, other + "-wal", other + "-shm":
		default:
			t.Errorf("unexpected file %q beside the requested databases", e.Name())
		}
	}
}

func openReserved(t *testing.T, path string) *database.DB {
	t.Helper()
	db, err := database.Open(path)
	if err != nil {
		t.Fatalf("Open(%q): %v", path, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
