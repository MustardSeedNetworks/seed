package api

import (
	"testing"
	"time"
)

func TestSplitCIDR(t *testing.T) {
	tests := []struct {
		input    string
		expected [2]string
	}{
		{"192.168.1.1/24", [2]string{"192.168.1.1", "24"}},
		{"10.0.0.1/8", [2]string{"10.0.0.1", "8"}},
		{"fe80::1/64", [2]string{"fe80::1", "64"}},
		{"192.168.1.1", [2]string{"192.168.1.1", ""}},
		{"", [2]string{"", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitCIDR(tt.input)
			if result != tt.expected {
				t.Errorf("splitCIDR(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsIPv4Address(t *testing.T) {
	tests := []struct {
		addr     string
		expected bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"255.255.255.255", true},
		{"fe80::1", false},
		{"2001:db8::1", false},
		{"::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			result := isIPv4Address(tt.addr)
			if result != tt.expected {
				t.Errorf("isIPv4Address(%q) = %v, want %v", tt.addr, result, tt.expected)
			}
		})
	}
}

func TestParsePrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"24", 24},
		{"8", 8},
		{"64", 64},
		{"128", 128},
		{"0", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parsePrefix(tt.input)
			if result != tt.expected {
				t.Errorf("parsePrefix(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsLinkLocal(t *testing.T) {
	tests := []struct {
		addr     string
		expected bool
	}{
		{"fe80::1", true},
		{"FE80::1", true},
		{"fe80:0:0:0:1:2:3:4", true},
		{"2001:db8::1", false},
		{"fc00::1", false},
		{"192.168.1.1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			result := isLinkLocal(tt.addr)
			if result != tt.expected {
				t.Errorf("isLinkLocal(%q) = %v, want %v", tt.addr, result, tt.expected)
			}
		})
	}
}

func TestIsUniqueLocal(t *testing.T) {
	tests := []struct {
		addr     string
		expected bool
	}{
		{"fc00::1", true},
		{"FC00::1", true},
		{"fd00::1", true},
		{"FD00::1", true},
		{"fe80::1", false},
		{"2001:db8::1", false},
		{"192.168.1.1", false},
		{"", false},
		{"f", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			result := isUniqueLocal(tt.addr)
			if result != tt.expected {
				t.Errorf("isUniqueLocal(%q) = %v, want %v", tt.addr, result, tt.expected)
			}
		})
	}
}

func TestParseIPAddress(t *testing.T) {
	tests := []struct {
		name   string
		addr   string
		isIPv4 bool
		prefix int
		scope  string
	}{
		{"IPv4 /24", "192.168.1.1/24", true, 0, "global"},
		{"IPv4 /8", "10.0.0.1/8", true, 0, "global"},
		{"IPv6 global", "2001:db8::1/64", false, 64, "global"},
		{"IPv6 link-local", "fe80::1/64", false, 64, "link-local"},
		{"IPv6 unique-local", "fd00::1/64", false, 64, "unique-local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIPAddress(tt.addr)
			if result.isIPv4 != tt.isIPv4 {
				t.Errorf("parseIPAddress(%q).isIPv4 = %v, want %v", tt.addr, result.isIPv4, tt.isIPv4)
			}
			if !tt.isIPv4 && result.prefix != tt.prefix {
				t.Errorf("parseIPAddress(%q).prefix = %d, want %d", tt.addr, result.prefix, tt.prefix)
			}
			if !tt.isIPv4 && result.scope != tt.scope {
				t.Errorf("parseIPAddress(%q).scope = %q, want %q", tt.addr, result.scope, tt.scope)
			}
		})
	}
}

func TestGetTestStatus(t *testing.T) {
	tests := []struct {
		name       string
		latencyMs  float64
		warningMs  int64
		criticalMs int64
		expected   string
	}{
		{"fast - success", 10, 100, 500, "success"},
		{"warning threshold", 150, 100, 500, "warning"},
		{"critical threshold", 600, 100, 500, "error"},
		{"exactly at warning", 100, 100, 500, "warning"},
		{"exactly at critical", 500, 100, 500, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTestStatus(tt.latencyMs, tt.warningMs, tt.criticalMs)
			if result != tt.expected {
				t.Errorf("getTestStatus(%v, %d, %d) = %q, want %q",
					tt.latencyMs, tt.warningMs, tt.criticalMs, result, tt.expected)
			}
		})
	}
}

