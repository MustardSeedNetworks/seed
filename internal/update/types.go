package update

import "time"

// Configuration and operation constants.
const (
	// defaultCheckInterval is the default duration between automatic update checks.
	defaultCheckInterval = 24 * time.Hour
	// semverParts is the number of parts in a semantic version (major.minor.patch).
	semverParts = 3
	// progressComplete represents 100% completion.
	progressComplete = 100
)

// Release represents a GitHub release with relevant metadata.
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Assets      []Asset   `json:"assets"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
}

// Asset represents a downloadable file attached to a release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// UpdateInfo contains information about an available update.
//
//nolint:revive // Type name stuttering is acceptable for API clarity.
type UpdateInfo struct {
	Available       bool      `json:"available"`
	CurrentVersion  string    `json:"currentVersion"`
	LatestVersion   string    `json:"latestVersion"`
	ReleaseNotes    string    `json:"releaseNotes,omitempty"`
	ReleaseURL      string    `json:"releaseUrl,omitempty"`
	PublishedAt     time.Time `json:"publishedAt,omitzero"`
	DownloadURL     string    `json:"downloadUrl,omitempty"`
	DownloadSize    int64     `json:"downloadSize,omitempty"`
	ChecksumURL     string    `json:"checksumUrl,omitempty"`
	CanAutoUpdate   bool      `json:"canAutoUpdate"`
	RequiresRestart bool      `json:"requiresRestart"`
}

// UpdateStatus represents the current state of an update operation.
//
//nolint:revive // Type name stuttering is acceptable for API clarity.
type UpdateStatus struct {
	State           UpdateState `json:"state"`
	Progress        float64     `json:"progress"` // 0-100
	Message         string      `json:"message,omitempty"`
	Error           string      `json:"error,omitempty"`
	DownloadedBytes int64       `json:"downloadedBytes,omitempty"`
	TotalBytes      int64       `json:"totalBytes,omitempty"`
	StartedAt       time.Time   `json:"startedAt,omitzero"`
}

// UpdateState represents the state of an update operation.
//
//nolint:revive // Type name stuttering is acceptable for API clarity.
type UpdateState string

const (
	StateIdle        UpdateState = "idle"
	StateChecking    UpdateState = "checking"
	StateDownloading UpdateState = "downloading"
	StateVerifying   UpdateState = "verifying"
	StateApplying    UpdateState = "applying"
	StateRestarting  UpdateState = "restarting"
	StateComplete    UpdateState = "complete"
	StateFailed      UpdateState = "failed"
	StateRolledBack  UpdateState = "rolled_back"
)

// UpdateConfig holds configuration for the update system.
//
//nolint:revive // Type name stuttering is acceptable for API clarity.
type UpdateConfig struct {
	// Enabled determines if update checking is enabled
	Enabled bool `json:"enabled"`
	// CheckInterval is the duration between automatic update checks
	CheckInterval time.Duration `json:"checkInterval"`
	// AutoDownload enables automatic downloading of updates
	AutoDownload bool `json:"autoDownload"`
	// AutoApply enables automatic application of updates (requires restart)
	AutoApply bool `json:"autoApply"`
	// IncludePrerelease includes pre-release versions in update checks
	IncludePrerelease bool `json:"includePrerelease"`
	// GitHubOwner is the GitHub repository owner
	GitHubOwner string `json:"githubOwner"`
	// GitHubRepo is the GitHub repository name
	GitHubRepo string `json:"githubRepo"`
}

// DefaultConfig returns the default update configuration.
func DefaultConfig() UpdateConfig {
	return UpdateConfig{
		Enabled:           true,
		CheckInterval:     defaultCheckInterval,
		AutoDownload:      false,
		AutoApply:         false,
		IncludePrerelease: false,
		GitHubOwner:       "MustardSeedNetworks",
		GitHubRepo:        "seed",
	}
}
