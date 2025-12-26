// Package discovery implements multi-protocol network device discovery.
// This file implements Phase 2 (Name Resolution) of the discovery pipeline.
// It performs DNS, NetBIOS, and mDNS name resolution for discovered devices.
package discovery

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ResolutionPhase implements the Phase interface for name resolution.
// Phase 2 enriches discovered devices with hostnames using:
//   - Reverse DNS (PTR records)
//   - NetBIOS name queries (Windows devices - UDP 137)
//   - mDNS queries (Apple/Linux devices - UDP 5353)
type ResolutionPhase struct {
	netbiosResolver *NetBIOSResolver
	mdnsResolver    *MDNSResolver
	config          *ResolutionConfig
	broadcaster     EventBroadcaster
}

// ResolutionConfig controls Phase 2 behavior.
type ResolutionConfig struct {
	// DNS enables reverse DNS (PTR) lookups.
	DNS bool `yaml:"dns" json:"dns"`

	// NetBIOS enables NetBIOS name resolution for Windows devices.
	NetBIOS bool `yaml:"netbios" json:"netbios"`

	// MDNS enables mDNS name resolution for Apple/Linux devices.
	MDNS bool `yaml:"mdns" json:"mdns"`

	// Timing controls resolution delays and concurrency.
	Timing ResolutionTiming `yaml:"timing" json:"timing"`
}

// ResolutionTiming controls name resolution rate limiting.
type ResolutionTiming struct {
	// DNSTimeout for individual DNS lookups.
	DNSTimeout time.Duration `yaml:"dns_timeout" json:"dnsTimeout"`

	// NetBIOSTimeout for individual NetBIOS queries.
	NetBIOSTimeout time.Duration `yaml:"netbios_timeout" json:"netbiosTimeout"`

	// MDNSTimeout for individual mDNS queries.
	MDNSTimeout time.Duration `yaml:"mdns_timeout" json:"mdnsTimeout"`

	// PhaseTimeout for the entire resolution phase.
	PhaseTimeout time.Duration `yaml:"phase_timeout" json:"phaseTimeout"`

	// MaxConcurrentDNS limits parallel DNS lookups.
	MaxConcurrentDNS int `yaml:"max_concurrent_dns" json:"maxConcurrentDns"`

	// MaxConcurrentNetBIOS limits parallel NetBIOS queries.
	MaxConcurrentNetBIOS int `yaml:"max_concurrent_netbios" json:"maxConcurrentNetbios"`

	// MaxConcurrentMDNS limits parallel mDNS queries.
	MaxConcurrentMDNS int `yaml:"max_concurrent_mdns" json:"maxConcurrentMdns"`
}

// DefaultResolutionConfig returns default resolution settings.
func DefaultResolutionConfig() *ResolutionConfig {
	return &ResolutionConfig{
		DNS:     true,
		NetBIOS: true,
		MDNS:    true,
		Timing: ResolutionTiming{
			DNSTimeout:           500 * time.Millisecond,
			NetBIOSTimeout:       500 * time.Millisecond,
			MDNSTimeout:          2 * time.Second,
			PhaseTimeout:         5 * time.Minute,
			MaxConcurrentDNS:     50,
			MaxConcurrentNetBIOS: 20,
			MaxConcurrentMDNS:    10,
		},
	}
}

// NewResolutionPhase creates a new Phase 2 implementation.
func NewResolutionPhase(interfaceName string, config *ResolutionConfig, broadcaster EventBroadcaster) *ResolutionPhase {
	if config == nil {
		config = DefaultResolutionConfig()
	}
	return &ResolutionPhase{
		netbiosResolver: NewNetBIOSResolver(),
		mdnsResolver:    NewMDNSResolver(interfaceName),
		config:          config,
		broadcaster:     broadcaster,
	}
}

// Name returns the phase name.
func (p *ResolutionPhase) Name() string {
	return "resolution"
}

