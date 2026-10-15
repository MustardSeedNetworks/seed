package pipeline_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	alertmodel "github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

const (
	rootA = "aa:aa:aa:00:00:01"
	rootB = "bb:bb:bb:00:00:02"
)

// stpPort is one dot1dStpPortTable row as the stp collector publishes it.
type stpPort struct {
	Port, IfIndex, State int
}

func stpObs(observed time.Time, root string, topChanges uint32, ports ...stpPort) *observation.SNMPObservation {
	b, _ := json.Marshal(map[string]any{
		"Root":            map[string]any{"Priority": 32768, "Address": root},
		"RootPort":        1,
		"TopologyChanges": topChanges,
		"Ports":           ports,
	})
	return &observation.SNMPObservation{
		ClientID: "default", TargetID: "sw-1", Kind: "stp",
		ObservedAt: observed, PayloadJSON: string(b),
	}
}

func TestObservationScanOnce_STP(t *testing.T) {
	t.Parallel()
	earlier := at().Add(-time.Minute)
	tests := []struct {
		name     string
		rows     []*observation.SNMPObservation
		rule     string
		severity string
		evidence any
	}{
		{
			name: "steady tree is quiet",
			rows: []*observation.SNMPObservation{
				stpObs(earlier, rootA, 4, stpPort{1, 10001, 5}),
				stpObs(at(), rootA, 4, stpPort{1, 10001, 5}),
			},
		},
		{
			name: "first poll is not a change",
			rows: []*observation.SNMPObservation{stpObs(at(), rootA, 9)},
		},
		{
			name: "counter rise with a local port transition",
			rows: []*observation.SNMPObservation{
				stpObs(earlier, rootA, 4, stpPort{1, 10001, 5}, stpPort{2, 10002, 5}),
				stpObs(at(), rootA, 6, stpPort{1, 10001, 5}, stpPort{2, 10002, 2}),
			},
			rule:     alertmodel.RuleSTPTopologyChange,
			severity: alertmodel.SeverityWarning,
			evidence: alertmodel.STPTopologyChangeEvidence{
				Changes: 2, TopologyChanges: 6,
				Ports: []alertmodel.STPPortTransition{{Port: 2, IfIndex: 10002, From: 5, To: 2}},
			},
		},
		{
			name: "counter rise learned from elsewhere in the tree",
			rows: []*observation.SNMPObservation{
				stpObs(earlier, rootA, 4, stpPort{1, 10001, 5}),
				stpObs(at(), rootA, 5, stpPort{1, 10001, 5}),
			},
			rule:     alertmodel.RuleSTPTopologyChange,
			severity: alertmodel.SeverityWarning,
			evidence: alertmodel.STPTopologyChangeEvidence{
				Changes: 1, TopologyChanges: 5, Ports: []alertmodel.STPPortTransition{},
			},
		},
		{
			name: "root change is reported as a root change only",
			rows: []*observation.SNMPObservation{
				stpObs(earlier, rootA, 4, stpPort{1, 10001, 5}),
				stpObs(at(), rootB, 7, stpPort{1, 10001, 2}),
			},
			rule:     alertmodel.RuleSTPRootChange,
			severity: alertmodel.SeverityError,
			evidence: alertmodel.STPRootChangeEvidence{
				PreviousRoot: "32768/" + rootA, Root: "32768/" + rootB, RootPort: 1,
			},
		},
		{
			name: "counter falling back after an agent restart is quiet",
			rows: []*observation.SNMPObservation{
				stpObs(earlier, rootA, 40),
				stpObs(at(), rootA, 1),
			},
		},
		{
			name: "a switch running no spanning tree is skipped",
			rows: []*observation.SNMPObservation{
				stpObs(earlier, "", 0),
				stpObs(at(), "", 3),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			alerts := &fakeAlerts{}
			p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
				Observations: &fakeObservations{rows: tt.rows}, Alerts: alerts,
				Settings: newFakeSettings(), Logger: silentLogger(), Now: at,
			})
			if err := p.ScanOnce(context.Background()); err != nil {
				t.Fatalf("ScanOnce: %v", err)
			}
			var want []string
			if tt.rule != "" {
				want = []string{stpAlertLine(tt.rule, tt.severity, tt.evidence)}
			}
			var got []string
			for _, a := range alerts.created {
				got = append(got, a.Rule+" "+a.Severity+" "+a.Source+" "+a.Metadata)
			}
			if !slices.Equal(got, want) {
				t.Errorf("alerts =\n%q\nwant\n%q", got, want)
			}
		})
	}
}

// stpAlertLine is how one alert on sw-1 reads in TestObservationScanOnce_STP.
// Metadata is [json.Marshal] of the evidence struct, so equal evidence is
// byte-identical.
func stpAlertLine(rule, severity string, evidence any) string {
	b, _ := json.Marshal(evidence)
	return rule + " " + severity + " sw-1 " + string(b)
}

// A switch whose tree keeps changing raises one topology-change alert per
// suppression window, not one per poll.
func TestObservationScanOnce_STPRepeatedChangesSuppressed(t *testing.T) {
	t.Parallel()
	alerts := &fakeAlerts{}
	o := &fakeObservations{rows: []*observation.SNMPObservation{
		stpObs(at().Add(-2*time.Minute), rootA, 1),
		stpObs(at().Add(-time.Minute), rootA, 2),
		stpObs(at(), rootA, 3),
	}}
	p, _ := pipeline.NewObservationPipeline(pipeline.ObservationConfig{
		Observations: o, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Errorf("alerts = %d, want 1", len(alerts.created))
	}
}
