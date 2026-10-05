package dbtest_test

import (
	"os"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
)

// Two copies are two databases: a write to one is invisible to the other.
// A template handed out by reference, or a shared path, would fail this.
func TestCopiesAreIndependent(t *testing.T) {
	t.Parallel()
	a, b := dbtest.Open(t), dbtest.Open(t)
	if a.Path() == b.Path() {
		t.Fatalf("both copies share %s", a.Path())
	}

	if _, err := a.CreateUser(t.Context(), "only-in-a", "$2a$10$x", roles.Viewer); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := b.GetUser(t.Context(), "only-in-a"); err == nil {
		t.Fatal("a user written to one copy is visible in the other")
	}
}

// The copy holds credentials like the real file, so it must be owner-only.
func TestPathIsOwnerOnly(t *testing.T) {
	t.Parallel()
	info, err := os.Stat(dbtest.Path(t))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("template copy mode %o, want 600", mode)
	}
}