// Run executes the name resolution phase.
// Devices from Phase 1 are enriched with hostnames.
//
//nolint:gocyclo // Multi-protocol resolution requires coordinating DNS, NetBIOS, and mDNS.
func (p *ResolutionPhase) Run(ctx context.Context, devices []*DiscoveredDevice, progressCh chan<- PhaseProgressPayload) ([]*DiscoveredDevice, error) {
	start := time.Now()
	slog.Info("Resolution phase starting",
		"devices", len(devices),
		"dns", p.config.DNS,
		"netbios", p.config.NetBIOS,
		"mdns", p.config.MDNS)

	if len(devices) == 0 {
		return devices, nil
	}

	// Use phase timeout if configured
	resolveCtx := ctx
	if p.config.Timing.PhaseTimeout > 0 {
		var cancel context.CancelFunc
		resolveCtx, cancel = context.WithTimeout(ctx, p.config.Timing.PhaseTimeout)
		defer cancel()
	}

	// Track progress
	var progress ResolutionProgress
	progress.Start(len(devices))

	// Collect IPs for resolution
	var ips []string
	deviceByIP := make(map[string]*DiscoveredDevice)
	for _, device := range devices {
		if device.IP != "" {
			ips = append(ips, device.IP)
			deviceByIP[device.IP] = device
		}
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
						Phase:           "resolution",
						ProcessedCount:  int(progress.Resolved()),
						TotalCount:      len(ips),
						PercentComplete: progress.PercentComplete(),
						CurrentTarget:   progress.CurrentTarget(),
						ElapsedMs:       time.Since(start).Milliseconds(),
					}
				}
			}
		}
	}()

	// Run all resolution methods in parallel
	var wg sync.WaitGroup

	// 1. DNS reverse lookup (PTR records)
	if p.config.DNS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.resolveDNS(resolveCtx, ips, deviceByIP, &progress)
		}()
	}

	// 2. NetBIOS name resolution (Windows)
	if p.config.NetBIOS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.resolveNetBIOS(resolveCtx, ips, deviceByIP, &progress)
		}()
	}

	// 3. mDNS name resolution (Apple/Linux)
	if p.config.MDNS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.resolveMDNS(resolveCtx, ips, deviceByIP, &progress)
		}()
	}

	// Wait for all resolution to complete
	wg.Wait()
	close(done)

	// Compute display names for all devices
	for _, device := range devices {
		device.DisplayName = device.ComputeDisplayName()
	}

	// Count resolved names
	resolved := 0
	for _, device := range devices {
		if device.DisplayName != "" && device.DisplayName != device.IP {
			resolved++
		}
	}

	slog.Info("Resolution phase completed",
		"resolved", resolved,
		"total", len(devices),
		"duration", time.Since(start))

	return devices, nil
}

// resolveDNS performs reverse DNS lookups for all devices.
func (p *ResolutionPhase) resolveDNS(ctx context.Context, ips []string, deviceByIP map[string]*DiscoveredDevice, progress *ResolutionProgress) {
	sem := make(chan struct{}, p.config.Timing.MaxConcurrentDNS)
	var wg sync.WaitGroup

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(ipAddr string) {
			defer wg.Done()
			defer func() { <-sem }()

			progress.SetCurrentTarget(ipAddr)

			// Skip if device already has a hostname
			device := deviceByIP[ipAddr]
			if device.Hostname != "" {
				progress.IncrementResolved()
				return
			}

			// Create timeout context for this lookup
			lookupCtx, cancel := context.WithTimeout(ctx, p.config.Timing.DNSTimeout)
			defer cancel()

			names, err := net.DefaultResolver.LookupAddr(lookupCtx, ipAddr)
			if err != nil {
				slog.Debug("DNS lookup failed", "ip", ipAddr, "error", err)
				return
			}

			if len(names) > 0 {
				hostname := strings.TrimSuffix(names[0], ".")
				device.Hostname = hostname
				progress.IncrementResolved()
				slog.Debug("DNS resolved", "ip", ipAddr, "hostname", hostname)
			}
		}(ip)
	}

	wg.Wait()
}

// resolveNetBIOS performs NetBIOS name resolution for Windows devices.
//
//nolint:dupl // Similar to resolveMDNS but uses NetBIOSResolver with NetBIOSResult type.
func (p *ResolutionPhase) resolveNetBIOS(ctx context.Context, ips []string, deviceByIP map[string]*DiscoveredDevice, progress *ResolutionProgress) {
	// Filter IPs that don't already have NetBIOS names
	var toResolve []string
	for _, ip := range ips {
		device := deviceByIP[ip]
		if device.NetBIOSName == "" {
			toResolve = append(toResolve, ip)
		}
	}

	if len(toResolve) == 0 {
		return
	}

	slog.Debug("NetBIOS: resolving names", "count", len(toResolve))

	// Use batch resolution
	results := p.netbiosResolver.ResolveBatch(ctx, toResolve)

	for _, result := range results {
		if result.Err == nil && result.Name != "" {
			if device, ok := deviceByIP[result.IP]; ok {
				device.NetBIOSName = result.Name
				progress.IncrementResolved()
				slog.Debug("NetBIOS resolved", "ip", result.IP, "name", result.Name)
			}
		}
	}
}

