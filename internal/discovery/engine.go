package discovery

// engine.go implements the central Discovery Engine.
//
// The Engine is the main orchestrator for all discovery operations.
// It coordinates:
// - Device discovery (wired, WiFi, Bluetooth)
// - Device enrichment (SNMP, port scanning, profiling)
// - Vulnerability assessment
// - Event distribution to subscribers
//
// All device data flows through the DeviceRegistry, making it the single
// source of truth for the entire discovery system.

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Scan type constants.
const (
	ScanTypeFull  = "full"
	ScanTypeQuick = "quick"
)

// Engine configuration defaults.
const (
	// defaultScanTimeoutMinutes is the default scan timeout in minutes.
	defaultScanTimeoutMinutes = 5
	// defaultEventBufferSize is the default event buffer size.
	defaultEventBufferSize = 1000
	// defaultDeviceTTLHours is the default device TTL in hours.
	defaultDeviceTTLHours = 24
	// discoveryPhaseChannelBuffer is the buffer size for discovery phase error channel.
	discoveryPhaseChannelBuffer = 3
)

// Engine is the central orchestrator for all discovery operations.
type Engine struct {
	// Core components
	registry *DeviceRegistry
	eventBus *EventBus

	// Discovery sources (collectors)
	wiredCollector     *DeviceDiscovery
	wifiCollector      *WiFiBridge
	bluetoothCollector *BluetoothScanner

	// Enrichment components
	snmpCollector *SNMPCollector
	portScanner   *PortScanner
	profiler      *DeviceProfiler

	// Assessment components
	vulnScanner *VulnerabilityScanner

	// Configuration
	config *EngineConfig

	// State
	mu        sync.RWMutex
	running   bool
	scanning  bool
	lastScan  *ScanResult
	scanCount int64

	// Lifecycle
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// EngineConfig configures the discovery engine.
type EngineConfig struct {
	// Enable/disable discovery sources
	EnableWired     bool
	EnableWiFi      bool
	EnableBluetooth bool

	// Enable/disable enrichment
	EnableSNMP      bool
	EnablePortScan  bool
	EnableProfiling bool

	// Enable/disable assessment
	EnableVulnScan bool

	// Scan behavior
	AutoScanInterval time.Duration // 0 = disabled
	ScanTimeout      time.Duration

	// Event buffer size (0 = sync delivery)
	EventBufferSize int

	// Registry settings
	DeviceTTL time.Duration
}

// DefaultEngineConfig returns sensible defaults.
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		EnableWired:      true,
		EnableWiFi:       true,
		EnableBluetooth:  true,
		EnableSNMP:       true,
		EnablePortScan:   true,
		EnableProfiling:  true,
		EnableVulnScan:   true,
		AutoScanInterval: 0, // manual scans only
		ScanTimeout:      defaultScanTimeoutMinutes * time.Minute,
		EventBufferSize:  defaultEventBufferSize,
		DeviceTTL:        defaultDeviceTTLHours * time.Hour,
	}
}

// ScanOptions configures a discovery scan.
type ScanOptions struct {
	// What to discover
	IncludeWired     bool
	IncludeWiFi      bool
	IncludeBluetooth bool

	// Fresh scans (vs cached data)
	FreshWiredScan     bool
	FreshWiFiScan      bool
	FreshBluetoothScan bool

	// Enrichment options
	IncludeSNMP      bool
	IncludePortScan  bool
	IncludeProfiling bool
	IncludeNameRes   bool // DNS/NetBIOS/mDNS resolution

	// Assessment options
	IncludeVulnScan bool

	// Scan timeout (0 = use engine default)
	Timeout time.Duration

	// Port-scan + timing configuration, folded from the pipeline orchestrator
	// (ADR-0007, Phase 7 S4). These tune the profiler's port scan the same way
	// Pipeline does via DeviceProfiler.UpdateScanConfig. Zero values are inert:
	// an empty PortScanIntensity leaves the profiler's existing config
	// untouched, so the engine's prior behavior is unchanged when unset.
	PortScanIntensity   PortScanIntensity
	PortScanCustomPorts []int
	TimingProfile       ScanTimingProfile

	// Progress, if non-nil, is invoked once per completed scan phase with the
	// cumulative fraction in [0,1] and the phase name. It lets a caller (e.g.
	// the engine-scan job) surface phase-grained progress; nil disables it.
	// Not serialized — it is a server-side observability hook, not wire config.
	Progress ProgressFunc
}

// ProgressFunc reports cumulative scan progress at a phase boundary.
type ProgressFunc func(fraction float64, phase string)

// DefaultQuickScanOpts returns options for a quick correlation-only scan.
func DefaultQuickScanOpts() *ScanOptions {
	return &ScanOptions{
		IncludeWired:       true,
		IncludeWiFi:        true,
		IncludeBluetooth:   true,
		FreshWiredScan:     false,
		FreshWiFiScan:      false,
		FreshBluetoothScan: false,
		IncludeSNMP:        false,
		IncludePortScan:    false,
		IncludeProfiling:   false,
		IncludeNameRes:     false,
		IncludeVulnScan:    false,
	}
}

