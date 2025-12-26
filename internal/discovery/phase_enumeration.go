// Package discovery implements multi-protocol network device discovery.
// This file implements Phase 1 (Enumeration) of the discovery pipeline.
// It wraps existing scanners to provide comprehensive device enumeration.
package discovery

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// EnumerationPhase implements the Phase interface for device enumeration.
// Phase 1 discovers ALL devices on local and target subnets using:
//   - ARP scan (local subnet - Layer 2)
//   - ICMP ping sweep (additional subnets - Layer 3)
//   - NDP discovery (IPv6)
//   - Passive LLDP/CDP/EDP data merge
type EnumerationPhase struct {
	deviceDiscovery *DeviceDiscovery
	config          *EnumerationConfig
	broadcaster     EventBroadcaster
}

// EnumerationConfig controls Phase 1 behavior.
type EnumerationConfig struct {
	// ARPScan enables ARP scanning for local subnet discovery.
	ARPScan bool `yaml:"arp_scan" json:"arpScan"`

	// ICMPScan enables ICMP ping sweep for remote subnets.
	ICMPScan bool `yaml:"icmp_scan" json:"icmpScan"`

	// NDPScan enables IPv6 Neighbor Discovery Protocol scanning.
	NDPScan bool `yaml:"ndp_scan" json:"ndpScan"`

	// PassiveProtocols controls which passive discovery protocols are active.
	PassiveProtocols struct {
		LLDP bool `yaml:"lldp" json:"lldp"`
		CDP  bool `yaml:"cdp" json:"cdp"`
		EDP  bool `yaml:"edp" json:"edp"`
	} `yaml:"passive_protocols" json:"passiveProtocols"`

	// Timing controls scan rate limiting.
	Timing EnumerationTiming `yaml:"timing" json:"timing"`

	// AdditionalSubnets to scan beyond local subnet.
	AdditionalSubnets []string `yaml:"additional_subnets" json:"additionalSubnets"`
}

// EnumerationTiming controls scan delays and concurrency.
type EnumerationTiming struct {
	// PingTimeout for individual ICMP requests.
	PingTimeout time.Duration `yaml:"ping_timeout" json:"pingTimeout"`

	// ScanTimeout for the entire enumeration phase.
	ScanTimeout time.Duration `yaml:"scan_timeout" json:"scanTimeout"`

	// PingWorkers for concurrent ICMP pinging.
	PingWorkers int `yaml:"ping_workers" json:"pingWorkers"`

	// ARPDelay between ARP probes (rate limiting).
	ARPDelay time.Duration `yaml:"arp_delay" json:"arpDelay"`
}

// DefaultEnumerationConfig returns default enumeration settings.
func DefaultEnumerationConfig() *EnumerationConfig {
	return &EnumerationConfig{
		ARPScan:  true,
		ICMPScan: true,
		NDPScan:  true,
		PassiveProtocols: struct {
			LLDP bool `yaml:"lldp" json:"lldp"`
			CDP  bool `yaml:"cdp" json:"cdp"`
			EDP  bool `yaml:"edp" json:"edp"`
		}{
			LLDP: true,
			CDP:  true,
			EDP:  true,
		},
		Timing: EnumerationTiming{
			PingTimeout: 1 * time.Second,
			ScanTimeout: 5 * time.Minute,
			PingWorkers: 50,
			ARPDelay:    0, // No delay by default (ARP is very fast)
		},
	}
}

// NewEnumerationPhase creates a new Phase 1 implementation.
func NewEnumerationPhase(deviceDiscovery *DeviceDiscovery, config *EnumerationConfig, broadcaster EventBroadcaster) *EnumerationPhase {
	if config == nil {
		config = DefaultEnumerationConfig()
	}
	return &EnumerationPhase{
		deviceDiscovery: deviceDiscovery,
		config:          config,
		broadcaster:     broadcaster,
	}
}

// Name returns the phase name.
func (p *EnumerationPhase) Name() string {
	return "enumeration"
}

