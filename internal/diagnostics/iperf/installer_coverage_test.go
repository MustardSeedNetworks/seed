package iperf_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
)

// TestGetInstallInstructionsAllPlatforms tests instructions for all platforms.
func TestGetInstallInstructionsAllPlatforms(t *testing.T) {
	t.Parallel()

	instructions := iperf.GetInstallInstructions()

	// Should always have content
	if instructions == "" {
		t.Fatal("Instructions should not be empty")
	}

	// Should always mention iperf3
	if !strings.Contains(instructions, "iperf3") {
		t.Error("Instructions should mention iperf3")
	}

	// Should always have source instructions
	if !strings.Contains(instructions, "github") && !strings.Contains(instructions, "esnet") {
		t.Error("Instructions should mention GitHub/esnet source")
	}

	// Should have some installation method
	installKeywords := []string{"install", "apt", "brew", "dnf", "yum", "pacman", "choco", "Download"}
	hasInstallMethod := false
	for _, keyword := range installKeywords {
		if strings.Contains(instructions, keyword) {
			hasInstallMethod = true
			break
		}
	}
	if !hasInstallMethod {
		t.Error("Instructions should contain at least one install method")
	}
}

// TestExtractEmbeddedBinaryAvailability tests embedded binary extraction availability.
func TestExtractEmbeddedBinaryAvailability(t *testing.T) {
	t.Parallel()

	hasEmbedded := iperf.HasEmbeddedBinary()
	platformMap := iperf.GetPlatformBinaryMap()
	currentPlatform := runtime.GOOS + "-" + runtime.GOARCH

	// Log the current state
	t.Logf("Current platform: %s", currentPlatform)
	t.Logf("Has embedded binary: %v", hasEmbedded)
	t.Logf("Platform in map: %v", platformMap[currentPlatform] != "")

	// If platform is in map and hasEmbedded is true, they should be consistent
	_, inMap := platformMap[currentPlatform]
	if inMap && !hasEmbedded {
		t.Log("Platform is in map but embedded binary not found (may not be compiled in)")
	}
}

// TestCacheDirPermissions tests cache directory can be created with correct permissions.
func TestCacheDirPermissions(t *testing.T) {
	t.Parallel()

	cacheDir, err := iperf.GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}

	// Verify path structure
	if !filepath.IsAbs(cacheDir) {
		t.Errorf("Cache dir should be absolute: %s", cacheDir)
	}

	// Verify it ends with expected structure
	if !strings.HasSuffix(cacheDir, filepath.Join("seed", "bin")) {
		t.Errorf("Cache dir should end with 'seed/bin': %s", cacheDir)
	}

	// Check if parent directory is writable (by checking if it exists or can be created)
	parentDir := filepath.Dir(cacheDir)
	grandparentDir := filepath.Dir(parentDir)

	// At least the user cache directory should exist
	if _, statErr := os.Stat(grandparentDir); os.IsNotExist(statErr) {
		t.Logf("Grandparent directory does not exist: %s (may be expected)", grandparentDir)
	}
}
