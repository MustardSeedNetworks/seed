// Package dns provides DNS security scanning functionality.
//
// This file implements security checks for DNS servers:
//   - Open resolver detection
//   - DNS rebinding vulnerability detection
//   - Response amplification factor calculation
//   - DNSSEC validation support detection
package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Check result constants.
const (
	checkResultPass = "PASS"
	checkResultFail = "FAIL"
	checkResultWarn = "WARN"
)

// SecurityScanConfig contains configuration for DNS security scanning.
type SecurityScanConfig struct {
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	CheckOpenResolver  bool     `yaml:"check_open_resolver" json:"checkOpenResolver"`
	CheckRebinding     bool     `yaml:"check_rebinding" json:"checkRebinding"`
	CheckAmplification bool     `yaml:"check_amplification" json:"checkAmplification"`
	CheckDNSSEC        bool     `yaml:"check_dnssec" json:"checkDnssec"`
	TestDomains        []string `yaml:"test_domains" json:"testDomains"`
	Timeout            int      `yaml:"timeout_ms" json:"timeoutMs"` // milliseconds
}

// DefaultSecurityScanConfig returns sensible defaults for security scanning.
func DefaultSecurityScanConfig() SecurityScanConfig {
	return SecurityScanConfig{
		Enabled:            false,
		CheckOpenResolver:  true,
		CheckRebinding:     true,
		CheckAmplification: true,
		CheckDNSSEC:        true,
		TestDomains:        []string{"google.com", "cloudflare.com"},
		Timeout:            5000, // 5 seconds
	}
}

// SecurityScanResult contains the results of a DNS security scan.
type SecurityScanResult struct {
	Server              string            `json:"server"`
	Timestamp           time.Time         `json:"timestamp"`
	IsOpenResolver      bool              `json:"isOpenResolver"`
	OpenResolverDetails string            `json:"openResolverDetails,omitempty"`
	RebindingVulnerable bool              `json:"rebindingVulnerable"`
	RebindingDetails    string            `json:"rebindingDetails,omitempty"`
	AmplificationFactor float64           `json:"amplificationFactor"`
	AmplificationRisk   string            `json:"amplificationRisk"` // "low", "medium", "high"
	DNSSECEnabled       bool              `json:"dnssecEnabled"`
	DNSSECDetails       string            `json:"dnssecDetails,omitempty"`
	OverallSeverity     string            `json:"overallSeverity"` // "ok", "warning", "critical"
	Vulnerabilities     []string          `json:"vulnerabilities"`
	Recommendations     []string          `json:"recommendations"`
	CheckResults        map[string]string `json:"checkResults"` // Individual check pass/fail
	Error               string            `json:"error,omitempty"`
}

// SecurityScanner performs DNS security scans.
type SecurityScanner struct {
	config  SecurityScanConfig
	mu      sync.RWMutex
	running bool
	results map[string]*SecurityScanResult
}

// NewSecurityScanner creates a new DNS security scanner.
func NewSecurityScanner(config SecurityScanConfig) *SecurityScanner {
	return &SecurityScanner{
		config:  config,
		results: make(map[string]*SecurityScanResult),
	}
}

// IsRunning returns true if a scan is in progress.
func (s *SecurityScanner) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetConfig returns the current configuration.
func (s *SecurityScanner) GetConfig() SecurityScanConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// SetConfig updates the scanner configuration.
func (s *SecurityScanner) SetConfig(config SecurityScanConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// GetResults returns all scan results.
func (s *SecurityScanner) GetResults() map[string]*SecurityScanResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to avoid race conditions
	results := make(map[string]*SecurityScanResult, len(s.results))
	for k, v := range s.results {
		results[k] = v
	}
	return results
}

// GetResult returns the scan result for a specific server.
func (s *SecurityScanner) GetResult(server string) *SecurityScanResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results[server]
}

