package sap

import (
	"context"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/sap/cable"
	"github.com/krisarmstrong/seed/internal/sap/dns"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
	"github.com/krisarmstrong/seed/internal/sap/speedtest"
	"github.com/krisarmstrong/seed/internal/sap/vlan"
	"github.com/krisarmstrong/seed/internal/snmp"
)

// DefaultInterface is the default network interface to use when none is configured.
const DefaultInterface = "eth0"

// LinkService monitors network link status.
type LinkService struct {
	cfg        *config.Config
	monitor    *network.LinkMonitor
	netManager *network.Manager
	cancel     context.CancelFunc
}

// NewLinkService creates a new link service.
func NewLinkService(cfg *config.Config) *LinkService {
	return &LinkService{cfg: cfg}
}

// Start begins link monitoring.
func (s *LinkService) Start(ctx context.Context) error {
	_, s.cancel = context.WithCancel(ctx)

	// Get active interface from config
	iface, ok := s.cfg.GetActiveInterface()
	if !ok || iface == "" {
		iface = DefaultInterface
	}

	// Create network manager for interface enumeration
	var err error
	s.netManager, err = network.NewManager(iface)
	if err != nil {
		return fmt.Errorf("creating network manager: %w", err)
	}

	s.monitor = network.NewLinkMonitor(iface)
	if startErr := s.monitor.Start(); startErr != nil {
		return fmt.Errorf("starting link monitor: %w", startErr)
	}
	return nil
}

// Stop halts link monitoring.
func (s *LinkService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.monitor != nil {
		s.monitor.Stop()
	}
}

// GetStatus returns current link status for all interfaces.
func (s *LinkService) GetStatus(_ context.Context) ([]LinkStatus, error) {
	if s.netManager == nil {
		return nil, ErrNotInitialized
	}

	// Get interfaces list from network manager
	interfaces := s.netManager.GetInterfaces()

	result := make([]LinkStatus, 0, len(interfaces))
	for _, iface := range interfaces {
		var state LinkState
		if iface.Up {
			state = LinkStateUp
		} else {
			state = LinkStateDown
		}

		result = append(result, LinkStatus{
			Interface:  iface.Name,
			State:      state,
			Speed:      iface.SpeedDisplay,
			Duplex:     "", // Not available from GetInterfaces
			MTU:        iface.MTU,
			MACAddress: iface.HardwareAddr,
			IPAddress:  joinAddresses(iface.Addresses),
			Carrier:    iface.Up,
			UpdatedAt:  time.Now(),
		})
	}

	return result, nil
}

// GetInterfaceStatus returns status for a specific interface.
func (s *LinkService) GetInterfaceStatus(_ context.Context, name string) (*LinkStatus, error) {
	// Create a temporary monitor for this interface
	mon := network.NewLinkMonitor(name)
	defer mon.Stop()

	// Get current state
	if err := mon.Start(); err != nil {
		return nil, fmt.Errorf("starting interface monitor: %w", err)
	}
	time.Sleep(100 * time.Millisecond) // Brief wait for initial state
	mon.Stop()

	var state LinkState
	switch mon.GetState() {
	case network.LinkStateUp:
		state = LinkStateUp
	case network.LinkStateDown:
		state = LinkStateDown
	case network.LinkStateUnknown:
		state = LinkStateUnknown
	default:
		state = LinkStateUnknown
	}

	return &LinkStatus{
		Interface: name,
		State:     state,
		Carrier:   state == LinkStateUp,
		UpdatedAt: time.Now(),
	}, nil
}

// CableService provides TDR cable testing.
type CableService struct {
	cfg *config.Config
}

// NewCableService creates a new cable service.
func NewCableService(cfg *config.Config) *CableService {
	return &CableService{cfg: cfg}
}

