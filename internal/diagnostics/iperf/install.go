package iperf

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
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

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
	path, err := exec.LookPath(iperfPackageName)
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
	path, err := exec.LookPath(iperfPackageName)
	if err != nil {
		// Check if installed to custom prefix
		if opts.InstallDir != "" {
			customPath := filepath.Join(opts.InstallDir, "bin", iperfPackageName)
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