// resolveMDNS performs mDNS name resolution for Apple/Linux devices.
//
//nolint:dupl // Similar to resolveNetBIOS but uses MDNSResolver with MDNSResult type.
func (p *ResolutionPhase) resolveMDNS(ctx context.Context, ips []string, deviceByIP map[string]*DiscoveredDevice, progress *ResolutionProgress) {
	// Filter IPs that don't already have mDNS names
	var toResolve []string
	for _, ip := range ips {
		device := deviceByIP[ip]
		if device.MDNSName == "" {
			toResolve = append(toResolve, ip)
		}
	}

	if len(toResolve) == 0 {
		return
	}

	slog.Debug("mDNS: resolving names", "count", len(toResolve))

	// Use batch resolution
	results := p.mdnsResolver.ResolveBatch(ctx, toResolve)

	for _, result := range results {
		if result.Err == nil && result.Name != "" {
			if device, ok := deviceByIP[result.IP]; ok {
				device.MDNSName = result.Name
				progress.IncrementResolved()
				slog.Debug("mDNS resolved", "ip", result.IP, "name", result.Name)
			}
		}
	}
}

// ResolutionProgress tracks progress during name resolution.
type ResolutionProgress struct {
	mu            sync.RWMutex
	startTime     time.Time
	totalDevices  int
	resolved      int64
	currentTarget string
}

// Start initializes progress tracking.
func (p *ResolutionProgress) Start(totalDevices int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startTime = time.Now()
	p.totalDevices = totalDevices
}

// SetCurrentTarget updates the target being resolved.
func (p *ResolutionProgress) SetCurrentTarget(target string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentTarget = target
}

// CurrentTarget returns the current resolution target.
func (p *ResolutionProgress) CurrentTarget() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentTarget
}

// IncrementResolved adds to the resolved count.
func (p *ResolutionProgress) IncrementResolved() {
	atomic.AddInt64(&p.resolved, 1)
}

// Resolved returns the number of devices with resolved names.
func (p *ResolutionProgress) Resolved() int64 {
	return atomic.LoadInt64(&p.resolved)
}

// PercentComplete returns completion percentage.
func (p *ResolutionProgress) PercentComplete() float64 {
	p.mu.RLock()
	total := p.totalDevices
	p.mu.RUnlock()

	if total == 0 {
		return 100
	}
	resolved := p.Resolved()
	return float64(resolved) / float64(total) * 100
}

// DNSResolver wraps the standard library resolver with additional features.
type DNSResolver struct {
	timeout     time.Duration
	maxParallel int
}

// NewDNSResolver creates a new DNS resolver with custom settings.
func NewDNSResolver(timeout time.Duration, maxParallel int) *DNSResolver {
	if timeout == 0 {
		timeout = 500 * time.Millisecond
	}
	if maxParallel == 0 {
		maxParallel = 50
	}
	return &DNSResolver{
		timeout:     timeout,
		maxParallel: maxParallel,
	}
}

// DNSResult represents a DNS lookup result.
type DNSResult struct {
	IP       string
	Hostname string
	Err      error
}

// ResolveBatch performs reverse DNS lookups for multiple IPs concurrently.
func (r *DNSResolver) ResolveBatch(ctx context.Context, ips []string) []DNSResult {
	results := make([]DNSResult, len(ips))
	resultCh := make(chan struct {
		idx    int
		result DNSResult
	}, len(ips))

	sem := make(chan struct{}, r.maxParallel)
	var wg sync.WaitGroup

	for i, ip := range ips {
		wg.Add(1)
		go func(idx int, ipAddr string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				resultCh <- struct {
					idx    int
					result DNSResult
				}{idx, DNSResult{IP: ipAddr, Err: ctx.Err()}}
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			lookupCtx, cancel := context.WithTimeout(ctx, r.timeout)
			defer cancel()

			names, err := net.DefaultResolver.LookupAddr(lookupCtx, ipAddr)
			result := DNSResult{IP: ipAddr, Err: err}
			if err == nil && len(names) > 0 {
				result.Hostname = strings.TrimSuffix(names[0], ".")
			}
			resultCh <- struct {
				idx    int
				result DNSResult
			}{idx, result}
		}(i, ip)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		results[res.idx] = res.result
	}

	return results
}

// ResolveForward performs forward DNS lookup for a hostname.
func (r *DNSResolver) ResolveForward(ctx context.Context, hostname string) ([]string, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return net.DefaultResolver.LookupHost(lookupCtx, hostname)
}

// ResolveReverse performs reverse DNS lookup for an IP.
func (r *DNSResolver) ResolveReverse(ctx context.Context, ip string) (string, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	names, err := net.DefaultResolver.LookupAddr(lookupCtx, ip)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", nil
	}
	return strings.TrimSuffix(names[0], "."), nil
}
