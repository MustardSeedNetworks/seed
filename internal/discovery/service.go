package discovery

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/config"
)

// Service is the unified discovery orchestrator that applies profile-based
// configuration to control which discovery methods are active.
type Service struct {
	cfg             *config.Config
	interfaceName   string
	deviceDiscovery *DeviceDiscovery
	profiler        *DeviceProfiler

	// Runtime state
	mu            sync.RWMutex
	running       bool
	activeProfile config.DiscoveryProfile
	rescanTicker  *time.Ticker
	stopCh        chan struct{}
}

// ServiceStatus represents the current state of the discovery service.
type ServiceStatus struct {
	Running        bool                    `json:"running"`
	Profile        config.DiscoveryProfile `json:"profile"`
	Scanning       bool                    `json:"scanning"`
	DeviceCount    int                     `json:"deviceCount"`
	LastScan       time.Time               `json:"lastScan"`
	Subnet         string                  `json:"subnet"`
	LocalIP        string                  `json:"localIP"`
	Interface      string                  `json:"interface"`
	ActiveMethods  []string                `json:"activeMethods"`
	RescanInterval time.Duration           `json:"rescanInterval"`
}

// NewService creates a new unified discovery service.
func NewService(cfg *config.Config, interfaceName string) *Service {
	return &Service{
		cfg:             cfg,
		interfaceName:   interfaceName,
		deviceDiscovery: NewDeviceDiscoveryWithOUI(interfaceName, cfg.NetworkDiscovery.OUIFilePath, cfg.NetworkDiscovery.OUIMaxAge),
		profiler:        NewDeviceProfiler(DefaultProfilerConfig(), &cfg.SNMP),
		activeProfile:   cfg.NetworkDiscovery.Profile,
	}
}

// Start begins the discovery service based on the configured profile.
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.stopCh = make(chan struct{})
	profile := s.cfg.NetworkDiscovery.Profile
	s.activeProfile = profile

	log.Printf("Starting discovery service with profile: %s", profile)

	// Apply profile settings
	if err := s.applyProfile(profile); err != nil {
		return err
	}

	s.running = true

	// Start the profiler
	s.profiler.Start()

	// Start background rescan loop if configured
	rescanInterval := s.cfg.NetworkDiscovery.Timing.RescanInterval
	if rescanInterval > 0 && s.shouldDoActiveScan(profile) {
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
	log.Printf("Discovery service stopped")
}

// applyProfile configures discovery methods based on the selected profile.
// Must be called with s.mu held.
func (s *Service) applyProfile(profile config.DiscoveryProfile) error {
	switch profile {
	case config.ProfileStealth:
		// Stealth: passive listening only (LLDP, CDP, EDP, DHCP)
		// No active scanning
		log.Printf("Profile stealth: enabling passive protocol listeners only")
		return s.deviceDiscovery.Start() // Starts LLDP/CDP/EDP listeners

	case config.ProfileStandard:
		// Standard: passive + ARP/ICMP on local subnet
		log.Printf("Profile standard: enabling passive listeners + ARP/ICMP scan")
		if err := s.deviceDiscovery.Start(); err != nil {
			return err
		}
		// Configure for local subnet only
		if err := s.deviceDiscovery.SetAdditionalSubnets(nil); err != nil {
			log.Printf("Warning: failed to clear additional subnets: %v", err)
		}
		return nil

	case config.ProfileFullScan:
		// Full scan: all methods including additional subnets and port scanning
		log.Printf("Profile full_scan: enabling all discovery methods")
		if err := s.deviceDiscovery.Start(); err != nil {
			return err
		}
		// Configure additional subnets from config
		cidrs := make([]string, 0)
		for _, subnet := range s.cfg.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled && subnet.CIDR != "" {
				cidrs = append(cidrs, subnet.CIDR)
			}
		}
		return s.deviceDiscovery.SetAdditionalSubnets(cidrs)

	case config.ProfileCustom:
		// Custom: fine-grained control via CustomOptions
		log.Printf("Profile custom: applying custom options")
		return s.applyCustomOptions()

	default:
		// Default to standard
		log.Printf("Unknown profile %s, defaulting to standard", profile)
		return s.applyProfile(config.ProfileStandard)
	}
}

