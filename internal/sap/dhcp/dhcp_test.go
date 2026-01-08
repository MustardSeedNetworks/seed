package dhcp_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/dhcp"
)

func TestDefaultThresholds(t *testing.T) {
	thresholds := dhcp.DefaultThresholds()

	if thresholds.Warning != 500*time.Millisecond {
		t.Errorf("expected warning threshold 500ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 2000*time.Millisecond {
		t.Errorf("expected critical threshold 2000ms, got %v", thresholds.Critical)
	}
}

func TestNewTester(t *testing.T) {
	tester := dhcp.NewTester("eth0", dhcp.DefaultThresholds())
	if tester == nil {
		t.Fatal("NewTester returned nil")
	}

	if tester.TesterInterfaceName() != "eth0" {
		t.Errorf("expected interface 'eth0', got %s", tester.TesterInterfaceName())
	}
	if tester.TesterTestTimeout() != dhcp.DefaultTestTimeout {
		t.Errorf("expected timeout %v, got %v", dhcp.DefaultTestTimeout, tester.TesterTestTimeout())
	}
}

func TestNewTesterEmptyInterface(t *testing.T) {
	tester := dhcp.NewTester("", dhcp.DefaultThresholds())
	if tester == nil {
		t.Fatal("NewTester returned nil")
	}

	if tester.TesterInterfaceName() != "" {
		t.Errorf("expected empty interface, got %s", tester.TesterInterfaceName())
	}
}

func TestSetInterface(t *testing.T) {
	tester := dhcp.NewTester("eth0", dhcp.DefaultThresholds())
	tester.SetInterface("en0")

	if tester.GetInterface() != "en0" {
		t.Errorf("expected interface 'en0', got %s", tester.GetInterface())
	}
}

func TestGetInterface(t *testing.T) {
	tester := dhcp.NewTester("wlan0", dhcp.DefaultThresholds())

	if tester.GetInterface() != "wlan0" {
		t.Errorf("expected interface 'wlan0', got %s", tester.GetInterface())
	}
}

func TestSetTimeout(t *testing.T) {
	tester := dhcp.NewTester("eth0", dhcp.DefaultThresholds())

	err := tester.SetTimeout(5 * time.Second)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if tester.GetTimeout() != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", tester.GetTimeout())
	}
}

func TestSetTimeoutInvalid(t *testing.T) {
	tester := dhcp.NewTester("eth0", dhcp.DefaultThresholds())

	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"too short", 500 * time.Millisecond, true},
		{"too long", 120 * time.Second, true},
		{"exactly min", dhcp.MinDHCPTimeout, false},
		{"exactly max", dhcp.MaxDHCPTimeout, false},
		{"valid 10s", 10 * time.Second, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tester.SetTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetTimeout(%v) error = %v, wantErr %v", tt.timeout, err, tt.wantErr)
			}
		})
	}
}

func TestSetThresholds(t *testing.T) {
	tester := dhcp.NewTester("eth0", dhcp.DefaultThresholds())

	newThresholds := dhcp.Thresholds{
		Warning:  300 * time.Millisecond,
		Critical: 1000 * time.Millisecond,
	}

	tester.SetThresholds(newThresholds)
	got := tester.GetThresholds()

	if got.Warning != newThresholds.Warning {
		t.Errorf("expected warning %v, got %v", newThresholds.Warning, got.Warning)
	}
	if got.Critical != newThresholds.Critical {
		t.Errorf("expected critical %v, got %v", newThresholds.Critical, got.Critical)
	}
}

func TestGetLastResultNil(t *testing.T) {
	tester := dhcp.NewTester("eth0", dhcp.DefaultThresholds())

	result := tester.GetLastResult()
	if result != nil {
		t.Error("expected nil result before any test")
	}
}

