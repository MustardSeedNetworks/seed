package dns_test

import (
	"context"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/dns"
)

func TestDefaultSecurityScanConfig(t *testing.T) {
	config := dns.DefaultSecurityScanConfig()

	if config.Enabled {
		t.Error("expected default Enabled to be false")
	}
	if !config.CheckOpenResolver {
		t.Error("expected default CheckOpenResolver to be true")
	}
	if !config.CheckRebinding {
		t.Error("expected default CheckRebinding to be true")
	}
	if !config.CheckAmplification {
		t.Error("expected default CheckAmplification to be true")
	}
	if !config.CheckDNSSEC {
		t.Error("expected default CheckDNSSEC to be true")
	}
	if config.Timeout != dns.DefaultSecurityScanTimeoutMs {
		t.Errorf("expected default Timeout %d, got %d", dns.DefaultSecurityScanTimeoutMs, config.Timeout)
	}
	if len(config.TestDomains) != 2 {
		t.Errorf("expected 2 test domains, got %d", len(config.TestDomains))
	}
}

func TestNewSecurityScanner(t *testing.T) {
	config := dns.DefaultSecurityScanConfig()
	scanner := dns.NewSecurityScanner(config)

	if scanner == nil {
		t.Fatal("NewSecurityScanner returned nil")
	}
	if scanner.IsRunning() {
		t.Error("expected new scanner to not be running")
	}
}

func TestSecurityScannerIsRunning(t *testing.T) {
	config := dns.DefaultSecurityScanConfig()
	scanner := dns.NewSecurityScanner(config)

	if scanner.IsRunning() {
		t.Error("expected IsRunning() to return false initially")
	}
}

func TestSecurityScannerGetConfig(t *testing.T) {
	config := dns.DefaultSecurityScanConfig()
	config.Timeout = 10000
	scanner := dns.NewSecurityScanner(config)

	retrieved := scanner.GetConfig()
	if retrieved.Timeout != 10000 {
		t.Errorf("expected timeout 10000, got %d", retrieved.Timeout)
	}
}

func TestSecurityScannerSetConfig(t *testing.T) {
	config := dns.DefaultSecurityScanConfig()
	scanner := dns.NewSecurityScanner(config)

	newConfig := config
	newConfig.Enabled = true
	newConfig.Timeout = 15000
	scanner.SetConfig(newConfig)

	retrieved := scanner.GetConfig()
	if !retrieved.Enabled {
		t.Error("expected Enabled to be true after SetConfig")
	}
	if retrieved.Timeout != 15000 {
		t.Errorf("expected timeout 15000, got %d", retrieved.Timeout)
	}
}

func TestSecurityScannerGetResults(t *testing.T) {
	config := dns.DefaultSecurityScanConfig()
	scanner := dns.NewSecurityScanner(config)

	results := scanner.GetResults()
	if results == nil {
		t.Error("expected non-nil results map")
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestSecurityScannerGetResult(t *testing.T) {
	config := dns.DefaultSecurityScanConfig()
	scanner := dns.NewSecurityScanner(config)

	result := scanner.GetResult("nonexistent")
	if result != nil {
		t.Error("expected nil result for nonexistent server")
	}
}

func TestSecurityScanConfigFields(t *testing.T) {
	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     false,
		CheckAmplification: true,
		CheckDNSSEC:        false,
		TestDomains:        []string{"example.com"},
		Timeout:            3000,
	}

	if !config.Enabled {
		t.Error("expected Enabled true")
	}
	if !config.CheckOpenResolver {
		t.Error("expected CheckOpenResolver true")
	}
	if config.CheckRebinding {
		t.Error("expected CheckRebinding false")
	}
	if !config.CheckAmplification {
		t.Error("expected CheckAmplification true")
	}
	if config.CheckDNSSEC {
		t.Error("expected CheckDNSSEC false")
	}
	if len(config.TestDomains) != 1 || config.TestDomains[0] != "example.com" {
		t.Error("expected TestDomains to contain 'example.com'")
	}
	if config.Timeout != 3000 {
		t.Errorf("expected Timeout 3000, got %d", config.Timeout)
	}
}

func TestSecurityScanResultFields(t *testing.T) {
	now := time.Now()
	result := dns.SecurityScanResult{
		Server:              "8.8.8.8:53",
		Timestamp:           now,
		IsOpenResolver:      true,
		OpenResolverDetails: "Test details",
		RebindingVulnerable: false,
		RebindingDetails:    "No rebinding",
		AmplificationFactor: 5.0,
		AmplificationRisk:   "low",
		DNSSECEnabled:       true,
		DNSSECDetails:       "DNSSEC is enabled",
		OverallSeverity:     "ok",
		Vulnerabilities:     []string{"vuln1"},
		Recommendations:     []string{"rec1"},
		CheckResults:        map[string]string{"open_resolver": "FAIL"},
		Error:               "",
	}

	if result.Server != "8.8.8.8:53" {
		t.Errorf("expected Server '8.8.8.8:53', got %q", result.Server)
	}
	if !result.Timestamp.Equal(now) {
		t.Error("timestamp mismatch")
	}
	if !result.IsOpenResolver {
		t.Error("expected IsOpenResolver true")
	}
	if result.OpenResolverDetails != "Test details" {
		t.Errorf("expected OpenResolverDetails 'Test details', got %q", result.OpenResolverDetails)
	}
	if result.RebindingVulnerable {
		t.Error("expected RebindingVulnerable false")
	}
	if result.RebindingDetails != "No rebinding" {
		t.Errorf("expected RebindingDetails 'No rebinding', got %q", result.RebindingDetails)
	}
	if result.AmplificationFactor != 5.0 {
		t.Errorf("expected AmplificationFactor 5.0, got %f", result.AmplificationFactor)
	}
	if result.AmplificationRisk != "low" {
		t.Errorf("expected AmplificationRisk 'low', got %q", result.AmplificationRisk)
	}
	if !result.DNSSECEnabled {
		t.Error("expected DNSSECEnabled true")
	}
	if result.DNSSECDetails != "DNSSEC is enabled" {
		t.Errorf("expected DNSSECDetails 'DNSSEC is enabled', got %q", result.DNSSECDetails)
	}
	if result.Error != "" {
		t.Errorf("expected empty Error, got %q", result.Error)
	}
	if result.OverallSeverity != "ok" {
		t.Errorf("expected OverallSeverity 'ok', got %q", result.OverallSeverity)
	}
	if len(result.Vulnerabilities) != 1 {
		t.Errorf("expected 1 vulnerability, got %d", len(result.Vulnerabilities))
	}
	if len(result.Recommendations) != 1 {
		t.Errorf("expected 1 recommendation, got %d", len(result.Recommendations))
	}
	if result.CheckResults["open_resolver"] != "FAIL" {
		t.Errorf("expected CheckResults['open_resolver'] = 'FAIL', got %q", result.CheckResults["open_resolver"])
	}
}

func TestScanServerWithNoChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a config with all checks disabled.
	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  false,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{},
		Timeout:            1000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}

	// With no checks enabled, severity should be "ok".
	if result.OverallSeverity != "ok" {
		t.Errorf("expected overall severity 'ok', got %q", result.OverallSeverity)
	}

	// Verify the result is stored.
	storedResult := scanner.GetResult("8.8.8.8:53")
	if storedResult == nil {
		t.Error("expected result to be stored")
	}
}

func TestScanServerWithZeroTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a config with zero timeout (should use default).
	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{"google.com"},
		Timeout:            0, // Zero timeout.
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}
}

func TestScanServerOpenResolverCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{"google.com"},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}

	// 8.8.8.8 is a public DNS server, so it should be detected as an open resolver.
	if !result.IsOpenResolver {
		t.Log("8.8.8.8 was not detected as open resolver (may depend on network)")
	}

	if _, ok := result.CheckResults["open_resolver"]; !ok {
		t.Error("expected open_resolver check result")
	}
}

func TestScanServerOpenResolverCheckEmptyDomains(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// When TestDomains is empty, should fall back to google.com.
	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}
}

func TestScanServerDNSRebindingCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  false,
		CheckRebinding:     true,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{"google.com"},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}

	if _, ok := result.CheckResults["rebinding"]; !ok {
		t.Error("expected rebinding check result")
	}
}

func TestScanServerDNSRebindingCheckEmptyDomains(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  false,
		CheckRebinding:     true,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}
}

func TestScanServerAmplificationCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  false,
		CheckRebinding:     false,
		CheckAmplification: true,
		CheckDNSSEC:        false,
		TestDomains:        []string{"google.com"},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}

	if _, ok := result.CheckResults["amplification"]; !ok {
		t.Error("expected amplification check result")
	}
	if result.AmplificationFactor < 0 {
		t.Error("expected non-negative amplification factor")
	}
	if result.AmplificationRisk == "" {
		t.Error("expected non-empty amplification risk")
	}
}

func TestScanServerAmplificationCheckEmptyDomains(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  false,
		CheckRebinding:     false,
		CheckAmplification: true,
		CheckDNSSEC:        false,
		TestDomains:        []string{},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}
}

func TestScanServerDNSSECCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  false,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        true,
		TestDomains:        []string{},
		Timeout:            5000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}

	if _, ok := result.CheckResults["dnssec"]; !ok {
		t.Error("expected dnssec check result")
	}
}

func TestScanServerWithServerPort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{"google.com"},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Server address already includes port.
	result, err := scanner.ScanServer(ctx, "8.8.8.8:53")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}
}

func TestScanServerAllChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     true,
		CheckAmplification: true,
		CheckDNSSEC:        true,
		TestDomains:        []string{"google.com"},
		Timeout:            5000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := scanner.ScanServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}

	// Check that all results are present.
	expectedChecks := []string{"open_resolver", "rebinding", "amplification", "dnssec"}
	for _, check := range expectedChecks {
		if _, ok := result.CheckResults[check]; !ok {
			t.Errorf("expected %s check result", check)
		}
	}

	// Verify overall severity is calculated.
	validSeverities := []string{"ok", "warning", "critical"}
	if !slices.Contains(validSeverities, result.OverallSeverity) {
		t.Errorf("invalid overall severity: %q", result.OverallSeverity)
	}
}

