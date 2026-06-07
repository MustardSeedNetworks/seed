package database_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// TestOpenRestrictsFileMode verifies the database file (and any WAL/SHM sidecars)
// are created owner-only (0600). The DB holds tokens, credentials, and audit
// data, so the mode must be explicit rather than left to the process umask.
func TestOpenRestrictsFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file-mode semantics do not apply on Windows")
	}

	dbPath := filepath.Join(t.TempDir(), "perm.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		info, statErr := os.Stat(p)
		if statErr != nil {
			continue // sidecars are created lazily; absence is fine
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Errorf("%s mode = %o, want 600", p, mode)
		}
	}
}
