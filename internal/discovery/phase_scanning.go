// Package discovery implements multi-protocol network device discovery.
// This file implements Phase 3 (Service Discovery) of the discovery pipeline.
// It performs port scanning and extended SNMP MIB collection for discovered devices.
package discovery

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ScanningPhase implements the Phase interface for service discovery.
// Phase 3 enriches discovered devices with:
//   - Open ports and service banners
//   - HTTP/HTTPS probing
//   - Extended SNMP MIB data (interfaces, IPs, VLANs, MACs, etc.)
type ScanningPhase struct {
	profiler       *DeviceProfiler
	snmpCollector  *SNMPCollector
	pipelineConfig *PipelineConfig
	snmpConfig     *config.SNMPConfig
	broadcaster    EventBroadcaster
}

// NewScanningPhase creates a new Phase 3 implementation.
func NewScanningPhase(
	pipelineConfig *PipelineConfig,
	snmpConfig *config.SNMPConfig,
	broadcaster EventBroadcaster,
) *ScanningPhase {
	// Create profiler config from pipeline config
	profilerCfg := &ProfilerConfig{
		Enabled:           pipelineConfig.PortScan.Intensity != PortScanOff,
		Timeout:           pipelineConfig.Timing.PhaseTimeout,
		MaxConcurrent:     pipelineConfig.Timing.MaxConcurrentHosts,
		QuickPorts:        QuickPorts,
		PortScanIntensity: pipelineConfig.PortScan.Intensity,
		TimingProfile:     pipelineConfig.Timing.Profile,
		CustomPorts:       pipelineConfig.PortScan.CustomPorts,
		BannerGrab:        pipelineConfig.PortScan.BannerGrab,
		ProbeDelay:        pipelineConfig.Timing.ProbeDelay,
		HostDelay:         pipelineConfig.Timing.HostDelay,
		ConnectTimeout:    pipelineConfig.PortScan.ConnectTimeout,
	}

	profiler := NewDeviceProfiler(profilerCfg, snmpConfig)

	// Create SNMP collector if enabled
	var snmpCollector *SNMPCollector
	if pipelineConfig.SNMPCollection.Enabled && snmpConfig != nil {
		snmpCollector = NewSNMPCollector(snmpConfig, pipelineConfig.SNMPCollection.MIBs)
		snmpCollector.SetTimeout(pipelineConfig.SNMPCollection.WalkTimeout)
		snmpCollector.SetMaxOIDsPerRequest(pipelineConfig.SNMPCollection.MaxOIDsPerRequest)
	}

	return &ScanningPhase{
		profiler:       profiler,
		snmpCollector:  snmpCollector,
		pipelineConfig: pipelineConfig,
		snmpConfig:     snmpConfig,
		broadcaster:    broadcaster,
	}
}

// Name returns the phase name.
func (p *ScanningPhase) Name() string {
	return "scanning"
}

