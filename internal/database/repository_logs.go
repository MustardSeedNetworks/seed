// Package database provides log repository for persistent log storage.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// LogEntry represents a log entry stored in the database.
type LogEntry struct {
	ID         int64
	Timestamp  time.Time
	Level      string
	Layer      string
	Message    string
	Component  string
	RequestID  string
	SessionID  string
	DurationMs int64
	Metadata   string // JSON string
	Stack      string
}

// LogRepository provides operations for log entries.
type LogRepository struct {
	db *DB
}

// Create inserts a new log entry into the database.
func (r *LogRepository) Create(ctx context.Context, entry *LogEntry) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO logs
		(timestamp, level, layer, message, component, request_id, session_id, duration_ms, metadata_json, stack)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.Timestamp.Format(time.RFC3339Nano), entry.Level, entry.Layer, entry.Message,
		nullString(entry.Component), nullString(entry.RequestID), nullString(entry.SessionID),
		entry.DurationMs, nullString(entry.Metadata), nullString(entry.Stack))
	if err != nil {
		return fmt.Errorf("failed to create log entry: %w", err)
	}
	return nil
}

// BatchCreate inserts multiple log entries in a single transaction.
func (r *LogRepository) BatchCreate(ctx context.Context, entries []*LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO logs
			(timestamp, level, layer, message, component, request_id, session_id, duration_ms, metadata_json, stack)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer func() { _ = stmt.Close() }()

		for _, entry := range entries {
			_, execErr := stmt.ExecContext(ctx,
				entry.Timestamp.Format(time.RFC3339Nano), entry.Level, entry.Layer, entry.Message,
				nullString(entry.Component), nullString(entry.RequestID), nullString(entry.SessionID),
				entry.DurationMs, nullString(entry.Metadata), nullString(entry.Stack))
			if execErr != nil {
				return fmt.Errorf("failed to insert log entry: %w", execErr)
			}
		}
		return nil
	})
}

// LogListOptions specifies criteria for listing logs.
type LogListOptions struct {
	Level     string    // Filter by level (ERROR, WARN, INFO, DEBUG)
	Layer     string    // Filter by layer (backend, api, frontend)
	Component string    // Filter by component
	RequestID string    // Filter by request ID
	Since     time.Time // Filter logs after this time
	Until     time.Time // Filter logs before this time
	Search    string    // Search in message text
	Limit     int       // Maximum entries to return
	Offset    int       // Pagination offset
}

// List retrieves log entries matching the criteria.
//

func (r *LogRepository) List(ctx context.Context, opts LogListOptions) ([]*LogEntry, error) {
	query := `
		SELECT id, timestamp, level, layer, message, component, request_id, session_id, duration_ms, metadata_json, stack
		FROM logs
		WHERE 1=1
	`
	var args []any

	if opts.Level != "" {
		query += " AND level = ?"
		args = append(args, opts.Level)
	}

	if opts.Layer != "" {
		query += " AND layer = ?"
		args = append(args, opts.Layer)
	}

	if opts.Component != "" {
		query += " AND component = ?"
		args = append(args, opts.Component)
	}

	if opts.RequestID != "" {
		query += " AND request_id = ?"
		args = append(args, opts.RequestID)
	}

	if !opts.Since.IsZero() {
		query += sqlAndTimestampGte
		args = append(args, opts.Since.Format(time.RFC3339Nano))
	}

	if !opts.Until.IsZero() {
		query += sqlAndTimestampLte
		args = append(args, opts.Until.Format(time.RFC3339Nano))
	}

	if opts.Search != "" {
		query += " AND message LIKE ?"
		args = append(args, "%"+opts.Search+"%")
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += sqlLimit
		args = append(args, opts.Limit)
	} else {
		query += " LIMIT 1000" // Default limit
	}

	if opts.Offset > 0 {
		query += sqlOffset
		args = append(args, opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []*LogEntry
	for rows.Next() {
		entry, scanErr := r.scanLogEntry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetRecent returns the most recent n log entries.
func (r *LogRepository) GetRecent(ctx context.Context, n int) ([]*LogEntry, error) {
	return r.List(ctx, LogListOptions{Limit: n})
}

// Count returns the total number of log entries.
func (r *LogRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	row := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM logs")
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count logs: %w", err)
	}
	return count, nil
}

// DeleteOlderThan removes log entries older than the given time.
func (r *LogRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		DELETE FROM logs WHERE timestamp < ?
	`, before.Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", err)
	}
	return result.RowsAffected()
}

// Clear removes all log entries.
func (r *LogRepository) Clear(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "DELETE FROM logs")
	if err != nil {
		return fmt.Errorf("failed to clear logs: %w", err)
	}
	return nil
}

// scanLogEntry scans a log entry from rows.
func (r *LogRepository) scanLogEntry(rows *sql.Rows) (*LogEntry, error) {
	var entry LogEntry
	var timestamp string
	var component, requestID, sessionID, metadata, stack sql.NullString

	err := rows.Scan(
		&entry.ID, &timestamp, &entry.Level, &entry.Layer, &entry.Message,
		&component, &requestID, &sessionID, &entry.DurationMs, &metadata, &stack,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan log entry: %w", err)
	}

	// Parse timestamp
	if tNano, parseNanoErr := time.Parse(time.RFC3339Nano, timestamp); parseNanoErr == nil {
		entry.Timestamp = tNano
	} else if tRFC, parseRFCErr := time.Parse(time.RFC3339, timestamp); parseRFCErr == nil {
		entry.Timestamp = tRFC
	}

	entry.Component = component.String
	entry.RequestID = requestID.String
	entry.SessionID = sessionID.String
	entry.Metadata = metadata.String
	entry.Stack = stack.String

	return &entry, nil
}

// nullString returns an empty interface suitable for SQL NULL if the string is empty.
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ConvertMetadataToJSON converts a map to JSON string for storage.
func ConvertMetadataToJSON(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return ""
	}
	return string(data)
}
