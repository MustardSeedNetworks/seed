//go:build windows

package paths_test

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"

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

// A directory under %ProgramData% is not proof of an install: any user can
// create one. With no service registered, ModeAuto must stay on the user layout
// even when %ProgramData%\Seed\config exists.
func TestResolve_AutoMode_IgnoresPlantedConfigDir_Windows(t *testing.T) {
	scm, err := mgr.Connect()
	if err != nil {
		t.Skipf("cannot query the SCM: %v", err)
	}
	defer func() { _ = scm.Disconnect() }()
	if service, openErr := scm.OpenService(paths.WindowsServiceName); openErr == nil {
		_ = service.Close()
		t.Skip("a Seed service is registered on this machine")
	}

	system := paths.Resolve(paths.ModeSystem)
	if _, statErr := os.Stat(system.Root); statErr == nil {
		t.Skipf("%s already exists", system.Root)
	}
	if mkErr := os.MkdirAll(system.ConfigDir, 0o750); mkErr != nil {
		t.Fatalf("plant config directory: %v", mkErr)
	}
	t.Cleanup(func() { _ = os.RemoveAll(system.Root) })

	if p := paths.Resolve(paths.ModeAuto); p.Mode != paths.ModeUser {
		t.Errorf("a planted %s selected mode %v, want ModeUser", system.ConfigDir, p.Mode)
	}
	if dir, ok := paths.WindowsSystemConfigDir(); ok {
		t.Errorf("WindowsSystemConfigDir = %q, true; want false with no service registered", dir)
	}
}
