package mcp_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/mcp"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/sap/dns"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// mockServiceProvider implements mcp.ServiceProvider for testing.
type mockServiceProvider struct {
	discoveryService mcp.DiscoveryService
	deviceDiscovery  mcp.DeviceDiscovery
	netManager       mcp.NetworkManager
	linkMonitor      mcp.LinkMonitor
	vlanManager      mcp.VLANManager
	dnsTester        mcp.DNSTester
	gatewayTester    mcp.GatewayTester
	speedtestTester  mcp.SpeedtestTester
	iperfManager     mcp.IperfManager
	wifiScanner      mcp.WiFiScanner
	wifiManager      mcp.WiFiManager
	rogueDetector    mcp.RogueDetector
	vulnScanner      mcp.VulnScanner
	publicIPChecker  mcp.PublicIPChecker
	cfg              *config.Config
	icmpAvailable    bool
}

var errInterfaceNotFound = errors.New("interface not found")

func (m *mockServiceProvider) GetDiscoveryService() mcp.DiscoveryService {
	return m.discoveryService
}

func (m *mockServiceProvider) GetDeviceDiscovery() mcp.DeviceDiscovery {
	return m.deviceDiscovery
}

func (m *mockServiceProvider) GetNetManager() mcp.NetworkManager {
	return m.netManager
}

func (m *mockServiceProvider) GetLinkMonitor() mcp.LinkMonitor {
	return m.linkMonitor
}

func (m *mockServiceProvider) GetVLANManager() mcp.VLANManager {
	return m.vlanManager
}

func (m *mockServiceProvider) GetDNSTester() mcp.DNSTester {
	return m.dnsTester
}

func (m *mockServiceProvider) GetGatewayTester() mcp.GatewayTester {
	return m.gatewayTester
}

func (m *mockServiceProvider) GetSpeedtestTester() mcp.SpeedtestTester {
	return m.speedtestTester
}

func (m *mockServiceProvider) GetIperfManager() mcp.IperfManager {
	return m.iperfManager
}

func (m *mockServiceProvider) GetWiFiScanner() mcp.WiFiScanner {
	return m.wifiScanner
}

func (m *mockServiceProvider) GetWiFiManager() mcp.WiFiManager {
	return m.wifiManager
}

func (m *mockServiceProvider) GetRogueDetector() mcp.RogueDetector {
	return m.rogueDetector
}

func (m *mockServiceProvider) GetVulnScanner() mcp.VulnScanner {
	return m.vulnScanner
}

func (m *mockServiceProvider) GetPublicIPChecker() mcp.PublicIPChecker {
	return m.publicIPChecker
}

func (m *mockServiceProvider) GetConfig() *config.Config {
	return m.cfg
}

func (m *mockServiceProvider) IsICMPAvailable() bool {
	return m.icmpAvailable
}

// mockDiscoveryService implements mcp.DiscoveryService for testing.
type mockDiscoveryService struct {
	devices []*discovery.DiscoveredDevice
	status  *discovery.ServiceStatus
	options config.DiscoveryOptions
	scanErr error
}

func (m *mockDiscoveryService) Scan(_ context.Context) error {
	return m.scanErr
}

func (m *mockDiscoveryService) GetDevices() []*discovery.DiscoveredDevice {
	return m.devices
}

func (m *mockDiscoveryService) GetStatus() *discovery.ServiceStatus {
	return m.status
}

func (m *mockDiscoveryService) GetOptions() config.DiscoveryOptions {
	return m.options
}

// mockNetworkManager implements mcp.NetworkManager for testing.
type mockNetworkManager struct {
	interfaces       []*network.InterfaceInfo
	currentInterface string
	getInterfaceErr  error
}

func (m *mockNetworkManager) GetInterfaces() []*network.InterfaceInfo {
	return m.interfaces
}

func (m *mockNetworkManager) GetInterface(name string) (*network.InterfaceInfo, error) {
	if m.getInterfaceErr != nil {
		return nil, m.getInterfaceErr
	}
	for _, iface := range m.interfaces {
		if iface.Name == name {
			return iface, nil
		}
	}
	return nil, errInterfaceNotFound
}

func (m *mockNetworkManager) GetCurrentInterface() string {
	return m.currentInterface
}

// mockLinkMonitor implements mcp.LinkMonitor for testing.
type mockLinkMonitor struct {
	state network.LinkState
	isUp  bool
}

func (m *mockLinkMonitor) GetState() network.LinkState {
	return m.state
}

func (m *mockLinkMonitor) IsUp() bool {
	return m.isUp
}

// mockVLANManager implements mcp.VLANManager for testing.
type mockVLANManager struct {
	info any
}

func (m *mockVLANManager) GetInfo() any {
	return m.info
}

