// Package api provides the HTTP/WebSocket server.
// This file implements the mcp.ServiceProvider interface to expose
// server services to the MCP server.
package api

import (
	"context"
	"fmt"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/mcp"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/roots/publicip"
	"github.com/krisarmstrong/seed/internal/sap/dns"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
	"github.com/krisarmstrong/seed/internal/sap/speedtest"
	"github.com/krisarmstrong/seed/internal/sap/vlan"
)

// Ensure Server implements mcp.ServiceProvider.
var _ mcp.ServiceProvider = (*Server)(nil)

// discoveryServiceAdapter adapts the Server's discovery service to the mcp.DiscoveryService interface.
type discoveryServiceAdapter struct {
	svc *discovery.Service
}

func (a *discoveryServiceAdapter) Scan(ctx context.Context) error {
	if err := a.svc.Scan(ctx); err != nil {
		return fmt.Errorf("discovery scan: %w", err)
	}
	return nil
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
	if err := a.dd.Scan(ctx); err != nil {
		return fmt.Errorf("device discovery scan: %w", err)
	}
	return nil
}

func (a *deviceDiscoveryAdapter) SetAdditionalSubnets(cidrs []string) error {
	if err := a.dd.SetAdditionalSubnets(cidrs); err != nil {
		return fmt.Errorf("set additional subnets: %w", err)
	}
	return nil
}

// networkManagerAdapter adapts network.Manager to the mcp.NetworkManager interface.
type networkManagerAdapter struct {
	mgr *network.Manager
}

func (a *networkManagerAdapter) GetInterfaces() []*network.InterfaceInfo {
	return a.mgr.GetInterfaces()
}

func (a *networkManagerAdapter) GetInterface(name string) (*network.InterfaceInfo, error) {
	iface, err := a.mgr.GetInterface(name)
	if err != nil {
		return nil, fmt.Errorf("get interface: %w", err)
	}
	return iface, nil
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
	result, err := a.tester.RunTest(ctx)
	if err != nil {
		return nil, fmt.Errorf("run speedtest: %w", err)
	}
	return result, nil
}

func (a *speedtestTesterAdapter) GetStatus() speedtest.Status {
	return a.tester.GetStatus()
}

// iperfManagerAdapter adapts iperf.Manager to the mcp.IperfManager interface.
type iperfManagerAdapter struct {
	mgr *iperf.Manager
}

func (a *iperfManagerAdapter) RunClient(ctx context.Context, cfg *iperf.ClientConfig) (*iperf.Result, error) {
	result, err := a.mgr.RunClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("run iperf client: %w", err)
	}
	return result, nil
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
		return nil, fmt.Errorf("wifi scan: %w", err)
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

func (a *vulnScannerAdapter) ScanDevice(ctx context.Context, device *discovery.DiscoveredDevice) (any, error) {
	result, err := a.scanner.ScanDevice(ctx, device)
	if err != nil {
		return nil, fmt.Errorf("scan device vulnerabilities: %w", err)
	}
	return result, nil
}

func (a *vulnScannerAdapter) GetAllVulnerabilities() any {
	return a.scanner.GetAllVulnerabilities()
}

