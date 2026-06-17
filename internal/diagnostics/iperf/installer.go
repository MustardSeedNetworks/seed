package iperf

//
// This file handles automated detection, download, build, and installation
// of iperf3 from GitHub with OS-specific package manager support.

import (
	"os/exec"
	"runtime"
	"time"
)

// iperfPackageName is the canonical package name for iperf3 across all
// supported OS package managers (apt/dnf/yum/pacman/apk/zypper/brew/etc).
const iperfPackageName = "iperf3"

const (
	// GitHub API endpoint for iperf3 releases.
	iperfReleasesAPI = "https://api.github.com/repos/esnet/iperf/releases/latest"

	// Download timeout.
	downloadTimeout = 5 * time.Minute

	// Build timeout.
	buildTimeout = 10 * time.Minute

	// githubAPITimeoutSeconds is the timeout for GitHub API requests to fetch release info.
	githubAPITimeoutSeconds = 30

	// packageUpdateTimeoutMinutes is the timeout for package manager update operations.
	packageUpdateTimeoutMinutes = 2

	// packageInstallTimeoutMinutes is the timeout for package manager install operations.
	packageInstallTimeoutMinutes = 5

	// extractTimeoutMinutes is the timeout for extracting downloaded tarballs.
	extractTimeoutMinutes = 2

	// makeInstallTimeoutMinutes is the timeout for make install operations.
	makeInstallTimeoutMinutes = 2

	// OS constants for [runtime.GOOS] checks.
	osLinux   = "linux"
	osDarwin  = "darwin"
	osWindows = "windows"

	// versionUnknown is returned when version detection fails.
	versionUnknown = "unknown"
)

// InstallMethod represents how iperf3 should be installed.
type InstallMethod string

// InstallMethod values for iperf3 installation.
const (
	InstallMethodPackageManager InstallMethod = "package_manager"
	InstallMethodGitHub         InstallMethod = "github"
	InstallMethodManual         InstallMethod = "manual"
)

// InstallOptions configures the installation process.
type InstallOptions struct {
	// Method specifies how to install (package manager, github, etc.)
	Method InstallMethod

	// Version to install (empty = latest)
	Version string

	// InstallDir where to install (empty = system default)
	InstallDir string

	// UseSudo whether to use sudo for system installation
	UseSudo bool

	// Verbose enables detailed output
	Verbose bool
}

// InstallResult contains the result of an installation attempt.
type InstallResult struct {
	Success     bool
	Path        string
	Version     string
	Method      InstallMethod
	Error       error
	NeedsSudo   bool
	SudoCommand string
}

// PackageManagerInfo contains info about available package managers.
type PackageManagerInfo struct {
	Name           string
	InstallCommand []string
	UpdateCommand  []string
	Available      bool
}

// DetectPackageManager returns info about the system's package manager.
func DetectPackageManager() *PackageManagerInfo {
	switch runtime.GOOS {
	case osLinux:
		return detectLinuxPackageManager()
	case osDarwin:
		return detectMacOSPackageManager()
	case osWindows:
		return detectWindowsPackageManager()
	default:
		return nil
	}
}

func detectLinuxPackageManager() *PackageManagerInfo {
	// Check in order of preference
	managers := []struct {
		name    string
		check   string
		install []string
		update  []string
	}{
		{"apt", "apt", []string{"apt", "install", "-y", iperfPackageName}, []string{"apt", "update"}},
		{"dnf", "dnf", []string{"dnf", "install", "-y", iperfPackageName}, nil},
		{"yum", "yum", []string{"yum", "install", "-y", iperfPackageName}, nil},
		{
			"pacman",
			"pacman",
			[]string{"pacman", "-S", "--noconfirm", iperfPackageName},
			[]string{"pacman", "-Sy"},
		},
		{"apk", "apk", []string{"apk", "add", iperfPackageName}, []string{"apk", "update"}},
		{
			"zypper",
			"zypper",
			[]string{"zypper", "install", "-y", iperfPackageName},
			[]string{"zypper", "refresh"},
		},
	}

	for _, m := range managers {
		if _, err := exec.LookPath(m.check); err == nil {
			return &PackageManagerInfo{
				Name:           m.name,
				InstallCommand: m.install,
				UpdateCommand:  m.update,
				Available:      true,
			}
		}
	}
	return nil
}

func detectMacOSPackageManager() *PackageManagerInfo {
	// Prefer Homebrew
	if _, err := exec.LookPath("brew"); err == nil {
		return &PackageManagerInfo{
			Name:           "homebrew",
			InstallCommand: []string{"brew", "install", iperfPackageName},
			UpdateCommand:  []string{"brew", "update"},
			Available:      true,
		}
	}
	// MacPorts as fallback
	if _, err := exec.LookPath("port"); err == nil {
		return &PackageManagerInfo{
			Name:           "macports",
			InstallCommand: []string{"port", "install", iperfPackageName},
			UpdateCommand:  []string{"port", "selfupdate"},
			Available:      true,
		}
	}
	return nil
}

func detectWindowsPackageManager() *PackageManagerInfo {
	// Chocolatey
	if _, err := exec.LookPath("choco"); err == nil {
		return &PackageManagerInfo{
			Name:           "chocolatey",
			InstallCommand: []string{"choco", "install", iperfPackageName, "-y"},
			Available:      true,
		}
	}
	// Scoop
	if _, err := exec.LookPath("scoop"); err == nil {
		return &PackageManagerInfo{
			Name:           "scoop",
			InstallCommand: []string{"scoop", "install", iperfPackageName},
			Available:      true,
		}
	}
	// winget
	if _, err := exec.LookPath("winget"); err == nil {
		return &PackageManagerInfo{
			Name:           "winget",
			InstallCommand: []string{"winget", "install", iperfPackageName},
			Available:      true,
		}
	}
	return nil
}

func needsSudo(packageManager string) bool {
	// Homebrew and scoop don't need sudo.
	if packageManager == "homebrew" || packageManager == "scoop" {
		return false
	}
	// Most Linux package managers need sudo.
	if runtime.GOOS == osLinux {
		return true
	}
	return false
}

// CheckBuildDependencies verifies that required build tools are available.
func CheckBuildDependencies() []string {
	var missing []string

	required := []string{"make", "gcc", "tar"}
	optional := []string{"autoreconf", "autoconf", "automake"}

	for _, tool := range required {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}

	// Check for autoreconf or autoconf
	hasAutoreconf := false
	for _, tool := range optional {
		if _, err := exec.LookPath(tool); err == nil {
			hasAutoreconf = true
			break
		}
	}
	if !hasAutoreconf {
		missing = append(missing, "autoconf")
	}

	return missing
}

// GetBuildDependencyInstallCommand returns the command to install build dependencies.
func GetBuildDependencyInstallCommand() string {
	switch runtime.GOOS {
	case osLinux:
		if _, err := exec.LookPath("apt"); err == nil {
			return "sudo apt install -y build-essential autoconf automake libtool"
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			return "sudo dnf install -y gcc make autoconf automake libtool"
		}
		if _, err := exec.LookPath("pacman"); err == nil {
			return "sudo pacman -S base-devel autoconf automake libtool"
		}
		return "Install: gcc, make, autoconf, automake, libtool"
	case osDarwin:
		return "xcode-select --install && brew install autoconf automake libtool"
	default:
		return "Install: gcc, make, autoconf, automake, libtool"
	}
}
