package app

// logs.go wires the composition root to the log-query use-case (ADR-0020): a Store
// adapter over the database log repository and a Buffer adapter over the in-memory
// log broadcaster. It also supplies the broadcaster's database writer. The database handle is resolved through a lazy accessor so a
// later-set value (the api test harness) is honored and a nil handle degrades the
// store to "unavailable" rather than panicking.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	logquery "github.com/MustardSeedNetworks/seed/internal/logs/query"
)

// NewLogQuery builds the log-query use-case over the database log store and the
// in-memory broadcaster buffer.
func NewLogQuery(db func() *database.DB) *logquery.Service {
	return logquery.NewService(logStore{db: db}, logBuffer{})
}

// logStore implements logquery.Store over the database log repository, resolving
// the handle lazily.
type logStore struct {
	db func() *database.DB
}

func (a logStore) Available() bool { return a.db() != nil }

func (a logStore) List(ctx context.Context, p logquery.Params) ([]*logging.LogEntry, error) {
	opts := database.LogListOptions{
		Search: p.Search,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	// The store filters on the first value of each multi-valued filter — the
	// original handler's contract.
	if len(p.Levels) > 0 {
		opts.Level = p.Levels[0]
	}
	if len(p.Layers) > 0 {
		opts.Layer = p.Layers[0]
	}
	if len(p.Components) > 0 {
		opts.Component = p.Components[0]
	}

	entries, err := a.db().Logs().List(ctx, opts)
	if err != nil {
		return nil, err
	}
	out := make([]*logging.LogEntry, len(entries))
	for i, e := range entries {
		out[i] = &logging.LogEntry{
			Timestamp:  e.Timestamp,
			Level:      e.Level,
			Layer:      e.Layer,
			Message:    e.Message,
			Component:  e.Component,
			RequestID:  e.RequestID,
			SessionID:  e.SessionID,
			DurationMs: e.DurationMs,
			Stack:      e.Stack,
		}
	}
	return out, nil
}

// logBuffer implements logquery.Buffer over the process-local log broadcaster. A
// nil broadcaster (not yet initialized) reports unavailable.
type logBuffer struct{}

func (logBuffer) Available() bool { return logging.GetBroadcaster() != nil }

func (logBuffer) All() []*logging.LogEntry {
	b := logging.GetBroadcaster()
	if b == nil {
		return nil
	}
	return b.GetAllLogs()
}

// NewLogWriter returns the log broadcaster's database writer.
func NewLogWriter(db *database.DB) logging.DBLogWriter {
	return logWriter{db: db}
}

type logWriter struct {
	db *database.DB
}

func (w logWriter) WriteLog(ctx context.Context, entry *logging.LogEntry) error {
	return w.db.Logs().Create(ctx, toDBLogEntry(entry))
}

func (w logWriter) WriteBatch(ctx context.Context, entries []*logging.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	rows := make([]*database.LogEntry, len(entries))
	for i, e := range entries {
		rows[i] = toDBLogEntry(e)
	}
	return w.db.Logs().BatchCreate(ctx, rows)
}

func toDBLogEntry(e *logging.LogEntry) *database.LogEntry {
	return &database.LogEntry{
		Timestamp:  e.Timestamp,
		Level:      e.Level,
		Layer:      e.Layer,
		Message:    e.Message,
		Component:  e.Component,
		RequestID:  e.RequestID,
		SessionID:  e.SessionID,
		DurationMs: e.DurationMs,
		Metadata:   database.ConvertMetadataToJSON(e.Metadata),
		Stack:      e.Stack,
	}
}
