// Package discovery implements multi-protocol network device discovery.
// This file implements Phase 4 (Vulnerability Assessment) of the discovery pipeline.
// It performs CVE lookup and risk scoring for discovered devices based on
// software versions extracted from SNMP, banners, and service fingerprints.
package discovery

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Severity level constants.
const (
	severityCritical = "CRITICAL"
	severityHigh     = "HIGH"
	severityMedium   = "MEDIUM"
	severityLow      = "LOW"
)

// AssessmentPhase implements the Phase interface for vulnerability assessment.
// Phase 4 enriches discovered devices with:
//   - CVE vulnerabilities from NVD or local database
//   - Risk scoring based on CVSS
//   - Version extraction from service banners and SNMP
type AssessmentPhase struct {
	scanner     *VulnerabilityScanner
	config      *AssessmentConfig
	broadcaster EventBroadcaster
}

// AssessmentConfig controls Phase 4 behavior.
type AssessmentConfig struct {
	// Enabled controls whether assessment runs.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// CVEDatabase specifies the vulnerability data source.
	CVEDatabase string `yaml:"cve_database" json:"cveDatabase"` // "nvd" or "local"

	// NVDAPIKey for NVD API access (optional, improves rate limits).
	NVDAPIKey string `yaml:"nvd_api_key" json:"nvdApiKey,omitempty"`

	// LocalDBPath for local CVE database file.
	LocalDBPath string `yaml:"local_db_path" json:"localDbPath,omitempty"`

	// SeverityThreshold filters which CVEs to report.
	SeverityThreshold string `yaml:"severity_threshold" json:"severityThreshold"` // low, medium, high, critical

	// MaxConcurrent limits parallel vulnerability checks.
	MaxConcurrent int `yaml:"max_concurrent" json:"maxConcurrent"`

	// Timing controls assessment rate limiting.
	Timing AssessmentTiming `yaml:"timing" json:"timing"`
}

// AssessmentTiming controls rate limiting for vulnerability assessment.
type AssessmentTiming struct {
	// DeviceDelay between devices.
	DeviceDelay time.Duration `yaml:"device_delay" json:"deviceDelay"`

	// PhaseTimeout for the entire assessment phase.
	PhaseTimeout time.Duration `yaml:"phase_timeout" json:"phaseTimeout"`
}

// DefaultAssessmentConfig returns default assessment settings.
// Assessment is disabled by default - requires explicit opt-in.
func DefaultAssessmentConfig() *AssessmentConfig {
	return &AssessmentConfig{
		Enabled:           false, // Disabled by default
		CVEDatabase:       "nvd",
		SeverityThreshold: "medium",
		MaxConcurrent:     5,
		Timing: AssessmentTiming{
			DeviceDelay:  100 * time.Millisecond,
			PhaseTimeout: 15 * time.Minute,
		},
	}
}

// NewAssessmentPhase creates a new Phase 4 implementation.
func NewAssessmentPhase(config *AssessmentConfig, broadcaster EventBroadcaster) (*AssessmentPhase, error) {
	if config == nil {
		config = DefaultAssessmentConfig()
	}

	// Only create scanner if assessment is enabled
	var scanner *VulnerabilityScanner
	var err error

	if config.Enabled {
		scannerConfig := &VulnerabilityScannerConfig{
			Enabled:           true,
			CVEDatabase:       config.CVEDatabase,
			NVDAPIKey:         config.NVDAPIKey,
			LocalDBPath:       config.LocalDBPath,
			SeverityThreshold: config.SeverityThreshold,
			MaxConcurrent:     config.MaxConcurrent,
			UpdateInterval:    86400, // 24 hours
		}

		scanner, err = NewVulnerabilityScanner(scannerConfig)
		if err != nil {
			return nil, err
		}
	}

	return &AssessmentPhase{
		scanner:     scanner,
		config:      config,
		broadcaster: broadcaster,
	}, nil
}

// Name returns the phase name.
func (p *AssessmentPhase) Name() string {
	return "assessment"
}