// Test performs a cable test on the specified interface.
func (s *CableService) Test(_ context.Context, iface string) (*CableTestResult, error) {
	tester := cable.NewTester(iface)

	if !tester.IsSupported() {
		return &CableTestResult{
			Interface: iface,
			Status:    CableStatusUnknown,
			TestedAt:  time.Now(),
		}, ErrNotSupported
	}

	result := tester.Test()
	if result == nil {
		return nil, ErrTestFailed
	}

	// Convert cable.TestResult to sap.CableTestResult
	sapResult := &CableTestResult{
		Interface:   iface,
		Status:      convertCableStatus(result.Status),
		PairResults: convertPairResults(result.Pairs),
		TestedAt:    time.Now(),
	}

	if result.Length != nil {
		sapResult.Length = *result.Length
	}

	return sapResult, nil
}

// DHCPService provides DHCP testing.
type DHCPService struct {
	cfg *config.Config
}

// NewDHCPService creates a new DHCP service.
func NewDHCPService(cfg *config.Config) *DHCPService {
	return &DHCPService{cfg: cfg}
}

// Test performs a DHCP discovery test.
func (s *DHCPService) Test(_ context.Context, iface string) (*DHCPTestResult, error) {
	// Get DHCP lease info from the system
	leaseInfo, err := dhcp.GetLeaseInfo(iface)

	result := &DHCPTestResult{
		Success:  leaseInfo != nil && err == nil,
		TestedAt: time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
	}

	if leaseInfo != nil {
		result.ServerIP = leaseInfo.DHCPServer
		result.Gateway = leaseInfo.Gateway
		result.DNSServers = leaseInfo.DNS
		if leaseInfo.LeaseTime > 0 {
			result.LeaseTime = time.Duration(leaseInfo.LeaseTime) * time.Second
			result.LeaseTimeSec = leaseInfo.LeaseTime
		}
	}

	return result, nil
}

// DNSService provides DNS testing and security scanning.
type DNSService struct {
	cfg             *config.Config
	tester          *dns.Tester
	securityScanner *dns.SecurityScanner
}

// NewDNSService creates a new DNS service.
func NewDNSService(cfg *config.Config) *DNSService {
	tester := dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds())
	// Configure servers from config
	configuredServers := make([]dns.ConfiguredServer, 0)
	for _, srv := range cfg.DNS.Servers {
		configuredServers = append(configuredServers, dns.ConfiguredServer{
			Address: srv.Address,
			Enabled: srv.Enabled,
		})
	}
	tester.SetConfiguredServers(configuredServers)

	return &DNSService{
		cfg:             cfg,
		tester:          tester,
		securityScanner: dns.NewSecurityScanner(dns.DefaultSecurityScanConfig()),
	}
}

// Tester returns the underlying DNS tester for direct access.
// This allows handlers to use existing patterns during transition.
func (s *DNSService) Tester() *dns.Tester {
	return s.tester
}

// SecurityScanner returns the underlying security scanner for direct access.
func (s *DNSService) SecurityScanner() *dns.SecurityScanner {
	return s.securityScanner
}

// Test performs a DNS query test.
func (s *DNSService) Test(ctx context.Context, server, hostname string) (*DNSTestResult, error) {
	if hostname == "" {
		hostname = "google.com"
	}

	// Update tester settings
	if server != "" {
		s.tester.SetServer(server)
	}
	s.tester.SetTestHostname(hostname)

	// Perform the test
	result := s.tester.Test(ctx)

	// Convert to sap.DNSTestResult
	sapResult := &DNSTestResult{
		Query:    hostname,
		Server:   result.Server,
		Success:  result.Forward != nil && result.Forward.Status != dns.StatusError,
		TestedAt: time.Now(),
	}

	if result.Forward != nil {
		sapResult.ResponseTime = result.Forward.Time
		sapResult.ResponseMs = float64(result.Forward.TimeMs)
		if len(result.Forward.Resolved) > 0 {
			sapResult.Answers = make([]DNSAnswer, 0, len(result.Forward.Resolved))
			for _, addr := range result.Forward.Resolved {
				sapResult.Answers = append(sapResult.Answers, DNSAnswer{
					Name:  hostname,
					Type:  "A",
					Value: addr,
				})
			}
		}
		if result.Forward.Error != "" {
			sapResult.Error = result.Forward.Error
		}
	}

	return sapResult, nil
}

