package discovery

// pipeline_phases.go contains the per-phase runners (enumeration, resolution,
// scanning, assessment) plus the scanning-phase helpers (queue, poll, attach
// profiles, finalize) and the port-selection helper.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// runEnumerationPhase executes the device enumeration phase.
func (p *Pipeline) runEnumerationPhase(
	ctx context.Context,
	phaseNumber int,
) ([]*DiscoveredDevice, error) {
	start := time.Now()
	totalPhases := p.countEnabledPhases()

	p.broadcastEvent(EventPhaseStarted, PhaseStartedPayload{
		Phase:       phaseNameEnumeration,
		PhaseNumber: phaseNumber,
		TotalPhases: totalPhases,
		DeviceCount: 0,
	})

	// Use existing device discovery
	if err := p.deviceDiscovery.Scan(ctx); err != nil {
		// ErrScanInProgress is a soft error - just means another scan was running
		// Continue with existing device data rather than failing the pipeline
		if errors.Is(err, ErrScanInProgress) {
			logging.GetLogger().WarnContext(ctx, "Pipeline enumeration: scan skipped - using existing device data")
		} else {
			return nil, fmt.Errorf("enumeration failed: %w", err)
		}
	}

	// Fixes #962: Defensive copy to prevent mutation of internal DeviceDiscovery state
	deviceRefs := p.deviceDiscovery.GetDevices()
	devices := make([]*DiscoveredDevice, len(deviceRefs))
	copy(devices, deviceRefs)

	duration := time.Since(start)
	p.mu.Lock()
	p.currentRun.PhaseDurations[phaseNameEnumeration] = duration
	p.mu.Unlock()

	p.broadcastEvent(EventPhaseCompleted, PhaseCompletedPayload{
		Phase:        phaseNameEnumeration,
		DevicesFound: len(devices),
		Duration:     duration,
	})

	logging.GetLogger().InfoContext(ctx, "Enumeration phase completed",
		"devices", len(devices),
		"duration", duration)

	return devices, nil
}

// runResolutionPhase executes the name resolution phase.
// Uses DNS, NetBIOS, and mDNS to resolve device hostnames.
func (p *Pipeline) runResolutionPhase(
	ctx context.Context,
	devices []*DiscoveredDevice,
	phaseNumber int,
) ([]*DiscoveredDevice, error) {
	start := time.Now()
	totalPhases := p.countEnabledPhases()

	p.broadcastEvent(EventPhaseStarted, PhaseStartedPayload{
		Phase:       phaseNameResolution,
		PhaseNumber: phaseNumber,
		TotalPhases: totalPhases,
		DeviceCount: len(devices),
	})

	// Create progress channel for resolution updates
	progressCh := make(chan PhaseProgressPayload, pipelineProgressChannelSize)
	go func() {
		for progress := range progressCh {
			p.broadcastEvent(EventPhaseProgress, progress)
		}
	}()

	// Run active name resolution (DNS, NetBIOS, mDNS)
	var resolvedDevices []*DiscoveredDevice
	var err error
	if p.resolutionPhase != nil {
		resolvedDevices, err = p.resolutionPhase.Run(ctx, devices, progressCh)
	} else {
		// Fallback: just pass through without resolution
		resolvedDevices = devices
	}
	close(progressCh)

	if err != nil {
		return devices, err // Return original devices on error
	}

	duration := time.Since(start)
	p.mu.Lock()
	p.currentRun.PhaseDurations[phaseNameResolution] = duration
	p.mu.Unlock()

	namesResolved := 0
	for _, d := range resolvedDevices {
		if d.DisplayName != "" && d.DisplayName != d.IP {
			namesResolved++
		}
	}

	p.broadcastEvent(EventPhaseCompleted, PhaseCompletedPayload{
		Phase:         phaseNameResolution,
		NamesResolved: namesResolved,
		Duration:      duration,
	})

	logging.GetLogger().InfoContext(ctx, "Resolution phase completed",
		"namesResolved", namesResolved,
		"duration", duration)

	return resolvedDevices, nil
}

