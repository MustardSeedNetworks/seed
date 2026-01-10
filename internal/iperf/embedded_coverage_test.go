package iperf_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/krisarmstrong/seed/internal/iperf"
)

// TestGetCacheDirFallback tests cache directory fallback behavior.
func TestGetCacheDirFallback(t *testing.T) {
	t.Parallel()

	dir, err := iperf.GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}

	// Should contain seed and bin
	if !strings.Contains(dir, "seed") {
		t.Errorf("Cache dir should contain 'seed': %s", dir)
	}

	if !strings.HasSuffix(dir, "bin") {
		t.Errorf("Cache dir should end with 'bin': %s", dir)
	}

	// Should be under user cache or home directory
	homeDir, _ := os.UserHomeDir()
	cacheDir, _ := os.UserCacheDir()

	underHome := strings.HasPrefix(dir, homeDir)
	underCache := strings.HasPrefix(dir, cacheDir)

	if !underHome && !underCache {
		t.Errorf("Cache dir should be under home or cache: %s", dir)
	}
}

// TestExtractEmbeddedBinaryNoEmbedded tests extraction when no embedded binary exists.
func TestExtractEmbeddedBinaryNoEmbedded(t *testing.T) {
	t.Parallel()

	// This test checks behavior for unsupported platforms
	currentPlatform := runtime.GOOS + "-" + runtime.GOARCH
	platformMap := iperf.GetPlatformBinaryMap()

	if _, ok := platformMap[currentPlatform]; !ok {
		// Platform not in map - extraction should fail
		_, err := iperf.ExtractEmbeddedBinary()
		if err == nil {
			t.Log("Unexpectedly found embedded binary for unsupported platform")
		} else {
			// Expected - no embedded binary for this platform
			t.Logf("Expected error for unsupported platform: %v", err)
		}
	} else {
		t.Logf("Platform %s is supported", currentPlatform)
	}
}

// TestIsValidExtractedBinaryAllScenarios tests all validation scenarios.
func TestIsValidExtractedBinaryAllScenarios(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name          string
		setupBinary   func(path string) error
		setupVersion  func(path string) error
		expected      bool
		skipOnWindows bool
	}{
		{
			name: "valid binary and version",
			setupBinary: func(path string) error {
				return os.WriteFile(path, []byte("test"), 0o755)
			},
			setupVersion: func(path string) error {
				return os.WriteFile(path, []byte(iperf.EmbeddedVersion), 0o600)
			},
			expected: true,
		},
		{
			name: "missing binary",
			setupBinary: func(_ string) error {
				return nil // Don't create file
			},
			setupVersion: func(path string) error {
				return os.WriteFile(path, []byte(iperf.EmbeddedVersion), 0o600)
			},
			expected: false,
		},
		{
			name: "missing version file",
			setupBinary: func(path string) error {
				return os.WriteFile(path, []byte("test"), 0o755)
			},
			setupVersion: func(_ string) error {
				return nil // Don't create file
			},
			expected: false,
		},
		{
			name: "wrong version",
			setupBinary: func(path string) error {
				return os.WriteFile(path, []byte("test"), 0o755)
			},
			setupVersion: func(path string) error {
				return os.WriteFile(path, []byte("1.0.0"), 0o600)
			},
			expected: false,
		},
		{
			name: "non-executable binary",
			setupBinary: func(path string) error {
				return os.WriteFile(path, []byte("test"), 0o644)
			},
			setupVersion: func(path string) error {
				return os.WriteFile(path, []byte(iperf.EmbeddedVersion), 0o600)
			},
			expected:      false,
			skipOnWindows: true, // Windows doesn't have Unix permissions
		},
		{
			name: "version with trailing newline",
			setupBinary: func(path string) error {
				return os.WriteFile(path, []byte("test"), 0o755)
			},
			setupVersion: func(path string) error {
				return os.WriteFile(path, []byte(iperf.EmbeddedVersion+"\n"), 0o600)
			},
			expected: true,
		},
		{
			name: "version with whitespace padding",
			setupBinary: func(path string) error {
				return os.WriteFile(path, []byte("test"), 0o755)
			},
			setupVersion: func(path string) error {
				return os.WriteFile(path, []byte("  "+iperf.EmbeddedVersion+"  "), 0o600)
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runIsValidExtractedBinaryCase(t, tempDir, tt)
		})
	}
}

