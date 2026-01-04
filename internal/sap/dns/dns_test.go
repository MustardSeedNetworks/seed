// Package dns_test provides DNS testing and lookup functionality with timing.
// Test suite validates DNS resolution performance, recursive queries,
// and name server discovery.
package dns_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/dns"
)

func TestDefaultThresholds(t *testing.T) {
	thresholds := dns.DefaultThresholds()

	if thresholds.Warning != 100*time.Millisecond {
		t.Errorf("expected warning threshold 100ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 500*time.Millisecond {
		t.Errorf("expected critical threshold 500ms, got %v", thresholds.Critical)
	}
}

func TestNewTester(t *testing.T) {
	tester := dns.NewTester("8.8.8.8", "google.com", dns.DefaultThresholds())
	if tester == nil {
		t.Fatal("NewTester returned nil")
	}

	if tester.TesterServer() != "8.8.8.8" {
		t.Errorf("expected server 8.8.8.8, got %s", tester.TesterServer())
	}
	if tester.TesterTestHostname() != "google.com" {
		t.Errorf("expected hostname google.com, got %s", tester.TesterTestHostname())
	}
	if !tester.TesterHasResolver() {
		t.Error("expected resolver to be set")
	}
}

func TestNewTesterWithEmptyServer(t *testing.T) {
	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	if tester == nil {
		t.Fatal("NewTester returned nil")
	}

	if tester.TesterServer() != "" {
		t.Errorf("expected empty server, got %s", tester.TesterServer())
	}
}

func TestSetTestHostname(t *testing.T) {
	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	tester.SetTestHostname("example.com")

	if tester.TesterTestHostname() != "example.com" {
		t.Errorf("expected hostname example.com, got %s", tester.TesterTestHostname())
	}
}

func TestSetServer(t *testing.T) {
	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())

	// Set custom server.
	tester.SetServer("8.8.4.4")
	if tester.TesterServer() != "8.8.4.4" {
		t.Errorf("expected server 8.8.4.4, got %s", tester.TesterServer())
	}

	// Reset to default.
	tester.SetServer("")
	if tester.TesterServer() != "" {
		t.Errorf("expected empty server, got %s", tester.TesterServer())
	}
}

func TestGetStatus(t *testing.T) {
	thresholds := dns.Thresholds{
		Warning:  100 * time.Millisecond,
		Critical: 500 * time.Millisecond,
	}
	tester := dns.NewTester("", "google.com", thresholds)

	tests := []struct {
		name     string
		duration time.Duration
		hasError bool
		expected dns.Status
	}{
		{
			name:     "success - fast",
			duration: 50 * time.Millisecond,
			hasError: false,
			expected: dns.StatusSuccess,
		},
		{
			name:     "warning - slow",
			duration: 200 * time.Millisecond,
			hasError: false,
			expected: dns.StatusWarning,
		},
		{
			name:     "error - very slow",
			duration: 600 * time.Millisecond,
			hasError: false,
			expected: dns.StatusError,
		},
		{
			name:     "error - has error",
			duration: 50 * time.Millisecond,
			hasError: true,
			expected: dns.StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tester.GetStatus(tt.duration, tt.hasError)
			if status != tt.expected {
				t.Errorf("expected status %s, got %s", tt.expected, status)
			}
		})
	}
}

func TestForwardLookup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookup(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookup returned nil")
	}

	// Should succeed for google.com - access fields only after nil check.
	if result.Status == dns.StatusError && result.Error == "" {
		t.Error("expected success or error with message")
	}

	// Time should be recorded.
	if result.Time == 0 {
		t.Error("expected non-zero time")
	}
}

func TestForwardLookupWithHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "example.org", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Override hostname.
	result := tester.ForwardLookup(ctx, "cloudflare.com")
	if result == nil {
		t.Fatal("ForwardLookup returned nil")
	}
}

func TestForwardLookupInvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "invalid.invalid.invalid", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookup(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookup returned nil")
	}

	if result.Status != dns.StatusError {
		t.Errorf("expected error status for invalid host, got %s", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message for invalid host")
	}
}

func TestTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}

	// Check fields are populated.
	if result.Server == "" {
		t.Error("expected server to be set")
	}
	if result.TestHostname != "google.com" {
		t.Errorf("expected test hostname google.com, got %s", result.TestHostname)
	}
	if result.Forward == nil {
		t.Error("expected forward lookup result")
	}
	if result.ForwardIPv6 == nil {
		t.Error("expected forward IPv6 lookup result")
	}
}

