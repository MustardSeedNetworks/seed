// Package version provides build-time version information.
// Uses runtime/[debug.ReadBuildInfo]() to extract VCS and module information.
// embedded by the Go toolchain during build, with ldflags override support.
package version

import (
	"runtime/debug"
)

// These variables are set via ldflags at build time.
// Example: -ldflags "-X github.com/krisarmstrong/seed/internal/version.Version=v1.0.0".
//
//nolint:gochecknoglobals // Build metadata injected via ldflags.
var (
	Version     string // Set via -ldflags
	Commit      string // Set via -ldflags
	BuildTime   string // Set via -ldflags
	UIBuildHash string // Set via -ldflags; md5 of all files under ui/dist
)

const (
	// shortCommitLen is the number of characters to use for shortened commit hashes.
	shortCommitLen = 7
	// defaultVersion is the version string when no build info is available.
	defaultVersion = "dev"
	// unknownValue is used for commit and build time when not available.
	unknownValue = "unknown"
)

// extractVersionFromBuildInfo processes a [debug.BuildInfo] and extracts version information.
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
// Ldflags variables take precedence over [debug.ReadBuildInfo]().
func getVersionInfo() (string, string, string) {
	// Use ldflags values if set.
	if Version != "" {
		ver := Version
		commit := Commit
		buildTime := BuildTime
		if commit == "" {
			commit = unknownValue
		}
		if buildTime == "" {
			buildTime = unknownValue
		}
		return ver, commit, buildTime
	}

	// Fall back to debug.ReadBuildInfo.
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

// GetUIBuildHash returns the md5 hash of the embedded UI assets.
// Returns unknownValue when not injected via -ldflags at build time.
func GetUIBuildHash() string {
	if UIBuildHash == "" {
		return unknownValue
	}
	return UIBuildHash
}

// Info returns all version information.
func Info() map[string]string {
	ver, commit, buildTime := getVersionInfo()
	return map[string]string{
		"version":     ver,
		"commit":      commit,
		"buildTime":   buildTime,
		"uiBuildHash": GetUIBuildHash(),
	}
}
