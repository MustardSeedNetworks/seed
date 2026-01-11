package discovery

// events.go implements the event bus for real-time discovery updates.
//
// The event system enables:
// - Real-time device correlation as discoveries happen
// - UI subscriptions for live updates
// - Decoupled communication between collectors and the registry
// - Audit trail of all discovery activity

import (
	"fmt"
	"sync"
	"time"
)

// EventType identifies the type of discovery event.
type EventType string

// Discovery event types.
// Note: Using "Evt" prefix to avoid conflict with pipeline.go's PipelineEventType constants.
const (
	// Device lifecycle events
	EvtDeviceDiscovered EventType = "device.discovered" // New device found
	EvtDeviceUpdated    EventType = "device.updated"    // Existing device changed
	EvtDeviceLost       EventType = "device.lost"       // Device went offline
	EvtDeviceMerged     EventType = "device.merged"     // Two devices merged (same physical device)

	// Wired discovery events
	EventARPDiscovery  EventType = "wired.arp"  // ARP response received
	EventNDPDiscovery  EventType = "wired.ndp"  // NDP neighbor discovered
	EventLLDPDiscovery EventType = "wired.lldp" // LLDP neighbor discovered
	EventCDPDiscovery  EventType = "wired.cdp"  // CDP neighbor discovered
	EventMDNSDiscovery EventType = "wired.mdns" // mDNS service discovered

	// WiFi discovery events
	EventWiFiAPDiscovered     EventType = "wifi.ap.discovered"     // New AP found
	EventWiFiAPUpdated        EventType = "wifi.ap.updated"        // AP signal/channel changed
	EventWiFiAPLost           EventType = "wifi.ap.lost"           // AP no longer visible
	EventWiFiClientDiscovered EventType = "wifi.client.discovered" // New WiFi client
	EventWiFiClientLost       EventType = "wifi.client.lost"       // WiFi client disconnected

	// Bluetooth discovery events
	EventBTDeviceDiscovered EventType = "bt.device.discovered" // New BT device found
	EventBTDeviceUpdated    EventType = "bt.device.updated"    // BT device changed
	EventBTDeviceLost       EventType = "bt.device.lost"       // BT device out of range

	// Enrichment events
	EventPortDiscovered    EventType = "enrichment.port"    // Open port found
	EventSNMPDataCollected EventType = "enrichment.snmp"    // SNMP data collected
	EventProfileCompleted  EventType = "enrichment.profile" // Device profiling done
	EventNameResolved      EventType = "enrichment.name"    // Hostname resolved

	// Assessment events
	EventVulnDiscovered EventType = "assessment.vuln"     // Vulnerability found
	EventVulnResolved   EventType = "assessment.resolved" // Vulnerability fixed

	// Scan lifecycle events
	EventScanStarted   EventType = "scan.started"   // Scan began
	EventScanProgress  EventType = "scan.progress"  // Scan progress update
	EventScanCompleted EventType = "scan.completed" // Scan finished
	EventScanFailed    EventType = "scan.failed"    // Scan errored
	EventScanCanceled  EventType = "scan.canceled"  // Scan was canceled
)

// EventSource identifies where an event originated.
type EventSource string

// Event sources.
const (
	SourceWired      EventSource = "wired"
	SourceWiFi       EventSource = "wifi"
	SourceBluetooth  EventSource = "bluetooth"
	SourceEnrichment EventSource = "enrichment"
	SourceAssessment EventSource = "assessment"
	SourceEngine     EventSource = "engine"
	SourceAPI        EventSource = "api"
)