// Run executes the enumeration phase.
// This is the primary device discovery phase that finds all devices.
func (p *EnumerationPhase) Run(ctx context.Context, _ []*DiscoveredDevice, progressCh chan<- PhaseProgressPayload) ([]*DiscoveredDevice, error) {
	start := time.Now()
	slog.Info("Enumeration phase starting",
		"arp", p.config.ARPScan,
		"icmp", p.config.ICMPScan,
		"ndp", p.config.NDPScan)

	// Use scan timeout if configured
	scanCtx := ctx
	if p.config.Timing.ScanTimeout > 0 {
		var cancel context.CancelFunc
		scanCtx, cancel = context.WithTimeout(ctx, p.config.Timing.ScanTimeout)
		defer cancel()
	}

	// Track progress
	var progress EnumerationProgress
	progress.Start()

	// Run parallel discovery methods
	var wg sync.WaitGroup

	// 1. ARP scan for local subnet (Layer 2)
	if p.config.ARPScan {
		wg.Add(1)
		go func() {
			defer wg.Done()
			progress.SetPhase("arp_scan")
			if err := p.runARPScan(scanCtx, &progress); err != nil {
				slog.Warn("ARP scan failed", "error", err)
				progress.AddError("arp_scan", err)
			}
			progress.MarkComplete("arp_scan")
		}()
	}

	// 2. ICMP ping sweep for additional subnets (Layer 3)
	if p.config.ICMPScan && len(p.config.AdditionalSubnets) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			progress.SetPhase("icmp_scan")
			if err := p.runICMPScan(scanCtx, &progress); err != nil {
				slog.Warn("ICMP scan failed", "error", err)
				progress.AddError("icmp_scan", err)
			}
			progress.MarkComplete("icmp_scan")
		}()
	}

	// 3. NDP scan for IPv6 (if enabled)
	if p.config.NDPScan {
		wg.Add(1)
		go func() {
			defer wg.Done()
			progress.SetPhase("ndp_scan")
			// NDP is handled by the background NDP scanner
			// Just trigger a quick refresh
			progress.MarkComplete("ndp_scan")
		}()
	}

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
						Phase:           "enumeration",
						ProcessedCount:  int(progress.DevicesFound()),
						TotalCount:      progress.EstimatedTotal(),
						PercentComplete: progress.PercentComplete(),
						CurrentTarget:   progress.CurrentTarget(),
						ElapsedMs:       time.Since(start).Milliseconds(),
					}
				}
			}
		}
	}()

	// Wait for all scans to complete
	wg.Wait()
	close(done)

	// Aggregate all results
	devices := p.deviceDiscovery.GetDevices()

	// Broadcast device discoveries
	for _, device := range devices {
		if p.broadcaster != nil {
			p.broadcaster.BroadcastPipelineEvent(PipelineEvent{
				Type:      EventDeviceDiscovered,
				Timestamp: time.Now(),
				Payload: DeviceDiscoveredPayload{
					IP:      device.IP,
					MAC:     device.MAC,
					Vendor:  device.Vendor,
					Methods: methodsToStrings(device.DiscoveryMethod),
					IsNew:   true, // TODO: Track new vs existing
				},
			})
		}
	}

	slog.Info("Enumeration phase completed",
		"devices", len(devices),
		"duration", time.Since(start))

	return devices, nil
}

// runARPScan performs ARP-based Layer 2 discovery.
func (p *EnumerationPhase) runARPScan(ctx context.Context, progress *EnumerationProgress) error {
	// Use the existing device discovery scan which includes ARP
	if err := p.deviceDiscovery.Scan(ctx); err != nil {
		return err
	}

	// Count discovered devices
	devices := p.deviceDiscovery.GetDevices()
	for _, d := range devices {
		if containsMethod(d.DiscoveryMethod, MethodARP) {
			progress.IncrementDevices()
		}
	}

	return nil
}

// runICMPScan performs ICMP ping sweep on additional subnets.
func (p *EnumerationPhase) runICMPScan(_ context.Context, _ *EnumerationProgress) error {
	if len(p.config.AdditionalSubnets) == 0 {
		return nil
	}

	// Set additional subnets in the scanner
	if err := p.deviceDiscovery.SetAdditionalSubnets(p.config.AdditionalSubnets); err != nil {
		return err
	}

	// The ARP scan already handles additional subnets with ICMP
	// This is handled in arp.go's pingSweep method
	return nil
}

