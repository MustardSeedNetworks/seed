package probeanomaly_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/probe"
	probeanomaly "github.com/MustardSeedNetworks/seed/internal/probe/anomaly"
)

func TestCatalogBuildsAndCoversMappedDefs(t *testing.T) {
	t.Parallel()
	cat, err := probeanomaly.Catalog()
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	// Every def the mapper can emit must resolve, or the engine would reject the
	// detection at runtime (no orphan defKeys).
	for _, id := range []string{
		probeanomaly.DefUnreachable,
		probeanomaly.DefHighLatency,
		probeanomaly.DefCertExpiry,
		probeanomaly.DefThresholdBreach,
	} {
		if _, ok := cat.Lookup(id); !ok {
			t.Errorf("catalog missing def %q", id)
		}
	}
}

func TestDetectionsMapsBreaches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		event     probe.ResultEvent
		wantDef   string
		wantSev   anomaly.Severity
		wantSubID string
	}{
		{
			name: "latency breach -> high-latency",
			event: probe.ResultEvent{
				Result: probe.Result{ProbeID: "p1", Kind: "http"},
				Breaches: []probe.Breach{
					{
						ProbeID:   "p1",
						Severity:  "critical",
						Field:     "latency_ms",
						Threshold: 100.0,
						Actual:    250.0,
					},
				},
			},
			wantDef: probeanomaly.DefHighLatency, wantSev: anomaly.SeverityCritical, wantSubID: "p1",
		},
		{
			name: "success breach -> unreachable",
			event: probe.ResultEvent{
				Result: probe.Result{ProbeID: "p2", Kind: "ping"},
				Breaches: []probe.Breach{
					{
						ProbeID:   "p2",
						Severity:  "critical",
						Field:     "success",
						Threshold: true,
						Actual:    false,
					},
				},
			},
			wantDef: probeanomaly.DefUnreachable, wantSev: anomaly.SeverityCritical, wantSubID: "p2",
		},
		{
			name: "cert days-remaining breach -> cert-expiry",
			event: probe.ResultEvent{
				Result: probe.Result{ProbeID: "p3", Kind: "tls"},
				Breaches: []probe.Breach{
					{
						ProbeID:   "p3",
						Severity:  "critical",
						Field:     "cert_days_remaining",
						Threshold: 7,
						Actual:    3,
					},
				},
			},
			wantDef: probeanomaly.DefCertExpiry, wantSev: anomaly.SeverityCritical, wantSubID: "p3",
		},
		{
			name: "still-unmapped field -> generic threshold breach",
			event: probe.ResultEvent{
				Result: probe.Result{ProbeID: "p4", Kind: "bgp"},
				Breaches: []probe.Breach{
					{
						ProbeID:   "p4",
						Severity:  "warning",
						Field:     "bgp_state",
						Threshold: "established",
						Actual:    "idle",
					},
				},
			},
			wantDef: probeanomaly.DefThresholdBreach, wantSev: anomaly.SeverityWarning, wantSubID: "p4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := probeanomaly.Detections(tc.event)
			if len(got) != 1 {
				t.Fatalf("want 1 detection, got %d", len(got))
			}
			d := got[0]
			if d.DefKey != tc.wantDef {
				t.Errorf("defKey = %q, want %q", d.DefKey, tc.wantDef)
			}
			if d.Severity != tc.wantSev {
				t.Errorf("severity = %q, want %q", d.Severity, tc.wantSev)
			}
			if d.Subject.Kind != anomaly.SubjectProbe || d.Subject.ID != tc.wantSubID {
				t.Errorf("subject = %+v, want {probe %q}", d.Subject, tc.wantSubID)
			}
			if d.Evidence["field"] == "" || d.Evidence["kind"] == "" {
				t.Errorf("evidence missing field/kind: %+v", d.Evidence)
			}
		})
	}
}

func TestDetectionsCleanRunYieldsNothing(t *testing.T) {
	t.Parallel()
	got := probeanomaly.Detections(
		probe.ResultEvent{Result: probe.Result{ProbeID: "p1", Success: true}},
	)
	if got != nil {
		t.Fatalf("clean run must yield no detections, got %+v", got)
	}
}

func TestDetectionsEvidenceIncludesError(t *testing.T) {
	t.Parallel()
	ev := probe.ResultEvent{
		Result:   probe.Result{ProbeID: "p1", Kind: "dns", Error: "no such host"},
		Breaches: []probe.Breach{{ProbeID: "p1", Severity: "critical", Field: "success"}},
	}
	got := probeanomaly.Detections(ev)
	if len(got) != 1 || got[0].Evidence["error"] != "no such host" {
		t.Fatalf("want error captured in evidence, got %+v", got)
	}
}
