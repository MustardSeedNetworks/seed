package vuln

// Vulnerability scanning integrates with the NVD (National Vulnerability Database) to identify
// CVEs (Common Vulnerabilities and Exposures) affecting detected devices. Maps software versions
// and OS information to potential vulnerabilities with CVSS severity scoring and advisory references.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Vulnerability scanner configuration constants.
const (
	vulnUpdateIntervalS = 86400 // CVE update interval in seconds (24 hours)
	vulnMaxConcurrent   = 5     // Default max concurrent vulnerability checks
)

// Severity ranking constants for filtering.
const (
	severityRankLow      = 1 // Low severity rank
	severityRankMedium   = 2 // Medium severity rank
	severityRankHigh     = 3 // High severity rank
	severityRankCritical = 4 // Critical severity rank
)

// Vulnerability and DeviceVulnerabilities are the kernel result types (see
// aliases.go); they are defined in package discovery because DiscoveredDevice
// embeds them.

// VulnerabilityScannerConfig holds configuration for vulnerability scanning.
type VulnerabilityScannerConfig struct {
	Enabled           bool   `json:"enabled"`           // Enable vulnerability scanning
	CVEDatabase       string `json:"cveDatabase"`       // "nvd" or "local"
	NVDAPIKey         string `json:"nvdApiKey"`         // Optional NVD API key for higher rate limits
	LocalDBPath       string `json:"localDbPath"`       // Path to local CVE database JSON file (for "local" provider)
	UpdateInterval    int    `json:"updateInterval"`    // Seconds between CVE database updates
	SeverityThreshold string `json:"severityThreshold"` // Minimum severity to report: "low", "medium", "high", "critical"
	MaxConcurrent     int    `json:"maxConcurrent"`     // Max concurrent vulnerability checks
	KEVEnabled        bool   `json:"kevEnabled"`        // Enable CISA KEV catalog enrichment
	KEVCachePath      string `json:"kevCachePath"`      // Path to cache KEV catalog (empty for no caching)
}

// VulnerabilityScanner scans devices for known vulnerabilities.
type VulnerabilityScanner struct {
	mu            sync.RWMutex
	config        *VulnerabilityScannerConfig
	cveProvider   CVEProvider
	kevProvider   *KEVProvider                      // CISA KEV catalog for enrichment
	deviceResults map[string]*DeviceVulnerabilities // key is device IP
	lastUpdate    time.Time
	running       bool
	stopCh        chan struct{} // Channel to signal goroutine shutdown (fixes #820)
}

// CVEProvider is an interface for CVE data sources (NVD API, local DB, etc).
type CVEProvider interface {
	// SearchByCPE searches for vulnerabilities affecting a CPE string
	SearchByCPE(ctx context.Context, cpe string) ([]Vulnerability, error)

	// SearchByProduct searches for vulnerabilities by vendor/product/version
	SearchByProduct(ctx context.Context, vendor, product, version string) ([]Vulnerability, error)

	// UpdateDatabase updates the local CVE database (if applicable)
	UpdateDatabase(ctx context.Context) error

	// GetLastUpdate returns the last database update time
	GetLastUpdate() time.Time
}

// NewVulnerabilityScanner creates a new vulnerability scanner.
func NewVulnerabilityScanner(config *VulnerabilityScannerConfig) (*VulnerabilityScanner, error) {
	if config == nil {
		config = &VulnerabilityScannerConfig{
			Enabled:           false,
			CVEDatabase:       "nvd",
			UpdateInterval:    vulnUpdateIntervalS,
			SeverityThreshold: "medium",
			MaxConcurrent:     vulnMaxConcurrent,
			KEVEnabled:        true, // KEV enrichment enabled by default
		}
	}

	var provider CVEProvider
	var err error

	switch strings.ToLower(config.CVEDatabase) {
	case "nvd":
		provider, err = NewNVDProvider(config.NVDAPIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create NVD provider: %w", err)
		}
	case "local":
		// Issue #606: Local CVE database provider implemented
		provider, err = NewLocalProvider(config.LocalDBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create local CVE provider: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported CVE database: %s", config.CVEDatabase)
	}

	// Create KEV provider for enrichment (optional but recommended)
	var kevProvider *KEVProvider
	if config.KEVEnabled {
		kevProvider, err = NewKEVProvider(config.KEVCachePath)
		if err != nil {
			// KEV is optional - log warning but don't fail
			logging.GetLogger().Warn(
				"Failed to create KEV provider, CVEs won't have exploitation context",
				"error",
				err,
			)
		}
	}

	return &VulnerabilityScanner{
		config:        config,
		cveProvider:   provider,
		kevProvider:   kevProvider,
		deviceResults: make(map[string]*DeviceVulnerabilities),
	}, nil
}

// Start begins vulnerability scanning.
func (vs *VulnerabilityScanner) Start(ctx context.Context) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.running {
		return errors.New("vulnerability scanner already running")
	}

	vs.running = true
	vs.stopCh = make(chan struct{}) // Create stop channel (fixes #820)
	logging.GetLogger().InfoContext(ctx, "Vulnerability scanner started")

	// Start background update routine
	go vs.updateRoutine(ctx)

	return nil
}

