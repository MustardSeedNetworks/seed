// Package version provides build-time version information.
// Uses runtime/debug.ReadBuildInfo() to extract VCS and module information
// embedded by the Go toolchain during build.
package version

import (
	"runtime/debug"
)

const (
	// shortCommitLen is the number of characters to use for shortened commit hashes.
	shortCommitLen = 7
	// defaultVersion is the version string when no build info is available.
	defaultVersion = "dev"
	// unknownValue is used for commit and build time when not available.
	unknownValue = "unknown"
)

// extractVersionFromBuildInfo processes a debug.BuildInfo and extracts version information.
// This function is separated from getVersionInfo to enable testing with mock build info.
func extractVersionFromBuildInfo(info *debug.BuildInfo) (string, string, string) {
	ver := defaultVersion
	commit := unknownValue
	buildTime := unknownValue

	if info == nil {
		return ver, commit, buildTime
	}

	// Get module version.
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		ver = info.Main.Version
	}

	// Extract VCS settings.
	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			commit = setting.Value
			// Shorten for display.
			if len(commit) > shortCommitLen {
				commit = commit[:shortCommitLen]
			}
		case "vcs.time":
			buildTime = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if modified && ver != defaultVersion {
		ver += "-dirty"
	}

	return ver, commit, buildTime
}

// getVersionInfo extracts version, commit, and build time from build info.
// The Go runtime caches ReadBuildInfo() results, so no additional caching is needed.
func getVersionInfo() (string, string, string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return defaultVersion, unknownValue, unknownValue
	}
	return extractVersionFromBuildInfo(info)
}

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
