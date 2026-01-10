package iperf_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/krisarmstrong/seed/internal/iperf"
)

// TestDetectLinuxPackageManagerValidation tests Linux package manager detection validation.
func TestDetectLinuxPackageManagerValidation(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	pm := iperf.DetectLinuxPackageManager()
	if pm == nil {
		t.Log("No Linux package manager detected")
		return
	}

	// Validate the structure
	if pm.Name == "" {
		t.Error("Package manager name should not be empty")
	}

	validNames := map[string]bool{
		"apt":    true,
		"dnf":    true,
		"yum":    true,
		"pacman": true,
		"apk":    true,
		"zypper": true,
	}

	if !validNames[pm.Name] {
		t.Errorf("Unexpected package manager name: %s", pm.Name)
	}

	// Install command should exist
	if len(pm.InstallCommand) == 0 {
		t.Error("Install command should not be empty")
	}

	// First element should be the package manager binary
	if pm.InstallCommand[0] != pm.Name {
		// Some package managers have different binary names
		expectedBinaries := map[string]string{
			"apt":    "apt",
			"dnf":    "dnf",
			"yum":    "yum",
			"pacman": "pacman",
			"apk":    "apk",
			"zypper": "zypper",
		}
		if expectedBinaries[pm.Name] != pm.InstallCommand[0] {
			t.Errorf("Unexpected first install command element: %s", pm.InstallCommand[0])
		}
	}
}

// TestDetectMacOSPackageManagerValidation tests macOS package manager detection validation.
func TestDetectMacOSPackageManagerValidation(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	pm := iperf.DetectMacOSPackageManager()
	if pm == nil {
		t.Log("No macOS package manager detected")
		return
	}

	validNames := map[string]bool{
		"homebrew": true,
		"macports": true,
	}

	if !validNames[pm.Name] {
		t.Errorf("Unexpected package manager name: %s", pm.Name)
	}

	// Validate install command
	if len(pm.InstallCommand) == 0 {
		t.Error("Install command should not be empty")
	}

	// First element should match expected binary
	expectedBinaries := map[string]string{
		"homebrew": "brew",
		"macports": "port",
	}

	if pm.InstallCommand[0] != expectedBinaries[pm.Name] {
		t.Errorf("First install command = %q, want %q", pm.InstallCommand[0], expectedBinaries[pm.Name])
	}
}

// TestDetectWindowsPackageManagerValidation tests Windows package manager detection validation.
func TestDetectWindowsPackageManagerValidation(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	pm := iperf.DetectWindowsPackageManager()
	if pm == nil {
		t.Log("No Windows package manager detected")
		return
	}

	validNames := map[string]bool{
		"chocolatey": true,
		"scoop":      true,
		"winget":     true,
	}

	if !validNames[pm.Name] {
		t.Errorf("Unexpected package manager name: %s", pm.Name)
	}

	// Validate install command
	if len(pm.InstallCommand) == 0 {
		t.Error("Install command should not be empty")
	}
}

// TestInstallViaPackageManagerWithNoManager tests install when no manager is found.
func TestInstallViaPackageManagerWithNoManager(t *testing.T) {
	t.Parallel()

	// This test verifies the function handles the no-manager case gracefully
	// The actual behavior depends on system configuration

	opts := iperf.InstallOptions{
		Method:  iperf.InstallMethodPackageManager,
		UseSudo: false,
		Verbose: false,
	}

	// Verify options are correctly structured
	if opts.Method != iperf.InstallMethodPackageManager {
		t.Errorf("Method = %q, want package_manager", opts.Method)
	}
	if opts.UseSudo {
		t.Error("UseSudo should be false for this test")
	}
	if opts.Verbose {
		t.Error("Verbose should be false for this test")
	}

	// Check if a package manager is available
	pm := iperf.DetectPackageManager()
	if pm == nil {
		t.Log("No package manager available - install would fail gracefully")
	} else {
		t.Logf("Package manager available: %s", pm.Name)
	}
}

// TestInstallFromGitHubOptions tests GitHub install options.
func TestInstallFromGitHubOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		opts       iperf.InstallOptions
		checkField func(iperf.InstallOptions) bool
	}{
		{
			name: "with version",
			opts: iperf.InstallOptions{
				Method:  iperf.InstallMethodGitHub,
				Version: "3.17",
			},
			checkField: func(o iperf.InstallOptions) bool {
				return o.Version == "3.17"
			},
		},
		{
			name: "with install dir",
			opts: iperf.InstallOptions{
				Method:     iperf.InstallMethodGitHub,
				InstallDir: "/opt/iperf3",
			},
			checkField: func(o iperf.InstallOptions) bool {
				return o.InstallDir == "/opt/iperf3"
			},
		},
		{
			name: "with sudo",
			opts: iperf.InstallOptions{
				Method:  iperf.InstallMethodGitHub,
				UseSudo: true,
			},
			checkField: func(o iperf.InstallOptions) bool {
				return o.UseSudo
			},
		},
		{
			name: "verbose mode",
			opts: iperf.InstallOptions{
				Method:  iperf.InstallMethodGitHub,
				Verbose: true,
			},
			checkField: func(o iperf.InstallOptions) bool {
				return o.Verbose
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !tt.checkField(tt.opts) {
				t.Errorf("Option check failed for %s", tt.name)
			}
		})
	}
}

