package outbox_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/outbox"
)

// fakeStore is an in-memory outbox.Store for the relay tests. It is safe for
// concurrent use since the relay's ticker may drain from its own goroutine.
type fakeStore struct {
	mu        sync.Mutex
	records   []outbox.Record
	published map[string]bool
	failMark  bool // when true, MarkPublished returns an error (at-least-once probe)
	markCalls [][]string
}

func newFakeStore(recs ...outbox.Record) *fakeStore {
	return &fakeStore{records: recs, published: map[string]bool{}}
}

func (s *fakeStore) FetchUnpublished(_ context.Context, limit int) ([]outbox.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []outbox.Record
	for _, r := range s.records {
		if s.published[r.ID] {
			continue
		}
		out = append(out, r)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) MarkPublished(_ context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failMark {
		return errors.New("mark failed")
	}
	s.markCalls = append(s.markCalls, ids)
	for _, id := range ids {
		s.published[id] = true
	}
	return nil
}

func (s *fakeStore) DeletePublishedBefore(_ context.Context, _ time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	kept := s.records[:0]
	for _, r := range s.records {
		if s.published[r.ID] {
			n++
			continue
		}
		kept = append(kept, r)
	}
	s.records = kept
	return n, nil
}

// recorder collects Message IDs a subscriber receives, guarded for the race
// detector (delivery is on the bus's own goroutines).
type recorder struct {
	mu  sync.Mutex
	got []string
}

func (r *recorder) handle(_ context.Context, ev events.Event) {
	m, ok := ev.(outbox.Message)
	if !ok {
		return
	}
	r.mu.Lock()
	r.got = append(r.got, m.ID)
	r.mu.Unlock()
}

func (r *recorder) ids() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.got))
	copy(out, r.got)
	return out
}

func quiet() *slog.Logger { return slog.New(slog.DiscardHandler) }

func drainBus(t *testing.T, bus *events.Bus) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := bus.Close(ctx); err != nil {
		t.Fatalf("bus close: %v", err)
	}
}

func TestRelayDrainPublishesOnTopicAndMarks(t *testing.T) {
	t.Parallel()
	store := newFakeStore(
		outbox.Record{ID: "1", Topic: "device.discovered", Payload: []byte(`{"id":"a"}`)},
		outbox.Record{ID: "2", Topic: "device.discovered", Payload: []byte(`{"id":"b"}`)},
	)
	bus := events.New(quiet())
	var rec recorder
	bus.Subscribe("device.discovered", rec.handle)

	relay := outbox.NewRelay(store, bus, quiet())
	n, err := relay.Drain(context.Background())
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if n != 2 {
		t.Fatalf("drained %d, want 2", n)
	}
	drainBus(t, bus)

	if got := rec.ids(); len(got) != 2 || got[0] != "1" || got[1] != "2" {
		t.Fatalf("subscriber got %v, want [1 2] on the original topic", got)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if !store.published["1"] || !store.published["2"] {
		t.Fatalf("rows not marked published: %v", store.published)
	}
}

// TestRelayRepublishesAfterRestart models a process restart: rows enqueued by a
// prior process are still unpublished (its ephemeral subscriber is gone). A fresh
// relay + subscriber must redeliver them — durability across restart.
func TestRelayRepublishesAfterRestart(t *testing.T) {
	t.Parallel()
	store := newFakeStore(
		outbox.Record{ID: "7", Topic: "survey.completed", Payload: []byte(`{}`)},
	)
	bus := events.New(quiet())
	var rec recorder
	bus.Subscribe("survey.completed", rec.handle)

	relay := outbox.NewRelay(store, bus, quiet())
	relay.Start(context.Background()) // initial drain == the across-restart replay
	t.Cleanup(relay.Stop)
	drainBus(t, bus)

	if got := rec.ids(); len(got) != 1 || got[0] != "7" {
		t.Fatalf("after restart got %v, want [7]", got)
	}
}

// TestRelayAtLeastOnceOnMarkFailure: if MarkPublished fails after the publish,
// the rows stay pending and a later drain republishes them — at-least-once.
func TestRelayAtLeastOnceOnMarkFailure(t *testing.T) {
	t.Parallel()
	store := newFakeStore(
		outbox.Record{ID: "1", Topic: "t", Payload: []byte(`x`)},
	)
	store.failMark = true
	bus := events.New(quiet())
	var rec recorder
	bus.Subscribe("t", rec.handle)

	relay := outbox.NewRelay(store, bus, quiet())
	if _, err := relay.Drain(context.Background()); err == nil {
		t.Fatal("drain should surface the mark failure")
	}
	// Recover the store and drain again — the row must still be deliverable.
	store.mu.Lock()
	store.failMark = false
	store.mu.Unlock()
	if _, err := relay.Drain(context.Background()); err != nil {
		t.Fatalf("second drain: %v", err)
	}
	drainBus(t, bus)

	// The subscriber saw the event at least once (here twice — that is why
	// consumers must dedupe).
	if got := rec.ids(); len(got) < 1 {
		t.Fatalf("at-least-once violated: got %v", got)
	}
}

func TestRelayCleanupDeletesPublished(t *testing.T) {
	t.Parallel()
	store := newFakeStore(
		outbox.Record{ID: "1", Topic: "t", Payload: []byte(`x`)},
	)
	bus := events.New(quiet())
	bus.Subscribe("t", func(context.Context, events.Event) {})
	relay := outbox.NewRelay(store, bus, quiet())
	if _, err := relay.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	drainBus(t, bus)

	n, err := relay.Cleanup(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if n != 1 {
		t.Fatalf("cleanup removed %d, want 1", n)
	}
}

func TestRelayStartStopIdempotent(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := events.New(quiet())
	relay := outbox.NewRelay(store, bus, quiet())
	relay.Start(context.Background())
	relay.Start(context.Background()) // second start is a no-op
	relay.Stop()
	relay.Stop() // second stop is a no-op
	drainBus(t, bus)
}

func TestDedupeSkipsRepeatedID(t *testing.T) {
	t.Parallel()
	bus := events.New(quiet())
	var rec recorder
	d := outbox.NewDeduper(16)
	bus.Subscribe("t", outbox.Dedupe(d, rec.handle))

	bus.Publish(outbox.Message{ID: "dup", EventTopic: "t", Payload: []byte(`x`)})
	bus.Publish(outbox.Message{ID: "dup", EventTopic: "t", Payload: []byte(`x`)})
	bus.Publish(outbox.Message{ID: "other", EventTopic: "t", Payload: []byte(`y`)})
	drainBus(t, bus)

	if got := rec.ids(); len(got) != 2 || got[0] != "dup" || got[1] != "other" {
		t.Fatalf("dedupe got %v, want [dup other] (second dup skipped)", got)
	}
}

// TestDeduperForgetsBeyondCapacity documents the bounded contract: Seen reports
// true for a duplicate it still remembers, but an id evicted once capacity is
// exceeded is forgotten (and would be reprocessed). The window must be sized
// above the relay's redelivery horizon.
func TestDeduperForgetsBeyondCapacity(t *testing.T) {
	t.Parallel()
	d := outbox.NewDeduper(2)

	if d.Seen("a") {
		t.Fatal("first sight of a should be new")
	}
	if d.Seen("b") {
		t.Fatal("first sight of b should be new")
	}
	if !d.Seen("b") {
		t.Fatal("b is still in the window, should be a duplicate")
	}
	// "c" evicts the oldest tracked id ("a").
	if d.Seen("c") {
		t.Fatal("first sight of c should be new")
	}
	if d.Seen("a") {
		t.Fatal("a was evicted, should read as new again")
	}
}
