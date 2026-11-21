//go:build !windows

package paths

import "path/filepath"

// resolveSystemPaths sets system-level paths (FHS compliant).
func resolveSystemPaths(p *Paths) {
	p.ConfigDir = filepath.Join("/etc", appName)
	p.DataDir = filepath.Join("/var/lib", appName)
	p.LogDir = filepath.Join("/var/log", appName)
	p.CacheDir = filepath.Join("/var/cache", appName)
	p.BinaryDir = "/usr/local/bin"
}

// windowsSystemLayout is always false off Windows.
func windowsSystemLayout() bool { return false }
