package pipeline_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

type fakeObservations struct {
	rows []*observation.SNMPObservation
}

func (f *fakeObservations) List(
	_ context.Context,
	opts observation.ListOptions,
) ([]*observation.SNMPObservation, error) {
	out := make([]*observation.SNMPObservation, 0, len(f.rows))
	for _, row := range f.rows {
		if opts.Kind != "" && row.Kind != opts.Kind {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func iftableObs(target string, observed time.Time, rows []map[string]any) *observation.SNMPObservation {
	b, _ := json.Marshal(map[string]any{"Rows": rows})
	return &observation.SNMPObservation{
		ClientID: "default", TargetID: target, Kind: "if_table",
		ObservedAt: observed, PayloadJSON: string(b),
	}
}

// bgpObs and hostObs hardcode target="t-1" because the V1.0 tests
// only exercise single-target scenarios for these kinds; iftable
// tests cover the multi-target case. If multi-target bgp/storage
// tests arrive later, lift target back into a parameter.
func bgpObs(observed time.Time, peers []map[string]any) *observation.SNMPObservation {
	b, _ := json.Marshal(map[string]any{"Peers": peers})
	return &observation.SNMPObservation{
		ClientID: "default", TargetID: "t-1", Kind: "bgp4_mib",
		ObservedAt: observed, PayloadJSON: string(b),
	}
}

func hostObs(observed time.Time, storage []map[string]any) *observation.SNMPObservation {
	b, _ := json.Marshal(map[string]any{"Storage": storage})
	return &observation.SNMPObservation{
		ClientID: "default", TargetID: "t-1", Kind: "host_resources",
		ObservedAt: observed, PayloadJSON: string(b),
	}
}

func TestNewObservationPipeline_RejectsMissingDeps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  pipeline.ObservationConfig
	}{
		{"missing Observations", pipeline.ObservationConfig{Alerts: &fakeAlerts{}, Settings: newFakeSettings()}},
		{"missing Alerts", pipeline.ObservationConfig{Observations: &fakeObservations{}, Settings: newFakeSettings()}},
		{"missing Settings", pipeline.ObservationConfig{Observations: &fakeObservations{}, Alerts: &fakeAlerts{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := pipeline.NewObservationPipeline(tt.cfg); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestObservationScanOnce_InterfaceUpToDownEmitsAlert(t *testing.T) {
	t.Parallel()
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		iftableObs("t-1", at().Add(-time.Minute), []map[string]any{
			{"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 1, "IfOper": 1},
		}),
		iftableObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 1, "IfOper": 2},
		}),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := p.ScanOnce(context.Background()); err != nil {
		t.Fatalf("ScanOnce: %v", err)
	}
	if len(alerts.created) != 1 {
		t.Fatalf("alerts = %d, want 1 (only the up->down transition)", len(alerts.created))
	}
	a := alerts.created[0]
	if a.Severity != database.AlertSeverityWarning || a.Type != database.AlertTypeConnectivity {
		t.Errorf("type/severity = %q/%q", a.Type, a.Severity)
	}
}

func TestObservationScanOnce_AdminDownDoesNotAlert(t *testing.T) {
	t.Parallel()
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		iftableObs("t-1", at().Add(-time.Minute), []map[string]any{
			{"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 1, "IfOper": 1},
		}),
		// Operator pulled the port down intentionally — admin=2,
		// oper=2. Must not alert.
		iftableObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 2, "IfOper": 2},
		}),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 0 {
		t.Errorf("admin-down should not alert, got %d", len(alerts.created))
	}
}

func TestObservationScanOnce_FirstObservationDoesNotAlert(t *testing.T) {
	t.Parallel()
	// No prior state -> the first observation cannot be a transition.
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		iftableObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 1, "IfOper": 2},
		}),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 0 {
		t.Errorf("first-ever observation should not alert, got %d", len(alerts.created))
	}
}

func TestObservationScanOnce_BGPLeavingEstablishedFires(t *testing.T) {
	t.Parallel()
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		bgpObs(at().Add(-time.Minute), []map[string]any{
			{"RemoteAddr": "192.0.2.1", "State": 6, "RemoteAS": 65001},
		}),
		bgpObs(at(), []map[string]any{
			{"RemoteAddr": "192.0.2.1", "State": 3, "RemoteAS": 65001}, // Active
		}),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Fatalf("alerts = %d, want 1", len(alerts.created))
	}
	if alerts.created[0].Severity != database.AlertSeverityError {
		t.Errorf("BGP flap should be error severity, got %q", alerts.created[0].Severity)
	}
}

