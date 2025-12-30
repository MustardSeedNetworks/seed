// Package api provides the HTTP/WebSocket server.
// This file implements the mcp.ServiceProvider interface to expose
// server services to the MCP server.
package api

import (
	"context"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/dns"
	"github.com/krisarmstrong/seed/internal/gateway"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/mcp"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/publicip"
	"github.com/krisarmstrong/seed/internal/speedtest"
	"github.com/krisarmstrong/seed/internal/vlan"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

// Ensure Server implements mcp.ServiceProvider.
var _ mcp.ServiceProvider = (*Server)(nil)

// discoveryServiceAdapter adapts the Server's discovery service to the mcp.DiscoveryService interface.
type discoveryServiceAdapter struct {
	svc *discovery.Service
}

func (a *discoveryServiceAdapter) Scan(ctx context.Context) error {
	return a.svc.Scan(ctx)
}

func (a *discoveryServiceAdapter) GetDevices() []*discovery.DiscoveredDevice {
	return a.svc.GetDevices()
}

func (a *discoveryServiceAdapter) GetStatus() *discovery.ServiceStatus {
	return a.svc.GetStatus()
}

func (a *discoveryServiceAdapter) GetOptions() config.DiscoveryOptions {
	return a.svc.GetOptions()
}

// deviceDiscoveryAdapter adapts DeviceDiscovery to the mcp.DeviceDiscovery interface.
type deviceDiscoveryAdapter struct {
	dd *discovery.DeviceDiscovery
}

func (a *deviceDiscoveryAdapter) GetDiscoveredDevices() []*discovery.DiscoveredDevice {
	return a.dd.GetDevices()
}

func (a *deviceDiscoveryAdapter) Scan(ctx context.Context) error {
	return a.dd.Scan(ctx)
}

func (a *deviceDiscoveryAdapter) SetAdditionalSubnets(cidrs []string) error {
	return a.dd.SetAdditionalSubnets(cidrs)
}

// networkManagerAdapter adapts network.Manager to the mcp.NetworkManager interface.
type networkManagerAdapter struct {
	mgr *network.Manager
}

func (a *networkManagerAdapter) GetInterfaces() []*network.InterfaceInfo {
	return a.mgr.GetInterfaces()
}

func (a *networkManagerAdapter) GetInterface(name string) (*network.InterfaceInfo, error) {
	return a.mgr.GetInterface(name)
}

func (a *networkManagerAdapter) GetCurrentInterface() string {
	return a.mgr.GetCurrentInterface()
}

// linkMonitorAdapter adapts network.LinkMonitor to the mcp.LinkMonitor interface.
type linkMonitorAdapter struct {
	mon *network.LinkMonitor
}

func (a *linkMonitorAdapter) GetState() network.LinkState {
	return a.mon.GetState()
}

func (a *linkMonitorAdapter) IsUp() bool {
	return a.mon.IsUp()
}

// dnsTesterAdapter adapts dns.Tester to the mcp.DNSTester interface.
type dnsTesterAdapter struct {
	tester *dns.Tester
}

func (a *dnsTesterAdapter) Test(ctx context.Context) *dns.TestResult {
	return a.tester.Test(ctx)
}

// gatewayTesterAdapter adapts gateway.Tester to the mcp.GatewayTester interface.
type gatewayTesterAdapter struct {
	tester *gateway.Tester
}

func (a *gatewayTesterAdapter) Test() *gateway.PingStats {
	return a.tester.Test()
}

func (a *gatewayTesterAdapter) Ping() *gateway.PingStats {
	// Note: gateway.Tester.Ping() returns *PingResult but interface expects *PingStats
	// They should be the same type, but if not, we return Test() result
	return a.tester.Test()
}

// speedtestTesterAdapter adapts speedtest.Tester to the mcp.SpeedtestTester interface.
type speedtestTesterAdapter struct {
	tester *speedtest.Tester
}

func (a *speedtestTesterAdapter) Run(ctx context.Context) (*speedtest.Result, error) {
	return a.tester.RunTest(ctx)
}

func (a *speedtestTesterAdapter) GetStatus() speedtest.Status {
	return a.tester.GetStatus()
}

// iperfManagerAdapter adapts iperf.Manager to the mcp.IperfManager interface.
type iperfManagerAdapter struct {
	mgr *iperf.Manager
}

func (a *iperfManagerAdapter) RunClient(ctx context.Context, cfg *iperf.ClientConfig) (*iperf.Result, error) {
	return a.mgr.RunClient(ctx, cfg)
}

func (a *iperfManagerAdapter) GetClientStatus() iperf.ClientStatus {
	return a.mgr.GetClientStatus()
}

// wifiScannerAdapter adapts wifi.Scanner to the mcp.WiFiScanner interface.
type wifiScannerAdapter struct {
	scanner *wifi.Scanner
}

func (a *wifiScannerAdapter) Scan(_ context.Context) ([]mcp.WiFiNetwork, error) {
	networks, err := a.scanner.Scan()
	if err != nil {
		return nil, err
	}
	result := make([]mcp.WiFiNetwork, len(networks))
	for i, n := range networks {
		result[i] = mcp.WiFiNetwork{
			SSID:      n.SSID,
			BSSID:     n.BSSID,
			Signal:    n.Signal,
			Channel:   n.Channel,
			Frequency: n.Frequency,
			Security:  n.Security,
		}
	}
	return result, nil
}

// wifiManagerAdapter adapts wifi.Manager to the mcp.WiFiManager interface.
type wifiManagerAdapter struct {
	mgr *wifi.Manager
}

func (a *wifiManagerAdapter) GetCurrentNetwork() (*mcp.WiFiConnectionInfo, error) {
	info := a.mgr.GetInfo()
	if info == nil || info.SSID == "" {
		return nil, nil //nolint:nilnil // nil result is valid when not connected to WiFi
	}
	return &mcp.WiFiConnectionInfo{
		SSID:      info.SSID,
		BSSID:     info.BSSID,
		Signal:    info.Signal,
		Channel:   info.Channel,
		Frequency: info.Frequency,
		Security:  info.Security,
	}, nil
}

func (a *wifiManagerAdapter) GetSignalStrength() (int, error) {
	info := a.mgr.GetInfo()
	if info == nil {
		return 0, nil
	}
	return info.Signal, nil
}

// vulnScannerAdapter adapts discovery.VulnerabilityScanner to the mcp.VulnScanner interface.
type vulnScannerAdapter struct {
	scanner *discovery.VulnerabilityScanner
}

func (a *vulnScannerAdapter) ScanDevice(ctx context.Context, device *discovery.DiscoveredDevice) (interface{}, error) {
	return a.scanner.ScanDevice(ctx, device)
}

func (a *vulnScannerAdapter) GetAllVulnerabilities() interface{} {
	return a.scanner.GetAllVulnerabilities()
}

// GetDiscoveryService returns the discovery service adapter.
func (s *Server) GetDiscoveryService() mcp.DiscoveryService {
	if s.discoveryService == nil {
		return nil
	}
	return &discoveryServiceAdapter{svc: s.discoveryService}
}

// GetDeviceDiscovery returns the device discovery adapter.
func (s *Server) GetDeviceDiscovery() mcp.DeviceDiscovery {
	if s.deviceDiscovery == nil {
		return nil
	}
	return &deviceDiscoveryAdapter{dd: s.deviceDiscovery}
}

// GetNetManager returns the network manager adapter.
func (s *Server) GetNetManager() mcp.NetworkManager {
	if s.netManager == nil {
		return nil
	}
	return &networkManagerAdapter{mgr: s.netManager}
}

// GetLinkMonitor returns the link monitor adapter.
func (s *Server) GetLinkMonitor() mcp.LinkMonitor {
	if s.linkMonitor == nil {
		return nil
	}
	return &linkMonitorAdapter{mon: s.linkMonitor}
}

// GetVLANManager returns the VLAN manager adapter.
func (s *Server) GetVLANManager() mcp.VLANManager {
	if s.vlanManager == nil {
		return nil
	}
	return &vlanManagerWrapperImpl{mgr: s.vlanManager}
}

// vlanManagerWrapperImpl wraps vlan.Manager to implement mcp.VLANManager.
type vlanManagerWrapperImpl struct {
	mgr *vlan.Manager
}

func (w *vlanManagerWrapperImpl) GetInfo() interface{} {
	return w.mgr.GetInfo()
}

// GetDNSTester returns the DNS tester adapter.
func (s *Server) GetDNSTester() mcp.DNSTester {
	if s.dnsTester == nil {
		return nil
	}
	return &dnsTesterAdapter{tester: s.dnsTester}
}

// GetGatewayTester returns the gateway tester adapter.
func (s *Server) GetGatewayTester() mcp.GatewayTester {
	if s.gatewayTester == nil {
		return nil
	}
	return &gatewayTesterAdapter{tester: s.gatewayTester}
}

// GetSpeedtestTester returns the speedtest tester adapter.
func (s *Server) GetSpeedtestTester() mcp.SpeedtestTester {
	if s.speedtestTester == nil {
		return nil
	}
	return &speedtestTesterAdapter{tester: s.speedtestTester}
}

// GetIperfManager returns the iPerf manager adapter.
func (s *Server) GetIperfManager() mcp.IperfManager {
	if s.iperfManager == nil {
		return nil
	}
	return &iperfManagerAdapter{mgr: s.iperfManager}
}

// GetWiFiScanner returns the WiFi scanner adapter.
func (s *Server) GetWiFiScanner() mcp.WiFiScanner {
	if s.wifiScanner == nil {
		return nil
	}
	return &wifiScannerAdapter{scanner: s.wifiScanner}
}

// GetWiFiManager returns the WiFi manager adapter.
func (s *Server) GetWiFiManager() mcp.WiFiManager {
	if s.wifiManager == nil {
		return nil
	}
	return &wifiManagerAdapter{mgr: s.wifiManager}
}

// GetRogueDetector returns the rogue DHCP detector adapter.
func (s *Server) GetRogueDetector() mcp.RogueDetector {
	if s.rogueDetector == nil {
		return nil
	}
	return &rogueDetectorWrapperImpl{detector: s.rogueDetector}
}

// rogueDetectorWrapperImpl wraps dhcp.RogueDetector to implement mcp.RogueDetector.
type rogueDetectorWrapperImpl struct {
	detector *dhcp.RogueDetector
}

func (w *rogueDetectorWrapperImpl) GetDetectedServers() interface{} {
	return w.detector.GetDetectedServers()
}

func (w *rogueDetectorWrapperImpl) IsRunning() bool {
	return w.detector.IsRunning()
}

// GetVulnScanner returns the vulnerability scanner adapter.
func (s *Server) GetVulnScanner() mcp.VulnScanner {
	if s.vulnScanner == nil {
		return nil
	}
	return &vulnScannerAdapter{scanner: s.vulnScanner}
}

// GetPublicIPChecker returns the public IP checker adapter.
func (s *Server) GetPublicIPChecker() mcp.PublicIPChecker {
	if s.publicipChecker == nil {
		return nil
	}
	return &publicIPCheckerWrapperImpl{checker: s.publicipChecker}
}

// publicIPCheckerWrapperImpl wraps publicip.Checker to implement mcp.PublicIPChecker.
type publicIPCheckerWrapperImpl struct {
	checker *publicip.Checker
}

func (w *publicIPCheckerWrapperImpl) GetPublicIP(ctx context.Context) interface{} {
	return w.checker.GetPublicIP(ctx)
}

// GetConfig returns the server configuration.
func (s *Server) GetConfig() *config.Config {
	return s.config
}

// IsICMPAvailable returns whether raw ICMP sockets are available.
func (s *Server) IsICMPAvailable() bool {
	return s.icmpAvailable
}
