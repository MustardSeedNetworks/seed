package pipeline_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

func TestListenerPipeline_Status_BeforeFirstScan(t *testing.T) {
	t.Parallel()
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: &fakeEvents{}, Alerts: &fakeAlerts{}, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	r, ok := any(p).(engine.Reporter)
	if !ok {
		t.Fatal("ListenerPipeline must implement engine.Reporter")
	}
	s := r.Status()
	if s.State != engine.StateOK {
		t.Errorf("State before any scan = %q, want ok", s.State)
	}
	if !s.LastTickAt.IsZero() {
		t.Errorf("LastTickAt = %v, want zero before first scan", s.LastTickAt)
	}
}

func TestListenerPipeline_Status_AfterScanRecordsTick(t *testing.T) {
	t.Parallel()
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: &fakeEvents{}, Alerts: &fakeAlerts{}, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	s := any(p).(engine.Reporter).Status()
	if s.LastTickAt.IsZero() {
		t.Error("LastTickAt should be non-zero after ScanOnce")
	}
	if s.LastError != "" {
		t.Errorf("LastError = %q, want empty after clean scan", s.LastError)
	}
}

func TestListenerPipeline_Status_AfterScanErrorRecordsError(t *testing.T) {
	t.Parallel()
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events:   &fakeEvents{listErr: errors.New("db down")},
		Alerts:   &fakeAlerts{},
		Settings: newFakeSettings(),
		Logger:   silentLogger(),
		Now:      at,
	})
	_ = p.ScanOnce(context.Background())
	s := any(p).(engine.Reporter).Status()
	if s.LastError == "" {
		t.Error("LastError should be populated after failed scan")
	}
}

func TestListenerPipeline_Status_AfterStop(t *testing.T) {
	t.Parallel()
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: &fakeEvents{}, Alerts: &fakeAlerts{}, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at, Interval: 500 * time.Millisecond,
	})
	if startErr := p.Start(context.Background()); startErr != nil {
		t.Fatalf("Start: %v", startErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if stopErr := p.Stop(ctx); stopErr != nil {
		t.Fatalf("Stop: %v", stopErr)
	}
	s := any(p).(engine.Reporter).Status()
	if s.State != engine.StateStopped {
		t.Errorf("State after Stop = %q, want stopped", s.State)
	}
}

// fakeObs lets us drive ObservationPipeline.ScanOnce.
type fakeObs struct {
	rows    []*observation.SNMPObservation
	listErr error
}

func (f *fakeObs) List(_ context.Context, _ observation.ListOptions) ([]*observation.SNMPObservation, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*observation.SNMPObservation, len(f.rows))
	copy(out, f.rows)
	return out, nil
}

func TestObservationPipeline_Status_LifecycleTransitions(t *testing.T) {
	t.Parallel()
	p, err := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: &fakeObs{},
		Alerts:       &fakeAlerts{},
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
	})
	if err != nil {
		t.Fatalf("NewObservationPipeline: %v", err)
	}
	rep, ok := any(p).(engine.Reporter)
	if !ok {
		t.Fatal("ObservationPipeline must implement engine.Reporter")
	}

	// Before scan
	if state := rep.Status().State; state != engine.StateOK {
		t.Errorf("pre-scan state = %q, want ok", state)
	}
	// After clean scan
	_ = p.ScanOnce(context.Background())
	if rep.Status().LastTickAt.IsZero() {
		t.Error("LastTickAt zero after ScanOnce")
	}
}