func TestGetStatus(t *testing.T) {
	thresholds := dhcp.Thresholds{
		Warning:  500 * time.Millisecond,
		Critical: 2000 * time.Millisecond,
	}
	tester := dhcp.NewTester("eth0", thresholds)

	tests := []struct {
		name     string
		duration time.Duration
		hasError bool
		expected dhcp.Status
	}{
		{
			name:     "success - fast",
			duration: 200 * time.Millisecond,
			hasError: false,
			expected: dhcp.StatusSuccess,
		},
		{
			name:     "warning - slow",
			duration: 800 * time.Millisecond,
			hasError: false,
			expected: dhcp.StatusWarning,
		},
		{
			name:     "error - very slow",
			duration: 2500 * time.Millisecond,
			hasError: false,
			expected: dhcp.StatusError,
		},
		{
			name:     "error - has error",
			duration: 200 * time.Millisecond,
			hasError: true,
			expected: dhcp.StatusError,
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

func TestGetStatusEdgeCases(t *testing.T) {
	thresholds := dhcp.Thresholds{
		Warning:  500 * time.Millisecond,
		Critical: 2000 * time.Millisecond,
	}
	tester := dhcp.NewTester("eth0", thresholds)

	// Exactly at warning threshold
	status := tester.GetStatus(500*time.Millisecond, false)
	if status != dhcp.StatusWarning {
		t.Errorf("expected StatusWarning at warning threshold, got %v", status)
	}

	// Exactly at critical threshold
	status = tester.GetStatus(2000*time.Millisecond, false)
	if status != dhcp.StatusError {
		t.Errorf("expected StatusError at critical threshold, got %v", status)
	}

	// Just below warning
	status = tester.GetStatus(499*time.Millisecond, false)
	if status != dhcp.StatusSuccess {
		t.Errorf("expected StatusSuccess just below warning, got %v", status)
	}
}

func TestTestNoInterface(t *testing.T) {
	tester := dhcp.NewTester("", dhcp.DefaultThresholds())
	ctx := context.Background()

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}

	if result.Success {
		t.Error("expected failure with no interface")
	}
	if result.Status != dhcp.StatusError {
		t.Errorf("expected StatusError, got %v", result.Status)
	}
	if result.Error != "no interface specified" {
		t.Errorf("expected 'no interface specified' error, got %q", result.Error)
	}
}

func TestTestInvalidInterface(t *testing.T) {
	tester := dhcp.NewTester("nonexistent-interface-xyz", dhcp.DefaultThresholds())
	ctx := context.Background()

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}

	if result.Success {
		t.Error("expected failure with invalid interface")
	}
	if result.Status != dhcp.StatusError {
		t.Errorf("expected StatusError, got %v", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message for invalid interface")
	}
}

func TestStatusConstants(t *testing.T) {
	if dhcp.StatusSuccess != "success" {
		t.Error("StatusSuccess should be 'success'")
	}
	if dhcp.StatusWarning != "warning" {
		t.Error("StatusWarning should be 'warning'")
	}
	if dhcp.StatusError != "error" {
		t.Error("StatusError should be 'error'")
	}
	if dhcp.StatusUnknown != "unknown" {
		t.Error("StatusUnknown should be 'unknown'")
	}
}

func TestValidateDHCPTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"valid 5s", 5 * time.Second, false},
		{"valid 1s", time.Second, false},
		{"valid 60s", 60 * time.Second, false},
		{"too short", 500 * time.Millisecond, true},
		{"too long", 120 * time.Second, true},
		{"exactly min", dhcp.MinDHCPTimeout, false},
		{"exactly max", dhcp.MaxDHCPTimeout, false},
		{"below min", dhcp.MinDHCPTimeout - 1, true},
		{"above max", dhcp.MaxDHCPTimeout + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dhcp.ValidateDHCPTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"ValidateDHCPTimeout(%v) error = %v, wantErr %v",
					tt.timeout,
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestTimeoutError(t *testing.T) {
	err := &dhcp.TimeoutError{
		Value: 500 * time.Millisecond,
		Min:   dhcp.MinDHCPTimeout,
		Max:   dhcp.MaxDHCPTimeout,
	}

	msg := err.Error()
	if msg == "" {
		t.Error("TimeoutError.Error() should return non-empty string")
	}
	if len(msg) < 10 {
		t.Error("TimeoutError.Error() message too short")
	}
}

func TestDHCPTimeoutConstants(t *testing.T) {
	if dhcp.MinDHCPTimeout != 1*time.Second {
		t.Errorf("MinDHCPTimeout = %v, want 1s", dhcp.MinDHCPTimeout)
	}
	if dhcp.MaxDHCPTimeout != 60*time.Second {
		t.Errorf("MaxDHCPTimeout = %v, want 60s", dhcp.MaxDHCPTimeout)
	}
}

func TestThresholdConstants(t *testing.T) {
	if dhcp.DefaultWarningThresholdMs != 500 {
		t.Errorf(
			"DefaultWarningThresholdMs = %d, want 500",
			dhcp.DefaultWarningThresholdMs,
		)
	}
	if dhcp.DefaultCriticalThresholdMs != 2000 {
		t.Errorf(
			"DefaultCriticalThresholdMs = %d, want 2000",
			dhcp.DefaultCriticalThresholdMs,
		)
	}
}

func TestLeaseInfoFields(t *testing.T) {
	now := time.Now()
	lease := dhcp.LeaseInfo{
		Interface:    "eth0",
		IPAddress:    "192.168.1.100",
		SubnetMask:   "255.255.255.0",
		Gateway:      "192.168.1.1",
		ServerIP:     "192.168.1.1",
		DNSServers:   []string{"8.8.8.8", "8.8.4.4"},
		DomainName:   "local",
		LeaseTime:    86400 * time.Second,
		LeaseTimeSec: 86400,
		RenewTime:    43200 * time.Second,
		RebindTime:   75600 * time.Second,
		Expiry:       now.Add(86400 * time.Second),
		ObtainedAt:   now,
	}

	if lease.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", lease.Interface)
	}
	if lease.IPAddress != "192.168.1.100" {
		t.Errorf("expected IPAddress '192.168.1.100', got %q", lease.IPAddress)
	}
	if lease.SubnetMask != "255.255.255.0" {
		t.Errorf("expected SubnetMask '255.255.255.0', got %q", lease.SubnetMask)
	}
	if lease.Gateway != "192.168.1.1" {
		t.Errorf("expected Gateway '192.168.1.1', got %q", lease.Gateway)
	}
	if lease.ServerIP != "192.168.1.1" {
		t.Errorf("expected ServerIP '192.168.1.1', got %q", lease.ServerIP)
	}
	if len(lease.DNSServers) != 2 {
		t.Errorf("expected 2 DNS servers, got %d", len(lease.DNSServers))
	}
	if lease.DomainName != "local" {
		t.Errorf("expected DomainName 'local', got %q", lease.DomainName)
	}
	if lease.LeaseTime != 86400*time.Second {
		t.Errorf("expected LeaseTime 86400s, got %v", lease.LeaseTime)
	}
	if lease.LeaseTimeSec != 86400 {
		t.Errorf("expected LeaseTimeSec 86400, got %d", lease.LeaseTimeSec)
	}
}

func TestTestResultFields(t *testing.T) {
	now := time.Now()
	result := dhcp.TestResult{
		Interface:    "eth0",
		Success:      true,
		Status:       dhcp.StatusSuccess,
		ServerIP:     "192.168.1.1",
		OfferedIP:    "192.168.1.100",
		SubnetMask:   "255.255.255.0",
		Gateway:      "192.168.1.1",
		DNSServers:   []string{"8.8.8.8"},
		DomainName:   "local",
		LeaseTime:    86400 * time.Second,
		LeaseTimeSec: 86400,
		ResponseTime: 150 * time.Millisecond,
		ResponseMs:   150.0,
		Error:        "",
		TestedAt:     now,
	}

	if result.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", result.Interface)
	}
	if !result.Success {
		t.Error("expected Success true")
	}
	if result.Status != dhcp.StatusSuccess {
		t.Errorf("expected StatusSuccess, got %v", result.Status)
	}
	if result.ServerIP != "192.168.1.1" {
		t.Errorf("expected ServerIP '192.168.1.1', got %q", result.ServerIP)
	}
	if result.OfferedIP != "192.168.1.100" {
		t.Errorf("expected OfferedIP '192.168.1.100', got %q", result.OfferedIP)
	}
	if result.ResponseTime != 150*time.Millisecond {
		t.Errorf("expected ResponseTime 150ms, got %v", result.ResponseTime)
	}
	if result.ResponseMs != 150.0 {
		t.Errorf("expected ResponseMs 150.0, got %v", result.ResponseMs)
	}
}

