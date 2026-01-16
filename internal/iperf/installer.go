package iperf

//
// This file handles automated detection, download, build, and installation
// of iperf3 from GitHub with OS-specific package manager support.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

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
		{"apt", "apt", []string{"apt", "install", "-y", "iperf3"}, []string{"apt", "update"}},
		{"dnf", "dnf", []string{"dnf", "install", "-y", "iperf3"}, nil},
		{"yum", "yum", []string{"yum", "install", "-y", "iperf3"}, nil},
		{
			"pacman",
			"pacman",
			[]string{"pacman", "-S", "--noconfirm", "iperf3"},
			[]string{"pacman", "-Sy"},
		},
		{"apk", "apk", []string{"apk", "add", "iperf3"}, []string{"apk", "update"}},
		{
			"zypper",
			"zypper",
			[]string{"zypper", "install", "-y", "iperf3"},
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
			InstallCommand: []string{"brew", "install", "iperf3"},
			UpdateCommand:  []string{"brew", "update"},
			Available:      true,
		}
	}
	// MacPorts as fallback
	if _, err := exec.LookPath("port"); err == nil {
		return &PackageManagerInfo{
			Name:           "macports",
			InstallCommand: []string{"port", "install", "iperf3"},
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
			InstallCommand: []string{"choco", "install", "iperf3", "-y"},
			Available:      true,
		}
	}
	// Scoop
	if _, err := exec.LookPath("scoop"); err == nil {
		return &PackageManagerInfo{
			Name:           "scoop",
			InstallCommand: []string{"scoop", "install", "iperf3"},
			Available:      true,
		}
	}
	// winget
	if _, err := exec.LookPath("winget"); err == nil {
		return &PackageManagerInfo{
			Name:           "winget",
			InstallCommand: []string{"winget", "install", "iperf3"},
			Available:      true,
		}
	}
	return nil
}

