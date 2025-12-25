// Package discovery implements multi-protocol network device discovery.
// Discovery service coordinates all discovery methods (ARP, NDP, LLDP, CDP, EDP, ICMP, profiling)
// with direct settings configuration. Manages orchestration, timing, and aggregation of discovery results.
package discovery

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

// Service is the unified discovery orchestrator that applies direct
// configuration settings to control which discovery methods are active.
type Service struct {
	cfg             *config.Config
	interfaceName   string
	deviceDiscovery *DeviceDiscovery
	profiler        *DeviceProfiler

	// Runtime state
	mu           sync.RWMutex
	running      bool
	rescanTicker *time.Ticker
	stopCh       chan struct{}
}

// ServiceStatus represents the current state of the discovery service.
type ServiceStatus struct {
	Running        bool          `json:"running"`
	Scanning       bool          `json:"scanning"`
	DeviceCount    int           `json:"deviceCount"`
	LastScan       time.Time     `json:"lastScan"`
	Subnet         string        `json:"subnet"`  // Primary subnet (for backwards compatibility)
	Subnets        []string      `json:"subnets"` // All subnets being scanned (I3)
	LocalIP        string        `json:"localIP"`
	Interface      string        `json:"interface"`
	ActiveMethods  []string      `json:"activeMethods"`
	RescanInterval time.Duration `json:"rescanInterval"`
}

// NewService creates a new unified discovery service.
func NewService(cfg *config.Config, interfaceName string) *Service {
	return &Service{
		cfg:             cfg,
		interfaceName:   interfaceName,
		deviceDiscovery: NewDeviceDiscoveryWithOUI(interfaceName, cfg.NetworkDiscovery.OUIFilePath, cfg.NetworkDiscovery.OUIMaxAge),
		profiler:        NewDeviceProfiler(DefaultProfilerConfig(), &cfg.SNMP),
	}
}

// Start begins the discovery service based on the configured options.
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.stopCh = make(chan struct{})
	opts := &s.cfg.NetworkDiscovery.Options

	slog.Info("Starting discovery service", "methods", s.getActiveMethods())

	// Apply discovery settings
	if err := s.applyOptions(opts); err != nil {
		return err
	}

	s.running = true

	// Start the profiler
	s.profiler.Start()

	// Start background rescan loop if configured
	rescanInterval := s.cfg.NetworkDiscovery.Timing.RescanInterval
	if rescanInterval > 0 && s.shouldDoActiveScan() {
		s.rescanTicker = time.NewTicker(rescanInterval)
		go s.rescanLoop()
	}

	return nil
}

// Stop stops the discovery service.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	if s.stopCh != nil {
		close(s.stopCh)
	}

	if s.rescanTicker != nil {
		s.rescanTicker.Stop()
		s.rescanTicker = nil
	}

	s.deviceDiscovery.Stop()
	s.profiler.Stop()
	s.running = false
	slog.Info("Discovery service stopped")
}

// applyOptions configures discovery methods based on the direct settings.
// Must be called with s.mu held.
func (s *Service) applyOptions(opts *config.DiscoveryOptions) error {
	// Start passive listeners if any passive protocols are enabled
	passiveEnabled := opts.PassiveProtocols.LLDP || opts.PassiveProtocols.CDP ||
		opts.PassiveProtocols.EDP || opts.PassiveProtocols.NDP
	if passiveEnabled {
		if err := s.deviceDiscovery.Start(); err != nil {
			return err
		}
	}

	// Configure additional subnets if any active scanning is enabled
	if opts.ARPScan || opts.ICMPScan || opts.PortScan.Enabled {
		cidrs := make([]string, 0)
		for _, subnet := range s.cfg.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled && subnet.CIDR != "" {
				cidrs = append(cidrs, subnet.CIDR)
			}
		}
		if err := s.deviceDiscovery.SetAdditionalSubnets(cidrs); err != nil {
			slog.Warn("Failed to set additional subnets", "error", err)
		}
	}

	return nil
}

// shouldDoActiveScan returns true if any active scanning methods are enabled.
func (s *Service) shouldDoActiveScan() bool {
	opts := s.cfg.NetworkDiscovery.Options
	return opts.ARPScan || opts.ICMPScan || opts.PortScan.Enabled
}

// rescanLoop periodically triggers network scans based on RescanInterval.
func (s *Service) rescanLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		case <-s.rescanTicker.C:
			slog.Debug("Discovery: starting scheduled rescan")
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.NetworkDiscovery.ScanTimeout)
			if err := s.Scan(ctx); err != nil {
				slog.Warn("Discovery: scheduled rescan failed", "error", err)
			}
			cancel()
		}
	}
}

// Scan performs an active network scan (if enabled in options).
func (s *Service) Scan(ctx context.Context) error {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return nil
	}

	if !s.shouldDoActiveScan() {
		slog.Debug("Discovery: active scanning disabled in options")
		return nil
	}

	if err := s.deviceDiscovery.Scan(ctx); err != nil {
		return err
	}

	// Queue all discovered devices for profiling (port scan, SNMP, HTTP detection)
	s.queueDevicesForProfiling()

	return nil
}