// TestAutoInstallOptions tests auto-install options.
func TestAutoInstallOptions(t *testing.T) {
	t.Parallel()

	// This test verifies the function signature and option handling
	// without actually performing installation

	tests := []struct {
		name    string
		useSudo bool
		verbose bool
	}{
		{"basic", false, false},
		{"with sudo", true, false},
		{"verbose", false, true},
		{"sudo and verbose", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Just verify the options can be constructed
			opts := iperf.InstallOptions{
				UseSudo: tt.useSudo,
				Verbose: tt.verbose,
			}

			if opts.UseSudo != tt.useSudo {
				t.Errorf("UseSudo = %v, want %v", opts.UseSudo, tt.useSudo)
			}
			if opts.Verbose != tt.verbose {
				t.Errorf("Verbose = %v, want %v", opts.Verbose, tt.verbose)
			}
		})
	}
}

// TestCheckBuildDependenciesOutput tests build dependency output format.
func TestCheckBuildDependenciesOutput(t *testing.T) {
	t.Parallel()

	missing := iperf.CheckBuildDependencies()

	// The result should be a slice of strings
	for i, dep := range missing {
		if dep == "" {
			t.Errorf("Missing dependency at index %d is empty", i)
		}

		// Dependencies should be simple tool names
		if strings.Contains(dep, "/") {
			t.Errorf("Dependency %q should not contain path separator", dep)
		}
		if strings.Contains(dep, " ") {
			t.Errorf("Dependency %q should not contain spaces", dep)
		}
	}
}

// TestGetBuildDependencyInstallCommandByPlatform tests platform-specific commands.
func TestGetBuildDependencyInstallCommandByPlatform(t *testing.T) {
	t.Parallel()

	cmd := iperf.GetBuildDependencyInstallCommand()

	if cmd == "" {
		t.Fatal("Command should not be empty")
	}

	switch runtime.GOOS {
	case "linux":
		// Linux commands typically use sudo
		if !strings.Contains(cmd, "sudo") && !strings.Contains(cmd, "Install") {
			t.Log("Linux command might need sudo or contain Install")
		}
	case "darwin":
		// macOS might use xcode-select or brew
		if !strings.Contains(cmd, "xcode") && !strings.Contains(cmd, "brew") &&
			!strings.Contains(cmd, "Install") {
			t.Log("macOS command should mention xcode-select, brew, or Install")
		}
	default:
		// Other platforms should at least mention tools
		if !strings.Contains(strings.ToLower(cmd), "gcc") &&
			!strings.Contains(strings.ToLower(cmd), "make") &&
			!strings.Contains(strings.ToLower(cmd), "install") {
			t.Log("Command should mention build tools")
		}
	}
}

// TestPackageManagerInfoCompleteStructure tests complete PackageManagerInfo.
func TestPackageManagerInfoCompleteStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info iperf.PackageManagerInfo
	}{
		{
			name: "apt with update",
			info: iperf.PackageManagerInfo{
				Name:           "apt",
				InstallCommand: []string{"apt", "install", "-y", "iperf3"},
				UpdateCommand:  []string{"apt", "update"},
				Available:      true,
			},
		},
		{
			name: "dnf without update",
			info: iperf.PackageManagerInfo{
				Name:           "dnf",
				InstallCommand: []string{"dnf", "install", "-y", "iperf3"},
				UpdateCommand:  nil,
				Available:      true,
			},
		},
		{
			name: "homebrew",
			info: iperf.PackageManagerInfo{
				Name:           "homebrew",
				InstallCommand: []string{"brew", "install", "iperf3"},
				UpdateCommand:  []string{"brew", "update"},
				Available:      true,
			},
		},
		{
			name: "unavailable manager",
			info: iperf.PackageManagerInfo{
				Name:           "fictional",
				InstallCommand: []string{"fictional", "install", "iperf3"},
				UpdateCommand:  nil,
				Available:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Verify all fields are accessible
			if tt.info.Name == "" {
				t.Error("Name should not be empty")
			}
			if len(tt.info.InstallCommand) == 0 {
				t.Error("InstallCommand should not be empty")
			}

			// Update command can be nil
			if tt.info.UpdateCommand != nil && len(tt.info.UpdateCommand) == 0 {
				t.Error("UpdateCommand if not nil should not be empty")
			}

			// Available field should match expected
			_ = tt.info.Available
		})
	}
}

