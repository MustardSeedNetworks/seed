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

// The rttMonLatestRttOperSense values the tests use.
const (
	senseOK            = 1
	senseOverThreshold = 3
	senseTimeout       = 4
)

// slaOp is one operation as the ip_sla collector publishes it, with its
// latest run; run 0 is an operation that has never run.
type slaOp struct {
	index, sense int
	rtt, run     uint32
}

func ipslaObs(observed time.Time, ops ...slaOp) *observation.SNMPObservation {
	payload := make([]map[string]any, 0, len(ops))
	for _, op := range ops {
		row := map[string]any{
			"Index": op.index, "Tag": "wan-gw", "RttType": 1, "Target": "192.0.2.1",
			"ThresholdMs": 100, "FrequencyS": 60, "Status": 1, "Latest": nil,
		}
		if op.run != 0 {
			row["Latest"] = map[string]any{"Sense": op.sense, "RTTMs": op.rtt, "RunUptime": op.run, "Jitter": nil}
		}
		payload = append(payload, row)
	}
	b, _ := json.Marshal(map[string]any{"Operations": payload})
	return &observation.SNMPObservation{
		ClientID: "default", TargetID: "rtr-1", Kind: "ip_sla",
		ObservedAt: observed, PayloadJSON: string(b),
	}
}

func TestObservationScanOnce_IPSLA(t *testing.T) {
	t.Parallel()
	earlier := at().Add(-time.Minute)
	tests := []struct {
		name     string
		rows     []*observation.SNMPObservation
		rule     string
		severity string
		evidence alertmodel.IPSLAEvidence
	}{
		{
			name: "completed runs are quiet",
			rows: []*observation.SNMPObservation{
				ipslaObs(earlier, slaOp{10, senseOK, 7, 100}),
				ipslaObs(at(), slaOp{10, senseOK, 9, 200}),
			},
		},
		{
			name: "first poll is the baseline, even of a failing operation",
			rows: []*observation.SNMPObservation{ipslaObs(at(), slaOp{10, senseTimeout, 0, 100})},
		},
		{
			name: "a run failing after a completed one",
			rows: []*observation.SNMPObservation{
				ipslaObs(earlier, slaOp{10, senseOK, 7, 100}),
				ipslaObs(at(), slaOp{10, senseTimeout, 0, 200}),
			},
			rule:     alertmodel.RuleIPSLAFailed,
			severity: alertmodel.SeverityError,
			evidence: alertmodel.IPSLAEvidence{
				Index: 10, Tag: "wan-gw", Target: "192.0.2.1", Sense: senseTimeout, ThresholdMs: 100,
			},
		},
		{
			name: "a run failing after one over threshold",
			rows: []*observation.SNMPObservation{
				ipslaObs(earlier, slaOp{10, senseOverThreshold, 140, 100}),
				ipslaObs(at(), slaOp{10, senseTimeout, 0, 200}),
			},
			rule:     alertmodel.RuleIPSLAFailed,
			severity: alertmodel.SeverityError,
			evidence: alertmodel.IPSLAEvidence{
				Index: 10, Tag: "wan-gw", Target: "192.0.2.1", Sense: senseTimeout, ThresholdMs: 100,
			},
		},
		{
			name: "a completed run crossing the threshold",
			rows: []*observation.SNMPObservation{
				ipslaObs(earlier, slaOp{10, senseOK, 7, 100}),
				ipslaObs(at(), slaOp{10, senseOverThreshold, 140, 200}),
			},
			rule:     alertmodel.RuleIPSLAOverThreshold,
			severity: alertmodel.SeverityWarning,
			evidence: alertmodel.IPSLAEvidence{
				Index: 10, Tag: "wan-gw", Target: "192.0.2.1", Sense: senseOverThreshold,
				RTTMs: 140, ThresholdMs: 100,
			},
		},
		{
			name: "a poll that sees the same run again is quiet",
			rows: []*observation.SNMPObservation{
				ipslaObs(earlier, slaOp{10, senseOK, 7, 100}),
				ipslaObs(at(), slaOp{10, senseTimeout, 0, 100}),
			},
		},
		{
			name: "an operation that stays down is quiet",
			rows: []*observation.SNMPObservation{
				ipslaObs(earlier, slaOp{10, senseTimeout, 0, 100}),
				ipslaObs(at(), slaOp{10, senseTimeout, 0, 200}),
			},
		},
		{
			name: "recovering over threshold is quiet",
			rows: []*observation.SNMPObservation{
				ipslaObs(earlier, slaOp{10, senseTimeout, 0, 100}),
				ipslaObs(at(), slaOp{10, senseOverThreshold, 140, 200}),
			},
		},
		{
			name: "an operation added since the previous poll is its own baseline",
			rows: []*observation.SNMPObservation{
				ipslaObs(earlier, slaOp{10, senseOK, 7, 100}, slaOp{20, 0, 0, 0}),
				ipslaObs(at(), slaOp{10, senseOK, 7, 100}, slaOp{20, senseTimeout, 0, 200}),
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
				b, _ := json.Marshal(tt.evidence)
				want = []string{tt.rule + " " + tt.severity + " rtr-1 " + string(b)}
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
