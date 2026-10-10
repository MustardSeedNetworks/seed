package vuln_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
)

func TestNewVulnerabilityScanner(t *testing.T) {
	// Test with nil config (should use defaults)
	config := &vuln.VulnerabilityScannerConfig{
		Enabled:           true,
		CVEDatabase:       "nvd",
		SeverityThreshold: "medium",
	}

	scanner, err := vuln.NewVulnerabilityScanner(config)
	if err != nil {
		t.Fatalf("NewVulnerabilityScanner failed: %v", err)
	}
	if scanner == nil {
		t.Fatal("scanner should not be nil")
	}
}

func TestVulnerabilityScanner_StartStop(t *testing.T) {
	config := &vuln.VulnerabilityScannerConfig{
		Enabled:           true,
		CVEDatabase:       "nvd",
		SeverityThreshold: "medium",
		UpdateInterval:    3600,
	}

	scanner, scannerErr := vuln.NewVulnerabilityScanner(config)
	if scannerErr != nil {
		t.Fatalf("NewVulnerabilityScanner failed: %v", scannerErr)
	}

	// Should not be running initially
	if scanner.IsRunning() {
		t.Error("scanner should not be running initially")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start scanner
	if startErr := scanner.Start(ctx); startErr != nil {
		t.Fatalf("Start failed: %v", startErr)
	}

	if !scanner.IsRunning() {
		t.Error("scanner should be running after start")
	}

	// Starting again should fail
	if restartErr := scanner.Start(ctx); restartErr == nil {
		t.Error("starting already running scanner should fail")
	}

	// Stop scanner
	if stopErr := scanner.Stop(); stopErr != nil {
		t.Fatalf("Stop failed: %v", stopErr)
	}

	if scanner.IsRunning() {
		t.Error("scanner should not be running after stop")
	}

	// Stopping again should succeed (no-op)
	if restopErr := scanner.Stop(); restopErr != nil {
		t.Error("stopping stopped scanner should not error")
	}
}

func TestVulnerabilityScanner_FilterBySeverity(t *testing.T) {
	config := &vuln.VulnerabilityScannerConfig{
		Enabled:           true,
		CVEDatabase:       "nvd",
		SeverityThreshold: "high",
	}

	scanner, err := vuln.NewVulnerabilityScanner(config)
	if err != nil {
		t.Fatalf("NewVulnerabilityScanner failed: %v", err)
	}

	// Create test vulnerabilities with different severities
	vulns := []vuln.Vulnerability{
		{CVEID: "CVE-2023-0001", Severity: "LOW", Score: 3.5},
		{CVEID: "CVE-2023-0002", Severity: "MEDIUM", Score: 5.5},
		{CVEID: "CVE-2023-0003", Severity: "HIGH", Score: 7.5},
		{CVEID: "CVE-2023-0004", Severity: "CRITICAL", Score: 9.5},
	}

	// Filter with "high" threshold should only return HIGH and CRITICAL
	accessor := &vuln.VulnerabilityScannerTestAccessor{Scanner: scanner}
	filtered := accessor.FilterBySeverity(vulns)
	if len(filtered) != 2 {
		t.Errorf("expected 2 vulnerabilities, got %d", len(filtered))
	}

	// Verify correct ones were kept
	for _, v := range filtered {
		if v.Severity != "HIGH" && v.Severity != "CRITICAL" {
			t.Errorf("unexpected severity %s in filtered results", v.Severity)
		}
	}
}

func TestVulnerabilityScanner_FilterBySeverityLow(t *testing.T) {
	config := &vuln.VulnerabilityScannerConfig{
		Enabled:           true,
		CVEDatabase:       "nvd",
		SeverityThreshold: "low",
	}

	scanner, err := vuln.NewVulnerabilityScanner(config)
	if err != nil {
		t.Fatalf("NewVulnerabilityScanner failed: %v", err)
	}

	vulns := []vuln.Vulnerability{
		{CVEID: "CVE-2023-0001", Severity: "LOW", Score: 3.5},
		{CVEID: "CVE-2023-0002", Severity: "MEDIUM", Score: 5.5},
		{CVEID: "CVE-2023-0003", Severity: "HIGH", Score: 7.5},
		{CVEID: "CVE-2023-0004", Severity: "CRITICAL", Score: 9.5},
	}

	// Filter with "low" threshold should return all
	accessor := &vuln.VulnerabilityScannerTestAccessor{Scanner: scanner}
	filtered := accessor.FilterBySeverity(vulns)
	if len(filtered) != len(vulns) {
		t.Errorf("expected %d vulnerabilities, got %d", len(vulns), len(filtered))
	}
}

func TestVulnerabilityScanner_GetStats(t *testing.T) {
	config := &vuln.VulnerabilityScannerConfig{
		Enabled:           true,
		CVEDatabase:       "nvd",
		SeverityThreshold: "low",
	}

	scanner, err := vuln.NewVulnerabilityScanner(config)
	if err != nil {
		t.Fatalf("NewVulnerabilityScanner failed: %v", err)
	}

	// Add some mock results
	accessor := &vuln.VulnerabilityScannerTestAccessor{Scanner: scanner}
	accessor.GetMu().Lock()
	accessor.SetDeviceResults(map[string]*vuln.DeviceVulnerabilities{
		"192.168.1.100": {
			DeviceIP: "192.168.1.100",
			Vulnerabilities: []vuln.Vulnerability{
				{CVEID: "CVE-2023-0001", Severity: "CRITICAL", Score: 9.5},
				{CVEID: "CVE-2023-0002", Severity: "HIGH", Score: 7.5},
			},
		},
		"192.168.1.101": {
			DeviceIP: "192.168.1.101",
			Vulnerabilities: []vuln.Vulnerability{
				{CVEID: "CVE-2023-0003", Severity: "MEDIUM", Score: 5.5},
			},
		},
	})
	accessor.GetMu().Unlock()

	stats := scanner.GetStats()

	want := vuln.ScannerStats{
		Enabled:        true,
		DevicesScanned: 2,
		TotalVulns:     3,
		CriticalCount:  1,
		HighCount:      1,
		MediumCount:    1,
		CVEDatabase:    "nvd",
	}
	if stats != want {
		t.Errorf("GetStats() = %+v, want %+v", stats, want)
	}
}

func TestVulnerabilityScanner_ClearResults(t *testing.T) {
	config := &vuln.VulnerabilityScannerConfig{
		Enabled:           true,
		CVEDatabase:       "nvd",
		SeverityThreshold: "low",
	}

	scanner, err := vuln.NewVulnerabilityScanner(config)
	if err != nil {
		t.Fatalf("NewVulnerabilityScanner failed: %v", err)
	}

	// Add some mock results
	accessor := &vuln.VulnerabilityScannerTestAccessor{Scanner: scanner}
	accessor.GetMu().Lock()
	results := accessor.GetDeviceResults()
	results["192.168.1.100"] = &vuln.DeviceVulnerabilities{
		DeviceIP: "192.168.1.100",
	}
	accessor.GetMu().Unlock()

	// Verify results exist
	if len(scanner.GetAllVulnerabilities()) != 1 {
		t.Error("expected 1 result before clear")
	}

	// Clear results
	scanner.ClearResults()

	// Verify cleared
	if len(scanner.GetAllVulnerabilities()) != 0 {
		t.Error("expected 0 results after clear")
	}
}

func TestVulnerabilityScanner_GetConfig(t *testing.T) {
	config := &vuln.VulnerabilityScannerConfig{
		Enabled:           true,
		CVEDatabase:       "nvd",
		SeverityThreshold: "high",
		UpdateInterval:    7200,
	}

	scanner, err := vuln.NewVulnerabilityScanner(config)
	if err != nil {
		t.Fatalf("NewVulnerabilityScanner failed: %v", err)
	}

	retrievedConfig := scanner.GetConfig()
	accessor := &vuln.VulnerabilityScannerTestAccessor{Scanner: scanner}

	// Verify it's a copy, not the original
	if retrievedConfig == accessor.GetConfig() {
		t.Error("GetConfig should return a copy, not the original")
	}

	// Verify values
	if !retrievedConfig.Enabled {
		t.Error("expected Enabled to be true")
	}
	if retrievedConfig.CVEDatabase != "nvd" {
		t.Errorf("expected CVEDatabase to be 'nvd', got %s", retrievedConfig.CVEDatabase)
	}
	if retrievedConfig.SeverityThreshold != "high" {
		t.Errorf(
			"expected SeverityThreshold to be 'high', got %s",
			retrievedConfig.SeverityThreshold,
		)
	}
}

func TestNVDProvider_NewNVDProvider(t *testing.T) {
	// Test without API key
	provider, err := vuln.NewNVDProvider("")
	if err != nil {
		t.Fatalf("NewNVDProvider failed: %v", err)
	}
	if provider == nil {
		t.Fatal("provider should not be nil")
	}

	accessor := &vuln.NVDProviderTestAccessor{Provider: provider}
	if accessor.GetRateLimit() != vuln.NVDRateLimitNoKey {
		t.Errorf(
			"expected rate limit %d, got %d",
			vuln.NVDRateLimitNoKey,
			accessor.GetRateLimit(),
		)
	}

	// Test with API key
	provider, err = vuln.NewNVDProvider("test-api-key")
	if err != nil {
		t.Fatalf("NewNVDProvider with key failed: %v", err)
	}

	accessor = &vuln.NVDProviderTestAccessor{Provider: provider}
	if accessor.GetAPIKey() != "test-api-key" {
		t.Error("API key not set correctly")
	}
	if accessor.GetRateLimit() != vuln.NVDRateLimitWithKey {
		t.Errorf(
			"expected rate limit %d, got %d",
			vuln.NVDRateLimitWithKey,
			accessor.GetRateLimit(),
		)
	}
}

func TestNVDProvider_UpdateDatabase(t *testing.T) {
	provider, providerErr := vuln.NewNVDProvider("")
	if providerErr != nil {
		t.Fatalf("NewNVDProvider failed: %v", providerErr)
	}

	ctx := context.Background()
	if updateErr := provider.UpdateDatabase(ctx); updateErr != nil {
		t.Errorf("UpdateDatabase should not error: %v", updateErr)
	}

	// Verify lastUpdate was set
	if provider.GetLastUpdate().IsZero() {
		t.Error("lastUpdate should be set after UpdateDatabase")
	}
}
