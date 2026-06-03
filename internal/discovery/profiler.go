package discovery

// Package discovery's device profiler performs deep inspection of discovered
// devices through HTTP, SNMP, mDNS, and port scanning to gather detailed
// information about capabilities, services, and device types.
//
// profiler.go holds the DeviceProfiler struct, NewDeviceProfiler, the
// Start/Stop/worker lifecycle, the QueueProfile entry point, and the
// shutdown / context helpers. The types, config, active probes (scan / HTTP
// / SNMP), name resolution, classification, and read-only accessors each
// live in sibling profiler_*.go files.

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
)

// DeviceProfiler automatically profiles newly discovered devices.
type DeviceProfiler struct {
	config          *ProfilerConfig
	snmpConfig      *config.SNMPConfig
	snmpCollector   *SNMPCollector   // For full MIB collection
	netbiosResolver *NetBIOSResolver // For NetBIOS name queries
	mdnsResolver    *MDNSResolver    // For mDNS name queries
	httpClient      *http.Client
	transport       *http.Transport // Store transport for cleanup (fixes #825)
	mu              sync.RWMutex
	profiles        map[string]*DeviceProfile // key by IP
	snmpData        map[string]*SNMPFullData  // Full SNMP MIB data by IP
	resolvedNames   map[string]*ResolvedNames // Resolved names by IP
	profiling       map[string]bool           // track in-progress profiles
	queue           chan string               // IPs to profile
	stopCh          chan struct{}
	wg              sync.WaitGroup
	interfaceName   string // For mDNS resolver
}

// NewDeviceProfiler creates a new device profiler.
func NewDeviceProfiler(cfg *ProfilerConfig, snmpCfg *config.SNMPConfig) *DeviceProfiler {
	if cfg == nil {
		cfg = DefaultProfilerConfig()
	}

	transport := &http.Transport{
		TLSClientConfig: newProfilerTLSConfig(cfg.SkipTLSVerify),
		DialContext: (&net.Dialer{
			Timeout: cfg.Timeout,
		}).DialContext,
	}

	p := &DeviceProfiler{
		config:     cfg,
		snmpConfig: snmpCfg,
		transport:  transport, // Store for cleanup (fixes #825)
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
		profiles:      make(map[string]*DeviceProfile),
		snmpData:      make(map[string]*SNMPFullData),
		resolvedNames: make(map[string]*ResolvedNames),
		profiling:     make(map[string]bool),
		queue:         make(chan string, profilerQueueSize),
	}

	// Initialize SNMP collector if credentials are configured and collection is enabled
	if cfg.EnableSNMPCollection && snmpCfg != nil && p.hasSNMPCredentials() {
		mibConfig := SNMPMIBSelection{
			System:      true,
			Interfaces:  true,
			IPAddresses: true,
			Bridge:      true,
			VLAN:        true,
			LLDP:        true,
		}
		if cfg.SNMPMIBs != nil {
			mibConfig = *cfg.SNMPMIBs
		}
		p.snmpCollector = NewSNMPCollector(snmpCfg, mibConfig)
		logging.GetLogger().Info("SNMP collector initialized for automatic MIB polling",
			"communities", len(snmpCfg.Communities),
			"v3creds", len(snmpCfg.V3Credentials))
	}

	// Initialize name resolvers if enabled
	if cfg.EnableNameResolution {
		if cfg.ResolveNetBIOS {
			p.netbiosResolver = NewNetBIOSResolver()
		}
		if cfg.ResolveMDNS {
			p.mdnsResolver = NewMDNSResolver("") // Interface set later via SetInterface
		}
		logging.GetLogger().Info("Name resolution initialized",
			"dns", cfg.ResolveDNS,
			"netbios", cfg.ResolveNetBIOS,
			"mdns", cfg.ResolveMDNS)
	}

	return p
}

// Start begins the profiler worker pool.
func (p *DeviceProfiler) Start() {
	p.mu.Lock()
	if p.stopCh != nil {
		p.mu.Unlock()
		return // Already running
	}
	p.stopCh = make(chan struct{})

	// Add to WaitGroup before unlocking to prevent race with Stop()
	p.wg.Add(p.config.MaxConcurrent)
	p.mu.Unlock()

	// Start worker goroutines
	for range p.config.MaxConcurrent {
		go p.worker()
	}

	logging.GetLogger().Info("Device profiler started", "workers", p.config.MaxConcurrent)
}