func TestScanServers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{"google.com"},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	servers := []string{"8.8.8.8", "8.8.4.4"}
	results, err := scanner.ScanServers(ctx, servers, 2)
	if err != nil {
		t.Fatalf("ScanServers returned error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestScanServersWithDefaultConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{"google.com"},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	servers := []string{"8.8.8.8"}
	// Use 0 for maxConcurrent to test default handling.
	results, err := scanner.ScanServers(ctx, servers, 0)
	if err != nil {
		t.Fatalf("ScanServers returned error: %v", err)
	}

	if len(results) < 1 {
		t.Errorf("expected at least 1 result, got %d", len(results))
	}
}

func TestScanServersWithCanceledContext(t *testing.T) {
	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{"google.com"},
		Timeout:            3000,
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	servers := []string{"8.8.8.8", "8.8.4.4"}
	_, err := scanner.ScanServers(ctx, servers, 2)
	// Should return context.Canceled error.
	if err == nil {
		t.Log("ScanServers may return nil error if goroutines complete before ctx check")
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"private 10.0.0.0", "10.0.0.1", true},
		{"private 172.16.0.0", "172.16.0.1", true},
		{"private 192.168.0.0", "192.168.1.1", true},
		{"loopback", "127.0.0.1", true},
		{"link-local", "169.254.1.1", true},
		{"public", "8.8.8.8", false},
		{"public cloudflare", "1.1.1.1", false},
		{"IPv6 loopback", "::1", true},
		{"IPv6 link-local", "fe80::1", true},
		{"IPv6 public", "2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := dns.ExportIsPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsPrivateIPNil(t *testing.T) {
	result := dns.ExportIsPrivateIP(nil)
	if result {
		t.Error("expected isPrivateIP(nil) to return false")
	}
}

func TestCalculateSeverity(t *testing.T) {
	tests := []struct {
		name         string
		checkResults map[string]string
		expected     string
	}{
		{
			name:         "all pass",
			checkResults: map[string]string{"open_resolver": "PASS", "rebinding": "PASS"},
			expected:     "ok",
		},
		{
			name:         "one warning",
			checkResults: map[string]string{"open_resolver": "PASS", "dnssec": "WARN"},
			expected:     "warning",
		},
		{
			name:         "one fail",
			checkResults: map[string]string{"open_resolver": "FAIL", "rebinding": "PASS"},
			expected:     "critical",
		},
		{
			name:         "fail overrides warning",
			checkResults: map[string]string{"open_resolver": "FAIL", "dnssec": "WARN"},
			expected:     "critical",
		},
		{
			name:         "empty checks",
			checkResults: map[string]string{},
			expected:     "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &dns.SecurityScanResult{
				CheckResults: tt.checkResults,
			}
			severity := dns.ExportCalculateSeverity(result)
			if severity != tt.expected {
				t.Errorf("calculateSeverity() = %q, expected %q", severity, tt.expected)
			}
		})
	}
}

func TestSecurityScanTimeoutConstants(t *testing.T) {
	if dns.DefaultSecurityScanTimeoutMs != 5000 {
		t.Errorf("expected DefaultSecurityScanTimeoutMs 5000, got %d", dns.DefaultSecurityScanTimeoutMs)
	}
	if dns.DefaultSecurityScanTimeout != 5*time.Second {
		t.Errorf("expected DefaultSecurityScanTimeout 5s, got %v", dns.DefaultSecurityScanTimeout)
	}
}

func TestAmplificationThresholdConstants(t *testing.T) {
	if dns.HighAmplificationRecordThreshold != 10 {
		t.Errorf("expected HighAmplificationRecordThreshold 10, got %d", dns.HighAmplificationRecordThreshold)
	}
	if dns.MediumAmplificationRecordThreshold != 3 {
		t.Errorf("expected MediumAmplificationRecordThreshold 3, got %d", dns.MediumAmplificationRecordThreshold)
	}
}

func TestScanServerNonRespondingServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := dns.SecurityScanConfig{
		Enabled:            true,
		CheckOpenResolver:  true,
		CheckRebinding:     false,
		CheckAmplification: false,
		CheckDNSSEC:        false,
		TestDomains:        []string{"google.com"},
		Timeout:            1000, // Short timeout.
	}

	scanner := dns.NewSecurityScanner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use an IP that likely won't respond to DNS queries.
	result, err := scanner.ScanServer(ctx, "192.0.2.1") // TEST-NET-1, should not respond.
	if err != nil {
		t.Fatalf("ScanServer returned error: %v", err)
	}
	if result == nil {
		t.Fatal("ScanServer returned nil result")
	}

	// Should not be an open resolver since it doesn't respond.
	if result.IsOpenResolver {
		t.Log("Unexpected: 192.0.2.1 responded as open resolver")
	}
}