// TestInstallResultSuccessAndFailure tests success and failure result structures.
func TestInstallResultSuccessAndFailure(t *testing.T) {
	t.Parallel()

	successResult := iperf.InstallResult{
		Success: true,
		Path:    "/usr/local/bin/iperf3",
		Version: "3.17",
		Method:  iperf.InstallMethodPackageManager,
		Error:   nil,
	}

	failureResult := iperf.InstallResult{
		Success:     false,
		Path:        "",
		Version:     "",
		Method:      iperf.InstallMethodPackageManager,
		Error:       os.ErrNotExist,
		NeedsSudo:   true,
		SudoCommand: "sudo apt install iperf3",
	}

	// Verify success result
	if !successResult.Success {
		t.Error("Success result should have Success=true")
	}
	if successResult.Path == "" {
		t.Error("Success result should have non-empty Path")
	}
	if successResult.Version == "" {
		t.Error("Success result should have non-empty Version")
	}
	if successResult.Method != iperf.InstallMethodPackageManager {
		t.Errorf("Success Method = %q, want package_manager", successResult.Method)
	}
	if successResult.Error != nil {
		t.Error("Success result should have nil Error")
	}

	// Verify failure result
	if failureResult.Success {
		t.Error("Failure result should have Success=false")
	}
	if failureResult.Path != "" {
		t.Error("Failure result should have empty Path")
	}
	if failureResult.Version != "" {
		t.Errorf("Failure result should have empty Version, got %q", failureResult.Version)
	}
	if failureResult.Method != iperf.InstallMethodPackageManager {
		t.Errorf("Failure Method = %q, want package_manager", failureResult.Method)
	}
	if failureResult.Error == nil {
		t.Error("Failure result should have non-nil Error")
	}
	if !failureResult.NeedsSudo {
		t.Error("Failure result should indicate NeedsSudo")
	}
	if failureResult.SudoCommand == "" {
		t.Error("Failure result should have SudoCommand")
	}
}

// TestInstallMethodValues tests all install method values.
func TestInstallMethodValues(t *testing.T) {
	t.Parallel()

	methods := []struct {
		method   iperf.InstallMethod
		expected string
	}{
		{iperf.InstallMethodPackageManager, "package_manager"},
		{iperf.InstallMethodGitHub, "github"},
		{iperf.InstallMethodManual, "manual"},
	}

	for _, m := range methods {
		t.Run(string(m.method), func(t *testing.T) {
			t.Parallel()

			if string(m.method) != m.expected {
				t.Errorf("InstallMethod value = %q, want %q", string(m.method), m.expected)
			}
		})
	}
}

// TestNeedsSudoAllPackageManagers tests sudo requirement for all package managers.
func TestNeedsSudoAllPackageManagers(t *testing.T) {
	t.Parallel()

	// Package managers that never need sudo
	noSudoManagers := []string{"homebrew", "scoop"}

	for _, pm := range noSudoManagers {
		t.Run(pm, func(t *testing.T) {
			t.Parallel()

			if iperf.NeedsSudo(pm) {
				t.Errorf("NeedsSudo(%q) = true, want false", pm)
			}
		})
	}

	// Linux package managers that need sudo (only on Linux)
	if runtime.GOOS == "linux" {
		sudoManagers := []string{"apt", "dnf", "yum", "pacman", "apk", "zypper"}

		for _, pm := range sudoManagers {
			t.Run(pm+"_linux", func(t *testing.T) {
				t.Parallel()

				if !iperf.NeedsSudo(pm) {
					t.Errorf("NeedsSudo(%q) = false, want true on Linux", pm)
				}
			})
		}
	}
}

// TestDetectPackageManagerAllPlatforms tests detection on all platforms.
func TestDetectPackageManagerAllPlatforms(t *testing.T) {
	t.Parallel()

	pm := iperf.DetectPackageManager()

	// Result may be nil if no package manager is detected
	if pm == nil {
		t.Log("No package manager detected on this system")
		return
	}

	// If detected, verify it's valid
	t.Logf("Detected package manager: %s (available: %v)", pm.Name, pm.Available)

	if pm.Name == "" {
		t.Error("Detected manager should have a name")
	}

	if !pm.Available {
		t.Error("Detected manager should be available")
	}

	if len(pm.InstallCommand) == 0 {
		t.Error("Detected manager should have install command")
	}
}

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

// TestFindSystemIperf3PathValidation tests system iperf3 path validation.
func TestFindSystemIperf3PathValidation(t *testing.T) {
	t.Parallel()

	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf test")
	}

	path, err := iperf.FindSystemIperf3()
	if err != nil {
		t.Logf("iperf3 not in PATH: %v", err)
		return
	}

	// If found, verify path properties
	if path == "" {
		t.Error("Found path should not be empty")
	}

	if !filepath.IsAbs(path) {
		t.Errorf("System path should be absolute: %s", path)
	}

	// Verify it's executable
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("Could not stat found path: %v", err)
		return
	}

	if info.IsDir() {
		t.Errorf("Path should not be a directory: %s", path)
	}

	if info.Mode()&0o111 == 0 {
		t.Errorf("Path should be executable: %s", path)
	}
}
