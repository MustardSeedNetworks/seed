// Package version provides build-time version information.
package version

// Version information set via ldflags at build time.
// Build with: go build -ldflags "-X github.com/krisarmstrong/netscope/internal/version.Version=v1.0.0"
var (
	// Version is the semantic version (set via ldflags)
	Version = "dev"
	// Commit is the git commit hash (set via ldflags)
	Commit = "unknown"
	// BuildTime is the build timestamp (set via ldflags)
	BuildTime = "unknown"
)

// Info returns all version information.
func Info() map[string]string {
	return map[string]string{
		"version":   Version,
		"commit":    Commit,
		"buildTime": BuildTime,
	}
}
