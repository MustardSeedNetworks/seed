package app

// discovery.go wires the composition root to the discovery application (use-case)
// services (ADR-0020 clean-hexagonal): the device-inventory engine, the network
// problem detector, and the Bluetooth scanner. The adapters below implement the
// narrow ports declared in internal/discovery/{devices,problems,bluetooth} over
// the concrete collaborators (the discovery engine, the problem detector, the
// device-discovery service, and the Bluetooth scanner), so the API handlers
// depend on use-cases instead of reaching into the service container directly.
// Collaborators are resolved through lazy accessors on each call so a later-set
// value (the api test harness) is honored, and a nil collaborator degrades the
// use-case to its unavailable behavior rather than panicking.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/bluetooth"
	"github.com/MustardSeedNetworks/seed/internal/discovery/devices"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/discovery/problems"
	"github.com/MustardSeedNetworks/seed/internal/discovery/settings"
)

// NewDiscoveryDevices builds the unified-discovery use-case (ADR-0020) over a lazy
// accessor for the discovery engine. A nil engine makes every method degrade to
// devices.ErrUnavailable (the golden-pinned 503 path).
func NewDiscoveryDevices(eng func() *discovery.Engine) *devices.Service {
	return devices.NewService(discoveryEngineAdapter{eng: eng})
}

// NewProblems builds the problem-detection use-case (ADR-0020) over lazy accessors
// for the problem detector and the device-discovery source. A nil detector makes
// every method degrade to problems.ErrUnavailable; a nil source yields no devices.
func NewProblems(
	detector func() *discovery.ProblemDetector,
	deviceSrc func() *enumerate.Service,
) *problems.Service {
	return problems.NewService(
		problemDetectorAdapter{detector: detector},
		discoveryDeviceSource{deviceSrc: deviceSrc},
	)
}

// NewBluetooth builds the Bluetooth-discovery use-case (ADR-0020) over a lazy
// accessor for the Bluetooth scanner. A nil scanner makes the scan/devices/stats
// methods degrade to bluetooth.ErrUnavailable, while Status reports unavailable.
func NewBluetooth(scanner func() *enumerate.BluetoothScanner) *bluetooth.Service {
	return bluetooth.NewService(bluetoothScannerAdapter{scanner: scanner})
}

// NewDiscoverySettings builds the network-discovery settings service (ADR-0020)
// over the live config (read/merge/persist) and a lazy accessor for the
// device-discovery scanner (subnet re-sync). cfg/path are fixed for the process
// lifetime; the scanner is resolved per call so a later-set value (the api test
// harness) is honored, and a nil scanner makes the subnet sync a no-op.
func NewDiscoverySettings(
	cfg *config.Config, path string, dd func() *enumerate.DeviceDiscovery,
) *settings.Service {
	return settings.NewService(
		discoverySettingsStore{cfg: cfg, path: path},
		discoverySubnetSink{dd: dd},
	)
}

// discoverySettingsStore implements settings.Store over the live config, owning
// the lock + on-disk save the port abstracts away.
type discoverySettingsStore struct {
	cfg  *config.Config
	path string
}

func (s discoverySettingsStore) Discovery() config.NetworkDiscoveryConfig {
	s.cfg.RLock()
	defer s.cfg.RUnlock()
	nd := s.cfg.NetworkDiscovery
	// Copy the subnet slice so the service can mutate its working copy without
	// touching the live config's backing array before SaveDiscovery commits.
	nd.AdditionalSubnets = append([]config.SubnetConfig(nil), nd.AdditionalSubnets...)
	return nd
}

func (s discoverySettingsStore) SaveDiscovery(nd config.NetworkDiscoveryConfig) error {
	// Lock for the in-memory mutation only; Save acquires its own RLock and so
	// must run unlocked to avoid the historic deadlock (fixes #783).
	s.cfg.Lock()
	s.cfg.NetworkDiscovery = nd
	s.cfg.Unlock()
	return s.cfg.Save(s.path)
}

