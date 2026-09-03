package api

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// fakeTracer returns a canned trace and counts how many rounds ran, which is
// what the probe budget is asserted against.
type fakeTracer struct {
	mu      sync.Mutex
	rounds  int
	maxHops int
	onTrace func(round int)
}

func (f *fakeTracer) TraceICMP(_ context.Context, target string) *discovery.TracerouteResult {
	f.mu.Lock()
	f.rounds++
	round := f.rounds
	onTrace := f.onTrace
	f.mu.Unlock()

	if onTrace != nil {
		onTrace(round)
	}
	return &discovery.TracerouteResult{
		Target: target,
		Hops: []discovery.TracerouteHop{
			{TTL: 1, IP: "10.0.0.1", RTT: time.Millisecond, State: "reply"},
		},
	}
}

func (f *fakeTracer) roundCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rounds
}

// testDeps wires a monitor to a fake tracer with a client always attached, so
// the idle guard stays out of the way unless a test asks for it.
func testDeps(tracer *fakeTracer, published *[]PathMonitorUpdate, mu *sync.Mutex) pathMonitorDeps {
	return pathMonitorDeps{
		newTracer: func(maxHops int) pathTracer {
			tracer.maxHops = maxHops
			return tracer
		},
		publish: func(update PathMonitorUpdate) {
			mu.Lock()
			defer mu.Unlock()
			*published = append(*published, update)
		},
		clientCount: func() int { return 1 },
		now:         time.Now,
	}
}

// Cancelling is the normal way a monitor ends, and the evidence it gathered is
// the reason the operator started it. Discarding the snapshot on stop would
// throw away the answer they were waiting for.
func TestCancelledMonitorReturnsWhatItLearned(t *testing.T) {
	tracer := &fakeTracer{}
	var mu sync.Mutex
	var published []PathMonitorUpdate

	ctx, cancel := context.WithCancel(t.Context())
	tracer.onTrace = func(round int) {
		if round == 3 {
			cancel()
		}
	}

	snapshot := runPathMonitor(
		ctx,
		PathMonitorRequest{Destination: "example.test"},
		testDeps(tracer, &published, &mu),
		nil,
	)

	if snapshot.Rounds < 3 {
		t.Fatalf("rounds = %d, want at least 3 before cancellation", snapshot.Rounds)
	}
	if len(snapshot.Hops) != 1 || snapshot.Hops[0].Received != snapshot.Rounds {
		t.Errorf("hops = %+v, want one hop that answered every round", snapshot.Hops)
	}
}

// The probe budget is a number, not an adjective: EDITIONS.md targets a ~5W
// Pi-class unit, so a monitor must not free-run.
func TestRoundsArePacedByTheInterval(t *testing.T) {
	tracer := &fakeTracer{}
	var mu sync.Mutex
	var published []PathMonitorUpdate

	ctx, cancel := context.WithTimeout(t.Context(), 1500*time.Millisecond)
	defer cancel()

	runPathMonitor(ctx, PathMonitorRequest{Destination: "example.test"}, testDeps(tracer, &published, &mu), nil)

	// One round immediately, one after the 1s floor. A third would mean the
	// interval was not honoured.
	if got := tracer.roundCount(); got > 2 {
		t.Errorf("ran %d rounds in 1.5s at a 1s interval, want at most 2", got)
	}
	if tracer.roundCount() == 0 {
		t.Error("ran no rounds at all")
	}
}

