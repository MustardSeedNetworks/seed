package discovery

// stages.go expresses the discovery scan as four explicit pipeline stages behind
// ports (ADR-0018, Phase 6): enumerate -> resolve -> fingerprint -> vuln. These
// are the existing Engine.runScanPhases boundaries, lifted into stage types so
// the Engine orchestrates ports rather than inlining each phase. The stage types
// and the Engine still live in this package; subsequent PRs relocate each stage
// into its own subpackage (with depguard enforcing the one-way direction) — at
// which point these ports are the stable seam the orchestrator depends on.
//
// Behaviour is unchanged: each stage holds the same components the Engine held
// and runs the same logic, reading from / writing to the shared DeviceRegistry.

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Enumerator is stage 1: discover hosts and links from the enabled sources
// (wired/Wi-Fi/Bluetooth), writing each device into the registry.
type Enumerator interface {
	Enumerate(ctx context.Context, opts *ScanOptions) error
}

// Resolver is stage 2: attach identity/naming (NetBIOS, mDNS) to discovered
// devices already in the registry.
type Resolver interface {
	Resolve(ctx context.Context)
}

// Enricher is stage 3 (the engine's "enrichment" phase, a.k.a. fingerprint):
// actively probe registry devices for services, OS, and SNMP detail, recording
// the results back on each device. Named Enricher rather than Fingerprinter
// because the latter is already an advanced-probing capability used within this
// stage.
type Enricher interface {
	Enrich(ctx context.Context, opts *ScanOptions, stats *ScanStats)
}

// Assessor is stage 4: evaluate registry devices for vulnerabilities, recording
// findings and emitting a discovery event per finding.
type Assessor interface {
	Assess(ctx context.Context, stats *ScanStats)
}

// PortScannerPort is the fingerprint stage's port-scan seam (ADR-0018, Phase 6):
// a quick scan of a host's common ports, returning the open services. The
// concrete scanner lives in internal/discovery/fingerprint; the composition root
// injects it so the enrich stage depends only on this narrow port, never on the
// stage subpackage. PortScanResult stays kernel-side (portscan_types.go) so it
// sits in this signature without inverting the dependency.
type PortScannerPort interface {
	QuickScan(ctx context.Context, target string) *PortScanResult
}

// --- Stage 1: enumerate ------------------------------------------------------

// enumerateStage runs the discovery sources concurrently and writes results to
// the registry. Per-source gating (opts + collector presence + config enable)
// matches the Engine's prior behaviour exactly.
type enumerateStage struct {
	registry           *DeviceRegistry
	config             *EngineConfig
	wiredCollector     *DeviceDiscovery
	wifiCollector      *WiFiBridge
	bluetoothCollector *BluetoothScanner
}