func TestObservationScanOnce_BGPStillEstablishedNoAlert(t *testing.T) {
	t.Parallel()
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		bgpObs(at().Add(-time.Minute), []map[string]any{
			{"RemoteAddr": "192.0.2.1", "State": 6, "RemoteAS": 65001},
		}),
		bgpObs(at(), []map[string]any{
			{"RemoteAddr": "192.0.2.1", "State": 6, "RemoteAS": 65001},
		}),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 0 {
		t.Errorf("steady BGP shouldn't alert, got %d", len(alerts.created))
	}
}

func TestObservationScanOnce_StorageCrosses85FiresWarning(t *testing.T) {
	t.Parallel()
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		hostObs(at().Add(-time.Minute), []map[string]any{
			{"Index": 1, "Description": "/", "SizeBytes": 1000, "UsedBytes": 800}, // 80%
		}),
		hostObs(at(), []map[string]any{
			{"Index": 1, "Description": "/", "SizeBytes": 1000, "UsedBytes": 870}, // 87%
		}),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 || alerts.created[0].Severity != database.AlertSeverityWarning {
		t.Errorf("expected one warning alert, got %+v", alerts.created)
	}
}

func TestObservationScanOnce_StorageCrosses95FiresCritical(t *testing.T) {
	t.Parallel()
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		hostObs(at().Add(-time.Minute), []map[string]any{
			{"Index": 1, "Description": "/", "SizeBytes": 1000, "UsedBytes": 800},
		}),
		hostObs(at(), []map[string]any{
			{"Index": 1, "Description": "/", "SizeBytes": 1000, "UsedBytes": 970}, // 97%
		}),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 || alerts.created[0].Severity != database.AlertSeverityCritical {
		t.Errorf("expected one critical alert, got %+v", alerts.created)
	}
}

func TestObservationScanOnce_StorageRemainingHighNoRepeatAlert(t *testing.T) {
	t.Parallel()
	// Two consecutive 87% observations -> first transitions from 0%
	// (initial unknown) to 87%, so warning fires once. Second is
	// 87% to 87%, no upward crossing -> no alert.
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		hostObs(at().Add(-time.Minute), []map[string]any{
			{"Index": 1, "Description": "/", "SizeBytes": 1000, "UsedBytes": 870},
		}),
		hostObs(at(), []map[string]any{
			{"Index": 1, "Description": "/", "SizeBytes": 1000, "UsedBytes": 880},
		}),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Errorf("only first crossing should fire, got %d", len(alerts.created))
	}
}

func TestObservationScanOnce_DifferentTargetsTrackedIndependently(t *testing.T) {
	t.Parallel()
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		iftableObs("t-1", at().Add(-time.Minute), []map[string]any{
			{"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 1, "IfOper": 1},
		}),
		iftableObs("t-2", at().Add(-time.Minute), []map[string]any{
			{"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 1, "IfOper": 1},
		}),
		iftableObs("t-1", at(), []map[string]any{
			{"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 1, "IfOper": 2},
		}),
		iftableObs("t-2", at(), []map[string]any{
			{"IfIndex": 1, "IfName": "Gi0/0", "IfAdmin": 1, "IfOper": 2},
		}),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 2 {
		t.Errorf("two targets transitioning should fire 2 alerts, got %d", len(alerts.created))
	}
}

func TestObservationScanOnce_PersistsMaxObservedAtAcrossKinds(t *testing.T) {
	t.Parallel()
	newer := at()
	older := at().Add(-2 * time.Hour)
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		iftableObs("t-1", older, nil),
		bgpObs(newer, nil),
	}}
	s := newFakeSettings()
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: &fakeAlerts{}, Settings: s,
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())

	hw, _ := s.GetWithDefault(context.Background(), "alerts.observation.high_water", "")
	parsed, perr := time.Parse(time.RFC3339Nano, hw)
	if perr != nil {
		t.Fatalf("parse: %v", perr)
	}
	if !parsed.Equal(newer) {
		t.Errorf("high-water = %v, want %v", parsed, newer)
	}
}

func TestObservationPipeline_EngineName(t *testing.T) {
	t.Parallel()
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: &fakeObservations{},
		Alerts:       &fakeAlerts{},
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
	})
	if p.Name() != pipeline.ObservationPipelineName {
		t.Errorf("Name() = %q, want %q", p.Name(), pipeline.ObservationPipelineName)
	}
}

func TestObservationStartStop_Idempotent(t *testing.T) {
	t.Parallel()
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: &fakeObservations{},
		Alerts:       &fakeAlerts{},
		Settings:     newFakeSettings(),
		Logger:       silentLogger(),
		Now:          at,
		Interval:     500 * time.Millisecond,
	})
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
