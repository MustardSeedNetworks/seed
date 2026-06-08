package vuln

// export_test.go exposes vuln-package internals for the external vuln_test
// package (moved from internal/discovery/export_test.go with the stage, ADR-0018).

import "sync"

// Export NVD rate limit constants for testing.
const (
	NVDRateLimitNoKey   = nvdRateLimitNoKey
	NVDRateLimitWithKey = nvdRateLimitWithKey
)

// VulnerabilityScannerTestAccessor provides access to VulnerabilityScanner's private fields for testing.
type VulnerabilityScannerTestAccessor struct {
	Scanner *VulnerabilityScanner
}

// GetMu returns the scanner's mutex for testing.
func (v *VulnerabilityScannerTestAccessor) GetMu() *sync.RWMutex {
	return &v.Scanner.mu
}

// GetDeviceResults returns the scanner's deviceResults map.
func (v *VulnerabilityScannerTestAccessor) GetDeviceResults() map[string]*DeviceVulnerabilities {
	return v.Scanner.deviceResults
}

// SetDeviceResults sets the scanner's deviceResults map.
func (v *VulnerabilityScannerTestAccessor) SetDeviceResults(
	results map[string]*DeviceVulnerabilities,
) {
	v.Scanner.deviceResults = results
}

// GetConfig returns the scanner's config.
func (v *VulnerabilityScannerTestAccessor) GetConfig() *VulnerabilityScannerConfig {
	return v.Scanner.config
}

// FilterBySeverity exposes the private filterBySeverity method.
func (v *VulnerabilityScannerTestAccessor) FilterBySeverity(vulns []Vulnerability) []Vulnerability {
	return v.Scanner.filterBySeverity(vulns)
}

// NVDProviderTestAccessor provides access to NVDProvider's private fields for testing.
type NVDProviderTestAccessor struct {
	Provider *NVDProvider
}

// GetAPIKey returns the provider's API key.
func (n *NVDProviderTestAccessor) GetAPIKey() string {
	return n.Provider.apiKey
}

// GetRateLimit returns the provider's rate limit.
func (n *NVDProviderTestAccessor) GetRateLimit() int {
	return n.Provider.rateLimit
}
