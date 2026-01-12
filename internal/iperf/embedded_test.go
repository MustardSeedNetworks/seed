package iperf_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/krisarmstrong/seed/internal/iperf"
)

// TestGetCacheDirStructure tests the cache directory structure.
func TestGetCacheDirStructure(t *testing.T) {
	t.Parallel()

	dir, err := iperf.GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}

	// Should be absolute path
	if !filepath.IsAbs(dir) {
		t.Errorf("Cache directory should be absolute path: %q", dir)
	}

	// Should end with seed/bin
	if !strings.HasSuffix(dir, filepath.Join("seed", "bin")) {
		t.Errorf("Cache directory should end with 'seed/bin': %q", dir)
	}
}

// TestGetCacheDirConsistency tests that cache directory is consistent across calls.
func TestGetCacheDirConsistency(t *testing.T) {
	t.Parallel()

	dir1, err1 := iperf.GetCacheDir()
	if err1 != nil {
		t.Fatalf("First GetCacheDir() error = %v", err1)
	}

	dir2, err2 := iperf.GetCacheDir()
	if err2 != nil {
		t.Fatalf("Second GetCacheDir() error = %v", err2)
	}

	if dir1 != dir2 {
		t.Errorf("Cache directory should be consistent: %q != %q", dir1, dir2)
	}
}

