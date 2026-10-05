package iperf

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// findIperf3Binary returns the path of the resolved iperf3 binary.
func findIperf3Binary() (string, error) {
	b, err := resolveIperf3()
	return b.path, err
}

// resolveIperf3 locates the iperf3 binary using a robust detection strategy,
// probing each candidate once and caching the first that answers:
//  1. Try embedded binary (extracted to user cache directory).
//  2. Search system PATH using [exec.LookPath] (the proper way).
//  3. Return detailed error with OS-specific install instructions if not found.
func resolveIperf3() (iperfBinary, error) {
	if cached := getIperfBinary(); cached.path != "" {
		return cached, nil
	}

	searchedPaths := make([]string, 0, searchPathsPrealloc) // Preallocate for typical search paths
	var embeddedErr, systemErr error

	// Strategy 1: Try embedded binary (extract to cache if needed)
	if HasEmbeddedBinary() {
		if path, err := extractEmbeddedBinary(); err == nil {
			if version, ok := probeVersion(path); ok {
				b := iperfBinary{path: path, version: version}
				setIperfBinary(b)
				logging.GetLogger().
					Info("Using embedded iperf3 binary", "path", path, "version", EmbeddedVersion)
				return b, nil
			}
			logging.GetLogger().Warn("Extracted iperf3 binary failed validation", "path", path)
		} else {
			embeddedErr = err
			logging.GetLogger().Debug("Failed to extract embedded iperf3", "error", err)
		}
	}

	// Strategy 2: Search system PATH using exec.LookPath
	// This is the proper way to find executables - it searches the entire PATH
	if path, err := findSystemIperf3(); err == nil {
		if version, ok := probeVersion(path); ok {
			b := iperfBinary{path: path, version: version}
			setIperfBinary(b)
			logging.GetLogger().Info("Using system iperf3 binary", "path", path)
			return b, nil
		}
		logging.GetLogger().Warn("System iperf3 binary failed validation", "path", path)
	} else {
		systemErr = err
	}

	// Strategy 3: Check legacy paths for backwards compatibility
	// These are less reliable but may help in edge cases
	legacyPaths := getLegacyPaths()
	for _, path := range legacyPaths {
		searchedPaths = append(searchedPaths, path)
		if info, err := os.Stat(path); err == nil && info.Mode()&0o111 != 0 {
			if version, ok := probeVersion(path); ok {
				b := iperfBinary{path: path, version: version}
				setIperfBinary(b)
				logging.GetLogger().Info("Using iperf3 from legacy path", "path", path)
				return b, nil
			}
		}
	}

	// Not found - return detailed error with install instructions.
	return iperfBinary{}, &NotFoundError{
		SearchedPaths: searchedPaths,
		SystemError:   systemErr,
		EmbeddedError: embeddedErr,
	}
}

// getLegacyPaths returns platform-specific paths to check as a last resort.
func getLegacyPaths() []string {
	var paths []string

	// Paths relative to executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		paths = append(paths,
			filepath.Join(execDir, "bin", "iperf3"),
			filepath.Join(execDir, "iperf3"),
		)
	}

	// Paths relative to working directory (for development)
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths,
			filepath.Join(cwd, "bin", "iperf3"),
		)
	}

	return paths
}

// probeVersion runs the binary's --version and reports whether it answered as
// iperf3, with the version it named.
func probeVersion(path string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil || !strings.Contains(string(out), "iperf") {
		return "", false
	}

	return parseVersion(string(out)), true
}

// parseVersion extracts the version from --version output such as
// "iperf 3.16 (cJSON 1.7.17)\n...", returning "v3.16".
func parseVersion(out string) string {
	line, _, _ := strings.Cut(out, "\n")
	line = strings.TrimSpace(line)
	if parts := strings.Fields(line); len(parts) >= minVersionParts {
		return "v" + parts[1]
	}
	return line
}

// CheckInstalled checks if iperf3 is available.
func CheckInstalled() error {
	_, err := resolveIperf3()
	return err
}

// GetVersion returns the installed iperf3 version, read once when the binary
// was resolved.
func GetVersion() (string, error) {
	b, err := resolveIperf3()
	if err != nil {
		return "", err
	}
	return b.version, nil
}

// ValidateVersion checks if the installed iperf3 version meets minimum requirements.
func ValidateVersion() error {
	version, err := GetVersion()
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Remove 'v' prefix for comparison
	version = strings.TrimPrefix(version, "v")

	// Compare version strings (simple lexicographic comparison works for x.y format)
	if compareVersions(version, minSupportedVersion) < 0 {
		return fmt.Errorf(
			"iperf3 version %s is below minimum supported version %s",
			version,
			minSupportedVersion,
		)
	}

	return nil
}

// compareVersions compares two version strings in format "x.y" or "x.y.z"
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2.
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	for i := 0; i < len(parts1) || i < len(parts2); i++ {
		var n1, n2 int

		if i < len(parts1) {
			_, _ = fmt.Sscanf(parts1[i], "%d", &n1)
		}
		if i < len(parts2) {
			_, _ = fmt.Sscanf(parts2[i], "%d", &n2)
		}

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	return 0
}

// waitForPortReady checks if a TCP port is ready to accept connections.
func waitForPortReady(port int, timeout time.Duration) error {
	// Validate port number
	if err := validation.ValidatePort(port); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	dialer := &net.Dialer{Timeout: portCheckIntervalMs * time.Millisecond}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("port %d not ready after %v", port, timeout)
		default:
			conn, err := dialer.DialContext(ctx, "tcp", addr)
			if err == nil {
				_ = conn.Close()
				return nil
			}
			time.Sleep(portCheckIntervalMs * time.Millisecond)
		}
	}
}