// Stop stops the profiler.
func (p *DeviceProfiler) Stop() {
	p.mu.Lock()
	if p.stopCh == nil {
		p.mu.Unlock()
		return
	}
	close(p.stopCh)
	p.stopCh = nil
	p.mu.Unlock()

	p.wg.Wait()

	// Close idle connections to prevent resource leak (fixes #825)
	if p.transport != nil {
		p.transport.CloseIdleConnections()
	}

	logging.GetLogger().Info("Device profiler stopped")
}

// worker processes profile requests from the queue.
// Fixes #981: Added nil check and defensive handling for stopCh.
func (p *DeviceProfiler) worker() {
	defer p.wg.Done()

	// Capture stopCh locally to avoid race with Stop() setting it to nil
	p.mu.Lock()
	stopCh := p.stopCh
	queue := p.queue
	p.mu.Unlock()

	// Fixes #981: If stopCh is nil, worker should exit immediately
	if stopCh == nil {
		return
	}

	for {
		select {
		case <-stopCh:
			return
		case ip, ok := <-queue:
			// Fixes #981: Handle queue close (ok=false means channel closed)
			if !ok {
				return
			}
			p.profileDevice(ip)
		}
	}
}

// QueueProfile adds an IP to the profiling queue.
// Fixes #888: Returns error to distinguish between skip reasons.
// Fixes #930: Check if profiler is started before accepting queue items.
func (p *DeviceProfiler) QueueProfile(ip string) error {
	if !p.config.Enabled {
		logging.GetLogger().Debug("QueueProfile skipped - profiler disabled", "ip", ip)
		return errors.New("profiler disabled")
	}
	if ip == "" {
		return errors.New("empty IP address")
	}

	p.mu.Lock()
	// Fixes #930: Check if profiler is started
	if p.stopCh == nil {
		p.mu.Unlock()
		logging.GetLogger().Debug("QueueProfile skipped - profiler not started", "ip", ip)
		return errors.New("profiler not started")
	}
	// Skip if already profiled or in progress
	if _, exists := p.profiles[ip]; exists {
		p.mu.Unlock()
		logging.GetLogger().Debug("QueueProfile skipped - already profiled", "ip", ip)
		return nil // Not an error, just already done
	}
	if p.profiling[ip] {
		p.mu.Unlock()
		logging.GetLogger().Debug("QueueProfile skipped - in progress", "ip", ip)
		return nil // Not an error, already in progress
	}
	p.profiling[ip] = true
	p.mu.Unlock()

	select {
	case p.queue <- ip:
		logging.GetLogger().Info("Queued device for profiling", "ip", ip)
		return nil
	default:
		// Queue full, skip (fixes #888)
		p.mu.Lock()
		delete(p.profiling, ip)
		p.mu.Unlock()
		logging.GetLogger().Warn("Profile queue full, skipped", "ip", ip)
		return errors.New("profile queue full")
	}
}

// checkShutdown checks if the profiler is shutting down.
// Returns the stopCh if active, or nil if shutdown is in progress/complete.
func (p *DeviceProfiler) checkShutdown() chan struct{} {
	p.mu.RLock()
	stopCh := p.stopCh
	p.mu.RUnlock()

	if stopCh == nil {
		return nil
	}

	select {
	case <-stopCh:
		return nil
	default:
		return stopCh
	}
}

// createProfilingContext creates a context for profiling that respects shutdown signals.
// Returns the context, a cancel function, and a cleanup function that must be called when done.
func (p *DeviceProfiler) createProfilingContext(
	stopCh chan struct{},
) (context.Context, context.CancelFunc, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), profilerDefaultTimeoutS*time.Second)

	// Create cancellable context that respects shutdown (fixes #828)
	ctx, cancelWithShutdown := context.WithCancel(ctx)
	go func() {
		select {
		case <-stopCh:
			cancelWithShutdown()
		case <-ctx.Done():
		}
	}()

	cleanup := func() {
		cancelWithShutdown()
		cancel()
	}

	return ctx, cancelWithShutdown, cleanup
}
