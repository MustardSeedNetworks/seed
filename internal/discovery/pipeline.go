package discovery

// Package discovery implements the phased discovery pipeline that orchestrates
// sequential execution of discovery phases: enumeration, resolution, service
// discovery, and vulnerability assessment.
//
// pipeline.go holds the Pipeline struct, NewPipeline, the Start/Cancel/Get
// public API, the run() orchestrator, and the state/error/broadcast helpers.
// The types/enums/payloads, default config + port presets, per-phase runners,
// and helper functions each live in sibling pipeline_*.go files.

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/krisarmstrong/seed/internal/logging"
)

// maxPipelineErrors limits error accumulation to prevent unbounded memory growth (fixes #880).
const maxPipelineErrors = 100

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
	resolutionPhase *ResolutionPhase // Phase 2: Active name resolution

	// Callbacks
	onComplete func(devices []*DiscoveredDevice) // Called when pipeline completes successfully
}

// NewPipeline creates a new discovery pipeline.
func NewPipeline(
	config *PipelineConfig,
	deviceDiscovery *DeviceDiscovery,
	profiler *DeviceProfiler,
	broadcaster EventBroadcaster,
) *Pipeline {
	// Create resolution config from pipeline settings
	resConfig := &ResolutionConfig{
		DNS:     config.Resolution.DNS,
		NetBIOS: config.Resolution.NetBIOS,
		MDNS:    config.Resolution.MDNS,
		Timing: ResolutionTiming{
			DNSTimeout:           pipelineResDNSTimeoutMs * time.Millisecond,
			NetBIOSTimeout:       pipelineResDNSTimeoutMs * time.Millisecond,
			MDNSTimeout:          pipelineResMDNSTimeoutS * time.Second,
			PhaseTimeout:         config.Timing.PhaseTimeout,
			MaxConcurrentDNS:     pipelineResMaxConcDNS,
			MaxConcurrentNetBIOS: pipelineResMaxConcNetBIOS,
			MaxConcurrentMDNS:    pipelineResMaxConcMDNS,
		},
	}

	interfaceName := ""
	if deviceDiscovery != nil {
		interfaceName = deviceDiscovery.GetInterfaceName()
	}

	p := &Pipeline{
		config:          *config,
		deviceDiscovery: deviceDiscovery,
		profiler:        profiler,
		broadcaster:     broadcaster,
		resolutionPhase: NewResolutionPhase(interfaceName, resConfig, broadcaster),
	}

	// Apply timing profile if set
	if config.Timing.Profile != "" {
		if preset, ok := GetScanTimingPresets()[config.Timing.Profile]; ok {
			profileName := config.Timing.Profile // Save profile name before overwriting
			p.config.Timing = preset
			p.config.Timing.Profile = profileName // Restore profile name
		}
	}

	return p
}

// SetOnComplete sets a callback that is invoked when the pipeline completes successfully.
// The callback receives the final list of discovered devices with all phases applied.
// This allows the Service to sync pipeline results back to its device cache.
func (p *Pipeline) SetOnComplete(callback func(devices []*DiscoveredDevice)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onComplete = callback
}

