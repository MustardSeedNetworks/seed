package discovery

// This file implements Phase 3 (Service Discovery) of the discovery pipeline.
// It performs port scanning and extended SNMP MIB collection for discovered devices.

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
)

// Scanning phase constants.
const (
	scanProgressTickerMs = 500 // Progress reporting interval in milliseconds
	scanPercentComplete  = 100 // Value representing 100% completion

	// Common port numbers for HTTP probing.
	portHTTP     = 80
	portHTTPAlt  = 8080
	portHTTPS    = 443
	portHTTPSAlt = 8443
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
		QuickPorts:        GetQuickPorts(),
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
		ticker := time.NewTicker(scanProgressTickerMs * time.Millisecond)
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
		if p.shouldStopScanning(ctx) {
			return
		}

		if device.IP == "" {
			continue
		}

		p.processDevice(ctx, device, progress, portScan, snmpScan)
	}
}

// shouldStopScanning checks if the context has been cancelled.
func (p *ScanningPhase) shouldStopScanning(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// processDevice handles scanning for a single device.
func (p *ScanningPhase) processDevice(
	ctx context.Context,
	device *DiscoveredDevice,
	progress *ScanningProgress,
	portScan, snmpScan bool,
) {
	progress.SetCurrentTarget(device.IP)
	p.applyHostDelay()

	var wg sync.WaitGroup
	var mu sync.Mutex

	p.startPortScan(ctx, &wg, &mu, device, progress, portScan)
	p.startSNMPCollection(ctx, &wg, &mu, device, progress, snmpScan)

	wg.Wait()
	progress.IncrementScanned()

	p.broadcastDeviceUpdate(device)
}

// applyHostDelay sleeps for the configured host delay (IDS-friendly scanning).
func (p *ScanningPhase) applyHostDelay() {
	if p.pipelineConfig.Timing.HostDelay > 0 {
		time.Sleep(p.pipelineConfig.Timing.HostDelay)
	}
}

// startPortScan launches a goroutine to scan ports if enabled.
func (p *ScanningPhase) startPortScan(
	ctx context.Context,
	wg *sync.WaitGroup,
	mu *sync.Mutex,
	device *DiscoveredDevice,
	progress *ScanningProgress,
	enabled bool,
) {
	if !enabled {
		return
	}

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

// startSNMPCollection launches a goroutine to collect SNMP data if enabled.
func (p *ScanningPhase) startSNMPCollection(
	ctx context.Context,
	wg *sync.WaitGroup,
	mu *sync.Mutex,
	device *DiscoveredDevice,
	progress *ScanningProgress,
	enabled bool,
) {
	if !enabled {
		return
	}

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

// broadcastDeviceUpdate sends a device update event if a broadcaster is configured.
func (p *ScanningPhase) broadcastDeviceUpdate(device *DiscoveredDevice) {
	if p.broadcaster == nil {
		return
	}

	p.broadcaster.BroadcastPipelineEvent(PipelineEvent{
		Type:      EventDeviceUpdated,
		Timestamp: time.Now(),
		Payload: DeviceUpdatedPayload{
			Device: device,
			Phase:  "scanning",
		},
	})
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

	p.scanAllPorts(ctx, ip, ports, profile)
	p.probeHTTPServices(ctx, ip, profile)
	p.probeBasicSNMP(ctx, ip, profile)

	p.profiler.inferDeviceType(profile)

	logging.GetLogger().DebugContext(ctx, "Port scan completed",
		"ip", ip,
		"openPorts", len(profile.OpenPorts),
		"deviceType", profile.DeviceType)

	return profile
}

// scanAllPorts scans all specified ports concurrently and populates the profile.
func (p *ScanningPhase) scanAllPorts(
	ctx context.Context,
	ip string,
	ports []int,
	profile *DeviceProfile,
) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.pipelineConfig.Timing.MaxConcurrentHosts)

	for _, port := range ports {
		if p.shouldStopScanning(ctx) {
			return
		}

		wg.Add(1)
		go p.scanSinglePort(ctx, ip, port, sem, &mu, &wg, profile)
	}

	wg.Wait()
}

// scanSinglePort scans a single port with proper semaphore and delay handling.
func (p *ScanningPhase) scanSinglePort(
	ctx context.Context,
	ip string,
	port int,
	sem chan struct{},
	mu *sync.Mutex,
	wg *sync.WaitGroup,
	profile *DeviceProfile,
) {
	defer wg.Done()

	if !p.acquireSemaphore(ctx, sem) {
		return
	}
	defer func() { <-sem }()

	if !p.waitProbeDelay(ctx) {
		return
	}

	result := p.profiler.checkPortWithConfig(ctx, ip, port)
	if result.IsOpen {
		mu.Lock()
		profile.OpenPorts = append(profile.OpenPorts, result)
		mu.Unlock()
	}
}

// acquireSemaphore attempts to acquire a slot in the semaphore, respecting context cancellation.
func (p *ScanningPhase) acquireSemaphore(ctx context.Context, sem chan struct{}) bool {
	select {
	case <-ctx.Done():
		return false
	case sem <- struct{}{}:
		return true
	}
}

// waitProbeDelay waits for the configured probe delay, respecting context cancellation.
func (p *ScanningPhase) waitProbeDelay(ctx context.Context) bool {
	if p.pipelineConfig.Timing.ProbeDelay <= 0 {
		return true
	}

	select {
	case <-ctx.Done():
		return false
	case <-time.After(p.pipelineConfig.Timing.ProbeDelay):
		return true
	}
}

// probeHTTPServices probes HTTP/HTTPS services on open ports.
func (p *ScanningPhase) probeHTTPServices(ctx context.Context, ip string, profile *DeviceProfile) {
	for _, op := range profile.OpenPorts {
		if info := p.tryHTTPProbe(ctx, ip, op.Port); info != nil {
			profile.HTTPInfo = info
			return
		}
	}
}

// tryHTTPProbe attempts to probe HTTP or HTTPS based on the port.
func (p *ScanningPhase) tryHTTPProbe(ctx context.Context, ip string, port int) *HTTPInfo {
	switch port {
	case portHTTP, portHTTPAlt:
		return p.profiler.probeHTTP(ctx, ip, port, false)
	case portHTTPS, portHTTPSAlt:
		return p.profiler.probeHTTP(ctx, ip, port, true)
	default:
		return nil
	}
}

// probeBasicSNMP performs basic SNMP probing for device type inference.
func (p *ScanningPhase) probeBasicSNMP(ctx context.Context, ip string, profile *DeviceProfile) {
	if info := p.profiler.probeSNMP(ctx, ip); info != nil {
		profile.SNMPInfo = info
	}
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
		return scanPercentComplete
	}
	scanned := p.Scanned()
	return float64(scanned) / float64(total) * scanPercentComplete
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