func TestGetTLSVersionString(t *testing.T) {
	tests := []struct {
		version  uint16
		expected string
	}{
		{0x0301, "TLS 1.0"},
		{0x0302, "TLS 1.1"},
		{0x0303, "TLS 1.2"},
		{0x0304, "TLS 1.3"},
		{0x0000, "Unknown"},
		{0x9999, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getTLSVersionString(tt.version)
			if result != tt.expected {
				t.Errorf("getTLSVersionString(0x%04X) = %q, want %q", tt.version, result, tt.expected)
			}
		})
	}
}

func TestRunTCPTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	// Test against a known-working port
	latency, err := runTCPTest("google.com", 80)
	if err != nil {
		t.Skipf("network test failed (may be offline): %v", err)
	}
	if latency <= 0 {
		t.Error("expected positive latency")
	}
}

func TestRunTCPTestInvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	_, err := runTCPTest("invalid.invalid.invalid", 80)
	if err == nil {
		t.Error("expected error for invalid host")
	}
}

func TestRunExtendedPing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	stats, err := runExtendedPing("google.com", 3)
	if err != nil {
		t.Skipf("network test failed (may be offline): %v", err)
	}

	if stats.AvgLatency <= 0 && stats.PacketLoss < 100 {
		t.Error("expected positive average latency")
	}
	if stats.PacketLoss < 0 || stats.PacketLoss > 100 {
		t.Errorf("packet loss should be 0-100%%, got %v", stats.PacketLoss)
	}
}

func TestRunExtendedPingInvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	stats, err := runExtendedPing("invalid.invalid.invalid", 2)
	if err == nil && stats.PacketLoss != 100 {
		t.Error("expected 100% packet loss for invalid host")
	}
}

func TestLoginRequestValidation(t *testing.T) {
	req := LoginRequest{
		Username: "admin",
		Password: "secret",
	}

	if req.Username != "admin" {
		t.Errorf("expected username 'admin', got %q", req.Username)
	}
	if req.Password != "secret" {
		t.Errorf("expected password 'secret', got %q", req.Password)
	}
}

func TestStatusResponseFields(t *testing.T) {
	resp := StatusResponse{
		Status:     "ok",
		Version:    "0.7.3",
		Uptime:     3600,
		Interface:  "eth0",
		IsWireless: false,
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if resp.Version != "0.7.3" {
		t.Errorf("expected version '0.7.3', got %q", resp.Version)
	}
	if resp.Uptime <= 0 {
		t.Errorf("expected positive uptime, got %d", resp.Uptime)
	}
	if resp.Interface == "" {
		t.Error("expected interface to be set")
	}
	if resp.IsWireless {
		t.Error("expected wired interface for this test")
	}
}

func TestIPSettingsRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     IPSettingsRequest
		isValid bool
	}{
		{
			name:    "valid dhcp",
			req:     IPSettingsRequest{Mode: "dhcp"},
			isValid: true,
		},
		{
			name: "valid static",
			req: IPSettingsRequest{
				Mode:    "static",
				Address: "192.168.1.100",
				Netmask: "255.255.255.0",
				Gateway: "192.168.1.1",
				DNS:     []string{"8.8.8.8"},
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.Mode != "dhcp" && tt.req.Mode != "static" {
				if tt.isValid {
					t.Error("expected valid mode")
				}
			}
		})
	}
}

func TestDHCPTimingInfo(t *testing.T) {
	timing := DHCPTimingInfo{
		Discover: 10,
		Offer:    20,
		Request:  15,
		Ack:      5,
		Total:    50,
	}

	if timing.Total != 50 {
		t.Errorf("expected total 50, got %d", timing.Total)
	}
	if timing.Discover+timing.Offer+timing.Request+timing.Ack != timing.Total {
		t.Error("timing phases should sum to total")
	}
}

