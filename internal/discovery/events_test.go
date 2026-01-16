package discovery_test

import (
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestNewEvent(t *testing.T) {
	event := discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired)

	if event.Type != discovery.EvtDeviceDiscovered {
		t.Errorf("expected type %s, got %s", discovery.EvtDeviceDiscovered, event.Type)
	}
	if event.Source != discovery.SourceWired {
		t.Errorf("expected source %s, got %s", discovery.SourceWired, event.Source)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if event.Changes == nil {
		t.Error("expected non-nil Changes map")
	}
}

func TestEventWithDevice(t *testing.T) {
	device := &discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:FF",
		IP:  "192.168.1.100",
	}
	event := discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired).
		WithDevice(device)

	if event.Device != device {
		t.Error("expected device to be attached")
	}
	if event.DeviceMAC != device.MAC {
		t.Errorf("expected DeviceMAC %s, got %s", device.MAC, event.DeviceMAC)
	}
}

func TestEventWithChange(t *testing.T) {
	event := discovery.NewEvent(discovery.EvtDeviceUpdated, discovery.SourceEngine).
		WithChange("ip", "192.168.1.100").
		WithChange("hostname", "test.local")

	if len(event.Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(event.Changes))
	}
	if event.Changes["ip"] != "192.168.1.100" {
		t.Error("ip change not recorded")
	}
	if event.Changes["hostname"] != "test.local" {
		t.Error("hostname change not recorded")
	}
}

func TestEventFilterMatches(t *testing.T) {
	tests := []struct {
		name     string
		filter   discovery.EventFilter
		event    *discovery.Event
		expected bool
	}{
		{
			name:     "empty filter matches all",
			filter:   discovery.EventFilter{},
			event:    discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired),
			expected: true,
		},
		{
			name: "type filter matches",
			filter: discovery.EventFilter{
				Types: []discovery.EventType{
					discovery.EvtDeviceDiscovered,
					discovery.EvtDeviceUpdated,
				},
			},
			event:    discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired),
			expected: true,
		},
		{
			name: "type filter does not match",
			filter: discovery.EventFilter{
				Types: []discovery.EventType{discovery.EvtDeviceUpdated},
			},
			event:    discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired),
			expected: false,
		},
		{
			name: "source filter matches",
			filter: discovery.EventFilter{
				Sources: []discovery.EventSource{discovery.SourceWired, discovery.SourceWiFi},
			},
			event:    discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired),
			expected: true,
		},
		{
			name: "source filter does not match",
			filter: discovery.EventFilter{
				Sources: []discovery.EventSource{discovery.SourceBluetooth},
			},
			event:    discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired),
			expected: false,
		},
		{
			name: "MAC filter matches",
			filter: discovery.EventFilter{
				DeviceMACs: []string{"aa:bb:cc:dd:ee:ff"},
			},
			event: discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired).
				WithMAC("AA:BB:CC:DD:EE:FF"),
			expected: true,
		},
		{
			name: "MAC filter does not match",
			filter: discovery.EventFilter{
				DeviceMACs: []string{"11:22:33:44:55:66"},
			},
			event: discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired).
				WithMAC("AA:BB:CC:DD:EE:FF"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Matches(tt.event)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEventBusSubscribeAndPublish(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0}) // sync delivery
	defer eb.Stop()

	var received *discovery.Event
	var mu sync.Mutex
	done := make(chan struct{})

	sub := eb.SubscribeAll(func(e *discovery.Event) {
		mu.Lock()
		received = e
		mu.Unlock()
		close(done)
	})

	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}
	if sub.ID() == "" {
		t.Error("expected non-empty subscription ID")
	}
	if !sub.IsActive() {
		t.Error("expected subscription to be active")
	}

	event := discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired)
	eb.Publish(event)

	// Wait for delivery
	select {
	case <-done:
		mu.Lock()
		if received != event {
			t.Error("did not receive expected event")
		}
		mu.Unlock()
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestEventBusSubscribeTypes(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	var received []*discovery.Event
	var mu sync.Mutex
	done := make(chan struct{}, 2)

	eb.SubscribeTypes(
		[]discovery.EventType{discovery.EvtDeviceDiscovered},
		func(e *discovery.Event) {
			mu.Lock()
			received = append(received, e)
			mu.Unlock()
			done <- struct{}{}
		},
	)

	// Publish matching event
	eb.Publish(discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired))
	// Publish non-matching event
	eb.Publish(discovery.NewEvent(discovery.EvtDeviceUpdated, discovery.SourceWired))

	// Wait for first event
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}

	// Give time for any additional events
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Errorf("expected 1 event, got %d", len(received))
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	var count int
	var mu sync.Mutex

	sub := eb.SubscribeAll(func(_ *discovery.Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	// Publish first event
	eb.Publish(discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired))
	time.Sleep(50 * time.Millisecond)

	// Unsubscribe
	eb.Unsubscribe(sub.ID())

	// Publish second event (should not be received)
	eb.Publish(discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired))
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 event received, got %d", count)
	}
}

