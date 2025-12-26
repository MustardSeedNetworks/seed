// Package discovery provides CVE (Common Vulnerabilities and Exposures) scanning.
//
// This file integrates with the CISA KEV (Known Exploited Vulnerabilities) catalog
// to identify vulnerabilities that are actively being exploited in the wild.
//
// The KEV catalog is a critical prioritization tool - any CVE in this list should
// be treated as HIGH priority regardless of CVSS score, as it represents a real-world
// exploitation risk.
//
// Features:
//   - Download and cache the KEV catalog locally
//   - Enrich NVD results with "actively exploited" flag
//   - Prioritize remediation based on real-world exploitation
//
// See: https://www.cisa.gov/known-exploited-vulnerabilities-catalog
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	kevCatalogURL      = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"
	kevCacheFile       = "kev_catalog.json"
	kevMaxAge          = 24 * time.Hour // Refresh daily
	kevRansomwareKnown = "known"        // KEV ransomware campaign status
	kevCacheFilePerms  = 0o600          // Cache file permissions
	kevCacheDirPerms   = 0o750          // Cache directory permissions
)

// KEVCatalog represents the CISA Known Exploited Vulnerabilities catalog.
type KEVCatalog struct {
	Title           string     `json:"title"`
	CatalogVersion  string     `json:"catalogVersion"`
	DateReleased    string     `json:"dateReleased"`
	Count           int        `json:"count"`
	Vulnerabilities []KEVEntry `json:"vulnerabilities"`
}

// KEVEntry represents a single vulnerability in the KEV catalog.
type KEVEntry struct {
	CVEID                   string `json:"cveID"`
	VendorProject           string `json:"vendorProject"`
	Product                 string `json:"product"`
	VulnerabilityName       string `json:"vulnerabilityName"`
	DateAdded               string `json:"dateAdded"`
	ShortDescription        string `json:"shortDescription"`
	RequiredAction          string `json:"requiredAction"`
	DueDate                 string `json:"dueDate"`
	KnownRansomwareCampaign string `json:"knownRansomwareCampaignUse"` // "Known" or "Unknown"
	Notes                   string `json:"notes"`
}

// KEVProvider provides CISA KEV catalog data as a CVE enrichment source.
// It's designed to work alongside NVD - use it to mark CVEs as "actively exploited".
type KEVProvider struct {
	mu         sync.RWMutex
	catalog    *KEVCatalog
	cveIndex   map[string]*KEVEntry // CVE ID -> KEV entry for fast lookup
	cachePath  string
	lastUpdate time.Time
	client     *http.Client
}

// NewKEVProvider creates a new CISA KEV provider.
// cachePath specifies where to store the cached catalog (empty for no caching).
func NewKEVProvider(cachePath string) (*KEVProvider, error) {
	provider := &KEVProvider{
		cveIndex:  make(map[string]*KEVEntry),
		cachePath: cachePath,
		client:    &http.Client{Timeout: 30 * time.Second},
	}

	// Try to load cached catalog
	if cachePath != "" {
		if err := provider.loadCache(); err != nil {
			slog.Debug("KEV cache not available, will download fresh", "error", err)
		}
	}

	return provider, nil
}

// SearchByCPE is not applicable for KEV - use IsExploited instead.
func (kev *KEVProvider) SearchByCPE(_ context.Context, _ string) ([]Vulnerability, error) {
	return nil, fmt.Errorf("KEV provider does not support CPE search - use IsExploited() to check if a CVE is in the catalog")
}