// GatewayService monitors gateway health.
type GatewayService struct {
	cfg    *config.Config
	tester *gateway.Tester
	cancel context.CancelFunc
}

// NewGatewayService creates a new gateway service.
func NewGatewayService(cfg *config.Config) *GatewayService {
	tester := gateway.NewTester(gateway.DefaultThresholds())
	return &GatewayService{
		cfg:    cfg,
		tester: tester,
	}
}

// Tester returns the underlying gateway tester for direct access.
func (s *GatewayService) Tester() *gateway.Tester {
	return s.tester
}

// Start begins gateway monitoring.
func (s *GatewayService) Start(ctx context.Context) error {
	_, s.cancel = context.WithCancel(ctx)

	// Auto-detect gateway
	gw, err := gateway.DetectGateway()
	if err == nil && gw != "" {
		s.tester.SetGateway(gw)
	}

	return nil
}

// Stop halts gateway monitoring.
func (s *GatewayService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.tester != nil {
		s.tester.Close()
	}
}

// GetHealth returns current gateway health.
func (s *GatewayService) GetHealth(_ context.Context) (*GatewayHealth, error) {
	if s.tester == nil {
		return nil, ErrNotInitialized
	}

	stats := s.tester.Test()

	// Convert gateway.PingStats to sap.GatewayHealth
	health := &GatewayHealth{
		IP:         stats.Gateway,
		Reachable:  stats.Reachable,
		RTT:        time.Duration(stats.AvgTime * float64(time.Millisecond)),
		RTTMs:      stats.AvgTime,
		PacketLoss: stats.LossPercent,
		Status:     convertGatewayStatus(stats.Status),
		LastCheck:  stats.LastUpdated,
	}

	return health, nil
}

// SNMPService collects SNMP data from network devices.
type SNMPService struct {
	cfg *config.Config
}

// NewSNMPService creates a new SNMP service.
func NewSNMPService(cfg *config.Config) *SNMPService {
	return &SNMPService{cfg: cfg}
}

// Collect gathers SNMP data from a device.
func (s *SNMPService) Collect(ctx context.Context, ip, community string) (*SNMPDevice, error) {
	// Copy SNMP config to allow overriding community
	snmpCfg := s.cfg.SNMP
	if community != "" {
		// Prepend the specified community to try it first
		snmpCfg.Communities = append([]string{community}, snmpCfg.Communities...)
	}

	sysInfo, err := snmp.GetSystemInfo(ctx, ip, &snmpCfg)
	if err != nil {
		return nil, fmt.Errorf("collecting SNMP data from %s: %w", ip, err)
	}

	device := &SNMPDevice{
		IP:          ip,
		SysName:     sysInfo.SysName,
		SysDescr:    sysInfo.SysDescr,
		SysLocation: sysInfo.SysLocation,
		SysContact:  sysInfo.SysContact,
		SysUpTime:   time.Duration(sysInfo.SysUpTime) * time.Second / 100, // timeticks to duration
		CollectedAt: time.Now(),
	}

	return device, nil
}

// GetInterfaces retrieves interface data via SNMP.
func (s *SNMPService) GetInterfaces(_ context.Context, _, _ string) ([]SNMPInterface, error) {
	// TODO: Implement via snmp.GetInterfaces when available
	return nil, ErrNotImplemented
}

// GetMACTable retrieves MAC address table via SNMP.
func (s *SNMPService) GetMACTable(_ context.Context, _, _ string) ([]MACTableEntry, error) {
	// TODO: Implement via snmp.GetMACTable when available
	return nil, ErrNotImplemented
}

// PerformanceService provides speed and throughput testing.
type PerformanceService struct {
	cfg       *config.Config
	speedtest *speedtest.Tester
	iperfMgr  *iperf.Manager
	cancel    context.CancelFunc
}