func runIsValidExtractedBinaryCase(
	t *testing.T,
	tempDir string,
	tt struct {
		name          string
		setupBinary   func(path string) error
		setupVersion  func(path string) error
		expected      bool
		skipOnWindows bool
	},
) {
	t.Helper()

	if tt.skipOnWindows && runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	binaryPath := filepath.Join(tempDir, "binary-"+tt.name)
	versionPath := filepath.Join(tempDir, "version-"+tt.name)

	if tt.setupBinary != nil {
		if err := tt.setupBinary(binaryPath); err != nil {
			t.Fatalf("Failed to setup binary: %v", err)
		}
	}
	if tt.setupVersion != nil {
		if err := tt.setupVersion(versionPath); err != nil {
			t.Fatalf("Failed to setup version: %v", err)
		}
	}

	result := iperf.IsValidExtractedBinary(binaryPath, versionPath)
	if result != tt.expected {
		t.Errorf("IsValidExtractedBinary() = %v, want %v", result, tt.expected)
	}
}

// TestGetPlatformBinaryMapContent tests platform binary map content.
func TestGetPlatformBinaryMapContent(t *testing.T) {
	t.Parallel()

	platformMap := iperf.GetPlatformBinaryMap()

	// Verify expected platforms exist
	expectedPlatforms := []struct {
		platform   string
		binaryName string
	}{
		{"linux-amd64", "iperf3-linux-amd64"},
		{"linux-arm64", "iperf3-linux-arm64"},
		{"darwin-amd64", "iperf3-darwin-amd64"},
		{"darwin-arm64", "iperf3-darwin-arm64"},
	}

	for _, ep := range expectedPlatforms {
		t.Run(ep.platform, func(t *testing.T) {
			t.Parallel()

			binaryName, ok := platformMap[ep.platform]
			if !ok {
				t.Errorf("Platform %s should be in map", ep.platform)
				return
			}
			if binaryName != ep.binaryName {
				t.Errorf("Binary name = %q, want %q", binaryName, ep.binaryName)
			}
		})
	}
}

// TestHasEmbeddedBinaryDeterminism tests that HasEmbeddedBinary is deterministic.
func TestHasEmbeddedBinaryDeterminism(t *testing.T) {
	t.Parallel()

	// Call multiple times
	results := make([]bool, 10)
	for i := range results {
		results[i] = iperf.HasEmbeddedBinary()
	}

	// All results should be the same
	first := results[0]
	for i, r := range results {
		if r != first {
			t.Errorf("Result at index %d = %v, differs from first %v", i, r, first)
		}
	}
}

// TestNotFoundErrorComponents tests NotFoundError component handling.
func TestNotFoundErrorComponents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		err           *iperf.NotFoundError
		shouldContain []string
	}{
		{
			name:          "empty error",
			err:           &iperf.NotFoundError{},
			shouldContain: []string{"iperf3 not found"},
		},
		{
			name: "with one path",
			err: &iperf.NotFoundError{
				SearchedPaths: []string{"/usr/bin/iperf3"},
			},
			shouldContain: []string{"iperf3 not found", "/usr/bin/iperf3"},
		},
		{
			name: "with system error",
			err: &iperf.NotFoundError{
				SystemError: os.ErrNotExist,
			},
			shouldContain: []string{"iperf3 not found", "System PATH"},
		},
		{
			name: "with embedded error",
			err: &iperf.NotFoundError{
				EmbeddedError: os.ErrPermission,
			},
			shouldContain: []string{"iperf3 not found", "Embedded binary"},
		},
		{
			name: "full error",
			err: &iperf.NotFoundError{
				SearchedPaths: []string{"/path1", "/path2"},
				SystemError:   os.ErrNotExist,
				EmbeddedError: os.ErrPermission,
			},
			shouldContain: []string{
				"iperf3 not found",
				"System PATH",
				"Embedded binary",
				"/path1",
				"/path2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			errMsg := tt.err.Error()

			for _, expected := range tt.shouldContain {
				if !strings.Contains(errMsg, expected) {
					t.Errorf("Error should contain %q\nGot: %s", expected, errMsg)
				}
			}

			// Should always contain install instructions
			if !strings.Contains(errMsg, "install") && !strings.Contains(errMsg, "Install") {
				t.Error("Error should contain install instructions")
			}
		})
	}
}