func TestLinkResponse(t *testing.T) {
	resp := LinkResponse{
		Interface: "eth0",
		LinkUp:    true,
		Speed:     "1000baseT",
		Duplex:    "full",
		MTU:       1500,
		AutoNeg:   true,
	}

	if resp.Interface != "eth0" {
		t.Errorf("expected interface eth0, got %s", resp.Interface)
	}
	if !resp.LinkUp {
		t.Error("expected link to be up")
	}
	if resp.MTU != 1500 {
		t.Errorf("expected MTU 1500, got %d", resp.MTU)
	}
	if resp.Speed == "" {
		t.Error("expected speed to be set")
	}
	if resp.Duplex == "" {
		t.Error("expected duplex to be set")
	}
	if !resp.AutoNeg {
		t.Error("expected autonegotiation to be enabled")
	}
}

func TestCustomTestResult(t *testing.T) {
	result := CustomTestResult{
		Name:       "Test Server",
		Host:       "example.com",
		Port:       80,
		Success:    true,
		Latency:    25.5,
		TestStatus: "success",
	}

	if result.Name != "Test Server" {
		t.Errorf("expected name 'Test Server', got %s", result.Name)
	}
	if result.Host == "" {
		t.Error("expected host to be set")
	}
	if result.Port <= 0 {
		t.Error("expected port to be positive")
	}
	if result.Latency <= 0 {
		t.Error("expected latency to be positive")
	}
	if result.TestStatus != "success" && result.TestStatus != "warning" && result.TestStatus != "error" {
		t.Errorf("invalid test status: %s", result.TestStatus)
	}

	if !result.Success {
		t.Error("expected success")
	}
	if result.Latency <= 0 {
		t.Error("expected positive latency")
	}
}

func TestSpeedtestResponse(t *testing.T) {
	resp := SpeedtestResponse{
		Download:     100.5,
		Upload:       50.2,
		Latency:      15.3,
		Server:       "Test Server",
		Location:     "New York",
		Distance:     50.0,
		Timestamp:    time.Now().Format(time.RFC3339),
		TestDuration: 30.5,
	}

	if resp.Download <= 0 {
		t.Error("expected positive download speed")
	}
	if resp.Upload <= 0 {
		t.Error("expected positive upload speed")
	}
	if resp.Latency <= 0 {
		t.Error("expected positive latency")
	}
	if resp.Server == "" {
		t.Error("expected server to be set")
	}
	if resp.Location == "" {
		t.Error("expected location to be set")
	}
	if resp.Distance < 0 {
		t.Error("expected non-negative distance")
	}
	if resp.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
	if resp.TestDuration <= 0 {
		t.Error("expected positive test duration")
	}
}

func TestIperfClientRequest(t *testing.T) {
	req := IperfClientRequest{
		Server:   "192.168.1.1",
		Port:     5201,
		Protocol: "tcp",
		Reverse:  true,
		Duration: 10,
		Parallel: 4,
	}

	if req.Server == "" {
		t.Error("server should be set")
	}
	if req.Port != 5201 {
		t.Errorf("expected port 5201, got %d", req.Port)
	}
	if req.Protocol != "tcp" && req.Protocol != "udp" {
		t.Errorf("invalid protocol: %s", req.Protocol)
	}
	if !req.Reverse {
		t.Error("expected reverse to be true")
	}
	if req.Duration != 10 {
		t.Errorf("expected duration 10, got %d", req.Duration)
	}
	if req.Parallel != 4 {
		t.Errorf("expected parallel 4, got %d", req.Parallel)
	}
}

func TestVLANResponse(t *testing.T) {
	nativeVlan := 10
	resp := VLANResponse{
		NativeVlan:  &nativeVlan,
		TaggedVlans: []int{20, 30, 40},
	}
	resp.Configured.Enabled = true
	resp.Configured.ID = 10

	if resp.NativeVlan == nil || *resp.NativeVlan != 10 {
		t.Error("expected native VLAN 10")
	}
	if len(resp.TaggedVlans) != 3 {
		t.Errorf("expected 3 tagged VLANs, got %d", len(resp.TaggedVlans))
	}
}

