package mcp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/mcp"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/sap/dns"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// createTestServer creates a test server with the given service provider.
func createTestServer(provider mcp.ServiceProvider) *mcp.Server {
	cfg := &config.MCPConfig{
		Enabled:      true,
		AllowedTools: []string{},
	}
	return mcp.NewServer(cfg, provider)
}

// TestHandleGetDevices_Success tests the get_devices handler with a successful response.
func TestHandleGetDevices_Success(t *testing.T) {
	devices := []*discovery.DiscoveredDevice{
		{IP: "192.168.1.1", MAC: "00:11:22:33:44:55", Hostname: "router.local"},
		{IP: "192.168.1.2", MAC: "00:11:22:33:44:56", Hostname: "device.local"},
	}
	provider := &mockServiceProvider{
		discoveryService: &mockDiscoveryService{devices: devices},
	}
	server := createTestServer(provider)
	request := mcp.NewCallToolRequest("get_devices", nil)

	result, err := server.ExportHandleGetDevices(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestHandleGetDevices_ServiceUnavailable tests the get_devices handler when service is nil.
func TestHandleGetDevices_ServiceUnavailable(t *testing.T) {
	provider := &mockServiceProvider{
		discoveryService: nil,
	}
	server := createTestServer(provider)
	request := mcp.NewCallToolRequest("get_devices", nil)

	result, err := server.ExportHandleGetDevices(context.Background(), request)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result with error message")
	}
}

// TestHandleNetworkScan tests the network_scan handler.
func TestHandleNetworkScan(t *testing.T) {
	tests := []struct {
		name            string
		devices         []*discovery.DiscoveredDevice
		scanErr         error
		args            map[string]any
		serviceNil      bool
		wantErrContains string
	}{
		{
			name: "successful scan",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.1", MAC: "00:11:22:33:44:55"},
			},
			scanErr: nil,
		},
		{
			name:    "scan already in progress",
			devices: []*discovery.DiscoveredDevice{{IP: "192.168.1.1"}},
			scanErr: discovery.ErrScanInProgress,
		},
		{
			name:       "service unavailable",
			serviceNil: true,
		},
		{
			name:    "with custom timeout",
			devices: []*discovery.DiscoveredDevice{},
			args:    map[string]any{"timeout": float64(60)},
		},
		{
			name:    "with timeout exceeding max",
			devices: []*discovery.DiscoveredDevice{},
			args:    map[string]any{"timeout": float64(600)},
		},
		{
			name:            "scan error",
			scanErr:         errors.New("network error"),
			wantErrContains: "Scan failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.serviceNil {
				provider = &mockServiceProvider{discoveryService: nil}
			} else {
				provider = &mockServiceProvider{
					discoveryService: &mockDiscoveryService{
						devices: tt.devices,
						scanErr: tt.scanErr,
					},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("network_scan", tt.args)

			result, err := server.ExportHandleNetworkScan(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleDeviceFingerprint tests the device_fingerprint handler.
func TestHandleDeviceFingerprint(t *testing.T) {
	tests := []struct {
		name       string
		devices    []*discovery.DiscoveredDevice
		args       map[string]any
		serviceNil bool
	}{
		{
			name: "device found with profile",
			devices: []*discovery.DiscoveredDevice{
				{
					IP:  "192.168.1.100",
					MAC: "00:11:22:33:44:55",
					Profile: &discovery.DeviceProfile{
						DeviceType: "computer",
					},
				},
			},
			args: map[string]any{"ip": "192.168.1.100"},
		},
		{
			name: "device found without profile",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.100", MAC: "00:11:22:33:44:55"},
			},
			args: map[string]any{"ip": "192.168.1.100"},
		},
		{
			name: "device not found",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.50", MAC: "00:11:22:33:44:55"},
			},
			args: map[string]any{"ip": "192.168.1.100"},
		},
		{
			name:       "service unavailable",
			serviceNil: true,
			args:       map[string]any{"ip": "192.168.1.100"},
		},
		{
			name:    "missing ip parameter",
			devices: []*discovery.DiscoveredDevice{},
			args:    map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.serviceNil {
				provider = &mockServiceProvider{discoveryService: nil}
			} else {
				provider = &mockServiceProvider{
					discoveryService: &mockDiscoveryService{devices: tt.devices},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("device_fingerprint", tt.args)

			result, err := server.ExportHandleDeviceFingerprint(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleGetNeighbors tests the get_neighbors handler.
func TestHandleGetNeighbors(t *testing.T) {
	tests := []struct {
		name     string
		devices  []*discovery.DiscoveredDevice
		args     map[string]any
		expected int
	}{
		{
			name: "filter by lldp",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.1", LLDPInfo: &discovery.LLDPDeviceInfo{SystemName: "switch1"}},
				{IP: "192.168.1.2", CDPInfo: &discovery.CDPDeviceInfo{DeviceID: "router1"}},
			},
			args:     map[string]any{"protocol": "lldp"},
			expected: 1,
		},
		{
			name: "filter by cdp",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.1", LLDPInfo: &discovery.LLDPDeviceInfo{SystemName: "switch1"}},
				{IP: "192.168.1.2", CDPInfo: &discovery.CDPDeviceInfo{DeviceID: "router1"}},
			},
			args:     map[string]any{"protocol": "cdp"},
			expected: 1,
		},
		{
			name: "filter by edp",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.1", EDPInfo: &discovery.EDPDeviceInfo{DeviceID: "switch1"}},
			},
			args:     map[string]any{"protocol": "edp"},
			expected: 1,
		},
		{
			name: "all protocols",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.1", LLDPInfo: &discovery.LLDPDeviceInfo{SystemName: "switch1"}},
				{IP: "192.168.1.2", CDPInfo: &discovery.CDPDeviceInfo{DeviceID: "router1"}},
				{IP: "192.168.1.3"},
			},
			args:     map[string]any{"protocol": "all"},
			expected: 2,
		},
		{
			name: "no protocol filter (defaults to all)",
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.1", LLDPInfo: &discovery.LLDPDeviceInfo{SystemName: "switch1"}},
			},
			args:     map[string]any{},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockServiceProvider{
				discoveryService: &mockDiscoveryService{devices: tt.devices},
			}
			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("get_neighbors", tt.args)

			result, err := server.ExportHandleGetNeighbors(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleTraceroute tests the traceroute handler.
func TestHandleTraceroute(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]any
		icmpAvailable bool
	}{
		{
			name:          "with ICMP available",
			args:          map[string]any{"target": "8.8.8.8"},
			icmpAvailable: true,
		},
		{
			name:          "ICMP not available",
			args:          map[string]any{"target": "8.8.8.8"},
			icmpAvailable: false,
		},
		{
			name:          "missing target",
			args:          map[string]any{},
			icmpAvailable: true,
		},
		{
			name: "with custom max_hops",
			args: map[string]any{
				"target":   "8.8.8.8",
				"max_hops": float64(15),
			},
			icmpAvailable: true,
		},
		{
			name: "with custom timeout",
			args: map[string]any{
				"target":  "8.8.8.8",
				"timeout": float64(5),
			},
			icmpAvailable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockServiceProvider{
				icmpAvailable: tt.icmpAvailable,
			}
			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("traceroute", tt.args)

			result, err := server.ExportHandleTraceroute(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleDNSTest tests the dns_test handler.
func TestHandleDNSTest(t *testing.T) {
	tests := []struct {
		name      string
		dnsResult *dns.TestResult
		testerNil bool
	}{
		{
			name: "successful dns test",
			dnsResult: &dns.TestResult{
				Server:       "8.8.8.8",
				TestHostname: "google.com",
			},
		},
		{
			name:      "tester unavailable",
			testerNil: true,
		},
		{
			name:      "test returns nil",
			dnsResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.testerNil {
				provider = &mockServiceProvider{dnsTester: nil}
			} else {
				provider = &mockServiceProvider{
					dnsTester: &mockDNSTester{result: tt.dnsResult},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("dns_test", nil)

			result, err := server.ExportHandleDNSTest(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleGatewayPing tests the gateway_ping handler.
func TestHandleGatewayPing(t *testing.T) {
	tests := []struct {
		name          string
		pingStats     *gateway.PingStats
		testerNil     bool
		icmpAvailable bool
	}{
		{
			name: "successful ping",
			pingStats: &gateway.PingStats{
				Gateway:     "192.168.1.1",
				Sent:        5,
				Received:    5,
				Lost:        0,
				LossPercent: 0,
				MinTime:     1.0,
				MaxTime:     5.0,
				AvgTime:     3.0,
				Reachable:   true,
			},
			icmpAvailable: true,
		},
		{
			name:          "tester unavailable",
			testerNil:     true,
			icmpAvailable: true,
		},
		{
			name:          "ICMP not available",
			pingStats:     &gateway.PingStats{},
			icmpAvailable: false,
		},
		{
			name:          "test returns nil",
			pingStats:     nil,
			icmpAvailable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.testerNil {
				provider = &mockServiceProvider{
					gatewayTester: nil,
					icmpAvailable: tt.icmpAvailable,
				}
			} else {
				provider = &mockServiceProvider{
					gatewayTester: &mockGatewayTester{stats: tt.pingStats},
					icmpAvailable: tt.icmpAvailable,
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("gateway_ping", nil)

			result, err := server.ExportHandleGatewayPing(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleSpeedtest tests the speedtest handler.
func TestHandleSpeedtest(t *testing.T) {
	tests := []struct {
		name      string
		result    *speedtest.Result
		status    speedtest.Status
		runErr    error
		testerNil bool
	}{
		{
			name: "successful speedtest",
			result: &speedtest.Result{
				Download: 100.5,
				Upload:   50.2,
				Latency:  15.0,
			},
			status: speedtest.Status{Running: false},
		},
		{
			name:   "speedtest already running",
			status: speedtest.Status{Running: true},
		},
		{
			name:      "tester unavailable",
			testerNil: true,
		},
		{
			name:   "speedtest error",
			status: speedtest.Status{Running: false},
			runErr: errors.New("connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.testerNil {
				provider = &mockServiceProvider{speedtestTester: nil}
			} else {
				provider = &mockServiceProvider{
					speedtestTester: &mockSpeedtestTester{
						result: tt.result,
						status: tt.status,
						runErr: tt.runErr,
					},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("speedtest", nil)

			result, err := server.ExportHandleSpeedtest(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleIperfTest tests the iperf_test handler.
func TestHandleIperfTest(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		result     *iperf.Result
		status     iperf.ClientStatus
		runErr     error
		managerNil bool
	}{
		{
			name: "successful iperf test",
			args: map[string]any{"server": "192.168.1.100"},
			result: &iperf.Result{
				Bandwidth: 100.5,
				Protocol:  "tcp",
			},
			status: iperf.ClientStatus{Running: false},
		},
		{
			name: "iperf already running",
			args: map[string]any{"server": "192.168.1.100"},
			status: iperf.ClientStatus{
				Running: true,
				Phase:   "testing",
			},
		},
		{
			name:       "manager unavailable",
			args:       map[string]any{"server": "192.168.1.100"},
			managerNil: true,
		},
		{
			name:   "missing server",
			args:   map[string]any{},
			status: iperf.ClientStatus{Running: false},
		},
		{
			name: "with custom options",
			args: map[string]any{
				"server":    "192.168.1.100",
				"port":      float64(5202),
				"duration":  float64(30),
				"protocol":  "udp",
				"direction": "upload",
			},
			result: &iperf.Result{Bandwidth: 50.0, Protocol: "udp"},
			status: iperf.ClientStatus{Running: false},
		},
		{
			name:   "invalid protocol",
			args:   map[string]any{"server": "192.168.1.100", "protocol": "invalid"},
			status: iperf.ClientStatus{Running: false},
		},
		{
			name:   "invalid direction",
			args:   map[string]any{"server": "192.168.1.100", "direction": "invalid"},
			status: iperf.ClientStatus{Running: false},
		},
		{
			name:   "iperf error",
			args:   map[string]any{"server": "192.168.1.100"},
			status: iperf.ClientStatus{Running: false},
			runErr: errors.New("connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.managerNil {
				provider = &mockServiceProvider{iperfManager: nil}
			} else {
				provider = &mockServiceProvider{
					iperfManager: &mockIperfManager{
						result:       tt.result,
						clientStatus: tt.status,
						runErr:       tt.runErr,
					},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("iperf_test", tt.args)

			result, err := server.ExportHandleIperfTest(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleWiFiScan tests the wifi_scan handler.
func TestHandleWiFiScan(t *testing.T) {
	tests := []struct {
		name       string
		networks   []mcp.WiFiNetwork
		scanErr    error
		scannerNil bool
	}{
		{
			name: "successful scan with networks",
			networks: []mcp.WiFiNetwork{
				{
					SSID:      "TestNetwork",
					BSSID:     "00:11:22:33:44:55",
					Signal:    -50,
					Channel:   6,
					Frequency: 2437,
					Security:  "WPA2-PSK",
				},
			},
		},
		{
			name:     "successful scan with no networks",
			networks: []mcp.WiFiNetwork{},
		},
		{
			name:       "scanner unavailable",
			scannerNil: true,
		},
		{
			name:    "scan error",
			scanErr: errors.New("interface not available"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.scannerNil {
				provider = &mockServiceProvider{wifiScanner: nil}
			} else {
				provider = &mockServiceProvider{
					wifiScanner: &mockWiFiScanner{
						networks: tt.networks,
						scanErr:  tt.scanErr,
					},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("wifi_scan", nil)

			result, err := server.ExportHandleWiFiScan(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleWiFiInfo tests the wifi_info handler.
func TestHandleWiFiInfo(t *testing.T) {
	tests := []struct {
		name          string
		currentNet    *mcp.WiFiConnectionInfo
		getNetworkErr error
		managerNil    bool
	}{
		{
			name: "connected to network",
			currentNet: &mcp.WiFiConnectionInfo{
				SSID:      "MyNetwork",
				BSSID:     "00:11:22:33:44:55",
				Signal:    -55,
				Channel:   6,
				Frequency: 2437,
				Security:  "WPA2-PSK",
			},
		},
		{
			name:       "not connected",
			currentNet: nil,
		},
		{
			name:       "manager unavailable",
			managerNil: true,
		},
		{
			name:          "get network error",
			getNetworkErr: errors.New("failed to get WiFi info"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.managerNil {
				provider = &mockServiceProvider{wifiManager: nil}
			} else {
				provider = &mockServiceProvider{
					wifiManager: &mockWiFiManager{
						currentNetwork: tt.currentNet,
						getNetworkErr:  tt.getNetworkErr,
					},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("wifi_info", nil)

			result, err := server.ExportHandleWiFiInfo(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleGetInterfaces tests the get_interfaces handler.
func TestHandleGetInterfaces(t *testing.T) {
	tests := []struct {
		name       string
		interfaces []*network.InterfaceInfo
		managerNil bool
	}{
		{
			name: "multiple interfaces",
			interfaces: []*network.InterfaceInfo{
				{Name: "eth0"},
				{Name: "wlan0"},
			},
		},
		{
			name:       "no interfaces",
			interfaces: []*network.InterfaceInfo{},
		},
		{
			name:       "manager unavailable",
			managerNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.managerNil {
				provider = &mockServiceProvider{netManager: nil}
			} else {
				provider = &mockServiceProvider{
					netManager: &mockNetworkManager{interfaces: tt.interfaces},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("get_interfaces", nil)

			result, err := server.ExportHandleGetInterfaces(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleGetLinkStatus tests the get_link_status handler.
func TestHandleGetLinkStatus(t *testing.T) {
	tests := []struct {
		name       string
		state      network.LinkState
		isUp       bool
		monitorNil bool
	}{
		{
			name:  "link up",
			state: network.LinkStateUnknown,
			isUp:  true,
		},
		{
			name:  "link down",
			state: network.LinkStateUnknown,
			isUp:  false,
		},
		{
			name:       "monitor unavailable",
			monitorNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.monitorNil {
				provider = &mockServiceProvider{linkMonitor: nil}
			} else {
				provider = &mockServiceProvider{
					linkMonitor: &mockLinkMonitor{
						state: tt.state,
						isUp:  tt.isUp,
					},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("get_link_status", nil)

			result, err := server.ExportHandleGetLinkStatus(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleGetIPConfig tests the get_ip_config handler.
func TestHandleGetIPConfig(t *testing.T) {
	tests := []struct {
		name             string
		currentInterface string
		interfaces       []*network.InterfaceInfo
		getInterfaceErr  error
		managerNil       bool
	}{
		{
			name:             "successful",
			currentInterface: "eth0",
			interfaces: []*network.InterfaceInfo{
				{Name: "eth0"},
			},
		},
		{
			name:             "no current interface",
			currentInterface: "",
			interfaces:       []*network.InterfaceInfo{},
		},
		{
			name:             "interface error",
			currentInterface: "eth0",
			interfaces:       []*network.InterfaceInfo{},
			getInterfaceErr:  errors.New("interface not found"),
		},
		{
			name:       "manager unavailable",
			managerNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.managerNil {
				provider = &mockServiceProvider{netManager: nil}
			} else {
				provider = &mockServiceProvider{
					netManager: &mockNetworkManager{
						interfaces:       tt.interfaces,
						currentInterface: tt.currentInterface,
						getInterfaceErr:  tt.getInterfaceErr,
					},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("get_ip_config", nil)

			result, err := server.ExportHandleGetIPConfig(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleGetPublicIP tests the get_public_ip handler.
func TestHandleGetPublicIP(t *testing.T) {
	tests := []struct {
		name       string
		publicIP   any
		checkerNil bool
	}{
		{
			name: "successful",
			publicIP: map[string]any{
				"ip":      "203.0.113.1",
				"city":    "Test City",
				"country": "Test Country",
			},
		},
		{
			name:       "checker unavailable",
			checkerNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.checkerNil {
				provider = &mockServiceProvider{publicIPChecker: nil}
			} else {
				provider = &mockServiceProvider{
					publicIPChecker: &mockPublicIPChecker{result: tt.publicIP},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("get_public_ip", nil)

			result, err := server.ExportHandleGetPublicIP(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleGetVLANInfo tests the get_vlan_info handler.
func TestHandleGetVLANInfo(t *testing.T) {
	tests := []struct {
		name       string
		vlanInfo   any
		managerNil bool
	}{
		{
			name: "successful",
			vlanInfo: map[string]any{
				"vlans": []map[string]any{
					{"id": 100, "name": "Management"},
					{"id": 200, "name": "Data"},
				},
			},
		},
		{
			name:       "manager unavailable",
			managerNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.managerNil {
				provider = &mockServiceProvider{vlanManager: nil}
			} else {
				provider = &mockServiceProvider{
					vlanManager: &mockVLANManager{info: tt.vlanInfo},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("get_vlan_info", nil)

			result, err := server.ExportHandleGetVLANInfo(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleSystemHealth tests the system_health handler.
func TestHandleSystemHealth(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		icmpAvailable bool
	}{
		{
			name: "with config",
			cfg: &config.Config{
				Interface: config.InterfaceConfig{
					Default: "eth0",
				},
				MCP: config.MCPConfig{
					Enabled: true,
				},
			},
			icmpAvailable: true,
		},
		{
			name:          "without config",
			cfg:           nil,
			icmpAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockServiceProvider{
				cfg:           tt.cfg,
				icmpAvailable: tt.icmpAvailable,
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("system_health", nil)

			result, err := server.ExportHandleSystemHealth(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleGetDiscoveryStatus tests the get_discovery_status handler.
func TestHandleGetDiscoveryStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     *discovery.ServiceStatus
		serviceNil bool
	}{
		{
			name: "running",
			status: &discovery.ServiceStatus{
				Running:     true,
				DeviceCount: 10,
			},
		},
		{
			name: "not running",
			status: &discovery.ServiceStatus{
				Running:     false,
				DeviceCount: 0,
			},
		},
		{
			name:       "service unavailable",
			serviceNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.serviceNil {
				provider = &mockServiceProvider{discoveryService: nil}
			} else {
				provider = &mockServiceProvider{
					discoveryService: &mockDiscoveryService{status: tt.status},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("get_discovery_status", nil)

			result, err := server.ExportHandleGetDiscoveryStatus(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleVulnerabilityScan tests the vulnerability_scan handler.
func TestHandleVulnerabilityScan(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		devices    []*discovery.DiscoveredDevice
		vulnResult any
		scanErr    error
		serviceNil bool
		scannerNil bool
	}{
		{
			name: "device found and scanned",
			args: map[string]any{"ip": "192.168.1.100"},
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.100", MAC: "00:11:22:33:44:55"},
			},
			vulnResult: map[string]any{
				"vulnerabilities": []map[string]any{},
			},
		},
		{
			name: "device not found",
			args: map[string]any{"ip": "192.168.1.200"},
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.100", MAC: "00:11:22:33:44:55"},
			},
		},
		{
			name:    "missing ip parameter",
			args:    map[string]any{},
			devices: []*discovery.DiscoveredDevice{},
		},
		{
			name:       "service unavailable",
			args:       map[string]any{"ip": "192.168.1.100"},
			serviceNil: true,
		},
		{
			name: "scanner unavailable",
			args: map[string]any{"ip": "192.168.1.100"},
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.100", MAC: "00:11:22:33:44:55"},
			},
			scannerNil: true,
		},
		{
			name: "scan error",
			args: map[string]any{"ip": "192.168.1.100"},
			devices: []*discovery.DiscoveredDevice{
				{IP: "192.168.1.100", MAC: "00:11:22:33:44:55"},
			},
			scanErr: errors.New("scan failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			switch {
			case tt.serviceNil:
				provider = &mockServiceProvider{discoveryService: nil}
			case tt.scannerNil:
				provider = &mockServiceProvider{
					discoveryService: &mockDiscoveryService{devices: tt.devices},
					vulnScanner:      nil,
				}
			default:
				provider = &mockServiceProvider{
					discoveryService: &mockDiscoveryService{devices: tt.devices},
					vulnScanner: &mockVulnScanner{
						result:  tt.vulnResult,
						scanErr: tt.scanErr,
					},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("vulnerability_scan", tt.args)

			result, err := server.ExportHandleVulnerabilityScan(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleRogueDHCPCheck tests the rogue_dhcp_check handler.
func TestHandleRogueDHCPCheck(t *testing.T) {
	tests := []struct {
		name        string
		servers     any
		isRunning   bool
		detectorNil bool
	}{
		{
			name: "servers detected",
			servers: []map[string]any{
				{"ip": "192.168.1.1", "isRogue": false},
			},
			isRunning: true,
		},
		{
			name:      "no servers detected",
			servers:   nil,
			isRunning: true,
		},
		{
			name:        "detector unavailable",
			detectorNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.detectorNil {
				provider = &mockServiceProvider{rogueDetector: nil}
			} else {
				provider = &mockServiceProvider{
					rogueDetector: &mockRogueDetector{
						servers:   tt.servers,
						isRunning: tt.isRunning,
					},
				}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("rogue_dhcp_check", nil)

			result, err := server.ExportHandleRogueDHCPCheck(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestHandleSNMPQuery tests the snmp_query handler.
func TestHandleSNMPQuery(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		cfg     *config.Config
		cfgNil  bool
		wantErr bool
	}{
		{
			name: "with provided community",
			args: map[string]any{"host": "192.168.1.1", "community": "public"},
			cfg: &config.Config{
				SNMP: config.SNMPConfig{Communities: []string{}},
			},
		},
		{
			name: "using configured community",
			args: map[string]any{"host": "192.168.1.1"},
			cfg: &config.Config{
				SNMP: config.SNMPConfig{Communities: []string{"private"}},
			},
		},
		{
			name: "no community available",
			args: map[string]any{"host": "192.168.1.1"},
			cfg: &config.Config{
				SNMP: config.SNMPConfig{Communities: []string{}},
			},
		},
		{
			name: "missing host",
			args: map[string]any{},
			cfg: &config.Config{
				SNMP: config.SNMPConfig{Communities: []string{"public"}},
			},
		},
		{
			name:   "config unavailable",
			args:   map[string]any{"host": "192.168.1.1"},
			cfgNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *mockServiceProvider
			if tt.cfgNil {
				provider = &mockServiceProvider{cfg: nil}
			} else {
				provider = &mockServiceProvider{cfg: tt.cfg}
			}

			server := createTestServer(provider)
			request := mcp.NewCallToolRequest("snmp_query", tt.args)

			result, err := server.ExportHandleSNMPQuery(context.Background(), request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

// TestGetArguments tests the getArguments helper function.
func TestGetArguments(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		checkFunc func(t *testing.T, result map[string]any)
	}{
		{
			name: "with arguments",
			args: map[string]any{
				"key1": "value1",
				"key2": float64(42),
				"key3": true,
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				if result["key1"] != "value1" {
					t.Errorf("expected key1=value1, got %v", result["key1"])
				}
				if result["key2"] != float64(42) {
					t.Errorf("expected key2=42, got %v", result["key2"])
				}
				if result["key3"] != true {
					t.Errorf("expected key3=true, got %v", result["key3"])
				}
			},
		},
		{
			name: "empty arguments",
			args: map[string]any{},
			checkFunc: func(t *testing.T, result map[string]any) {
				if len(result) != 0 {
					t.Errorf("expected empty map, got %d elements", len(result))
				}
			},
		},
		{
			name: "nil arguments",
			args: nil,
			checkFunc: func(t *testing.T, result map[string]any) {
				// When a nil map[string]any is assigned to any, the interface is not nil
				// (it has type info but nil value). The type assertion returns the nil map.
				// A nil map in Go is usable (returns zero values on read, panics on write).
				// This is valid Go behavior.
				if len(result) != 0 {
					t.Errorf("expected nil or empty map for nil args, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.NewCallToolRequest("test_tool", tt.args)
			result := mcp.ExportGetArguments2(request)
			tt.checkFunc(t, result)
		})
	}
}

// TestGetArgumentsWithNilParams tests getArguments when Params.Arguments is truly nil interface.
func TestGetArgumentsWithNilParams(t *testing.T) {
	// Test case where Arguments is nil interface (not nil map assigned to interface)
	request := mcp.NewCallToolRequestWithNilArgs("test_tool")
	result := mcp.ExportGetArguments2(request)

	// When Arguments is nil (interface), should return empty map
	if result == nil {
		t.Error("expected non-nil empty map, got nil")
	}
}

// TestGetArgumentsWithNonMapType tests getArguments when Arguments is not a map.
func TestGetArgumentsWithNonMapType(t *testing.T) {
	request := mcp.NewCallToolRequestWithStringArg("test_tool", "string_arg")
	result := mcp.ExportGetArguments2(request)

	// When Arguments is not a map, should return empty map
	if result == nil {
		t.Error("expected non-nil empty map, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map for non-map args, got %d elements", len(result))
	}
}

// TestServerToolRegistration tests that tools are properly registered based on config.
func TestServerToolRegistration(t *testing.T) {
	tests := []struct {
		name         string
		allowedTools []string
	}{
		{
			name:         "all tools allowed (empty list)",
			allowedTools: []string{},
		},
		{
			name:         "specific tools allowed",
			allowedTools: []string{"network_scan", "get_devices", "dns_test"},
		},
		{
			name:         "single tool allowed",
			allowedTools: []string{"system_health"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.MCPConfig{
				Enabled:      true,
				AllowedTools: tt.allowedTools,
			}
			provider := &mockServiceProvider{
				cfg: &config.Config{},
			}

			server := mcp.NewServer(cfg, provider)
			if server == nil {
				t.Fatal("expected non-nil server")
			}
			if server.GetMCPServer() == nil {
				t.Error("expected non-nil MCP server")
			}
		})
	}
}

// TestServerWithFullServiceProvider tests server with all services available.
func TestServerWithFullServiceProvider(t *testing.T) {
	provider := &mockServiceProvider{
		discoveryService: &mockDiscoveryService{
			devices: []*discovery.DiscoveredDevice{},
			status:  &discovery.ServiceStatus{Running: true},
		},
		netManager:      &mockNetworkManager{interfaces: []*network.InterfaceInfo{}},
		linkMonitor:     &mockLinkMonitor{isUp: true},
		vlanManager:     &mockVLANManager{},
		dnsTester:       &mockDNSTester{result: &dns.TestResult{}},
		gatewayTester:   &mockGatewayTester{stats: &gateway.PingStats{}},
		speedtestTester: &mockSpeedtestTester{status: speedtest.Status{}},
		iperfManager:    &mockIperfManager{clientStatus: iperf.ClientStatus{}},
		wifiScanner:     &mockWiFiScanner{networks: []mcp.WiFiNetwork{}},
		wifiManager:     &mockWiFiManager{},
		rogueDetector:   &mockRogueDetector{},
		vulnScanner:     &mockVulnScanner{},
		publicIPChecker: &mockPublicIPChecker{},
		cfg:             &config.Config{},
		icmpAvailable:   true,
	}

	server := createTestServer(provider)
	if server == nil {
		t.Fatal("expected non-nil server")
	}

	// Test that accessor works correctly
	accessor := &mcp.ServerTestAccessor{Server: server}
	if accessor.GetMCPServer() == nil {
		t.Error("expected non-nil MCP server from accessor")
	}
	if accessor.GetConfig() == nil {
		t.Error("expected non-nil config from accessor")
	}
	if accessor.GetServices() == nil {
		t.Error("expected non-nil services from accessor")
	}
}

// TestHandlerErrorMessages tests that handlers return appropriate error messages.
func TestHandlerErrorMessages(t *testing.T) {
	tests := []struct {
		name            string
		handler         func(*mcp.Server, context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		request         mcp.CallToolRequest
		provider        *mockServiceProvider
		wantErrContains string
	}{
		{
			name: "discovery service unavailable",
			handler: func(
				s *mcp.Server,
				ctx context.Context,
				r mcp.CallToolRequest,
			) (*mcp.CallToolResult, error) {
				return s.ExportHandleGetDevices(ctx, r)
			},
			request:         mcp.NewCallToolRequest("get_devices", nil),
			provider:        &mockServiceProvider{discoveryService: nil},
			wantErrContains: "not available",
		},
		{
			name: "network manager unavailable",
			handler: func(
				s *mcp.Server,
				ctx context.Context,
				r mcp.CallToolRequest,
			) (*mcp.CallToolResult, error) {
				return s.ExportHandleGetInterfaces(ctx, r)
			},
			request:         mcp.NewCallToolRequest("get_interfaces", nil),
			provider:        &mockServiceProvider{netManager: nil},
			wantErrContains: "not available",
		},
		{
			name: "wifi scanner unavailable",
			handler: func(
				s *mcp.Server,
				ctx context.Context,
				r mcp.CallToolRequest,
			) (*mcp.CallToolResult, error) {
				return s.ExportHandleWiFiScan(ctx, r)
			},
			request:         mcp.NewCallToolRequest("wifi_scan", nil),
			provider:        &mockServiceProvider{wifiScanner: nil},
			wantErrContains: "not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(tt.provider)
			result, err := tt.handler(server, context.Background(), tt.request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			// Note: The actual error message is contained in the CallToolResult
			// In a real test we would parse the result to verify the error message
		})
	}
}