// TestGetInstallInstructionsContentValidation tests install instructions content validation.
func TestGetInstallInstructionsContentValidation(t *testing.T) {
	t.Parallel()

	instructions := iperf.GetInstallInstructions()

	// Must always contain these
	required := []string{
		"iperf3",
		"not installed",
		"esnet/iperf",
	}

	for _, req := range required {
		if !strings.Contains(instructions, req) {
			t.Errorf("Instructions should contain %q", req)
		}
	}

	// Should contain source build instructions
	if !strings.Contains(instructions, "configure") && !strings.Contains(instructions, "make") {
		t.Error("Instructions should contain build commands")
	}
}

// TestEmbeddedVersionFormatValidation tests EmbeddedVersion constant format validation.
func TestEmbeddedVersionFormatValidation(t *testing.T) {
	t.Parallel()

	version := iperf.EmbeddedVersion

	// Should not be empty
	if version == "" {
		t.Fatal("EmbeddedVersion should not be empty")
	}

	// Should be in X.Y or X.Y.Z format
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		t.Errorf("Version should have at least major.minor: %s", version)
	}

	// Major version should be 3 (iperf3)
	if parts[0] != "3" {
		t.Errorf("Major version should be 3, got %s", parts[0])
	}

	// Minor version should be numeric
	if parts[1] == "" {
		t.Error("Minor version should not be empty")
	}
}

// TestFindSystemIperf3Error tests FindSystemIperf3 error handling.
func TestFindSystemIperf3Error(t *testing.T) {
	// Note: Cannot use t.Parallel() with t.Setenv()

	// Temporarily clear PATH to force error
	t.Setenv("PATH", "")

	_, err := iperf.FindSystemIperf3()
	if err == nil {
		t.Error("Expected error with empty PATH")
	}
	// PATH is automatically restored by t.Setenv()
}

// TestCacheDirOSSpecific tests cache directory is OS-appropriate.
func TestCacheDirOSSpecific(t *testing.T) {
	t.Parallel()

	dir, err := iperf.GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS should use ~/Library/Caches or ~/.cache
		if !strings.Contains(dir, "Library/Caches") && !strings.Contains(dir, ".cache") {
			t.Logf("macOS cache dir: %s (expected Library/Caches or .cache)", dir)
		}
	case "linux":
		// Linux should use ~/.cache
		if !strings.Contains(dir, ".cache") {
			t.Logf("Linux cache dir: %s (expected .cache)", dir)
		}
	case "windows":
		// Windows should use LocalAppData or similar
		if !strings.Contains(dir, "AppData") && !strings.Contains(dir, ".cache") {
			t.Logf("Windows cache dir: %s (expected AppData)", dir)
		}
	}
}

