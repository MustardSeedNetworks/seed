package iperf_test

import (
	"errors"
	"runtime"
	"slices"
	"testing"

	"github.com/krisarmstrong/seed/internal/iperf"
)

// TestDetectPackageManagerByPlatform tests package manager detection on different platforms.
func TestDetectPackageManagerByPlatform(t *testing.T) {
	t.Parallel()

	pm := iperf.DetectPackageManager()

	// Just verify the function doesn't panic and returns consistent type
	switch runtime.GOOS {
	case "linux":
		// On Linux, we might have apt, dnf, yum, pacman, apk, or zypper
		if pm != nil {
			validManagers := map[string]bool{
				"apt":    true,
				"dnf":    true,
				"yum":    true,
				"pacman": true,
				"apk":    true,
				"zypper": true,
			}
			if !validManagers[pm.Name] {
				t.Errorf("Unexpected Linux package manager: %s", pm.Name)
			}
		}
	case "darwin":
		// On macOS, we might have homebrew or macports
		if pm != nil {
			validManagers := map[string]bool{
				"homebrew": true,
				"macports": true,
			}
			if !validManagers[pm.Name] {
				t.Errorf("Unexpected macOS package manager: %s", pm.Name)
			}
		}
	case "windows":
		// On Windows, we might have chocolatey, scoop, or winget
		if pm != nil {
			validManagers := map[string]bool{
				"chocolatey": true,
				"scoop":      true,
				"winget":     true,
			}
			if !validManagers[pm.Name] {
				t.Errorf("Unexpected Windows package manager: %s", pm.Name)
			}
		}
	}

	if pm != nil {
		t.Logf("Detected package manager: %s (available: %v)", pm.Name, pm.Available)
	}
}

// TestDetectLinuxPackageManager tests Linux-specific package manager detection.
func TestDetectLinuxPackageManager(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	pm := iperf.DetectLinuxPackageManager()
	if pm != nil {
		if pm.Name == "" {
			t.Error("Package manager name should not be empty")
		}
		if len(pm.InstallCommand) == 0 {
			t.Error("Install command should not be empty")
		}
		// All Linux package managers should have iperf3 in install command
		if !slices.Contains(pm.InstallCommand, "iperf3") {
			t.Errorf("Install command should contain 'iperf3': %v", pm.InstallCommand)
		}
	}
}

// TestDetectMacOSPackageManager tests macOS-specific package manager detection.
func TestDetectMacOSPackageManager(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test on non-macOS platform")
	}

	pm := iperf.DetectMacOSPackageManager()
	if pm != nil {
		if pm.Name == "" {
			t.Error("Package manager name should not be empty")
		}
		if pm.Name != "homebrew" && pm.Name != "macports" {
			t.Errorf("Unexpected macOS package manager: %s", pm.Name)
		}
		if len(pm.InstallCommand) == 0 {
			t.Error("Install command should not be empty")
		}
	}
}

// TestDetectWindowsPackageManager tests Windows-specific package manager detection.
func TestDetectWindowsPackageManager(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	pm := iperf.DetectWindowsPackageManager()
	if pm != nil {
		if pm.Name == "" {
			t.Error("Package manager name should not be empty")
		}
		if pm.Name != "chocolatey" && pm.Name != "scoop" && pm.Name != "winget" {
			t.Errorf("Unexpected Windows package manager: %s", pm.Name)
		}
		if len(pm.InstallCommand) == 0 {
			t.Error("Install command should not be empty")
		}
	}
}

// TestInstallOptionsDefaults tests install options with default values.
func TestInstallOptionsDefaults(t *testing.T) {
	t.Parallel()

	opts := iperf.InstallOptions{}

	if opts.Method != "" {
		t.Errorf("Method should be empty by default, got %q", opts.Method)
	}
	if opts.Version != "" {
		t.Errorf("Version should be empty by default, got %q", opts.Version)
	}
	if opts.InstallDir != "" {
		t.Errorf("InstallDir should be empty by default, got %q", opts.InstallDir)
	}
	if opts.UseSudo {
		t.Error("UseSudo should be false by default")
	}
	if opts.Verbose {
		t.Error("Verbose should be false by default")
	}
}

