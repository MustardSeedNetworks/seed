package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/krisarmstrong/seed/internal/database"
)

// createTestLogEntries creates sample log entries for testing.
func createTestLogEntries(ctx context.Context, t *testing.T, repo *database.LogRepository) {
	t.Helper()

	entries := []*database.LogEntry{
		{
			Timestamp:  time.Now().UTC(),
			Level:      "INFO",
			Layer:      "backend",
			Message:    "Test log message",
			Component:  "test",
			RequestID:  "req-123",
			SessionID:  "sess-456",
			DurationMs: 100,
			Metadata:   `{"key": "value"}`,
			Stack:      "stack trace",
		},
		{
			Timestamp: time.Now().UTC(),
			Level:     "ERROR",
			Layer:     "api",
			Message:   "Minimal log entry",
		},
		{
			Timestamp: time.Now().UTC(),
			Level:     "DEBUG",
			Layer:     "backend",
			Message:   "Batch log 1",
		},
		{
			Timestamp: time.Now().UTC(),
			Level:     "WARN",
			Layer:     "frontend",
			Message:   "Batch log 2",
			Component: "ui",
		},
		{
			Timestamp: time.Now().UTC(),
			Level:     "ERROR",
			Layer:     "api",
			Message:   "Batch log 3",
			RequestID: "batch-req-789",
		},
	}

	for _, entry := range entries {
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}
}

func TestLogRepositoryCreate(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Logs()

	t.Run("Create with all fields", func(t *testing.T) {
		entry := &database.LogEntry{
			Timestamp:  time.Now().UTC(),
			Level:      "INFO",
			Layer:      "backend",
			Message:    "Test log message",
			Component:  "test",
			RequestID:  "req-123",
			SessionID:  "sess-456",
			DurationMs: 100,
			Metadata:   `{"key": "value"}`,
			Stack:      "stack trace",
		}

		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	})

	t.Run("Create with empty optional fields", func(t *testing.T) {
		entry := &database.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     "ERROR",
			Layer:     "api",
			Message:   "Minimal log entry",
		}

		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	})
}

func TestLogRepositoryBatchCreate(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Logs()

	t.Run("BatchCreate multiple entries", func(t *testing.T) {
		entries := []*database.LogEntry{
			{
				Timestamp: time.Now().UTC(),
				Level:     "DEBUG",
				Layer:     "backend",
				Message:   "Batch log 1",
			},
			{
				Timestamp: time.Now().UTC(),
				Level:     "WARN",
				Layer:     "frontend",
				Message:   "Batch log 2",
				Component: "ui",
			},
			{
				Timestamp: time.Now().UTC(),
				Level:     "ERROR",
				Layer:     "api",
				Message:   "Batch log 3",
				RequestID: "batch-req-789",
			},
		}

		err := repo.BatchCreate(ctx, entries)
		require.NoError(t, err)

		// Verify entries were created
		count, err := repo.Count(ctx)
		require.NoError(t, err)
		if count != 3 {
			t.Errorf("expected 3 entries, got %d", count)
		}
	})

	t.Run("BatchCreate empty list", func(t *testing.T) {
		err := repo.BatchCreate(ctx, []*database.LogEntry{})
		require.NoError(t, err)
	})
}

// logListTestCase defines a test case for log list filtering.
type logListTestCase struct {
	name      string
	opts      database.LogListOptions
	validate  func(t *testing.T, entries []*database.LogEntry)
	minCount  int
	exactSize int
}

