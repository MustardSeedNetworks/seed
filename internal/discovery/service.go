// Package discovery implements multi-protocol network device discovery.
// Discovery service coordinates all discovery methods (ARP, NDP, LLDP, CDP, EDP, ICMP, profiling)
// with direct settings configuration. Manages orchestration, timing, and aggregation of discovery results.
package discovery

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
)

// Service is the unified discovery orchestrator that applies direct
// configuration settings to control which discovery methods are active.
type Service struct {
	cfg             *config.Config
	interfaceName   string
	deviceDiscovery *DeviceDiscovery
	profiler        *DeviceProfiler
	pipeline        *Pipeline // Reference to discovery pipeline for coordination

	// Runtime state
	mu           sync.RWMutex
	running      bool
	rescanTicker *time.Ticker
	stopCh       chan struct{}

	// Metrics tracking
	metrics       *Metrics
	previousScan  []*DiscoveredDevice // For delta computation
	lastDelta     *ScanDelta
	lastDeltaTime time.Time

	// Callback for pipeline completion
	onPipelineComplete func(devices []*DiscoveredDevice)
}

// ServiceStatus represents the current state of the discovery service.
type ServiceStatus struct {
	Running         bool             `json:"running"`
	Scanning        bool             `json:"scanning"`
	DeviceCount     int              `json:"deviceCount"`
	LastScan        time.Time        `json:"lastScan"`
	Subnet          string           `json:"subnet"`  // Primary subnet (for backwards compatibility)
	Subnets         []string         `json:"subnets"` // All subnets being scanned (I3)
	LocalIP         string           `json:"localIP"`
	Interface       string           `json:"interface"`
	ActiveMethods   []string         `json:"activeMethods"`
	RescanInterval  time.Duration    `json:"rescanInterval"`
	PipelineStatus  string           `json:"pipelineStatus,omitempty"`  // "idle", "running", "completed", "failed"
	PipelinePhase   string           `json:"pipelinePhase,omitempty"`   // Current phase name
	ProfilingStatus *ProfilingStatus `json:"profilingStatus,omitempty"` // Detailed profiling state

	// Metrics and health
	Metrics           *Metrics           `json:"metrics,omitempty"`
	LastDelta         *ScanDelta         `json:"lastDelta,omitempty"`
	DegradationStatus *DegradationStatus `json:"degradationStatus,omitempty"`
}

// NewService creates a new unified discovery service.
// If profiler is nil, a new DeviceProfiler is created internally.
// If profiler is provided, it will be shared (e.g., with Pipeline).
func NewService(cfg *config.Config, interfaceName string, profiler *DeviceProfiler) *Service {
	if profiler == nil {
		profiler = NewDeviceProfiler(DefaultProfilerConfig(), &cfg.SNMP)
	}
	return &Service{
		cfg:           cfg,
		interfaceName: interfaceName,
		deviceDiscovery: NewDeviceDiscoveryWithOUI(
			interfaceName,
			cfg.NetworkDiscovery.OUIFilePath,
			cfg.NetworkDiscovery.OUIMaxAge,
		),
		profiler: profiler,
		metrics:  NewMetrics(),
	}
}

// SetPipeline sets the discovery pipeline reference for coordination.
// This enables the service to trigger pipeline runs and receive completion callbacks.
// Automatically wires up the onComplete callback to sync pipeline results.
func (s *Service) SetPipeline(pipeline *Pipeline) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pipeline = pipeline

	// Wire up completion callback to sync pipeline results back to Service
	if pipeline != nil {
		pipeline.SetOnComplete(func(devices []*DiscoveredDevice) {
			s.syncPipelineResults(devices)
		})
	}
}

// SetOnPipelineComplete sets a callback that's called when pipeline completes.
// This allows the service to update its device cache with pipeline results.
func (s *Service) SetOnPipelineComplete(callback func(devices []*DiscoveredDevice)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onPipelineComplete = callback
}

// GetProfiler returns the shared DeviceProfiler instance.
func (s *Service) GetProfiler() *DeviceProfiler {
	return s.profiler
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

	logging.GetLogger().Info("Starting discovery service", "methods", s.getActiveMethods())

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
	logging.GetLogger().Info("Discovery service stopped")
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
			logging.GetLogger().Warn("Failed to set additional subnets", "error", err)
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
	// Capture ticker channel at start to avoid race with Stop() setting ticker to nil
	s.mu.RLock()
	ticker := s.rescanTicker
	s.mu.RUnlock()

	if ticker == nil {
		return
	}

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			logging.GetLogger().Debug("Discovery: starting scheduled rescan")
			ctx, cancel := context.WithTimeout(
				context.Background(),
				s.cfg.NetworkDiscovery.ScanTimeout,
			)
			if err := s.Scan(ctx); err != nil {
				logging.GetLogger().Warn("Discovery: scheduled rescan failed", "error", err)
			}
			cancel()
		}
	}
}

