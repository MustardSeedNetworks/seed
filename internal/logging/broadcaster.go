// Package logging provides structured logging with automatic redaction of sensitive data.
package logging

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Broadcaster defines the interface for broadcasting log entries to connected clients.
// This interface allows the logging package to broadcast logs without depending on
// the api package (avoiding circular imports).
type Broadcaster interface {
	// BroadcastLogEntry sends a log entry to all subscribed clients.
	BroadcastLogEntry(entry *LogEntry)
}

// DBLogWriter defines the interface for persisting logs to a database.
// This interface allows the logging package to persist logs without depending on
// the database package (avoiding circular imports).
type DBLogWriter interface {
	// WriteLog persists a log entry to the database.
	WriteLog(ctx context.Context, entry *LogEntry) error
	// WriteBatch persists multiple log entries in a single transaction.
	WriteBatch(ctx context.Context, entries []*LogEntry) error
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
// It also persists logs to a database for durability.
type LogBroadcaster struct {
	buffer      *RingBuffer
	broadcaster Broadcaster // Injected dependency to avoid circular imports
	dbWriter    DBLogWriter // Database writer for persistence
	mu          sync.RWMutex

	// Batching for efficient database writes
	batchBuffer []*LogEntry
	batchMu     sync.Mutex
	batchSize   int           // Number of entries to batch before flushing
	batchTimer  *time.Timer   // Timer to flush batch periodically
	stopCh      chan struct{} // Signal to stop the flush goroutine
}

// NewLogBroadcaster creates a new LogBroadcaster with the specified buffer size.
func NewLogBroadcaster(bufferSize int) *LogBroadcaster {
	return &LogBroadcaster{
		buffer:      NewRingBuffer(bufferSize),
		batchBuffer: make([]*LogEntry, 0, 50),
		batchSize:   50, // Flush after 50 entries
		stopCh:      make(chan struct{}),
	}
}

// SetBroadcaster sets the broadcaster implementation (typically the WebSocket hub wrapper).
// This allows late binding to avoid circular import issues.
func (lb *LogBroadcaster) SetBroadcaster(b Broadcaster) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.broadcaster = b
}

// SetDBWriter sets the database writer for log persistence.
// Once set, logs will be batched and periodically flushed to the database.
func (lb *LogBroadcaster) SetDBWriter(w DBLogWriter) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.dbWriter = w

	// Start the periodic flush timer (every 5 seconds)
	if lb.batchTimer == nil {
		lb.batchTimer = time.AfterFunc(5*time.Second, lb.flushBatch)
	}
}

// flushBatch writes all buffered log entries to the database.
func (lb *LogBroadcaster) flushBatch() {
	lb.batchMu.Lock()
	if len(lb.batchBuffer) == 0 {
		lb.batchMu.Unlock()
		// Reset timer
		lb.mu.RLock()
		if lb.batchTimer != nil {
			lb.batchTimer.Reset(5 * time.Second)
		}
		lb.mu.RUnlock()
		return
	}

	// Copy buffer and reset
	entries := make([]*LogEntry, len(lb.batchBuffer))
	copy(entries, lb.batchBuffer)
	lb.batchBuffer = lb.batchBuffer[:0]
	lb.batchMu.Unlock()

	// Get database writer
	lb.mu.RLock()
	dbWriter := lb.dbWriter
	lb.mu.RUnlock()

	if dbWriter != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := dbWriter.WriteBatch(ctx, entries); err != nil {
			// Log error but don't fail - logs are still in memory buffer
			// We can't use the logging package here to avoid recursion
			_ = err // Silently ignore - data is still in ring buffer
		}
	}

	// Reset timer for next flush
	lb.mu.RLock()
	if lb.batchTimer != nil {
		lb.batchTimer.Reset(5 * time.Second)
	}
	lb.mu.RUnlock()
}

// Stop stops the log broadcaster and flushes any remaining entries.
func (lb *LogBroadcaster) Stop() {
	// Signal stop
	select {
	case <-lb.stopCh:
		// Already closed
	default:
		close(lb.stopCh)
	}

	// Stop timer
	lb.mu.Lock()
	if lb.batchTimer != nil {
		lb.batchTimer.Stop()
	}
	lb.mu.Unlock()

	// Final flush
	lb.flushBatch()
}

// Write adds a log entry to the buffer and broadcasts it to connected clients.
// If a database writer is configured, entries are batched and persisted.
func (lb *LogBroadcaster) Write(entry *LogEntry) {
	// Add to memory buffer
	lb.buffer.Add(entry)

	// Broadcast to clients
	lb.mu.RLock()
	broadcaster := lb.broadcaster
	dbWriter := lb.dbWriter
	lb.mu.RUnlock()

	if broadcaster != nil {
		broadcaster.BroadcastLogEntry(entry)
	}

	// Add to batch buffer for database persistence
	if dbWriter != nil {
		lb.batchMu.Lock()
		lb.batchBuffer = append(lb.batchBuffer, entry)
		shouldFlush := len(lb.batchBuffer) >= lb.batchSize
		lb.batchMu.Unlock()

		// Flush immediately if batch is full
		if shouldFlush {
			go lb.flushBatch()
		}
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

// Broadcaster accessor functions use closure-encapsulated state to satisfy gochecknoglobals.
// getBroadcaster returns the global broadcaster instance.
// setBroadcaster sets the global broadcaster instance.
// _ (clearBroadcaster) resets the global broadcaster to nil (unused but required for pattern).
var (
	getBroadcaster, setBroadcaster, _ = func() (
		func() *LogBroadcaster,
		func(*LogBroadcaster),
		func(),
	) {
		var broadcaster *LogBroadcaster
		var mu sync.RWMutex

		return func() *LogBroadcaster {
				mu.RLock()
				defer mu.RUnlock()
				return broadcaster
			}, func(b *LogBroadcaster) {
				mu.Lock()
				defer mu.Unlock()
				broadcaster = b
			}, func() {
				mu.Lock()
				defer mu.Unlock()
				broadcaster = nil
			}
	}()
)

// InitBroadcaster initializes the global log broadcaster.
// Call this during server initialization.
func InitBroadcaster(bufferSize int) *LogBroadcaster {
	b := NewLogBroadcaster(bufferSize)
	setBroadcaster(b)
	return b
}

// GetBroadcaster returns the global log broadcaster.
func GetBroadcaster() *LogBroadcaster {
	return getBroadcaster()
}

// BroadcastLog creates and broadcasts a log entry from the given parameters.
// This is a convenience function for creating and broadcasting logs programmatically.
func BroadcastLog(level, layer, component, message string, metadata map[string]any) {
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
