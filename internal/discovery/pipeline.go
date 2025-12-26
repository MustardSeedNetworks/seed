// Package discovery implements multi-protocol network device discovery.
// This file implements the phased discovery pipeline that orchestrates
// sequential execution of discovery phases: enumeration, resolution,
// service discovery, and vulnerability assessment.
package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PipelineState represents the current state of the discovery pipeline.
type PipelineState string

// Pipeline state constants.
const (
	PipelineStateIdle        PipelineState = "idle"
	PipelineStateEnumerating PipelineState = "enumerating"
	PipelineStateResolving   PipelineState = "resolving"
	PipelineStateScanning    PipelineState = "scanning"
	PipelineStateAssessing   PipelineState = "assessing"
	PipelineStateComplete    PipelineState = "complete"
	PipelineStateFailed      PipelineState = "failed"
	PipelineStateCanceled    PipelineState = "canceled"
)

// PipelineEventType defines WebSocket event types for pipeline updates.
type PipelineEventType string

// Pipeline event type constants for WebSocket broadcasting.
const (
	EventPipelineStarted   PipelineEventType = "pipeline_started"
	EventPhaseStarted      PipelineEventType = "phase_started"
	EventPhaseProgress     PipelineEventType = "phase_progress"
	EventPhaseCompleted    PipelineEventType = "phase_completed"
	EventPhaseFailed       PipelineEventType = "phase_failed"
	EventDeviceDiscovered  PipelineEventType = "device_discovered"
	EventDeviceUpdated     PipelineEventType = "device_updated"
	EventPipelineCompleted PipelineEventType = "pipeline_completed"
	EventPipelineFailed    PipelineEventType = "pipeline_failed"
	EventPipelineCanceled  PipelineEventType = "pipeline_canceled"
)

