// Package logging provides structured logging with automatic redaction of sensitive data.
package logging

import (
	"encoding/json"
	"sync"
)

// Broadcaster defines the interface for broadcasting log entries to connected clients.
// This interface allows the logging package to broadcast logs without depending on
// the api package (avoiding circular imports).
type Broadcaster interface {
	// BroadcastLogEntry sends a log entry to all subscribed clients.
	BroadcastLogEntry(entry *LogEntry)
}

// RingBuffer is a fixed-size circular buffer for storing recent log entries.
// It provides O(1) add and O(n) retrieval operations.
type RingBuffer struct {
	entries []*LogEntry
	size    int
	head    int // Next write position
	count   int // Number of entries currently stored
	mu      sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 1000 // Default to 1000 entries
	}
	return &RingBuffer{
		entries: make([]*LogEntry, size),
		size:    size,
	}
}

// Add inserts a log entry into the buffer, overwriting the oldest entry if full.
func (rb *RingBuffer) Add(entry *LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.entries[rb.head] = entry
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// GetAll returns all entries in the buffer in chronological order.
func (rb *RingBuffer) GetAll() []*LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return nil
	}

	result := make([]*LogEntry, rb.count)
	start := 0
	if rb.count == rb.size {
		start = rb.head
	}

	for i := range rb.count {
		idx := (start + i) % rb.size
		result[i] = rb.entries[idx]
	}

	return result
}

// GetRecent returns the most recent n entries in chronological order.
func (rb *RingBuffer) GetRecent(n int) []*LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 || n <= 0 {
		return nil
	}

	if n > rb.count {
		n = rb.count
	}

	result := make([]*LogEntry, n)

	for i := range n {
		var idx int
		if rb.count == rb.size {
			idx = (rb.head - n + i + rb.size) % rb.size
		} else {
			idx = rb.count - n + i
		}
		result[i] = rb.entries[idx]
	}

	return result
}

// Count returns the number of entries currently in the buffer.
func (rb *RingBuffer) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Clear removes all entries from the buffer.
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.entries = make([]*LogEntry, rb.size)
	rb.head = 0
	rb.count = 0
}

// LogBroadcaster manages broadcasting log entries to WebSocket clients
// and maintains a buffer of recent logs for new client connections.
type LogBroadcaster struct {
	buffer      *RingBuffer
	broadcaster Broadcaster // Injected dependency to avoid circular imports
	mu          sync.RWMutex
}

// NewLogBroadcaster creates a new LogBroadcaster with the specified buffer size.
func NewLogBroadcaster(bufferSize int) *LogBroadcaster {
	return &LogBroadcaster{
		buffer: NewRingBuffer(bufferSize),
	}
}

// SetBroadcaster sets the broadcaster implementation (typically the WebSocket hub wrapper).
// This allows late binding to avoid circular import issues.
func (lb *LogBroadcaster) SetBroadcaster(b Broadcaster) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.broadcaster = b
}

// Write adds a log entry to the buffer and broadcasts it to connected clients.
func (lb *LogBroadcaster) Write(entry *LogEntry) {
	// Add to buffer
	lb.buffer.Add(entry)

	// Broadcast to clients
	lb.mu.RLock()
	broadcaster := lb.broadcaster
	lb.mu.RUnlock()

	if broadcaster != nil {
		broadcaster.BroadcastLogEntry(entry)
	}
}

// GetRecentLogs returns the most recent n log entries.
func (lb *LogBroadcaster) GetRecentLogs(n int) []*LogEntry {
	return lb.buffer.GetRecent(n)
}

// GetAllLogs returns all buffered log entries.
func (lb *LogBroadcaster) GetAllLogs() []*LogEntry {
	return lb.buffer.GetAll()
}

// LogCount returns the number of buffered log entries.
func (lb *LogBroadcaster) LogCount() int {
	return lb.buffer.Count()
}

// Global log broadcaster instance.
var (
	globalBroadcaster *LogBroadcaster
	broadcasterMu     sync.RWMutex
)

// InitBroadcaster initializes the global log broadcaster.
// Call this during server initialization.
func InitBroadcaster(bufferSize int) *LogBroadcaster {
	broadcasterMu.Lock()
	defer broadcasterMu.Unlock()
	globalBroadcaster = NewLogBroadcaster(bufferSize)
	return globalBroadcaster
}

// GetBroadcaster returns the global log broadcaster.
func GetBroadcaster() *LogBroadcaster {
	broadcasterMu.RLock()
	defer broadcasterMu.RUnlock()
	return globalBroadcaster
}

// BroadcastLog creates and broadcasts a log entry from the given parameters.
// This is a convenience function for creating and broadcasting logs programmatically.
func BroadcastLog(level, layer, component, message string, metadata map[string]interface{}) {
	entry := NewLogEntry(level, message).
		WithLayer(layer).
		WithComponent(component).
		WithMetadata(metadata)

	broadcaster := GetBroadcaster()
	if broadcaster != nil {
		broadcaster.Write(entry)
	}
}

// LogEntryJSON returns the JSON representation of a log entry.
// Useful for WebSocket message payloads.
func LogEntryJSON(entry *LogEntry) ([]byte, error) {
	return json.Marshal(entry)
}