// Run executes the vulnerability assessment phase.
// Devices from Phase 3 are enriched with CVE data and risk scores.
func (p *AssessmentPhase) Run(
	ctx context.Context,
	devices []*DiscoveredDevice,
	progressCh chan<- PhaseProgressPayload,
) ([]*DiscoveredDevice, error) {
	start := time.Now()

	slog.Info("Assessment phase starting",
		"devices", len(devices),
		"enabled", p.config.Enabled,
		"cveDatabase", p.config.CVEDatabase,
		"severityThreshold", p.config.SeverityThreshold)

	if !p.config.Enabled || p.scanner == nil {
		slog.Info("Assessment phase skipped - vulnerability scanning disabled")
		return devices, nil
	}

	if len(devices) == 0 {
		return devices, nil
	}

	// Use phase timeout if configured
	assessCtx := ctx
	if p.config.Timing.PhaseTimeout > 0 {
		var cancel context.CancelFunc
		assessCtx, cancel = context.WithTimeout(ctx, p.config.Timing.PhaseTimeout)
		defer cancel()
	}

	// Track progress
	var progress AssessmentProgress
	progress.Start(len(devices))

	// Progress reporting goroutine
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			// Fixes #920: Also check context cancellation to prevent goroutine leak
			case <-ctx.Done():
				return
			case <-ticker.C:
				if progressCh != nil {
					progressCh <- PhaseProgressPayload{
						Phase:           "assessment",
						ProcessedCount:  int(progress.Assessed()),
						TotalCount:      len(devices),
						PercentComplete: progress.PercentComplete(),
						CurrentTarget:   progress.CurrentTarget(),
						ElapsedMs:       time.Since(start).Milliseconds(),
					}
				}
			}
		}
	}()

	// Create work channels
	deviceCh := make(chan *DiscoveredDevice, len(devices))
	for _, device := range devices {
		deviceCh <- device
	}
	close(deviceCh)

	// Run parallel workers
	var wg sync.WaitGroup
	workerCount := p.config.MaxConcurrent
	if workerCount <= 0 {
		workerCount = 5
	}

	for range workerCount {
		wg.Go(func() {
			p.assessWorker(assessCtx, deviceCh, &progress)
		})
	}

	wg.Wait()
	close(done)

	// Log summary
	assessed := progress.Assessed()
	vulnsFound := progress.VulnerabilitiesFound()
	criticalCount := progress.CriticalCount()
	highCount := progress.HighCount()

	slog.Info("Assessment phase completed",
		"assessed", assessed,
		"total", len(devices),
		"vulnsFound", vulnsFound,
		"critical", criticalCount,
		"high", highCount,
		"duration", time.Since(start))

	return devices, nil
}

// assessWorker processes devices from the channel.
func (p *AssessmentPhase) assessWorker(
	ctx context.Context,
	deviceCh <-chan *DiscoveredDevice,
	progress *AssessmentProgress,
) {
	for device := range deviceCh {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if device.IP == "" {
			continue
		}

		progress.SetCurrentTarget(device.IP)

		// Apply device delay
		if p.config.Timing.DeviceDelay > 0 {
			time.Sleep(p.config.Timing.DeviceDelay)
		}

		// Perform vulnerability assessment
		result, err := p.scanner.ScanDevice(ctx, device)
		if err != nil {
			slog.Debug("Vulnerability scan failed", "ip", device.IP, "error", err)
		}

		if result != nil && len(result.Vulnerabilities) > 0 {
			// Store vulnerability results on the device
			device.Vulnerabilities = result

			// Update progress counters
			for i := range result.Vulnerabilities {
				progress.AddVulnerability(result.Vulnerabilities[i].Severity)
			}

			slog.Debug("Vulnerabilities found",
				"ip", device.IP,
				"count", len(result.Vulnerabilities),
				"product", result.Product,
				"version", result.Version)

			// Broadcast vulnerability discovery
			if p.broadcaster != nil {
				p.broadcaster.BroadcastPipelineEvent(PipelineEvent{
					Type:      EventDeviceUpdated,
					Timestamp: time.Now(),
					Payload: VulnerabilityDiscoveredPayload{
						DeviceIP:        device.IP,
						Product:         result.Product,
						Version:         result.Version,
						Vulnerabilities: result.Vulnerabilities,
					},
				})
			}
		}

		progress.IncrementAssessed()
	}
}

// AssessmentProgress tracks progress during the assessment phase.
type AssessmentProgress struct {
	mu            sync.RWMutex
	startTime     time.Time
	totalDevices  int
	assessed      int64
	vulnsFound    int64
	criticalCount int64
	highCount     int64
	mediumCount   int64
	lowCount      int64
	currentTarget string
}