func TestThresholdsFields(t *testing.T) {
	thresholds := dhcp.Thresholds{
		Warning:  300 * time.Millisecond,
		Critical: 1500 * time.Millisecond,
	}

	if thresholds.Warning != 300*time.Millisecond {
		t.Errorf("expected Warning 300ms, got %v", thresholds.Warning)
	}
	if thresholds.Critical != 1500*time.Millisecond {
		t.Errorf("expected Critical 1500ms, got %v", thresholds.Critical)
	}
}

func TestInterfaceError(t *testing.T) {
	err := &dhcp.InterfaceError{Message: "test error message"}

	if err.Error() != "test error message" {
		t.Errorf("expected 'test error message', got %q", err.Error())
	}
}

func TestGetSystemInterfaces(t *testing.T) {
	interfaces, err := dhcp.GetSystemInterfaces()
	if err != nil {
		t.Fatalf("GetSystemInterfaces failed: %v", err)
	}

	// Should return at least an empty slice, not nil
	if interfaces == nil {
		t.Error("GetSystemInterfaces returned nil, expected empty slice at minimum")
	}
}

func TestIsValidIPAddress(t *testing.T) {
	tests := []struct {
		name  string
		addr  string
		valid bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv4 zeros", "0.0.0.0", true},
		{"valid IPv4 broadcast", "255.255.255.255", true},
		{"valid IPv6", "::1", true},
		{"valid IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"invalid - empty", "", false},
		{"invalid - text", "not-an-ip", false},
		{"invalid - partial", "192.168.1", false},
		{"invalid - too many octets", "192.168.1.1.1", false},
		{"invalid - out of range", "256.256.256.256", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.IsValidIPAddress(tt.addr)
			if result != tt.valid {
				t.Errorf("IsValidIPAddress(%q) = %v, want %v", tt.addr, result, tt.valid)
			}
		})
	}
}

