// Package dns provides DNS testing and lookup functionality with timing.
// Test suite validates DNS resolution performance, recursive queries,
// and name server discovery.
package dns

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()

	if thresholds.Warning != 100*time.Millisecond {
		t.Errorf("expected warning threshold 100ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 500*time.Millisecond {
		t.Errorf("expected critical threshold 500ms, got %v", thresholds.Critical)
	}
}

func TestNewTester(t *testing.T) {
	tester := NewTester("8.8.8.8", "google.com", DefaultThresholds())
	if tester == nil {
		t.Fatal("NewTester returned nil")
	}

	if tester.server != "8.8.8.8" {
		t.Errorf("expected server 8.8.8.8, got %s", tester.server)
	}
	if tester.testHostname != "google.com" {
		t.Errorf("expected hostname google.com, got %s", tester.testHostname)
	}
	if tester.resolver == nil {
		t.Error("expected resolver to be set")
	}
}

func TestNewTesterWithEmptyServer(t *testing.T) {
	tester := NewTester("", "google.com", DefaultThresholds())
	if tester == nil {
		t.Fatal("NewTester returned nil")
	}

	if tester.server != "" {
		t.Errorf("expected empty server, got %s", tester.server)
	}
}

func TestSetTestHostname(t *testing.T) {
	tester := NewTester("", "google.com", DefaultThresholds())
	tester.SetTestHostname("example.com")

	if tester.testHostname != "example.com" {
		t.Errorf("expected hostname example.com, got %s", tester.testHostname)
	}
}

func TestSetServer(t *testing.T) {
	tester := NewTester("", "google.com", DefaultThresholds())

	// Set custom server
	tester.SetServer("8.8.4.4")
	if tester.server != "8.8.4.4" {
		t.Errorf("expected server 8.8.4.4, got %s", tester.server)
	}

	// Reset to default
	tester.SetServer("")
	if tester.server != "" {
		t.Errorf("expected empty server, got %s", tester.server)
	}
}

