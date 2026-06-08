package discovery_test

// stages_test.go covers the stage seam introduced in ADR-0018 (Phase 6): the
// pure device converters and each stage's nil-component safety. Behavioural
// equivalence with the prior inline phases is additionally guarded by the
// engine scan tests (engine_test.go) and the golden HTTP harness.

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

func newTestRegistry() *discovery.DeviceRegistry {
	return discovery.NewDeviceRegistry(
		discovery.NewEventBus(&discovery.EventBusConfig{}),
		&discovery.RegistryConfig{EmitEvents: false},
	)
}

func TestWiFiAPToDeviceMapsFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	ap := &discovery.WiFiAccessPoint{
		BSSID: "aa:bb:cc:dd:ee:ff", Vendor: "Acme", SSIDName: "net",
		Channel: 6, ChannelWidth: 40, FrequencyMHz: 2437, SignalDBm: -50,
		IsAuthorized: true, Band: discovery.WiFiBand("2.4GHz"), LastSeen: now,
	}
	d := discovery.ExportWiFiAPToDevice(ap)
	if d.MAC != ap.BSSID || d.Vendor != "Acme" {
		t.Fatalf("identity not mapped: mac=%q vendor=%q", d.MAC, d.Vendor)
	}
	if len(d.ConnectionTypes) != 1 || d.ConnectionTypes[0] != discovery.ConnectionWiFi {
		t.Fatalf("connection type = %v, want [wifi]", d.ConnectionTypes)
	}
	if d.WiFiPresence == nil || !d.WiFiPresence.IsAccessPoint || d.WiFiPresence.SSID != "net" ||
		d.WiFiPresence.Channel != 6 || d.WiFiPresence.Band != "2.4GHz" {
		t.Fatalf("wifi presence not mapped: %+v", d.WiFiPresence)
	}
}

func TestBluetoothDeviceToDeviceMapsFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	bt := &discovery.BluetoothDevice{
		Address: "11:22:33:44:55:66", Vendor: "BT Inc", Name: "phone",
		RSSI: -60, TxPower: 4, IsPaired: true, IsConnected: true, IsAuthorized: true,
		ServiceUUIDs: []string{"180f"}, LastSeen: now,
	}
	d := discovery.ExportBluetoothDeviceToDevice(bt)
	if d.MAC != bt.Address || d.Vendor != "BT Inc" {
		t.Fatalf("identity not mapped: mac=%q vendor=%q", d.MAC, d.Vendor)
	}
	if len(d.ConnectionTypes) != 1 || d.ConnectionTypes[0] != discovery.ConnectionBluetooth {
		t.Fatalf("connection type = %v, want [bluetooth]", d.ConnectionTypes)
	}
	if d.BluetoothPresence == nil || d.BluetoothPresence.Name != "phone" ||
		!d.BluetoothPresence.IsPaired || len(d.BluetoothPresence.Services) != 1 {
		t.Fatalf("bluetooth presence not mapped: %+v", d.BluetoothPresence)
	}
}

func TestEnumerateStageNilCollectorsIsNoop(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	stage := discovery.NewEnumerateStageForTest(reg, discovery.DefaultEngineConfig())
	// All collectors nil; opts ask for everything. Gating must skip each source.
	if err := stage.Enumerate(context.Background(), discovery.DefaultFullScanOpts()); err != nil {
		t.Fatalf("nil-collector enumerate returned error: %v", err)
	}
	if got := len(reg.GetDevices()); got != 0 {
		t.Fatalf("expected no devices, got %d", got)
	}
}

func TestEnrichStageNilComponentsIsNoop(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry()
	reg.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "de:ad:be:ef:00:01", IP: "10.0.0.1"})
	stage := discovery.NewEnrichStageForTest(reg, discovery.DefaultEngineConfig())
	stats := &discovery.ScanStats{}

	// No SNMP/port/profiler components: must not panic and must enrich nothing.
	stage.Enrich(context.Background(), discovery.DefaultFullScanOpts(), stats)

	if stats.EnrichedDevices != 0 {
		t.Fatalf("EnrichedDevices = %d, want 0 with nil components", stats.EnrichedDevices)
	}
	d := reg.GetDevice("de:ad:be:ef:00:01")
	if d == nil || d.SNMPData != nil || d.Profile != nil {
		t.Fatalf("device unexpectedly enriched: %+v", d)
	}
}

// The assess stage's nil-safety test moved to internal/discovery/vuln with the
// stage (TestStageNilScannerIsNoop).