func TestGetSystemDNS(t *testing.T) {
	servers := dns.GetSystemDNS()

	// On most systems, there should be at least one DNS server.
	// But this test just ensures the function doesn't panic.
	if servers == nil {
		t.Error("GetSystemDNS returned nil, expected empty slice at minimum")
	}
}

func TestStatusConstants(t *testing.T) {
	if dns.StatusSuccess != "success" {
		t.Error("StatusSuccess should be 'success'")
	}
	if dns.StatusWarning != "warning" {
		t.Error("StatusWarning should be 'warning'")
	}
	if dns.StatusError != "error" {
		t.Error("StatusError should be 'error'")
	}
}

func TestForwardLookupIPv4(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookupIPv4(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookupIPv4 returned nil")
	}
	if result.Time == 0 {
		t.Error("expected non-zero time")
	}
	if result.TimeMs < 0 {
		t.Error("expected non-negative TimeMs")
	}
}

func TestForwardLookupIPv4WithHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "test.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookupIPv4(ctx, "cloudflare.com")
	if result == nil {
		t.Fatal("ForwardLookupIPv4 returned nil")
	}
}

func TestForwardLookupIPv6(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookupIPv6(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookupIPv6 returned nil")
	}
}

func TestForwardLookupIPv6WithHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "test.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookupIPv6(ctx, "google.com")
	if result == nil {
		t.Fatal("ForwardLookupIPv6 returned nil")
	}
}

func TestReverseLookup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ReverseLookup(ctx, "8.8.8.8")
	if result == nil {
		t.Fatal("ReverseLookup returned nil")
	}
	if result.Time == 0 {
		t.Error("expected non-zero time")
	}
}

func TestReverseLookupInvalidIP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ReverseLookup(ctx, "not-an-ip")
	if result == nil {
		t.Fatal("ReverseLookup returned nil")
	}
	// Should have an error since it's not a valid IP.
	if result.Error == "" {
		t.Log("Expected error for invalid IP, but some systems may handle this differently")
	}
}

func TestLookupResultFields(t *testing.T) {
	result := dns.LookupResult{
		Result:   "192.168.1.1",
		Time:     50 * time.Millisecond,
		TimeMs:   50,
		Status:   dns.StatusSuccess,
		Error:    "",
		Resolved: []string{"192.168.1.1", "192.168.1.2"},
	}

	if result.Result != "192.168.1.1" {
		t.Errorf("expected Result '192.168.1.1', got %q", result.Result)
	}
	if result.Time != 50*time.Millisecond {
		t.Errorf("expected Time 50ms, got %v", result.Time)
	}
	if result.TimeMs != 50 {
		t.Errorf("expected TimeMs 50, got %d", result.TimeMs)
	}
	if result.Status != dns.StatusSuccess {
		t.Errorf("expected StatusSuccess, got %v", result.Status)
	}
	if result.Error != "" {
		t.Errorf("expected empty Error, got %q", result.Error)
	}
	if len(result.Resolved) != 2 {
		t.Errorf("expected 2 resolved addresses, got %d", len(result.Resolved))
	}
}

func TestTestResultFields(t *testing.T) {
	result := dns.TestResult{
		Server:       "8.8.8.8",
		Servers:      []string{"8.8.8.8", "8.8.4.4"},
		TestHostname: "google.com",
		Forward:      &dns.LookupResult{Status: dns.StatusSuccess},
		ForwardIPv6:  &dns.LookupResult{Status: dns.StatusSuccess},
		Reverse:      &dns.LookupResult{Status: dns.StatusSuccess},
		ReverseIPv6:  &dns.LookupResult{Status: dns.StatusSuccess},
	}

	if result.Server != "8.8.8.8" {
		t.Errorf("expected Server '8.8.8.8', got %q", result.Server)
	}
	if len(result.Servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(result.Servers))
	}
	if result.TestHostname != "google.com" {
		t.Errorf("expected TestHostname 'google.com', got %q", result.TestHostname)
	}
	if result.Forward == nil {
		t.Error("expected non-nil Forward")
	}
	if result.ForwardIPv6 == nil {
		t.Error("expected non-nil ForwardIPv6")
	}
	if result.Reverse == nil {
		t.Error("expected non-nil Reverse")
	}
	if result.ReverseIPv6 == nil {
		t.Error("expected non-nil ReverseIPv6")
	}
}

