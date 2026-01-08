package logging_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

func TestNewRingBuffer(t *testing.T) {
	t.Run("creates buffer with specified size", func(t *testing.T) {
		rb := logging.NewRingBuffer(100)
		if rb == nil {
			t.Fatal("NewRingBuffer() returned nil")
		}
		if rb.Count() != 0 {
			t.Errorf("Count() = %d, want 0", rb.Count())
		}
	})

	t.Run("uses default size for zero", func(t *testing.T) {
		rb := logging.NewRingBuffer(0)
		if rb == nil {
			t.Fatal("NewRingBuffer(0) returned nil")
		}
	})

	t.Run("uses default size for negative", func(t *testing.T) {
		rb := logging.NewRingBuffer(-5)
		if rb == nil {
			t.Fatal("NewRingBuffer(-5) returned nil")
		}
	})
}

func TestRingBuffer_Add(t *testing.T) {
	t.Run("adds entries up to capacity", func(t *testing.T) {
		rb := logging.NewRingBuffer(5)

		for i := range 5 {
			entry := logging.NewLogEntry("INFO", "message")
			entry.AddMetadata("index", i)
			rb.Add(entry)
		}

		if rb.Count() != 5 {
			t.Errorf("Count() = %d, want 5", rb.Count())
		}
	})

	t.Run("overwrites oldest when full", func(t *testing.T) {
		rb := logging.NewRingBuffer(3)

		// Add 5 entries to a buffer of size 3
		for i := range 5 {
			entry := logging.NewLogEntry("INFO", "message")
			entry.AddMetadata("index", i)
			rb.Add(entry)
		}

		// Should still only have 3 entries
		if rb.Count() != 3 {
			t.Errorf("Count() = %d, want 3", rb.Count())
		}

		// The oldest entries (0, 1) should be overwritten
		entries := rb.GetAll()
		if len(entries) != 3 {
			t.Fatalf("GetAll() returned %d entries, want 3", len(entries))
		}

		// Entries should be 2, 3, 4 (in order)
		for i, entry := range entries {
			expectedIdx := i + 2
			if idx, ok := entry.Metadata["index"].(int); !ok || idx != expectedIdx {
				t.Errorf("entry[%d] index = %v, want %d", i, entry.Metadata["index"], expectedIdx)
			}
		}
	})
}

func TestRingBuffer_GetAll_Empty(t *testing.T) {
	rb := logging.NewRingBuffer(5)
	entries := rb.GetAll()
	if entries != nil {
		t.Errorf("GetAll() = %v, want nil", entries)
	}
}

func TestRingBuffer_GetAll_ChronologicalOrder(t *testing.T) {
	rb := logging.NewRingBuffer(5)

	for i := range 3 {
		entry := logging.NewLogEntry("INFO", "message")
		entry.AddMetadata("index", i)
		rb.Add(entry)
	}

	entries := rb.GetAll()
	if len(entries) != 3 {
		t.Fatalf("GetAll() returned %d entries, want 3", len(entries))
	}

	for i, entry := range entries {
		if idx, ok := entry.Metadata["index"].(int); !ok || idx != i {
			t.Errorf("entry[%d] index = %v, want %d", i, entry.Metadata["index"], i)
		}
	}
}

func TestRingBuffer_GetAll_AfterWraparound(t *testing.T) {
	rb := logging.NewRingBuffer(3)

	// Add 5 entries to cause wraparound
	for i := range 5 {
		entry := logging.NewLogEntry("INFO", "message")
		entry.AddMetadata("index", i)
		rb.Add(entry)
	}

	entries := rb.GetAll()
	if len(entries) != 3 {
		t.Fatalf("GetAll() returned %d entries, want 3", len(entries))
	}

	// Should be 2, 3, 4 in order
	for i, entry := range entries {
		expectedIdx := i + 2
		if idx, ok := entry.Metadata["index"].(int); !ok || idx != expectedIdx {
			t.Errorf("entry[%d] index = %v, want %d", i, entry.Metadata["index"], expectedIdx)
		}
	}
}