// runScanningPhase executes the service discovery phase.
//
//nolint:unparam // Error return kept for interface consistency with other phase methods.
func (p *Pipeline) runScanningPhase(
	ctx context.Context,
	devices []*DiscoveredDevice,
	phaseNumber int,
) ([]*DiscoveredDevice, error) {
	start := time.Now()

	p.broadcastEvent(EventPhaseStarted, PhaseStartedPayload{
		Phase:       phaseNameScanning,
		PhaseNumber: phaseNumber,
		TotalPhases: p.countEnabledPhases(),
		DeviceCount: len(devices),
	})

	// Sync profiler config with pipeline settings (fixes profiler race condition)
	p.profiler.UpdateScanConfig(
		p.config.PortScan.Intensity,
		p.config.PortScan.CustomPorts,
		p.config.Timing.Profile,
	)

	// Get ports based on intensity and run profiling if enabled
	if ports := p.getPortsForIntensity(); len(ports) == 0 {
		logging.GetLogger().InfoContext(ctx, "Port scanning disabled, skipping service discovery")
	} else {
		p.queueDevicesForProfiling(devices)

		spc := &scanningPhaseContext{
			ctx:          ctx,
			devices:      devices,
			start:        start,
			processedSet: make(map[string]bool, len(devices)),
		}
		p.waitForProfilingCompletion(spc)
	}

	// Attach profiles to devices and finalize phase
	openPorts := p.attachProfilesToDevices(devices)
	p.finalizeScanningPhase(ctx, start, openPorts)

	return devices, nil
}

// runAssessmentPhase executes the vulnerability assessment phase.
//
//nolint:unparam // Error return kept for interface consistency with other phase methods.
func (p *Pipeline) runAssessmentPhase(
	ctx context.Context,
	devices []*DiscoveredDevice,
	phaseNumber int,
) ([]*DiscoveredDevice, error) {
	start := time.Now()
	totalPhases := p.countEnabledPhases()

	p.broadcastEvent(EventPhaseStarted, PhaseStartedPayload{
		Phase:       phaseNameAssessment,
		PhaseNumber: phaseNumber,
		TotalPhases: totalPhases,
		DeviceCount: len(devices),
	})

	// Vulnerability assessment will be implemented in phase_assessment.go
	// For now, this is a placeholder

	duration := time.Since(start)
	p.mu.Lock()
	p.currentRun.PhaseDurations[phaseNameAssessment] = duration
	p.mu.Unlock()

	p.broadcastEvent(EventPhaseCompleted, PhaseCompletedPayload{
		Phase:      phaseNameAssessment,
		VulnsFound: 0, // Will be populated by actual implementation
		Duration:   duration,
	})

	logging.GetLogger().InfoContext(ctx, "Assessment phase completed",
		"duration", duration)

	return devices, nil
}

// scanningPhaseContext holds the context for scanning phase execution.
// This struct reduces the number of parameters passed between helper functions.
type scanningPhaseContext struct {
	ctx          context.Context
	devices      []*DiscoveredDevice
	start        time.Time
	processedSet map[string]bool
}

// queueDevicesForProfiling queues all devices with valid IPs for profiling.
func (p *Pipeline) queueDevicesForProfiling(devices []*DiscoveredDevice) {
	for _, device := range devices {
		if device.IP != "" {
			_ = p.profiler.QueueProfile(device.IP)
		}
	}
}

// getScanningTimeout returns the timeout for the scanning phase.
func (p *Pipeline) getScanningTimeout() time.Duration {
	timeout := p.config.Timing.PhaseTimeout
	if timeout == 0 {
		timeout = pipelineDefaultPhaseTimeoutM * time.Minute
	}
	return timeout
}

// checkDeviceProfilingStatus checks if a single device's profiling is complete.
// Returns true if the device is done (either profiled or not in progress).
func (p *Pipeline) checkDeviceProfilingStatus(
	spc *scanningPhaseContext,
	device *DiscoveredDevice,
) bool {
	if device.IP == "" {
		return true
	}

	// Only check profiler state for devices not yet processed
	if !spc.processedSet[device.IP] {
		if p.profiler.GetProfile(device.IP) != nil {
			spc.processedSet[device.IP] = true
			return true
		}
		if p.profiler.IsProfiling(device.IP) {
			return false
		}
	}
	return true
}

