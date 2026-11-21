//go:build windows

package paths_test

import (
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

// A console process is not a service, so ModeAuto keeps resolving user paths
// for it; only the SCM-started service flips to the system layout.
func TestResolve_AutoMode_ConsoleIsUser_Windows(t *testing.T) {
	if p := paths.Resolve(paths.ModeAuto); p.Mode != paths.ModeUser {
		t.Errorf("console process resolved mode %v, want ModeUser", p.Mode)
	}
}
