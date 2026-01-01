// Package discovery provides CVE (Common Vulnerabilities and Exposures) scanning.
//
// This file integrates with the National Vulnerability Database (NVD) to identify known
// security vulnerabilities in discovered network devices based on their fingerprinted
// profiles (vendor, product, version).
//
// Features:
//   - Query NVD API for CVE information
//   - Match device profiles against vulnerability databases
//   - Severity classification (Critical, High, Medium, Low)
//   - CVE caching to reduce API load
//   - Rate limiting for NVD API compliance
//
// The scanner uses device fingerprinting results (OS, services, versions) to identify
// applicable CVEs and provides detailed vulnerability reports with remediation guidance.
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	nvdAPIBaseURL       = "https://services.nvd.nist.gov/rest/json/cves/2.0"
	nvdRateLimitNoKey   = 10  // requests per 30 seconds without API key
	nvdRateLimitWithKey = 100 // requests per 30 seconds with API key
)

// NVDProvider implements CVEProvider using the NVD API.
type NVDProvider struct {
	apiKey     string
	client     *http.Client
	rateLimit  int
	lastUpdate time.Time
}

// nvdCVEResponse represents the NVD API response structure.
type nvdCVEResponse struct {
	ResultsPerPage  int    `json:"resultsPerPage"`
	StartIndex      int    `json:"startIndex"`
	TotalResults    int    `json:"totalResults"`
	Format          string `json:"format"`
	Version         string `json:"version"`
	Timestamp       string `json:"timestamp"`
	Vulnerabilities []struct {
		CVE struct {
			ID               string `json:"id"`
			SourceIdentifier string `json:"sourceIdentifier"`
			Published        string `json:"published"`
			LastModified     string `json:"lastModified"`
			VulnStatus       string `json:"vulnStatus"`
			Descriptions     []struct {
				Lang  string `json:"lang"`
				Value string `json:"value"`
			} `json:"descriptions"`
			Metrics struct {
				CVSSMetricV31 []struct {
					Type     string `json:"type"`
					CVSSData struct {
						Version      string  `json:"version"`
						VectorString string  `json:"vectorString"`
						BaseScore    float64 `json:"baseScore"`
						BaseSeverity string  `json:"baseSeverity"`
					} `json:"cvssData"`
				} `json:"cvssMetricV31"`
				CVSSMetricV2 []struct {
					Type     string `json:"type"`
					CVSSData struct {
						Version      string  `json:"version"`
						VectorString string  `json:"vectorString"`
						BaseScore    float64 `json:"baseScore"`
						Severity     string  `json:"severity"`
					} `json:"cvssData"`
				} `json:"cvssMetricV2"`
			} `json:"metrics"`
			References []struct {
				URL    string   `json:"url"`
				Source string   `json:"source"`
				Tags   []string `json:"tags,omitempty"`
			} `json:"references"`
			Configurations []struct {
				Nodes []struct {
					Operator string `json:"operator"`
					Negate   bool   `json:"negate"`
					CPEMatch []struct {
						Vulnerable            bool   `json:"vulnerable"`
						Criteria              string `json:"criteria"`
						VersionStartIncluding string `json:"versionStartIncluding,omitempty"`
						VersionEndExcluding   string `json:"versionEndExcluding,omitempty"`
						MatchCriteriaID       string `json:"matchCriteriaId"`
					} `json:"cpeMatch"`
				} `json:"nodes"`
			} `json:"configurations"`
		} `json:"cve"`
	} `json:"vulnerabilities"`
}

// NewNVDProvider creates a new NVD API provider.
func NewNVDProvider(apiKey string) (*NVDProvider, error) {
	rateLimit := nvdRateLimitNoKey
	if apiKey != "" {
		rateLimit = nvdRateLimitWithKey
	}

	return &NVDProvider{
		apiKey:    apiKey,
		client:    &http.Client{Timeout: 30 * time.Second},
		rateLimit: rateLimit,
	}, nil
}

// SearchByCPE searches for vulnerabilities affecting a CPE string.
func (nvd *NVDProvider) SearchByCPE(ctx context.Context, cpe string) ([]Vulnerability, error) {
	// Build query URL
	params := url.Values{}
	params.Set("cpeName", cpe)
	params.Set("resultsPerPage", "100")

	queryURL := fmt.Sprintf("%s?%s", nvdAPIBaseURL, params.Encode())

	// Make API request
	resp, err := nvd.makeRequest(ctx, queryURL)
	if err != nil {
		return nil, err
	}

	return nvd.parseResponse(resp)
}