// Run executes the service discovery phase.
// Devices from Phase 2 are enriched with open ports, services, and SNMP data.
func (p *ScanningPhase) Run(
	ctx context.Context,
	devices []*DiscoveredDevice,
	progressCh chan<- PhaseProgressPayload,
) ([]*DiscoveredDevice, error) {
	start := time.Now()
	portScanEnabled := p.pipelineConfig.PortScan.Intensity != PortScanOff
	snmpEnabled := p.pipelineConfig.SNMPCollection.Enabled && p.snmpCollector != nil

	logging.GetLogger().InfoContext(ctx, "Scanning phase starting",
		"devices", len(devices),
		"portScan", portScanEnabled,
		"portIntensity", p.pipelineConfig.PortScan.Intensity,
		"snmp", snmpEnabled)

	if len(devices) == 0 {
		return devices, nil
	}

	// Check if anything is enabled
	if !portScanEnabled && !snmpEnabled {
		logging.GetLogger().InfoContext(ctx, "Scanning phase skipped - both port scanning and SNMP disabled")
		return devices, nil
	}

	// Use phase timeout if configured
	scanCtx := ctx
	if p.pipelineConfig.Timing.PhaseTimeout > 0 {
		var cancel context.CancelFunc
		scanCtx, cancel = context.WithTimeout(ctx, p.pipelineConfig.Timing.PhaseTimeout)
		defer cancel()
	}

	// Track progress
	var progress ScanningProgress
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
			case <-ticker.C:
				if progressCh != nil {
					progressCh <- PhaseProgressPayload{
						Phase:           "scanning",
						ProcessedCount:  int(progress.Scanned()),
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
	workerCount := p.pipelineConfig.Timing.MaxConcurrentHosts
	if workerCount <= 0 {
		workerCount = 20
	}

	for range workerCount {
		wg.Go(func() {
			p.scanWorker(scanCtx, deviceCh, &progress, portScanEnabled, snmpEnabled)
		})
	}

	wg.Wait()
	close(done)

	// Log summary
	scanned := progress.Scanned()
	portsFound := progress.PortsFound()
	snmpSuccess := progress.SNMPSuccess()

	logging.GetLogger().InfoContext(ctx, "Scanning phase completed",
		"scanned", scanned,
		"total", len(devices),
		"portsFound", portsFound,
		"snmpSuccess", snmpSuccess,
		"duration", time.Since(start))

	return devices, nil
}

// scanWorker processes devices from the channel.
func (p *ScanningPhase) scanWorker(
	ctx context.Context,
	deviceCh <-chan *DiscoveredDevice,
	progress *ScanningProgress,
	portScan, snmpScan bool,
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

		// Apply host delay for IDS-friendly scanning
		if p.pipelineConfig.Timing.HostDelay > 0 {
			time.Sleep(p.pipelineConfig.Timing.HostDelay)
		}

		var wg sync.WaitGroup
		var mu sync.Mutex

		// Port scanning (if enabled)
		if portScan {
			wg.Go(func() {
				profile := p.scanPorts(ctx, device.IP)
				if profile != nil {
					mu.Lock()
					device.Profile = profile
					progress.AddPortsFound(len(profile.OpenPorts))
					mu.Unlock()
				}
			})
		}

		// Extended SNMP collection (if enabled)
		if snmpScan {
			wg.Go(func() {
				snmpData := p.collectSNMP(ctx, device.IP)
				if snmpData != nil {
					mu.Lock()
					device.SNMPData = snmpData
					if len(snmpData.Errors) == 0 || snmpData.System != nil {
						progress.IncrementSNMPSuccess()
					}
					mu.Unlock()
				}
			})
		}

		wg.Wait()
		progress.IncrementScanned()

		// Broadcast device update if we have a broadcaster
		if p.broadcaster != nil {
			p.broadcaster.BroadcastPipelineEvent(PipelineEvent{
				Type:      EventDeviceUpdated,
				Timestamp: time.Now(),
				Payload: DeviceUpdatedPayload{
					Device: device,
					Phase:  "scanning",
				},
			})
		}
	}
}

// scanPorts performs port scanning on a device.
func (p *ScanningPhase) scanPorts(ctx context.Context, ip string) *DeviceProfile {
	ports := p.profiler.config.GetPortsForIntensity()
	if len(ports) == 0 {
		return nil
	}

	profile := &DeviceProfile{
		ProfiledAt:  time.Now(),
		OpenPorts:   []OpenPort{},
		DeviceIcons: []string{},
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.pipelineConfig.Timing.MaxConcurrentHosts)

	for _, port := range ports {
		select {
		case <-ctx.Done():
			return profile
		default:
		}

		wg.Add(1)
		go func(port int) {
			defer wg.Done()

			// Check for cancellation before acquiring semaphore
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			// Check for cancellation before sleeping
			if p.pipelineConfig.Timing.ProbeDelay > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(p.pipelineConfig.Timing.ProbeDelay):
				}
			}

			result := p.profiler.checkPortWithConfig(ctx, ip, port)
			if result.IsOpen {
				mu.Lock()
				profile.OpenPorts = append(profile.OpenPorts, result)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()

	// Probe HTTP/HTTPS if ports are open
	for _, op := range profile.OpenPorts {
		if op.Port == 80 || op.Port == 8080 {
			if info := p.profiler.probeHTTP(ctx, ip, op.Port, false); info != nil {
				profile.HTTPInfo = info
				break
			}
		}
		if op.Port == 443 || op.Port == 8443 {
			if info := p.profiler.probeHTTP(ctx, ip, op.Port, true); info != nil {
				profile.HTTPInfo = info
				break
			}
		}
	}

	// Try SNMP probing (basic - for device type inference)
	if info := p.profiler.probeSNMP(ctx, ip); info != nil {
		profile.SNMPInfo = info
	}

	// Infer device type and icons
	p.profiler.inferDeviceType(profile)

	logging.GetLogger().DebugContext(ctx, "Port scan completed",
		"ip", ip,
		"openPorts", len(profile.OpenPorts),
		"deviceType", profile.DeviceType)

	return profile
}

// collectSNMP performs extended SNMP MIB collection.
func (p *ScanningPhase) collectSNMP(ctx context.Context, ip string) *SNMPFullData {
	if p.snmpCollector == nil {
		return nil
	}

	data, err := p.snmpCollector.Collect(ctx, ip)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "SNMP collection failed", "ip", ip, "error", err)
		return &SNMPFullData{
			CollectedAt: time.Now(),
			Errors:      []string{fmt.Sprintf("collection failed: %v", err)},
		}
	}

	// Log collection summary
	ifCount := len(data.Interfaces)
	macCount := len(data.MACTable)
	vlanCount := len(data.VLANs)
	lldpCount := len(data.LLDPNeighbors)

	if ifCount > 0 || macCount > 0 {
		logging.GetLogger().DebugContext(ctx, "SNMP collection completed",
			"ip", ip,
			"interfaces", ifCount,
			"macs", macCount,
			"vlans", vlanCount,
			"lldp", lldpCount)
	}

	return data
}

