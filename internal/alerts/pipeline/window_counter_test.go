package pipeline_test

// Behavioral tests for the time-windowed counter exercise the
// counter via the Rule struct (the only public seam). A rule with
// Threshold=N and Window=W needs N matching events inside W to
// fire; old hits outside the window get pruned.

import (
	"context"
	"testing"
	"time"

	alertmodel "github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

func windowedRule(window time.Duration, threshold int) *alertmodel.Rule {
	return &alertmodel.Rule{
		ID: 1, Name: "win", Enabled: true,
		MatchKind:      "syslog-udp",
		MatchSeverity:  "error",
		AlertType:      alertmodel.TypeSystem,
		AlertSeverity:  alertmodel.SeverityError,
		AlertTitle:     "Windowed fire",
		WindowSeconds:  int(window.Seconds()),
		ThresholdCount: threshold,
	}
}

func runScanWithEvents(
	t *testing.T, rule *alertmodel.Rule, events []*listener.EventRecord,
) []*alertmodel.Alert {
	t.Helper()
	// Each event triggers one scan with a fresh fakeEvents slice so
	// we control timing via the event's ObservedAt + the pipeline's
	// Now func.
	fakeEv := &fakeEvents{}
	alerts := &fakeAlerts{}
	rules := pipeline.CompileRulesFromDB([]*alertmodel.Rule{rule})
	currentNow := events[0].ObservedAt
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events:   fakeEv,
		Alerts:   alerts,
		Settings: newFakeSettings(),
		Logger:   silentLogger(),
		Now:      func() time.Time { return currentNow },
		Rules:    rules,
	})
	for _, evt := range events {
		currentNow = evt.ObservedAt
		fakeEv.rows = []*listener.EventRecord{evt}
		_ = p.ScanOnce(context.Background())
	}
	return alerts.created
}

func TestWindowedRule_BelowThresholdDoesNotFire(t *testing.T) {
	t.Parallel()
	rule := windowedRule(time.Minute, 3)
	t0 := at()
	events := []*listener.EventRecord{
		syslogEvent("error", "10.0.0.1:514", t0, "a"),
		syslogEvent("error", "10.0.0.1:514", t0.Add(time.Second), "b"),
	}
	got := runScanWithEvents(t, rule, events)
	if len(got) != 0 {
		t.Errorf("two hits with threshold=3 should not fire; got %d alerts", len(got))
	}
}

func TestWindowedRule_ThirdHitFires(t *testing.T) {
	t.Parallel()
	rule := windowedRule(time.Minute, 3)
	t0 := at()
	events := []*listener.EventRecord{
		syslogEvent("error", "10.0.0.1:514", t0, "a"),
		syslogEvent("error", "10.0.0.1:514", t0.Add(time.Second), "b"),
		syslogEvent("error", "10.0.0.1:514", t0.Add(2*time.Second), "c"),
	}
	got := runScanWithEvents(t, rule, events)
	if len(got) != 1 {
		t.Errorf("third hit inside window should fire once; got %d alerts", len(got))
	}
}

func TestWindowedRule_StaleHitsDoNotAccumulate(t *testing.T) {
	t.Parallel()
	rule := windowedRule(time.Minute, 3)
	t0 := at()
	events := []*listener.EventRecord{
		syslogEvent("error", "10.0.0.1:514", t0, "a"),
		syslogEvent("error", "10.0.0.1:514", t0.Add(time.Second), "b"),
		// Two minutes later — first two are past the window.
		syslogEvent("error", "10.0.0.1:514", t0.Add(2*time.Minute), "c"),
	}
	got := runScanWithEvents(t, rule, events)
	if len(got) != 0 {
		t.Errorf("expired predecessors should not count toward threshold; got %d alerts", len(got))
	}
}

func TestWindowedRule_DifferentEntitiesIndependent(t *testing.T) {
	t.Parallel()
	// Tighter window exercises the helper's window parameter across
	// values so unparam doesn't flag it as constant-folded.
	rule := windowedRule(30*time.Second, 2)
	t0 := at()
	events := []*listener.EventRecord{
		syslogEvent("error", "10.0.0.1:514", t0, "a-1"),
		syslogEvent("error", "10.0.0.2:514", t0.Add(time.Second), "b-1"),
	}
	got := runScanWithEvents(t, rule, events)
	if len(got) != 0 {
		t.Errorf("first hit per entity should not fire (threshold=2); got %d", len(got))
	}
}

func TestWindowedRule_DefaultPreservesLegacyBehavior(t *testing.T) {
	t.Parallel()
	// Window=0, Threshold=1 (defaults): fire on first match.
	rule := &alertmodel.Rule{
		ID: 1, Name: "legacy", Enabled: true,
		MatchKind:     "syslog-udp",
		MatchSeverity: "error",
		AlertType:     alertmodel.TypeSystem,
		AlertSeverity: alertmodel.SeverityError,
		AlertTitle:    "Legacy",
	}
	t0 := at()
	got := runScanWithEvents(t, rule,
		[]*listener.EventRecord{syslogEvent("error", "10.0.0.1:514", t0, "x")})
	if len(got) != 1 {
		t.Errorf("legacy threshold=1 should fire on first match; got %d", len(got))
	}
}