// Start initializes progress tracking.
func (p *AssessmentProgress) Start(totalDevices int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startTime = time.Now()
	p.totalDevices = totalDevices
}

// SetCurrentTarget updates the current assessment target.
func (p *AssessmentProgress) SetCurrentTarget(target string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentTarget = target
}

// CurrentTarget returns the current assessment target.
func (p *AssessmentProgress) CurrentTarget() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentTarget
}

// IncrementAssessed increments the assessed device count.
func (p *AssessmentProgress) IncrementAssessed() {
	atomic.AddInt64(&p.assessed, 1)
}

// Assessed returns the number of assessed devices.
func (p *AssessmentProgress) Assessed() int64 {
	return atomic.LoadInt64(&p.assessed)
}

// AddVulnerability increments vulnerability counters by severity.
func (p *AssessmentProgress) AddVulnerability(severity string) {
	atomic.AddInt64(&p.vulnsFound, 1)

	switch severity {
	case severityCritical:
		atomic.AddInt64(&p.criticalCount, 1)
	case severityHigh:
		atomic.AddInt64(&p.highCount, 1)
	case severityMedium:
		atomic.AddInt64(&p.mediumCount, 1)
	case severityLow:
		atomic.AddInt64(&p.lowCount, 1)
	}
}

// VulnerabilitiesFound returns the total vulnerabilities found.
func (p *AssessmentProgress) VulnerabilitiesFound() int64 {
	return atomic.LoadInt64(&p.vulnsFound)
}

// CriticalCount returns the number of critical vulnerabilities.
func (p *AssessmentProgress) CriticalCount() int64 {
	return atomic.LoadInt64(&p.criticalCount)
}

// HighCount returns the number of high severity vulnerabilities.
func (p *AssessmentProgress) HighCount() int64 {
	return atomic.LoadInt64(&p.highCount)
}

// MediumCount returns the number of medium severity vulnerabilities.
func (p *AssessmentProgress) MediumCount() int64 {
	return atomic.LoadInt64(&p.mediumCount)
}

// LowCount returns the number of low severity vulnerabilities.
func (p *AssessmentProgress) LowCount() int64 {
	return atomic.LoadInt64(&p.lowCount)
}

// PercentComplete returns completion percentage.
func (p *AssessmentProgress) PercentComplete() float64 {
	p.mu.RLock()
	total := p.totalDevices
	p.mu.RUnlock()

	if total == 0 {
		return 100
	}
	assessed := p.Assessed()
	return float64(assessed) / float64(total) * 100
}

// VulnerabilityDiscoveredPayload is sent when vulnerabilities are found.
type VulnerabilityDiscoveredPayload struct {
	DeviceIP        string          `json:"deviceIp"`
	Product         string          `json:"product"`
	Version         string          `json:"version"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

// AssessmentStatsPayload returns statistics for WebSocket broadcast.
type AssessmentStatsPayload struct {
	TotalDevices         int     `json:"totalDevices"`
	AssessedDevices      int64   `json:"assessedDevices"`
	VulnerabilitiesFound int64   `json:"vulnerabilitiesFound"`
	CriticalCount        int64   `json:"criticalCount"`
	HighCount            int64   `json:"highCount"`
	MediumCount          int64   `json:"mediumCount"`
	LowCount             int64   `json:"lowCount"`
	PercentComplete      float64 `json:"percentComplete"`
	ElapsedMs            int64   `json:"elapsedMs"`
}

// GetStats returns current assessment statistics.
func (p *AssessmentProgress) GetStats(start time.Time) AssessmentStatsPayload {
	p.mu.RLock()
	total := p.totalDevices
	p.mu.RUnlock()

	return AssessmentStatsPayload{
		TotalDevices:         total,
		AssessedDevices:      p.Assessed(),
		VulnerabilitiesFound: p.VulnerabilitiesFound(),
		CriticalCount:        p.CriticalCount(),
		HighCount:            p.HighCount(),
		MediumCount:          p.MediumCount(),
		LowCount:             p.LowCount(),
		PercentComplete:      p.PercentComplete(),
		ElapsedMs:            time.Since(start).Milliseconds(),
	}
}