// SearchByProduct searches for vulnerabilities by vendor/product.
// Note: KEV catalog uses different vendor/product naming than CPE.
func (kev *KEVProvider) SearchByProduct(_ context.Context, vendor, product, _ string) ([]Vulnerability, error) {
	kev.mu.RLock()
	defer kev.mu.RUnlock()

	if kev.catalog == nil {
		return nil, fmt.Errorf("KEV catalog not loaded")
	}

	vendor = strings.ToLower(vendor)
	product = strings.ToLower(product)

	var vulns []Vulnerability
	for i := range kev.catalog.Vulnerabilities {
		entry := &kev.catalog.Vulnerabilities[i]
		entryVendor := strings.ToLower(entry.VendorProject)
		entryProduct := strings.ToLower(entry.Product)

		// Match vendor and product (fuzzy match)
		if strings.Contains(entryVendor, vendor) || strings.Contains(vendor, entryVendor) {
			if strings.Contains(entryProduct, product) || strings.Contains(product, entryProduct) {
				vulns = append(vulns, kev.entryToVulnerability(entry))
			}
		}
	}

	return vulns, nil
}

// UpdateDatabase downloads and caches the KEV catalog.
func (kev *KEVProvider) UpdateDatabase(ctx context.Context) error {
	slog.Info("Downloading CISA KEV catalog...")

	req, err := http.NewRequestWithContext(ctx, "GET", kevCatalogURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := kev.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download KEV catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KEV catalog download failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read KEV catalog: %w", err)
	}

	var catalog KEVCatalog
	if err := json.Unmarshal(body, &catalog); err != nil {
		return fmt.Errorf("failed to parse KEV catalog: %w", err)
	}

	// Build index
	cveIndex := make(map[string]*KEVEntry, len(catalog.Vulnerabilities))
	for i := range catalog.Vulnerabilities {
		entry := &catalog.Vulnerabilities[i]
		cveIndex[entry.CVEID] = entry
	}

	// Update state
	kev.mu.Lock()
	kev.catalog = &catalog
	kev.cveIndex = cveIndex
	kev.lastUpdate = time.Now()
	kev.mu.Unlock()

	// Save cache
	if kev.cachePath != "" {
		if err := kev.saveCache(body); err != nil {
			slog.Warn("Failed to cache KEV catalog", "error", err)
		}
	}

	slog.Info("CISA KEV catalog updated",
		"version", catalog.CatalogVersion,
		"count", catalog.Count,
		"date", catalog.DateReleased)

	return nil
}

// GetLastUpdate returns the last catalog update time.
func (kev *KEVProvider) GetLastUpdate() time.Time {
	kev.mu.RLock()
	defer kev.mu.RUnlock()
	return kev.lastUpdate
}

// IsExploited checks if a CVE is in the KEV catalog (actively exploited).
func (kev *KEVProvider) IsExploited(cveID string) bool {
	kev.mu.RLock()
	defer kev.mu.RUnlock()
	_, exists := kev.cveIndex[cveID]
	return exists
}

// GetKEVEntry returns the KEV entry for a CVE if it exists.
func (kev *KEVProvider) GetKEVEntry(cveID string) *KEVEntry {
	kev.mu.RLock()
	defer kev.mu.RUnlock()
	return kev.cveIndex[cveID]
}

// IsRansomwareRelated checks if a CVE is known to be used in ransomware campaigns.
func (kev *KEVProvider) IsRansomwareRelated(cveID string) bool {
	kev.mu.RLock()
	defer kev.mu.RUnlock()

	entry, exists := kev.cveIndex[cveID]
	if !exists {
		return false
	}
	return strings.EqualFold(entry.KnownRansomwareCampaign, kevRansomwareKnown)
}

// GetCatalogStats returns statistics about the KEV catalog.
func (kev *KEVProvider) GetCatalogStats() map[string]interface{} {
	kev.mu.RLock()
	defer kev.mu.RUnlock()

	if kev.catalog == nil {
		return map[string]interface{}{
			"loaded":     false,
			"lastUpdate": kev.lastUpdate,
		}
	}

	// Count ransomware-related entries
	ransomwareCount := 0
	for i := range kev.catalog.Vulnerabilities {
		if strings.EqualFold(kev.catalog.Vulnerabilities[i].KnownRansomwareCampaign, kevRansomwareKnown) {
			ransomwareCount++
		}
	}

	return map[string]interface{}{
		"loaded":          true,
		"version":         kev.catalog.CatalogVersion,
		"dateReleased":    kev.catalog.DateReleased,
		"count":           kev.catalog.Count,
		"ransomwareCount": ransomwareCount,
		"lastUpdate":      kev.lastUpdate,
	}
}

