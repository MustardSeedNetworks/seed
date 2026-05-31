package engine_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/krisarmstrong/seed/internal/engine"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// fakeEngine records the lifecycle calls it received so tests can
// assert ordering. startErr / stopErr are returned to test rollback
// and error propagation.
type fakeEngine struct {
	mu       sync.Mutex
	name     string
	startErr error
	stopErr  error
	starts   int
	stops    int

	startOrder *[]string
	stopOrder  *[]string
}

func (f *fakeEngine) Name() string { return f.name }

func (f *fakeEngine) Start(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.startErr != nil {
		return f.startErr
	}
	f.starts++
	if f.startOrder != nil {
		*f.startOrder = append(*f.startOrder, f.name)
	}
	return nil
}

func (f *fakeEngine) Stop(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops++
	if f.stopOrder != nil {
		*f.stopOrder = append(*f.stopOrder, f.name)
	}
	return f.stopErr
}

func TestRegistry_RegisterRejectsNilOrEmptyName(t *testing.T) {
	t.Parallel()
	r := engine.NewRegistry(silentLogger())
	if err := r.Register(nil); err == nil {
		t.Error("expected nil engine to be rejected")
	}
	if err := r.Register(&fakeEngine{name: ""}); err == nil {
		t.Error("expected empty Name() to be rejected")
	}
}

func TestRegistry_RegisterRejectsDuplicateName(t *testing.T) {
	t.Parallel()
	r := engine.NewRegistry(silentLogger())
	if err := r.Register(&fakeEngine{name: "probe"}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := r.Register(&fakeEngine{name: "probe"}); err == nil {
		t.Error("expected duplicate name to be rejected")
	}
}

func TestRegistry_StartInRegistrationOrderStopInReverse(t *testing.T) {
	t.Parallel()
	var startOrder, stopOrder []string
	r := engine.NewRegistry(silentLogger())
	for _, n := range []string{"probe", "retention", "snmp"} {
		if err := r.Register(&fakeEngine{
			name:       n,
			startOrder: &startOrder,
			stopOrder:  &stopOrder,
		}); err != nil {
			t.Fatalf("register %s: %v", n, err)
		}
	}

	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	wantStart := []string{"probe", "retention", "snmp"}
	wantStop := []string{"snmp", "retention", "probe"}
	if !equalSlices(startOrder, wantStart) {
		t.Errorf("start order = %v, want %v", startOrder, wantStart)
	}
	if !equalSlices(stopOrder, wantStop) {
		t.Errorf("stop order = %v, want %v", stopOrder, wantStop)
	}
}

func TestRegistry_StartTwiceReturnsErrAlreadyStarted(t *testing.T) {
	t.Parallel()
	r := engine.NewRegistry(silentLogger())
	_ = r.Register(&fakeEngine{name: "probe"})
	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	if err := r.Start(context.Background()); !errors.Is(err, engine.ErrAlreadyStarted) {
		t.Errorf("second Start = %v, want ErrAlreadyStarted", err)
	}
}

func TestRegistry_StartErrorRollsBackStartedEngines(t *testing.T) {
	t.Parallel()
	a := &fakeEngine{name: "a"}
	b := &fakeEngine{name: "b"}
	c := &fakeEngine{name: "c", startErr: errors.New("boom")}
	d := &fakeEngine{name: "d"} // should never start

	r := engine.NewRegistry(silentLogger())
	for _, e := range []engine.Engine{a, b, c, d} {
		if err := r.Register(e); err != nil {
			t.Fatalf("register: %v", err)
		}
	}

	err := r.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start error")
	}

	if a.stops != 1 || b.stops != 1 {
		t.Errorf("a.stops=%d, b.stops=%d, want both = 1", a.stops, b.stops)
	}
	if c.starts != 0 || d.starts != 0 {
		t.Errorf("failed engine + later engines must not have started; starts c=%d d=%d",
			c.starts, d.starts)
	}
	if d.stops != 0 {
		t.Errorf("never-started engine must not have been stopped; d.stops=%d", d.stops)
	}
}

func TestRegistry_StopWithoutStartIsNoOp(t *testing.T) {
	t.Parallel()
	a := &fakeEngine{name: "a"}
	r := engine.NewRegistry(silentLogger())
	_ = r.Register(a)
	if err := r.Stop(context.Background()); err != nil {
		t.Errorf("Stop before Start should be nil, got %v", err)
	}
	if a.stops != 0 {
		t.Errorf("Stop before Start should not invoke engine.Stop, got stops=%d", a.stops)
	}
}

func TestRegistry_StopAfterFailedStartIsNoOp(t *testing.T) {
	t.Parallel()
	a := &fakeEngine{name: "a", startErr: errors.New("boom")}
	r := engine.NewRegistry(silentLogger())
	_ = r.Register(a)
	_ = r.Start(context.Background())
	priorStops := a.stops
	if err := r.Stop(context.Background()); err != nil {
		t.Errorf("Stop after failed Start should be nil, got %v", err)
	}
	if a.stops != priorStops {
		t.Errorf("Stop after failed Start should not re-Stop already-stopped engines")
	}
}

func TestRegistry_StopContinuesPastErrors(t *testing.T) {
	t.Parallel()
	a := &fakeEngine{name: "a", stopErr: errors.New("a stop failed")}
	b := &fakeEngine{name: "b", stopErr: errors.New("b stop failed")}
	c := &fakeEngine{name: "c"}
	r := engine.NewRegistry(silentLogger())
	for _, e := range []engine.Engine{a, b, c} {
		_ = r.Register(e)
	}
	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	err := r.Stop(context.Background())
	if err == nil {
		t.Fatal("expected Stop to surface first error")
	}
	if a.stops != 1 || b.stops != 1 || c.stops != 1 {
		t.Errorf("every engine must have been Stop()ed; stops a=%d b=%d c=%d",
			a.stops, b.stops, c.stops)
	}
}

func TestRegistry_RegisterRejectedAfterStart(t *testing.T) {
	t.Parallel()
	r := engine.NewRegistry(silentLogger())
	_ = r.Register(&fakeEngine{name: "a"})
	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := r.Register(&fakeEngine{name: "b"}); err == nil {
		t.Error("expected Register after Start to be rejected")
	}
}

func TestRegistry_EnginesReturnsSnapshot(t *testing.T) {
	t.Parallel()
	r := engine.NewRegistry(silentLogger())
	for _, n := range []string{"probe", "retention", "snmp"} {
		_ = r.Register(&fakeEngine{name: n})
	}
	got := r.Engines()
	if len(got) != 3 {
		t.Fatalf("Engines() returned %d, want 3", len(got))
	}
	wantNames := []string{"probe", "retention", "snmp"}
	for i, e := range got {
		if e.Name() != wantNames[i] {
			t.Errorf("Engines()[%d].Name() = %q, want %q", i, e.Name(), wantNames[i])
		}
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