func TestRingBuffer_GetRecent_Empty(t *testing.T) {
	rb := logging.NewRingBuffer(5)
	entries := rb.GetRecent(3)
	if entries != nil {
		t.Errorf("GetRecent(3) = %v, want nil", entries)
	}
}

func TestRingBuffer_GetRecent_ZeroCount(t *testing.T) {
	rb := logging.NewRingBuffer(5)
	entry := logging.NewLogEntry("INFO", "message")
	rb.Add(entry)
	entries := rb.GetRecent(0)
	if entries != nil {
		t.Errorf("GetRecent(0) = %v, want nil", entries)
	}
}

func TestRingBuffer_GetRecent_NegativeCount(t *testing.T) {
	rb := logging.NewRingBuffer(5)
	entry := logging.NewLogEntry("INFO", "message")
	rb.Add(entry)
	entries := rb.GetRecent(-1)
	if entries != nil {
		t.Errorf("GetRecent(-1) = %v, want nil", entries)
	}
}

func TestRingBuffer_GetRecent_ExceedsCount(t *testing.T) {
	rb := logging.NewRingBuffer(10)

	for i := range 3 {
		entry := logging.NewLogEntry("INFO", "message")
		entry.AddMetadata("index", i)
		rb.Add(entry)
	}

	entries := rb.GetRecent(10)
	if len(entries) != 3 {
		t.Fatalf("GetRecent(10) returned %d entries, want 3", len(entries))
	}
}

func TestRingBuffer_GetRecent_MostRecentN(t *testing.T) {
	rb := logging.NewRingBuffer(10)

	for i := range 5 {
		entry := logging.NewLogEntry("INFO", "message")
		entry.AddMetadata("index", i)
		rb.Add(entry)
	}

	entries := rb.GetRecent(2)
	if len(entries) != 2 {
		t.Fatalf("GetRecent(2) returned %d entries, want 2", len(entries))
	}

	// Should be 3, 4 in order
	if idx, ok := entries[0].Metadata["index"].(int); !ok || idx != 3 {
		t.Errorf("entries[0] index = %v, want 3", entries[0].Metadata["index"])
	}
	if idx, ok := entries[1].Metadata["index"].(int); !ok || idx != 4 {
		t.Errorf("entries[1] index = %v, want 4", entries[1].Metadata["index"])
	}
}

func TestRingBuffer_GetRecent_AfterWraparound(t *testing.T) {
	rb := logging.NewRingBuffer(3)

	// Add 5 entries to cause wraparound
	for i := range 5 {
		entry := logging.NewLogEntry("INFO", "message")
		entry.AddMetadata("index", i)
		rb.Add(entry)
	}

	entries := rb.GetRecent(2)
	if len(entries) != 2 {
		t.Fatalf("GetRecent(2) returned %d entries, want 2", len(entries))
	}

	// Should be 3, 4 in order (most recent 2 of [2, 3, 4])
	if idx, ok := entries[0].Metadata["index"].(int); !ok || idx != 3 {
		t.Errorf("entries[0] index = %v, want 3", entries[0].Metadata["index"])
	}
	if idx, ok := entries[1].Metadata["index"].(int); !ok || idx != 4 {
		t.Errorf("entries[1] index = %v, want 4", entries[1].Metadata["index"])
	}
}