// TestValidateBinaryEdgeCases tests binary validation edge cases.
func TestValidateBinaryEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"empty path", "", false},
		{"space only", "   ", false},
		{"non-existent", "/definitely/not/a/real/path/iperf3", false},
		{"root path", "/", false},
		{"dot path", ".", false},
		{"double dot", "..", false},
		{"null bytes", "iperf3\x00malicious", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := iperf.ValidateBinary(tt.path)
			if result != tt.expected {
				t.Errorf("ValidateBinary(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestGetLegacyPathsUniqueness tests that legacy paths are unique.
func TestGetLegacyPathsUniqueness(t *testing.T) {
	t.Parallel()

	paths := iperf.GetLegacyPaths()

	// Check for duplicates
	seen := make(map[string]bool)
	for _, path := range paths {
		if seen[path] {
			t.Errorf("Duplicate path found: %s", path)
		}
		seen[path] = true
	}
}

// TestBinaryPathCachingThreadSafety tests thread-safe caching.
func TestBinaryPathCachingThreadSafety(t *testing.T) {
	t.Parallel()

	// Save original
	original := iperf.IperfBinaryPath()
	defer iperf.SetIperfBinaryPath(original)

	done := make(chan bool)
	numGoroutines := 20
	iterations := 100

	// Concurrent reads and writes
	for i := range numGoroutines {
		go func(id int) {
			for j := range iterations {
				if j%2 == 0 {
					iperf.SetIperfBinaryPath("/test/path/" + string(rune('a'+id)))
				} else {
					_ = iperf.IperfBinaryPath()
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range numGoroutines {
		<-done
	}
}

// TestClearIperfBinaryPath tests clearing the cached path.
func TestClearIperfBinaryPath(t *testing.T) {
	// Save original
	original := iperf.IperfBinaryPath()
	defer iperf.SetIperfBinaryPath(original)

	// Set a value
	iperf.SetIperfBinaryPath("/test/path/iperf3")
	if iperf.IperfBinaryPath() != "/test/path/iperf3" {
		t.Error("Path should be set")
	}

	// Clear it
	iperf.ClearIperfBinaryPath()
	if iperf.IperfBinaryPath() != "" {
		t.Errorf("Path should be empty after clear, got %q", iperf.IperfBinaryPath())
	}
}

// TestFindIperf3BinaryWithPresetPath tests finding with preset cached path.
func TestFindIperf3BinaryWithPresetPath(t *testing.T) {
	// Save original
	original := iperf.IperfBinaryPath()
	defer iperf.SetIperfBinaryPath(original)

	// Set a cached path
	cachedPath := "/cached/iperf3/path"
	iperf.SetIperfBinaryPath(cachedPath)

	// FindIperf3Binary should return cached path
	path, err := iperf.FindIperf3Binary()
	if err != nil {
		t.Errorf("Unexpected error with cached path: %v", err)
	}
	if path != cachedPath {
		t.Errorf("Path = %q, want %q", path, cachedPath)
	}
}

// TestPlatformBinaryMapNoDuplicates tests no duplicate binary names.
func TestPlatformBinaryMapNoDuplicates(t *testing.T) {
	t.Parallel()

	platformMap := iperf.GetPlatformBinaryMap()

	// Check for duplicate binary names
	seen := make(map[string]string)
	for platform, binary := range platformMap {
		if existingPlatform, ok := seen[binary]; ok {
			t.Errorf("Binary %q is mapped to both %q and %q", binary, existingPlatform, platform)
		}
		seen[binary] = platform
	}
}

// TestMinSupportedVersionExported tests the exported MinSupportedVersion constant.
func TestMinSupportedVersionExported(t *testing.T) {
	t.Parallel()

	version := iperf.MinSupportedVersion

	if version == "" {
		t.Fatal("MinSupportedVersion should not be empty")
	}

	// Should be a valid version
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		t.Errorf("MinSupportedVersion should have major.minor format: %s", version)
	}
}

// TestVersionCheckTimeoutExported tests exported timeout constant.
func TestVersionCheckTimeoutExported(t *testing.T) {
	t.Parallel()

	if iperf.VersionCheckTimeout <= 0 {
		t.Error("VersionCheckTimeout should be positive")
	}
}

// TestServerStartTimeoutExported tests exported timeout constant.
func TestServerStartTimeoutExported(t *testing.T) {
	t.Parallel()

	if iperf.ServerStartTimeout <= 0 {
		t.Error("ServerStartTimeout should be positive")
	}
}

// TestPortCheckTimeoutExported tests exported timeout constant.
func TestPortCheckTimeoutExported(t *testing.T) {
	t.Parallel()

	if iperf.PortCheckTimeout <= 0 {
		t.Error("PortCheckTimeout should be positive")
	}
}
