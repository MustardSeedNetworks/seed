//go:build windows

package paths_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/MustardSeedNetworks/seed/internal/paths"
)

func TestResolve_SystemMode_Windows(t *testing.T) {
	programData, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		t.Fatalf("resolve ProgramData: %v", err)
	}
	programFiles, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, windows.KF_FLAG_DEFAULT)
	if err != nil {
		t.Fatalf("resolve ProgramFiles: %v", err)
	}
	root := filepath.Join(programData, "Seed")

	p := paths.Resolve(paths.ModeSystem)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Root", p.Root, root},
		{"ConfigDir", p.ConfigDir, filepath.Join(root, "config")},
		{"ConfigFile", p.ConfigFile, filepath.Join(root, "config", "seed.json")},
		{"DataDir", p.DataDir, filepath.Join(root, "data")},
		{"LogDir", p.LogDir, filepath.Join(root, "logs")},
		{"CacheDir", p.CacheDir, filepath.Join(root, "cache")},
		{"CertDir", p.CertDir, filepath.Join(root, "certs")},
		{"BinaryDir", p.BinaryDir, filepath.Join(programFiles, "Seed")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// A console resolves the system layout exactly when a service install is
// visible to it, so an administrator's commands and the service read one
// config; with no install it keeps the user layout.
func TestResolve_AutoMode_FollowsServiceInstall_Windows(t *testing.T) {
	// The config directory is the install signal a test can read without the
	// SCM; the job that runs this has no service registered before it installs.
	want := paths.ModeUser
	if _, err := os.Stat(paths.Resolve(paths.ModeSystem).ConfigDir); err == nil || !errors.Is(err, fs.ErrNotExist) {
		want = paths.ModeSystem
	}
	p := paths.Resolve(paths.ModeAuto)
	if p.Mode != want {
		t.Errorf("ModeAuto resolved %v, want %v", p.Mode, want)
	}
	dir, ok := paths.WindowsSystemConfigDir()
	if ok != (want == paths.ModeSystem) {
		t.Errorf("WindowsSystemConfigDir ok = %v, want %v", ok, want == paths.ModeSystem)
	}
	if ok && dir != paths.Resolve(paths.ModeSystem).ConfigDir {
		t.Errorf("WindowsSystemConfigDir = %q, want the system ConfigDir", dir)
	}
}