// SearchByProduct searches for vulnerabilities by vendor/product/version.
func (nvd *NVDProvider) SearchByProduct(ctx context.Context, vendor, product, version string) ([]Vulnerability, error) {
	// Construct CPE 2.3 string
	// Format: cpe:2.3:part:vendor:product:version:update:edition:language:sw_edition:target_sw:target_hw:other
	// For software: cpe:2.3:a:vendor:product:version:*:*:*:*:*:*:*
	cpe := fmt.Sprintf("cpe:2.3:a:%s:%s:%s:*:*:*:*:*:*:*",
		strings.ToLower(vendor),
		strings.ToLower(product),
		version,
	)

	return nvd.SearchByCPE(ctx, cpe)
}

// UpdateDatabase is a no-op for NVD as it's always up-to-date via API.
func (nvd *NVDProvider) UpdateDatabase(_ context.Context) error {
	// NVD API is always current, no local database to update
	nvd.lastUpdate = time.Now()
	return nil
}

// GetLastUpdate returns the last database update time.
func (nvd *NVDProvider) GetLastUpdate() time.Time {
	return nvd.lastUpdate
}

// makeRequest makes an HTTP request to the NVD API with proper headers.
func (nvd *NVDProvider) makeRequest(ctx context.Context, requestURL string) (*nvdCVEResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key if available
	if nvd.apiKey != "" {
		req.Header.Set("Apikey", nvd.apiKey)
	}

	// Make request
	resp, err := nvd.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("NVD API returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("NVD API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var nvdResp nvdCVEResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&nvdResp); decodeErr != nil {
		return nil, fmt.Errorf("failed to parse response: %w", decodeErr)
	}

	return &nvdResp, nil
}

// parseResponse converts NVD API response to our Vulnerability format.
func (nvd *NVDProvider) parseResponse(resp *nvdCVEResponse) ([]Vulnerability, error) {
	vulns := make([]Vulnerability, 0, len(resp.Vulnerabilities))

	for i := range resp.Vulnerabilities {
		cve := &resp.Vulnerabilities[i].CVE

		// Extract description (prefer English)
		description := ""
		for _, desc := range cve.Descriptions {
			if desc.Lang == "en" {
				description = desc.Value
				break
			}
		}
		if description == "" && len(cve.Descriptions) > 0 {
			description = cve.Descriptions[0].Value
		}

		// Extract CVSS score and severity
		score := 0.0
		severity := "UNKNOWN"

		// Prefer CVSS v3.1
		if len(cve.Metrics.CVSSMetricV31) > 0 {
			metric := cve.Metrics.CVSSMetricV31[0]
			score = metric.CVSSData.BaseScore
			severity = metric.CVSSData.BaseSeverity
		} else if len(cve.Metrics.CVSSMetricV2) > 0 {
			// Fallback to CVSS v2
			metric := cve.Metrics.CVSSMetricV2[0]
			score = metric.CVSSData.BaseScore
			severity = metric.CVSSData.Severity
		}

		// Extract references
		references := make([]string, 0, len(cve.References))
		for _, ref := range cve.References {
			references = append(references, ref.URL)
		}

		// Parse timestamps (ignore parse errors, use zero time if invalid)
		published, _ := time.Parse(
			time.RFC3339,
			cve.Published,
		)
		modified, _ := time.Parse(
			time.RFC3339,
			cve.LastModified,
		)

		// Extract affected CPE (use first one if available)
		affectedCPE := ""
		if len(cve.Configurations) > 0 && len(cve.Configurations[0].Nodes) > 0 {
			node := cve.Configurations[0].Nodes[0]
			if len(node.CPEMatch) > 0 {
				affectedCPE = node.CPEMatch[0].Criteria
			}
		}

		vuln := Vulnerability{
			CVEID:       cve.ID,
			Description: description,
			Severity:    severity,
			Score:       score,
			Published:   published,
			Modified:    modified,
			References:  references,
			AffectedCPE: affectedCPE,
		}

		vulns = append(vulns, vuln)
	}

	return vulns, nil
}

// ValidateNVDAPIKey validates an NVD API key by making a test request.
// Returns true if the key is valid, false otherwise.
func ValidateNVDAPIKey(ctx context.Context, apiKey string) (bool, error) {
	if apiKey == "" {
		return false, nil
	}

	// Make a minimal test request to NVD API
	// We query for a small result to minimize bandwidth
	testURL := nvdAPIBaseURL + "?resultsPerPage=1&startIndex=0"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, http.NoBody)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header
	req.Header.Set("Apikey", apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	switch resp.StatusCode {
	case http.StatusOK:
		// API key is valid
		return true, nil
	case http.StatusForbidden, http.StatusUnauthorized:
		// API key is invalid
		return false, nil
	case http.StatusTooManyRequests:
		// Rate limited - key might still be valid, but we can't verify
		return false, errors.New("rate limited - try again later")
	default:
		// Unexpected status
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return false, fmt.Errorf("unexpected status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return false, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
}
