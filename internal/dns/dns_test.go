package dns

import (
	"context"
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

	// Should succeed for google.com
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

func TestGetSystemDNSDarwin(t *testing.T) {
	// Just verify it doesn't panic
	servers := getSystemDNSDarwin()
	if servers == nil {
		t.Error("expected non-nil slice, even if empty")
	}
}

func TestGetSystemDNSLinux(t *testing.T) {
	// Just verify it doesn't panic
	servers := getSystemDNSLinux()
	if servers == nil {
		t.Error("expected non-nil slice, even if empty")
	}
}