// GetLatestGitHubRelease fetches the latest iperf3 release info from GitHub.
func GetLatestGitHubRelease() (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), githubAPITimeoutSeconds*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iperfReleasesAPI, http.NoBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "seed-network-tool")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName    string `json:"tag_name"`
		TarballURL string `json:"tarball_url"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&release); decodeErr != nil {
		return "", "", fmt.Errorf("failed to parse release info: %w", decodeErr)
	}

	return strings.TrimPrefix(release.TagName, "v"), release.TarballURL, nil
}

// InstallViaPackageManager attempts to install iperf3 using the system package manager.
func InstallViaPackageManager(opts InstallOptions) *InstallResult {
	pm := DetectPackageManager()
	if pm == nil || !pm.Available {
		return &InstallResult{
			Success: false,
			Error:   errors.New("no package manager detected"),
			Method:  InstallMethodPackageManager,
		}
	}

	logging.GetLogger().Info("Installing iperf3 via package manager", "manager", pm.Name)

	// Run update first if available
	if pm.UpdateCommand != nil {
		updateCmd := pm.UpdateCommand
		if opts.UseSudo && needsSudo(pm.Name) {
			updateCmd = append([]string{"sudo"}, updateCmd...)
		}
		logging.GetLogger().
			Debug("Running package manager update", "command", strings.Join(updateCmd, " "))

		ctx, cancel := context.WithTimeout(context.Background(), packageUpdateTimeoutMinutes*time.Minute)
		//nolint:gosec // G204: commands are from controlled sources
		cmd := exec.CommandContext(ctx, updateCmd[0], updateCmd[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			cancel()
			logging.GetLogger().Warn("Package manager update failed", "error", err)
			// Continue anyway - update failure shouldn't block install
		}
		cancel()
	}

	// Install iperf3
	installCmd := pm.InstallCommand
	if opts.UseSudo && needsSudo(pm.Name) {
		installCmd = append([]string{"sudo"}, installCmd...)
	}

	logging.GetLogger().Info("Running install command", "command", strings.Join(installCmd, " "))

	ctx, cancel := context.WithTimeout(context.Background(), packageInstallTimeoutMinutes*time.Minute)
	defer cancel()

	//nolint:gosec // G204: commands are from controlled sources
	cmd := exec.CommandContext(ctx, installCmd[0], installCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Check if it's a permission error
		if strings.Contains(err.Error(), "permission") || strings.Contains(err.Error(), "EACCES") {
			return &InstallResult{
				Success:     false,
				Error:       errors.New("permission denied - try with sudo"),
				Method:      InstallMethodPackageManager,
				NeedsSudo:   true,
				SudoCommand: "sudo " + strings.Join(pm.InstallCommand, " "),
			}
		}
		return &InstallResult{
			Success: false,
			Error:   fmt.Errorf("installation failed: %w", err),
			Method:  InstallMethodPackageManager,
		}
	}

	// Verify installation
	path, err := exec.LookPath("iperf3")
	if err != nil {
		return &InstallResult{
			Success: false,
			Error:   errors.New("installation succeeded but iperf3 not found in PATH"),
			Method:  InstallMethodPackageManager,
		}
	}

	version, verr := GetVersion()
	if verr != nil {
		version = versionUnknown
	}

	return &InstallResult{
		Success: true,
		Path:    path,
		Version: version,
		Method:  InstallMethodPackageManager,
	}
}

// InstallFromGitHub downloads and builds iperf3 from GitHub source.
func InstallFromGitHub(opts InstallOptions) *InstallResult {
	logging.GetLogger().Info("Installing iperf3 from GitHub source")

	// Get latest release info
	version, tarballURL, err := GetLatestGitHubRelease()
	if err != nil {
		return &InstallResult{
			Success: false,
			Error:   fmt.Errorf("failed to get release info: %w", err),
			Method:  InstallMethodGitHub,
		}
	}

	if opts.Version != "" {
		version = opts.Version
		tarballURL = fmt.Sprintf(
			"https://github.com/esnet/iperf/archive/refs/tags/%s.tar.gz",
			version,
		)
	}

	logging.GetLogger().Info("Downloading iperf3", "version", version, "url", tarballURL)

	// Create temp directory for build
	tempDir, err := os.MkdirTemp("", "iperf3-build-*")
	if err != nil {
		return &InstallResult{
			Success: false,
			Error:   fmt.Errorf("failed to create temp directory: %w", err),
			Method:  InstallMethodGitHub,
		}
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Download tarball
	tarballPath := filepath.Join(tempDir, "iperf3.tar.gz")
	if downloadErr := downloadFile(tarballURL, tarballPath); downloadErr != nil {
		return &InstallResult{
			Success: false,
			Error:   fmt.Errorf("failed to download: %w", downloadErr),
			Method:  InstallMethodGitHub,
		}
	}

	// Extract tarball.
	logging.GetLogger().Info("Extracting source...")
	extractCtx, extractCancel := context.WithTimeout(context.Background(), extractTimeoutMinutes*time.Minute)
	defer extractCancel()
	extractCmd := exec.CommandContext(
		extractCtx,
		"tar",
		"-xzf",
		tarballPath,
		"-C",
		tempDir,
	)
	if extractErr := extractCmd.Run(); extractErr != nil {
		return &InstallResult{
			Success: false,
			Error:   fmt.Errorf("failed to extract: %w", extractErr),
			Method:  InstallMethodGitHub,
		}
	}

	// Find extracted directory
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return &InstallResult{
			Success: false,
			Error:   fmt.Errorf("failed to read temp directory: %w", err),
			Method:  InstallMethodGitHub,
		}
	}

	var sourceDir string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "iperf") {
			sourceDir = filepath.Join(tempDir, entry.Name())
			break
		}
	}
	if sourceDir == "" {
		return &InstallResult{
			Success: false,
			Error:   errors.New("could not find extracted source directory"),
			Method:  InstallMethodGitHub,
		}
	}

	// Build
	logging.GetLogger().Info("Building iperf3...", "sourceDir", sourceDir)
	result := buildIperf3(sourceDir, opts)
	if result != nil {
		return result
	}

	// Install
	logging.GetLogger().Info("Installing iperf3...")
	return installIperf3(sourceDir, opts)
}

func buildIperf3(sourceDir string, opts InstallOptions) *InstallResult {
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	// Run autoreconf if needed
	if _, statErr := os.Stat(filepath.Join(sourceDir, "configure")); os.IsNotExist(statErr) {
		logging.GetLogger().Debug("Running autoreconf...")
		autoreconfCmd := exec.CommandContext(ctx, "autoreconf", "-i")
		autoreconfCmd.Dir = sourceDir
		autoreconfCmd.Stdout = os.Stdout
		autoreconfCmd.Stderr = os.Stderr
		if autoErr := autoreconfCmd.Run(); autoErr != nil {
			// Try bootstrap.sh as fallback
			bootstrapCmd := exec.CommandContext(ctx, "./bootstrap.sh")
			bootstrapCmd.Dir = sourceDir
			bootstrapCmd.Stdout = os.Stdout
			bootstrapCmd.Stderr = os.Stderr
			if bootstrapErr := bootstrapCmd.Run(); bootstrapErr != nil {
				return &InstallResult{
					Success: false,
					Error:   fmt.Errorf("failed to run autoreconf/bootstrap: %w", bootstrapErr),
					Method:  InstallMethodGitHub,
				}
			}
		}
	}

	// Configure.
	logging.GetLogger().Debug("Running configure...")
	var configureCmd *exec.Cmd
	if opts.InstallDir != "" {
		// Sanitize install directory - filepath.Clean normalizes path
		// Also validate it's an absolute path to prevent path traversal
		cleanDir := filepath.Clean(opts.InstallDir)
		if !filepath.IsAbs(cleanDir) {
			absDir, err := filepath.Abs(cleanDir)
			if err != nil {
				return &InstallResult{
					Success: false,
					Error:   fmt.Errorf("invalid install directory: %w", err),
				}
			}
			cleanDir = absDir
		}
		// #nosec G204 -- cleanDir is sanitized via filepath.Clean/Abs, configure is a trusted script
		configureCmd = exec.CommandContext(
			ctx,
			"./configure",
			"--prefix="+cleanDir,
		)
	} else {
		configureCmd = exec.CommandContext(ctx, "./configure")
	}
	configureCmd.Dir = sourceDir
	configureCmd.Stdout = os.Stdout
	configureCmd.Stderr = os.Stderr
	if err := configureCmd.Run(); err != nil {
		return &InstallResult{
			Success: false,
			Error:   fmt.Errorf("configure failed: %w", err),
			Method:  InstallMethodGitHub,
		}
	}

	// Make
	logging.GetLogger().Debug("Running make...")
	makeCmd := exec.CommandContext(ctx, "make", "-j4")
	makeCmd.Dir = sourceDir
	makeCmd.Stdout = os.Stdout
	makeCmd.Stderr = os.Stderr
	if err := makeCmd.Run(); err != nil {
		return &InstallResult{
			Success: false,
			Error:   fmt.Errorf("make failed: %w", err),
			Method:  InstallMethodGitHub,
		}
	}

	return nil // Success, continue to install
}

func installIperf3(sourceDir string, opts InstallOptions) *InstallResult {
	ctx, cancel := context.WithTimeout(context.Background(), makeInstallTimeoutMinutes*time.Minute)
	defer cancel()

	// Make install
	var installCmd *exec.Cmd
	if opts.UseSudo {
		installCmd = exec.CommandContext(ctx, "sudo", "make", "install")
	} else {
		installCmd = exec.CommandContext(ctx, "make", "install")
	}
	installCmd.Dir = sourceDir
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		if strings.Contains(err.Error(), "permission") {
			return &InstallResult{
				Success:     false,
				Error:       errors.New("permission denied - try with sudo"),
				Method:      InstallMethodGitHub,
				NeedsSudo:   true,
				SudoCommand: fmt.Sprintf("cd %s && sudo make install", sourceDir),
			}
		}
		return &InstallResult{
			Success: false,
			Error:   fmt.Errorf("make install failed: %w", err),
			Method:  InstallMethodGitHub,
		}
	}

	// Verify installation
	path, err := exec.LookPath("iperf3")
	if err != nil {
		// Check if installed to custom prefix
		if opts.InstallDir != "" {
			customPath := filepath.Join(opts.InstallDir, "bin", "iperf3")
			if _, statErr := os.Stat(customPath); statErr == nil {
				path = customPath
			}
		}
		if path == "" {
			return &InstallResult{
				Success: false,
				Error:   errors.New("installation succeeded but iperf3 not found"),
				Method:  InstallMethodGitHub,
			}
		}
	}

	version, verr := GetVersion()
	if verr != nil {
		version = versionUnknown
	}

	return &InstallResult{
		Success: true,
		Path:    path,
		Version: version,
		Method:  InstallMethodGitHub,
	}
}

func downloadFile(url, destPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "seed-network-tool")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
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

// AutoInstall attempts to install iperf3 automatically with the best available method.
// It tries package manager first (faster, more reliable), then falls back to GitHub.
func AutoInstall(useSudo, verbose bool) *InstallResult {
	opts := InstallOptions{
		UseSudo: useSudo,
		Verbose: verbose,
	}

	// Try package manager first (faster and more reliable)
	pm := DetectPackageManager()
	if pm != nil && pm.Available {
		logging.GetLogger().Info("Attempting installation via package manager", "manager", pm.Name)
		result := InstallViaPackageManager(opts)
		if result.Success {
			return result
		}
		logging.GetLogger().
			Warn("Package manager installation failed, trying GitHub", "error", result.Error)
	}

	// Fall back to GitHub
	opts.Method = InstallMethodGitHub
	return InstallFromGitHub(opts)
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
