package probeanomaly

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// fakeStore records anomaly.Store calls for assertion.
type fakeStore struct {
	mu       sync.Mutex
	upserts  int
	rows     []anomaly.Record
	resolved []string
}

func (f *fakeStore) Upsert(_ context.Context, recs []anomaly.Record) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.upserts++
	f.rows = append(f.rows, recs...)
	return nil
}

func (f *fakeStore) MarkResolved(_ context.Context, ids []string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resolved = append(f.resolved, ids...)
	return nil
}

func (f *fakeStore) LoadActive(_ context.Context) ([]anomaly.Record, error) { return nil, nil }

func (f *fakeStore) snapshot() (int, []anomaly.Record, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := append([]anomaly.Record(nil), f.rows...)
	res := append([]string(nil), f.resolved...)
	return f.upserts, rows, res
}

func latencyEvent(probeID string) probe.ResultEvent {
	return probe.ResultEvent{
		Result: probe.Result{ProbeID: probeID, Kind: "http"},
		Breaches: []probe.Breach{
			{
				ProbeID:   probeID,
				Severity:  "warning",
				Field:     "latency_ms",
				Threshold: 100.0,
				Actual:    200.0,
			},
		},
	}
}

func TestObservePersistsThroughCoordinator(t *testing.T) {
	t.Parallel()
	fs := &fakeStore{}
	p, err := New(nil, fs)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.observe(context.Background(), latencyEvent("p1"), time.Unix(1000, 0).UTC())

	upserts, rows, _ := fs.snapshot()
	if upserts != 1 || len(rows) != 1 {
		t.Fatalf(
			"want one write-through upsert of one row, got %d upserts / %d rows",
			upserts,
			len(rows),
		)
	}
	got := rows[0]
	if got.Source != anomaly.SourceProbe {
		t.Errorf("source = %q, want probe", got.Source)
	}
	if got.Anomaly.DefKey != DefHighLatency || got.Anomaly.Subject.ID != "p1" {
		t.Errorf(
			"record = def %q subject %q, want high-latency/p1",
			got.Anomaly.DefKey,
			got.Anomaly.Subject.ID,
		)
	}
	wantID := anomaly.RecordID(
		DefHighLatency,
		anomaly.SubjectRef{Kind: anomaly.SubjectProbe, ID: "p1"},
	)
	if got.ID != wantID {
		t.Errorf("record id = %q, want %q", got.ID, wantID)
	}
}

func TestFlushAndPruneResolvesAfterSilence(t *testing.T) {
	t.Parallel()
	fs := &fakeStore{}
	p, err := New(nil, fs, WithResolveWindow(10*time.Minute))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t0 := time.Unix(2000, 0).UTC()
	p.observe(context.Background(), latencyEvent("p1"), t0)

	// Within the window: not yet resolved.
	p.flushAndPrune(context.Background(), t0.Add(5*time.Minute))
	if _, _, res := fs.snapshot(); len(res) != 0 {
		t.Fatalf("must not resolve within the window, got %v", res)
	}

	// Past the window with no re-breach: the probe is considered recovered.
	p.flushAndPrune(context.Background(), t0.Add(11*time.Minute))
	_, _, res := fs.snapshot()
	wantID := anomaly.RecordID(
		DefHighLatency,
		anomaly.SubjectRef{Kind: anomaly.SubjectProbe, ID: "p1"},
	)
	if len(res) != 1 || res[0] != wantID {
		t.Fatalf("want %q resolved after silence, got %v", wantID, res)
	}
}

func TestStartStopIsIdempotentAndDrains(t *testing.T) {
	t.Parallel()
	fs := &fakeStore{}
	events := make(chan probe.ResultEvent, 4)
	p, err := New(events, fs, WithFlushInterval(5*time.Millisecond), WithResolveWindow(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	if err = p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Second Start is a no-op.
	if err = p.Start(ctx); err != nil {
		t.Fatalf("second Start: %v", err)
	}

	events <- latencyEvent("p1")
	// Wait for the consume goroutine to persist it.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if upserts, _, _ := fs.snapshot(); upserts > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("event was not consumed within 2s")
		}
		time.Sleep(2 * time.Millisecond)
	}

	if err = p.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	// Second Stop is safe.
	if err = p.Stop(ctx); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
}