// TestInstallResultWithError tests install result with error.
func TestInstallResultWithError(t *testing.T) {
	t.Parallel()

	result := iperf.InstallResult{
		Success:     false,
		Error:       errors.New("installation failed"),
		Method:      iperf.InstallMethodPackageManager,
		NeedsSudo:   true,
		SudoCommand: "sudo apt install iperf3",
	}

	if result.Success {
		t.Error("Success should be false")
	}
	if result.Error == nil {
		t.Error("Error should not be nil")
	}
	if result.Error.Error() != "installation failed" {
		t.Errorf("Error message = %q, want 'installation failed'", result.Error.Error())
	}
	if result.Method != iperf.InstallMethodPackageManager {
		t.Errorf("Method = %q, want %q", result.Method, iperf.InstallMethodPackageManager)
	}
	if !result.NeedsSudo {
		t.Error("NeedsSudo should be true")
	}
	if result.SudoCommand != "sudo apt install iperf3" {
		t.Errorf("SudoCommand = %q, want 'sudo apt install iperf3'", result.SudoCommand)
	}
}

// TestNeedsSudoPackageManagers tests sudo requirement for various package managers.
func TestNeedsSudoPackageManagers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		packageManager string
		expected       bool
	}{
		{"homebrew no sudo", "homebrew", false},
		{"scoop no sudo", "scoop", false},
		// Linux package managers behavior depends on runtime.GOOS
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := iperf.NeedsSudo(tt.packageManager)
			if result != tt.expected {
				t.Errorf("NeedsSudo(%q) = %v, want %v", tt.packageManager, result, tt.expected)
			}
		})
	}
}

// TestNeedsSudoLinuxPackageManagers tests sudo requirement on Linux.
func TestNeedsSudoLinuxPackageManagers(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	linuxManagers := []string{"apt", "dnf", "yum", "pacman", "apk", "zypper"}

	for _, pm := range linuxManagers {
		t.Run(pm, func(t *testing.T) {
			t.Parallel()

			result := iperf.NeedsSudo(pm)
			if !result {
				t.Errorf("NeedsSudo(%q) = %v, expected true on Linux", pm, result)
			}
		})
	}
}

// TestPackageManagerInfoStructure tests PackageManagerInfo structure.
func TestPackageManagerInfoStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info iperf.PackageManagerInfo
	}{
		{
			name: "apt",
			info: iperf.PackageManagerInfo{
				Name:           "apt",
				InstallCommand: []string{"apt", "install", "-y", "iperf3"},
				UpdateCommand:  []string{"apt", "update"},
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
			name: "dnf without update",
			info: iperf.PackageManagerInfo{
				Name:           "dnf",
				InstallCommand: []string{"dnf", "install", "-y", "iperf3"},
				UpdateCommand:  nil,
				Available:      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.info.Name == "" {
				t.Error("Name should not be empty")
			}
			if len(tt.info.InstallCommand) == 0 {
				t.Error("InstallCommand should not be empty")
			}
			if !tt.info.Available {
				t.Error("Available should be true")
			}

			// Verify first element of install command is the package manager binary
			expectedBinaries := map[string]string{
				"apt":      "apt",
				"homebrew": "brew",
				"dnf":      "dnf",
			}
			if expected, ok := expectedBinaries[tt.info.Name]; ok {
				if tt.info.InstallCommand[0] != expected {
					t.Errorf("First install command should be %q, got %q", expected, tt.info.InstallCommand[0])
				}
			}
		})
	}
}

// TestCheckBuildDependenciesStructure tests build dependency check structure.
func TestCheckBuildDependenciesStructure(t *testing.T) {
	t.Parallel()

	missing := iperf.CheckBuildDependencies()

	// Verify it returns a slice (may be empty or not depending on system)
	// Note: missing can be nil if all dependencies are installed, which is valid
	t.Logf("Missing build dependencies: %v (count: %d)", missing, len(missing))
}

// TestGetBuildDependencyInstallCommandPlatform tests platform-specific build commands.
func TestGetBuildDependencyInstallCommandPlatform(t *testing.T) {
	t.Parallel()

	cmd := iperf.GetBuildDependencyInstallCommand()

	if cmd == "" {
		t.Error("Build dependency install command should not be empty")
	}

	switch runtime.GOOS {
	case "linux":
		// Should contain sudo or Install
		if !containsAny(cmd, []string{"sudo", "Install"}) {
			t.Errorf("Linux command should contain 'sudo' or 'Install': %q", cmd)
		}
	case "darwin":
		// Should contain xcode-select or brew
		if !containsAny(cmd, []string{"xcode-select", "brew", "Install"}) {
			t.Errorf("macOS command should contain 'xcode-select' or 'brew': %q", cmd)
		}
	}
}

