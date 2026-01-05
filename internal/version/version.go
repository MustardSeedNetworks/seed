// Package version provides build-time version information.
// Uses runtime/debug.ReadBuildInfo() to extract VCS and module information
// embedded by the Go toolchain during build.
package version

import (
	"runtime/debug"
	"sync"
)

// shortCommitLen is the number of characters to use for shortened commit hashes.
const shortCommitLen = 7

// Version accessor functions use closure-encapsulated state to satisfy gochecknoglobals.
// getVersionInfo returns (version, commit, buildTime) from build info.
// _ (initVersionInfo) ensures version info is loaded (unused but required for pattern).
var (
	getVersionInfo, _ = func() (
		func() (version, commit, buildTime string),
		func(),
	) {
		var once sync.Once
		var ver, com, bt string

		loadVersionInfo := func() {
			once.Do(func() {
				ver = "dev"
				com = "unknown"
				bt = "unknown"

				info, ok := debug.ReadBuildInfo()
				if !ok {
					return
				}

				// Get module version
				if info.Main.Version != "" && info.Main.Version != "(devel)" {
					ver = info.Main.Version
				}

				// Extract VCS settings
				for _, setting := range info.Settings {
					switch setting.Key {
					case "vcs.revision":
						com = setting.Value
						// Shorten for display
						if len(com) > shortCommitLen {
							com = com[:shortCommitLen]
						}
					case "vcs.time":
						bt = setting.Value
					case "vcs.modified":
						if setting.Value == "true" && ver != "dev" {
							ver += "-dirty"
						}
					}
				}
			})
		}

		return func() (string, string, string) {
			loadVersionInfo()
			return ver, com, bt
		}, loadVersionInfo
	}()
)

// GetVersion returns the semantic version from build info.
func GetVersion() string {
	ver, _, _ := getVersionInfo()
	return ver
}

// GetCommit returns the git commit hash from build info.
func GetCommit() string {
	_, commit, _ := getVersionInfo()
	return commit
}

// GetBuildTime returns the build timestamp from build info.
func GetBuildTime() string {
	_, _, buildTime := getVersionInfo()
	return buildTime
}

// Info returns all version information.
func Info() map[string]string {
	ver, commit, buildTime := getVersionInfo()
	return map[string]string{
		"version":   ver,
		"commit":    commit,
		"buildTime": buildTime,
	}
}