// Start begins a new pipeline run.
// Fixes #919: Use isRunningState helper for cleaner state check.
func (p *Pipeline) Start(ctx context.Context, trigger string) (*PipelineRun, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentRun != nil && isRunningState(p.currentRun.Status) {
		return nil, errors.New("pipeline already running")
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
// Fixes #877: Capture runID before modifying state to prevent race with new Start() calls.
func (p *Pipeline) Cancel() error {
	p.mu.Lock()

	if p.currentRun == nil || p.cancelFunc == nil {
		p.mu.Unlock()
		return errors.New("no pipeline running")
	}

	// Capture run ID before cancellation for the run() goroutine to check
	runID := p.currentRun.ID

	p.cancelFunc()
	p.currentRun.Status = PipelineStateCanceled
	now := time.Now()
	p.currentRun.CompletedAt = &now
	p.mu.Unlock()

	logging.GetLogger().Info("Pipeline cancelled", "runId", runID)
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
		maps.Copy(run.PhaseDurations, p.currentRun.PhaseDurations)
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
		return errors.New("cannot update config while pipeline is running")
	}

	// Apply timing profile if set
	if config.Timing.Profile != "" {
		if preset, ok := GetScanTimingPresets()[config.Timing.Profile]; ok {
			profileName := config.Timing.Profile // Save profile name before overwriting
			config.Timing = preset
			config.Timing.Profile = profileName // Restore profile name
		}
	}

	// Check if resolution config changed - recreate resolution phase if needed
	if p.config.Resolution.DNS != config.Resolution.DNS ||
		p.config.Resolution.NetBIOS != config.Resolution.NetBIOS ||
		p.config.Resolution.MDNS != config.Resolution.MDNS {
		resConfig := &ResolutionConfig{
			DNS:     config.Resolution.DNS,
			NetBIOS: config.Resolution.NetBIOS,
			MDNS:    config.Resolution.MDNS,
			Timing: ResolutionTiming{
				DNSTimeout:           pipelineResDNSTimeoutMs * time.Millisecond,
				NetBIOSTimeout:       pipelineResDNSTimeoutMs * time.Millisecond,
				MDNSTimeout:          pipelineResMDNSTimeoutS * time.Second,
				PhaseTimeout:         config.Timing.PhaseTimeout,
				MaxConcurrentDNS:     pipelineResMaxConcDNS,
				MaxConcurrentNetBIOS: pipelineResMaxConcNetBIOS,
				MaxConcurrentMDNS:    pipelineResMaxConcMDNS,
			},
		}

		interfaceName := ""
		if p.deviceDiscovery != nil {
			interfaceName = p.deviceDiscovery.GetInterfaceName()
		}
		p.resolutionPhase = NewResolutionPhase(interfaceName, resConfig, p.broadcaster)

		logging.GetLogger().Info("Resolution phase config updated",
			"dns", config.Resolution.DNS,
			"netbios", config.Resolution.NetBIOS,
			"mdns", config.Resolution.MDNS)
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
		if p.currentRun.Status != PipelineStateCanceled &&
			p.currentRun.Status != PipelineStateFailed {
			p.currentRun.Status = PipelineStateComplete
		}
		// Copy values under lock to avoid race after unlock (fixes #824)
		status := p.currentRun.Status
		startedAt := p.currentRun.StartedAt
		phaseDurations := make(map[string]time.Duration, len(p.currentRun.PhaseDurations))
		maps.Copy(phaseDurations, p.currentRun.PhaseDurations)
		onComplete := p.onComplete // Capture callback under lock
		p.mu.Unlock()

		// Broadcast completion using copied values
		if status == PipelineStateComplete {
			p.broadcastEvent(EventPipelineCompleted, PipelineCompletedPayload{
				TotalDevices:   len(devices),
				TotalDuration:  time.Since(startedAt),
				PhaseDurations: phaseDurations,
			})

			// Notify Service of completion so it can sync devices
			if onComplete != nil {
				onComplete(devices)
			}
		}
	}()

	// Phase 1: Enumeration (always runs)
	phaseNumber++
	p.updateState(PipelineStateEnumerating, phaseNameEnumeration)
	devices, err = p.runEnumerationPhase(ctx, phaseNumber)
	if err != nil {
		p.handlePhaseError(phaseNameEnumeration, err)
		return
	}

	// Phase 2: Name Resolution (if enabled)
	if p.config.Phases.NameResolution {
		phaseNumber++
		p.updateState(PipelineStateResolving, phaseNameResolution)
		devices, err = p.runResolutionPhase(ctx, devices, phaseNumber)
		if err != nil {
			p.handlePhaseError(phaseNameResolution, err)
			return
		}
	}

	// Phase 3: Service Discovery (if enabled)
	if p.config.Phases.ServiceDiscovery {
		phaseNumber++
		p.updateState(PipelineStateScanning, phaseNameScanning)
		devices, err = p.runScanningPhase(ctx, devices, phaseNumber)
		if err != nil {
			p.handlePhaseError(phaseNameScanning, err)
			return
		}
	}

	// Phase 4: Vulnerability Assessment (if enabled)
	if p.config.Phases.VulnAssessment {
		phaseNumber++
		p.updateState(PipelineStateAssessing, phaseNameAssessment)
		devices, err = p.runAssessmentPhase(ctx, devices, phaseNumber)
		if err != nil {
			p.handlePhaseError(phaseNameAssessment, err)
			return
		}
	}

	p.logPipelineCompletion(len(devices))
}

// logPipelineCompletion records final device count and logs completion.
func (p *Pipeline) logPipelineCompletion(deviceCount int) {
	p.mu.Lock()
	p.currentRun.DevicesFound = deviceCount
	runID := p.currentRun.ID
	startedAt := p.currentRun.StartedAt
	p.mu.Unlock()

	logging.GetLogger().Info("Pipeline completed",
		"runId", runID,
		"devicesFound", deviceCount,
		"duration", time.Since(startedAt))
}

// updateState updates the current run state.
func (p *Pipeline) updateState(state PipelineState, phase string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentRun.Status = state
	p.currentRun.CurrentPhase = phase
}

// isRunningState returns true if the state represents an active pipeline run.
// Fixes #919: Centralized helper for state checking to avoid redundant conditions.
func isRunningState(state PipelineState) bool {
	return state == PipelineStateEnumerating ||
		state == PipelineStateResolving ||
		state == PipelineStateScanning ||
		state == PipelineStateAssessing
}

// handlePhaseError handles errors during phase execution.
// Fixes #880: Limit error array to maxPipelineErrors to prevent unbounded growth.
func (p *Pipeline) handlePhaseError(phase string, err error) {
	logging.GetLogger().Error("Pipeline phase failed",
		"phase", phase,
		"error", err)

	p.mu.Lock()
	p.currentRun.Status = PipelineStateFailed
	p.currentRun.Errors = append(p.currentRun.Errors, fmt.Sprintf("%s: %v", phase, err))
	// Fixes #880: Cap errors to prevent unbounded memory growth
	if len(p.currentRun.Errors) > maxPipelineErrors {
		p.currentRun.Errors = p.currentRun.Errors[len(p.currentRun.Errors)-maxPipelineErrors:]
	}
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
	phases := []string{phaseNameEnumeration}
	if p.config.Phases.NameResolution {
		phases = append(phases, phaseNameResolution)
	}
	if p.config.Phases.ServiceDiscovery {
		phases = append(phases, phaseNameScanning)
	}
	if p.config.Phases.VulnAssessment {
		phases = append(phases, phaseNameAssessment)
	}
	return phases
}