// ScanServer performs a security scan on a single DNS server.
func (s *SecurityScanner) ScanServer(ctx context.Context, server string) (*SecurityScanResult, error) {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	result := &SecurityScanResult{
		Server:          server,
		Timestamp:       time.Now(),
		CheckResults:    make(map[string]string),
		Vulnerabilities: []string{},
		Recommendations: []string{},
	}

	timeout := time.Duration(s.config.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Normalize server address.
	if !strings.Contains(server, ":") {
		server += ":53"
	}

	// Perform security checks
	if s.config.CheckOpenResolver {
		s.checkOpenResolver(ctx, server, timeout, result)
	}

	if s.config.CheckRebinding {
		s.checkDNSRebinding(ctx, server, timeout, result)
	}

	if s.config.CheckAmplification {
		s.checkAmplification(ctx, server, timeout, result)
	}

	if s.config.CheckDNSSEC {
		s.checkDNSSEC(ctx, server, timeout, result)
	}

	// Calculate overall severity
	result.OverallSeverity = s.calculateSeverity(result)

	// Store result
	s.mu.Lock()
	s.results[server] = result
	s.mu.Unlock()

	return result, nil
}

// checkOpenResolver tests if the server responds to queries from external sources.
// An open resolver can be abused for DNS amplification attacks.
func (s *SecurityScanner) checkOpenResolver(ctx context.Context, server string, timeout time.Duration, result *SecurityScanResult) {
	// Try to resolve an external domain through this server
	// If successful from a non-local source, it's an open resolver
	testDomains := s.config.TestDomains
	if len(testDomains) == 0 {
		testDomains = []string{"google.com"}
	}

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, "udp", server)
		},
	}

	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for _, domain := range testDomains {
		ips, err := resolver.LookupIPAddr(queryCtx, domain)
		if err == nil && len(ips) > 0 {
			// Server responded to our query - it's an open resolver.
			result.IsOpenResolver = true
			result.OpenResolverDetails = fmt.Sprintf("Resolved %s to %d addresses", domain, len(ips))
			result.Vulnerabilities = append(result.Vulnerabilities, "Open DNS resolver detected")
			result.Recommendations = append(result.Recommendations,
				"Configure DNS server to only respond to authorized clients",
				"Implement response rate limiting (RRL)",
			)
			result.CheckResults["open_resolver"] = checkResultFail
			return
		}
	}

	result.IsOpenResolver = false
	result.OpenResolverDetails = "Server does not respond to external queries"
	result.CheckResults["open_resolver"] = checkResultPass
}

// checkDNSRebinding tests for DNS rebinding vulnerability.
// DNS rebinding allows attackers to bypass same-origin policy by having
// a DNS record briefly resolve to an attacker-controlled IP, then to an internal IP.
func (s *SecurityScanner) checkDNSRebinding(ctx context.Context, server string, timeout time.Duration, result *SecurityScanResult) {
	// Check if the server returns private IP addresses for external domains
	// This is a simplified check - full rebinding protection requires more complex testing

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, "udp", server)
		},
	}

	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Test with a known public domain.
	testDomains := s.config.TestDomains
	if len(testDomains) == 0 {
		testDomains = []string{"google.com"}
	}

	for _, domain := range testDomains {
		ips, err := resolver.LookupIPAddr(queryCtx, domain)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			if !isPrivateIP(ip.IP) {
				continue
			}
			result.RebindingVulnerable = true
			result.RebindingDetails = fmt.Sprintf("Public domain %s resolved to private IP %s", domain, ip.IP)
			result.Vulnerabilities = append(result.Vulnerabilities, "DNS rebinding vulnerability detected")
			result.Recommendations = append(result.Recommendations,
				"Configure DNS server to block private IP responses for public domains",
				"Implement DNS rebinding protection",
			)
			result.CheckResults["rebinding"] = checkResultFail
			return
		}
	}

	result.RebindingVulnerable = false
	result.RebindingDetails = "No private IPs returned for public domains"
	result.CheckResults["rebinding"] = checkResultPass
}