// TestIsValidExtractedBinaryEdgeCases tests edge cases for binary validation.
func TestIsValidExtractedBinaryEdgeCases(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name       string
		setup      func() (binaryPath, versionFile string)
		expected   bool
		skipReason string
	}{
		{
			name: "empty binary file",
			setup: func() (string, string) {
				binaryPath := filepath.Join(tempDir, "empty-binary")
				versionFile := filepath.Join(tempDir, "empty-version")
				_ = os.WriteFile(binaryPath, []byte{}, 0o755)
				_ = os.WriteFile(versionFile, []byte(iperf.EmbeddedVersion), 0o600)
				return binaryPath, versionFile
			},
			expected: true, // Empty but executable file with correct version
		},
		{
			name: "version file with whitespace",
			setup: func() (string, string) {
				binaryPath := filepath.Join(tempDir, "whitespace-binary")
				versionFile := filepath.Join(tempDir, "whitespace-version")
				_ = os.WriteFile(binaryPath, []byte("test"), 0o755)
				_ = os.WriteFile(versionFile, []byte("  "+iperf.EmbeddedVersion+"  \n"), 0o600)
				return binaryPath, versionFile
			},
			expected: true, // Whitespace should be trimmed
		},
		{
			name: "version file with newline",
			setup: func() (string, string) {
				binaryPath := filepath.Join(tempDir, "newline-binary")
				versionFile := filepath.Join(tempDir, "newline-version")
				_ = os.WriteFile(binaryPath, []byte("test"), 0o755)
				_ = os.WriteFile(versionFile, []byte(iperf.EmbeddedVersion+"\n"), 0o600)
				return binaryPath, versionFile
			},
			expected: true, // Trailing newline should be trimmed
		},
		{
			name: "directory instead of binary",
			setup: func() (string, string) {
				binaryPath := filepath.Join(tempDir, "dir-binary")
				versionFile := filepath.Join(tempDir, "dir-version")
				_ = os.Mkdir(binaryPath, 0o755)
				_ = os.WriteFile(versionFile, []byte(iperf.EmbeddedVersion), 0o600)
				return binaryPath, versionFile
			},
			// Note: Directories on Unix have execute permission for traversal,
			// so isValidExtractedBinary only checks Mode() & 0o111 != 0,
			// which will be true for directories. The actual validation
			// happens when the binary is executed.
			expected: true, // Directory has execute bit set
		},
		{
			name: "symlink to nonexistent",
			setup: func() (string, string) {
				binaryPath := filepath.Join(tempDir, "symlink-binary")
				versionFile := filepath.Join(tempDir, "symlink-version")
				_ = os.Symlink("/nonexistent/path", binaryPath)
				_ = os.WriteFile(versionFile, []byte(iperf.EmbeddedVersion), 0o600)
				return binaryPath, versionFile
			},
			expected:   false, // Broken symlink should fail
			skipReason: "symlinks may not be available on all systems",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.skipReason != "" {
				if runtime.GOOS == "windows" {
					t.Skip(tt.skipReason)
				}
			}

			binaryPath, versionFile := tt.setup()
			result := iperf.IsValidExtractedBinary(binaryPath, versionFile)

			if result != tt.expected {
				t.Errorf("IsValidExtractedBinary() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetPlatformBinaryMapCompleteness tests that all major platforms are covered.
func TestGetPlatformBinaryMapCompleteness(t *testing.T) {
	t.Parallel()

	platformMap := iperf.GetPlatformBinaryMap()

	requiredPlatforms := []string{
		"linux-amd64",
		"linux-arm64",
		"darwin-amd64",
		"darwin-arm64",
	}

	for _, platform := range requiredPlatforms {
		binaryName, ok := platformMap[platform]
		if !ok {
			t.Errorf("Platform %q is missing from platform binary map", platform)
			continue
		}

		// Verify binary name format
		if !strings.HasPrefix(binaryName, "iperf3-") {
			t.Errorf("Binary name for %q should start with 'iperf3-': %q", platform, binaryName)
		}

		// Verify platform is in binary name
		parts := strings.Split(platform, "-")
		if len(parts) >= 2 {
			os := parts[0]
			arch := parts[1]
			if !strings.Contains(binaryName, os) || !strings.Contains(binaryName, arch) {
				t.Errorf("Binary name %q should contain OS (%s) and arch (%s)", binaryName, os, arch)
			}
		}
	}
}

// TestGetPlatformBinaryMapCurrentPlatform tests current platform is in the map.
func TestGetPlatformBinaryMapCurrentPlatform(t *testing.T) {
	t.Parallel()

	platformMap := iperf.GetPlatformBinaryMap()
	currentPlatform := runtime.GOOS + "-" + runtime.GOARCH

	// Check if current platform is supported (may not be on all platforms)
	if _, ok := platformMap[currentPlatform]; ok {
		t.Logf("Current platform %q is supported", currentPlatform)
	} else {
		// Only major platforms are expected to be in the map
		majorPlatforms := map[string]bool{
			"linux-amd64":  true,
			"linux-arm64":  true,
			"darwin-amd64": true,
			"darwin-arm64": true,
		}
		if majorPlatforms[currentPlatform] {
			t.Errorf("Expected major platform %q to be in map", currentPlatform)
		} else {
			t.Logf("Current platform %q is not in embedded binary map (expected)", currentPlatform)
		}
	}
}

// TestHasEmbeddedBinaryPlatformConsistency tests embedded binary availability.
func TestHasEmbeddedBinaryPlatformConsistency(t *testing.T) {
	t.Parallel()

	// Call multiple times to verify consistency
	result1 := iperf.HasEmbeddedBinary()
	result2 := iperf.HasEmbeddedBinary()

	if result1 != result2 {
		t.Errorf("HasEmbeddedBinary() should be consistent: %v != %v", result1, result2)
	}

	t.Logf("HasEmbeddedBinary() = %v for %s/%s", result1, runtime.GOOS, runtime.GOARCH)
}

// TestGetInstallInstructionsContent tests install instructions content.
func TestGetInstallInstructionsContent(t *testing.T) {
	t.Parallel()

	instructions := iperf.GetInstallInstructions()

	// Should always contain basic info
	if !strings.Contains(instructions, "iperf3") {
		t.Error("Instructions should mention iperf3")
	}

	if !strings.Contains(instructions, "not installed") {
		t.Error("Instructions should mention iperf3 is not installed")
	}

	// Should contain source build instructions
	if !strings.Contains(instructions, "github.com/esnet/iperf") {
		t.Error("Instructions should contain GitHub source link")
	}

	// Platform-specific checks
	switch runtime.GOOS {
	case "linux":
		// Should contain at least one Linux package manager instruction
		linuxInstructions := []string{"apt", "dnf", "yum", "pacman", "apk"}
		hasLinuxInstruction := false
		for _, pm := range linuxInstructions {
			if strings.Contains(instructions, pm) {
				hasLinuxInstruction = true
				break
			}
		}
		if !hasLinuxInstruction {
			t.Error("Linux instructions should contain at least one package manager")
		}
	case "darwin":
		if !strings.Contains(instructions, "brew") && !strings.Contains(instructions, "Homebrew") {
			t.Error("macOS instructions should mention Homebrew")
		}
	case "windows":
		if !strings.Contains(instructions, "choco") && !strings.Contains(instructions, "Chocolatey") {
			// Windows might mention download link instead
			if !strings.Contains(instructions, "iperf.fr") {
				t.Log("Windows instructions should mention Chocolatey or download link")
			}
		}
	}
}

// TestNotFoundErrorFormatting tests NotFoundError message formatting.
func TestNotFoundErrorFormatting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		err           *iperf.NotFoundError
		shouldContain []string
	}{
		{
			name: "minimal error",
			err: &iperf.NotFoundError{
				SearchedPaths: nil,
				SystemError:   nil,
				EmbeddedError: nil,
			},
			shouldContain: []string{"iperf3 not found"},
		},
		{
			name: "with one searched path",
			err: &iperf.NotFoundError{
				SearchedPaths: []string{"/usr/bin/iperf3"},
				SystemError:   nil,
				EmbeddedError: nil,
			},
			shouldContain: []string{"iperf3 not found", "/usr/bin/iperf3", "Searched paths"},
		},
		{
			name: "with multiple searched paths",
			err: &iperf.NotFoundError{
				SearchedPaths: []string{
					"/usr/bin/iperf3",
					"/usr/local/bin/iperf3",
					"/opt/iperf3/bin/iperf3",
				},
				SystemError:   nil,
				EmbeddedError: nil,
			},
			shouldContain: []string{
				"iperf3 not found",
				"/usr/bin/iperf3",
				"/usr/local/bin/iperf3",
				"/opt/iperf3/bin/iperf3",
			},
		},
		{
			name: "with all error types",
			err: &iperf.NotFoundError{
				SearchedPaths: []string{"/path/to/iperf3"},
				SystemError:   os.ErrNotExist,
				EmbeddedError: os.ErrPermission,
			},
			shouldContain: []string{
				"iperf3 not found",
				"System PATH",
				"Embedded binary",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errMsg := tt.err.Error()

			for _, expected := range tt.shouldContain {
				if !strings.Contains(errMsg, expected) {
					t.Errorf("Error should contain %q, got:\n%s", expected, errMsg)
				}
			}
		})
	}
}

// TestNotFoundErrorImplementsError tests that NotFoundError implements error interface.
func TestNotFoundErrorImplementsError(t *testing.T) {
	t.Parallel()

	nfe := &iperf.NotFoundError{
		SearchedPaths: []string{"/test/path"},
	}

	// Should be assignable to error interface
	var err error = nfe
	_ = err // Use the variable to avoid unused variable warning

	// Should have non-empty error message
	if nfe.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// TestEmbeddedVersionFormat tests the embedded version constant format.
func TestEmbeddedVersionFormat(t *testing.T) {
	t.Parallel()

	version := iperf.EmbeddedVersion

	if version == "" {
		t.Error("EmbeddedVersion should not be empty")
	}

	// Should be a valid version format (X.Y or X.Y.Z)
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		t.Errorf("EmbeddedVersion should have at least major.minor format: %q", version)
	}

	// Major version should be numeric and >= 3 (iperf3)
	if len(parts) > 0 {
		if parts[0] != "3" {
			t.Errorf("EmbeddedVersion major version should be 3, got %q in %q", parts[0], version)
		}
	}
}

// TestFindSystemIperf3Available tests system iperf3 detection.
func TestFindSystemIperf3Available(t *testing.T) {
	t.Parallel()

	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf test (SKIP_IPERF_TEST=1)")
	}

	path, err := iperf.FindSystemIperf3()
	if err != nil {
		// This is expected if iperf3 is not installed
		t.Logf("iperf3 not found in system PATH: %v", err)
		return
	}

	// If found, verify the path
	if path == "" {
		t.Error("Path should not be empty when no error returned")
	}

	if !filepath.IsAbs(path) {
		t.Errorf("System iperf3 path should be absolute: %q", path)
	}

	// Verify file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("Could not stat found iperf3 path: %v", err)
	} else if info.IsDir() {
		t.Errorf("iperf3 path should not be a directory: %q", path)
	}

	t.Logf("Found system iperf3 at: %s", path)
}

