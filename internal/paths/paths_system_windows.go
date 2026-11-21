//go:build windows

package paths

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

// systemDirName is the directory under %ProgramData% and %ProgramFiles% that
// belongs to Seed. It is capitalised, unlike the Unix appName, because it is
// the name an administrator sees in Explorer; deploy/windows/install.ps1
// creates the same directories.
const systemDirName = "Seed"

// WindowsServiceName is the Service Control Manager name install.ps1 and
// `seed service install` register.
const WindowsServiceName = "SeedNetworkDiagnostics"

// resolveSystemPaths lays the service out under %ProgramData%\Seed, with the
// binary in %ProgramFiles%\Seed. The folders come from the Known Folder API,
// not the environment: an environment variable is whatever the launching
// process says, and this tree holds the database, the TLS key and the JWT
// secret.
func resolveSystemPaths(p *Paths) {
	programData := knownFolder(windows.FOLDERID_ProgramData, "ProgramData", `C:\ProgramData`)
	programFiles := knownFolder(windows.FOLDERID_ProgramFiles, "ProgramFiles", `C:\Program Files`)

	p.Root = filepath.Join(programData, systemDirName)
	p.ConfigDir = filepath.Join(p.Root, "config")
	p.DataDir = filepath.Join(p.Root, "data")
	p.LogDir = filepath.Join(p.Root, "logs")
	p.CacheDir = filepath.Join(p.Root, "cache")
	p.CertDir = filepath.Join(p.Root, "certs")
	p.BinaryDir = filepath.Join(programFiles, systemDirName)
}

// knownFolder returns the Known Folder path, falling back to the environment
// and then to the stock location when the API fails, which it does not on a
// working install.
func knownFolder(id *windows.KNOWNFOLDERID, envVar, stock string) string {
	if dir, err := windows.KnownFolderPath(id, windows.KF_FLAG_DEFAULT); err == nil {
		return dir
	}
	if dir := os.Getenv(envVar); dir != "" {
		return dir
	}
	return stock
}

// windowsSystemLayout reports whether ModeAuto should resolve to the system
// layout: the SCM started this process, or a service install exists, so the
// config an administrator's console edits is the one the service reads. A
// console on a machine with no install keeps the user layout.
func windowsSystemLayout() bool {
	return isWindowsService() || serviceRegistered() || systemConfigDirExists()
}

// isWindowsService reports whether the Service Control Manager started this
// process.
func isWindowsService() bool {
	isService, err := svc.IsWindowsService()
	return err == nil && isService
}

// serviceRegistered reports whether the SCM knows the Seed service. A denied
// query still means the service exists.
func serviceRegistered() bool {
	manager, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseServiceHandle(manager) }()

	name, err := windows.UTF16PtrFromString(WindowsServiceName)
	if err != nil {
		return false
	}
	service, err := windows.OpenService(manager, name, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return !errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST)
	}
	_ = windows.CloseServiceHandle(service)
	return true
}

// systemConfigDirExists reports whether %ProgramData%\Seed\config exists. The
// root is closed to ordinary users, so an access-denied stat from a
// non-elevated console is a yes, not a no.
func systemConfigDirExists() bool {
	var system Paths
	resolveSystemPaths(&system)
	_, err := os.Stat(system.ConfigDir)
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}