// Event represents a discovery event.
type Event struct {
	// Type identifies what happened
	Type EventType `json:"type"`

	// Source identifies where the event originated
	Source EventSource `json:"source"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// DeviceMAC is the primary device MAC (if applicable)
	DeviceMAC string `json:"deviceMac,omitempty"`

	// Device is the full device data (for device events)
	Device *DiscoveredDevice `json:"device,omitempty"`

	// Changes maps field names to their new values (for updates)
	Changes map[string]any `json:"changes,omitempty"`

	// Payload contains event-specific data
	Payload any `json:"payload,omitempty"`

	// Error contains error details (for failure events)
	Error string `json:"error,omitempty"`
}

// NewEvent creates a new event with the current timestamp.
func NewEvent(eventType EventType, source EventSource) *Event {
	return &Event{
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Changes:   make(map[string]any),
	}
}

// WithDevice attaches device information to the event.
func (e *Event) WithDevice(device *DiscoveredDevice) *Event {
	e.Device = device
	if device != nil {
		e.DeviceMAC = device.MAC
	}
	return e
}

// WithMAC sets the device MAC for the event.
func (e *Event) WithMAC(mac string) *Event {
	e.DeviceMAC = mac
	return e
}

// WithChange records a field change.
func (e *Event) WithChange(field string, value any) *Event {
	if e.Changes == nil {
		e.Changes = make(map[string]any)
	}
	e.Changes[field] = value
	return e
}

// WithPayload attaches event-specific data.
func (e *Event) WithPayload(payload any) *Event {
	e.Payload = payload
	return e
}

// WithError attaches error information.
func (e *Event) WithError(err error) *Event {
	if err != nil {
		e.Error = err.Error()
	}
	return e
}

// EventHandler is a callback function for handling events.
type EventHandler func(event *Event)

// EventFilter determines which events a subscriber receives.
type EventFilter struct {
	// Types to include (empty = all types)
	Types []EventType

	// Sources to include (empty = all sources)
	Sources []EventSource

	// DeviceMACs to include (empty = all devices)
	DeviceMACs []string
}

// Matches returns true if the event passes the filter.
func (f *EventFilter) Matches(event *Event) bool {
	// Check type filter
	if len(f.Types) > 0 {
		matched := false
		for _, t := range f.Types {
			if t == event.Type {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check source filter
	if len(f.Sources) > 0 {
		matched := false
		for _, s := range f.Sources {
			if s == event.Source {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check device MAC filter
	if len(f.DeviceMACs) > 0 && event.DeviceMAC != "" {
		matched := false
		normalizedEventMAC := normalizeMAC(event.DeviceMAC)
		for _, mac := range f.DeviceMACs {
			if normalizeMAC(mac) == normalizedEventMAC {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// Subscription represents an active event subscription.
type Subscription struct {
	id      string
	filter  *EventFilter
	handler EventHandler
	active  bool
	mu      sync.RWMutex
}

// ID returns the subscription identifier.
func (s *Subscription) ID() string {
	return s.id
}

// Cancel deactivates the subscription.
func (s *Subscription) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
}

// IsActive returns whether the subscription is still active.
func (s *Subscription) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// EventBus manages event publishing and subscriptions.
type EventBus struct {
	subscriptions map[string]*Subscription
	mu            sync.RWMutex

	// Metrics
	eventCount      int64
	subscriberCount int64

	// Buffer for async delivery (optional)
	buffer     chan *Event
	bufferSize int
	useBuffer  bool

	// Lifecycle
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// EventBusConfig configures the event bus behavior.
type EventBusConfig struct {
	// BufferSize enables async delivery with a buffer (0 = sync delivery)
	BufferSize int

	// DropOnFull drops events when buffer is full (vs blocking)
	DropOnFull bool
}

// DefaultEventBusConfig returns default configuration.
func DefaultEventBusConfig() *EventBusConfig {
	return &EventBusConfig{
		BufferSize: 1000,
		DropOnFull: false,
	}
}

// NewEventBus creates a new event bus.
func NewEventBus(config *EventBusConfig) *EventBus {
	if config == nil {
		config = DefaultEventBusConfig()
	}

	eb := &EventBus{
		subscriptions: make(map[string]*Subscription),
		stopCh:        make(chan struct{}),
	}

	if config.BufferSize > 0 {
		eb.buffer = make(chan *Event, config.BufferSize)
		eb.bufferSize = config.BufferSize
		eb.useBuffer = true

		// Start async delivery goroutine
		eb.wg.Add(1)
		go eb.deliveryLoop()
	}

	return eb
}

// Subscribe registers a handler for events matching the filter.
// Returns a Subscription that can be used to cancel the subscription.
func (eb *EventBus) Subscribe(filter *EventFilter, handler EventHandler) *Subscription {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	id := generateSubscriptionID()
	sub := &Subscription{
		id:      id,
		filter:  filter,
		handler: handler,
		active:  true,
	}

	eb.subscriptions[id] = sub
	eb.subscriberCount++

	return sub
}

// SubscribeAll subscribes to all events.
func (eb *EventBus) SubscribeAll(handler EventHandler) *Subscription {
	return eb.Subscribe(&EventFilter{}, handler)
}

// SubscribeTypes subscribes to specific event types.
func (eb *EventBus) SubscribeTypes(types []EventType, handler EventHandler) *Subscription {
	return eb.Subscribe(&EventFilter{Types: types}, handler)
}

// SubscribeDevice subscribes to events for a specific device.
func (eb *EventBus) SubscribeDevice(mac string, handler EventHandler) *Subscription {
	return eb.Subscribe(&EventFilter{DeviceMACs: []string{mac}}, handler)
}

// Unsubscribe removes a subscription by ID.
func (eb *EventBus) Unsubscribe(id string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if sub, exists := eb.subscriptions[id]; exists {
		sub.Cancel()
		delete(eb.subscriptions, id)
		eb.subscriberCount--
	}
}

// Publish sends an event to all matching subscribers.
func (eb *EventBus) Publish(event *Event) {
	if event == nil {
		return
	}

	eb.mu.RLock()
	eb.eventCount++
	eb.mu.RUnlock()

	if eb.useBuffer {
		// Async delivery via buffer
		select {
		case eb.buffer <- event:
			// Event queued
		default:
			// Buffer full - event dropped (could log this)
		}
	} else {
		// Sync delivery
		eb.deliver(event)
	}
}

// PublishSync delivers an event synchronously, even if buffering is enabled.
func (eb *EventBus) PublishSync(event *Event) {
	if event == nil {
		return
	}

	eb.mu.RLock()
	eb.eventCount++
	eb.mu.RUnlock()

	eb.deliver(event)
}

// deliver sends the event to all matching subscribers.
func (eb *EventBus) deliver(event *Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for _, sub := range eb.subscriptions {
		if !sub.IsActive() {
			continue
		}

		if sub.filter == nil || sub.filter.Matches(event) {
			// Deliver in a goroutine to prevent blocking
			go func(s *Subscription, e *Event) {
				defer func() {
					if r := recover(); r != nil {
						// Handler panicked - log but don't crash
					}
				}()
				s.handler(e)
			}(sub, event)
		}
	}
}

// deliveryLoop processes buffered events.
func (eb *EventBus) deliveryLoop() {
	defer eb.wg.Done()

	for {
		select {
		case event := <-eb.buffer:
			eb.deliver(event)
		case <-eb.stopCh:
			// Drain remaining events
			for {
				select {
				case event := <-eb.buffer:
					eb.deliver(event)
				default:
					return
				}
			}
		}
	}
}

// Stop shuts down the event bus.
func (eb *EventBus) Stop() {
	close(eb.stopCh)
	eb.wg.Wait()

	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Cancel all subscriptions
	for _, sub := range eb.subscriptions {
		sub.Cancel()
	}
	eb.subscriptions = make(map[string]*Subscription)
}

// Stats returns event bus statistics.
func (eb *EventBus) Stats() EventBusStats {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	buffered := 0
	if eb.useBuffer {
		buffered = len(eb.buffer)
	}

	return EventBusStats{
		EventCount:      eb.eventCount,
		SubscriberCount: eb.subscriberCount,
		BufferedEvents:  buffered,
		BufferSize:      eb.bufferSize,
	}
}

// EventBusStats contains event bus metrics.
type EventBusStats struct {
	EventCount      int64 `json:"eventCount"`
	SubscriberCount int64 `json:"subscriberCount"`
	BufferedEvents  int   `json:"bufferedEvents"`
	BufferSize      int   `json:"bufferSize"`
}

// generateSubscriptionID creates a unique subscription ID.
func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d_%d", time.Now().UnixNano(), subscriptionCounter.Add(1))
}

var subscriptionCounter atomicInt64

// atomicInt64 is a simple atomic counter.
type atomicInt64 struct {
	value int64
	mu    sync.Mutex
}

func (a *atomicInt64) Add(delta int64) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.value += delta
	return a.value
}

// Convenience functions for creating common events

// NewDeviceDiscoveredEvent creates a device discovered event.
func NewDeviceDiscoveredEvent(source EventSource, device *DiscoveredDevice) *Event {
	return NewEvent(EvtDeviceDiscovered, source).WithDevice(device)
}

// NewDeviceUpdatedEvent creates a device updated event.
func NewDeviceUpdatedEvent(source EventSource, device *DiscoveredDevice, changes map[string]any) *Event {
	e := NewEvent(EvtDeviceUpdated, source).WithDevice(device)
	e.Changes = changes
	return e
}

// NewDeviceLostEvent creates a device lost event.
func NewDeviceLostEvent(source EventSource, mac string) *Event {
	return NewEvent(EvtDeviceLost, source).WithMAC(mac)
}

// NewScanStartedEvent creates a scan started event.
func NewScanStartedEvent(scanType string) *Event {
	return NewEvent(EventScanStarted, SourceEngine).WithPayload(map[string]string{
		"scanType": scanType,
	})
}

// NewScanCompletedEvent creates a scan completed event.
func NewScanCompletedEvent(scanType string, deviceCount int, duration time.Duration) *Event {
	return NewEvent(EventScanCompleted, SourceEngine).WithPayload(map[string]any{
		"scanType":    scanType,
		"deviceCount": deviceCount,
		"durationMs":  duration.Milliseconds(),
	})
}

// NewVulnDiscoveredEvent creates a vulnerability discovered event.
func NewVulnDiscoveredEvent(device *DiscoveredDevice, cveID string, severity string) *Event {
	return NewEvent(EventVulnDiscovered, SourceAssessment).
		WithDevice(device).
		WithPayload(map[string]string{
			"cveId":    cveID,
			"severity": severity,
		})
}
