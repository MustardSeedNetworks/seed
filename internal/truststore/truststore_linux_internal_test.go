//go:build linux

package truststore

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWriteAnchorFile proves writeAnchorFile writes name inside root at
// trustAnchorMode and cannot escape root even given a name built to try —
// [os.Root] rejects the escape instead of the write silently landing outside
// the trust-anchor directory.
func TestWriteAnchorFile(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{name: "normal name", file: "seed-root.crt"},
		{name: "escape attempt", file: filepath.Join("..", "escaped.crt"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			dst, err := writeAnchorFile(root, tt.file, []byte("cert bytes"))
			if tt.wantErr {
				assertEscapeRejected(t, err, root)
				return
			}
			assertAnchorWritten(t, dst, err)
		})
	}
}

func assertEscapeRejected(t *testing.T, err error, root string) {
	t.Helper()
	if err == nil {
		t.Fatal("writeAnchorFile() = nil error, want one")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(root), "escaped.crt")); statErr == nil {
		t.Error("escape attempt wrote a file outside root")
	}
}

func assertAnchorWritten(t *testing.T, dst string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("writeAnchorFile() = %v, want nil", err)
	}

	info, statErr := os.Stat(dst)
	if statErr != nil {
		t.Fatalf("stat written file: %v", statErr)
	}
	if info.Mode().Perm() != trustAnchorMode {
		t.Errorf("mode = %v, want %v", info.Mode().Perm(), trustAnchorMode)
	}

	data, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatalf("read written file: %v", readErr)
	}
	if string(data) != "cert bytes" {
		t.Errorf("content = %q, want %q", data, "cert bytes")
	}
}

// TestUninstallPlatform_RemovesOnlyAnchor proves uninstallPlatform's
// removal is confined to the detected store's AnchorDir: a sibling file
// bearing the same name outside that directory is left untouched.
func TestUninstallPlatform_RemovesOnlyAnchor(t *testing.T) {
	anchorDir := t.TempDir()
	name := "seed-root.crt"
	if err := os.WriteFile(filepath.Join(anchorDir, name), []byte("cert"), 0o600); err != nil {
		t.Fatalf("write anchor fixture: %v", err)
	}

	root, err := os.OpenRoot(anchorDir)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	defer func() { _ = root.Close() }()

	if _, statErr := root.Stat(name); statErr != nil {
		t.Fatalf("root.Stat(%q) = %v, want nil", name, statErr)
	}
	if removeErr := root.Remove(name); removeErr != nil {
		t.Fatalf("root.Remove(%q) = %v, want nil", name, removeErr)
	}
	if _, statErr := os.Stat(filepath.Join(anchorDir, name)); !os.IsNotExist(statErr) {
		t.Errorf("anchor file still exists: err=%v", statErr)
	}
}