// discoverySubnetSink implements settings.SubnetSink over the device-discovery
// scanner, resolved lazily; a nil scanner is a no-op.
type discoverySubnetSink struct {
	dd func() *enumerate.DeviceDiscovery
}

func (a discoverySubnetSink) SetAdditionalSubnets(cidrs []string) error {
	d := a.dd()
	if d == nil {
		return nil
	}
	return d.SetAdditionalSubnets(cidrs)
}

// discoveryEngineAdapter implements devices.Engine over the discovery engine,
// resolving the engine lazily. Methods beyond Available/Scanning are only invoked
// by the use-case once Available reports true, so they assume a non-nil engine.
type discoveryEngineAdapter struct {
	eng func() *discovery.Engine
}

func (a discoveryEngineAdapter) Available() bool { return a.eng() != nil }
func (a discoveryEngineAdapter) Scanning() bool  { return a.eng().IsScanning() }
func (a discoveryEngineAdapter) Devices() []*discovery.DiscoveredDevice {
	return a.eng().GetDevices()
}

func (a discoveryEngineAdapter) Device(mac string) *discovery.DiscoveredDevice {
	return a.eng().GetDevice(mac)
}
func (a discoveryEngineAdapter) Stats() *discovery.EngineStats   { return a.eng().GetStats() }
func (a discoveryEngineAdapter) LastScan() *discovery.ScanResult { return a.eng().GetLastScan() }
func (a discoveryEngineAdapter) Capabilities() map[string]bool   { return a.eng().GetCapabilities() }

func (a discoveryEngineAdapter) Scan(
	ctx context.Context, opts *discovery.ScanOptions,
) (*discovery.ScanResult, error) {
	return a.eng().Scan(ctx, opts)
}

func (a discoveryEngineAdapter) SubscribeAll(handler func(*discovery.Event)) string {
	return a.eng().SubscribeAll(handler).ID()
}
func (a discoveryEngineAdapter) Unsubscribe(id string) { a.eng().Unsubscribe(id) }

// problemDetectorAdapter implements problems.Detector over the problem detector,
// resolving it lazily.
type problemDetectorAdapter struct {
	detector func() *discovery.ProblemDetector
}

func (a problemDetectorAdapter) Available() bool { return a.detector() != nil }
func (a problemDetectorAdapter) ActiveProblems() []discovery.NetworkProblem {
	return a.detector().GetActiveProblems()
}

func (a problemDetectorAdapter) Summary() *discovery.ProblemSummary {
	return a.detector().GetSummary()
}

func (a problemDetectorAdapter) Scan(
	ctx context.Context, devs []*discovery.DiscoveredDevice,
) (*discovery.ProblemDetectionResult, error) {
	return a.detector().Scan(ctx, devs)
}

func (a problemDetectorAdapter) Thresholds() discovery.ProblemThresholds {
	return a.detector().GetThresholds()
}

func (a problemDetectorAdapter) SetThresholds(t discovery.ProblemThresholds) {
	a.detector().SetThresholds(t)
}

// discoveryDeviceSource implements problems.DeviceSource over the device-discovery
// service, returning no devices when the service is absent.
type discoveryDeviceSource struct {
	deviceSrc func() *enumerate.Service
}

func (a discoveryDeviceSource) Devices() []*discovery.DiscoveredDevice {
	svc := a.deviceSrc()
	if svc == nil {
		return nil
	}
	return svc.GetDevices()
}

// bluetoothScannerAdapter implements bluetooth.Scanner over the Bluetooth scanner,
// resolving it lazily.
type bluetoothScannerAdapter struct {
	scanner func() *enumerate.BluetoothScanner
}

func (a bluetoothScannerAdapter) Available() bool { return a.scanner() != nil }
func (a bluetoothScannerAdapter) Scan(ctx context.Context) (*discovery.BluetoothScanResult, error) {
	return a.scanner().Scan(ctx)
}

func (a bluetoothScannerAdapter) LastScan() *discovery.BluetoothScanResult {
	return a.scanner().GetLastScan()
}

func (a bluetoothScannerAdapter) Stats() *discovery.BluetoothDiscoveryStats {
	return a.scanner().GetStats()
}
