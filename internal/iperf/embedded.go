// Package iperf provides embedded iperf3 binary management.
//
// This file handles extracting embedded iperf3 binaries and provides
// robust detection with OS-specific installation guidance.
package iperf

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/krisarmstrong/seed/internal/logging"
)

// embeddedBinaries contains pre-built iperf3 binaries for supported platforms.
// Build these using: make build-iperf3-all
//
//go:embed binaries/*
var embeddedBinaries embed.FS

// platformBinaryMap maps GOOS-GOARCH to embedded binary filenames.
var platformBinaryMap = map[string]string{
	"linux-amd64":  "iperf3-linux-amd64",
	"linux-arm64":  "iperf3-linux-arm64",
	"darwin-amd64": "iperf3-darwin-amd64",
	"darwin-arm64": "iperf3-darwin-arm64",
}

// EmbeddedVersion is the version of the embedded iperf3 binaries.
// Update this when rebuilding embedded binaries.
const EmbeddedVersion = "3.20"

// getCacheDir returns the OS-appropriate cache directory for extracted binaries.
// Linux: ~/.cache/seed/bin.
// macOS: ~/Library/Caches/seed/bin.
// Windows: %LocalAppData%\seed\bin.
func getCacheDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		// Fallback to home directory
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("cannot determine cache directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "seed", "bin"), nil
}

// extractEmbeddedBinary extracts the platform-specific iperf3 binary to cache.
// Returns the path to the extracted binary or an error.
func extractEmbeddedBinary() (string, error) {
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	binaryName, ok := platformBinaryMap[platform]
	if !ok {
		return "", fmt.Errorf("no embedded iperf3 binary for platform %s", platform)
	}

	cacheDir, err := getCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache directory: %w", err)
	}

	destPath := filepath.Join(cacheDir, "iperf3")
	versionFile := filepath.Join(cacheDir, ".iperf3-version")

	// Check if already extracted with correct version
	if isValidExtractedBinary(destPath, versionFile) {
		logging.GetLogger().Debug("Using cached iperf3 binary", "path", destPath, "version", EmbeddedVersion)
		return destPath, nil
	}

	// Read embedded binary
	data, err := embeddedBinaries.ReadFile("binaries/" + binaryName)
	if err != nil {
		return "", fmt.Errorf("embedded binary %s not found: %w", binaryName, err)
	}

	// Create cache directory.
	if mkdirErr := os.MkdirAll(cacheDir, 0o750); mkdirErr != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", mkdirErr)
	}

	// Write binary.
	if writeErr := os.WriteFile(destPath, data, 0o750); writeErr != nil { //nolint:gosec // G306: binary needs execute permission
		return "", fmt.Errorf("failed to extract iperf3 binary: %w", writeErr)
	}

	// Write version marker.
	if versionErr := os.WriteFile(versionFile, []byte(EmbeddedVersion), 0o600); versionErr != nil {
		logging.GetLogger().Warn("Failed to write version marker", "error", versionErr)
	}

	logging.GetLogger().Info("Extracted embedded iperf3 binary", "path", destPath, "version", EmbeddedVersion)
	return destPath, nil
}

// isValidExtractedBinary checks if the extracted binary exists and matches expected version.
func isValidExtractedBinary(binaryPath, versionFile string) bool {
	// Check binary exists and is executable
	info, err := os.Stat(binaryPath)
	if err != nil || info.Mode()&0o111 == 0 {
		return false
	}

	// Check version marker.
	versionData, err := os.ReadFile(versionFile)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(versionData)) == EmbeddedVersion
}

// findSystemIperf3 searches for iperf3 in the system PATH.
// This is the proper way to find executables - it searches the entire PATH.
func findSystemIperf3() (string, error) {
	path, err := exec.LookPath("iperf3")
	if err != nil {
		return "", errors.New("iperf3 not found in system PATH")
	}
	return path, nil
}

// GetInstallInstructions returns OS-specific instructions for installing iperf3.
func GetInstallInstructions() string {
	var instructions strings.Builder
	instructions.WriteString("iperf3 is not installed. Install it using:\n\n")

	switch runtime.GOOS {
	case osLinux:
		// Detect available package managers.
		if _, err := exec.LookPath("apt"); err == nil {
			instructions.WriteString("  Ubuntu/Debian:\n")
			instructions.WriteString("    sudo apt update && sudo apt install -y iperf3\n\n")
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			instructions.WriteString("  Fedora/RHEL:\n")
			instructions.WriteString("    sudo dnf install -y iperf3\n\n")
		}
		if _, err := exec.LookPath("yum"); err == nil {
			instructions.WriteString("  CentOS/RHEL (older):\n")
			instructions.WriteString("    sudo yum install -y iperf3\n\n")
		}
		if _, err := exec.LookPath("pacman"); err == nil {
			instructions.WriteString("  Arch Linux:\n")
			instructions.WriteString("    sudo pacman -S iperf3\n\n")
		}
		if _, err := exec.LookPath("apk"); err == nil {
			instructions.WriteString("  Alpine Linux:\n")
			instructions.WriteString("    sudo apk add iperf3\n\n")
		}

	case osDarwin:
		instructions.WriteString("  macOS (Homebrew):\n")
		instructions.WriteString("    brew install iperf3\n\n")
		instructions.WriteString("  macOS (MacPorts):\n")
		instructions.WriteString("    sudo port install iperf3\n\n")

	case osWindows:
		instructions.WriteString("  Windows (Chocolatey):\n")
		instructions.WriteString("    choco install iperf3\n\n")
		instructions.WriteString("  Windows (Manual):\n")
		instructions.WriteString("    Download from: https://iperf.fr/iperf-download.php\n\n")

	default:
		instructions.WriteString("  Build from source:\n")
		instructions.WriteString("    https://github.com/esnet/iperf\n\n")
	}

	instructions.WriteString("  Or build from source (any platform):\n")
	instructions.WriteString("    git clone https://github.com/esnet/iperf.git\n")
	instructions.WriteString("    cd iperf && ./configure && make && sudo make install\n")

	return instructions.String()
}

// NotFoundError provides detailed error information when iperf3 is not found.
type NotFoundError struct {
	SearchedPaths []string
	SystemError   error
	EmbeddedError error
}

func (e *NotFoundError) Error() string {
	var msg strings.Builder
	msg.WriteString("iperf3 not found\n\n")

	if e.EmbeddedError != nil {
		msg.WriteString(fmt.Sprintf("Embedded binary: %v\n", e.EmbeddedError))
	}

	if e.SystemError != nil {
		msg.WriteString(fmt.Sprintf("System PATH: %v\n", e.SystemError))
	}

	if len(e.SearchedPaths) > 0 {
		msg.WriteString("\nSearched paths:\n")
		for _, p := range e.SearchedPaths {
			msg.WriteString(fmt.Sprintf("  - %s\n", p))
		}
	}

	msg.WriteString("\n")
	msg.WriteString(GetInstallInstructions())

	return msg.String()
}

// HasEmbeddedBinary returns true if an embedded binary exists for the current platform.
func HasEmbeddedBinary() bool {
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	_, ok := platformBinaryMap[platform]
	if !ok {
		return false
	}

	binaryName := platformBinaryMap[platform]
	_, err := embeddedBinaries.ReadFile("binaries/" + binaryName)
	return err == nil
}
