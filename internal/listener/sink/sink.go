// Package sink is the default [listener.Sink] implementation. It
// persists every passive-ingress event into the listener_events
// table — the place Stage A4 alert rules and the operator-facing
// event log read from.
//
// Listeners (syslog, snmp traps, etc.) call Sink.Publish; the sink
// writes one row per event. Marshalling of the payload is performed
// once at write time so the row's payload_json carries exactly what
// the listener observed.
package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/listener"
)

// eventsStore is the narrowed surface the sink uses. Tests inject a
// fake.
type eventsStore interface {
	Insert(ctx context.Context, evt *database.ListenerEvent) error
}

// Sink persists [listener.Event] instances into listener_events.
// Implements [listener.Sink].
type Sink struct {
	store  eventsStore
	logger *slog.Logger
	now    func() time.Time
}

// New returns a Sink bound to store. Pass nil logger to use
// [slog.Default]; pass nil now to use [time.Now] in UTC.
func New(store eventsStore, logger *slog.Logger, now func() time.Time) *Sink {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Sink{store: store, logger: logger, now: now}
}

// Publish implements [listener.Sink]. Errors are logged at warn
// because dropping an event is preferable to blocking the listener's
// read loop — a slow database must not back-pressure inbound syslog
// or trap reception.
func (s *Sink) Publish(ctx context.Context, evt listener.Event) error {
	// Payload arrives as json.RawMessage already; if empty, write a
	// JSON null so downstream consumers don't have to special-case.
	payload := string(evt.Payload)
	if payload == "" {
		payload = "null"
	}
	if !json.Valid([]byte(payload)) {
		s.logger.WarnContext(ctx, "listener sink: invalid JSON payload, dropping",
			"kind", evt.Kind, "source", evt.SourceAddr)
		return fmt.Errorf("listener sink: invalid JSON payload for %s", evt.Kind)
	}

	if err := s.store.Insert(ctx, &database.ListenerEvent{
		ClientID:    evt.ClientID,
		Kind:        evt.Kind,
		SourceAddr:  evt.SourceAddr,
		TargetKind:  evt.TargetKind,
		TargetID:    evt.TargetID,
		Severity:    evt.Severity,
		ObservedAt:  evt.Timestamp,
		PayloadJSON: payload,
		IngestedAt:  s.now(),
	}); err != nil {
		s.logger.WarnContext(ctx, "listener sink: insert failed",
			"kind", evt.Kind, "source", evt.SourceAddr, "error", err)
		return fmt.Errorf("listener sink: insert %s: %w", evt.Kind, err)
	}
	return nil
}