// checkAllDevicesProfiled checks if all devices have completed profiling.
// Returns (true, false) when all devices are done, (false, true) if cancelled.
func (p *Pipeline) checkAllDevicesProfiled(
	waitCtx context.Context,
	spc *scanningPhaseContext,
) (bool, bool) {
	for _, device := range spc.devices {
		// Check for cancellation during iteration (fixes #878)
		select {
		case <-waitCtx.Done():
			return false, true
		default:
		}
		if !p.checkDeviceProfilingStatus(spc, device) {
			return false, false
		}
	}
	return true, false
}

// broadcastScanProgress broadcasts the current scanning phase progress.
func (p *Pipeline) broadcastScanProgress(spc *scanningPhaseContext) {
	processed := len(spc.processedSet)
	totalDevices := len(spc.devices)

	// Prevent division by zero (fixes #821)
	percentComplete := float64(pipelinePercentComplete)
	if totalDevices > 0 {
		percentComplete = float64(processed) / float64(totalDevices) * pipelinePercentComplete
	}

	p.broadcastEvent(EventPhaseProgress, PhaseProgressPayload{
		Phase:           phaseNameScanning,
		ProcessedCount:  processed,
		TotalCount:      totalDevices,
		PercentComplete: percentComplete,
		ElapsedMs:       time.Since(spc.start).Milliseconds(),
	})
}

// handleScanningContextDone handles context cancellation or timeout.
func (p *Pipeline) handleScanningContextDone(
	ctx context.Context,
	waitCtx context.Context,
	timeout time.Duration,
) {
	// Fixes #938: Distinguish between timeout and intentional cancellation
	if ctx.Err() != nil {
		logging.GetLogger().InfoContext(ctx, "Scanning phase cancelled", "reason", ctx.Err())
	} else if waitCtx.Err() == context.DeadlineExceeded {
		logging.GetLogger().WarnContext(ctx, "Scanning phase timed out", "timeout", timeout)
	}
}

// waitForProfilingCompletion waits for all device profiles to complete.
// Returns when all devices are profiled, context is cancelled, or timeout occurs.
func (p *Pipeline) waitForProfilingCompletion(spc *scanningPhaseContext) {
	timeout := p.getScanningTimeout()
	waitCtx, cancel := context.WithTimeout(spc.ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pipelineProgressTickerMs * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			p.handleScanningContextDone(spc.ctx, waitCtx, timeout)
			return
		case <-ticker.C:
			allDone, cancelled := p.checkAllDevicesProfiled(waitCtx, spc)
			if cancelled || allDone {
				return
			}
			p.broadcastScanProgress(spc)
		}
	}
}

// attachProfilesToDevices attaches profiler results to devices and counts open ports.
func (p *Pipeline) attachProfilesToDevices(devices []*DiscoveredDevice) int {
	openPorts := 0
	for _, device := range devices {
		if device.IP != "" {
			if profile := p.profiler.GetProfile(device.IP); profile != nil {
				device.Profile = profile
				openPorts += len(profile.OpenPorts)
			}
		}
	}
	return openPorts
}

// finalizeScanningPhase records the phase duration and broadcasts completion.
func (p *Pipeline) finalizeScanningPhase(
	ctx context.Context,
	start time.Time,
	openPorts int,
) {
	duration := time.Since(start)

	p.mu.Lock()
	p.currentRun.PhaseDurations[phaseNameScanning] = duration
	p.mu.Unlock()

	p.broadcastEvent(EventPhaseCompleted, PhaseCompletedPayload{
		Phase:     phaseNameScanning,
		PortsOpen: openPorts,
		Duration:  duration,
	})

	logging.GetLogger().InfoContext(ctx, "Scanning phase completed",
		"openPorts", openPorts,
		"duration", duration)
}

// getPortsForIntensity returns the port list based on configured intensity.
func (p *Pipeline) getPortsForIntensity() []int {
	switch p.config.PortScan.Intensity {
	case PortScanOff:
		return nil
	case PortScanQuick:
		return GetQuickPorts()
	case PortScanStandard:
		return GetStandardPorts()
	case PortScanComprehensive:
		return GetComprehensivePorts()
	case PortScanCustom:
		return p.config.PortScan.CustomPorts
	default:
		return nil
	}
}