// mockDNSTester implements mcp.DNSTester for testing.
type mockDNSTester struct {
	result *dns.TestResult
}

func (m *mockDNSTester) Test(_ context.Context) *dns.TestResult {
	return m.result
}

// mockGatewayTester implements mcp.GatewayTester for testing.
type mockGatewayTester struct {
	stats *gateway.PingStats
}

func (m *mockGatewayTester) Test() *gateway.PingStats {
	return m.stats
}

func (m *mockGatewayTester) Ping() *gateway.PingStats {
	return m.stats
}

// mockSpeedtestTester implements mcp.SpeedtestTester for testing.
type mockSpeedtestTester struct {
	result *speedtest.Result
	status speedtest.Status
	runErr error
}

func (m *mockSpeedtestTester) Run(_ context.Context) (*speedtest.Result, error) {
	return m.result, m.runErr
}

func (m *mockSpeedtestTester) GetStatus() speedtest.Status {
	return m.status
}

// mockIperfManager implements mcp.IperfManager for testing.
type mockIperfManager struct {
	result       *iperf.Result
	clientStatus iperf.ClientStatus
	runErr       error
}

func (m *mockIperfManager) RunClient(
	_ context.Context,
	_ *iperf.ClientConfig,
) (*iperf.Result, error) {
	return m.result, m.runErr
}

func (m *mockIperfManager) GetClientStatus() iperf.ClientStatus {
	return m.clientStatus
}

// mockWiFiScanner implements mcp.WiFiScanner for testing.
type mockWiFiScanner struct {
	networks []mcp.WiFiNetwork
	scanErr  error
}

func (m *mockWiFiScanner) Scan(_ context.Context) ([]mcp.WiFiNetwork, error) {
	return m.networks, m.scanErr
}

// mockWiFiManager implements mcp.WiFiManager for testing.
type mockWiFiManager struct {
	currentNetwork *mcp.WiFiConnectionInfo
	signalStrength int
	getNetworkErr  error
	getSignalErr   error
}

func (m *mockWiFiManager) GetCurrentNetwork() (*mcp.WiFiConnectionInfo, error) {
	return m.currentNetwork, m.getNetworkErr
}

func (m *mockWiFiManager) GetSignalStrength() (int, error) {
	return m.signalStrength, m.getSignalErr
}

// mockRogueDetector implements mcp.RogueDetector for testing.
type mockRogueDetector struct {
	servers   any
	isRunning bool
}

func (m *mockRogueDetector) GetDetectedServers() any {
	return m.servers
}

func (m *mockRogueDetector) IsRunning() bool {
	return m.isRunning
}

// mockVulnScanner implements mcp.VulnScanner for testing.
type mockVulnScanner struct {
	result          any
	vulnerabilities any
	scanErr         error
}

func (m *mockVulnScanner) ScanDevice(
	_ context.Context,
	_ *discovery.DiscoveredDevice,
) (any, error) {
	return m.result, m.scanErr
}

func (m *mockVulnScanner) GetAllVulnerabilities() any {
	return m.vulnerabilities
}

// mockPublicIPChecker implements mcp.PublicIPChecker for testing.
type mockPublicIPChecker struct {
	result any
}

func (m *mockPublicIPChecker) GetPublicIP(_ context.Context) any {
	return m.result
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.MCPConfig
		services mcp.ServiceProvider
	}{
		{
			name: "basic server creation",
			cfg: &config.MCPConfig{
				Enabled:      true,
				AllowedTools: []string{},
			},
			services: &mockServiceProvider{},
		},
		{
			name: "server with allowed tools",
			cfg: &config.MCPConfig{
				Enabled:      true,
				AllowedTools: []string{"network_scan", "get_devices"},
			},
			services: &mockServiceProvider{},
		},
		{
			name: "server with all tools allowed",
			cfg: &config.MCPConfig{
				Enabled:      true,
				AllowedTools: []string{},
			},
			services: &mockServiceProvider{
				cfg: &config.Config{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mcp.NewServer(tt.cfg, tt.services)
			if server == nil {
				t.Fatal("expected non-nil server")
			}

			mcpServer := server.GetMCPServer()
			if mcpServer == nil {
				t.Error("expected non-nil MCP server")
			}
		})
	}
}