func TestIsValidSubnetMask(t *testing.T) {
	tests := []struct {
		name  string
		mask  string
		valid bool
	}{
		{"valid /24", "255.255.255.0", true},
		{"valid /16", "255.255.0.0", true},
		{"valid /8", "255.0.0.0", true},
		{"valid /32", "255.255.255.255", true},
		{"valid /0", "0.0.0.0", true},
		{"valid /25", "255.255.255.128", true},
		{"invalid - gap", "255.0.255.0", false},
		{"invalid - not contiguous", "255.255.128.255", false},
		{"invalid - text", "not-a-mask", false},
		{"invalid - empty", "", false},
		{"invalid - IPv6", "::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.IsValidSubnetMask(tt.mask)
			if result != tt.valid {
				t.Errorf("IsValidSubnetMask(%q) = %v, want %v", tt.mask, result, tt.valid)
			}
		})
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		wantIP   string
		wantMask string
		wantErr  bool
	}{
		{"valid /24", "192.168.1.100/24", "192.168.1.100", "255.255.255.0", false},
		{"valid /16", "10.0.5.1/16", "10.0.5.1", "255.255.0.0", false},
		{"valid /8", "172.16.0.1/8", "172.16.0.1", "255.0.0.0", false},
		{"valid /32", "8.8.8.8/32", "8.8.8.8", "255.255.255.255", false},
		{"invalid - no prefix", "192.168.1.100", "", "", true},
		{"invalid - bad IP", "not-an-ip/24", "", "", true},
		{"invalid - bad prefix", "192.168.1.100/99", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, mask, err := dhcp.ParseCIDR(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCIDR(%q) error = %v, wantErr %v", tt.cidr, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if ip != tt.wantIP {
					t.Errorf("ParseCIDR(%q) IP = %q, want %q", tt.cidr, ip, tt.wantIP)
				}
				if mask != tt.wantMask {
					t.Errorf("ParseCIDR(%q) mask = %q, want %q", tt.cidr, mask, tt.wantMask)
				}
			}
		})
	}
}

func TestCalculateNetworkAddress(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		mask    string
		want    string
		wantErr bool
	}{
		{"valid /24", "192.168.1.100", "255.255.255.0", "192.168.1.0", false},
		{"valid /16", "10.5.20.50", "255.255.0.0", "10.5.0.0", false},
		{"valid /8", "172.16.50.100", "255.0.0.0", "172.0.0.0", false},
		{"valid /32", "8.8.8.8", "255.255.255.255", "8.8.8.8", false},
		{"invalid IP", "not-an-ip", "255.255.255.0", "", true},
		{"invalid mask", "192.168.1.100", "not-a-mask", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dhcp.CalculateNetworkAddress(tt.ip, tt.mask)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"CalculateNetworkAddress(%q, %q) error = %v, wantErr %v",
					tt.ip, tt.mask, err, tt.wantErr,
				)
				return
			}
			if !tt.wantErr && result != tt.want {
				t.Errorf(
					"CalculateNetworkAddress(%q, %q) = %q, want %q",
					tt.ip, tt.mask, result, tt.want,
				)
			}
		})
	}
}