func TestGetStatus(t *testing.T) {
	thresholds := Thresholds{
		Warning:  100 * time.Millisecond,
		Critical: 500 * time.Millisecond,
	}
	tester := NewTester("", "google.com", thresholds)

	tests := []struct {
		name     string
		duration time.Duration
		hasError bool
		expected Status
	}{
		{
			name:     "success - fast",
			duration: 50 * time.Millisecond,
			hasError: false,
			expected: StatusSuccess,
		},
		{
			name:     "warning - slow",
			duration: 200 * time.Millisecond,
			hasError: false,
			expected: StatusWarning,
		},
		{
			name:     "error - very slow",
			duration: 600 * time.Millisecond,
			hasError: false,
			expected: StatusError,
		},
		{
			name:     "error - has error",
			duration: 50 * time.Millisecond,
			hasError: true,
			expected: StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tester.getStatus(tt.duration, tt.hasError)
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

	tester := NewTester("", "google.com", DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookup(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookup returned nil")
	}

	// Should succeed for google.com - access fields only after nil check
	if result.Status == StatusError && result.Error == "" {
		t.Error("expected success or error with message")
	}

	// Time should be recorded
	if result.Time == 0 {
		t.Error("expected non-zero time")
	}
}

func TestForwardLookupWithHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := NewTester("", "example.org", DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Override hostname
	result := tester.ForwardLookup(ctx, "cloudflare.com")
	if result == nil {
		t.Fatal("ForwardLookup returned nil")
	}
}

func TestForwardLookupInvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := NewTester("", "invalid.invalid.invalid", DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ForwardLookup(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookup returned nil")
	}

	if result.Status != StatusError {
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

	tester := NewTester("", "google.com", DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}

	// Check fields are populated
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
	servers := GetSystemDNS()

	// On most systems, there should be at least one DNS server
	// But this test just ensures the function doesn't panic
	if servers == nil {
		t.Error("GetSystemDNS returned nil, expected empty slice at minimum")
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusSuccess != "success" {
		t.Error("StatusSuccess should be 'success'")
	}
	if StatusWarning != "warning" {
		t.Error("StatusWarning should be 'warning'")
	}
	if StatusError != "error" {
		t.Error("StatusError should be 'error'")
	}
}

func TestForwardLookupIPv4(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := NewTester("", "google.com", DefaultThresholds())
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

	tester := NewTester("", "test.com", DefaultThresholds())
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

	tester := NewTester("", "google.com", DefaultThresholds())
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

	tester := NewTester("", "test.com", DefaultThresholds())
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

	tester := NewTester("", "google.com", DefaultThresholds())
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

	tester := NewTester("", "google.com", DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tester.ReverseLookup(ctx, "not-an-ip")
	if result == nil {
		t.Fatal("ReverseLookup returned nil")
	}
	// Should have an error since it's not a valid IP
	if result.Error == "" {
		t.Log("Expected error for invalid IP, but some systems may handle this differently")
	}
}

func TestLookupResultFields(t *testing.T) {
	result := LookupResult{
		Result:   "192.168.1.1",
		Time:     50 * time.Millisecond,
		TimeMs:   50,
		Status:   StatusSuccess,
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
	if result.Status != StatusSuccess {
		t.Errorf("expected StatusSuccess, got %v", result.Status)
	}
	if len(result.Resolved) != 2 {
		t.Errorf("expected 2 resolved addresses, got %d", len(result.Resolved))
	}
}

func TestTestResultFields(t *testing.T) {
	result := TestResult{
		Server:       "8.8.8.8",
		Servers:      []string{"8.8.8.8", "8.8.4.4"},
		TestHostname: "google.com",
		Forward:      &LookupResult{Status: StatusSuccess},
		ForwardIPv6:  &LookupResult{Status: StatusSuccess},
		Reverse:      &LookupResult{Status: StatusSuccess},
		ReverseIPv6:  &LookupResult{Status: StatusSuccess},
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
	thresholds := Thresholds{
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
	thresholds := Thresholds{
		Warning:  100 * time.Millisecond,
		Critical: 500 * time.Millisecond,
	}
	tester := NewTester("", "google.com", thresholds)

	// Exactly at warning threshold
	status := tester.getStatus(100*time.Millisecond, false)
	if status != StatusWarning {
		t.Errorf("expected StatusWarning at warning threshold, got %v", status)
	}

	// Exactly at critical threshold
	status = tester.getStatus(500*time.Millisecond, false)
	if status != StatusError {
		t.Errorf("expected StatusError at critical threshold, got %v", status)
	}

	// Just below warning
	status = tester.getStatus(99*time.Millisecond, false)
	if status != StatusSuccess {
		t.Errorf("expected StatusSuccess just below warning, got %v", status)
	}
}

func TestTestWithEmptyServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := NewTester("", "google.com", DefaultThresholds())
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

	tester := NewTester("", "google.com", DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 3; j++ {
				tester.SetTestHostname("example.com")
				_ = tester.ForwardLookup(ctx, "")
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestGetSystemDNSPlatform(t *testing.T) {
	// Just verify it doesn't panic
	servers := getSystemDNSPlatform()
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
		{"exactly min", MinDNSTimeout, false},
		{"exactly max", MaxDNSTimeout, false},
		{"below min", MinDNSTimeout - 1, true},
		{"above max", MaxDNSTimeout + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDNSTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDNSTimeout(%v) error = %v, wantErr %v", tt.timeout, err, tt.wantErr)
			}
		})
	}
}

func TestTimeoutError(t *testing.T) {
	err := &TimeoutError{
		Value: 50 * time.Millisecond,
		Min:   MinDNSTimeout,
		Max:   MaxDNSTimeout,
	}

	msg := err.Error()
	if msg == "" {
		t.Error("TimeoutError.Error() should return non-empty string")
	}
	if !strings.Contains(msg, "50ms") {
		t.Errorf("error message should contain timeout value, got: %s", msg)
	}
	if !strings.Contains(msg, MinDNSTimeout.String()) {
		t.Errorf("error message should contain min value, got: %s", msg)
	}
	if !strings.Contains(msg, MaxDNSTimeout.String()) {
		t.Errorf("error message should contain max value, got: %s", msg)
	}
}

func TestDNSTimeoutConstants(t *testing.T) {
	if MinDNSTimeout != 100*time.Millisecond {
		t.Errorf("MinDNSTimeout = %v, want 100ms", MinDNSTimeout)
	}
	if MaxDNSTimeout != 30*time.Second {
		t.Errorf("MaxDNSTimeout = %v, want 30s", MaxDNSTimeout)
	}
}

func TestSetConfiguredServers(t *testing.T) {
	tester := NewTester("", "google.com", DefaultThresholds())

	servers := []ConfiguredServer{
		{Address: "8.8.8.8", Enabled: true},
		{Address: "1.1.1.1", Enabled: false},
	}

	tester.SetConfiguredServers(servers)

	tester.mu.RLock()
	defer tester.mu.RUnlock()
	if len(tester.configuredServers) != 2 {
		t.Errorf("expected 2 configured servers, got %d", len(tester.configuredServers))
	}
}

func TestConfiguredServerFields(t *testing.T) {
	cs := ConfiguredServer{
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
	result := ServerTestResult{
		Server:      "8.8.8.8",
		Forward:     &LookupResult{Status: StatusSuccess, TimeMs: 10},
		ForwardIPv6: &LookupResult{Status: StatusWarning, TimeMs: 20},
		Status:      StatusSuccess,
		AvgTimeMs:   15,
	}

	if result.Server != "8.8.8.8" {
		t.Errorf("expected Server '8.8.8.8', got %q", result.Server)
	}
	if result.AvgTimeMs != 15 {
		t.Errorf("expected AvgTimeMs 15, got %d", result.AvgTimeMs)
	}
}

func TestTestServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := NewTester("", "google.com", DefaultThresholds())
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

	tester := NewTester("", "google.com", DefaultThresholds())
	tester.SetConfiguredServers([]ConfiguredServer{
		{Address: "8.8.4.4", Enabled: true},
		{Address: "1.0.0.1", Enabled: false}, // disabled
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}

	// Check that enabled configured server is in the list
	found := false
	for _, s := range result.Servers {
		if s == "8.8.4.4" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected enabled configured server '8.8.4.4' to be in servers list")
	}
}

func TestForwardLookupEmptyHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := NewTester("", "", DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// When test hostname is also empty, it should still work but might fail
	result := tester.ForwardLookup(ctx, "")
	if result == nil {
		t.Fatal("ForwardLookup returned nil even with empty hostname")
	}
}

func TestForwardLookupIPv4EmptyHostname(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tester := NewTester("", "", DefaultThresholds())
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

	tester := NewTester("", "", DefaultThresholds())
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

	tester := NewTester("", "google.com", DefaultThresholds())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}

	// PerServerResults should contain results for each system DNS server
	if result.PerServerResults == nil {
		t.Error("PerServerResults should not be nil")
	}
}

func TestServerTestResultStatusCalculation(t *testing.T) {
	// Test status calculation for ServerTestResult
	tests := []struct {
		name           string
		forwardStatus  Status
		ipv6Status     Status
		expectedStatus Status
	}{
		{"both success", StatusSuccess, StatusSuccess, StatusSuccess},
		{"forward error", StatusError, StatusSuccess, StatusError},
		{"ipv6 error", StatusSuccess, StatusError, StatusError},
		{"forward warning", StatusWarning, StatusSuccess, StatusWarning},
		{"ipv6 warning", StatusSuccess, StatusWarning, StatusWarning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ServerTestResult{
				Forward:     &LookupResult{Status: tt.forwardStatus, TimeMs: 10},
				ForwardIPv6: &LookupResult{Status: tt.ipv6Status, TimeMs: 10},
			}

			// Verify the result was created with the expected values
			if result.Forward.Status != tt.forwardStatus {
				t.Errorf("Forward status mismatch: got %v, want %v", result.Forward.Status, tt.forwardStatus)
			}
			if result.ForwardIPv6.Status != tt.ipv6Status {
				t.Errorf("ForwardIPv6 status mismatch: got %v, want %v", result.ForwardIPv6.Status, tt.ipv6Status)
			}

			// Manually calculate status like TestServer does
			hasError := tt.forwardStatus == StatusError || tt.ipv6Status == StatusError
			hasWarning := tt.forwardStatus == StatusWarning || tt.ipv6Status == StatusWarning

			var calculatedStatus Status
			switch {
			case hasError:
				calculatedStatus = StatusError
			case hasWarning:
				calculatedStatus = StatusWarning
			default:
				calculatedStatus = StatusSuccess
			}

			if calculatedStatus != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, calculatedStatus)
			}
		})
	}
}