// EnrichVulnerabilities adds KEV data to existing vulnerability results.
// This should be called after querying NVD to add exploitation context.
func (kev *KEVProvider) EnrichVulnerabilities(vulns []Vulnerability) []Vulnerability {
	kev.mu.RLock()
	defer kev.mu.RUnlock()

	for i := range vulns {
		entry := kev.cveIndex[vulns[i].CVEID]
		if entry != nil {
			// Mark as actively exploited
			vulns[i].ActivelyExploited = true
			vulns[i].RansomwareRelated = strings.EqualFold(entry.KnownRansomwareCampaign, kevRansomwareKnown)
			vulns[i].RequiredAction = entry.RequiredAction
			vulns[i].DueDate = entry.DueDate

			// Boost priority - any KEV entry should be treated as critical priority
			if vulns[i].Severity != "CRITICAL" {
				vulns[i].OriginalSeverity = vulns[i].Severity
				vulns[i].Severity = "CRITICAL" // Escalate to critical due to active exploitation
			}
		}
	}

	return vulns
}

// loadCache loads the KEV catalog from the cache file.
func (kev *KEVProvider) loadCache() error {
	cacheFile := filepath.Clean(filepath.Join(kev.cachePath, kevCacheFile))

	info, err := os.Stat(cacheFile)
	if err != nil {
		return err
	}

	// Check if cache is too old
	if time.Since(info.ModTime()) > kevMaxAge {
		return fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return err
	}

	var catalog KEVCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return err
	}

	// Build index
	cveIndex := make(map[string]*KEVEntry, len(catalog.Vulnerabilities))
	for i := range catalog.Vulnerabilities {
		entry := &catalog.Vulnerabilities[i]
		cveIndex[entry.CVEID] = entry
	}

	kev.mu.Lock()
	kev.catalog = &catalog
	kev.cveIndex = cveIndex
	kev.lastUpdate = info.ModTime()
	kev.mu.Unlock()

	slog.Info("Loaded KEV catalog from cache",
		"version", catalog.CatalogVersion,
		"count", catalog.Count)

	return nil
}

// saveCache saves the KEV catalog to the cache file.
func (kev *KEVProvider) saveCache(data []byte) error {
	if err := os.MkdirAll(kev.cachePath, kevCacheDirPerms); err != nil {
		return err
	}

	cacheFile := filepath.Join(kev.cachePath, kevCacheFile)
	return os.WriteFile(cacheFile, data, kevCacheFilePerms)
}

// entryToVulnerability converts a KEV entry to our Vulnerability format.
func (kev *KEVProvider) entryToVulnerability(entry *KEVEntry) Vulnerability {
	dateAdded, err := time.Parse("2006-01-02", entry.DateAdded)
	if err != nil {
		// Log warning for format changes but continue with zero time
		slog.Warn("Failed to parse KEV date, using zero time",
			"cve", entry.CVEID,
			"dateAdded", entry.DateAdded,
			"error", err)
	}

	return Vulnerability{
		CVEID:             entry.CVEID,
		Description:       entry.ShortDescription,
		Severity:          "CRITICAL", // All KEV entries are critical priority
		Score:             10.0,       // Max score for actively exploited
		Published:         dateAdded,
		References:        []string{fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", entry.CVEID)},
		ActivelyExploited: true,
		RansomwareRelated: strings.EqualFold(entry.KnownRansomwareCampaign, kevRansomwareKnown),
		RequiredAction:    entry.RequiredAction,
		DueDate:           entry.DueDate,
	}
}

// Count returns the number of entries in the KEV catalog.
func (kev *KEVProvider) Count() int {
	kev.mu.RLock()
	defer kev.mu.RUnlock()

	if kev.catalog == nil {
		return 0
	}
	return kev.catalog.Count
}
