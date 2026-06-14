package sink_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/listener"
	"github.com/MustardSeedNetworks/seed/internal/listener/sink"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

type fakeStore struct {
	mu   sync.Mutex
	rows []*listener.EventRecord
	err  error
}

func (f *fakeStore) Insert(_ context.Context, evt *listener.EventRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.rows = append(f.rows, evt)
	return nil
}

func staticClock(ts time.Time) func() time.Time {
	return func() time.Time { return ts }
}

func TestPublish_PersistsEvent(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	clock := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	s := sink.New(store, silentLogger(), staticClock(clock))

	evt := listener.Event{
		Kind:       "syslog",
		ClientID:   "client-a",
		SourceAddr: "10.0.0.1:514",
		Severity:   "warning",
		Timestamp:  time.Date(2026, 5, 31, 11, 59, 0, 0, time.UTC),
		Payload:    json.RawMessage(`{"msg":"link down"}`),
	}
	if err := s.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if len(store.rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(store.rows))
	}
	row := store.rows[0]
	if row.Kind != "syslog" || row.ClientID != "client-a" {
		t.Errorf("kind/client = %s/%s", row.Kind, row.ClientID)
	}
	if row.Severity != "warning" {
		t.Errorf("severity = %q", row.Severity)
	}
	if row.PayloadJSON != `{"msg":"link down"}` {
		t.Errorf("payload = %q", row.PayloadJSON)
	}
	if !row.IngestedAt.Equal(clock) {
		t.Errorf("IngestedAt = %v, want %v", row.IngestedAt, clock)
	}
}

func TestPublish_EmptyPayloadBecomesJSONNull(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	s := sink.New(store, silentLogger(), nil)
	if err := s.Publish(context.Background(), listener.Event{
		Kind:       "syslog",
		SourceAddr: "10.0.0.1:514",
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if store.rows[0].PayloadJSON != "null" {
		t.Errorf("empty payload should land as 'null', got %q", store.rows[0].PayloadJSON)
	}
}

func TestPublish_InvalidJSONRejected(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	s := sink.New(store, silentLogger(), nil)
	err := s.Publish(context.Background(), listener.Event{
		Kind:       "syslog",
		SourceAddr: "10.0.0.1:514",
		Payload:    json.RawMessage(`{broken json`),
	})
	if err == nil {
		t.Error("expected invalid JSON to be rejected")
	}
	if len(store.rows) != 0 {
		t.Errorf("invalid JSON should not reach store; rows = %d", len(store.rows))
	}
}

func TestPublish_StoreErrorPropagates(t *testing.T) {
	t.Parallel()
	store := &fakeStore{err: errors.New("disk full")}
	s := sink.New(store, silentLogger(), nil)
	if err := s.Publish(context.Background(), listener.Event{
		Kind:       "syslog",
		SourceAddr: "10.0.0.1:514",
		Payload:    json.RawMessage(`{}`),
	}); err == nil {
		t.Error("expected store error to propagate")
	}
}

func TestNew_NilDefaults(t *testing.T) {
	t.Parallel()
	// Should not panic when logger + now are nil.
	s := sink.New(&fakeStore{}, nil, nil)
	if err := s.Publish(context.Background(), listener.Event{
		Kind:       "syslog",
		SourceAddr: "10.0.0.1:514",
		Payload:    json.RawMessage(`{}`),
	}); err != nil {
		t.Errorf("Publish with nil defaults: %v", err)
	}
}