// GetDiscoveryService returns the discovery service adapter.
func (s *Server) GetDiscoveryService() mcp.DiscoveryService {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Shell != nil && s.modules.Shell.Discovery() != nil {
		if svc := s.modules.Shell.Discovery().Service(); svc != nil {
			return &discoveryServiceAdapter{svc: svc}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.discoveryService == nil {
		return nil
	}
	return &discoveryServiceAdapter{svc: s.discoveryService}
}

// GetDeviceDiscovery returns the device discovery adapter.
func (s *Server) GetDeviceDiscovery() mcp.DeviceDiscovery {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Shell != nil && s.modules.Shell.Discovery() != nil {
		if dd := s.modules.Shell.Discovery().DeviceDiscovery(); dd != nil {
			return &deviceDiscoveryAdapter{dd: dd}
		}
	}
	// Fallback to direct instance for backward compatibility
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
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Sap != nil && s.modules.Sap.VLAN() != nil {
		if mgr := s.modules.Sap.VLAN().Manager(); mgr != nil {
			return &vlanManagerWrapperImpl{mgr: mgr}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.vlanManager == nil {
		return nil
	}
	return &vlanManagerWrapperImpl{mgr: s.vlanManager}
}

// vlanManagerWrapperImpl wraps vlan.Manager to implement mcp.VLANManager.
type vlanManagerWrapperImpl struct {
	mgr *vlan.Manager
}

func (w *vlanManagerWrapperImpl) GetInfo() any {
	return w.mgr.GetInfo()
}

// GetDNSTester returns the DNS tester adapter.
func (s *Server) GetDNSTester() mcp.DNSTester {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Sap != nil && s.modules.Sap.DNS() != nil {
		if tester := s.modules.Sap.DNS().Tester(); tester != nil {
			return &dnsTesterAdapter{tester: tester}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.dnsTester == nil {
		return nil
	}
	return &dnsTesterAdapter{tester: s.dnsTester}
}

// GetGatewayTester returns the gateway tester adapter.
func (s *Server) GetGatewayTester() mcp.GatewayTester {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Sap != nil && s.modules.Sap.Gateway() != nil {
		if tester := s.modules.Sap.Gateway().Tester(); tester != nil {
			return &gatewayTesterAdapter{tester: tester}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.gatewayTester == nil {
		return nil
	}
	return &gatewayTesterAdapter{tester: s.gatewayTester}
}

// GetSpeedtestTester returns the speedtest tester adapter.
func (s *Server) GetSpeedtestTester() mcp.SpeedtestTester {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Sap != nil && s.modules.Sap.Performance() != nil {
		if tester := s.modules.Sap.Performance().SpeedtestTester(); tester != nil {
			return &speedtestTesterAdapter{tester: tester}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.speedtestTester == nil {
		return nil
	}
	return &speedtestTesterAdapter{tester: s.speedtestTester}
}

// GetIperfManager returns the iPerf manager adapter.
func (s *Server) GetIperfManager() mcp.IperfManager {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Sap != nil && s.modules.Sap.Performance() != nil {
		if mgr := s.modules.Sap.Performance().IPerfManager(); mgr != nil {
			return &iperfManagerAdapter{mgr: mgr}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.iperfManager == nil {
		return nil
	}
	return &iperfManagerAdapter{mgr: s.iperfManager}
}

// GetWiFiScanner returns the WiFi scanner adapter.
func (s *Server) GetWiFiScanner() mcp.WiFiScanner {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Canopy != nil && s.modules.Canopy.WiFi() != nil {
		if scanner := s.modules.Canopy.WiFi().Scanner(); scanner != nil {
			return &wifiScannerAdapter{scanner: scanner}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.wifiScanner == nil {
		return nil
	}
	return &wifiScannerAdapter{scanner: s.wifiScanner}
}

// GetWiFiManager returns the WiFi manager adapter.
func (s *Server) GetWiFiManager() mcp.WiFiManager {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Canopy != nil && s.modules.Canopy.WiFi() != nil {
		if mgr := s.modules.Canopy.WiFi().Manager(); mgr != nil {
			return &wifiManagerAdapter{mgr: mgr}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.wifiManager == nil {
		return nil
	}
	return &wifiManagerAdapter{mgr: s.wifiManager}
}

// GetRogueDetector returns the rogue DHCP detector adapter.
func (s *Server) GetRogueDetector() mcp.RogueDetector {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Shell != nil && s.modules.Shell.Rogue() != nil {
		if detector := s.modules.Shell.Rogue().Detector(); detector != nil {
			return &rogueDetectorWrapperImpl{detector: detector}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.rogueDetector == nil {
		return nil
	}
	return &rogueDetectorWrapperImpl{detector: s.rogueDetector}
}

// rogueDetectorWrapperImpl wraps dhcp.RogueDetector to implement mcp.RogueDetector.
type rogueDetectorWrapperImpl struct {
	detector *dhcp.RogueDetector
}

func (w *rogueDetectorWrapperImpl) GetDetectedServers() any {
	return w.detector.GetDetectedServers()
}

func (w *rogueDetectorWrapperImpl) IsRunning() bool {
	return w.detector.IsRunning()
}

// GetVulnScanner returns the vulnerability scanner adapter.
func (s *Server) GetVulnScanner() mcp.VulnScanner {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Shell != nil && s.modules.Shell.Vulnerability() != nil {
		if scanner := s.modules.Shell.Vulnerability().Scanner(); scanner != nil {
			return &vulnScannerAdapter{scanner: scanner}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.vulnScanner == nil {
		return nil
	}
	return &vulnScannerAdapter{scanner: s.vulnScanner}
}

// GetPublicIPChecker returns the public IP checker adapter.
func (s *Server) GetPublicIPChecker() mcp.PublicIPChecker {
	// Use module service accessor (Phase A transition)
	if s.modules != nil && s.modules.Roots != nil && s.modules.Roots.Enrichment() != nil {
		if checker := s.modules.Roots.Enrichment().Checker(); checker != nil {
			return &publicIPCheckerWrapperImpl{checker: checker}
		}
	}
	// Fallback to direct instance for backward compatibility
	if s.publicipChecker == nil {
		return nil
	}
	return &publicIPCheckerWrapperImpl{checker: s.publicipChecker}
}

// publicIPCheckerWrapperImpl wraps publicip.Checker to implement mcp.PublicIPChecker.
type publicIPCheckerWrapperImpl struct {
	checker *publicip.Checker
}

func (w *publicIPCheckerWrapperImpl) GetPublicIP(ctx context.Context) any {
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