func TestSubscriptionCancel(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 0})
	defer eb.Stop()

	sub := eb.SubscribeAll(func(_ *discovery.Event) {})

	if !sub.IsActive() {
		t.Error("subscription should be active")
	}

	sub.Cancel()

	if sub.IsActive() {
		t.Error("subscription should be inactive after cancel")
	}
}

func TestEventBusStats(t *testing.T) {
	eb := discovery.NewEventBus(&discovery.EventBusConfig{BufferSize: 100})
	defer eb.Stop()

	eb.SubscribeAll(func(_ *discovery.Event) {})
	eb.SubscribeAll(func(_ *discovery.Event) {})

	stats := eb.Stats()

	if stats.SubscriberCount != 2 {
		t.Errorf("expected 2 subscribers, got %d", stats.SubscriberCount)
	}
	if stats.BufferSize != 100 {
		t.Errorf("expected buffer size 100, got %d", stats.BufferSize)
	}
}

func TestConvenienceEventFunctions(t *testing.T) {
	device := &discovery.DiscoveredDevice{
		MAC: "AA:BB:CC:DD:EE:FF",
		IP:  "192.168.1.100",
	}

	t.Run("NewDeviceDiscoveredEvent", func(t *testing.T) {
		event := discovery.NewDeviceDiscoveredEvent(discovery.SourceWired, device)
		if event.Type != discovery.EvtDeviceDiscovered {
			t.Errorf("expected type %s, got %s", discovery.EvtDeviceDiscovered, event.Type)
		}
		if event.Device != device {
			t.Error("device not attached")
		}
	})

	t.Run("NewDeviceUpdatedEvent", func(t *testing.T) {
		changes := map[string]any{"ip": "192.168.1.200"}
		event := discovery.NewDeviceUpdatedEvent(discovery.SourceEngine, device, changes)
		if event.Type != discovery.EvtDeviceUpdated {
			t.Errorf("expected type %s, got %s", discovery.EvtDeviceUpdated, event.Type)
		}
		if len(event.Changes) != 1 {
			t.Errorf("expected 1 change, got %d", len(event.Changes))
		}
	})

	t.Run("NewDeviceLostEvent", func(t *testing.T) {
		event := discovery.NewDeviceLostEvent(discovery.SourceEngine, device.MAC)
		if event.Type != discovery.EvtDeviceLost {
			t.Errorf("expected type %s, got %s", discovery.EvtDeviceLost, event.Type)
		}
		if event.DeviceMAC != device.MAC {
			t.Errorf("expected MAC %s, got %s", device.MAC, event.DeviceMAC)
		}
	})

	t.Run("NewScanStartedEvent", func(t *testing.T) {
		event := discovery.NewScanStartedEvent("full")
		if event.Type != discovery.EventScanStarted {
			t.Errorf("expected type %s, got %s", discovery.EventScanStarted, event.Type)
		}
	})

	t.Run("NewScanCompletedEvent", func(t *testing.T) {
		event := discovery.NewScanCompletedEvent("quick", 10, 5*time.Second)
		if event.Type != discovery.EventScanCompleted {
			t.Errorf("expected type %s, got %s", discovery.EventScanCompleted, event.Type)
		}
	})
}