// Stop stops the vulnerability scanner.
func (vs *VulnerabilityScanner) Stop() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if !vs.running {
		return nil
	}

	// Signal update goroutine to stop (fixes #820)
	if vs.stopCh != nil {
		close(vs.stopCh)
		vs.stopCh = nil
	}

	vs.running = false
	logging.GetLogger().Info("Vulnerability scanner stopped")
	return nil
}

// IsRunning returns whether the scanner is running.
func (vs *VulnerabilityScanner) IsRunning() bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.running
}

// ScanDevice scans a specific device for vulnerabilities.
func (vs *VulnerabilityScanner) ScanDevice(
	ctx context.Context,
	device *DiscoveredDevice,
) (*DeviceVulnerabilities, error) {
	if !vs.config.Enabled {
		return nil, errors.New("vulnerability scanning is disabled")
	}

	result := &DeviceVulnerabilities{
		DeviceIP: device.IP,
		MAC:      device.MAC,
		Hostname: device.Hostname,
		Vendor:   device.Vendor,
		ScanTime: time.Now(),
	}

	// Extract product and version from device
	// This is a simplified version - real implementation would need
	// more sophisticated version extraction from device fingerprinting
	product, version := vs.extractProductVersion(device)
	if product == "" || version == "" {
		result.Error = "Unable to extract product/version information"
		return result, nil
	}

	result.Product = product
	result.Version = version

	// Search for vulnerabilities
	vulns, err := vs.cveProvider.SearchByProduct(ctx, device.Vendor, product, version)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// Enrich with CISA KEV data (marks actively exploited CVEs)
	if vs.kevProvider != nil {
		vulns = vs.kevProvider.EnrichVulnerabilities(vulns)
	}

	// Filter by severity threshold
	result.Vulnerabilities = vs.filterBySeverity(vulns)

	// Store result
	vs.mu.Lock()
	vs.deviceResults[device.IP] = result
	vs.mu.Unlock()

	return result, nil
}

// GetDeviceVulnerabilities returns cached vulnerability scan results for a device.
func (vs *VulnerabilityScanner) GetDeviceVulnerabilities(deviceIP string) *DeviceVulnerabilities {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.deviceResults[deviceIP]
}

// GetAllVulnerabilities returns all cached vulnerability scan results.
func (vs *VulnerabilityScanner) GetAllVulnerabilities() []*DeviceVulnerabilities {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	results := make([]*DeviceVulnerabilities, 0, len(vs.deviceResults))
	for _, result := range vs.deviceResults {
		results = append(results, result)
	}
	return results
}

// ClearResults clears all cached vulnerability results.
func (vs *VulnerabilityScanner) ClearResults() {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.deviceResults = make(map[string]*DeviceVulnerabilities)
}

// extractProductVersion attempts to extract product name and version from device
// Uses trust hierarchy: CDP/EDP (high) → LLDP (high) → Service banners (medium) → OS guess (low).
func (vs *VulnerabilityScanner) extractProductVersion(device *DiscoveredDevice) (string, string) {
	if p, v := vs.tryExtractFromDiscoveryProtocols(device); v != "" {
		return p, v
	}
	if p, v := vs.tryExtractFromProfile(device); v != "" {
		return p, v
	}
	if device.OSGuess != "" {
		return parseOSGuess(device.OSGuess)
	}
	return "", ""
}

// tryExtractFromDiscoveryProtocols tries to extract version from CDP/EDP/SNMP/LLDP.
func (*VulnerabilityScanner) tryExtractFromDiscoveryProtocols(
	device *DiscoveredDevice,
) (string, string) {
	if device.CDPInfo != nil && device.CDPInfo.SoftwareVersion != "" {
		if p, v := parseCDPVersion(device.CDPInfo.Platform, device.CDPInfo.SoftwareVersion); v != "" {
			return p, v
		}
	}
	if device.EDPInfo != nil && device.EDPInfo.SoftwareVersion != "" {
		if p, v := parseEDPVersion(device.EDPInfo.SoftwareVersion); v != "" {
			return p, v
		}
	}
	if device.Profile != nil && device.Profile.SNMPInfo != nil &&
		device.Profile.SNMPInfo.SysDescr != "" {
		if p, v := parseSNMPSysDescr(device.Profile.SNMPInfo.SysDescr); v != "" {
			return p, v
		}
	}
	if device.LLDPInfo != nil && device.LLDPInfo.SystemDescription != "" {
		if p, v := parseLLDPDescription(device.LLDPInfo.SystemDescription); v != "" {
			return p, v
		}
	}
	return "", ""
}

