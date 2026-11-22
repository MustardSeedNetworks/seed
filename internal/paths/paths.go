package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	appName       = "seed"
	defaultConfig = "seed.json"
)

// Daemon-relative defaults. The daemon opens these relative to its working
// directory (the data directory under the installed service), so anything that
// has to find them from outside the daemon resolves them against that same
// directory rather than guessing.
const (
	// DefaultLogFile is the log the daemon writes when logging.file is unset.
	DefaultLogFile = "logs/seed.log"
	// DefaultDatabaseFile is the SQLite file the daemon opens when
	// database.path is unset.
	DefaultDatabaseFile = "data/seed.db"
	// DefaultCertDir is where the daemon keeps its generated certificate pair
	// when server.cert_file is unset.
	DefaultCertDir = "certs"
)

// Mode indicates the installation mode.
type Mode int

const (
	// ModeAuto auto-detects based on UID and systemd context.
	ModeAuto Mode = iota
	// ModeUser forces user-level installation paths.
	ModeUser
	// ModeSystem forces system-level installation paths.
	ModeSystem
)

// Paths holds resolved paths for the application.
type Paths struct {
	Mode       Mode
	ConfigDir  string
	ConfigFile string
	DataDir    string
	LogDir     string
	CacheDir   string
	BinaryDir  string

	// Root and CertDir are set only by the Windows system layout, where every
	// directory hangs off one tree under %ProgramData%. Root is also what the
	// service resolves relative config paths against.
	Root    string
	CertDir string
}

// Resolve determines paths based on mode and environment.
//
// For ModeAuto, it detects whether to use system or user paths by checking:
//   - If running as root (UID 0)
//   - If running under systemd (NOTIFY_SOCKET or INVOCATION_ID env vars)
//   - On Windows, if the process is the service or a Windows service install
//     exists (os.Getuid is -1 there, so the UID test alone would put both the
//     service and an administrator's console in user mode, two different
//     configs for one install)
//
// Returns a Paths structure with all resolved directory and file paths.
func Resolve(mode Mode) *Paths {
	actualMode := detectActualMode(mode)
	p := &Paths{Mode: actualMode}

	if actualMode == ModeSystem {
		resolveSystemPaths(p)
	} else {
		resolveUserPaths(p)
	}

	p.ConfigFile = filepath.Join(p.ConfigDir, defaultConfig)

	return p
}

// WindowsSystemConfigDir returns the system config directory and true when
// ModeAuto resolves to the Windows system layout, and "", false everywhere else.
// Code that cannot take a path from [Resolve], such as the license manager,
// uses it to follow the same layout as the config.
func WindowsSystemConfigDir() (string, bool) {
	if !windowsSystemLayout() {
		return "", false
	}
	return Resolve(ModeSystem).ConfigDir, true
}

// detectActualMode resolves ModeAuto to either ModeSystem or ModeUser.
func detectActualMode(mode Mode) Mode {
	if mode != ModeAuto {
		return mode
	}
	if isSystemdService() || windowsSystemLayout() || os.Getuid() == 0 {
		return ModeSystem
	}
	return ModeUser
}

// resolveUserPaths sets user-level paths (XDG Base Directory compliant).
func resolveUserPaths(p *Paths) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home unavailable
		homeDir = "."
	}

	p.ConfigDir = filepath.Join(xdgDir("XDG_CONFIG_HOME", homeDir, ".config"), appName)
	p.DataDir = filepath.Join(xdgDir("XDG_DATA_HOME", homeDir, ".local", "share"), appName)
	p.LogDir = filepath.Join(xdgDir("XDG_STATE_HOME", homeDir, ".local", "state"), appName, "logs")
	p.CacheDir = filepath.Join(xdgDir("XDG_CACHE_HOME", homeDir, ".cache"), appName)
	p.BinaryDir = filepath.Join(homeDir, ".local", "bin")
}

// xdgDir returns the XDG directory from environment or constructs default.
func xdgDir(envVar, homeDir string, defaultSubdirs ...string) string {
	if dir := os.Getenv(envVar); dir != "" {
		return dir
	}
	return filepath.Join(append([]string{homeDir}, defaultSubdirs...)...)
}

// ResolveConfigPath returns the config file path with priority:
//  1. Explicit path (if non-empty and not default)
//  2. SEED_CONFIG_PATH environment variable
//  3. XDG-compliant path based on mode
//
// This allows users to override config location via CLI flag or environment.
func ResolveConfigPath(explicit string, mode Mode) string {
	// Priority 1: Explicit path (but ignore if it's just the default filename)
	if explicit != "" && explicit != defaultConfig {
		return explicit
	}

	// Priority 2: Environment variable
	if envPath := os.Getenv("SEED_CONFIG_PATH"); envPath != "" {
		return envPath
	}

	// Priority 3: XDG-compliant path.
	return Resolve(mode).ConfigFile
}

// isSystemdService detects if running under systemd by checking for
// systemd-specific environment variables.
//
// Returns true if NOTIFY_SOCKET or INVOCATION_ID are set, indicating
// the process is running as a systemd service.
func isSystemdService() bool {
	// Only check on Linux where systemd is relevant
	if runtime.GOOS != "linux" {
		return false
	}

	// NOTIFY_SOCKET indicates systemd Type=notify service
	if os.Getenv("NOTIFY_SOCKET") != "" {
		return true
	}

	// INVOCATION_ID is set by systemd for all service units
	if os.Getenv("INVOCATION_ID") != "" {
		return true
	}

	return false
}
