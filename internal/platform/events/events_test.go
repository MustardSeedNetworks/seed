package events_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/platform/events"
)

// testEvent is a minimal domain fact used by the bus tests.
type testEvent struct {
	topic string
	id    string
}

func (e testEvent) Topic() string { return e.topic }

// recorder collects the events a subscriber receives, guarded for the race
// detector since delivery happens on the bus's own goroutines.
type recorder struct {
	mu   sync.Mutex
	got  []string
	hook func() // optional per-delivery side effect (e.g. panic, signal)
}

func (r *recorder) handler(_ context.Context, ev events.Event) {
	if r.hook != nil {
		r.hook()
	}
	te, ok := ev.(testEvent)
	if !ok {
		return
	}
	r.mu.Lock()
	r.got = append(r.got, te.id)
	r.mu.Unlock()
}

func (r *recorder) ids() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.got))
	copy(out, r.got)
	return out
}

// quietBus returns a bus that discards log output (panicking subscribers log).
func quietBus(t *testing.T) *events.Bus {
	t.Helper()
	return events.New(slog.New(slog.DiscardHandler))
}

// closeBus drains and stops the bus with a generous deadline, failing the test
// if the drain does not complete (a hang means a delivery/shutdown bug).
func closeBus(t *testing.T, bus *events.Bus) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := bus.Close(ctx); err != nil {
		t.Fatalf("Close drained incompletely: %v", err)
	}
}

func TestBusDeliversEventToSubscriber(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	var rec recorder
	bus.Subscribe("device.discovered", rec.handler)

	bus.Publish(testEvent{topic: "device.discovered", id: "a"})
	closeBus(t, bus) // draining guarantees delivery without sleeps

	if got := rec.ids(); len(got) != 1 || got[0] != "a" {
		t.Fatalf("subscriber got %v, want [a]", got)
	}
}

func TestBusRoutesByTopic(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	var onTopicA, onTopicB recorder
	bus.Subscribe("topic.a", onTopicA.handler)
	bus.Subscribe("topic.b", onTopicB.handler)

	bus.Publish(testEvent{topic: "topic.a", id: "a1"})
	bus.Publish(testEvent{topic: "topic.b", id: "b1"})
	bus.Publish(testEvent{topic: "topic.a", id: "a2"})
	closeBus(t, bus)

	if got := onTopicA.ids(); len(got) != 2 {
		t.Fatalf("topic.a subscriber got %v, want 2 events", got)
	}
	if got := onTopicB.ids(); len(got) != 1 || got[0] != "b1" {
		t.Fatalf("topic.b subscriber got %v, want [b1]", got)
	}
}

func TestBusFansOutToMultipleSubscribers(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	var first, second recorder
	bus.Subscribe("device.discovered", first.handler)
	bus.Subscribe("device.discovered", second.handler)

	bus.Publish(testEvent{topic: "device.discovered", id: "x"})
	closeBus(t, bus)

	if got := first.ids(); len(got) != 1 || got[0] != "x" {
		t.Fatalf("first subscriber got %v, want [x]", got)
	}
	if got := second.ids(); len(got) != 1 || got[0] != "x" {
		t.Fatalf("second subscriber got %v, want [x]", got)
	}
}

func TestBusPreservesPerSubscriberOrder(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	var rec recorder
	bus.Subscribe("scan.progress", rec.handler)

	const n = 100
	want := make([]string, n)
	for i := range n {
		id := fmt.Sprintf("e%03d", i)
		want[i] = id
		bus.Publish(testEvent{topic: "scan.progress", id: id})
	}
	closeBus(t, bus)

	got := rec.ids()
	if len(got) != n {
		t.Fatalf("got %d events, want %d", len(got), n)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event %d out of order: got %s, want %s (per-topic ordering broken)", i, got[i], want[i])
		}
	}
}

func TestBusIsolatesPanickingSubscriber(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	healthy := recorder{}
	panicker := recorder{hook: func() { panic("subscriber boom") }}

	bus.Subscribe("device.discovered", panicker.handler)
	bus.Subscribe("device.discovered", healthy.handler)

	// A panicking subscriber must neither crash the bus nor stop a sibling
	// subscriber from receiving this or subsequent events.
	bus.Publish(testEvent{topic: "device.discovered", id: "p1"})
	bus.Publish(testEvent{topic: "device.discovered", id: "p2"})
	closeBus(t, bus)

	if got := healthy.ids(); len(got) != 2 {
		t.Fatalf("healthy subscriber got %v despite sibling panics, want 2 events", got)
	}
}

func TestBusUnsubscribeStopsDelivery(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	delivered := make(chan struct{}, 1)
	var rec recorder
	rec.hook = func() { delivered <- struct{}{} }

	cancel := bus.Subscribe("device.discovered", rec.handler)
	bus.Publish(testEvent{topic: "device.discovered", id: "before"})

	select {
	case <-delivered:
	case <-time.After(5 * time.Second):
		t.Fatal("first event never delivered")
	}

	cancel()
	bus.Publish(testEvent{topic: "device.discovered", id: "after"})
	closeBus(t, bus)

	if got := rec.ids(); len(got) != 1 || got[0] != "before" {
		t.Fatalf("after unsubscribe got %v, want only [before]", got)
	}
}

func TestBusCloseDrainsPendingEvents(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	var rec recorder
	// Each delivery is slow, so events queue up; Close must drain them all
	// rather than dropping the backlog (at-least-once).
	rec.hook = func() { time.Sleep(time.Millisecond) }
	bus.Subscribe("scan.progress", rec.handler)

	const n = 50
	for i := range n {
		bus.Publish(testEvent{topic: "scan.progress", id: fmt.Sprintf("e%d", i)})
	}
	closeBus(t, bus)

	if got := rec.ids(); len(got) != n {
		t.Fatalf("Close dropped queued events: got %d, want %d", len(got), n)
	}
}

func TestBusPublishAfterCloseIsNoop(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	var rec recorder
	bus.Subscribe("device.discovered", rec.handler)
	closeBus(t, bus)

	// Must not panic and must not deliver.
	bus.Publish(testEvent{topic: "device.discovered", id: "late"})

	if got := rec.ids(); len(got) != 0 {
		t.Fatalf("publish after Close delivered %v, want none", got)
	}
}

func TestBusConcurrentPublishIsRaceFree(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	var rec recorder
	bus.Subscribe("device.discovered", rec.handler)

	const publishers, each = 8, 50
	var wg sync.WaitGroup
	wg.Add(publishers)
	for p := range publishers {
		go func(p int) {
			defer wg.Done()
			for i := range each {
				bus.Publish(testEvent{topic: "device.discovered", id: fmt.Sprintf("p%d-%d", p, i)})
			}
		}(p)
	}
	wg.Wait()
	closeBus(t, bus)

	if got := rec.ids(); len(got) != publishers*each {
		t.Fatalf("got %d events from concurrent publishers, want %d", len(got), publishers*each)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	bus := quietBus(t)
	bus.Subscribe("device.discovered", func(context.Context, events.Event) {})
	closeBus(t, bus)
	closeBus(t, bus) // second Close must be a safe no-op
}