// tryExtractFromProfile tries to extract version from service banners and HTTP.
func (*VulnerabilityScanner) tryExtractFromProfile(device *DiscoveredDevice) (string, string) {
	if device.Profile == nil {
		return "", ""
	}
	for _, port := range device.Profile.OpenPorts {
		if port.Banner != "" {
			if p, v := parseServiceBanner(port.Banner); v != "" {
				return p, v
			}
		}
	}
	if device.Profile.HTTPInfo != nil && device.Profile.HTTPInfo.Server != "" {
		return parseHTTPServer(device.Profile.HTTPInfo.Server)
	}
	return "", ""
}

// filterBySeverity filters vulnerabilities by configured severity threshold.
func (vs *VulnerabilityScanner) filterBySeverity(vulns []Vulnerability) []Vulnerability {
	threshold := strings.ToLower(vs.config.SeverityThreshold)
	if threshold == "" || threshold == "low" {
		return vulns
	}

	severityRank := map[string]int{
		"low":      severityRankLow,
		"medium":   severityRankMedium,
		"high":     severityRankHigh,
		"critical": severityRankCritical,
	}

	minRank := severityRank[threshold]
	filtered := make([]Vulnerability, 0)

	for i := range vulns {
		rank := severityRank[strings.ToLower(vulns[i].Severity)]
		if rank >= minRank {
			filtered = append(filtered, vulns[i])
		}
	}

	return filtered
}

// updateRoutine periodically updates the CVE database.
func (vs *VulnerabilityScanner) updateRoutine(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(vs.config.UpdateInterval) * time.Second)
	defer ticker.Stop()

	// Capture stopCh locally under lock (fixes #820)
	vs.mu.RLock()
	stopCh := vs.stopCh
	vs.mu.RUnlock()

	if stopCh == nil {
		return
	}

	// Update on startup
	if err := vs.updateDatabase(ctx); err != nil {
		logging.GetLogger().ErrorContext(ctx, "Failed to update CVE database", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		case <-ticker.C:
			if !vs.IsRunning() {
				return
			}
			if err := vs.updateDatabase(ctx); err != nil {
				logging.GetLogger().ErrorContext(ctx, "Failed to update CVE database", "error", err)
			}
		}
	}
}

// updateDatabase updates the CVE database from the provider.
func (vs *VulnerabilityScanner) updateDatabase(ctx context.Context) error {
	logging.GetLogger().InfoContext(ctx, "Updating CVE database...")
	if err := vs.cveProvider.UpdateDatabase(ctx); err != nil {
		return err
	}

	// Also update KEV catalog if enabled
	if vs.kevProvider != nil {
		if err := vs.kevProvider.UpdateDatabase(ctx); err != nil {
			logging.GetLogger().WarnContext(ctx, "Failed to update KEV catalog", "error", err)
			// Don't fail - KEV is supplementary
		}
	}

	vs.mu.Lock()
	vs.lastUpdate = time.Now()
	vs.mu.Unlock()

	logging.GetLogger().InfoContext(ctx, "CVE database updated successfully")
	return nil
}

// GetConfig returns the scanner configuration.
func (vs *VulnerabilityScanner) GetConfig() *VulnerabilityScannerConfig {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	// Return a copy
	config := *vs.config
	return &config
}

// GetStats returns statistics about the vulnerability scanner.
func (vs *VulnerabilityScanner) GetStats() map[string]any {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	totalVulns := 0
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0
	activelyExploited := 0
	ransomwareRelated := 0

	for _, result := range vs.deviceResults {
		totalVulns += len(result.Vulnerabilities)
		for i := range result.Vulnerabilities {
			vuln := &result.Vulnerabilities[i]
			switch strings.ToLower(vuln.Severity) {
			case "critical":
				criticalCount++
			case "high":
				highCount++
			case "medium":
				mediumCount++
			case "low":
				lowCount++
			}
			if vuln.ActivelyExploited {
				activelyExploited++
			}
			if vuln.RansomwareRelated {
				ransomwareRelated++
			}
		}
	}

	stats := map[string]any{
		"enabled":           vs.config.Enabled,
		"running":           vs.running,
		"devicesScanned":    len(vs.deviceResults),
		"totalVulns":        totalVulns,
		"criticalCount":     criticalCount,
		"highCount":         highCount,
		"mediumCount":       mediumCount,
		"lowCount":          lowCount,
		"activelyExploited": activelyExploited,
		"ransomwareRelated": ransomwareRelated,
		"lastUpdate":        vs.lastUpdate,
		"cveDatabase":       vs.config.CVEDatabase,
		"kevEnabled":        vs.config.KEVEnabled,
	}

	// Add KEV catalog stats if available
	if vs.kevProvider != nil {
		stats["kevStats"] = vs.kevProvider.GetCatalogStats()
	}

	return stats
}
