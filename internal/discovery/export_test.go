package discovery

// This file is only compiled during testing (due to _test.go suffix)
// and provides access to internal implementation details.
//
// The wired-collector accessors (ARPScanner/DeviceDiscovery/Manager,
// normalizeMac/incrementIP/guessOSFromTTL/splitSubnetIntoChunks/
// isLocallyAdministeredMAC) moved to internal/discovery/enumerate/export_test.go
// alongside the collector (ADR-0018 Phase 6).

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