// applyCustomOptions applies fine-grained custom discovery options.
func (s *Service) applyCustomOptions() error {
	opts := s.cfg.NetworkDiscovery.CustomOptions

	// Start passive listeners if enabled
	if opts.PassiveListen {
		if err := s.deviceDiscovery.Start(); err != nil {
			return err
		}
	}

	// Configure additional subnets if needed
	if opts.ARPScan || opts.ICMPScan || opts.PortScan.Enabled {
		cidrs := make([]string, 0)
		for _, subnet := range s.cfg.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled && subnet.CIDR != "" {
				cidrs = append(cidrs, subnet.CIDR)
			}
		}
		if err := s.deviceDiscovery.SetAdditionalSubnets(cidrs); err != nil {
			log.Printf("Warning: failed to set additional subnets: %v", err)
		}
	}

	return nil
}

// shouldDoActiveScan returns true if the profile includes active scanning.
func (s *Service) shouldDoActiveScan(profile config.DiscoveryProfile) bool {
	switch profile {
	case config.ProfileStealth:
		return false
	case config.ProfileStandard, config.ProfileFullScan:
		return true
	case config.ProfileCustom:
		opts := s.cfg.NetworkDiscovery.CustomOptions
		return opts.ARPScan || opts.ICMPScan || opts.PortScan.Enabled
	default:
		return true
	}
}

// rescanLoop periodically triggers network scans based on RescanInterval.
func (s *Service) rescanLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		case <-s.rescanTicker.C:
			log.Printf("Discovery: starting scheduled rescan")
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.NetworkDiscovery.ScanTimeout)
			if err := s.Scan(ctx); err != nil {
				log.Printf("Discovery: scheduled rescan failed: %v", err)
			}
			cancel()
		}
	}
}

// Scan performs an active network scan (if the profile allows it).
func (s *Service) Scan(ctx context.Context) error {
	s.mu.RLock()
	profile := s.activeProfile
	running := s.running
	s.mu.RUnlock()

	if !running {
		return nil
	}

	if !s.shouldDoActiveScan(profile) {
		log.Printf("Discovery: active scanning disabled for profile %s", profile)
		return nil
	}

	return s.deviceDiscovery.Scan(ctx)
}

// SetProfile changes the active discovery profile at runtime.
func (s *Service) SetProfile(profile config.DiscoveryProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeProfile == profile {
		return nil // No change needed
	}

	wasRunning := s.running

	// Stop current operations
	if wasRunning {
		if s.rescanTicker != nil {
			s.rescanTicker.Stop()
			s.rescanTicker = nil
		}
		s.deviceDiscovery.Stop()
	}

	// Clear discovered devices when switching profiles
	s.deviceDiscovery.ClearDevices()

	// Apply new profile
	s.activeProfile = profile
	log.Printf("Discovery: switching to profile %s", profile)

	if wasRunning {
		if err := s.applyProfile(profile); err != nil {
			return err
		}

		// Restart rescan ticker if needed
		rescanInterval := s.cfg.NetworkDiscovery.Timing.RescanInterval
		if rescanInterval > 0 && s.shouldDoActiveScan(profile) {
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
	profile := s.activeProfile
	running := s.running
	rescanInterval := s.cfg.NetworkDiscovery.Timing.RescanInterval
	s.mu.RUnlock()

	deviceStatus := s.deviceDiscovery.GetStatus()

	status := &ServiceStatus{
		Running:        running,
		Profile:        profile,
		Scanning:       deviceStatus.Scanning,
		DeviceCount:    deviceStatus.DeviceCount,
		LastScan:       deviceStatus.LastScan,
		Subnet:         deviceStatus.Subnet,
		LocalIP:        deviceStatus.LocalIP,
		Interface:      deviceStatus.Interface,
		RescanInterval: rescanInterval,
		ActiveMethods:  s.getActiveMethods(profile),
	}

	return status
}

// getActiveMethods returns a list of currently active discovery methods.
func (s *Service) getActiveMethods(profile config.DiscoveryProfile) []string {
	methods := []string{}

	switch profile {
	case config.ProfileStealth:
		methods = append(methods, "lldp", "cdp", "edp")
	case config.ProfileStandard:
		methods = append(methods, "lldp", "cdp", "edp", "arp", "icmp")
	case config.ProfileFullScan:
		methods = append(methods, "lldp", "cdp", "edp", "arp", "icmp", "port_scan")
	case config.ProfileCustom:
		opts := s.cfg.NetworkDiscovery.CustomOptions
		if opts.PassiveListen {
			methods = append(methods, "lldp", "cdp", "edp")
		}
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
	}

	return methods
}

// IsRunning returns whether the service is currently running.
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetProfile returns the current active profile.
func (s *Service) GetProfile() config.DiscoveryProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeProfile
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
