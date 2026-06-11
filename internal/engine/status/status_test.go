package status_test

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/engine/status"
)

// ── test doubles ─────────────────────────────────────────────────────────────

// fakeRegistry implements status.Registry for testing.
type fakeRegistry struct {
	available bool
	engines   []engine.Engine
}

func (f *fakeRegistry) Available() bool          { return f.available }
func (f *fakeRegistry) Engines() []engine.Engine { return f.engines }

// minimalEngine implements engine.Engine but NOT engine.Reporter.
type minimalEngine struct{ name string }

func (m *minimalEngine) Name() string                { return m.name }
func (*minimalEngine) Start(_ context.Context) error { return nil }
func (*minimalEngine) Stop(_ context.Context) error  { return nil }

// reportingEngine implements both engine.Engine and engine.Reporter.
type reportingEngine struct {
	name   string
	status engine.Status
}

func (r *reportingEngine) Name() string                { return r.name }
func (*reportingEngine) Start(_ context.Context) error { return nil }
func (*reportingEngine) Stop(_ context.Context) error  { return nil }
func (r *reportingEngine) Status() engine.Status       { return r.status }

// ── tests ─────────────────────────────────────────────────────────────────────

func TestList_NilRegistry_ReturnsNil(t *testing.T) {
	svc := status.NewService(nil)
	got := svc.List()
	if got != nil {
		t.Errorf("nil registry: want nil, got %v", got)
	}
}

func TestList_UnavailableRegistry_ReturnsNil(t *testing.T) {
	reg := &fakeRegistry{available: false, engines: []engine.Engine{&minimalEngine{name: "x"}}}
	svc := status.NewService(reg)
	got := svc.List()
	if got != nil {
		t.Errorf("unavailable registry: want nil, got %v", got)
	}
}

func TestList_MinimalEngine_DefaultsToStateOK(t *testing.T) {
	reg := &fakeRegistry{
		available: true,
		engines:   []engine.Engine{&minimalEngine{name: "plain"}},
	}
	svc := status.NewService(reg)
	got := svc.List()
	if len(got) != 1 {
		t.Fatalf("want 1 entry, got %d", len(got))
	}
	e := got[0]
	if e.Name != "plain" {
		t.Errorf("Name = %q, want %q", e.Name, "plain")
	}
	if e.State != engine.StateOK {
		t.Errorf("State = %q, want %q (no Reporter → default StatusOK)", e.State, engine.StateOK)
	}
	if e.LastError != "" {
		t.Errorf("LastError = %q, want empty", e.LastError)
	}
	if e.Inflight != 0 {
		t.Errorf("Inflight = %d, want 0", e.Inflight)
	}
}

func TestList_ReportingEngine_SurfacesStatus(t *testing.T) {
	tick := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	reg := &fakeRegistry{
		available: true,
		engines: []engine.Engine{
			&reportingEngine{
				name: "rich",
				status: engine.Status{
					State:      engine.StateDegraded,
					LastTickAt: tick,
					LastError:  "scan timeout",
					Inflight:   3,
				},
			},
		},
	}
	svc := status.NewService(reg)
	got := svc.List()
	if len(got) != 1 {
		t.Fatalf("want 1 entry, got %d", len(got))
	}
	e := got[0]
	if e.Name != "rich" {
		t.Errorf("Name = %q, want %q", e.Name, "rich")
	}
	if e.State != engine.StateDegraded {
		t.Errorf("State = %q, want %q", e.State, engine.StateDegraded)
	}
	if e.LastTickAt != tick {
		t.Errorf("LastTickAt = %v, want %v", e.LastTickAt, tick)
	}
	if e.LastError != "scan timeout" {
		t.Errorf("LastError = %q, want %q", e.LastError, "scan timeout")
	}
	if e.Inflight != 3 {
		t.Errorf("Inflight = %d, want 3", e.Inflight)
	}
}

func TestList_ReporterEmptyState_NormalisedToStateOK(t *testing.T) {
	reg := &fakeRegistry{
		available: true,
		engines: []engine.Engine{
			&reportingEngine{
				name:   "blank-state",
				status: engine.Status{}, // State left empty
			},
		},
	}
	svc := status.NewService(reg)
	got := svc.List()
	if len(got) != 1 {
		t.Fatalf("want 1 entry, got %d", len(got))
	}
	if got[0].State != engine.StateOK {
		t.Errorf("empty Reporter.State should normalise to %q, got %q", engine.StateOK, got[0].State)
	}
}