// A monitor nobody is watching is spending a Pi's power budget on nothing.
// /api/events is a broadcast, so the server cannot tell which page navigated
// away -- the idle guard is what stops the loop when the explicit cancel that
// should have arrived did not.
func TestMonitorStopsWhenNobodyIsListening(t *testing.T) {
	tracer := &fakeTracer{}
	clock := time.Now()

	deps := pathMonitorDeps{
		newTracer:   func(int) pathTracer { return tracer },
		clientCount: func() int { return 0 },
		now: func() time.Time {
			// Each consultation advances past the grace period, so the second
			// round finds the monitor unobserved for long enough to stop.
			clock = clock.Add(pathMonitorIdleGrace)
			return clock
		},
	}

	snapshot := runPathMonitor(t.Context(), PathMonitorRequest{Destination: "example.test"}, deps, nil)

	if tracer.roundCount() > 2 {
		t.Errorf("ran %d rounds unobserved, want the idle guard to stop it", tracer.roundCount())
	}
	if snapshot.Rounds == 0 {
		t.Error("returned no snapshot; a monitor that stops still reports what it saw")
	}
}

// A client attached at any point resets the grace period -- an operator who
// reloads the page has not abandoned the monitor.
func TestAttachedClientKeepsTheMonitorAlive(t *testing.T) {
	idleSince, stop := pathMonitorIdleCheck(
		pathMonitorDeps{clientCount: func() int { return 1 }, now: time.Now},
		time.Now().Add(-time.Hour),
	)
	if stop {
		t.Error("stopped a monitor that has a client attached")
	}
	if !idleSince.IsZero() {
		t.Error("kept an idle timestamp while a client is attached")
	}
}

func TestEveryRoundIsPublished(t *testing.T) {
	tracer := &fakeTracer{}
	var mu sync.Mutex
	var published []PathMonitorUpdate

	ctx, cancel := context.WithCancel(t.Context())
	tracer.onTrace = func(round int) {
		if round == 2 {
			cancel()
		}
	}

	runPathMonitor(ctx, PathMonitorRequest{Destination: "example.test"}, testDeps(tracer, &published, &mu), nil)

	mu.Lock()
	defer mu.Unlock()
	if len(published) < 2 {
		t.Fatalf("published %d updates, want one per round", len(published))
	}
	// Each update carries the whole accumulated view, not just the new round,
	// so a client that joins late is not missing history.
	if published[len(published)-1].Rounds != len(published) {
		t.Errorf("last update reports %d rounds after %d publishes", published[len(published)-1].Rounds, len(published))
	}
}

func TestIntervalIsClampedIntoTheBudget(t *testing.T) {
	for _, tc := range []struct {
		name     string
		seconds  int
		expected time.Duration
	}{
		{"unset falls to the floor", 0, pathMonitorMinInterval},
		{"below the floor is raised", -5, pathMonitorMinInterval},
		{"in range is honoured", 5, 5 * time.Second},
		{"above the ceiling is capped", 3600, pathMonitorMaxInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := pathMonitorInterval(tc.seconds); got != tc.expected {
				t.Errorf("pathMonitorInterval(%d) = %v, want %v", tc.seconds, got, tc.expected)
			}
		})
	}
}

func TestHopsAreClampedIntoTheBudget(t *testing.T) {
	if got := pathMonitorHops(0); got != pathMonitorMaxHops {
		t.Errorf("unset hops = %d, want %d", got, pathMonitorMaxHops)
	}
	if got := pathMonitorHops(500); got != pathMonitorMaxHops {
		t.Errorf("excessive hops = %d, want %d", got, pathMonitorMaxHops)
	}
	if got := pathMonitorHops(5); got != 5 {
		t.Errorf("in-range hops = %d, want 5", got)
	}
}

func TestHandlerRejectsMissingDestination(t *testing.T) {
	handler := newPathMonitorHandler(pathMonitorDeps{now: time.Now})
	params := json.RawMessage(`{"intervalSeconds":5}`)

	if _, err := handler(t.Context(), params, nil); err == nil {
		t.Fatal("handler accepted a monitor with no destination")
	}
}

func TestHandlerRejectsAbsentParams(t *testing.T) {
	handler := newPathMonitorHandler(pathMonitorDeps{now: time.Now})
	if _, err := handler(t.Context(), nil, nil); err == nil {
		t.Fatal("handler accepted a job with no params")
	}
}