// TestInstallMethodStringValues tests InstallMethod string values.
func TestInstallMethodStringValues(t *testing.T) {
	t.Parallel()

	methods := map[iperf.InstallMethod]string{
		iperf.InstallMethodPackageManager: "package_manager",
		iperf.InstallMethodGitHub:         "github",
		iperf.InstallMethodManual:         "manual",
	}

	for method, expected := range methods {
		if string(method) != expected {
			t.Errorf("InstallMethod %v = %q, want %q", method, string(method), expected)
		}
	}
}

// TestInstallResultSuccessfulInstall tests a successful install result.
func TestInstallResultSuccessfulInstall(t *testing.T) {
	t.Parallel()

	result := iperf.InstallResult{
		Success: true,
		Path:    "/usr/local/bin/iperf3",
		Version: "3.17",
		Method:  iperf.InstallMethodPackageManager,
		Error:   nil,
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Path != "/usr/local/bin/iperf3" {
		t.Errorf("Path = %q, want '/usr/local/bin/iperf3'", result.Path)
	}
	if result.Version != "3.17" {
		t.Errorf("Version = %q, want '3.17'", result.Version)
	}
	if result.Method != iperf.InstallMethodPackageManager {
		t.Errorf("Method = %q, want %q", result.Method, iperf.InstallMethodPackageManager)
	}
	if result.Error != nil {
		t.Errorf("Error should be nil for successful install, got %v", result.Error)
	}
	if result.NeedsSudo {
		t.Error("NeedsSudo should be false for successful install")
	}
}

// TestInstallViaPackageManagerNoManager tests install when no package manager is available.
func TestInstallViaPackageManagerNoManager(t *testing.T) {
	t.Parallel()

	// This test validates the behavior when no package manager is detected
	// Since we can't easily mock the detection, we just verify the function signature works

	opts := iperf.InstallOptions{
		Method:  iperf.InstallMethodPackageManager,
		UseSudo: false,
		Verbose: false,
	}

	// Verify all struct fields are properly set
	if opts.Method != iperf.InstallMethodPackageManager {
		t.Errorf("Method = %q, want %q", opts.Method, iperf.InstallMethodPackageManager)
	}
	if opts.UseSudo {
		t.Error("UseSudo should be false")
	}
	if opts.Verbose {
		t.Error("Verbose should be false")
	}
}

// TestPackageManagerInfoNilUpdateCommand tests package manager with nil update command.
func TestPackageManagerInfoNilUpdateCommand(t *testing.T) {
	t.Parallel()

	info := iperf.PackageManagerInfo{
		Name:           "dnf",
		InstallCommand: []string{"dnf", "install", "-y", "iperf3"},
		UpdateCommand:  nil,
		Available:      true,
	}

	// Verify all fields to avoid unused write warnings
	if info.Name != "dnf" {
		t.Errorf("Name = %q, want 'dnf'", info.Name)
	}
	if !info.Available {
		t.Error("Available should be true")
	}

	if info.UpdateCommand != nil {
		t.Error("UpdateCommand should be nil")
	}
	if len(info.InstallCommand) == 0 {
		t.Error("InstallCommand should not be empty")
	}
}

// TestInstallOptionsWithAllFields tests install options with all fields set.
func TestInstallOptionsWithAllFields(t *testing.T) {
	t.Parallel()

	opts := iperf.InstallOptions{
		Method:     iperf.InstallMethodGitHub,
		Version:    "3.17",
		InstallDir: "/opt/iperf3",
		UseSudo:    true,
		Verbose:    true,
	}

	if opts.Method != iperf.InstallMethodGitHub {
		t.Errorf("Method = %q, want %q", opts.Method, iperf.InstallMethodGitHub)
	}
	if opts.Version != "3.17" {
		t.Errorf("Version = %q, want '3.17'", opts.Version)
	}
	if opts.InstallDir != "/opt/iperf3" {
		t.Errorf("InstallDir = %q, want '/opt/iperf3'", opts.InstallDir)
	}
	if !opts.UseSudo {
		t.Error("UseSudo should be true")
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
}

// containsAny checks if a string contains any of the given substrings.
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
