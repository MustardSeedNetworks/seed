package discovery_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestNewDeviceRegistry(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, nil)
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	if registry.Count() != 0 {
		t.Errorf("expected 0 devices, got %d", registry.Count())
	}
}

func TestRegistryAddOrUpdate(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	device := &discovery.DiscoveredDevice{
		MAC:      "AA:BB:CC:DD:EE:FF",
		IP:       "192.168.1.100",
		Hostname: "test.local",
		Vendor:   "TestVendor",
	}

	// Add new device
	result, isNew := registry.AddOrUpdate(device)
	if !isNew {
		t.Error("expected device to be new")
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if registry.Count() != 1 {
		t.Errorf("expected 1 device, got %d", registry.Count())
	}

	// Update existing device
	device2 := &discovery.DiscoveredDevice{
		MAC:      "aa:bb:cc:dd:ee:ff", // same MAC, different case
		IP:       "192.168.1.200",     // new IP
		Hostname: "updated.local",
	}
	result2, isNew2 := registry.AddOrUpdate(device2)
	if isNew2 {
		t.Error("expected device to be updated, not new")
	}
	if result2.IP != "192.168.1.200" {
		t.Errorf("expected IP to be updated, got %s", result2.IP)
	}
	if result2.Hostname != "updated.local" {
		t.Errorf("expected hostname to be updated, got %s", result2.Hostname)
	}
	if registry.Count() != 1 {
		t.Errorf("expected still 1 device, got %d", registry.Count())
	}
}

func TestRegistryAddOrUpdateNil(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, nil)

	// Add nil device
	result, isNew := registry.AddOrUpdate(nil)
	if result != nil || isNew {
		t.Error("expected nil result for nil input")
	}

	// Add device with empty MAC
	result, isNew = registry.AddOrUpdate(&discovery.DiscoveredDevice{IP: "192.168.1.1"})
	if result != nil || isNew {
		t.Error("expected nil result for device without MAC")
	}
}

func TestRegistryGetDevice(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	device := &discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:FF",
		IP:  "192.168.1.100",
	}
	registry.AddOrUpdate(device)

	// Get by MAC (case insensitive)
	result := registry.GetDevice("aa:bb:cc:dd:ee:ff")
	if result == nil {
		t.Error("expected to find device")
	}

	// Get non-existent device
	result = registry.GetDevice("11:22:33:44:55:66")
	if result != nil {
		t.Error("expected nil for non-existent device")
	}
}

func TestRegistryGetDeviceByIP(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	device := &discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:FF",
		IP:  "192.168.1.100",
	}
	registry.AddOrUpdate(device)

	// Get by IP
	result := registry.GetDeviceByIP("192.168.1.100")
	if result == nil {
		t.Error("expected to find device by IP")
	}

	// Non-existent IP
	result = registry.GetDeviceByIP("192.168.1.200")
	if result != nil {
		t.Error("expected nil for non-existent IP")
	}
}

func TestRegistryGetDeviceByHostname(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	device := &discovery.DiscoveredDevice{
		MAC:      "AA:BB:CC:DD:EE:FF",
		Hostname: "Test.Local",
	}
	registry.AddOrUpdate(device)

	// Get by hostname (case insensitive)
	result := registry.GetDeviceByHostname("test.local")
	if result == nil {
		t.Error("expected to find device by hostname")
	}
}

func TestRegistryGetDevices(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	registry.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "AA:BB:CC:DD:EE:01"})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "AA:BB:CC:DD:EE:02"})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "AA:BB:CC:DD:EE:03"})

	devices := registry.GetDevices()
	if len(devices) != 3 {
		t.Errorf("expected 3 devices, got %d", len(devices))
	}
}

