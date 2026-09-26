package fsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/fsutil"
)

func TestRootAt(t *testing.T) {
	dir := resolvedTempDir(t)
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

// TestRootAt_FollowsSymlinkedFile proves RootAt can still address a path
// that is itself a symlink pointing outside its own directory (an
// alternatives-style config layout, for example) — the case a bare
// [os.Root] scoped to the symlink's own (unresolved) parent directory
// would reject, since [os.Root] refuses to follow a symlink that leaves
// its scope.
func TestRootAt_FollowsSymlinkedFile(t *testing.T) {
	realDir := resolvedTempDir(t)
	realPath := filepath.Join(realDir, "seed.json")
	if err := os.WriteFile(realPath, []byte("original"), 0o600); err != nil {
		t.Fatalf("write real file: %v", err)
	}

	linkDir := resolvedTempDir(t)
	linkPath := filepath.Join(linkDir, "seed.json")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	root, name, err := fsutil.RootAt(linkPath)
	if err != nil {
		t.Fatalf("RootAt(%q) = %v, want nil", linkPath, err)
	}
	defer func() { _ = root.Close() }()

	if writeErr := root.WriteFile(name, []byte("updated"), 0o600); writeErr != nil {
		t.Fatalf("root.WriteFile(%q) = %v, want nil", name, writeErr)
	}

	data, err := os.ReadFile(realPath)
	if err != nil {
		t.Fatalf("read real file: %v", err)
	}
	if string(data) != "updated" {
		t.Errorf("real file contents = %q, want %q (write through the symlink did not land)", data, "updated")
	}
}

// TestRootAt_MissingFileCanStillBeCreated proves RootAt still lets a caller
// create path when nothing exists there yet — the config-restore-after-
// deletion case: [filepath.EvalSymlinks] fails on a missing leaf, so RootAt
// falls back to resolving path's parent directory and addresses the file
// by its own (unresolved) base name, matching what a direct
// [os.WriteFile] call would have done.
func TestRootAt_MissingFileCanStillBeCreated(t *testing.T) {
	dir := resolvedTempDir(t)
	missing := filepath.Join(dir, "config.json")

	root, name, err := fsutil.RootAt(missing)
	if err != nil {
		t.Fatalf("RootAt(%q) = %v, want nil", missing, err)
	}
	defer func() { _ = root.Close() }()

	if writeErr := root.WriteFile(name, []byte("created"), 0o600); writeErr != nil {
		t.Fatalf("root.WriteFile(%q) = %v, want nil", name, writeErr)
	}
	data, readErr := os.ReadFile(missing)
	if readErr != nil {
		t.Fatalf("read created file: %v", readErr)
	}
	if string(data) != "created" {
		t.Errorf("contents = %q, want %q", data, "created")
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
	dir := resolvedTempDir(t)
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

// resolvedTempDir returns t.TempDir() with its own symlinks resolved (on
// macOS, $TMPDIR sits under /var, itself a symlink to /private/var), so
// tests can compare it directly against RootAt's symlink-resolved output.
func resolvedTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks(TempDir): %v", err)
	}
	return dir
}
