package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/version"
)

const (
	// githubAPIBase is the base URL for GitHub API.
	githubAPIBase = "https://api.github.com"
	// defaultTimeout is the default HTTP timeout for API requests.
	defaultTimeout = 30 * time.Second
	// maxErrorBodySize is the maximum size for error body reads.
	maxErrorBodySize = 1024
)

// Checker handles checking for updates via GitHub releases API.
type Checker struct {
	config     UpdateConfig
	httpClient *http.Client
	lastCheck  time.Time
	lastResult *UpdateInfo
}

// NewChecker creates a new update checker with the given configuration.
func NewChecker(config UpdateConfig) *Checker {
	return &Checker{
		config: config,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// CheckForUpdate checks GitHub releases for a newer version.
func (c *Checker) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	currentVersion := version.GetVersion()

	// Build the releases URL
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest",
		githubAPIBase, c.config.GitHubOwner, c.config.GitHubRepo)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("Seed/%s", currentVersion))

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No releases yet
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse release
	var release Release
	if decodeErr := json.NewDecoder(resp.Body).Decode(&release); decodeErr != nil {
		return nil, fmt.Errorf("decode release: %w", decodeErr)
	}

	// Skip pre-releases unless configured to include them
	if release.Prerelease && !c.config.IncludePrerelease {
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
		}, nil
	}

	// Compare versions
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersionClean := strings.TrimPrefix(currentVersion, "v")

	// Simple version comparison (semver-ish)
	isNewer := compareVersions(latestVersion, currentVersionClean) > 0

	info := &UpdateInfo{
		Available:       isNewer,
		CurrentVersion:  currentVersion,
		LatestVersion:   release.TagName,
		ReleaseNotes:    release.Body,
		ReleaseURL:      release.HTMLURL,
		PublishedAt:     release.PublishedAt,
		RequiresRestart: true, // Binary updates always require restart
	}

	// Find the appropriate asset for this platform
	if isNewer {
		asset := c.findAssetForPlatform(release.Assets)
		if asset != nil {
			info.DownloadURL = asset.BrowserDownloadURL
			info.DownloadSize = asset.Size
			info.CanAutoUpdate = true

			// Look for checksum file
			checksumAsset := c.findChecksumAsset(release.Assets)
			if checksumAsset != nil {
				info.ChecksumURL = checksumAsset.BrowserDownloadURL
			}
		}
	}

	c.lastCheck = time.Now()
	c.lastResult = info

	return info, nil
}

// findAssetForPlatform finds the appropriate binary asset for the current platform.
func (c *Checker) findAssetForPlatform(assets []Asset) *Asset {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map common arch names
	archNames := map[string][]string{
		"amd64": {"amd64", "x86_64", "x64"},
		"arm64": {"arm64", "aarch64"},
		"386":   {"386", "i386", "x86"},
	}

	archVariants := archNames[arch]
	if archVariants == nil {
		archVariants = []string{arch}
	}

	// Look for matching asset
	for i := range assets {
		asset := &assets[i]
		name := strings.ToLower(asset.Name)

		// Skip checksum files
		if strings.Contains(name, "sha256") || strings.Contains(name, "checksum") {
			continue
		}

		// Check OS match
		if !strings.Contains(name, os) {
			continue
		}

		// Check arch match
		for _, archName := range archVariants {
			if strings.Contains(name, strings.ToLower(archName)) {
				return asset
			}
		}
	}

	return nil
}

// findChecksumAsset finds the SHA256 checksum file for verification.
func (c *Checker) findChecksumAsset(assets []Asset) *Asset {
	for i := range assets {
		asset := &assets[i]
		name := strings.ToLower(asset.Name)

		if strings.Contains(name, "sha256") || strings.HasSuffix(name, ".sha256") {
			return asset
		}
	}
	return nil
}

// GetLastCheckResult returns the result of the last update check.
func (c *Checker) GetLastCheckResult() *UpdateInfo {
	return c.lastResult
}

// GetLastCheckTime returns when the last update check occurred.
func (c *Checker) GetLastCheckTime() time.Time {
	return c.lastCheck
}

// compareVersions compares two semver-like version strings.
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal.
func compareVersions(v1, v2 string) int {
	// Clean up versions
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Remove any suffix like -dirty, -alpha, etc for comparison
	v1Base := strings.Split(v1, "-")[0]
	v2Base := strings.Split(v2, "-")[0]

	parts1 := strings.Split(v1Base, ".")
	parts2 := strings.Split(v2Base, ".")

	// Compare major.minor.patch
	for i := range semverParts {
		var p1, p2 int
		if i < len(parts1) {
			p1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			p2, _ = strconv.Atoi(parts2[i])
		}

		if p1 > p2 {
			return 1
		}
		if p1 < p2 {
			return -1
		}
	}

	return 0
}
