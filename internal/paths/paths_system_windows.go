//go:build windows

package paths

import (
	"errors"
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
// layout: the SCM started this process, or the SCM has the Seed service
// registered, so the config an administrator's console edits is the one the
// service reads. Only the SCM decides. A directory under %ProgramData% proves
// nothing, since any user can create one and steer an elevated console into a
// tree they control. A console on a machine with no install keeps the user
// layout.
func windowsSystemLayout() bool {
	return isWindowsService() || serviceRegistered()
}

// isWindowsService reports whether the Service Control Manager started this
// process.
func isWindowsService() bool {
	isService, err := svc.IsWindowsService()
	return err == nil && isService
}

// serviceRegistered reports whether the SCM knows the Seed service. A query
// refused for lack of rights still means the service exists, and the caller
// then needs elevation; any other failure is "not installed".
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
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	_ = windows.CloseServiceHandle(service)
	return true
}
