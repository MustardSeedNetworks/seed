package discovery

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Scan performs a discovery scan with the given options.
func (e *Engine) Scan(ctx context.Context, opts *ScanOptions) (*ScanResult, error) {
	if opts == nil {
		opts = DefaultQuickScanOpts()
	}

	if !e.tryStartScan() {
		return nil, errors.New("scan already in progress")
	}
	defer e.endScan()

	// Apply folded pipeline port-scan/timing config to the profiler (S4) for the
	// duration of THIS scan only. Gated on a set intensity so the default (unset)
	// path leaves the profiler's configuration untouched. The prior config is
	// captured and restored on return so a one-off override does not silently
	// become the engine's persistent scan policy (scans are serialized by
	// tryStartScan, so the snapshot/restore pair is not racing another scan).
	if e.profiler != nil && opts.PortScanIntensity != "" {
		prevIntensity, prevPorts, prevTiming := e.profiler.ScanConfigSnapshot()
		e.profiler.UpdateScanConfig(opts.PortScanIntensity, opts.PortScanCustomPorts, opts.TimingProfile)
		defer e.profiler.UpdateScanConfig(prevIntensity, prevPorts, prevTiming)
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = e.config.ScanTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	logger := logging.FromContext(ctx)
	result := &ScanResult{StartTime: time.Now(), Phases: []string{}, Stats: &ScanStats{}}

	if opts.FreshWiredScan || opts.FreshWiFiScan || opts.FreshBluetoothScan {
		result.ScanType = ScanTypeFull
	} else {
		result.ScanType = ScanTypeQuick
	}

	e.eventBus.Publish(NewScanStartedEvent(result.ScanType))

	e.runScanPhases(ctx, logger, opts, result)
	e.finalizeScanResult(result)

	e.eventBus.Publish(NewScanCompletedEvent(result.ScanType, len(result.Devices), result.Duration))
	logger.InfoContext(ctx, "Scan completed",
		"type", result.ScanType, "devices", len(result.Devices), "duration", result.Duration)

	return result, nil
}

// runScanPhases executes all scan phases in order, reporting cumulative
// progress at each phase boundary (S4.2).
func (e *Engine) runScanPhases(ctx context.Context, logger *slog.Logger, opts *ScanOptions, result *ScanResult) {
	total := e.countScanPhases(opts)
	done := 0
	complete := func(phase string) {
		done++
		e.reportScanProgress(opts, phase, float64(done)/float64(total))
	}

	// Phase 1: Discovery (enumerate stage)
	logger.InfoContext(ctx, "Starting discovery phase")
	result.Phases = append(result.Phases, "discovery")
	var enumerate Enumerator = &enumerateStage{
		registry:           e.registry,
		config:             e.config,
		wiredCollector:     e.wiredCollector,
		wifiCollector:      e.wifiCollector,
		bluetoothCollector: e.bluetoothCollector,
	}
	if err := enumerate.Enumerate(ctx, opts); err != nil {
		result.Error = err.Error()
		logger.ErrorContext(ctx, "Discovery phase failed", "error", err)
	}
	complete("discovery")

	// Phase 2: Correlation
	logger.InfoContext(ctx, "Starting correlation phase")
	result.Phases = append(result.Phases, "correlation")
	e.correlateDevices(ctx)
	complete("correlation")

	// Phase 3: Name Resolution (resolve stage)
	if opts.IncludeNameRes && e.wiredCollector != nil {
		logger.InfoContext(ctx, "Starting name resolution phase")
		result.Phases = append(result.Phases, "name_resolution")
		var resolve Resolver = &resolveStage{wiredCollector: e.wiredCollector}
		resolve.Resolve(ctx)
		complete("name_resolution")
	}

	// Phase 4: Enrichment (fingerprint stage)
	if opts.IncludeSNMP || opts.IncludePortScan || opts.IncludeProfiling {
		logger.InfoContext(ctx, "Starting enrichment phase")
		result.Phases = append(result.Phases, "enrichment")
		var enrich Enricher = &enrichStage{
			registry:      e.registry,
			config:        e.config,
			snmpCollector: e.snmpCollector,
			portScanner:   e.portScanner,
			profiler:      e.profiler,
		}
		enrich.Enrich(ctx, opts, result.Stats)
		complete("enrichment")
	}

	// Phase 5: Assessment (vuln stage)
	if opts.IncludeVulnScan && e.assessor != nil {
		logger.InfoContext(ctx, "Starting assessment phase")
		result.Phases = append(result.Phases, "assessment")
		e.assessor.Assess(ctx, result.Stats)
		complete("assessment")
	}
}

// countScanPhases returns how many phases this scan will run, so progress
// fractions are denominated against the actual (opts-gated) phase set.
// Discovery + correlation always run; the rest are conditional.
func (e *Engine) countScanPhases(opts *ScanOptions) int {
	total := 2 // discovery + correlation always run
	if opts.IncludeNameRes && e.wiredCollector != nil {
		total++
	}
	if opts.IncludeSNMP || opts.IncludePortScan || opts.IncludeProfiling {
		total++
	}
	if opts.IncludeVulnScan && e.assessor != nil {
		total++
	}
	return total
}

// reportScanProgress surfaces a completed-phase progress update on both the
// per-scan callback (opts.Progress, e.g. the engine-scan job) and the engine
// event bus (EventScanProgress, for /discovery/engine/events subscribers).
// Both are nil-safe / always-on respectively; behavior is unchanged when no
// caller supplies a Progress hook and nothing subscribes to the bus.
func (e *Engine) reportScanProgress(opts *ScanOptions, phase string, fraction float64) {
	if opts.Progress != nil {
		opts.Progress(fraction, phase)
	}
	e.eventBus.Publish(NewScanProgressEvent(phase, fraction))
}

// QuickScan performs a quick correlation-only scan.
func (e *Engine) QuickScan(ctx context.Context) (*ScanResult, error) {
	return e.Scan(ctx, DefaultQuickScanOpts())
}

// FullScan performs a comprehensive full scan.
func (e *Engine) FullScan(ctx context.Context) (*ScanResult, error) {
	return e.Scan(ctx, DefaultFullScanOpts())
}
