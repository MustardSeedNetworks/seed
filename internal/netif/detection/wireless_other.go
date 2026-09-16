//go:build !darwin

package detection

// wirelessInterfaces preserves name-based detection on non-macOS platforms.
func wirelessInterfaces() []string { return nil }