// NewPerformanceService creates a new performance service.
func NewPerformanceService(cfg *config.Config) *PerformanceService {
	return &PerformanceService{
		cfg:       cfg,
		speedtest: speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID),
		iperfMgr:  iperf.NewManager(),
	}
}

// SpeedtestTester returns the underlying speedtest tester for direct access.
func (s *PerformanceService) SpeedtestTester() *speedtest.Tester {
	return s.speedtest
}

// IPerfManager returns the underlying iPerf manager for direct access.
func (s *PerformanceService) IPerfManager() *iperf.Manager {
	return s.iperfMgr
}

// Stop halts any running tests.
func (s *PerformanceService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.iperfMgr != nil {
		_ = s.iperfMgr.StopServer()
	}
}

// Speedtest runs an internet speed test.
func (s *PerformanceService) Speedtest(ctx context.Context) (*SpeedtestResult, error) {
	result, err := s.speedtest.RunTest(ctx)
	if err != nil {
		return nil, fmt.Errorf("running speed test: %w", err)
	}

	return &SpeedtestResult{
		DownloadMbps: result.Download,
		UploadMbps:   result.Upload,
		PingMs:       result.Latency,
		ServerName:   result.Server,
		TestedAt:     result.Timestamp,
		TestDuration: time.Duration(result.TestDuration * float64(time.Second)),
	}, nil
}

// IPerf runs an iPerf test.
func (s *PerformanceService) IPerf(ctx context.Context, host string, options map[string]any) (*IPerfResult, error) {
	// Build client config
	clientCfg := &iperf.ClientConfig{
		Server:   host,
		Port:     5201,
		Protocol: "tcp",
		Duration: 10,
		Parallel: 1,
	}

	if p, ok := options["port"].(int); ok {
		clientCfg.Port = p
	}
	if d, ok := options["duration"].(int); ok {
		clientCfg.Duration = d
	}
	if proto, ok := options["protocol"].(string); ok {
		clientCfg.Protocol = proto
	}
	if reverse, ok := options["reverse"].(bool); ok {
		clientCfg.Reverse = reverse
	}

	result, err := s.iperfMgr.RunClient(ctx, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("running iPerf test to %s: %w", host, err)
	}

	return &IPerfResult{
		Protocol:      result.Protocol,
		Direction:     result.Direction,
		BandwidthMbps: result.Bandwidth, // Mbps
		TransferMB:    result.Transfer,  // Already in MB
		Duration:      time.Duration(result.Duration * float64(time.Second)),
		DurationSec:   result.Duration,
		Jitter:        result.Jitter, // ms
		PacketLoss:    result.LostPercent,
		Retransmits:   result.Retransmits,
		ServerAddr:    host,
		TestedAt:      time.Now(),
	}, nil
}

// VLANService manages VLAN configuration.
type VLANService struct {
	cfg            *config.Config
	manager        *vlan.Manager
	trafficMonitor *vlan.TrafficMonitor
}

// NewVLANService creates a new VLAN service.
func NewVLANService(cfg *config.Config) *VLANService {
	iface, ok := cfg.GetActiveInterface()
	if !ok || iface == "" {
		iface = DefaultInterface
	}
	return &VLANService{
		cfg:            cfg,
		manager:        vlan.NewManager(iface),
		trafficMonitor: vlan.NewTrafficMonitor(iface),
	}
}

// Manager returns the underlying VLAN manager for direct access.
func (s *VLANService) Manager() *vlan.Manager {
	return s.manager
}

// TrafficMonitor returns the underlying traffic monitor for direct access.
func (s *VLANService) TrafficMonitor() *vlan.TrafficMonitor {
	return s.trafficMonitor
}

// Create creates a VLAN subinterface.
func (s *VLANService) Create(_ context.Context, iface string, vlanID int) error {
	if err := vlan.CreateVlanInterface(iface, vlanID); err != nil {
		return fmt.Errorf("creating VLAN %d on %s: %w", vlanID, iface, err)
	}
	return nil
}

