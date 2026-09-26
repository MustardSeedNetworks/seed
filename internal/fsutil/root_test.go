package fsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/fsutil"
)

func TestRootAt(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sub", "target.txt")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"resolves to the same file", filePath, false},
		{"relative path from cwd", "root_test.go", false},
		{"nonexistent parent directory", filepath.Join(dir, "missing", "x.txt"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, name, err := fsutil.RootAt(tt.path)
			if tt.wantErr {
				if err == nil {
					_ = root.Close()
					t.Fatal("RootAt() = nil error, want one")
				}
				return
			}
			if err != nil {
				t.Fatalf("RootAt(%q) = %v, want nil", tt.path, err)
			}
			defer func() { _ = root.Close() }()

			if got, want := name, filepath.Base(tt.path); got != want {
				t.Errorf("name = %q, want %q", got, want)
			}
			if _, statErr := root.Stat(name); statErr != nil {
				t.Errorf("root.Stat(%q) = %v, want nil (RootAt should resolve to the same file)", name, statErr)
			}
		})
	}
}

// TestRootAt_ScopesToPathsParent proves RootAt scopes to whatever directory
// path lexically names as its parent — here [filepath.Join] has already
// resolved "sub/../secret.txt" down to "secret.txt" before RootAt ever sees
// it, so the parent is dir itself, not sub. RootAt confines to that lexical
// parent; it is not a directory-confinement boundary against a
// caller-chosen root (see internal/config's BackupManager or
// internal/truststore for that stronger guarantee, built on [os.Root]
// scoped to a fixed directory).
func TestRootAt_ScopesToPathsParent(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("do not read"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	root, name, err := fsutil.RootAt(filepath.Join(sub, "..", "secret.txt"))
	if err != nil {
		t.Fatalf("RootAt() = %v, want nil", err)
	}
	defer func() { _ = root.Close() }()

	if root.Name() != dir {
		t.Errorf("root scoped to %q, want %q", root.Name(), dir)
	}
	if name != "secret.txt" {
		t.Errorf("name = %q, want %q", name, "secret.txt")
	}
}
