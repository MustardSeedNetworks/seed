package database_test

import (
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// TestOpenCreatesParentDirectory pins the first-run case: the default path is
// data/seed.db, relative to the working directory, and nothing else creates
// data/. Without this, a fresh start fails to open the database, the daemon
// carries on with a nil handle, and it panics three layers later (#2380).
func TestOpenCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "seed.db")

	db, err := database.Open(path)
	if err != nil {
		t.Fatalf("Open(%q) with no parent directory: %v", path, err)
	}
	t.Cleanup(func() { _ = db.Close() })
}
