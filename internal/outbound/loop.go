package outbound

import (
	"context"
	"sync"
	"time"
)

// Loop runs one integration's refresh at start, on every tick and whenever
// [Loop.Wake] is called. It implements engine.Engine. The refresh itself
// decides whether the operator's answer lets it call out.
type Loop struct {
	name     string
	interval time.Duration
	refresh  func(context.Context)
	wake     chan struct{}

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewLoop returns a loop named name that calls refresh every interval.
func NewLoop(name string, interval time.Duration, refresh func(context.Context)) *Loop {
	return &Loop{name: name, interval: interval, refresh: refresh, wake: make(chan struct{}, 1)}
}

// Name implements engine.Engine.
func (l *Loop) Name() string { return l.name }

// Start refreshes once now and then on every tick or wake. It is a no-op when
// already started.
func (l *Loop) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel
	l.started = true
	l.wg.Add(1)
	go l.run(loopCtx)
	return nil
}

// Stop ends the loop and waits, up to ctx's deadline, for an in-flight
// refresh to finish.
func (l *Loop) Stop(ctx context.Context) error {
	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return nil
	}
	cancel := l.cancel
	l.mu.Unlock()
	cancel()

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	l.mu.Lock()
	l.started = false
	l.mu.Unlock()
	return nil
}

// Wake asks for a refresh now, after the operator changed the settings.
func (l *Loop) Wake() {
	select {
	case l.wake <- struct{}{}:
	default:
	}
}

func (l *Loop) run(ctx context.Context) {
	defer l.wg.Done()
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()
	for {
		l.refresh(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-l.wake:
		}
	}
}
