// Package paths provides XDG Base Directory compliant path resolution for Seed.
//
// The package implements the XDG Base Directory Specification for portable,
// standards-compliant configuration and data storage across different installation
// modes (user vs system).
//
// # Installation Modes
//
// The package supports three installation modes:
//
//   - ModeAuto: Automatically detects whether running as root/systemd (system mode)
//     or regular user (user mode)
//   - ModeUser: Forces user-level paths (~/.config, ~/.local/share, etc.)
//   - ModeSystem: Forces system-level paths (/etc, /var/lib, /var/log, etc.)
//
// # Path Resolution
//
// User mode paths (non-root):
//   - Config: $XDG_CONFIG_HOME/seed/ (default: ~/.config/seed/)
//   - Data: $XDG_DATA_HOME/seed/ (default: ~/.local/share/seed/)
//   - Logs: $XDG_STATE_HOME/seed/logs/ (default: ~/.local/state/seed/logs/)
//   - Cache: $XDG_CACHE_HOME/seed/ (default: ~/.cache/seed/)
//
// System mode paths (root/systemd):
//   - Config: /etc/seed/
//   - Data: /var/lib/seed/
//   - Logs: /var/log/seed/
//   - Cache: /var/cache/seed/
//   - Binary: /usr/local/bin/
//
// # Config File Priority
//
// ResolveConfigPath determines the config file location with this priority:
//
//  1. Explicit path argument (if non-empty and not the default "config.yaml")
//  2. SEED_CONFIG_PATH environment variable
//  3. XDG-compliant path based on detected mode
//
// # Legacy Support
//
// DetectLegacyConfig checks for configuration files in legacy locations
// (current directory) and returns the path if found, allowing smooth
// migration from old installations.
//
// # Example Usage
//
//	// Auto-detect mode and resolve paths
//	paths := paths.Resolve(paths.ModeAuto)
//	fmt.Printf("Config: %s\n", paths.ConfigFile)
//	fmt.Printf("Logs: %s\n", paths.LogDir)
//
//	// Resolve config with priority handling
//	configPath := paths.ResolveConfigPath(cliFlag, paths.ModeAuto)
//
//	// Check for legacy config
//	if legacy, ok := paths.DetectLegacyConfig(); ok {
//	    fmt.Printf("Found legacy config: %s\n", legacy)
//	}
package paths