// Scan performs an active network scan (if enabled in options).
// Returns ErrScanInProgress if a scan is already running.
func (s *Service) Scan(ctx context.Context) error {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return nil
	}

	if !s.shouldDoActiveScan() {
		logging.GetLogger().DebugContext(ctx, "Discovery: active scanning disabled in options")
		return nil
	}

	scanStart := time.Now()

	if err := s.deviceDiscovery.Scan(ctx); err != nil {
		// ErrScanInProgress is not a failure - just means scan was already running
		if errors.Is(err, ErrScanInProgress) {
			logging.GetLogger().DebugContext(ctx, "Discovery: scan skipped - already in progress")
			return err // Return error so callers know it was skipped
		}
		s.mu.Lock()
		s.metrics.RecordError("arp") // Record as ARP error (primary scan method)
		s.mu.Unlock()
		return err
	}

	// Update metrics and compute delta
	s.mu.Lock()
	devices := s.deviceDiscovery.GetDevices()
	s.metrics.UpdateFromDevices(devices)
	s.metrics.LastScanStart = scanStart
	s.metrics.LastScanDuration = time.Since(scanStart)

	// Compute delta from previous scan
	if s.previousScan != nil {
		s.lastDelta = ComputeDelta(s.previousScan, devices)
		s.lastDeltaTime = time.Now()

		if len(s.lastDelta.NewDevices) > 0 || len(s.lastDelta.RemovedDevices) > 0 {
			logging.GetLogger().InfoContext(ctx, "Discovery scan delta",
				"new", len(s.lastDelta.NewDevices),
				"updated", len(s.lastDelta.UpdatedDevices),
				"removed", len(s.lastDelta.RemovedDevices))
		}
	}
	// Store copy for next delta computation
	s.previousScan = make([]*DiscoveredDevice, len(devices))
	copy(s.previousScan, devices)
	s.mu.Unlock()

	// Queue all discovered devices for profiling (port scan, SNMP, HTTP detection)
	s.queueDevicesForProfiling()

	return nil
}

// queueDevicesForProfiling queues all discovered devices for profiling.
// Skips queuing if pipeline is running (pipeline handles profiling in Phase 3).
func (s *Service) queueDevicesForProfiling() {
	// Skip if pipeline is running - it handles profiling in Phase 3
	// This eliminates dual orchestration and duplicate profiling work
	s.mu.RLock()
	pipeline := s.pipeline
	s.mu.RUnlock()

	if pipeline != nil {
		status := pipeline.GetStatus()
		if isRunningPipelineState(status.Status) {
			logging.GetLogger().Debug("Skipping device profiling - pipeline is running")
			return
		}
	}

	devices := s.deviceDiscovery.GetDevices()
	queued := 0
	for _, device := range devices {
		if device.IP != "" && s.profiler.GetProfile(device.IP) == nil &&
			!s.profiler.IsProfiling(device.IP) {
			_ = s.profiler.QueueProfile(device.IP)
			queued++
		}
	}
	if queued > 0 {
		logging.GetLogger().Info("Queued devices for profiling after scan", "count", queued)
	}
}

// isRunningPipelineState checks if a pipeline state indicates active execution.
func isRunningPipelineState(state PipelineState) bool {
	return state == PipelineStateEnumerating ||
		state == PipelineStateResolving ||
		state == PipelineStateScanning ||
		state == PipelineStateAssessing
}