// Delete removes a VLAN subinterface.
func (s *VLANService) Delete(_ context.Context, iface string) error {
	// Parse interface name to get parent and VLAN ID
	// Format: eth0.100 -> parent=eth0, vlanID=100
	// This is handled by the vlan package
	if err := vlan.DeleteVlanInterface(iface, 0); err != nil {
		return fmt.Errorf("deleting VLAN interface %s: %w", iface, err)
	}
	return nil
}

// List returns all VLAN configurations.
func (s *VLANService) List(_ context.Context) ([]VLANConfig, error) {
	info := s.manager.GetInfo()
	if info == nil {
		return nil, nil
	}

	// Get active interface for config
	activeIface, _ := s.cfg.GetActiveInterface()
	if activeIface == "" {
		activeIface = DefaultInterface
	}

	configs := make([]VLANConfig, 0)

	if info.NativeVlan != nil {
		configs = append(configs, VLANConfig{
			ID:        *info.NativeVlan,
			Name:      "Native",
			Interface: activeIface,
			Tagged:    false,
		})
	}

	for _, vid := range info.TaggedVlans {
		configs = append(configs, VLANConfig{
			ID:        vid,
			Interface: activeIface,
			Tagged:    true,
		})
	}

	return configs, nil
}

// TelemetryService aggregates and stores telemetry data.
type TelemetryService struct {
	cfg    *config.Config
	db     *database.DB
	cancel context.CancelFunc
}

// NewTelemetryService creates a new telemetry service.
func NewTelemetryService(cfg *config.Config, db *database.DB) *TelemetryService {
	return &TelemetryService{
		cfg: cfg,
		db:  db,
	}
}

// Start begins telemetry collection.
func (s *TelemetryService) Start(ctx context.Context) error {
	_, s.cancel = context.WithCancel(ctx)
	// TODO: Implement telemetry aggregation loop
	return nil
}

// Stop halts telemetry collection.
func (s *TelemetryService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// GetSnapshot returns the latest telemetry snapshot.
func (s *TelemetryService) GetSnapshot(_ context.Context) (*TelemetrySnapshot, error) {
	// TODO: Implement snapshot retrieval from database
	return nil, ErrNotImplemented
}

// GetHistory returns historical telemetry data.
func (s *TelemetryService) GetHistory(_ context.Context, _, _ string) ([]TelemetrySnapshot, error) {
	// TODO: Implement history retrieval from database
	return nil, ErrNotImplemented
}

// Helper functions

func joinAddresses(addrs []string) string {
	if len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

func convertCableStatus(status cable.Status) CableStatus {
	switch status {
	case cable.StatusOK:
		return CableStatusOK
	case cable.StatusOpen:
		return CableStatusOpen
	case cable.StatusShort:
		return CableStatusShort
	case cable.StatusImpedanceMismatch:
		return CableStatusImpedance
	case cable.StatusCrosstalk, cable.StatusSplitPair, cable.StatusUnknown:
		return CableStatusUnknown
	}
	return CableStatusUnknown
}

func convertPairResults(pairs []cable.PairResult) []PairResult {
	if len(pairs) == 0 {
		return nil
	}

	result := make([]PairResult, len(pairs))
	for i, p := range pairs {
		result[i] = PairResult{
			Pair:   i + 1,
			Status: convertCableStatus(p.Status),
		}
		if p.LengthM != nil {
			result[i].Length = *p.LengthM
		}
	}
	return result
}

func convertGatewayStatus(status gateway.Status) HealthStatus {
	switch status {
	case gateway.StatusSuccess:
		return HealthStatusHealthy
	case gateway.StatusWarning:
		return HealthStatusDegraded
	case gateway.StatusError:
		return HealthStatusUnhealthy
	case gateway.StatusUnknown:
		return HealthStatusUnknown
	}
	return HealthStatusUnknown
}