// getLogListTestCases returns all filter test cases for log repository.
func getLogListTestCases() []logListTestCase {
	now := time.Now().UTC()
	return []logListTestCase{
		{
			name:     "List all",
			opts:     database.LogListOptions{},
			minCount: 5,
		},
		{
			name: "List with level filter",
			opts: database.LogListOptions{Level: "ERROR"},
			validate: func(t *testing.T, entries []*database.LogEntry) {
				t.Helper()
				for _, e := range entries {
					if e.Level != "ERROR" {
						t.Errorf("expected level ERROR, got %s", e.Level)
					}
				}
			},
		},
		{
			name: "List with layer filter",
			opts: database.LogListOptions{Layer: "backend"},
			validate: func(t *testing.T, entries []*database.LogEntry) {
				t.Helper()
				for _, e := range entries {
					if e.Layer != "backend" {
						t.Errorf("expected layer backend, got %s", e.Layer)
					}
				}
			},
		},
		{
			name: "List with component filter",
			opts: database.LogListOptions{Component: "test"},
			validate: func(t *testing.T, entries []*database.LogEntry) {
				t.Helper()
				for _, e := range entries {
					if e.Component != "test" {
						t.Errorf("expected component test, got %s", e.Component)
					}
				}
			},
		},
		{
			name: "List with request ID filter",
			opts: database.LogListOptions{RequestID: "req-123"},
			validate: func(t *testing.T, entries []*database.LogEntry) {
				t.Helper()
				for _, e := range entries {
					if e.RequestID != "req-123" {
						t.Errorf("expected requestID req-123, got %s", e.RequestID)
					}
				}
			},
		},
		{
			name:     "List with time range",
			opts:     database.LogListOptions{Since: now.Add(-time.Hour), Until: now.Add(time.Hour)},
			minCount: 1,
		},
		{
			name:     "List with search",
			opts:     database.LogListOptions{Search: "Batch log"},
			minCount: 1,
		},
		{
			name:      "List with pagination",
			opts:      database.LogListOptions{Limit: 2, Offset: 1},
			exactSize: 2,
		},
	}
}

func TestLogRepositoryList(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Logs()
	createTestLogEntries(ctx, t, repo)

	for _, tc := range getLogListTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := repo.List(ctx, tc.opts)
			require.NoError(t, err)

			if tc.minCount > 0 && len(entries) < tc.minCount {
				t.Errorf("expected at least %d entries, got %d", tc.minCount, len(entries))
			}
			if tc.exactSize > 0 && len(entries) != tc.exactSize {
				t.Errorf("expected exactly %d entries, got %d", tc.exactSize, len(entries))
			}
			if tc.validate != nil {
				tc.validate(t, entries)
			}
		})
	}
}

func TestLogRepositoryGetRecent(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Logs()

	// Create test data
	createTestLogEntries(ctx, t, repo)

	entries, err := repo.GetRecent(ctx, 3)
	require.NoError(t, err)
	if len(entries) != 3 {
		t.Errorf("expected 3 recent entries, got %d", len(entries))
	}
}

func TestLogRepositoryCount(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Logs()

	// Create test data
	createTestLogEntries(ctx, t, repo)

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	if count != 5 {
		t.Errorf("expected 5 entries, got %d", count)
	}
}

func TestLogRepositoryDeleteOlderThan(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Logs()

	// Create entries with captured time
	now := time.Now().UTC()
	for i := range 3 {
		entry := &database.LogEntry{
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
			Level:     "DEBUG",
			Layer:     "backend",
			Message:   "Entry to delete",
		}
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	// Verify entries exist
	countBefore, err := repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(3), countBefore)

	// Delete all logs older than 1 hour from now (should delete all)
	futureTime := now.Add(time.Hour)
	deleted, err := repo.DeleteOlderThan(ctx, futureTime)
	require.NoError(t, err)
	if deleted != 3 {
		t.Errorf("expected 3 deleted, got %d", deleted)
	}

	// Verify entries are gone
	countAfter, err := repo.Count(ctx)
	require.NoError(t, err)
	if countAfter != 0 {
		t.Errorf("expected 0 entries after delete, got %d", countAfter)
	}
}

func TestLogRepositoryClear(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := db.Logs()

	// Add some entries first
	for range 3 {
		err := repo.Create(ctx, &database.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     "INFO",
			Layer:     "backend",
			Message:   "Log to clear",
		})
		require.NoError(t, err)
	}

	// Clear all logs
	err := repo.Clear(ctx)
	require.NoError(t, err)

	// Verify logs are gone
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	if count != 0 {
		t.Errorf("expected 0 entries after clear, got %d", count)
	}
}

func TestConvertMetadataToJSON(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		result := database.ConvertMetadataToJSON(nil)
		if result != "" {
			t.Errorf("expected empty string, got %s", result)
		}

		result = database.ConvertMetadataToJSON(map[string]any{})
		if result != "" {
			t.Errorf("expected empty string for empty map, got %s", result)
		}
	})

	t.Run("with data", func(t *testing.T) {
		metadata := map[string]any{
			"key":   "value",
			"count": 42,
		}
		result := database.ConvertMetadataToJSON(metadata)
		if result == "" {
			t.Error("expected non-empty JSON string")
		}
	})
}
