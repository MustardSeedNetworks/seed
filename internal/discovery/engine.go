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
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
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
}

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

	// Start auto-scan if configured
	if e.config.AutoScanInterval > 0 {
		e.wg.Add(1)
		go e.autoScanLoop(ctx)
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

// runScanPhases executes all scan phases in order.
func (e *Engine) runScanPhases(ctx context.Context, logger *slog.Logger, opts *ScanOptions, result *ScanResult) {
	// Phase 1: Discovery
	logger.InfoContext(ctx, "Starting discovery phase")
	result.Phases = append(result.Phases, "discovery")
	if err := e.runDiscoveryPhase(ctx, opts, result.Stats); err != nil {
		result.Error = err.Error()
		logger.ErrorContext(ctx, "Discovery phase failed", "error", err)
	}

	// Phase 2: Correlation
	logger.InfoContext(ctx, "Starting correlation phase")
	result.Phases = append(result.Phases, "correlation")
	e.correlateDevices(ctx)

	// Phase 3: Name Resolution
	if opts.IncludeNameRes && e.wiredCollector != nil {
		logger.InfoContext(ctx, "Starting name resolution phase")
		result.Phases = append(result.Phases, "name_resolution")
		e.wiredCollector.ResolveNetBIOSNames(ctx)
		e.wiredCollector.ResolveMDNSNames(ctx)
	}

	// Phase 4: Enrichment
	if opts.IncludeSNMP || opts.IncludePortScan || opts.IncludeProfiling {
		logger.InfoContext(ctx, "Starting enrichment phase")
		result.Phases = append(result.Phases, "enrichment")
		e.runEnrichmentPhase(ctx, opts, result.Stats)
	}

	// Phase 5: Assessment
	if opts.IncludeVulnScan && e.vulnScanner != nil {
		logger.InfoContext(ctx, "Starting assessment phase")
		result.Phases = append(result.Phases, "assessment")
		e.runAssessmentPhase(ctx, result.Stats)
	}
}

// QuickScan performs a quick correlation-only scan.
func (e *Engine) QuickScan(ctx context.Context) (*ScanResult, error) {
	return e.Scan(ctx, DefaultQuickScanOpts())
}

// FullScan performs a comprehensive full scan.
func (e *Engine) FullScan(ctx context.Context) (*ScanResult, error) {
	return e.Scan(ctx, DefaultFullScanOpts())
}

// runWiredDiscovery performs wired device discovery.
func (e *Engine) runWiredDiscovery(ctx context.Context, opts *ScanOptions) error {
	if opts.FreshWiredScan {
		if err := e.wiredCollector.Scan(ctx); err != nil {
			return fmt.Errorf("wired scan: %w", err)
		}
	}
	for _, device := range e.wiredCollector.GetDevices() {
		device.ConnectionTypes = ensureConnectionType(device.ConnectionTypes, ConnectionWired)
		e.registry.AddOrUpdate(device)
	}
	return nil
}

// runWiFiDiscovery performs WiFi device discovery.
func (e *Engine) runWiFiDiscovery(ctx context.Context, opts *ScanOptions) error {
	if opts.FreshWiFiScan {
		if _, err := e.wifiCollector.Scan(ctx); err != nil {
			return fmt.Errorf("wifi scan: %w", err)
		}
	}
	aps := e.wifiCollector.GetAccessPoints()
	for i := range aps {
		device := e.wifiAPToDevice(&aps[i])
		e.registry.AddOrUpdate(device)
	}
	return nil
}

// runBluetoothDiscovery performs Bluetooth device discovery.
func (e *Engine) runBluetoothDiscovery(ctx context.Context, opts *ScanOptions) error {
	if opts.FreshBluetoothScan {
		if _, err := e.bluetoothCollector.Scan(ctx); err != nil {
			return fmt.Errorf("bluetooth scan: %w", err)
		}
	}
	scanResult := e.bluetoothCollector.GetLastScan()
	if scanResult != nil {
		for i := range scanResult.Devices {
			device := e.bluetoothDeviceToDevice(&scanResult.Devices[i])
			e.registry.AddOrUpdate(device)
		}
	}
	return nil
}

// runDiscoveryPhase collects devices from all enabled sources.
func (e *Engine) runDiscoveryPhase(ctx context.Context, opts *ScanOptions, _ *ScanStats) error {
	var wg sync.WaitGroup
	errCh := make(chan error, discoveryPhaseChannelBuffer)

	// Wired discovery
	if opts.IncludeWired && e.wiredCollector != nil && e.config.EnableWired {
		wg.Go(func() {
			if err := e.runWiredDiscovery(ctx, opts); err != nil {
				errCh <- err
			}
		})
	}

	// WiFi discovery
	if opts.IncludeWiFi && e.wifiCollector != nil && e.config.EnableWiFi {
		wg.Go(func() {
			if err := e.runWiFiDiscovery(ctx, opts); err != nil {
				errCh <- err
			}
		})
	}

	// Bluetooth discovery
	if opts.IncludeBluetooth && e.bluetoothCollector != nil && e.config.EnableBluetooth {
		wg.Go(func() {
			if err := e.runBluetoothDiscovery(ctx, opts); err != nil {
				errCh <- err
			}
		})
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// correlateDevices merges devices seen on multiple networks.
// Since we use MAC as the primary key, correlation happens automatically
// in AddOrUpdate. This method handles any additional correlation logic.
func (e *Engine) correlateDevices(_ context.Context) {
	// The registry already correlates by MAC on AddOrUpdate.
	// This method can be extended for additional correlation strategies
	// (e.g., IP-based correlation, hostname matching, etc.)
}

// runEnrichmentPhase performs SNMP, port scanning, and profiling.
func (e *Engine) runEnrichmentPhase(ctx context.Context, opts *ScanOptions, stats *ScanStats) {
	devices := e.registry.GetDevices()

	for _, device := range devices {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// SNMP enrichment
		if opts.IncludeSNMP && e.snmpCollector != nil && e.config.EnableSNMP {
			if snmpData := e.collectSNMPData(ctx, device); snmpData != nil {
				device.SNMPData = snmpData
				stats.EnrichedDevices++
			}
		}

		// Port scanning
		if opts.IncludePortScan && e.portScanner != nil && e.config.EnablePortScan {
			if profile := e.scanPorts(ctx, device); profile != nil {
				device.Profile = profile
			}
		}

		// Profiling
		if opts.IncludeProfiling && e.profiler != nil && e.config.EnableProfiling {
			e.profileDevice(ctx, device)
		}

		// Update device in registry
		e.registry.AddOrUpdate(device)
	}
}

// runAssessmentPhase performs vulnerability scanning.
func (e *Engine) runAssessmentPhase(ctx context.Context, stats *ScanStats) {
	devices := e.registry.GetDevices()

	for _, device := range devices {
		select {
		case <-ctx.Done():
			return
		default:
		}

		vulns := e.assessVulnerabilities(ctx, device)
		if vulns != nil && len(vulns.Vulnerabilities) > 0 {
			device.Vulnerabilities = vulns
			stats.VulnerableDevices++
			e.registry.AddOrUpdate(device)

			// Emit vulnerability event for each finding
			for _, v := range vulns.Vulnerabilities {
				e.eventBus.Publish(NewVulnDiscoveredEvent(device, v.CVEID, v.Severity))
			}
		}
	}
}

// collectSNMPData collects SNMP data for a device.
func (e *Engine) collectSNMPData(ctx context.Context, device *DiscoveredDevice) *SNMPFullData {
	if device.IP == "" || e.snmpCollector == nil {
		return nil
	}

	// Try SNMP collection (collector handles v2c and v3)
	data, err := e.snmpCollector.Collect(ctx, device.IP)
	if err != nil {
		return nil
	}
	return data
}

// scanPorts scans ports on a device.
func (e *Engine) scanPorts(ctx context.Context, device *DiscoveredDevice) *DeviceProfile {
	if device.IP == "" || e.portScanner == nil {
		return nil
	}

	result := e.portScanner.QuickScan(ctx, device.IP)
	if result == nil || result.Error != "" {
		return nil
	}

	// Convert ServiceInfo to OpenPort
	openPorts := make([]OpenPort, 0, len(result.Services))
	for _, svc := range result.Services {
		openPorts = append(openPorts, OpenPort{
			Port:     svc.Port,
			Protocol: svc.Protocol,
			Service:  svc.Service,
			Banner:   svc.Banner,
			IsOpen:   svc.State == "open",
		})
	}

	profile := &DeviceProfile{
		OpenPorts: openPorts,
	}
	return profile
}

// profileDevice performs device profiling.
func (e *Engine) profileDevice(_ context.Context, device *DiscoveredDevice) {
	if e.profiler == nil || device.IP == "" {
		return
	}

	// Queue the device for profiling (async)
	if err := e.profiler.QueueProfile(device.IP); err != nil {
		return
	}

	// Check if profile is already available (from previous profiling)
	if profile := e.profiler.GetProfile(device.IP); profile != nil {
		device.Profile = profile
	}
}

// assessVulnerabilities checks a device for vulnerabilities.
func (e *Engine) assessVulnerabilities(ctx context.Context, device *DiscoveredDevice) *DeviceVulnerabilities {
	if e.vulnScanner == nil {
		return nil
	}

	vulns, err := e.vulnScanner.ScanDevice(ctx, device)
	if err != nil {
		return nil
	}
	return vulns
}

// wifiAPToDevice converts a WiFi access point to a DiscoveredDevice.
func (e *Engine) wifiAPToDevice(ap *WiFiAccessPoint) *DiscoveredDevice {
	device := &DiscoveredDevice{
		MAC:             ap.BSSID,
		Vendor:          ap.Vendor,
		DiscoveryMethod: []Method{},
		ConnectionTypes: []ConnectionType{ConnectionWiFi},
		WiFiPresence: &WiFiPresence{
			SSID:          ap.SSIDName,
			Channel:       ap.Channel,
			ChannelWidth:  ap.ChannelWidth,
			FrequencyMHz:  ap.FrequencyMHz,
			SignalDBm:     ap.SignalDBm,
			IsAccessPoint: true,
			IsAuthorized:  ap.IsAuthorized,
			Band:          string(ap.Band),
			LastSeen:      ap.LastSeen,
		},
		LastSeen: ap.LastSeen,
	}
	return device
}

// bluetoothDeviceToDevice converts a Bluetooth device to a DiscoveredDevice.
func (e *Engine) bluetoothDeviceToDevice(bt *BluetoothDevice) *DiscoveredDevice {
	device := &DiscoveredDevice{
		MAC:             bt.Address,
		Vendor:          bt.Vendor,
		DiscoveryMethod: []Method{},
		ConnectionTypes: []ConnectionType{ConnectionBluetooth},
		BluetoothPresence: &BluetoothPresence{
			Name:         bt.Name,
			Type:         bt.Type,
			DeviceClass:  bt.DeviceClass,
			RSSI:         bt.RSSI,
			TxPower:      bt.TxPower,
			IsPaired:     bt.IsPaired,
			IsConnected:  bt.IsConnected,
			IsAuthorized: bt.IsAuthorized,
			Services:     bt.ServiceUUIDs,
			LastSeen:     bt.LastSeen,
		},
		LastSeen: bt.LastSeen,
	}
	return device
}

// autoScanLoop runs periodic scans.
func (e *Engine) autoScanLoop(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.AutoScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _ = e.QuickScan(ctx)
		case <-e.stopCh:
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
