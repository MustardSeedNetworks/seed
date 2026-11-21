//go:build windows

package paths

import (
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

// isWindowsService reports whether the Service Control Manager started this
// process.
func isWindowsService() bool {
	isService, err := svc.IsWindowsService()
	return err == nil && isService
}
