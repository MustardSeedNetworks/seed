package discovery

// This file is only compiled during testing (due to _test.go suffix)
// and provides access to internal implementation details.

import (
	"net"

	"github.com/MustardSeedNetworks/seed/internal/discovery/resolve"
)

// ExportIncrementIP exposes incrementIP for testing.
func ExportIncrementIP(ip net.IP, n int) net.IP {
	return incrementIP(ip, n)
}

// ExportNormalizeMac exposes normalizeMac for testing.
func ExportNormalizeMac(mac string) string {
	return normalizeMac(mac)
}

// ExportGuessOSFromTTL exposes guessOSFromTTL for testing.
func ExportGuessOSFromTTL(ttl int) string {
	return guessOSFromTTL(ttl)
}

// ExportSplitSubnetIntoChunks exposes splitSubnetIntoChunks for testing.
func ExportSplitSubnetIntoChunks(subnet *net.IPNet, maxChunks int) []*net.IPNet {
	return splitSubnetIntoChunks(subnet, maxChunks)
}

// ARPScannerTestAccessor provides access to ARPScanner's private fields for testing.
type ARPScannerTestAccessor struct {
	Scanner *ARPScanner
}

// GetInterfaceName returns the scanner's interface name.
func (a *ARPScannerTestAccessor) GetInterfaceName() string {
	return a.Scanner.interfaceName
}

// GetOUI returns the scanner's OUI database.
func (a *ARPScannerTestAccessor) GetOUI() *resolve.OUIDatabase {
	return a.Scanner.oui
}

// GetEntries returns the scanner's entries map.
func (a *ARPScannerTestAccessor) GetEntries() map[string]*ARPEntry {
	return a.Scanner.entries
}

// GetSubnet returns the scanner's subnet.
func (a *ARPScannerTestAccessor) GetSubnet() *net.IPNet {
	return a.Scanner.subnet
}

// SetSubnet sets the scanner's subnet.
func (a *ARPScannerTestAccessor) SetSubnet(subnet *net.IPNet) {
	a.Scanner.subnet = subnet
}

// GetLocalIP returns the scanner's local IP.
func (a *ARPScannerTestAccessor) GetLocalIP() net.IP {
	return a.Scanner.localIP
}

// IsScanning returns the scanner's scanning state.
func (a *ARPScannerTestAccessor) IsScanning() bool {
	return a.Scanner.scanning
}

// SetScanning sets the scanner's scanning state.
func (a *ARPScannerTestAccessor) SetScanning(scanning bool) {
	a.Scanner.scanning = scanning
}

// Lock locks the scanner's mutex.
func (a *ARPScannerTestAccessor) Lock() {
	a.Scanner.mu.Lock()
}

// Unlock unlocks the scanner's mutex.
func (a *ARPScannerTestAccessor) Unlock() {
	a.Scanner.mu.Unlock()
}

// IsInSubnet exposes the private isInSubnet method for testing.
func (s *ARPScanner) IsInSubnet(ip string) bool {
	return s.isInSubnet(ip)
}

// TracerTestAccessor provides access to Tracer's private fields for testing.
type TracerTestAccessor struct {
	Tracer *Tracer
}

// GetTimeout returns the tracer's timeout.
func (t *TracerTestAccessor) GetTimeout() any {
	return t.Tracer.timeout
}

// GetMaxHops returns the tracer's maxHops.
func (t *TracerTestAccessor) GetMaxHops() int {
	return t.Tracer.maxHops
}

// DeviceDiscoveryTestAccessor provides access to DeviceDiscovery's private fields for testing.
type DeviceDiscoveryTestAccessor struct {
	Discovery *DeviceDiscovery
}

// GetProtoManager returns the discovery's protocol manager.
func (d *DeviceDiscoveryTestAccessor) GetProtoManager() *Manager {
	return d.Discovery.protoManager
}

// ExportIsLocallyAdministeredMAC exposes isLocallyAdministeredMAC for testing.
func ExportIsLocallyAdministeredMAC(mac string) bool {
	return isLocallyAdministeredMAC(mac)
}

// ExportEnsureConnectionType exposes ensureConnectionType for testing.
func ExportEnsureConnectionType(types []ConnectionType, add ConnectionType) []ConnectionType {
	return ensureConnectionType(types, add)
}

// ExportWiFiAPToDevice exposes wifiAPToDevice (ADR-0018 enumerate stage) for testing.
func ExportWiFiAPToDevice(ap *WiFiAccessPoint) *DiscoveredDevice {
	return wifiAPToDevice(ap)
}

// ExportBluetoothDeviceToDevice exposes bluetoothDeviceToDevice for testing.
func ExportBluetoothDeviceToDevice(bt *BluetoothDevice) *DiscoveredDevice {
	return bluetoothDeviceToDevice(bt)
}

// NewEnumerateStageForTest builds the enumerate stage (no collectors) as its
// port, for nil-safety testing (ADR-0018).
func NewEnumerateStageForTest(reg *DeviceRegistry, cfg *EngineConfig) Enumerator {
	return &enumerateStage{registry: reg, config: cfg}
}

// NewEnrichStageForTest builds the enrich stage (no components) as its port.
func NewEnrichStageForTest(reg *DeviceRegistry, cfg *EngineConfig) Enricher {
	return &enrichStage{registry: reg, config: cfg}
}

// EngineTestAccessor provides access to Engine's private fields for testing.
type EngineTestAccessor struct {
	Engine *Engine
}

// GetRegistry returns the engine's registry.
func (e *EngineTestAccessor) GetRegistry() *DeviceRegistry {
	return e.Engine.registry
}

// GetEventBus returns the engine's eventBus.
func (e *EngineTestAccessor) GetEventBus() *EventBus {
	return e.Engine.eventBus
}
