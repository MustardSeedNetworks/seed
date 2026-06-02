// Package events is seed's in-process domain event bus (ADR-0004).
//
// Modules never import one another; instead they publish and subscribe to typed
// domain facts through a Bus. An event is a past-tense fact ("DeviceDiscovered"),
// never a command — it notifies, it does not request. That rule is what keeps
// the bus a decoupling seam rather than hidden control-flow.
//
// Delivery semantics:
//   - In-process, single binary — no broker.
//   - Asynchronous: Publish enqueues and returns; handlers run on the bus's own
//     per-subscriber goroutines.
//   - Ordered per subscriber: a subscriber sees events in the order they were
//     enqueued for it.
//   - At-least-once for the process lifetime: every event accepted by Publish
//     before Close is delivered, even if it was still queued when Close ran.
//     Durability across restarts (the transactional outbox) is layered on
//     separately by the persistence phase; this bus is the in-memory core.
//   - Panic-isolated: a handler that panics is recovered and logged; it never
//     crashes the publisher or a sibling subscriber, and the subscriber keeps
//     receiving later events.
//
// Subscribe before the publishers start (the supervisor wires subscribers
// first) so no early event is missed.
package events

import (
	"context"
	"log/slog"
	"sync"
)

// Event is a domain fact. Implementations carry whatever payload subscribers
// need and report the topic they are published under.
type Event interface {
	// Topic is the stable channel name the event is routed on, e.g.
	// "device.discovered". Subscribers register by topic.
	Topic() string
}

// Handler reacts to an event. It runs asynchronously from Publish on the bus's
// own goroutine. A panic inside a Handler is recovered and logged; it does not
// propagate to the publisher or other subscribers.
type Handler func(ctx context.Context, ev Event)

// Bus is an in-process domain event bus. The zero value is not usable; call
// New. A Bus is safe for concurrent use by multiple publishers and subscribers.
type Bus struct {
	logger *slog.Logger

	mu     sync.Mutex
	subs   map[string][]*subscription
	closed bool
	wg     sync.WaitGroup
}

// subscription is one handler's ordered, unbounded delivery queue plus the
// goroutine draining it. The queue is unbounded so Publish never blocks and
// never drops a fact; a persistently slow subscriber is a backpressure concern
// to bound in a later iteration, not a correctness one here.
type subscription struct {
	topic   string
	handler Handler

	mu      sync.Mutex
	cond    *sync.Cond
	queue   []Event
	stopped bool
}

// New returns a started bus that logs subscriber panics to logger.
func New(logger *slog.Logger) *Bus {
	return &Bus{
		logger: logger,
		subs:   make(map[string][]*subscription),
	}
}

// Subscribe registers handler for events published on topic and returns a
// cancel function that stops further delivery to it. Calling cancel more than
// once is safe. Subscribing on a closed bus is a no-op.
func (b *Bus) Subscribe(topic string, handler Handler) func() {
	sub := &subscription{topic: topic, handler: handler}
	sub.cond = sync.NewCond(&sub.mu)

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return func() {}
	}
	b.subs[topic] = append(b.subs[topic], sub)
	b.wg.Add(1)
	b.mu.Unlock()

	go b.run(sub)

	var once sync.Once
	return func() { once.Do(func() { b.unsubscribe(topic, sub) }) }
}

// Publish enqueues ev for every current subscriber of its topic and returns
// immediately. It is a no-op once the bus is closed.
func (b *Bus) Publish(ev Event) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	subs := b.subs[ev.Topic()]
	targets := make([]*subscription, len(subs))
	copy(targets, subs)
	b.mu.Unlock()

	for _, sub := range targets {
		sub.enqueue(ev)
	}
}

// Close stops accepting new events, drains every subscriber's queued events,
// and waits for the in-flight handlers to finish. It returns nil once the bus
// has fully drained, or ctx.Err() if ctx is cancelled first. Close is
// idempotent.
func (b *Bus) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	all := make([]*subscription, 0, len(b.subs))
	for _, subs := range b.subs {
		all = append(all, subs...)
	}
	b.subs = make(map[string][]*subscription)
	b.mu.Unlock()

	for _, sub := range all {
		sub.stop()
	}

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// run drains a subscription's queue in order until it is stopped and empty.
func (b *Bus) run(sub *subscription) {
	defer b.wg.Done()

	for {
		sub.mu.Lock()
		for len(sub.queue) == 0 && !sub.stopped {
			sub.cond.Wait()
		}
		if len(sub.queue) == 0 {
			// Stopped and drained.
			sub.mu.Unlock()
			return
		}
		ev := sub.queue[0]
		sub.queue = sub.queue[1:]
		sub.mu.Unlock()

		b.deliver(sub, ev)
	}
}

// deliver invokes the handler with panic isolation.
func (b *Bus) deliver(sub *subscription, ev Event) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("event subscriber panicked",
				"topic", sub.topic, "panic", r)
		}
	}()
	sub.handler(context.Background(), ev)
}

// unsubscribe removes target from its topic's fan-out set and stops its worker.
func (b *Bus) unsubscribe(topic string, target *subscription) {
	b.mu.Lock()
	subs := b.subs[topic]
	for i, sub := range subs {
		if sub == target {
			b.subs[topic] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	b.mu.Unlock()

	target.stop()
}

// enqueue appends ev to the subscription's queue and wakes its worker. It is a
// no-op once the subscription is stopped.
func (s *subscription) enqueue(ev Event) {
	s.mu.Lock()
	if !s.stopped {
		s.queue = append(s.queue, ev)
		s.cond.Signal()
	}
	s.mu.Unlock()
}

// stop signals the worker to drain its queue and exit.
func (s *subscription) stop() {
	s.mu.Lock()
	s.stopped = true
	s.cond.Signal()
	s.mu.Unlock()
}