// ScanningProgress tracks progress during the scanning phase.
type ScanningProgress struct {
	mu            sync.RWMutex
	startTime     time.Time
	totalDevices  int
	scanned       int64
	portsFound    int64
	snmpSuccess   int64
	currentTarget string
}

// Start initializes progress tracking.
func (p *ScanningProgress) Start(totalDevices int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startTime = time.Now()
	p.totalDevices = totalDevices
}

// SetCurrentTarget updates the current scanning target.
func (p *ScanningProgress) SetCurrentTarget(target string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentTarget = target
}

// CurrentTarget returns the current scanning target.
func (p *ScanningProgress) CurrentTarget() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentTarget
}

// IncrementScanned increments the scanned device count.
func (p *ScanningProgress) IncrementScanned() {
	atomic.AddInt64(&p.scanned, 1)
}

// Scanned returns the number of scanned devices.
func (p *ScanningProgress) Scanned() int64 {
	return atomic.LoadInt64(&p.scanned)
}

// AddPortsFound adds to the open ports count.
func (p *ScanningProgress) AddPortsFound(count int) {
	atomic.AddInt64(&p.portsFound, int64(count))
}

// PortsFound returns the total open ports found.
func (p *ScanningProgress) PortsFound() int64 {
	return atomic.LoadInt64(&p.portsFound)
}

// IncrementSNMPSuccess increments successful SNMP collections.
func (p *ScanningProgress) IncrementSNMPSuccess() {
	atomic.AddInt64(&p.snmpSuccess, 1)
}

// SNMPSuccess returns the number of successful SNMP collections.
func (p *ScanningProgress) SNMPSuccess() int64 {
	return atomic.LoadInt64(&p.snmpSuccess)
}

// PercentComplete returns completion percentage.
func (p *ScanningProgress) PercentComplete() float64 {
	p.mu.RLock()
	total := p.totalDevices
	p.mu.RUnlock()

	if total == 0 {
		return 100
	}
	scanned := p.Scanned()
	return float64(scanned) / float64(total) * 100
}

// DeviceUpdatedPayload is sent when a device is updated during scanning.
type DeviceUpdatedPayload struct {
	Device *DiscoveredDevice `json:"device"`
	Phase  string            `json:"phase"`
}

// ScanningStatsPayload returns statistics for WebSocket broadcast.
type ScanningStatsPayload struct {
	TotalDevices    int     `json:"totalDevices"`
	ScannedDevices  int64   `json:"scannedDevices"`
	OpenPortsFound  int64   `json:"openPortsFound"`
	SNMPSuccessful  int64   `json:"snmpSuccessful"`
	PercentComplete float64 `json:"percentComplete"`
	ElapsedMs       int64   `json:"elapsedMs"`
}

// GetStats returns current scanning statistics.
func (p *ScanningProgress) GetStats(start time.Time) ScanningStatsPayload {
	p.mu.RLock()
	total := p.totalDevices
	p.mu.RUnlock()

	return ScanningStatsPayload{
		TotalDevices:    total,
		ScannedDevices:  p.Scanned(),
		OpenPortsFound:  p.PortsFound(),
		SNMPSuccessful:  p.SNMPSuccess(),
		PercentComplete: p.PercentComplete(),
		ElapsedMs:       time.Since(start).Milliseconds(),
	}
}