// EnumerationProgress tracks progress during enumeration.
type EnumerationProgress struct {
	mu            sync.RWMutex
	startTime     time.Time
	currentPhase  string
	currentTarget string
	devicesFound  int64
	errors        []EnumerationError
	completed     map[string]bool
}

// EnumerationError records an error during enumeration.
type EnumerationError struct {
	Phase string
	Error error
	Time  time.Time
}

// Start initializes progress tracking.
func (p *EnumerationProgress) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startTime = time.Now()
	p.completed = make(map[string]bool)
}

// SetPhase updates the current phase.
func (p *EnumerationProgress) SetPhase(phase string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentPhase = phase
}

// SetCurrentTarget updates the target being scanned.
func (p *EnumerationProgress) SetCurrentTarget(target string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentTarget = target
}

// CurrentTarget returns the current scan target.
func (p *EnumerationProgress) CurrentTarget() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentTarget
}

// IncrementDevices adds to the device count.
func (p *EnumerationProgress) IncrementDevices() {
	atomic.AddInt64(&p.devicesFound, 1)
}

// DevicesFound returns the number of devices discovered.
func (p *EnumerationProgress) DevicesFound() int64 {
	return atomic.LoadInt64(&p.devicesFound)
}

// EstimatedTotal returns estimated total devices (for progress calculation).
func (p *EnumerationProgress) EstimatedTotal() int {
	// Rough estimate based on typical /24 subnet
	return 254
}

// PercentComplete returns completion percentage.
func (p *EnumerationProgress) PercentComplete() float64 {
	found := p.DevicesFound()
	if found == 0 {
		return 0
	}
	// We don't know total, so estimate based on typical discovery rate
	// After finding devices, we're "mostly done"
	return float64(found) / float64(p.EstimatedTotal()) * 100
}

// AddError records an error.
func (p *EnumerationProgress) AddError(phase string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errors = append(p.errors, EnumerationError{
		Phase: phase,
		Error: err,
		Time:  time.Now(),
	})
}

// MarkComplete marks a sub-phase as complete.
func (p *EnumerationProgress) MarkComplete(phase string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completed[phase] = true
}

// Errors returns all recorded errors.
func (p *EnumerationProgress) Errors() []EnumerationError {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]EnumerationError{}, p.errors...)
}

// methodsToStrings converts Method slice to string slice.
func methodsToStrings(methods []Method) []string {
	result := make([]string, len(methods))
	for i, m := range methods {
		result[i] = string(m)
	}
	return result
}

// EnhancedEnumerator provides more thorough enumeration capabilities.
// It extends the basic enumeration with additional techniques.
type EnhancedEnumerator struct {
	deviceDiscovery *DeviceDiscovery
	pinger          *ICMPPinger
	config          *EnhancedEnumerationConfig
}

// EnhancedEnumerationConfig extends EnumerationConfig with additional options.
type EnhancedEnumerationConfig struct {
	EnumerationConfig

	// MultiPassARP enables multiple ARP scan passes for reliability.
	MultiPassARP bool `yaml:"multi_pass_arp" json:"multiPassArp"`

	// ARPPasses is the number of ARP scan passes (if MultiPassARP is true).
	ARPPasses int `yaml:"arp_passes" json:"arpPasses"`

	// GratuitousARP captures gratuitous ARP announcements.
	GratuitousARP bool `yaml:"gratuitous_arp" json:"gratuitousArp"`

	// SlowScan uses longer delays between probes (IDS-friendly).
	SlowScan bool `yaml:"slow_scan" json:"slowScan"`

	// RetryUnresponsive retries devices that didn't respond initially.
	RetryUnresponsive bool `yaml:"retry_unresponsive" json:"retryUnresponsive"`

	// RetryCount for unresponsive devices.
	RetryCount int `yaml:"retry_count" json:"retryCount"`
}