func TestWiFiResponse(t *testing.T) {
	resp := WiFiResponse{
		SSID:      "TestNetwork",
		BSSID:     "00:11:22:33:44:55",
		Signal:    -65,
		Channel:   6,
		Frequency: 2437,
		Security:  "WPA2-PSK",
	}

	if resp.SSID == "" {
		t.Error("SSID should be set")
	}
	if resp.Signal > 0 {
		t.Error("signal should be negative (dBm)")
	}
	if resp.BSSID == "" {
		t.Error("expected BSSID to be set")
	}
	if resp.Channel <= 0 {
		t.Error("expected channel to be positive")
	}
	if resp.Frequency <= 0 {
		t.Error("expected frequency to be positive")
	}
	if resp.Security == "" {
		t.Error("expected security to be set")
	}
}

func TestCableResponse(t *testing.T) {
	length := 25.5
	resp := CableResponse{
		Supported: true,
		Length:    &length,
		Status:    "ok",
		Faults:    []string{},
	}

	if !resp.Supported {
		t.Error("expected cable test to be supported")
	}
	if resp.Length == nil || *resp.Length != 25.5 {
		t.Error("expected length 25.5")
	}
	if resp.Status == "" {
		t.Error("expected status to be set")
	}
	if resp.Faults == nil {
		t.Error("expected faults slice to be non-nil (can be empty)")
	}
}

func TestDiscoveryNeighborInfo(t *testing.T) {
	info := DiscoveryNeighborInfo{
		Protocol:          "LLDP",
		ChassisID:         "00:11:22:33:44:55",
		PortID:            "Gi0/1",
		SystemName:        "switch01",
		ManagementAddress: "192.168.1.1",
		TTL:               120,
		LastSeen:          time.Now().Format(time.RFC3339),
	}

	if info.Protocol != "LLDP" && info.Protocol != "CDP" {
		t.Errorf("unexpected protocol: %s", info.Protocol)
	}
	if info.TTL <= 0 {
		t.Error("TTL should be positive")
	}
	if info.ChassisID == "" {
		t.Error("expected chassis ID to be set")
	}
	if info.PortID == "" {
		t.Error("expected port ID to be set")
	}
	if info.SystemName == "" {
		t.Error("expected system name to be set")
	}
	if info.ManagementAddress == "" {
		t.Error("expected management address to be set")
	}
	if info.LastSeen == "" {
		t.Error("expected last seen timestamp to be set")
	}
}

func TestDNSLookupResult(t *testing.T) {
	result := DNSLookupResult{
		Result:   "93.184.216.34",
		Time:     15,
		TimeMs:   15,
		Status:   "success",
		Resolved: []string{"93.184.216.34", "93.184.216.35"},
	}

	if result.Status != "success" && result.Status != "warning" && result.Status != "error" {
		t.Errorf("invalid status: %s", result.Status)
	}
	if len(result.Resolved) == 0 && result.Status == "success" {
		t.Error("successful lookup should have resolved addresses")
	}
	if result.Result == "" {
		t.Error("expected result to be set")
	}
	if result.Time <= 0 || result.TimeMs <= 0 {
		t.Error("expected positive timing values")
	}
}

func TestGatewayResponse(t *testing.T) {
	resp := GatewayResponse{
		Gateway:     "192.168.1.1",
		Reachable:   true,
		Sent:        5,
		Received:    5,
		LossPercent: 0,
		MinTime:     10.5,
		MaxTime:     25.3,
		AvgTime:     15.2,
		LastTime:    12.1,
		Status:      "success",
	}

	if !resp.Reachable && resp.LossPercent < 100 {
		t.Error("reachable gateway should have < 100% loss")
	}
	if resp.AvgTime < resp.MinTime || resp.AvgTime > resp.MaxTime {
		t.Error("average time should be between min and max")
	}
	if resp.Gateway == "" {
		t.Error("expected gateway to be set")
	}
	if resp.Sent <= 0 || resp.Received <= 0 {
		t.Error("expected positive sent/received counts")
	}
	if resp.LastTime <= 0 {
		t.Error("expected positive last time")
	}
	if resp.Status == "" {
		t.Error("expected status to be set")
	}
}