func TestCalculateBroadcastAddress(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		mask    string
		want    string
		wantErr bool
	}{
		{"valid /24", "192.168.1.100", "255.255.255.0", "192.168.1.255", false},
		{"valid /16", "10.5.20.50", "255.255.0.0", "10.5.255.255", false},
		{"valid /8", "172.16.50.100", "255.0.0.0", "172.255.255.255", false},
		{"valid /32", "8.8.8.8", "255.255.255.255", "8.8.8.8", false},
		{"invalid IP", "not-an-ip", "255.255.255.0", "", true},
		{"invalid mask", "192.168.1.100", "not-a-mask", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dhcp.CalculateBroadcastAddress(tt.ip, tt.mask)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"CalculateBroadcastAddress(%q, %q) error = %v, wantErr %v",
					tt.ip, tt.mask, err, tt.wantErr,
				)
				return
			}
			if !tt.wantErr && result != tt.want {
				t.Errorf(
					"CalculateBroadcastAddress(%q, %q) = %q, want %q",
					tt.ip, tt.mask, result, tt.want,
				)
			}
		})
	}
}

func TestConcurrentTesterAccess(t *testing.T) {
	tester := dhcp.NewTester("eth0", dhcp.DefaultThresholds())

	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			for range 50 {
				tester.SetInterface("eth" + string(rune('0'+id)))
				_ = tester.GetInterface()
				_ = tester.GetTimeout()
				_ = tester.GetThresholds()
				_ = tester.GetLastResult()
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}
}

func TestMultipleInterfaceChanges(t *testing.T) {
	tester := dhcp.NewTester("eth0", dhcp.DefaultThresholds())

	interfaces := []string{"eth0", "eth1", "en0", "wlan0", ""}
	for _, iface := range interfaces {
		tester.SetInterface(iface)
		if got := tester.GetInterface(); got != iface {
			t.Errorf("expected interface %q, got %q", iface, got)
		}
	}
}

func TestThresholdsZeroValues(t *testing.T) {
	thresholds := dhcp.Thresholds{}

	if thresholds.Warning != 0 {
		t.Error("expected Warning 0")
	}
	if thresholds.Critical != 0 {
		t.Error("expected Critical 0")
	}
}

func TestTesterWithZeroThresholds(t *testing.T) {
	thresholds := dhcp.Thresholds{}
	tester := dhcp.NewTester("eth0", thresholds)

	// With zero thresholds, any latency >= 0 should be critical
	status := tester.GetStatus(1*time.Millisecond, false)
	if status != dhcp.StatusError {
		t.Errorf("expected StatusError with zero thresholds, got %v", status)
	}
}

func TestLeaseInfoZeroValues(t *testing.T) {
	lease := dhcp.LeaseInfo{}

	if lease.Interface != "" {
		t.Error("expected empty Interface")
	}
	if lease.IPAddress != "" {
		t.Error("expected empty IPAddress")
	}
	if lease.SubnetMask != "" {
		t.Error("expected empty SubnetMask")
	}
	if lease.Gateway != "" {
		t.Error("expected empty Gateway")
	}
	if lease.ServerIP != "" {
		t.Error("expected empty ServerIP")
	}
	if lease.DNSServers != nil {
		t.Error("expected nil DNSServers")
	}
	if lease.DomainName != "" {
		t.Error("expected empty DomainName")
	}
	if lease.LeaseTime != 0 {
		t.Error("expected LeaseTime 0")
	}
	if lease.LeaseTimeSec != 0 {
		t.Error("expected LeaseTimeSec 0")
	}
}

func TestTestResultZeroValues(t *testing.T) {
	result := dhcp.TestResult{}

	if result.Interface != "" {
		t.Error("expected empty Interface")
	}
	if result.Success {
		t.Error("expected Success false")
	}
	if result.Status != "" {
		t.Error("expected empty Status")
	}
	if result.ServerIP != "" {
		t.Error("expected empty ServerIP")
	}
	if result.OfferedIP != "" {
		t.Error("expected empty OfferedIP")
	}
	if result.Error != "" {
		t.Error("expected empty Error")
	}
}