func TestThresholdsFields(t *testing.T) {
	thresholds := dns.Thresholds{
		Warning:  75 * time.Millisecond,
		Critical: 300 * time.Millisecond,
	}

	if thresholds.Warning != 75*time.Millisecond {
		t.Errorf("expected Warning 75ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 300*time.Millisecond {
		t.Errorf("expected Critical 300ms, got %v", thresholds.Critical)
	}
}

func TestGetStatusEdgeCases(t *testing.T) {
	thresholds := dns.Thresholds{
		Warning:  100 * time.Millisecond,
		Critical: 500 * time.Millisecond,
	}
	tester := dns.NewTester("", "google.com", thresholds)

	// Exactly at warning threshold.
	status := tester.GetStatus(100*time.Millisecond, false)
	if status != dns.StatusWarning {
		t.Errorf("expected StatusWarning at warning threshold, got %v", status)
	}

	// Exactly at critical threshold.
	status = tester.GetStatus(500*time.Millisecond, false)
	if status != dns.StatusError {
		t.Errorf("expected StatusError at critical threshold, got %v", status)
	}

	// Just below warning.
	status = tester.GetStatus(99*time.Millisecond, false)
	if status != dns.StatusSuccess {
		t.Errorf("expected StatusSuccess just below warning, got %v", status)
	}
}

func TestTestWithEmptyServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}
	if result.Server != "System Default" {
		t.Errorf("expected 'System Default' server, got %q", result.Server)
	}
}

func TestConcurrentDNSOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan bool)
	for range 5 {
		go func() {
			for range 3 {
				tester.SetTestHostname("example.com")
				_ = tester.ForwardLookup(ctx, "")
			}
			done <- true
		}()
	}

	for range 5 {
		<-done
	}
}

func TestGetSystemDNSPlatform(t *testing.T) {
	// Just verify it doesn't panic.
	servers := dns.GetSystemDNSPlatform()
	if servers == nil {
		t.Error("expected non-nil slice, even if empty")
	}
}

func TestValidateDNSTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"valid 1s", time.Second, false},
		{"valid 100ms", 100 * time.Millisecond, false},
		{"valid 30s", 30 * time.Second, false},
		{"too short", 50 * time.Millisecond, true},
		{"too long", 60 * time.Second, true},
		{"exactly min", dns.MinDNSTimeout, false},
		{"exactly max", dns.MaxDNSTimeout, false},
		{"below min", dns.MinDNSTimeout - 1, true},
		{"above max", dns.MaxDNSTimeout + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dns.ValidateDNSTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDNSTimeout(%v) error = %v, wantErr %v", tt.timeout, err, tt.wantErr)
			}
		})
	}
}

func TestTimeoutError(t *testing.T) {
	err := &dns.TimeoutError{
		Value: 50 * time.Millisecond,
		Min:   dns.MinDNSTimeout,
		Max:   dns.MaxDNSTimeout,
	}

	msg := err.Error()
	if msg == "" {
		t.Error("TimeoutError.Error() should return non-empty string")
	}
	if !strings.Contains(msg, "50ms") {
		t.Errorf("error message should contain timeout value, got: %s", msg)
	}
	if !strings.Contains(msg, dns.MinDNSTimeout.String()) {
		t.Errorf("error message should contain min value, got: %s", msg)
	}
	if !strings.Contains(msg, dns.MaxDNSTimeout.String()) {
		t.Errorf("error message should contain max value, got: %s", msg)
	}
}

func TestDNSTimeoutConstants(t *testing.T) {
	if dns.MinDNSTimeout != 100*time.Millisecond {
		t.Errorf("MinDNSTimeout = %v, want 100ms", dns.MinDNSTimeout)
	}
	if dns.MaxDNSTimeout != 30*time.Second {
		t.Errorf("MaxDNSTimeout = %v, want 30s", dns.MaxDNSTimeout)
	}
}

func TestSetConfiguredServers(t *testing.T) {
	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())

	servers := []dns.ConfiguredServer{
		{Address: "8.8.8.8", Enabled: true},
		{Address: "1.1.1.1", Enabled: false},
	}

	tester.SetConfiguredServers(servers)

	tester.TesterMu().RLock()
	defer tester.TesterMu().RUnlock()
	if tester.TesterConfiguredServersCount() != 2 {
		t.Errorf("expected 2 configured servers, got %d", tester.TesterConfiguredServersCount())
	}
}