func (s *enumerateStage) Enumerate(ctx context.Context, opts *ScanOptions) error {
	var wg sync.WaitGroup
	errCh := make(chan error, discoveryPhaseChannelBuffer)

	if opts.IncludeWired && s.wiredCollector != nil && s.config.EnableWired {
		wg.Go(func() {
			if err := s.enumerateWired(ctx, opts); err != nil {
				errCh <- err
			}
		})
	}
	if opts.IncludeWiFi && s.wifiCollector != nil && s.config.EnableWiFi {
		wg.Go(func() {
			if err := s.enumerateWiFi(ctx, opts); err != nil {
				errCh <- err
			}
		})
	}
	if opts.IncludeBluetooth && s.bluetoothCollector != nil && s.config.EnableBluetooth {
		wg.Go(func() {
			if err := s.enumerateBluetooth(ctx, opts); err != nil {
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

func (s *enumerateStage) enumerateWired(ctx context.Context, opts *ScanOptions) error {
	if opts.FreshWiredScan {
		if err := s.wiredCollector.Scan(ctx); err != nil {
			return fmt.Errorf("wired scan: %w", err)
		}
	}
	for _, device := range s.wiredCollector.GetDevices() {
		device.ConnectionTypes = ensureConnectionType(device.ConnectionTypes, ConnectionWired)
		s.registry.AddOrUpdate(device)
	}
	return nil
}

func (s *enumerateStage) enumerateWiFi(ctx context.Context, opts *ScanOptions) error {
	if opts.FreshWiFiScan {
		if _, err := s.wifiCollector.Scan(ctx); err != nil {
			return fmt.Errorf("wifi scan: %w", err)
		}
	}
	aps := s.wifiCollector.GetAccessPoints()
	for i := range aps {
		s.registry.AddOrUpdate(wifiAPToDevice(&aps[i]))
	}
	return nil
}

func (s *enumerateStage) enumerateBluetooth(ctx context.Context, opts *ScanOptions) error {
	if opts.FreshBluetoothScan {
		if _, err := s.bluetoothCollector.Scan(ctx); err != nil {
			return fmt.Errorf("bluetooth scan: %w", err)
		}
	}
	scanResult := s.bluetoothCollector.GetLastScan()
	if scanResult != nil {
		for i := range scanResult.Devices {
			s.registry.AddOrUpdate(bluetoothDeviceToDevice(&scanResult.Devices[i]))
		}
	}
	return nil
}

// --- Stage 2: resolve --------------------------------------------------------

// resolveStage attaches names via the wired collector's resolvers.
type resolveStage struct {
	wiredCollector *DeviceDiscovery
}

func (s *resolveStage) Resolve(ctx context.Context) {
	s.wiredCollector.ResolveNetBIOSNames(ctx)
	s.wiredCollector.ResolveMDNSNames(ctx)
}

// --- Stage 3: fingerprint (enrichment) ---------------------------------------

// enrichStage enriches each registry device with SNMP, port-scan, and profiler
// results.
type enrichStage struct {
	registry      *DeviceRegistry
	config        *EngineConfig
	snmpCollector *SNMPCollector
	portScanner   PortScannerPort
	profiler      *DeviceProfiler
}

func (s *enrichStage) Enrich(ctx context.Context, opts *ScanOptions, stats *ScanStats) {
	for _, device := range s.registry.GetDevices() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if opts.IncludeSNMP && s.snmpCollector != nil && s.config.EnableSNMP {
			if snmpData := s.collectSNMP(ctx, device); snmpData != nil {
				device.SNMPData = snmpData
				stats.EnrichedDevices++
			}
		}
		if opts.IncludePortScan && s.portScanner != nil && s.config.EnablePortScan {
			if profile := s.scanPorts(ctx, device); profile != nil {
				device.Profile = profile
			}
		}
		if opts.IncludeProfiling && s.profiler != nil && s.config.EnableProfiling {
			s.profileDevice(device)
		}

		s.registry.AddOrUpdate(device)
	}
}

func (s *enrichStage) collectSNMP(ctx context.Context, device *DiscoveredDevice) *SNMPFullData {
	if device.IP == "" || s.snmpCollector == nil {
		return nil
	}
	data, err := s.snmpCollector.Collect(ctx, device.IP)
	if err != nil {
		return nil
	}
	return data
}

func (s *enrichStage) scanPorts(ctx context.Context, device *DiscoveredDevice) *DeviceProfile {
	if device.IP == "" || s.portScanner == nil {
		return nil
	}
	result := s.portScanner.QuickScan(ctx, device.IP)
	if result == nil || result.Error != "" {
		return nil
	}
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
	return &DeviceProfile{OpenPorts: openPorts}
}

func (s *enrichStage) profileDevice(device *DiscoveredDevice) {
	if s.profiler == nil || device.IP == "" {
		return
	}
	if err := s.profiler.QueueProfile(device.IP); err != nil {
		return
	}
	if profile := s.profiler.GetProfile(device.IP); profile != nil {
		device.Profile = profile
	}
}

// --- Stage 4: vuln (assess) --------------------------------------------------
//
// The assess stage's implementation lives in the internal/discovery/vuln
// subpackage (it owns the scanner + CVE providers); it satisfies the Assessor
// port above and is injected into the Engine by the composition root. The kernel
// keeps only the port, not the concrete stage (ADR-0018, Phase 6).

// --- shared converters (pure transforms; package-level so any stage can use) --

// wifiAPToDevice converts a Wi-Fi access point to a DiscoveredDevice.
func wifiAPToDevice(ap *WiFiAccessPoint) *DiscoveredDevice {
	return &DiscoveredDevice{
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
}

// bluetoothDeviceToDevice converts a Bluetooth device to a DiscoveredDevice.
func bluetoothDeviceToDevice(bt *BluetoothDevice) *DiscoveredDevice {
	return &DiscoveredDevice{
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
}