func TestRegistryGetDevicesByVendor(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	registry.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "AA:BB:CC:DD:EE:01", Vendor: "Apple"})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "AA:BB:CC:DD:EE:02", Vendor: "Apple"})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "AA:BB:CC:DD:EE:03", Vendor: "Samsung"})

	appleDevices := registry.GetDevicesByVendor("apple")
	if len(appleDevices) != 2 {
		t.Errorf("expected 2 Apple devices, got %d", len(appleDevices))
	}

	samsungDevices := registry.GetDevicesByVendor("Samsung")
	if len(samsungDevices) != 1 {
		t.Errorf("expected 1 Samsung device, got %d", len(samsungDevices))
	}
}

func TestRegistryGetDevicesByConnectionType(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC:             "AA:BB:CC:DD:EE:01",
		ConnectionTypes: []discovery.ConnectionType{discovery.ConnectionWired},
	})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC:             "AA:BB:CC:DD:EE:02",
		ConnectionTypes: []discovery.ConnectionType{discovery.ConnectionWiFi},
	})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:03",
		ConnectionTypes: []discovery.ConnectionType{
			discovery.ConnectionWired,
			discovery.ConnectionBluetooth,
		},
	})

	wiredDevices := registry.GetDevicesByConnectionType(discovery.ConnectionWired)
	if len(wiredDevices) != 2 {
		t.Errorf("expected 2 wired devices, got %d", len(wiredDevices))
	}

	wifiDevices := registry.GetDevicesByConnectionType(discovery.ConnectionWiFi)
	if len(wifiDevices) != 1 {
		t.Errorf("expected 1 WiFi device, got %d", len(wifiDevices))
	}
}

func TestRegistryGetMultiConnectedDevices(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC:             "AA:BB:CC:DD:EE:01",
		ConnectionTypes: []discovery.ConnectionType{discovery.ConnectionWired},
	})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:02",
		ConnectionTypes: []discovery.ConnectionType{
			discovery.ConnectionWired,
			discovery.ConnectionBluetooth,
		},
	})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:03",
		ConnectionTypes: []discovery.ConnectionType{
			discovery.ConnectionWired,
			discovery.ConnectionWiFi,
			discovery.ConnectionBluetooth,
		},
	})

	multiConnected := registry.GetMultiConnectedDevices()
	if len(multiConnected) != 2 {
		t.Errorf("expected 2 multi-connected devices, got %d", len(multiConnected))
	}
}

func TestRegistryRemove(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	device := &discovery.DiscoveredDevice{
		MAC:      "AA:BB:CC:DD:EE:FF",
		IP:       "192.168.1.100",
		Hostname: "test.local",
		Vendor:   "TestVendor",
	}
	registry.AddOrUpdate(device)

	// Remove existing
	if !registry.Remove("aa:bb:cc:dd:ee:ff") {
		t.Error("expected Remove to return true")
	}
	if registry.Count() != 0 {
		t.Error("expected 0 devices after remove")
	}
	if registry.GetDevice("aa:bb:cc:dd:ee:ff") != nil {
		t.Error("expected device to be removed")
	}
	if registry.GetDeviceByIP("192.168.1.100") != nil {
		t.Error("expected IP index to be cleared")
	}

	// Remove non-existent
	if registry.Remove("11:22:33:44:55:66") {
		t.Error("expected Remove to return false for non-existent")
	}
}

func TestRegistryClear(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	registry.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "AA:BB:CC:DD:EE:01"})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{MAC: "AA:BB:CC:DD:EE:02"})

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("expected 0 devices after clear, got %d", registry.Count())
	}
}

func TestRegistryExpireStale(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{
		EmitEvents: false,
		DeviceTTL:  100 * time.Millisecond,
	})

	// Add a device
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC:      "AA:BB:CC:DD:EE:FF",
		LastSeen: time.Now().Add(-200 * time.Millisecond), // already stale
	})

	expired := registry.ExpireStale()
	if expired != 1 {
		t.Errorf("expected 1 expired device, got %d", expired)
	}
	if registry.Count() != 0 {
		t.Error("expected device to be removed")
	}
}

