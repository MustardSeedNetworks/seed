package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	appName        = "seed"
	defaultConfig  = "config.yaml"
	legacyConfig   = "config.yaml"
	legacyConfigV1 = ".seed.yaml"
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
}

// Resolve determines paths based on mode and environment.
//
// For ModeAuto, it detects whether to use system or user paths by checking:
//   - If running as root (UID 0)
//   - If running under systemd (NOTIFY_SOCKET or INVOCATION_ID env vars)
//
// Returns a Paths structure with all resolved directory and file paths.
func Resolve(mode Mode) *Paths {
	// Auto-detect mode if needed
	actualMode := mode
	if mode == ModeAuto {
		if isSystemdService() || os.Getuid() == 0 {
			actualMode = ModeSystem
		} else {
			actualMode = ModeUser
		}
	}

	p := &Paths{Mode: actualMode}

	if actualMode == ModeSystem {
		p.ConfigDir = filepath.Join("/etc", appName)
		p.DataDir = filepath.Join("/var/lib", appName)
		p.LogDir = filepath.Join("/var/log", appName)
		p.CacheDir = filepath.Join("/var/cache", appName)
		p.BinaryDir = "/usr/local/bin"
	} else {
		// User mode - XDG Base Directory compliant
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if home unavailable
			homeDir = "."
		}

		// XDG_CONFIG_HOME
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(homeDir, ".config")
		}
		p.ConfigDir = filepath.Join(configHome, appName)

		// XDG_DATA_HOME
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			dataHome = filepath.Join(homeDir, ".local", "share")
		}
		p.DataDir = filepath.Join(dataHome, appName)

		// XDG_STATE_HOME (for logs)
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			stateHome = filepath.Join(homeDir, ".local", "state")
		}
		p.LogDir = filepath.Join(stateHome, appName, "logs")

		// XDG_CACHE_HOME
		cacheHome := os.Getenv("XDG_CACHE_HOME")
		if cacheHome == "" {
			cacheHome = filepath.Join(homeDir, ".cache")
		}
		p.CacheDir = filepath.Join(cacheHome, appName)

		// User binary dir
		p.BinaryDir = filepath.Join(homeDir, ".local", "bin")
	}

	p.ConfigFile = filepath.Join(p.ConfigDir, defaultConfig)

	return p
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

	// Priority 3: XDG-compliant path
	paths := Resolve(mode)
	return paths.ConfigFile
}

// DetectLegacyConfig checks for configs in legacy locations.
//
// It looks for config files in the current working directory:
//   - config.yaml (legacy default)
//   - .seed.yaml (v1 config)
//
// Returns the path and true if found, empty string and false otherwise.
func DetectLegacyConfig() (string, bool) {
	// Check current directory for legacy configs
	legacyPaths := []string{
		legacyConfig,
		legacyConfigV1,
	}

	for _, path := range legacyPaths {
		if _, statErr := os.Stat(path); statErr == nil {
			abs, absErr := filepath.Abs(path)
			if absErr == nil {
				return abs, true
			}
			return path, true
		}
	}

	return "", false
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