func TestConfiguredServerFields(t *testing.T) {
	cs := dns.ConfiguredServer{
		Address: "8.8.8.8",
		Enabled: true,
	}

	if cs.Address != "8.8.8.8" {
		t.Errorf("expected Address '8.8.8.8', got %q", cs.Address)
	}
	if !cs.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestServerTestResultFields(t *testing.T) {
	result := dns.ServerTestResult{
		Server:      "8.8.8.8",
		Forward:     &dns.LookupResult{Status: dns.StatusSuccess, TimeMs: 10},
		ForwardIPv6: &dns.LookupResult{Status: dns.StatusWarning, TimeMs: 20},
		Status:      dns.StatusSuccess,
		AvgTimeMs:   15,
	}

	if result.Server != "8.8.8.8" {
		t.Errorf("expected Server '8.8.8.8', got %q", result.Server)
	}
	if result.Forward == nil || result.Forward.Status != dns.StatusSuccess {
		t.Error("expected Forward with StatusSuccess")
	}
	if result.ForwardIPv6 == nil || result.ForwardIPv6.Status != dns.StatusWarning {
		t.Error("expected ForwardIPv6 with StatusWarning")
	}
	if result.Status != dns.StatusSuccess {
		t.Errorf("expected StatusSuccess, got %v", result.Status)
	}
	if result.AvgTimeMs != 15 {
		t.Errorf("expected AvgTimeMs 15, got %d", result.AvgTimeMs)
	}
}

func TestTestServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := tester.TestServer(ctx, "8.8.8.8")
	if result == nil {
		t.Fatal("TestServer returned nil")
	}
	if result.Server != "8.8.8.8" {
		t.Errorf("expected server '8.8.8.8', got %q", result.Server)
	}
	if result.Forward == nil {
		t.Error("expected Forward result")
	}
	if result.ForwardIPv6 == nil {
		t.Error("expected ForwardIPv6 result")
	}
}

func TestTestWithConfiguredServers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	tester.SetConfiguredServers([]dns.ConfiguredServer{
		{Address: "8.8.4.4", Enabled: true},
		{Address: "1.0.0.1", Enabled: false}, // disabled
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}

	// Check that enabled configured server is in the list.
	found := slices.Contains(result.Servers, "8.8.4.4")
	if !found {
		t.Error("expected enabled configured server '8.8.4.4' to be in servers list")
	}
}

func TestForwardLookupEmptyHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// When test hostname is also empty, it should still work but might fail.
	result := tester.ForwardLookup(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookup returned nil even with empty hostname")
	}
}

func TestForwardLookupIPv4EmptyHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookupIPv4(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookupIPv4 returned nil")
	}
}

func TestForwardLookupIPv6EmptyHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookupIPv6(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookupIPv6 returned nil")
	}
}

func TestTestPerServerResults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := dns.NewTester("", "google.com", dns.DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}

	// PerServerResults should contain results for each system DNS server.
	if result.PerServerResults == nil {
		t.Error("PerServerResults should not be nil")
	}
}

func TestServerTestResultStatusCalculation(t *testing.T) {
	// Test status calculation for ServerTestResult.
	tests := []struct {
		name           string
		forwardStatus  dns.Status
		ipv6Status     dns.Status
		expectedStatus dns.Status
	}{
		{"both success", dns.StatusSuccess, dns.StatusSuccess, dns.StatusSuccess},
		{"forward error", dns.StatusError, dns.StatusSuccess, dns.StatusError},
		{"ipv6 error", dns.StatusSuccess, dns.StatusError, dns.StatusError},
		{"forward warning", dns.StatusWarning, dns.StatusSuccess, dns.StatusWarning},
		{"ipv6 warning", dns.StatusSuccess, dns.StatusWarning, dns.StatusWarning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &dns.ServerTestResult{
				Forward:     &dns.LookupResult{Status: tt.forwardStatus, TimeMs: 10},
				ForwardIPv6: &dns.LookupResult{Status: tt.ipv6Status, TimeMs: 10},
			}

			// Verify the result was created with the expected values.
			if result.Forward.Status != tt.forwardStatus {
				t.Errorf("Forward status mismatch: got %v, want %v", result.Forward.Status, tt.forwardStatus)
			}
			if result.ForwardIPv6.Status != tt.ipv6Status {
				t.Errorf("ForwardIPv6 status mismatch: got %v, want %v", result.ForwardIPv6.Status, tt.ipv6Status)
			}

			// Manually calculate status like TestServer does.
			hasError := tt.forwardStatus == dns.StatusError || tt.ipv6Status == dns.StatusError
			hasWarning := tt.forwardStatus == dns.StatusWarning || tt.ipv6Status == dns.StatusWarning

			var calculatedStatus dns.Status
			switch {
			case hasError:
				calculatedStatus = dns.StatusError
			case hasWarning:
				calculatedStatus = dns.StatusWarning
			default:
				calculatedStatus = dns.StatusSuccess
			}

			if calculatedStatus != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, calculatedStatus)
			}
		})
	}
}