// Reload reapplies discovery options from config at runtime.
func (s *Service) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wasRunning := s.running

	// Stop current operations
	if wasRunning {
		// Close old stopCh to terminate existing rescanLoop goroutine (fixes #817)
		if s.stopCh != nil {
			close(s.stopCh)
		}
		if s.rescanTicker != nil {
			s.rescanTicker.Stop()
			s.rescanTicker = nil
		}
		s.deviceDiscovery.Stop()
	}

	opts := &s.cfg.NetworkDiscovery.Options
	logging.GetLogger().Info("Discovery: reloading options", "methods", s.getActiveMethods())

	if wasRunning {
		// Create new stopCh for the new rescanLoop goroutine
		s.stopCh = make(chan struct{})

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
				_ = s.profiler.QueueProfile(device.IP)
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
	pipeline := s.pipeline
	metrics := s.metrics
	lastDelta := s.lastDelta
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

	// Add pipeline status if available
	if pipeline != nil {
		pipelineRun := pipeline.GetStatus()
		status.PipelineStatus = string(pipelineRun.Status) // Convert PipelineState to string
		status.PipelinePhase = pipelineRun.CurrentPhase
	}

	// Add profiling status
	if s.profiler != nil {
		status.ProfilingStatus = s.profiler.GetProfilingStatus()
	}

	// Add metrics and health status
	if metrics != nil {
		status.Metrics = metrics.Clone()
		status.DegradationStatus = metrics.GetDegradationStatus()
	}
	if lastDelta != nil {
		status.LastDelta = lastDelta
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

// StartPipeline triggers a full discovery pipeline run.
// This is the recommended way to run comprehensive discovery.
// The trigger parameter identifies what initiated the pipeline (e.g., "manual", "scheduled", "api").
// Returns error if pipeline is not configured or already running.
func (s *Service) StartPipeline(ctx context.Context, trigger string) error {
	s.mu.RLock()
	pipeline := s.pipeline
	s.mu.RUnlock()

	if pipeline == nil {
		return &Error{
			Phase:   "start",
			Message: "discovery pipeline not configured",
			Code:    "PIPELINE_NOT_CONFIGURED",
		}
	}

	// Check if already running (any active phase state)
	status := pipeline.GetStatus()
	if status.Status == PipelineStateEnumerating ||
		status.Status == PipelineStateResolving ||
		status.Status == PipelineStateScanning ||
		status.Status == PipelineStateAssessing {
		return &Error{
			Phase:   "start",
			Message: "discovery pipeline already running",
			Code:    "PIPELINE_ALREADY_RUNNING",
		}
	}

	if trigger == "" {
		trigger = "service"
	}

	// Start pipeline in background
	run, err := pipeline.Start(ctx, trigger)
	if err != nil {
		return err
	}

	logging.GetLogger().InfoContext(ctx, "Discovery pipeline started from service",
		"run_id", run.ID,
		"trigger", trigger,
		"phases", len(run.PhaseDurations))

	return nil
}

// GetPipelineStatus returns the current pipeline status.
// Returns nil if pipeline is not configured.
func (s *Service) GetPipelineStatus() *PipelineRun {
	s.mu.RLock()
	pipeline := s.pipeline
	s.mu.RUnlock()

	if pipeline == nil {
		return nil
	}

	return pipeline.GetStatus()
}

// IsPipelineRunning returns true if the discovery pipeline is currently running.
func (s *Service) IsPipelineRunning() bool {
	s.mu.RLock()
	pipeline := s.pipeline
	s.mu.RUnlock()

	if pipeline == nil {
		return false
	}

	status := pipeline.GetStatus()
	return status.Status == PipelineStateEnumerating ||
		status.Status == PipelineStateResolving ||
		status.Status == PipelineStateScanning ||
		status.Status == PipelineStateAssessing
}

// syncPipelineResults updates the Service's device cache with pipeline results.
// This is called when the pipeline completes successfully.
// It ensures Service.GetDevices() returns the fully-enriched device list.
func (s *Service) syncPipelineResults(devices []*DiscoveredDevice) {
	if len(devices) == 0 {
		return
	}

	var callback func(devices []*DiscoveredDevice)

	s.mu.Lock()

	// Update metrics with pipeline results
	if s.metrics != nil {
		s.metrics.UpdateFromDevices(devices)
	}

	// Compute delta for change tracking
	if s.previousScan != nil {
		s.lastDelta = ComputeDelta(s.previousScan, devices)
		s.lastDeltaTime = time.Now()

		if len(s.lastDelta.NewDevices) > 0 || len(s.lastDelta.RemovedDevices) > 0 {
			logging.GetLogger().Info("Pipeline completed - synced devices",
				"total", len(devices),
				"new", len(s.lastDelta.NewDevices),
				"updated", len(s.lastDelta.UpdatedDevices),
				"removed", len(s.lastDelta.RemovedDevices))
		}
	}

	// Store copy for next delta computation
	s.previousScan = make([]*DiscoveredDevice, len(devices))
	copy(s.previousScan, devices)

	// Capture callback under lock
	callback = s.onPipelineComplete

	s.mu.Unlock()

	// Call user callback outside of lock to prevent deadlock
	if callback != nil {
		callback(devices)
	}
}

// Error represents a discovery-specific error with categorization.
type Error struct {
	Phase   string // Which phase/operation failed
	Message string // Human-readable message
	Code    string // Machine-readable error code
	Cause   error  // Underlying error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}