func TestRingBuffer_Count(t *testing.T) {
	rb := logging.NewRingBuffer(5)

	if rb.Count() != 0 {
		t.Errorf("Count() = %d, want 0", rb.Count())
	}

	entry := logging.NewLogEntry("INFO", "message")
	rb.Add(entry)
	if rb.Count() != 1 {
		t.Errorf("Count() = %d, want 1", rb.Count())
	}

	rb.Add(entry)
	rb.Add(entry)
	if rb.Count() != 3 {
		t.Errorf("Count() = %d, want 3", rb.Count())
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := logging.NewRingBuffer(5)

	for range 3 {
		entry := logging.NewLogEntry("INFO", "message")
		rb.Add(entry)
	}

	if rb.Count() != 3 {
		t.Fatalf("Count() = %d, want 3 before clear", rb.Count())
	}

	rb.Clear()

	if rb.Count() != 0 {
		t.Errorf("Count() = %d, want 0 after clear", rb.Count())
	}

	entries := rb.GetAll()
	if entries != nil {
		t.Errorf("GetAll() = %v, want nil after clear", entries)
	}
}

func TestRingBuffer_Concurrency(t *testing.T) {
	rb := logging.NewRingBuffer(100)
	var wg sync.WaitGroup

	// Multiple writers
	for i := range 10 {
		writerID := i
		wg.Go(func() {
			for j := range 20 {
				entry := logging.NewLogEntry("INFO", "message")
				entry.AddMetadata("writer", writerID)
				entry.AddMetadata("seq", j)
				rb.Add(entry)
			}
		})
	}

	// Multiple readers
	for range 5 {
		wg.Go(func() {
			for range 20 {
				_ = rb.GetAll()
				_ = rb.GetRecent(10)
				_ = rb.Count()
			}
		})
	}

	wg.Wait()

	// Buffer should have entries (200 writes, buffer size 100)
	if rb.Count() != 100 {
		t.Errorf("Count() = %d, want 100 (buffer should be full)", rb.Count())
	}
}

func TestNewLogBroadcaster(t *testing.T) {
	lb := logging.NewLogBroadcaster(100)
	if lb == nil {
		t.Fatal("NewLogBroadcaster() returned nil")
	}

	if lb.LogCount() != 0 {
		t.Errorf("LogCount() = %d, want 0", lb.LogCount())
	}
}

func TestLogBroadcaster_Write(t *testing.T) {
	t.Run("writes to internal buffer", func(t *testing.T) {
		lb := logging.NewLogBroadcaster(100)

		entry := logging.NewLogEntry("INFO", "test message")
		lb.Write(entry)

		if lb.LogCount() != 1 {
			t.Errorf("LogCount() = %d, want 1", lb.LogCount())
		}
	})

	t.Run("broadcasts to registered broadcaster", func(t *testing.T) {
		lb := logging.NewLogBroadcaster(100)
		mb := &mockBroadcaster{}
		lb.SetBroadcaster(mb)

		entry := logging.NewLogEntry("INFO", "test message")
		lb.Write(entry)

		if len(mb.entries) != 1 {
			t.Errorf("Broadcaster received %d entries, want 1", len(mb.entries))
		}
	})
}

func TestLogBroadcaster_GetRecentLogs(t *testing.T) {
	lb := logging.NewLogBroadcaster(100)

	for i := range 5 {
		entry := logging.NewLogEntry("INFO", "message")
		entry.AddMetadata("index", i)
		lb.Write(entry)
	}

	entries := lb.GetRecentLogs(3)
	if len(entries) != 3 {
		t.Fatalf("GetRecentLogs(3) returned %d entries, want 3", len(entries))
	}

	// Should be most recent 3: 2, 3, 4
	if idx, ok := entries[0].Metadata["index"].(int); !ok || idx != 2 {
		t.Errorf("entries[0] index = %v, want 2", entries[0].Metadata["index"])
	}
}

func TestLogBroadcaster_GetAllLogs(t *testing.T) {
	lb := logging.NewLogBroadcaster(100)

	for i := range 5 {
		entry := logging.NewLogEntry("INFO", "message")
		entry.AddMetadata("index", i)
		lb.Write(entry)
	}

	entries := lb.GetAllLogs()
	if len(entries) != 5 {
		t.Fatalf("GetAllLogs() returned %d entries, want 5", len(entries))
	}
}

func TestLogBroadcaster_SetBroadcaster(t *testing.T) {
	lb := logging.NewLogBroadcaster(100)
	mb := &mockBroadcaster{}

	lb.SetBroadcaster(mb)

	entry := logging.NewLogEntry("INFO", "test")
	lb.Write(entry)

	if len(mb.entries) != 1 {
		t.Errorf("Broadcaster should have received entry")
	}
}

func TestLogBroadcaster_SetDBWriter(t *testing.T) {
	lb := logging.NewLogBroadcaster(100)
	mw := &mockDBWriter{}

	lb.SetDBWriter(mw)

	// Write multiple entries to trigger batch
	for range 50 {
		entry := logging.NewLogEntry("INFO", "test")
		lb.Write(entry)
	}

	// Give time for the batch to be processed
	time.Sleep(100 * time.Millisecond)

	// Stop to flush any remaining entries
	lb.Stop()

	// The writer should have received entries
	if mw.writeCount == 0 && mw.batchCount == 0 {
		t.Error("DBWriter should have received entries")
	}
}

func TestLogBroadcaster_Stop(_ *testing.T) {
	lb := logging.NewLogBroadcaster(100)
	mw := &mockDBWriter{}
	lb.SetDBWriter(mw)

	for range 10 {
		entry := logging.NewLogEntry("INFO", "test")
		lb.Write(entry)
	}

	// Stop should flush remaining entries
	lb.Stop()

	// Multiple Stop calls should not panic
	lb.Stop()
	lb.Stop()
}

func TestInitBroadcaster(t *testing.T) {
	lb := logging.InitBroadcaster(100)
	if lb == nil {
		t.Fatal("InitBroadcaster() returned nil")
	}

	// Should be retrievable via GetBroadcaster
	retrieved := logging.GetBroadcaster()
	if retrieved != lb {
		t.Error("GetBroadcaster() did not return the initialized broadcaster")
	}
}

func TestGetBroadcaster(t *testing.T) {
	// Initialize a new broadcaster
	lb := logging.InitBroadcaster(50)

	got := logging.GetBroadcaster()
	if got != lb {
		t.Error("GetBroadcaster() returned different broadcaster than InitBroadcaster()")
	}
}

func TestBroadcastLog(t *testing.T) {
	// Initialize broadcaster
	lb := logging.InitBroadcaster(100)
	mb := &mockBroadcaster{}
	lb.SetBroadcaster(mb)

	metadata := map[string]any{
		"key": "value",
	}

	logging.BroadcastLog("INFO", "backend", "test-component", "test message", metadata)

	if len(mb.entries) != 1 {
		t.Fatalf("BroadcastLog should have broadcast 1 entry, got %d", len(mb.entries))
	}

	entry := mb.entries[0]
	if entry.Level != "INFO" {
		t.Errorf("Level = %q, want INFO", entry.Level)
	}
	if entry.Layer != "backend" {
		t.Errorf("Layer = %q, want backend", entry.Layer)
	}
	if entry.Component != "test-component" {
		t.Errorf("Component = %q, want test-component", entry.Component)
	}
	if entry.Message != "test message" {
		t.Errorf("Message = %q, want test message", entry.Message)
	}
}

func TestLogEntryJSON(t *testing.T) {
	entry := logging.NewLogEntry("ERROR", "test error")
	entry.WithComponent("test-component").WithLayer("api")

	data, err := logging.LogEntryJSON(entry)
	if err != nil {
		t.Fatalf("LogEntryJSON() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("LogEntryJSON() returned empty data")
	}

	// Verify it contains expected fields
	jsonStr := string(data)
	if !testContainsStr(jsonStr, "ERROR") {
		t.Error("JSON does not contain level")
	}
	if !testContainsStr(jsonStr, "test error") {
		t.Error("JSON does not contain message")
	}
}

// Helper types

type mockBroadcaster struct {
	entries []*logging.LogEntry
	mu      sync.Mutex
}

func (m *mockBroadcaster) BroadcastLogEntry(entry *logging.LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
}

type mockDBWriter struct {
	mu         sync.Mutex
	writeCount int
	batchCount int
}

func (m *mockDBWriter) WriteLog(_ context.Context, _ *logging.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeCount++
	return nil
}

func (m *mockDBWriter) WriteBatch(_ context.Context, entries []*logging.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batchCount += len(entries)
	return nil
}

func testContainsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