func TestServerAccessor(t *testing.T) {
	cfg := &config.MCPConfig{
		Enabled:      true,
		AllowedTools: []string{"network_scan"},
	}
	services := &mockServiceProvider{
		cfg: &config.Config{},
	}

	server := mcp.NewServer(cfg, services)
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

func TestMockServiceProvider(t *testing.T) {
	devices := []*discovery.DiscoveredDevice{
		{IP: "192.168.1.1", MAC: "00:11:22:33:44:55"},
		{IP: "192.168.1.2", MAC: "00:11:22:33:44:56"},
	}

	mockDiscovery := &mockDiscoveryService{
		devices: devices,
		status: &discovery.ServiceStatus{
			Running: true,
		},
	}

	mockNet := &mockNetworkManager{
		interfaces: []*network.InterfaceInfo{
			{Name: "eth0"},
			{Name: "wlan0"},
		},
		currentInterface: "eth0",
	}

	mockLink := &mockLinkMonitor{
		isUp: true,
	}

	provider := &mockServiceProvider{
		discoveryService: mockDiscovery,
		netManager:       mockNet,
		linkMonitor:      mockLink,
		icmpAvailable:    true,
		cfg: &config.Config{
			MCP: config.MCPConfig{
				Enabled: true,
			},
		},
	}

	// Test discovery service
	discoverySvc := provider.GetDiscoveryService()
	if discoverySvc == nil {
		t.Fatal("expected non-nil discovery service")
	}
	if len(discoverySvc.GetDevices()) != 2 {
		t.Errorf("expected 2 devices, got %d", len(discoverySvc.GetDevices()))
	}

	// Test network manager
	netMgr := provider.GetNetManager()
	if netMgr == nil {
		t.Fatal("expected non-nil network manager")
	}
	if len(netMgr.GetInterfaces()) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(netMgr.GetInterfaces()))
	}
	if netMgr.GetCurrentInterface() != "eth0" {
		t.Errorf("expected current interface eth0, got %s", netMgr.GetCurrentInterface())
	}

	// Test link monitor
	linkMon := provider.GetLinkMonitor()
	if linkMon == nil {
		t.Fatal("expected non-nil link monitor")
	}
	if !linkMon.IsUp() {
		t.Error("expected link to be up")
	}

	// Test ICMP availability
	if !provider.IsICMPAvailable() {
		t.Error("expected ICMP to be available")
	}

	// Test config
	cfg := provider.GetConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if !cfg.MCP.Enabled {
		t.Error("expected MCP to be enabled")
	}
}

