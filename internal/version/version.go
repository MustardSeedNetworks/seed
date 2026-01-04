// Package version provides build-time version information.
package version

// These variables are set via ldflags at build time.
// Build with: go build -ldflags "-X github.com/krisarmstrong/seed/internal/version.version=v1.0.0".
// Note: ldflags requires package-level variables; this is a Go language constraint.
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

// GetVersion returns the semantic version (set via ldflags).
func GetVersion() string {
	return version
}

// GetCommit returns the git commit hash (set via ldflags).
func GetCommit() string {
	return commit
}

// GetBuildTime returns the build timestamp (set via ldflags).
func GetBuildTime() string {
	return buildTime
}

// Info returns all version information.
func Info() map[string]string {
	return map[string]string{
		"version":   version,
		"commit":    commit,
		"buildTime": buildTime,
	}
}