func TestRegistryStats(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC:             "AA:BB:CC:DD:EE:01",
		ConnectionTypes: []discovery.ConnectionType{discovery.ConnectionWired},
	})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC:             "AA:BB:CC:DD:EE:02",
		ConnectionTypes: []discovery.ConnectionType{discovery.ConnectionWiFi},
	})
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:03",
		ConnectionTypes: []discovery.ConnectionType{
			discovery.ConnectionWired,
			discovery.ConnectionBluetooth,
		},
	})

	stats := registry.Stats()
	if stats.TotalDevices != 3 {
		t.Errorf("expected 3 total devices, got %d", stats.TotalDevices)
	}
	if stats.WiredDevices != 2 {
		t.Errorf("expected 2 wired devices, got %d", stats.WiredDevices)
	}
	if stats.WiFiDevices != 1 {
		t.Errorf("expected 1 WiFi device, got %d", stats.WiFiDevices)
	}
	if stats.BTDevices != 1 {
		t.Errorf("expected 1 Bluetooth device, got %d", stats.BTDevices)
	}
	if stats.MultiConnected != 1 {
		t.Errorf("expected 1 multi-connected device, got %d", stats.MultiConnected)
	}
}

func TestRegistryMergeDevice(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: false})

	// Add initial device
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC:             "AA:BB:CC:DD:EE:FF",
		IP:              "192.168.1.100",
		DiscoveryMethod: []discovery.Method{discovery.MethodARP},
		ConnectionTypes: []discovery.ConnectionType{discovery.ConnectionWired},
	})

	// Update with additional data
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC:             "AA:BB:CC:DD:EE:FF",
		Hostname:        "merged.local",
		DiscoveryMethod: []discovery.Method{discovery.MethodLLDP},
		ConnectionTypes: []discovery.ConnectionType{discovery.ConnectionBluetooth},
		BluetoothPresence: &discovery.BluetoothPresence{
			Name: "Test Device",
		},
	})

	device := registry.GetDevice("AA:BB:CC:DD:EE:FF")
	if device == nil {
		t.Fatal("expected to find device")
	}

	// Check merged discovery methods
	if len(device.DiscoveryMethod) != 2 {
		t.Errorf("expected 2 discovery methods, got %d", len(device.DiscoveryMethod))
	}

	// Check merged connection types
	if len(device.ConnectionTypes) != 2 {
		t.Errorf("expected 2 connection types, got %d", len(device.ConnectionTypes))
	}

	// Check hostname was added
	if device.Hostname != "merged.local" {
		t.Errorf("expected hostname 'merged.local', got %s", device.Hostname)
	}

	// Check Bluetooth presence was added
	if device.BluetoothPresence == nil {
		t.Error("expected BluetoothPresence to be merged")
	}

	// Original data should be preserved
	if device.IP != "192.168.1.100" {
		t.Errorf("expected IP to be preserved, got %s", device.IP)
	}
}

func TestRegistryEventsEmitted(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	registry := discovery.NewDeviceRegistry(eb, &discovery.RegistryConfig{EmitEvents: true})

	var events []*discovery.Event
	done := make(chan struct{}, 3)

	eb.SubscribeAll(func(e *discovery.Event) {
		events = append(events, e)
		done <- struct{}{}
	})

	// Add device - should emit discovered event
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:FF",
		IP:  "192.168.1.100",
	})

	// Wait for event
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	// Update device - should emit updated event
	registry.AddOrUpdate(&discovery.DiscoveredDevice{
		MAC:      "AA:BB:CC:DD:EE:FF",
		Hostname: "updated.local",
	})

	// Wait for event
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	// Remove device - should emit lost event
	registry.Remove("AA:BB:CC:DD:EE:FF")

	// Wait for event
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	// Give time for events to be processed
	time.Sleep(50 * time.Millisecond)

	if len(events) < 3 {
		t.Errorf("expected at least 3 events, got %d", len(events))
	}
}