func TestMockDiscoveryServiceScan(t *testing.T) {
	tests := []struct {
		name    string
		scanErr error
		wantErr bool
	}{
		{
			name:    "successful scan",
			scanErr: nil,
			wantErr: false,
		},
		{
			name:    "scan in progress",
			scanErr: discovery.ErrScanInProgress,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDiscoveryService{
				scanErr: tt.scanErr,
			}

			err := mock.Scan(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMockSpeedtestTester(t *testing.T) {
	tests := []struct {
		name    string
		result  *speedtest.Result
		status  speedtest.Status
		runErr  error
		wantErr bool
	}{
		{
			name: "successful speedtest",
			result: &speedtest.Result{
				Download: 100.5,
				Upload:   50.2,
				Latency:  15.0,
			},
			status: speedtest.Status{
				Running: false,
			},
			wantErr: false,
		},
		{
			name:    "speedtest already running",
			result:  nil,
			status:  speedtest.Status{Running: true},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSpeedtestTester{
				result: tt.result,
				status: tt.status,
				runErr: tt.runErr,
			}

			status := mock.GetStatus()
			if status.Running != tt.status.Running {
				t.Errorf("expected running=%v, got running=%v", tt.status.Running, status.Running)
			}

			if status.Running {
				return
			}

			result, err := mock.Run(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.result {
				t.Errorf("expected result %v, got %v", tt.result, result)
			}
		})
	}
}

func TestMockIperfManager(t *testing.T) {
	tests := []struct {
		name         string
		result       *iperf.Result
		clientStatus iperf.ClientStatus
		runErr       error
		wantErr      bool
	}{
		{
			name: "successful iperf test",
			result: &iperf.Result{
				Bandwidth: 100.5,
				Protocol:  "tcp",
			},
			clientStatus: iperf.ClientStatus{
				Running: false,
				Phase:   "idle",
			},
			wantErr: false,
		},
		{
			name:   "iperf test already running",
			result: nil,
			clientStatus: iperf.ClientStatus{
				Running: true,
				Phase:   "testing",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockIperfManager{
				result:       tt.result,
				clientStatus: tt.clientStatus,
				runErr:       tt.runErr,
			}

			status := mock.GetClientStatus()
			if status.Running != tt.clientStatus.Running {
				t.Errorf(
					"expected running=%v, got running=%v",
					tt.clientStatus.Running,
					status.Running,
				)
			}

			if status.Running {
				return
			}

			result, err := mock.RunClient(context.Background(), &iperf.ClientConfig{
				Server: "localhost",
			})
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.result {
				t.Errorf("expected result %v, got %v", tt.result, result)
			}
		})
	}
}

func TestMockWiFiScanner(t *testing.T) {
	tests := []struct {
		name     string
		networks []mcp.WiFiNetwork
		scanErr  error
		wantErr  bool
	}{
		{
			name: "successful scan with networks",
			networks: []mcp.WiFiNetwork{
				{
					SSID:      "TestNetwork1",
					BSSID:     "00:11:22:33:44:55",
					Signal:    -50,
					Channel:   6,
					Frequency: 2437,
					Security:  "WPA2-PSK",
				},
				{
					SSID:      "TestNetwork2",
					BSSID:     "00:11:22:33:44:56",
					Signal:    -70,
					Channel:   11,
					Frequency: 2462,
					Security:  "WPA3-SAE",
				},
			},
			wantErr: false,
		},
		{
			name:     "successful scan with no networks",
			networks: []mcp.WiFiNetwork{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWiFiScanner{
				networks: tt.networks,
				scanErr:  tt.scanErr,
			}

			networks, err := mock.Scan(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(networks) != len(tt.networks) {
					t.Errorf("expected %d networks, got %d", len(tt.networks), len(networks))
				}
			}
		})
	}
}

func TestMockRogueDetector(t *testing.T) {
	tests := []struct {
		name      string
		servers   any
		isRunning bool
	}{
		{
			name:      "detector running with servers",
			servers:   []map[string]any{{"ip": "192.168.1.1", "mac": "00:11:22:33:44:55"}},
			isRunning: true,
		},
		{
			name:      "detector not running",
			servers:   nil,
			isRunning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRogueDetector{
				servers:   tt.servers,
				isRunning: tt.isRunning,
			}

			if mock.IsRunning() != tt.isRunning {
				t.Errorf("expected isRunning=%v, got %v", tt.isRunning, mock.IsRunning())
			}

			servers := mock.GetDetectedServers()
			if tt.servers == nil && servers != nil {
				t.Error("expected nil servers")
			}
			if tt.servers != nil && servers == nil {
				t.Error("expected non-nil servers")
			}
		})
	}
}

func TestWiFiNetworkStruct(t *testing.T) {
	network := mcp.WiFiNetwork{
		SSID:      "TestSSID",
		BSSID:     "00:11:22:33:44:55",
		Signal:    -65,
		Channel:   6,
		Frequency: 2437,
		Security:  "WPA2-PSK",
	}

	if network.SSID != "TestSSID" {
		t.Errorf("expected SSID TestSSID, got %s", network.SSID)
	}
	if network.BSSID != "00:11:22:33:44:55" {
		t.Errorf("expected BSSID 00:11:22:33:44:55, got %s", network.BSSID)
	}
	if network.Signal > 0 {
		t.Error("signal strength should be negative (dBm)")
	}
	if network.Channel <= 0 {
		t.Error("channel should be positive")
	}
	if network.Frequency <= 0 {
		t.Error("frequency should be positive")
	}
	if network.Security != "WPA2-PSK" {
		t.Errorf("expected Security WPA2-PSK, got %s", network.Security)
	}
}

func TestWiFiConnectionInfoStruct(t *testing.T) {
	info := mcp.WiFiConnectionInfo{
		SSID:      "ConnectedNetwork",
		BSSID:     "00:11:22:33:44:55",
		Signal:    -55,
		Channel:   11,
		Frequency: 2462,
		Security:  "WPA3-SAE",
	}

	if info.SSID != "ConnectedNetwork" {
		t.Errorf("expected SSID ConnectedNetwork, got %s", info.SSID)
	}
	if info.BSSID != "00:11:22:33:44:55" {
		t.Errorf("expected BSSID 00:11:22:33:44:55, got %s", info.BSSID)
	}
	if info.Signal > 0 {
		t.Error("signal strength should be negative (dBm)")
	}
	if info.Channel != 11 {
		t.Errorf("expected Channel 11, got %d", info.Channel)
	}
	if info.Frequency != 2462 {
		t.Errorf("expected Frequency 2462, got %d", info.Frequency)
	}
	if info.Security != "WPA3-SAE" {
		t.Errorf("expected Security WPA3-SAE, got %s", info.Security)
	}
}

func TestTCPProbeResultStruct(t *testing.T) {
	result := mcp.TCPProbeResult{
		Host:    "example.com",
		Port:    443,
		Open:    true,
		Latency: 25 * time.Millisecond,
		Error:   "",
	}

	if result.Host != "example.com" {
		t.Errorf("expected host example.com, got %s", result.Host)
	}
	if result.Port != 443 {
		t.Errorf("expected port 443, got %d", result.Port)
	}
	if !result.Open {
		t.Error("expected port to be open")
	}
	if result.Latency <= 0 {
		t.Error("expected positive latency")
	}
	if result.Error != "" {
		t.Errorf("expected no error, got %s", result.Error)
	}
}