// DefaultFullScanOpts returns options for a comprehensive scan.
func DefaultFullScanOpts() *ScanOptions {
	return &ScanOptions{
		IncludeWired:       true,
		IncludeWiFi:        true,
		IncludeBluetooth:   true,
		FreshWiredScan:     true,
		FreshWiFiScan:      true,
		FreshBluetoothScan: true,
		IncludeSNMP:        true,
		IncludePortScan:    true,
		IncludeProfiling:   true,
		IncludeNameRes:     true,
		IncludeVulnScan:    true,
	}
}

// ScanResult contains the results of a discovery scan.
type ScanResult struct {
	Devices   []*DiscoveredDevice `json:"devices"`
	Stats     *ScanStats          `json:"stats"`
	Phases    []string            `json:"phases"`
	ScanType  string              `json:"scanType"`
	StartTime time.Time           `json:"startTime"`
	EndTime   time.Time           `json:"endTime"`
	Duration  time.Duration       `json:"duration"`
	Error     string              `json:"error,omitempty"`
}

// ScanStats contains statistics from a scan.
type ScanStats struct {
	TotalDevices      int `json:"totalDevices"`
	WiredDevices      int `json:"wiredDevices"`
	WiFiDevices       int `json:"wifiDevices"`
	BluetoothDevices  int `json:"bluetoothDevices"`
	MultiConnected    int `json:"multiConnected"`
	NewDevices        int `json:"newDevices"`
	UpdatedDevices    int `json:"updatedDevices"`
	EnrichedDevices   int `json:"enrichedDevices"`
	VulnerableDevices int `json:"vulnerableDevices"`
}

// NewEngine creates a new discovery engine.
func NewEngine(config *EngineConfig) *Engine {
	if config == nil {
		config = DefaultEngineConfig()
	}

	// Create event bus
	eventBusConfig := &EventBusConfig{
		BufferSize: config.EventBufferSize,
	}
	eventBus := NewEventBus(eventBusConfig)

	// Create registry
	registryConfig := &RegistryConfig{
		DeviceTTL:  config.DeviceTTL,
		EmitEvents: true,
	}
	registry := NewDeviceRegistry(eventBus, registryConfig)

	return &Engine{
		registry: registry,
		eventBus: eventBus,
		config:   config,
		stopCh:   make(chan struct{}),
	}
}

// SetWiredCollector sets the wired discovery collector.
func (e *Engine) SetWiredCollector(collector *DeviceDiscovery) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.wiredCollector = collector
}

// SetWiFiCollector sets the WiFi discovery collector.
func (e *Engine) SetWiFiCollector(collector *WiFiBridge) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.wifiCollector = collector
}

// SetBluetoothCollector sets the Bluetooth discovery collector.
func (e *Engine) SetBluetoothCollector(collector *BluetoothScanner) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.bluetoothCollector = collector
}

// SetSNMPCollector sets the SNMP collector.
func (e *Engine) SetSNMPCollector(collector *SNMPCollector) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snmpCollector = collector
}

// SetPortScanner sets the port scanner.
func (e *Engine) SetPortScanner(scanner *PortScanner) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.portScanner = scanner
}

// SetProfiler sets the device profiler.
func (e *Engine) SetProfiler(profiler *DeviceProfiler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.profiler = profiler
}

// SetVulnScanner sets the vulnerability scanner.
func (e *Engine) SetVulnScanner(scanner *VulnerabilityScanner) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.vulnScanner = scanner
}

// Start starts the discovery engine.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return errors.New("engine already running")
	}

	e.running = true
	e.stopCh = make(chan struct{})
	stopCh := e.stopCh

	// Start auto-scan if configured. Pass stopCh as a parameter so the
	// goroutine works on its captured copy (avoids racing on e.stopCh if
	// Stop reassigns it on a future restart cycle).
	if e.config.AutoScanInterval > 0 {
		e.wg.Add(1)
		go e.autoScanLoop(ctx, stopCh)
	}

	return nil
}

// Stop stops the discovery engine.
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	e.running = false
	close(e.stopCh)
	e.mu.Unlock()

	e.wg.Wait()
	e.eventBus.Stop()
}

// IsRunning returns whether the engine is running.
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// IsScanning returns whether a scan is in progress.
func (e *Engine) IsScanning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.scanning
}

// tryStartScan attempts to start a scan, returning false if already scanning.
func (e *Engine) tryStartScan() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.scanning {
		return false
	}
	e.scanning = true
	e.scanCount++
	return true
}

// endScan marks the scan as complete.
func (e *Engine) endScan() {
	e.mu.Lock()
	e.scanning = false
	e.mu.Unlock()
}

