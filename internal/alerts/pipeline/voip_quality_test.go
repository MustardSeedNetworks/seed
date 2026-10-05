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

func voipEvent(t *testing.T, src string, observed time.Time) *listener.EventRecord {
	t.Helper()
	payload, err := json.Marshal(listener.VoIPQuality{
		Interface: "eth0", Src: src + ":40000", Dst: "192.0.2.20:30000", SSRC: 0x1234abcd,
		Codec: "G729", Start: observed.Add(-time.Minute), End: observed,
		LossPct: 9.997, JitterMs: 14.14, MaxJitterMs: 14.65, RFactor: 51.72, MOS: 2.665,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &listener.EventRecord{
		ClientID: "default", Kind: listener.VoIPQualityKind, SourceAddr: src,
		Severity: "warning", ObservedAt: observed, PayloadJSON: string(payload),
	}
}

// The VoIP rule is pinned like the indicator rule: the operator enabled the
// analyser, so an operator rule set must not silence it. Suppression holds
// one alert per sender.
func TestScanOnce_VoIPQualityAlertsWhateverRulesAreActive(t *testing.T) {
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
				voipEvent(t, "192.0.2.10", at()),
				voipEvent(t, "192.0.2.10", at().Add(time.Second)),
				voipEvent(t, "192.0.2.11", at().Add(2*time.Second)),
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
				t.Fatalf("alerts = %d, want 2 (one per sender)", len(alerts.created))
			}
			a := alerts.created[0]
			want := alertmodel.Alert{
				Type:     alertmodel.TypePerformance,
				Severity: alertmodel.SeverityWarning,
				Rule:     alertmodel.RuleVoIPQuality,
				Source:   "192.0.2.10",
				Title:    "Poor call quality from 192.0.2.10:40000 to 192.0.2.20:30000: MOS 2.67",
				Message: "G729 stream 1234abcd on eth0 scored MOS 2.67 (R 51.7) between " +
					"2026-05-31T11:59:00Z and 2026-05-31T12:00:00Z: 10.00% loss, 14.1 ms mean jitter, 14.7 ms peak.",
			}
			if a.Type != want.Type || a.Severity != want.Severity || a.Rule != want.Rule ||
				a.Source != want.Source || a.Title != want.Title || a.Message != want.Message {
				t.Errorf("alert =\n%+v\nwant\n%+v", *a, want)
			}
			if alerts.created[1].Source != "192.0.2.11" {
				t.Errorf("second alert source = %q, want 192.0.2.11", alerts.created[1].Source)
			}
		})
	}
}