// PipelineEvent is the WebSocket message for pipeline updates.
type PipelineEvent struct {
	Type      PipelineEventType `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	RunID     string            `json:"runId"`
	Payload   any               `json:"payload"`
}

// PipelineStartedPayload for EventPipelineStarted.
type PipelineStartedPayload struct {
	TotalPhases int      `json:"totalPhases"`
	Phases      []string `json:"phases"`
}

// PhaseStartedPayload for EventPhaseStarted.
type PhaseStartedPayload struct {
	Phase       string `json:"phase"`
	PhaseNumber int    `json:"phaseNumber"`
	TotalPhases int    `json:"totalPhases"`
	DeviceCount int    `json:"deviceCount"`
}

// PhaseProgressPayload for EventPhaseProgress.
type PhaseProgressPayload struct {
	Phase             string  `json:"phase"`
	ProcessedCount    int     `json:"processedCount"`
	TotalCount        int     `json:"totalCount"`
	PercentComplete   float64 `json:"percentComplete"`
	CurrentTarget     string  `json:"currentTarget,omitempty"`
	ElapsedMs         int64   `json:"elapsedMs"`
	EstimatedRemainMs int64   `json:"estimatedRemainMs,omitempty"`
}

// PhaseCompletedPayload for EventPhaseCompleted.
type PhaseCompletedPayload struct {
	Phase         string        `json:"phase"`
	DevicesFound  int           `json:"devicesFound,omitempty"`
	NamesResolved int           `json:"namesResolved,omitempty"`
	PortsOpen     int           `json:"portsOpen,omitempty"`
	VulnsFound    int           `json:"vulnsFound,omitempty"`
	Duration      time.Duration `json:"duration"`
	Errors        []string      `json:"errors,omitempty"`
}

// DeviceDiscoveredPayload for EventDeviceDiscovered.
type DeviceDiscoveredPayload struct {
	IP      string   `json:"ip"`
	MAC     string   `json:"mac,omitempty"`
	Vendor  string   `json:"vendor,omitempty"`
	Methods []string `json:"methods"`
	IsNew   bool     `json:"isNew"`
}

// PipelineCompletedPayload for EventPipelineCompleted.
type PipelineCompletedPayload struct {
	TotalDevices   int                      `json:"totalDevices"`
	NewDevices     int                      `json:"newDevices"`
	UpdatedDevices int                      `json:"updatedDevices"`
	StaleDevices   int                      `json:"staleDevices"`
	TotalDuration  time.Duration            `json:"totalDuration"`
	PhaseDurations map[string]time.Duration `json:"phaseDurations"`
}

// PortScanIntensity defines port scanning levels.
type PortScanIntensity string

// Port scan intensity level constants.
const (
	PortScanOff           PortScanIntensity = "off"
	PortScanQuick         PortScanIntensity = "quick"
	PortScanStandard      PortScanIntensity = "standard"
	PortScanComprehensive PortScanIntensity = "comprehensive"
	PortScanCustom        PortScanIntensity = "custom"
)

// ScanTimingProfile defines pre-configured timing settings.
type ScanTimingProfile string

// Scan timing profile constants for IDS-aware scanning.
const (
	ScanProfilePolite     ScanTimingProfile = "polite"
	ScanProfileNormal     ScanTimingProfile = "normal"
	ScanProfileAggressive ScanTimingProfile = "aggressive"
)

// Port lists for different intensity levels.
var (
	// QuickPorts - minimal set for device type identification (6 ports).
	QuickPorts = []int{22, 23, 80, 443, 8080, 8443}

	// StandardPorts - common enterprise services (~50 ports).
	StandardPorts = []int{
		// Remote access
		22, 23, 3389, 5900,
		// Web
		80, 443, 8080, 8443, 8000, 8888,
		// Database
		1433, 1521, 3306, 5432, 27017,
		// Network management
		161, 162, 830,
		// File sharing
		21, 445, 139, 2049,
		// Email
		25, 110, 143, 465, 587, 993, 995,
		// Directory
		389, 636, 88,
		// Printers
		515, 631, 9100,
		// VoIP
		5060, 5061,
		// Monitoring
		10050, 10051, 5666,
	}

	// ComprehensivePorts - top 1000 most common ports (abbreviated here, full list in implementation).
	ComprehensivePorts = generateComprehensivePorts()
)

// ScanTimingPresets maps profiles to timing configurations.
var ScanTimingPresets = map[ScanTimingProfile]PipelineTiming{
	ScanProfilePolite: {
		ProbeDelay:         200 * time.Millisecond,
		HostDelay:          100 * time.Millisecond,
		MaxConcurrentHosts: 5,
		PhaseTimeout:       30 * time.Minute,
	},
	ScanProfileNormal: {
		ProbeDelay:         50 * time.Millisecond,
		HostDelay:          20 * time.Millisecond,
		MaxConcurrentHosts: 20,
		PhaseTimeout:       10 * time.Minute,
	},
	ScanProfileAggressive: {
		ProbeDelay:         10 * time.Millisecond,
		HostDelay:          5 * time.Millisecond,
		MaxConcurrentHosts: 100,
		PhaseTimeout:       5 * time.Minute,
	},
}

// PipelineConfig controls the sequential discovery pipeline.
type PipelineConfig struct {
	// Phases controls which pipeline phases are enabled.
	Phases PipelinePhaseConfig `yaml:"phases" json:"phases"`

	// Timing controls rate limiting and delays.
	Timing PipelineTiming `yaml:"timing" json:"timing"`

	// PortScan controls port scanning behavior and intensity.
	PortScan PipelinePortScanConfig `yaml:"port_scan" json:"portScan"`

	// SNMPCollection controls extended SNMP MIB collection.
	SNMPCollection SNMPCollectionConfig `yaml:"snmp_collection" json:"snmpCollection"`

	// Persistence controls how results are stored.
	Persistence PipelinePersistenceConfig `yaml:"persistence" json:"persistence"`
}

// PipelinePhaseConfig controls which phases are executed.
type PipelinePhaseConfig struct {
	Enumeration      bool `yaml:"enumeration" json:"enumeration"`            // Always true - core functionality
	NameResolution   bool `yaml:"name_resolution" json:"nameResolution"`     // Default: true
	ServiceDiscovery bool `yaml:"service_discovery" json:"serviceDiscovery"` // Default: false (passive only)
	VulnAssessment   bool `yaml:"vuln_assessment" json:"vulnAssessment"`     // Default: false
}

// PipelineTiming controls scan rate limiting.
type PipelineTiming struct {
	// ProbeDelay is the minimum time between probes to a single host.
	ProbeDelay time.Duration `yaml:"probe_delay" json:"probeDelay"`

	// HostDelay is the minimum time between starting scans of different hosts.
	HostDelay time.Duration `yaml:"host_delay" json:"hostDelay"`

	// MaxConcurrentHosts limits parallel host scanning.
	MaxConcurrentHosts int `yaml:"max_concurrent_hosts" json:"maxConcurrentHosts"`

	// PhaseTimeout is the max duration for any single phase.
	PhaseTimeout time.Duration `yaml:"phase_timeout" json:"phaseTimeout"`

	// Profile selects a pre-defined timing profile (overrides individual settings).
	Profile ScanTimingProfile `yaml:"profile" json:"profile"`
}

// PipelinePortScanConfig controls port scanning intensity.
type PipelinePortScanConfig struct {
	// Intensity controls which ports are scanned.
	Intensity PortScanIntensity `yaml:"intensity" json:"intensity"`

	// CustomPorts for Intensity="custom".
	CustomPorts []int `yaml:"custom_ports,omitempty" json:"customPorts,omitempty"`

	// BannerGrab enables service banner reading.
	BannerGrab bool `yaml:"banner_grab" json:"bannerGrab"`

	// ConnectTimeout for port connections.
	ConnectTimeout time.Duration `yaml:"connect_timeout" json:"connectTimeout"`
}

// SNMPCollectionConfig controls extended SNMP data collection.
type SNMPCollectionConfig struct {
	// Enabled turns on extended SNMP collection in Phase 3.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MIBs specifies which MIB groups to collect.
	MIBs SNMPMIBSelection `yaml:"mibs" json:"mibs"`

	// WalkTimeout per MIB walk operation.
	WalkTimeout time.Duration `yaml:"walk_timeout" json:"walkTimeout"`

	// MaxOIDsPerRequest for bulk requests.
	MaxOIDsPerRequest int `yaml:"max_oids_per_request" json:"maxOidsPerRequest"`
}

// SNMPMIBSelection controls which MIBs are collected.
type SNMPMIBSelection struct {
	System      bool `yaml:"system" json:"system"`            // SNMPv2-MIB::system (always on)
	Interfaces  bool `yaml:"interfaces" json:"interfaces"`    // IF-MIB (ifTable, ifXTable)
	IPAddresses bool `yaml:"ip_addresses" json:"ipAddresses"` // IP-MIB (ipAddrTable)
	Routing     bool `yaml:"routing" json:"routing"`          // IP-FORWARD-MIB
	Bridge      bool `yaml:"bridge" json:"bridge"`            // BRIDGE-MIB (MAC table)
	Entity      bool `yaml:"entity" json:"entity"`            // ENTITY-MIB (physical inventory)
	LLDP        bool `yaml:"lldp" json:"lldp"`                // LLDP-MIB
	VLAN        bool `yaml:"vlan" json:"vlan"`                // Q-BRIDGE-MIB
}

// PipelinePersistenceConfig controls database storage.
type PipelinePersistenceConfig struct {
	// StoreHistory keeps historical device state.
	StoreHistory bool `yaml:"store_history" json:"storeHistory"`

	// StalenessThreshold marks devices inactive after this duration.
	StalenessThreshold time.Duration `yaml:"staleness_threshold" json:"stalenessThreshold"`

	// PurgeAfter removes inactive devices after this duration.
	PurgeAfter time.Duration `yaml:"purge_after" json:"purgeAfter"`
}

// DefaultPipelineConfig returns default pipeline configuration.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Phases: PipelinePhaseConfig{
			Enumeration:      true,  // Always enabled
			NameResolution:   true,  // Enabled by default
			ServiceDiscovery: false, // Disabled by default (requires opt-in)
			VulnAssessment:   false, // Disabled by default (requires opt-in)
		},
		Timing: PipelineTiming{
			ProbeDelay:         50 * time.Millisecond,
			HostDelay:          20 * time.Millisecond,
			MaxConcurrentHosts: 20,
			PhaseTimeout:       10 * time.Minute,
			Profile:            ScanProfileNormal,
		},
		PortScan: PipelinePortScanConfig{
			Intensity:      PortScanOff, // OFF by default - security conscious
			BannerGrab:     true,
			ConnectTimeout: 2 * time.Second,
		},
		SNMPCollection: SNMPCollectionConfig{
			Enabled: true,
			MIBs: SNMPMIBSelection{
				System:      true, // Always collect system info
				Interfaces:  true, // Critical for network devices
				IPAddresses: true, // Essential for topology
				Routing:     false,
				Bridge:      false, // Can be large on switches
				Entity:      false,
				LLDP:        true,
				VLAN:        false, // Can be large
			},
			WalkTimeout:       30 * time.Second,
			MaxOIDsPerRequest: 10,
		},
		Persistence: PipelinePersistenceConfig{
			StoreHistory:       true,
			StalenessThreshold: 24 * time.Hour,
			PurgeAfter:         30 * 24 * time.Hour,
		},
	}
}

// PipelineRun represents a single execution of the discovery pipeline.
type PipelineRun struct {
	ID             string                   `json:"id"`
	StartedAt      time.Time                `json:"startedAt"`
	CompletedAt    *time.Time               `json:"completedAt,omitempty"`
	Status         PipelineState            `json:"status"`
	Trigger        string                   `json:"trigger"` // manual, scheduled, startup, api
	Config         PipelineConfig           `json:"config"`
	CurrentPhase   string                   `json:"currentPhase,omitempty"`
	PhaseDurations map[string]time.Duration `json:"phaseDurations,omitempty"`
	DevicesFound   int                      `json:"devicesFound"`
	Errors         []string                 `json:"errors,omitempty"`
}

// Phase represents a single phase in the discovery pipeline.
type Phase interface {
	// Name returns the phase name.
	Name() string

	// Run executes the phase with the given devices from the previous phase.
	// Returns updated devices and any errors encountered.
	Run(ctx context.Context, devices []*DiscoveredDevice, progressCh chan<- PhaseProgressPayload) ([]*DiscoveredDevice, error)
}

// EventBroadcaster is an interface for broadcasting pipeline events.
type EventBroadcaster interface {
	BroadcastPipelineEvent(event PipelineEvent)
}

// Pipeline orchestrates the sequential execution of discovery phases.
type Pipeline struct {
	mu sync.RWMutex

	// Configuration
	config PipelineConfig

	// State
	currentRun *PipelineRun
	cancelFunc context.CancelFunc

	// Dependencies
	deviceDiscovery *DeviceDiscovery
	profiler        *DeviceProfiler
	broadcaster     EventBroadcaster
}

// NewPipeline creates a new discovery pipeline.
func NewPipeline(config *PipelineConfig, deviceDiscovery *DeviceDiscovery, profiler *DeviceProfiler, broadcaster EventBroadcaster) *Pipeline {
	p := &Pipeline{
		config:          *config,
		deviceDiscovery: deviceDiscovery,
		profiler:        profiler,
		broadcaster:     broadcaster,
	}

	// Apply timing profile if set
	if config.Timing.Profile != "" {
		if preset, ok := ScanTimingPresets[config.Timing.Profile]; ok {
			profileName := config.Timing.Profile // Save profile name before overwriting
			p.config.Timing = preset
			p.config.Timing.Profile = profileName // Restore profile name
		}
	}

	return p
}

// Start begins a new pipeline run.
func (p *Pipeline) Start(ctx context.Context, trigger string) (*PipelineRun, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentRun != nil && p.currentRun.Status == PipelineStateEnumerating ||
		p.currentRun != nil && p.currentRun.Status == PipelineStateResolving ||
		p.currentRun != nil && p.currentRun.Status == PipelineStateScanning ||
		p.currentRun != nil && p.currentRun.Status == PipelineStateAssessing {
		return nil, fmt.Errorf("pipeline already running")
	}

	// Create new run
	runID := uuid.New().String()[:8]
	p.currentRun = &PipelineRun{
		ID:             runID,
		StartedAt:      time.Now(),
		Status:         PipelineStateEnumerating,
		Trigger:        trigger,
		Config:         p.config,
		PhaseDurations: make(map[string]time.Duration),
	}

	// Create cancellable context
	runCtx, cancel := context.WithCancel(ctx)
	p.cancelFunc = cancel

	// Broadcast pipeline started
	p.broadcastEvent(EventPipelineStarted, PipelineStartedPayload{
		TotalPhases: p.countEnabledPhases(),
		Phases:      p.GetEnabledPhaseNames(),
	})

	// Run pipeline in background
	go p.run(runCtx)

	return p.currentRun, nil
}

// Cancel cancels the current pipeline run.
func (p *Pipeline) Cancel() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentRun == nil || p.cancelFunc == nil {
		return fmt.Errorf("no pipeline running")
	}

	p.cancelFunc()
	p.currentRun.Status = PipelineStateCanceled
	now := time.Now()
	p.currentRun.CompletedAt = &now

	p.broadcastEvent(EventPipelineCanceled, nil)

	return nil
}

// GetStatus returns the current pipeline status.
func (p *Pipeline) GetStatus() *PipelineRun {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.currentRun == nil {
		return &PipelineRun{
			Status: PipelineStateIdle,
		}
	}

	// Return a deep copy including the map to prevent external mutation (fixes #845)
	run := *p.currentRun
	if p.currentRun.PhaseDurations != nil {
		run.PhaseDurations = make(map[string]time.Duration, len(p.currentRun.PhaseDurations))
		for k, v := range p.currentRun.PhaseDurations {
			run.PhaseDurations[k] = v
		}
	}
	if p.currentRun.Errors != nil {
		run.Errors = make([]string, len(p.currentRun.Errors))
		copy(run.Errors, p.currentRun.Errors)
	}
	return &run
}

// GetConfig returns the current pipeline configuration.
func (p *Pipeline) GetConfig() PipelineConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// UpdateConfig updates the pipeline configuration.
func (p *Pipeline) UpdateConfig(config *PipelineConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Don't allow config changes while running
	if p.currentRun != nil && (p.currentRun.Status == PipelineStateEnumerating ||
		p.currentRun.Status == PipelineStateResolving ||
		p.currentRun.Status == PipelineStateScanning ||
		p.currentRun.Status == PipelineStateAssessing) {
		return fmt.Errorf("cannot update config while pipeline is running")
	}

	// Apply timing profile if set
	if config.Timing.Profile != "" {
		if preset, ok := ScanTimingPresets[config.Timing.Profile]; ok {
			profileName := config.Timing.Profile // Save profile name before overwriting
			config.Timing = preset
			config.Timing.Profile = profileName // Restore profile name
		}
	}

	p.config = *config
	return nil
}

// run executes the pipeline phases sequentially.
func (p *Pipeline) run(ctx context.Context) {
	var devices []*DiscoveredDevice
	var err error
	phaseNumber := 0

	defer func() {
		p.mu.Lock()
		now := time.Now()
		p.currentRun.CompletedAt = &now
		if p.currentRun.Status != PipelineStateCanceled && p.currentRun.Status != PipelineStateFailed {
			p.currentRun.Status = PipelineStateComplete
		}
		// Copy values under lock to avoid race after unlock (fixes #824)
		status := p.currentRun.Status
		startedAt := p.currentRun.StartedAt
		phaseDurations := make(map[string]time.Duration, len(p.currentRun.PhaseDurations))
		for k, v := range p.currentRun.PhaseDurations {
			phaseDurations[k] = v
		}
		p.mu.Unlock()

		// Broadcast completion using copied values
		if status == PipelineStateComplete {
			p.broadcastEvent(EventPipelineCompleted, PipelineCompletedPayload{
				TotalDevices:   len(devices),
				TotalDuration:  time.Since(startedAt),
				PhaseDurations: phaseDurations,
			})
		}
	}()

	// Phase 1: Enumeration (always runs)
	phaseNumber++
	p.updateState(PipelineStateEnumerating, "enumeration")
	devices, err = p.runEnumerationPhase(ctx, phaseNumber)
	if err != nil {
		p.handlePhaseError("enumeration", err)
		return
	}

	// Phase 2: Name Resolution (if enabled)
	if p.config.Phases.NameResolution {
		phaseNumber++
		p.updateState(PipelineStateResolving, "resolution")
		devices, err = p.runResolutionPhase(ctx, devices, phaseNumber)
		if err != nil {
			p.handlePhaseError("resolution", err)
			return
		}
	}

	// Phase 3: Service Discovery (if enabled)
	if p.config.Phases.ServiceDiscovery {
		phaseNumber++
		p.updateState(PipelineStateScanning, "scanning")
		devices, err = p.runScanningPhase(ctx, devices, phaseNumber)
		if err != nil {
			p.handlePhaseError("scanning", err)
			return
		}
	}

	// Phase 4: Vulnerability Assessment (if enabled)
	if p.config.Phases.VulnAssessment {
		phaseNumber++
		p.updateState(PipelineStateAssessing, "assessment")
		devices, err = p.runAssessmentPhase(ctx, devices, phaseNumber)
		if err != nil {
			p.handlePhaseError("assessment", err)
			return
		}
	}

	p.mu.Lock()
	p.currentRun.DevicesFound = len(devices)
	// Copy values under lock to avoid race after unlock (fixes #827)
	runID := p.currentRun.ID
	startedAt := p.currentRun.StartedAt
	p.mu.Unlock()

	slog.Info("Pipeline completed",
		"runId", runID,
		"devicesFound", len(devices),
		"duration", time.Since(startedAt))
}

// runEnumerationPhase executes the device enumeration phase.
func (p *Pipeline) runEnumerationPhase(ctx context.Context, phaseNumber int) ([]*DiscoveredDevice, error) {
	start := time.Now()
	totalPhases := p.countEnabledPhases()

	p.broadcastEvent(EventPhaseStarted, PhaseStartedPayload{
		Phase:       "enumeration",
		PhaseNumber: phaseNumber,
		TotalPhases: totalPhases,
		DeviceCount: 0,
	})

	// Use existing device discovery
	if err := p.deviceDiscovery.Scan(ctx); err != nil {
		return nil, fmt.Errorf("enumeration failed: %w", err)
	}

	devices := p.deviceDiscovery.GetDevices()

	duration := time.Since(start)
	p.mu.Lock()
	p.currentRun.PhaseDurations["enumeration"] = duration
	p.mu.Unlock()

	p.broadcastEvent(EventPhaseCompleted, PhaseCompletedPayload{
		Phase:        "enumeration",
		DevicesFound: len(devices),
		Duration:     duration,
	})

	slog.Info("Enumeration phase completed",
		"devices", len(devices),
		"duration", duration)

	return devices, nil
}

// runResolutionPhase executes the name resolution phase.
//
//nolint:unparam // Error return kept for interface consistency with other phase methods.
func (p *Pipeline) runResolutionPhase(_ context.Context, devices []*DiscoveredDevice, phaseNumber int) ([]*DiscoveredDevice, error) {
	start := time.Now()
	totalPhases := p.countEnabledPhases()

	p.broadcastEvent(EventPhaseStarted, PhaseStartedPayload{
		Phase:       "resolution",
		PhaseNumber: phaseNumber,
		TotalPhases: totalPhases,
		DeviceCount: len(devices),
	})

	// Resolution is handled by the existing device discovery
	// The DeviceDiscovery.Scan already triggers name resolution
	// This phase is a no-op but maintains the pipeline structure

	duration := time.Since(start)
	p.mu.Lock()
	p.currentRun.PhaseDurations["resolution"] = duration
	p.mu.Unlock()

	namesResolved := 0
	for _, d := range devices {
		if d.DisplayName != "" && d.DisplayName != d.IP {
			namesResolved++
		}
	}

	p.broadcastEvent(EventPhaseCompleted, PhaseCompletedPayload{
		Phase:         "resolution",
		NamesResolved: namesResolved,
		Duration:      duration,
	})

	slog.Info("Resolution phase completed",
		"namesResolved", namesResolved,
		"duration", duration)

	return devices, nil
}

// runScanningPhase executes the service discovery phase.
//
//nolint:gocyclo // Scanning phase requires polling loop with context cancellation and timeout handling.
func (p *Pipeline) runScanningPhase(ctx context.Context, devices []*DiscoveredDevice, phaseNumber int) ([]*DiscoveredDevice, error) {
	start := time.Now()
	totalPhases := p.countEnabledPhases()

	p.broadcastEvent(EventPhaseStarted, PhaseStartedPayload{
		Phase:       "scanning",
		PhaseNumber: phaseNumber,
		TotalPhases: totalPhases,
		DeviceCount: len(devices),
	})

	// Get ports based on intensity
	ports := p.getPortsForIntensity()

	if len(ports) == 0 {
		slog.Info("Port scanning disabled, skipping service discovery")
	} else {
		// Queue devices for profiling
		for _, device := range devices {
			if device.IP != "" {
				p.profiler.QueueProfile(device.IP)
			}
		}

		// Wait for profiling to complete (with timeout)
		timeout := p.config.Timing.PhaseTimeout
		if timeout == 0 {
			timeout = 10 * time.Minute
		}

		waitCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Poll for completion
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

	waitLoop:
		for {
			select {
			case <-waitCtx.Done():
				break waitLoop
			case <-ticker.C:
				// Check if all devices are profiled
				allDone := true
				for _, device := range devices {
					if device.IP != "" && p.profiler.IsProfiling(device.IP) {
						allDone = false
						break
					}
				}
				if allDone {
					break waitLoop
				}

				// Broadcast progress
				processed := 0
				for _, device := range devices {
					if device.IP != "" && p.profiler.GetProfile(device.IP) != nil {
						processed++
					}
				}
				// Prevent division by zero (fixes #821)
				percentComplete := float64(100)
				if len(devices) > 0 {
					percentComplete = float64(processed) / float64(len(devices)) * 100
				}
				p.broadcastEvent(EventPhaseProgress, PhaseProgressPayload{
					Phase:           "scanning",
					ProcessedCount:  processed,
					TotalCount:      len(devices),
					PercentComplete: percentComplete,
					ElapsedMs:       time.Since(start).Milliseconds(),
				})
			}
		}
	}

	// Attach profiles to devices
	openPorts := 0
	for _, device := range devices {
		if device.IP != "" {
			if profile := p.profiler.GetProfile(device.IP); profile != nil {
				device.Profile = profile
				openPorts += len(profile.OpenPorts)
			}
		}
	}

	duration := time.Since(start)
	p.mu.Lock()
	p.currentRun.PhaseDurations["scanning"] = duration
	p.mu.Unlock()

	p.broadcastEvent(EventPhaseCompleted, PhaseCompletedPayload{
		Phase:     "scanning",
		PortsOpen: openPorts,
		Duration:  duration,
	})

	slog.Info("Scanning phase completed",
		"openPorts", openPorts,
		"duration", duration)

	return devices, nil
}

// runAssessmentPhase executes the vulnerability assessment phase.
//
//nolint:unparam // Error return kept for interface consistency with other phase methods.
func (p *Pipeline) runAssessmentPhase(_ context.Context, devices []*DiscoveredDevice, phaseNumber int) ([]*DiscoveredDevice, error) {
	start := time.Now()
	totalPhases := p.countEnabledPhases()

	p.broadcastEvent(EventPhaseStarted, PhaseStartedPayload{
		Phase:       "assessment",
		PhaseNumber: phaseNumber,
		TotalPhases: totalPhases,
		DeviceCount: len(devices),
	})

	// Vulnerability assessment will be implemented in phase_assessment.go
	// For now, this is a placeholder

	duration := time.Since(start)
	p.mu.Lock()
	p.currentRun.PhaseDurations["assessment"] = duration
	p.mu.Unlock()

	p.broadcastEvent(EventPhaseCompleted, PhaseCompletedPayload{
		Phase:      "assessment",
		VulnsFound: 0, // Will be populated by actual implementation
		Duration:   duration,
	})

	slog.Info("Assessment phase completed",
		"duration", duration)

	return devices, nil
}

// updateState updates the current run state.
func (p *Pipeline) updateState(state PipelineState, phase string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentRun.Status = state
	p.currentRun.CurrentPhase = phase
}

// handlePhaseError handles errors during phase execution.
func (p *Pipeline) handlePhaseError(phase string, err error) {
	slog.Error("Pipeline phase failed",
		"phase", phase,
		"error", err)

	p.mu.Lock()
	p.currentRun.Status = PipelineStateFailed
	p.currentRun.Errors = append(p.currentRun.Errors, fmt.Sprintf("%s: %v", phase, err))
	now := time.Now()
	p.currentRun.CompletedAt = &now
	p.mu.Unlock()

	p.broadcastEvent(EventPhaseFailed, PhaseCompletedPayload{
		Phase:  phase,
		Errors: []string{err.Error()},
	})

	p.broadcastEvent(EventPipelineFailed, map[string]string{
		"phase": phase,
		"error": err.Error(),
	})
}

// broadcastEvent sends an event to the broadcaster if available.
// Fixes #862: Capture broadcaster reference under lock to prevent TOCTOU race.
func (p *Pipeline) broadcastEvent(eventType PipelineEventType, payload any) {
	p.mu.RLock()
	broadcaster := p.broadcaster
	var runID string
	if p.currentRun != nil {
		runID = p.currentRun.ID
	}
	p.mu.RUnlock()

	// Check broadcaster after releasing lock - it was captured while lock was held
	if broadcaster == nil {
		return
	}

	broadcaster.BroadcastPipelineEvent(PipelineEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		RunID:     runID,
		Payload:   payload,
	})
}

// countEnabledPhases returns the number of enabled phases.
func (p *Pipeline) countEnabledPhases() int {
	count := 1 // Enumeration always runs
	if p.config.Phases.NameResolution {
		count++
	}
	if p.config.Phases.ServiceDiscovery {
		count++
	}
	if p.config.Phases.VulnAssessment {
		count++
	}
	return count
}

// GetEnabledPhaseNames returns the names of enabled phases.
func (p *Pipeline) GetEnabledPhaseNames() []string {
	phases := []string{"enumeration"}
	if p.config.Phases.NameResolution {
		phases = append(phases, "resolution")
	}
	if p.config.Phases.ServiceDiscovery {
		phases = append(phases, "scanning")
	}
	if p.config.Phases.VulnAssessment {
		phases = append(phases, "assessment")
	}
	return phases
}

// getPortsForIntensity returns the port list based on configured intensity.
func (p *Pipeline) getPortsForIntensity() []int {
	switch p.config.PortScan.Intensity {
	case PortScanOff:
		return nil
	case PortScanQuick:
		return QuickPorts
	case PortScanStandard:
		return StandardPorts
	case PortScanComprehensive:
		return ComprehensivePorts
	case PortScanCustom:
		return p.config.PortScan.CustomPorts
	default:
		return nil
	}
}

// ConfigPipelineAdapter is an interface for adapting configuration to PipelineConfig.
// Implemented by config.PipelineConfig.
type ConfigPipelineAdapter interface {
	GetPhases() (enumeration, nameResolution, serviceDiscovery, vulnAssessment bool)
	GetTiming() (probeDelay, hostDelay, phaseTimeout time.Duration, maxConcurrentHosts int, profile string)
	GetPortScan() (intensity string, customPorts []int, bannerGrab bool, connectTimeout time.Duration)
	GetSNMP() (enabled, system, interfaces, ipAddresses, routing, bridge, entity, lldp, vlan bool, walkTimeout time.Duration, maxOIDsPerRequest int)
	GetPersistence() (storeHistory bool, stalenessThreshold, purgeAfter time.Duration)
}

// PipelineConfigFromAdapter converts a ConfigPipelineAdapter to discovery.PipelineConfig.
func PipelineConfigFromAdapter(cfg ConfigPipelineAdapter) PipelineConfig {
	enumeration, nameRes, svcDisc, vulnAssess := cfg.GetPhases()
	probeDelay, hostDelay, phaseTimeout, maxConcurrent, profile := cfg.GetTiming()
	intensity, customPorts, bannerGrab, connectTimeout := cfg.GetPortScan()
	snmpEnabled, system, ifaces, ipAddrs, routing, bridge, entity, lldp, vlan, walkTimeout, maxOIDs := cfg.GetSNMP()
	storeHistory, staleness, purge := cfg.GetPersistence()

	return PipelineConfig{
		Phases: PipelinePhaseConfig{
			Enumeration:      enumeration,
			NameResolution:   nameRes,
			ServiceDiscovery: svcDisc,
			VulnAssessment:   vulnAssess,
		},
		Timing: PipelineTiming{
			ProbeDelay:         probeDelay,
			HostDelay:          hostDelay,
			PhaseTimeout:       phaseTimeout,
			MaxConcurrentHosts: maxConcurrent,
			Profile:            ScanTimingProfile(profile),
		},
		PortScan: PipelinePortScanConfig{
			Intensity:      PortScanIntensity(intensity),
			CustomPorts:    customPorts,
			BannerGrab:     bannerGrab,
			ConnectTimeout: connectTimeout,
		},
		SNMPCollection: SNMPCollectionConfig{
			Enabled: snmpEnabled,
			MIBs: SNMPMIBSelection{
				System:      system,
				Interfaces:  ifaces,
				IPAddresses: ipAddrs,
				Routing:     routing,
				Bridge:      bridge,
				Entity:      entity,
				LLDP:        lldp,
				VLAN:        vlan,
			},
			WalkTimeout:       walkTimeout,
			MaxOIDsPerRequest: maxOIDs,
		},
		Persistence: PipelinePersistenceConfig{
			StoreHistory:       storeHistory,
			StalenessThreshold: staleness,
			PurgeAfter:         purge,
		},
	}
}

// generateComprehensivePorts generates the top 1000 most common ports.
func generateComprehensivePorts() []int {
	// Top 100 most common ports - can be expanded
	return []int{
		1, 3, 7, 9, 13, 17, 19, 20, 21, 22, 23, 25, 26, 37, 53, 79, 80, 81, 82, 83,
		84, 85, 88, 89, 90, 99, 100, 106, 109, 110, 111, 113, 119, 125, 135, 139,
		143, 144, 146, 161, 162, 163, 179, 199, 211, 212, 222, 254, 255, 256, 259,
		264, 280, 301, 306, 311, 340, 366, 389, 406, 407, 416, 417, 425, 427, 443,
		444, 445, 458, 464, 465, 481, 497, 500, 512, 513, 514, 515, 524, 541, 543,
		544, 545, 548, 554, 555, 563, 587, 593, 616, 617, 625, 631, 636, 646, 648,
		666, 667, 668, 683, 687, 691, 700, 705, 711, 714, 720, 722, 726, 749, 765,
		777, 783, 787, 800, 801, 808, 843, 873, 880, 888, 898, 900, 901, 902, 903,
		911, 912, 981, 987, 990, 992, 993, 995, 999, 1000, 1001, 1002, 1007, 1009,
		1010, 1011, 1021, 1022, 1023, 1024, 1025, 1026, 1027, 1028, 1029, 1030,
		1031, 1032, 1033, 1034, 1035, 1036, 1037, 1038, 1039, 1040, 1041, 1042,
		1043, 1044, 1045, 1046, 1047, 1048, 1049, 1050, 1051, 1052, 1053, 1054,
		1055, 1056, 1057, 1058, 1059, 1060, 1061, 1062, 1063, 1064, 1065, 1066,
		1067, 1068, 1069, 1070, 1071, 1072, 1073, 1074, 1075, 1076, 1077, 1078,
		1079, 1080, 1081, 1082, 1083, 1084, 1085, 1086, 1087, 1088, 1089, 1090,
		1091, 1092, 1093, 1094, 1095, 1096, 1097, 1098, 1099, 1100, 1102, 1104,
		1105, 1106, 1107, 1108, 1110, 1111, 1112, 1113, 1114, 1117, 1119, 1121,
		1122, 1123, 1124, 1126, 1130, 1131, 1132, 1137, 1138, 1141, 1145, 1147,
		1148, 1149, 1151, 1152, 1154, 1163, 1164, 1165, 1166, 1169, 1174, 1175,
		1183, 1185, 1186, 1187, 1192, 1198, 1199, 1201, 1213, 1216, 1217, 1218,
		1233, 1234, 1236, 1244, 1247, 1248, 1259, 1271, 1272, 1277, 1287, 1296,
		1300, 1301, 1309, 1310, 1311, 1322, 1328, 1334, 1352, 1417, 1433, 1434,
		1443, 1455, 1461, 1494, 1500, 1501, 1503, 1521, 1524, 1533, 1556, 1580,
		1583, 1594, 1600, 1641, 1658, 1666, 1687, 1688, 1700, 1717, 1718, 1719,
		1720, 1721, 1723, 1755, 1761, 1782, 1783, 1801, 1805, 1812, 1839, 1840,
		1862, 1863, 1864, 1875, 1900, 1914, 1935, 1947, 1971, 1972, 1974, 1984,
		1998, 1999, 2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009,
		2010, 2013, 2020, 2021, 2022, 2030, 2033, 2034, 2035, 2038, 2040, 2041,
		2042, 2043, 2045, 2046, 2047, 2048, 2049, 2065, 2068, 2099, 2100, 2103,
		2105, 2106, 2107, 2111, 2119, 2121, 2126, 2135, 2144, 2160, 2161, 2170,
		2179, 2190, 2191, 2196, 2200, 2222, 2251, 2260, 2288, 2301, 2323, 2366,
		2381, 2382, 2383, 2393, 2394, 2399, 2401, 2492, 2500, 2522, 2525, 2557,
		2601, 2602, 2604, 2605, 2607, 2608, 2638, 2701, 2702, 2710, 2717, 2718,
		2725, 2800, 2809, 2811, 2869, 2875, 2909, 2910, 2920, 2967, 2968, 2998,
		3000, 3001, 3003, 3005, 3006, 3007, 3011, 3013, 3017, 3030, 3031, 3052,
		3071, 3077, 3128, 3168, 3211, 3221, 3260, 3261, 3268, 3269, 3283, 3300,
		3301, 3306, 3322, 3323, 3324, 3325, 3333, 3351, 3367, 3369, 3370, 3371,
		3372, 3389, 3390, 3404, 3476, 3493, 3517, 3527, 3546, 3551, 3580, 3659,
		3689, 3690, 3703, 3737, 3766, 3784, 3800, 3801, 3809, 3814, 3826, 3827,
		3828, 3851, 3869, 3871, 3878, 3880, 3889, 3905, 3914, 3918, 3920, 3945,
		3971, 3986, 3995, 3998, 4000, 4001, 4002, 4003, 4004, 4005, 4006, 4045,
		4111, 4125, 4126, 4129, 4224, 4242, 4279, 4321, 4343, 4443, 4444, 4445,
		4446, 4449, 4550, 4567, 4662, 4848, 4899, 4900, 4998, 5000, 5001, 5002,
		5003, 5004, 5009, 5030, 5033, 5050, 5051, 5054, 5060, 5061, 5080, 5087,
		5100, 5101, 5102, 5120, 5190, 5200, 5214, 5221, 5222, 5225, 5226, 5269,
		5280, 5298, 5357, 5405, 5414, 5431, 5432, 5440, 5500, 5510, 5544, 5550,
		5555, 5560, 5566, 5631, 5633, 5666, 5678, 5679, 5718, 5730, 5800, 5801,
		5802, 5810, 5811, 5815, 5822, 5825, 5850, 5859, 5862, 5877, 5900, 5901,
		5902, 5903, 5904, 5906, 5907, 5910, 5911, 5915, 5922, 5925, 5950, 5952,
		5959, 5960, 5961, 5962, 5963, 5987, 5988, 5989, 5998, 5999, 6000, 6001,
		6002, 6003, 6004, 6005, 6006, 6007, 6009, 6025, 6059, 6100, 6101, 6106,
		6112, 6123, 6129, 6156, 6346, 6389, 6502, 6510, 6543, 6547, 6565, 6566,
		6567, 6580, 6646, 6666, 6667, 6668, 6669, 6689, 6692, 6699, 6779, 6788,
		6789, 6792, 6839, 6881, 6901, 6969, 7000, 7001, 7002, 7004, 7007, 7019,
		7025, 7070, 7100, 7103, 7106, 7200, 7201, 7402, 7435, 7443, 7496, 7512,
		7625, 7627, 7676, 7741, 7777, 7778, 7800, 7911, 7920, 7921, 7937, 7938,
		7999, 8000, 8001, 8002, 8007, 8008, 8009, 8010, 8011, 8021, 8022, 8031,
		8042, 8045, 8080, 8081, 8082, 8083, 8084, 8085, 8086, 8087, 8088, 8089,
		8090, 8093, 8099, 8100, 8180, 8181, 8192, 8193, 8194, 8200, 8222, 8254,
		8290, 8291, 8292, 8300, 8333, 8383, 8400, 8402, 8443, 8500, 8600, 8649,
		8651, 8652, 8654, 8701, 8800, 8873, 8888, 8899, 8994, 9000, 9001, 9002,
		9003, 9009, 9010, 9011, 9040, 9050, 9071, 9080, 9081, 9090, 9091, 9099,
		9100, 9101, 9102, 9103, 9110, 9111, 9200, 9207, 9220, 9290, 9415, 9418,
		9485, 9500, 9502, 9503, 9535, 9575, 9593, 9594, 9595, 9618, 9666, 9876,
		9877, 9878, 9898, 9900, 9917, 9929, 9943, 9944, 9968, 9998, 9999, 10000,
		10001, 10002, 10003, 10004, 10009, 10010, 10012, 10024, 10025, 10082,
		10180, 10215, 10243, 10566, 10616, 10617, 10621, 10626, 10628, 10629,
		10778, 11110, 11111, 11967, 12000, 12174, 12265, 12345, 13456, 13722,
		13782, 13783, 14000, 14238, 14441, 14442, 15000, 15002, 15003, 15004,
		15660, 15742, 16000, 16001, 16012, 16016, 16018, 16080, 16113, 16992,
		16993, 17877, 17988, 18040, 18101, 18988, 19101, 19283, 19315, 19350,
		19780, 19801, 19842, 20000, 20005, 20031, 20221, 20222, 20828, 21571,
		22939, 23502, 24444, 24800, 25734, 25735, 26214, 27000, 27352, 27353,
		27355, 27356, 27715, 28201, 30000, 30718, 30951, 31038, 31337, 32768,
		32769, 32770, 32771, 32772, 32773, 32774, 32775, 32776, 32777, 32778,
		32779, 32780, 32781, 32782, 32783, 32784, 32785, 33354, 33899, 34571,
		34572, 34573, 35500, 38292, 40193, 40911, 41511, 42510, 44176, 44442,
		44443, 44501, 45100, 48080, 49152, 49153, 49154, 49155, 49156, 49157,
		49158, 49159, 49160, 49161, 49163, 49165, 49167, 49175, 49176, 49400,
		49999, 50000, 50001, 50002, 50003, 50006, 50300, 50389, 50500, 50636,
		50800, 51103, 51493, 52673, 52822, 52848, 52869, 54045, 54328, 55055,
		55056, 55555, 55600, 56737, 56738, 57294, 57797, 58080, 60020, 60443,
		61532, 61900, 62078, 63331, 64623, 64680, 65000, 65129, 65389,
	}
}