// NewEnhancedEnumerator creates an enhanced enumerator.
func NewEnhancedEnumerator(deviceDiscovery *DeviceDiscovery, config *EnhancedEnumerationConfig) (*EnhancedEnumerator, error) {
	pinger, err := NewICMPPinger(config.Timing.PingTimeout)
	if err != nil {
		// Non-fatal - ICMP may require elevated privileges
		slog.Warn("ICMP pinger unavailable", "error", err)
	}

	return &EnhancedEnumerator{
		deviceDiscovery: deviceDiscovery,
		pinger:          pinger,
		config:          config,
	}, nil
}

// EnumerateSubnet performs comprehensive enumeration of a subnet.
func (e *EnhancedEnumerator) EnumerateSubnet(ctx context.Context, cidr string) ([]*DiscoveredDevice, error) {
	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	slog.Info("Enhanced enumeration starting", "subnet", cidr)

	// Generate all host IPs in subnet
	hosts := generateHostIPs(subnet)
	slog.Debug("Host IPs generated", "count", len(hosts))

	// Multi-pass ARP if enabled
	if e.config.MultiPassARP && e.config.ARPPasses > 1 {
		for pass := range e.config.ARPPasses {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			slog.Debug("ARP pass", "pass", pass+1, "total", e.config.ARPPasses)

			// Add delay between passes for IDS-friendly scanning
			if e.config.SlowScan && pass > 0 {
				select {
				case <-time.After(2 * time.Second):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}

			if err := e.deviceDiscovery.Scan(ctx); err != nil {
				slog.Warn("ARP pass failed", "pass", pass+1, "error", err)
			}
		}
	} else {
		// Single pass
		if err := e.deviceDiscovery.Scan(ctx); err != nil {
			return nil, err
		}
	}

	// Retry unresponsive hosts
	if e.config.RetryUnresponsive && e.pinger != nil {
		e.retryUnresponsive(ctx, hosts)
	}

	return e.deviceDiscovery.GetDevices(), nil
}

// retryUnresponsive pings hosts that weren't discovered.
func (e *EnhancedEnumerator) retryUnresponsive(ctx context.Context, hosts []net.IP) {
	discovered := make(map[string]bool)
	for _, d := range e.deviceDiscovery.GetDevices() {
		discovered[d.IP] = true
	}

	var unresponsive []net.IP
	for _, ip := range hosts {
		if !discovered[ip.String()] {
			unresponsive = append(unresponsive, ip)
		}
	}

	if len(unresponsive) == 0 {
		return
	}

	slog.Debug("Retrying unresponsive hosts",
		"count", len(unresponsive),
		"retries", e.config.RetryCount)

	for range e.config.RetryCount {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Ping sweep the unresponsive hosts
		results := e.pinger.PingSweep(ctx, unresponsive, e.config.Timing.PingWorkers)

		// Filter out now-responsive hosts
		var stillUnresponsive []net.IP
		for _, r := range results {
			if !r.Reachable {
				ip := net.ParseIP(r.IP)
				if ip != nil {
					stillUnresponsive = append(stillUnresponsive, ip)
				}
			}
		}
		unresponsive = stillUnresponsive

		if len(unresponsive) == 0 {
			break
		}

		// Delay between retries
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

// Close releases resources.
func (e *EnhancedEnumerator) Close() error {
	if e.pinger != nil {
		return e.pinger.Close()
	}
	return nil
}

// generateHostIPs returns all host IPs in a subnet (excluding network and broadcast).
func generateHostIPs(subnet *net.IPNet) []net.IP {
	ones, bits := subnet.Mask.Size()
	numHosts := 1<<(bits-ones) - 2

	// Limit to reasonable size
	if numHosts > 65534 {
		numHosts = 65534
	}

	baseIP := subnet.IP.Mask(subnet.Mask).To4()
	if baseIP == nil {
		return nil
	}

	hosts := make([]net.IP, 0, numHosts)
	for i := 1; i <= numHosts; i++ {
		ip := incrementIP(baseIP, i)
		if ip != nil {
			hosts = append(hosts, ip)
		}
	}

	return hosts
}