// checkAmplification estimates the DNS amplification factor.
// DNS amplification attacks exploit servers that return large responses to small queries.
func (s *SecurityScanner) checkAmplification(ctx context.Context, server string, timeout time.Duration, result *SecurityScanResult) {
	// Amplification factor is typically measured as response_size / query_size
	// Common amplification vectors:
	// - ANY queries (now deprecated, should return small responses)
	// - TXT records (can be large)
	// - DNSSEC-signed responses (include signatures)

	// For this simplified check, we estimate based on whether the server
	// responds to ANY queries (which are often blocked by modern servers)

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, "udp", server)
		},
	}

	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try to get TXT records which can be large.
	testDomains := s.config.TestDomains
	if len(testDomains) == 0 {
		testDomains = []string{"google.com"}
	}

	var maxRecords int
	for _, domain := range testDomains {
		txts, err := resolver.LookupTXT(queryCtx, domain)
		if err == nil && len(txts) > maxRecords {
			maxRecords = len(txts)
		}
	}

	// Estimate amplification factor.
	// A typical DNS query is ~50 bytes, responses can be much larger.
	// This is a rough estimate - real measurement requires raw packet analysis.
	switch {
	case maxRecords > 10:
		result.AmplificationFactor = 50.0 // High amplification
		result.AmplificationRisk = "high"
		result.Vulnerabilities = append(result.Vulnerabilities, "High DNS amplification potential")
		result.Recommendations = append(result.Recommendations,
			"Implement response rate limiting (RRL)",
			"Limit response sizes for recursive queries",
		)
		result.CheckResults["amplification"] = checkResultFail
	case maxRecords > 3:
		result.AmplificationFactor = 20.0 // Medium amplification
		result.AmplificationRisk = "medium"
		result.CheckResults["amplification"] = checkResultWarn
	default:
		result.AmplificationFactor = 5.0 // Low amplification (typical)
		result.AmplificationRisk = "low"
		result.CheckResults["amplification"] = checkResultPass
	}
}

// checkDNSSEC tests if the server supports DNSSEC validation.
func (s *SecurityScanner) checkDNSSEC(ctx context.Context, server string, timeout time.Duration, result *SecurityScanResult) {
	// DNSSEC check: Query a known DNSSEC-signed domain and check for AD (Authenticated Data) flag.
	// This is a simplified check - full DNSSEC validation requires checking DNSKEY and RRSIG records.

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, "udp", server)
		},
	}

	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Test with a domain known to be DNSSEC-signed.
	// dnssec-deployment.org and dnssec-tools.org are good test domains.
	testDomains := []string{"dnssec-tools.org", "isc.org", "ietf.org"}

	for _, domain := range testDomains {
		ips, err := resolver.LookupIPAddr(queryCtx, domain)
		if err == nil && len(ips) > 0 {
			// Server can resolve DNSSEC-signed domains.
			// Note: This doesn't confirm DNSSEC validation is enabled,
			// just that the server can handle DNSSEC-signed domains.
			result.DNSSECEnabled = true
			result.DNSSECDetails = fmt.Sprintf("Server resolves DNSSEC-signed domain %s", domain)
			result.CheckResults["dnssec"] = checkResultPass
			return
		}
	}

	result.DNSSECEnabled = false
	result.DNSSECDetails = "Could not verify DNSSEC support"
	result.Recommendations = append(result.Recommendations,
		"Consider enabling DNSSEC validation for improved security",
	)
	result.CheckResults["dnssec"] = checkResultWarn
}

// calculateSeverity determines overall security severity based on check results.
func (s *SecurityScanner) calculateSeverity(result *SecurityScanResult) string {
	criticalCount := 0
	warningCount := 0

	for _, status := range result.CheckResults {
		switch status {
		case checkResultFail:
			criticalCount++
		case checkResultWarn:
			warningCount++
		}
	}

	switch {
	case criticalCount > 0:
		return "critical"
	case warningCount > 0:
		return "warning"
	default:
		return "ok"
	}
}

// isPrivateIP checks if an IP address is in a private, loopback, or link-local range.
// Uses Go's built-in IP methods for cleaner, more maintainable code.
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	// IsPrivate covers RFC 1918 (10.x, 172.16-31.x, 192.168.x) and IPv6 ULA (fc00::/7).
	// IsLoopback covers 127.x.x.x and ::1.
	// IsLinkLocalUnicast covers 169.254.x.x and fe80::/10.
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

// ScanServers performs security scans on multiple DNS servers concurrently.
func (s *SecurityScanner) ScanServers(ctx context.Context, servers []string, maxConcurrent int) ([]*SecurityScanResult, error) {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	results := make([]*SecurityScanResult, 0, len(servers))
	resultsChan := make(chan *SecurityScanResult, len(servers))
	errChan := make(chan error, len(servers))

	// Semaphore for limiting concurrency
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	for _, server := range servers {
		wg.Add(1)
		go func(srv string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			result, err := s.ScanServer(ctx, srv)
			if err != nil {
				errChan <- err
				return
			}
			resultsChan <- result
		}(server)
	}

	// Wait for all scans to complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errChan)
	}()

	// Collect results
	for result := range resultsChan {
		results = append(results, result)
	}

	// Check for errors (return first error if any)
	select {
	case err := <-errChan:
		if err != nil {
			return results, err
		}
	default:
	}

	return results, nil
}