// finalizeScanResult populates final result fields.
func (e *Engine) finalizeScanResult(result *ScanResult) {
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Devices = e.registry.GetDevices()

	registryStats := e.registry.Stats()
	result.Stats.TotalDevices = registryStats.TotalDevices
	result.Stats.WiredDevices = registryStats.WiredDevices
	result.Stats.WiFiDevices = registryStats.WiFiDevices
	result.Stats.BluetoothDevices = registryStats.BTDevices
	result.Stats.MultiConnected = registryStats.MultiConnected

	e.mu.Lock()
	e.lastScan = result
	e.mu.Unlock()
}

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
	if opts.IncludeVulnScan && e.vulnScanner != nil {
		logger.InfoContext(ctx, "Starting assessment phase")
		result.Phases = append(result.Phases, "assessment")
		var assess Assessor = &assessStage{
			registry:    e.registry,
			eventBus:    e.eventBus,
			vulnScanner: e.vulnScanner,
		}
		assess.Assess(ctx, result.Stats)
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
	if opts.IncludeVulnScan && e.vulnScanner != nil {
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

// correlateDevices merges devices seen on multiple networks.
// Since we use MAC as the primary key, correlation happens automatically
// in AddOrUpdate. This method handles any additional correlation logic.
func (e *Engine) correlateDevices(_ context.Context) {
	// The registry already correlates by MAC on AddOrUpdate.
	// This method can be extended for additional correlation strategies
	// (e.g., IP-based correlation, hostname matching, etc.)
}

// autoScanLoop runs periodic scans. stopCh is passed as a parameter rather
// than read from e.stopCh so Stop's potential reassignment doesn't race.
func (e *Engine) autoScanLoop(ctx context.Context, stopCh <-chan struct{}) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.AutoScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _ = e.QuickScan(ctx)
		case <-stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// GetLastScan returns the most recent scan result.
func (e *Engine) GetLastScan() *ScanResult {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastScan
}

// GetDevices returns all discovered devices.
func (e *Engine) GetDevices() []*DiscoveredDevice {
	return e.registry.GetDevices()
}

// GetDevice returns a device by MAC address.
func (e *Engine) GetDevice(mac string) *DiscoveredDevice {
	return e.registry.GetDevice(mac)
}

// GetDeviceByIP returns a device by IP address.
func (e *Engine) GetDeviceByIP(ip string) *DiscoveredDevice {
	return e.registry.GetDeviceByIP(ip)
}

// GetStats returns engine and registry statistics.
func (e *Engine) GetStats() *EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	regStats := e.registry.Stats()
	ebStats := e.eventBus.Stats()

	return &EngineStats{
		Registry:  regStats,
		Events:    ebStats,
		ScanCount: e.scanCount,
		Running:   e.running,
		Scanning:  e.scanning,
	}
}

// EngineStats contains comprehensive engine statistics.
type EngineStats struct {
	Registry  RegistryStats `json:"registry"`
	Events    EventBusStats `json:"events"`
	ScanCount int64         `json:"scanCount"`
	Running   bool          `json:"running"`
	Scanning  bool          `json:"scanning"`
}

// GetCapabilities returns which discovery capabilities are available.
func (e *Engine) GetCapabilities() map[string]bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]bool{
		"wired":       e.wiredCollector != nil && e.config.EnableWired,
		"wifi":        e.wifiCollector != nil && e.config.EnableWiFi,
		"bluetooth":   e.bluetoothCollector != nil && e.config.EnableBluetooth,
		"snmp":        e.snmpCollector != nil && e.config.EnableSNMP,
		"portScan":    e.portScanner != nil && e.config.EnablePortScan,
		"profiling":   e.profiler != nil && e.config.EnableProfiling,
		"vulnScan":    e.vulnScanner != nil && e.config.EnableVulnScan,
		"nameRes":     e.wiredCollector != nil,
		"correlation": true, // Always available
	}
}

// Subscribe subscribes to discovery events.
func (e *Engine) Subscribe(filter *EventFilter, handler EventHandler) *Subscription {
	return e.eventBus.Subscribe(filter, handler)
}

// SubscribeAll subscribes to all events.
func (e *Engine) SubscribeAll(handler EventHandler) *Subscription {
	return e.eventBus.SubscribeAll(handler)
}

// Unsubscribe removes an event subscription.
func (e *Engine) Unsubscribe(id string) {
	e.eventBus.Unsubscribe(id)
}

// Registry returns the device registry for advanced operations.
func (e *Engine) Registry() *DeviceRegistry {
	return e.registry
}

// EventBus returns the event bus for advanced operations.
func (e *Engine) EventBus() *EventBus {
	return e.eventBus
}

// ensureConnectionType ensures a connection type is in the slice.
func ensureConnectionType(types []ConnectionType, t ConnectionType) []ConnectionType {
	if slices.Contains(types, t) {
		return types
	}
	return append(types, t)
}
