// Package discovery provides local CVE database support for offline vulnerability scanning.
//
// The LocalProvider loads CVE data from a JSON file, enabling vulnerability scanning
// without requiring NVD API access. This is useful for:
//   - Air-gapped environments
//   - Faster scans without API rate limits
//   - Testing and development
//
// The database file should be in NIST JSON feed format.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LocalProvider implements CVEProvider using a local JSON database.
type LocalProvider struct {
	mu              sync.RWMutex
	vulnerabilities map[string][]Vulnerability // CPE -> vulnerabilities
	lastUpdate      time.Time
	dbPath          string
}

// localCVEEntry represents a CVE entry in the local database.
type localCVEEntry struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	CVSSScore   float64  `json:"cvss_score"`
	Published   string   `json:"published"`
	Modified    string   `json:"modified"`
	References  []string `json:"references"`
	CPEs        []string `json:"cpes"`
	Vendor      string   `json:"vendor,omitempty"`
	Product     string   `json:"product,omitempty"`
	Versions    []string `json:"versions,omitempty"`
}

// localCVEDatabase represents the structure of the local CVE JSON file.
type localCVEDatabase struct {
	Version     string          `json:"version"`
	LastUpdated string          `json:"last_updated"`
	CVEs        []localCVEEntry `json:"cves"`
}

// NewLocalProvider creates a new local CVE database provider.
// If dbPath is empty, it uses the default path in the config directory.
func NewLocalProvider(dbPath string) (*LocalProvider, error) {
	if dbPath == "" {
		// Default to configs/cve-database.json
		dbPath = filepath.Join("configs", "cve-database.json")
	}

	provider := &LocalProvider{
		vulnerabilities: make(map[string][]Vulnerability),
		dbPath:          dbPath,
	}

	// Load database if it exists
	if _, statErr := os.Stat(dbPath); statErr == nil {
		if loadErr := provider.loadDatabase(); loadErr != nil {
			return nil, fmt.Errorf("failed to load CVE database: %w", loadErr)
		}
	} else {
		// Create empty database structure
		provider.lastUpdate = time.Time{}
	}

	return provider, nil
}

// loadDatabase loads CVE data from the JSON file.
func (p *LocalProvider) loadDatabase() error {
	data, err := os.ReadFile(p.dbPath)
	if err != nil {
		return fmt.Errorf("failed to read database file: %w", err)
	}

	var db localCVEDatabase
	if unmarshalErr := json.Unmarshal(data, &db); unmarshalErr != nil {
		return fmt.Errorf("failed to parse database: %w", unmarshalErr)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear existing data
	p.vulnerabilities = make(map[string][]Vulnerability)

	// Parse last updated time
	if db.LastUpdated != "" {
		if t, parseErr := time.Parse(time.RFC3339, db.LastUpdated); parseErr == nil {
			p.lastUpdate = t
		}
	}

	// Index vulnerabilities by CPE
	for i := range db.CVEs {
		cve := &db.CVEs[i]
		vuln := Vulnerability{
			CVEID:       cve.ID,
			Description: cve.Description,
			Severity:    cve.Severity,
			Score:       cve.CVSSScore,
			References:  cve.References,
		}

		// Parse published date
		if cve.Published != "" {
			if t, parseErr := time.Parse(time.RFC3339, cve.Published); parseErr == nil {
				vuln.Published = t
			}
		}

		// Index by each CPE
		for _, cpe := range cve.CPEs {
			normalizedCPE := strings.ToLower(cpe)
			p.vulnerabilities[normalizedCPE] = append(p.vulnerabilities[normalizedCPE], vuln)
		}

		// Also index by vendor:product:version for SearchByProduct
		if cve.Vendor != "" && cve.Product != "" {
			for _, ver := range cve.Versions {
				key := fmt.Sprintf(
					"%s:%s:%s",
					strings.ToLower(cve.Vendor),
					strings.ToLower(cve.Product),
					strings.ToLower(ver),
				)
				p.vulnerabilities[key] = append(p.vulnerabilities[key], vuln)
			}
			// Also index without version for broader matches
			key := fmt.Sprintf("%s:%s:*", strings.ToLower(cve.Vendor), strings.ToLower(cve.Product))
			p.vulnerabilities[key] = append(p.vulnerabilities[key], vuln)
		}
	}

	return nil
}

// SearchByCPE searches for vulnerabilities affecting a CPE string.
func (p *LocalProvider) SearchByCPE(ctx context.Context, cpe string) ([]Vulnerability, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	normalizedCPE := strings.ToLower(cpe)
	if vulns, ok := p.vulnerabilities[normalizedCPE]; ok {
		// Return a copy to avoid race conditions
		result := make([]Vulnerability, len(vulns))
		copy(result, vulns)
		return result, nil
	}

	// Try prefix matching for partial CPE strings
	var results []Vulnerability
	for key, vulns := range p.vulnerabilities {
		if strings.HasPrefix(key, normalizedCPE) || strings.HasPrefix(normalizedCPE, key) {
			results = append(results, vulns...)
		}
	}

	return results, nil
}

// SearchByProduct searches for vulnerabilities by vendor/product/version.
func (p *LocalProvider) SearchByProduct(ctx context.Context, vendor, product, version string) ([]Vulnerability, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Try exact match first
	key := fmt.Sprintf("%s:%s:%s", strings.ToLower(vendor), strings.ToLower(product), strings.ToLower(version))
	if vulns, ok := p.vulnerabilities[key]; ok {
		result := make([]Vulnerability, len(vulns))
		copy(result, vulns)
		return result, nil
	}

	// Try wildcard version match
	wildcardKey := fmt.Sprintf("%s:%s:*", strings.ToLower(vendor), strings.ToLower(product))
	if vulns, ok := p.vulnerabilities[wildcardKey]; ok {
		result := make([]Vulnerability, len(vulns))
		copy(result, vulns)
		return result, nil
	}

	return nil, nil
}

// UpdateDatabase updates the local CVE database from a remote source.
// For the local provider, this reloads from the file if it has been updated.
func (p *LocalProvider) UpdateDatabase(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check if database file exists
	info, err := os.Stat(p.dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("CVE database file not found: %s (download from NVD or create manually)", p.dbPath)
		}
		return fmt.Errorf("failed to stat database file: %w", err)
	}

	// Check if file has been modified since last load
	p.mu.RLock()
	needsReload := info.ModTime().After(p.lastUpdate)
	p.mu.RUnlock()

	if needsReload {
		if loadErr := p.loadDatabase(); loadErr != nil {
			return loadErr
		}
	}

	return nil
}

// GetLastUpdate returns the last database update time.
func (p *LocalProvider) GetLastUpdate() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastUpdate
}

// GetDatabasePath returns the path to the local database file.
func (p *LocalProvider) GetDatabasePath() string {
	return p.dbPath
}

// GetVulnerabilityCount returns the total number of indexed vulnerabilities.
func (p *LocalProvider) GetVulnerabilityCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	seen := make(map[string]bool)
	for _, vulns := range p.vulnerabilities {
		for i := range vulns {
			if !seen[vulns[i].CVEID] {
				seen[vulns[i].CVEID] = true
				count++
			}
		}
	}
	return count
}
