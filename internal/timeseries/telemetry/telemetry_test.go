package telemetry_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/telemetry"
)

type fakeSampler struct {
	sample telemetry.Sample
}

func (f fakeSampler) Sample(context.Context) telemetry.Sample { return f.sample }

type fakeStore struct {
	mu      sync.Mutex
	samples []telemetry.Sample
	err     error
	got     chan struct{}
}

func newFakeStore() *fakeStore { return &fakeStore{got: make(chan struct{}, 16)} }

func (f *fakeStore) RecordTelemetry(_ context.Context, s telemetry.Sample) error {
	f.mu.Lock()
	f.samples = append(f.samples, s)
	f.mu.Unlock()
	select {
	case f.got <- struct{}{}:
	default:
	}
	return f.err
}

func (f *fakeStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.samples)
}

func carrierSample() telemetry.Sample {
	return telemetry.Sample{
		Interface: "eth0",
		At:        time.Unix(1_700_000_000, 0),
		Points:    []telemetry.Point{{Type: telemetry.LinkCarrier, Value: 1, Unit: telemetry.UnitBool}},
	}
}

func waitRecorded(t *testing.T, store *fakeStore, n int) {
	t.Helper()
	for range n {
		select {
		case <-store.got:
		case <-time.After(5 * time.Second):
			t.Fatalf("recorded %d samples, want %d", store.count(), n)
		}
	}
}

func TestEngine_SamplesAtStartAndEveryInterval(t *testing.T) {
	store := newFakeStore()
	eng := telemetry.New(fakeSampler{sample: carrierSample()}, store, 10*time.Millisecond, slog.Default())
	if err := eng.Start(t.Context()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := eng.Start(t.Context()); err != nil {
		t.Fatalf("second start: %v", err)
	}
	waitRecorded(t, store, 3)
	if err := eng.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if got := store.samples[0]; got.Interface != "eth0" || len(got.Points) != 1 {
		t.Errorf("stored %+v, want %+v", got, carrierSample())
	}
}

func TestEngine_FirstSampleIsImmediate(t *testing.T) {
	store := newFakeStore()
	eng := telemetry.New(fakeSampler{sample: carrierSample()}, store, time.Hour, slog.Default())
	if err := eng.Start(t.Context()); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = eng.Stop(context.Background()) })
	waitRecorded(t, store, 1)
}

func TestEngine_EmptySampleIsNotStored(t *testing.T) {
	store := newFakeStore()
	empty := telemetry.Sample{Interface: "eth0"}
	eng := telemetry.New(fakeSampler{sample: empty}, store, time.Millisecond, slog.Default())
	if err := eng.Start(t.Context()); err != nil {
		t.Fatalf("start: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if err := eng.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if n := store.count(); n != 0 {
		t.Errorf("stored %d empty samples, want 0", n)
	}
}

func TestEngine_StoreErrorKeepsSampling(t *testing.T) {
	store := newFakeStore()
	store.err = errors.New("disk full")
	eng := telemetry.New(fakeSampler{sample: carrierSample()}, store, time.Millisecond, slog.Default())
	if err := eng.Start(t.Context()); err != nil {
		t.Fatalf("start: %v", err)
	}
	waitRecorded(t, store, 2)
	if err := eng.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestEngine_StopBeforeStart(t *testing.T) {
	eng := telemetry.New(fakeSampler{}, newFakeStore(), time.Second, slog.Default())
	if err := eng.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
}