func TestGetCurrentLeaseNoInterface(t *testing.T) {
	tester := dhcp.NewTester("", dhcp.DefaultThresholds())

	lease, err := tester.GetCurrentLease()
	if err == nil {
		t.Error("expected error for no interface")
	}
	if lease != nil {
		t.Error("expected nil lease for no interface")
	}
}

func TestGetCurrentLeaseInvalidInterface(t *testing.T) {
	tester := dhcp.NewTester("nonexistent-interface-xyz", dhcp.DefaultThresholds())

	lease, err := tester.GetCurrentLease()
	if err == nil {
		t.Error("expected error for invalid interface")
	}
	if lease != nil {
		t.Error("expected nil lease for invalid interface")
	}
}

func TestTestWithContextCancellation(t *testing.T) {
	tester := dhcp.NewTester("lo0", dhcp.DefaultThresholds())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := tester.Test(ctx)
	if result == nil {
		t.Fatal("Test returned nil")
	}
	// Test should either fail due to context or interface issues
	// We just verify it doesn't panic and returns a result
	if result.Status == "" {
		t.Error("expected non-empty status")
	}
}

func TestTesterSetTestTimeoutDirect(t *testing.T) {
	tester := dhcp.NewTester("eth0", dhcp.DefaultThresholds())

	// Use the test helper to bypass validation
	tester.TesterSetTestTimeout(30 * time.Second)

	if tester.TesterTestTimeout() != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", tester.TesterTestTimeout())
	}
}

func TestCalculateNetworkAddressIPv6Error(t *testing.T) {
	// IPv6 addresses should return an error for these IPv4-only functions
	_, err := dhcp.CalculateNetworkAddress("::1", "ffff:ffff:ffff:ffff::")
	if err == nil {
		t.Error("expected error for IPv6 addresses")
	}
}

func TestCalculateBroadcastAddressIPv6Error(t *testing.T) {
	// IPv6 addresses should return an error for these IPv4-only functions
	_, err := dhcp.CalculateBroadcastAddress("::1", "ffff:ffff:ffff:ffff::")
	if err == nil {
		t.Error("expected error for IPv6 addresses")
	}
}

func TestIsContiguousMask(t *testing.T) {
	tests := []struct {
		name       string
		mask       []byte
		contiguous bool
	}{
		{"valid /24", []byte{255, 255, 255, 0}, true},
		{"valid /16", []byte{255, 255, 0, 0}, true},
		{"valid /8", []byte{255, 0, 0, 0}, true},
		{"valid /32", []byte{255, 255, 255, 255}, true},
		{"valid /0", []byte{0, 0, 0, 0}, true},
		{"valid /25", []byte{255, 255, 255, 128}, true},
		{"valid /1", []byte{128, 0, 0, 0}, true},
		{"invalid - wrong length", []byte{255, 255, 255}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.ExportIsContiguousMask(tt.mask)
			if result != tt.contiguous {
				t.Errorf(
					"ExportIsContiguousMask(%v) = %v, want %v",
					tt.mask,
					result,
					tt.contiguous,
				)
			}
		})
	}
}

func TestDefaultTestTimeout(t *testing.T) {
	if dhcp.DefaultTestTimeout != 10*time.Second {
		t.Errorf("DefaultTestTimeout = %v, want 10s", dhcp.DefaultTestTimeout)
	}
}

func TestTesterCopyResult(t *testing.T) {
	tester := dhcp.NewTester("nonexistent-interface", dhcp.DefaultThresholds())
	ctx := context.Background()

	// Run a test to populate lastResult
	_ = tester.Test(ctx)

	// Get the result twice
	result1 := tester.GetLastResult()
	result2 := tester.GetLastResult()

	if result1 == nil || result2 == nil {
		t.Fatal("expected non-nil results")
	}

	// Modify result1 and verify result2 is unaffected
	result1.Interface = "modified"
	result3 := tester.GetLastResult()

	if result3 != nil && result3.Interface == "modified" {
		t.Error("GetLastResult should return a copy, not the internal reference")
	}
}
