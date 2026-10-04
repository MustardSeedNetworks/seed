package pipeline_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	alertmodel "github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

func indicatorEvent(t *testing.T, host, listed string, observed time.Time) *listener.EventRecord {
	t.Helper()
	payload, err := json.Marshal(listener.FlowIndicatorHit{
		Host: host, Listed: listed, Indicator: listed + "/32", Exporter: "10.0.0.1",
		Flows: 2, Bytes: 5000, Packets: 18,
		FirstSeen: observed.Add(-time.Minute), LastSeen: observed,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &listener.EventRecord{
		ClientID: "default", Kind: listener.FlowIndicatorKind, SourceAddr: host,
		ObservedAt: observed, PayloadJSON: string(payload),
	}
}

// The indicator rule is pinned: an operator rule set that replaces the
// defaults, and matches nothing here, must not silence it. A second host
// gets its own alert; a second listed address from the first host inside
// the suppression window does not.
func TestScanOnce_FlowIndicatorAlertsWhateverRulesAreActive(t *testing.T) {
	t.Parallel()
	for name, reader := range map[string]*fakeAlertRulesReader{
		"defaults": {},
		"operator rules": {rows: []*alertmodel.Rule{
			dbRule("info-rule", "syslog-udp", "informational", "", true),
		}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			events := &fakeEvents{rows: []*listener.EventRecord{
				indicatorEvent(t, "10.0.0.5", "198.51.100.7", at()),
				indicatorEvent(t, "10.0.0.5", "203.0.113.9", at().Add(time.Second)),
				indicatorEvent(t, "10.0.0.6", "198.51.100.7", at().Add(2*time.Second)),
			}}
			alerts := &fakeAlerts{}
			p, err := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
				Events: events, Alerts: alerts, Settings: newFakeSettings(),
				Logger: silentLogger(), Now: at, AlertRules: reader,
			})
			if err != nil {
				t.Fatal(err)
			}
			if scanErr := p.ScanOnce(context.Background()); scanErr != nil {
				t.Fatal(scanErr)
			}
			if len(alerts.created) != 2 {
				t.Fatalf("alerts = %d, want 2 (one per host)", len(alerts.created))
			}
			a := alerts.created[0]
			want := alertmodel.Alert{
				Type:     alertmodel.TypeSecurity,
				Severity: alertmodel.SeverityCritical,
				Rule:     alertmodel.RuleFlowIndicator,
				Source:   "10.0.0.5",
				Title:    "10.0.0.5 exchanged traffic with listed address 198.51.100.7",
				Message: "198.51.100.7 is on the threat indicator list (entry 198.51.100.7/32). " +
					"Exporter 10.0.0.1 reported 2 flows, 5000 bytes, between 2026-05-31T11:59:00Z and 2026-05-31T12:00:00Z.",
			}
			if a.Type != want.Type || a.Severity != want.Severity || a.Rule != want.Rule ||
				a.Source != want.Source || a.Title != want.Title || a.Message != want.Message {
				t.Errorf("alert =\n%+v\nwant\n%+v", *a, want)
			}
			if a.Metadata != events.rows[0].PayloadJSON {
				t.Errorf("metadata = %s, want the event payload", a.Metadata)
			}
			if alerts.created[1].Source != "10.0.0.6" {
				t.Errorf("second alert source = %q, want 10.0.0.6", alerts.created[1].Source)
			}
		})
	}
}

func TestScanOnce_FlowIndicatorWithBadPayloadRaisesNothing(t *testing.T) {
	t.Parallel()
	events := &fakeEvents{rows: []*listener.EventRecord{{
		Kind: listener.FlowIndicatorKind, SourceAddr: "10.0.0.5", ObservedAt: at(), PayloadJSON: `{"host":`,
	}}}
	alerts := &fakeAlerts{}
	p, err := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(), Logger: silentLogger(), Now: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanErr := p.ScanOnce(context.Background()); scanErr != nil {
		t.Fatal(scanErr)
	}
	if len(alerts.created) != 0 {
		t.Errorf("alerts = %d from an undecodable payload, want 0", len(alerts.created))
	}
}