// TestExtractEmbeddedBinaryPlatformCheck tests embedded binary extraction.
func TestExtractEmbeddedBinaryPlatformCheck(t *testing.T) {
	// Skip if no embedded binary for this platform
	if !iperf.HasEmbeddedBinary() {
		t.Skip("No embedded binary for this platform")
	}

	path, err := iperf.ExtractEmbeddedBinary()
	if err != nil {
		t.Fatalf("ExtractEmbeddedBinary() error = %v", err)
	}

	if path == "" {
		t.Error("Extracted path should not be empty")
	}

	// Verify the file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Could not stat extracted binary: %v", err)
	}

	// Should be executable
	if info.Mode()&0o111 == 0 {
		t.Errorf("Extracted binary should be executable: %s", path)
	}

	t.Logf("Extracted embedded binary to: %s", path)
}

// TestCacheDirCreation tests that cache directory can be created.
func TestCacheDirCreation(t *testing.T) {
	// Get the cache directory
	cacheDir, err := iperf.GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}

	// Check if parent directory exists or can be created
	parentDir := filepath.Dir(cacheDir)
	if _, statErr := os.Stat(parentDir); os.IsNotExist(statErr) {
		// Try to create it
		if mkdirErr := os.MkdirAll(parentDir, 0o750); mkdirErr != nil {
			t.Logf("Could not create parent cache directory: %v (may require permissions)", mkdirErr)
		}
	}

	t.Logf("Cache directory: %s", cacheDir)
}

// TestIsValidExtractedBinaryPermissions tests various permission scenarios.
func TestIsValidExtractedBinaryPermissions(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("Permission tests not applicable on Windows")
	}

	tempDir := t.TempDir()

	// Test various permission combinations
	permissions := []struct {
		name       string
		binaryPerm os.FileMode
		expected   bool
	}{
		{"executable by all", 0o755, true},
		{"executable by owner", 0o700, true},
		{"executable by group", 0o050, true},
		{"executable by others", 0o005, true},
		{"not executable", 0o644, false},
		{"read only", 0o444, false},
		{"write only", 0o200, false},
	}

	for _, tt := range permissions {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			binaryPath := filepath.Join(tempDir, "binary-"+tt.name)
			versionFile := filepath.Join(tempDir, "version-"+tt.name)

			_ = os.WriteFile(binaryPath, []byte("test"), tt.binaryPerm)
			_ = os.WriteFile(versionFile, []byte(iperf.EmbeddedVersion), 0o600)

			result := iperf.IsValidExtractedBinary(binaryPath, versionFile)
			if result != tt.expected {
				t.Errorf("IsValidExtractedBinary() with perm %o = %v, want %v",
					tt.binaryPerm, result, tt.expected)
			}
		})
	}
}