// queueDevicesForProfiling queues all discovered devices for profiling.
func (s *Service) queueDevicesForProfiling() {
	devices := s.deviceDiscovery.GetDevices()
	queued := 0
	for _, device := range devices {
		if device.IP != "" && s.profiler.GetProfile(device.IP) == nil && !s.profiler.IsProfiling(device.IP) {
			s.profiler.QueueProfile(device.IP)
			queued++
		}
	}
	if queued > 0 {
		slog.Info("Queued devices for profiling after scan", "count", queued)
	}
}

// Reload reapplies discovery options from config at runtime.
func (s *Service) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wasRunning := s.running

	// Stop current operations
	if wasRunning {
		if s.rescanTicker != nil {
			s.rescanTicker.Stop()
			s.rescanTicker = nil
		}
		s.deviceDiscovery.Stop()
	}

	opts := &s.cfg.NetworkDiscovery.Options
	slog.Info("Discovery: reloading options", "methods", s.getActiveMethods())

	if wasRunning {
		if err := s.applyOptions(opts); err != nil {
			return err
		}

		// Restart rescan ticker if needed
		rescanInterval := s.cfg.NetworkDiscovery.Timing.RescanInterval
		if rescanInterval > 0 && s.shouldDoActiveScan() {
			s.rescanTicker = time.NewTicker(rescanInterval)
			go s.rescanLoop()
		}
	}

	return nil
}

// SetInterface changes the monitored network interface.
func (s *Service) SetInterface(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.interfaceName = name
	return s.deviceDiscovery.SetInterface(name)
}

// GetDevices returns all discovered devices with their profiles attached.
func (s *Service) GetDevices() []*DiscoveredDevice {
	devices := s.deviceDiscovery.GetDevices()

	// Attach profiles and queue profiling for unprofiled devices
	for _, device := range devices {
		if device.IP != "" {
			if profile := s.profiler.GetProfile(device.IP); profile != nil {
				device.Profile = profile
			} else if !s.profiler.IsProfiling(device.IP) {
				// Queue for profiling
				s.profiler.QueueProfile(device.IP)
			}
		}
	}

	return devices
}

// GetDevice returns a specific device by MAC address.
func (s *Service) GetDevice(mac string) *DiscoveredDevice {
	return s.deviceDiscovery.GetDevice(mac)
}

// GetDeviceByIP returns a device by IP address.
func (s *Service) GetDeviceByIP(ip string) *DiscoveredDevice {
	return s.deviceDiscovery.GetDeviceByIP(ip)
}

// GetNeighbors returns all discovered protocol neighbors (LLDP, CDP, EDP).
func (s *Service) GetNeighbors() []*Neighbor {
	return s.deviceDiscovery.protoManager.GetNeighbors()
}

// GetStatus returns the current status of the discovery service.
func (s *Service) GetStatus() *ServiceStatus {
	s.mu.RLock()
	running := s.running
	rescanInterval := s.cfg.NetworkDiscovery.Timing.RescanInterval
	s.mu.RUnlock()

	deviceStatus := s.deviceDiscovery.GetStatus()

	// Collect all subnets (primary + additional) for I3
	subnets := []string{}
	if deviceStatus.Subnet != "" {
		subnets = append(subnets, deviceStatus.Subnet)
	}
	subnets = append(subnets, s.deviceDiscovery.GetAdditionalSubnets()...)

	status := &ServiceStatus{
		Running:        running,
		Scanning:       deviceStatus.Scanning,
		DeviceCount:    deviceStatus.DeviceCount,
		LastScan:       deviceStatus.LastScan,
		Subnet:         deviceStatus.Subnet,
		Subnets:        subnets,
		LocalIP:        deviceStatus.LocalIP,
		Interface:      deviceStatus.Interface,
		RescanInterval: rescanInterval,
		ActiveMethods:  s.getActiveMethods(),
	}

	return status
}

// getActiveMethods returns a list of currently active discovery methods.
func (s *Service) getActiveMethods() []string {
	opts := s.cfg.NetworkDiscovery.Options
	methods := []string{}

	// Passive protocols
	if opts.PassiveProtocols.LLDP {
		methods = append(methods, "lldp")
	}
	if opts.PassiveProtocols.CDP {
		methods = append(methods, "cdp")
	}
	if opts.PassiveProtocols.EDP {
		methods = append(methods, "edp")
	}
	if opts.PassiveProtocols.NDP {
		methods = append(methods, "ndp")
	}

	// Active scanning methods
	if opts.ARPScan {
		methods = append(methods, "arp")
	}
	if opts.ICMPScan {
		methods = append(methods, "icmp")
	}
	if opts.PortScan.Enabled {
		methods = append(methods, "port_scan")
	}
	if opts.Traceroute {
		methods = append(methods, "traceroute")
	}
	if opts.SNMPQuery {
		methods = append(methods, "snmp")
	}

	return methods
}

// IsRunning returns whether the service is currently running.
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetOptions returns the current discovery options.
func (s *Service) GetOptions() config.DiscoveryOptions {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.NetworkDiscovery.Options
}

// ClearDevices removes all discovered devices and profiles from memory.
func (s *Service) ClearDevices() {
	s.deviceDiscovery.ClearDevices()
	s.profiler.ClearProfiles()
}

// DeviceDiscovery returns the underlying DeviceDiscovery for direct access if needed.
func (s *Service) DeviceDiscovery() *DeviceDiscovery {
	return s.deviceDiscovery
}
