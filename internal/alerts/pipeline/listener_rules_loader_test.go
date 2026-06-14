package pipeline_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	alertmodel "github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

// fakeAlertRulesReader implements the narrow alertRulesReader contract.
// Mutating the rows slice is safe — every List call returns a fresh
// copy so the pipeline can hold its own snapshot.
type fakeAlertRulesReader struct {
	mu      sync.Mutex
	rows    []*alertmodel.Rule
	listErr error
	calls   int
}

func (f *fakeAlertRulesReader) List(_ context.Context, enabledOnly bool) ([]*alertmodel.Rule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*alertmodel.Rule, 0, len(f.rows))
	for _, r := range f.rows {
		if enabledOnly && !r.Enabled {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeAlertRulesReader) set(rows []*alertmodel.Rule) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = rows
}

func dbRule(name, kind, severity, contains string, enabled bool) *alertmodel.Rule {
	return &alertmodel.Rule{
		ID:                   int64(len(name)),
		Name:                 name,
		Enabled:              enabled,
		MatchKind:            kind,
		MatchSeverity:        severity,
		MatchPayloadContains: contains,
		AlertType:            alertmodel.TypeSystem,
		AlertSeverity:        alertmodel.SeverityError,
		AlertTitle:           "Rule " + name + " matched",
		AlertMessage:         "Triggered by listener event",
	}
}

func TestCompileRulesFromDB_EmptySliceReturnsEmpty(t *testing.T) {
	t.Parallel()
	got := pipeline.CompileRulesFromDB(nil)
	if len(got) != 0 {
		t.Errorf("nil input -> %d rules, want 0 (caller falls back to defaults)", len(got))
	}
}

func TestCompileRulesFromDB_DisabledRowsSkipped(t *testing.T) {
	t.Parallel()
	rows := []*alertmodel.Rule{
		dbRule("active", "syslog-udp", "error", "", true),
		dbRule("inactive", "syslog-udp", "error", "", false),
	}
	got := pipeline.CompileRulesFromDB(rows)
	if len(got) != 1 {
		t.Fatalf("got %d compiled rules, want 1 (disabled row should be filtered)", len(got))
	}
	if got[0].ID != "db.6" { // len("active") = 6
		t.Errorf("rule ID = %q, want db.6", got[0].ID)
	}
}

func TestCompileRulesFromDB_MatchKindFilter(t *testing.T) {
	t.Parallel()
	rows := []*alertmodel.Rule{
		dbRule("syslog-only", "syslog-udp", "", "", true),
	}
	rules := pipeline.CompileRulesFromDB(rows)
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	rule := rules[0]

	syslogEvt := &listener.EventRecord{Kind: "syslog-udp"}
	trapEvt := &listener.EventRecord{Kind: "snmp-trap-v2c"}

	if !rule.Match(syslogEvt) {
		t.Error("syslog-udp event should match syslog-udp filter")
	}
	if rule.Match(trapEvt) {
		t.Error("snmp-trap event should NOT match syslog-udp filter")
	}
}

func TestCompileRulesFromDB_MatchKindEmptyMatchesAll(t *testing.T) {
	t.Parallel()
	rows := []*alertmodel.Rule{
		dbRule("any-kind", "", "", "", true),
	}
	rule := pipeline.CompileRulesFromDB(rows)[0]
	if !rule.Match(&listener.EventRecord{Kind: "syslog-udp"}) {
		t.Error("empty match_kind should match syslog-udp")
	}
	if !rule.Match(&listener.EventRecord{Kind: "snmp-trap-v2c"}) {
		t.Error("empty match_kind should match snmp-trap-v2c")
	}
}

func TestCompileRulesFromDB_MatchSeverityFilter(t *testing.T) {
	t.Parallel()
	rows := []*alertmodel.Rule{
		dbRule("error-only", "syslog-udp", "error", "", true),
	}
	rule := pipeline.CompileRulesFromDB(rows)[0]
	if !rule.Match(&listener.EventRecord{Kind: "syslog-udp", Severity: "error"}) {
		t.Error("severity=error should match")
	}
	if rule.Match(&listener.EventRecord{Kind: "syslog-udp", Severity: "informational"}) {
		t.Error("severity=informational should NOT match error-only rule")
	}
}

func TestCompileRulesFromDB_MatchPayloadContainsSubstring(t *testing.T) {
	t.Parallel()
	rows := []*alertmodel.Rule{
		dbRule("link-state", "syslog-udp", "", "link state", true),
	}
	rule := pipeline.CompileRulesFromDB(rows)[0]

	payload, _ := json.Marshal(map[string]string{"message": "interface eth0 link state changed to down"})
	matching := &listener.EventRecord{Kind: "syslog-udp", PayloadJSON: string(payload)}
	if !rule.Match(matching) {
		t.Error("substring 'link state' should match")
	}

	nonMatching := &listener.EventRecord{Kind: "syslog-udp", PayloadJSON: `{"message":"ssh login from 10.0.0.1"}`}
	if rule.Match(nonMatching) {
		t.Error("payload without substring should NOT match")
	}
}

func TestCompileRulesFromDB_BuildPopulatesAlertFields(t *testing.T) {
	t.Parallel()
	rows := []*alertmodel.Rule{
		{
			ID: 42, Name: "interface-down", Enabled: true,
			MatchKind:     "syslog-udp",
			AlertType:     alertmodel.TypeConnectivity,
			AlertSeverity: alertmodel.SeverityWarning,
			AlertTitle:    "Interface down",
			AlertMessage:  "Saw an interface-down event",
		},
	}
	rule := pipeline.CompileRulesFromDB(rows)[0]
	alert := rule.Build(&listener.EventRecord{
		Kind: "syslog-udp", SourceAddr: "10.0.0.1:514",
		PayloadJSON: `{"message":"eth0 down"}`,
	})
	if alert == nil {
		t.Fatal("Build returned nil alert")
	}
	if alert.Type != alertmodel.TypeConnectivity {
		t.Errorf("Type = %q, want %q", alert.Type, alertmodel.TypeConnectivity)
	}
	if alert.Severity != alertmodel.SeverityWarning {
		t.Errorf("Severity = %q, want %q", alert.Severity, alertmodel.SeverityWarning)
	}
	if alert.Title != "Interface down" {
		t.Errorf("Title = %q, want literal 'Interface down'", alert.Title)
	}
	if alert.Message != "Saw an interface-down event" {
		t.Errorf("Message = %q", alert.Message)
	}
	if alert.Source != "10.0.0.1:514" {
		t.Errorf("Source = %q, want event source", alert.Source)
	}
	if alert.Metadata != `{"message":"eth0 down"}` {
		t.Errorf("Metadata should mirror PayloadJSON, got %q", alert.Metadata)
	}
}

func TestCompileRulesFromDB_StableFingerprintID(t *testing.T) {
	t.Parallel()
	rows := []*alertmodel.Rule{{
		ID: 7, Name: "x", Enabled: true,
		AlertType: alertmodel.TypeSystem, AlertSeverity: alertmodel.SeverityInfo,
		AlertTitle: "x",
	}}
	rule := pipeline.CompileRulesFromDB(rows)[0]
	if rule.ID != "db.7" {
		t.Errorf("rule.ID = %q, want db.7 (used in suppression fingerprint)", rule.ID)
	}
}

// Integration tests for the pipeline+loader composition.

func TestScanOnce_DBRulesReplaceDefaultsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	reader := &fakeAlertRulesReader{rows: []*alertmodel.Rule{
		// One DB rule matching only "informational" syslog — would NEVER
		// fire under the defaults (defaults only alert on error+ severities).
		dbRule("info-rule", "syslog-udp", "informational", "", true),
	}}
	events := &fakeEvents{rows: []*listener.EventRecord{
		// An error syslog. Under DEFAULTS this would fire (defaults match error+).
		// Under DB rules only it should NOT fire (DB rule is informational-only).
		syslogEvent("error", "10.0.0.1:514", at(), "boom"),
		// An informational syslog. Under DEFAULTS this does NOT fire.
		// Under DB rules it SHOULD fire (DB rule matches informational).
		syslogEvent("informational", "10.0.0.2:514", at(), "noise"),
	}}
	alerts := &fakeAlerts{}
	p, err := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
		AlertRules: reader,
	})
	if err != nil {
		t.Fatalf("NewListenerPipeline: %v", err)
	}
	if scanErr := p.ScanOnce(context.Background()); scanErr != nil {
		t.Fatalf("ScanOnce: %v", scanErr)
	}
	if len(alerts.created) != 1 {
		t.Fatalf(
			"got %d alerts, want 1 (DB-only: informational fires, error must NOT)",
			len(alerts.created),
		)
	}
	if alerts.created[0].Source != "10.0.0.2:514" {
		t.Errorf("alert came from %q, expected 10.0.0.2:514 (the informational source)", alerts.created[0].Source)
	}
}

func TestScanOnce_FallsBackToDefaultsWhenDBEmpty(t *testing.T) {
	t.Parallel()
	reader := &fakeAlertRulesReader{rows: nil}
	events := &fakeEvents{rows: []*listener.EventRecord{
		syslogEvent("error", "10.0.0.1:514", at(), "default-must-fire"),
	}}
	alerts := &fakeAlerts{}
	p, err := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
		AlertRules: reader,
	})
	if err != nil {
		t.Fatalf("NewListenerPipeline: %v", err)
	}
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Errorf(
			"empty DB rules should fall back to defaults (severe syslog fires), got %d",
			len(alerts.created),
		)
	}
}

func TestScanOnce_FallsBackToDefaultsWhenAllDBRulesDisabled(t *testing.T) {
	t.Parallel()
	reader := &fakeAlertRulesReader{rows: []*alertmodel.Rule{
		dbRule("disabled", "syslog-udp", "informational", "", false),
	}}
	events := &fakeEvents{rows: []*listener.EventRecord{
		syslogEvent("error", "10.0.0.1:514", at(), "boom"),
	}}
	alerts := &fakeAlerts{}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
		AlertRules: reader,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Errorf("disabled DB rules => fall back to defaults (severe syslog should fire), got %d", len(alerts.created))
	}
}

func TestScanOnce_ReloadPicksUpNewlyInsertedRule(t *testing.T) {
	t.Parallel()
	reader := &fakeAlertRulesReader{rows: nil}
	events := &fakeEvents{rows: []*listener.EventRecord{
		syslogEvent("informational", "10.0.0.1:514", at(), "ignore-me-under-defaults"),
	}}
	alerts := &fakeAlerts{}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
		AlertRules: reader,
	})

	// First scan: DB empty -> defaults active -> informational does NOT fire.
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 0 {
		t.Fatalf("baseline scan should not alert, got %d", len(alerts.created))
	}

	// Operator inserts a rule, then triggers reload.
	reader.set([]*alertmodel.Rule{dbRule("informational-now-fires", "syslog-udp", "informational", "", true)})
	if reloadErr := p.ReloadRules(context.Background()); reloadErr != nil {
		t.Fatalf("ReloadRules: %v", reloadErr)
	}

	// Reset high-water so the same event is re-considered, then scan again.
	// In production a new event arrives; here we just bump the fake events
	// observed-at by 1ns to step the high-water forward.
	events.rows = []*listener.EventRecord{
		syslogEvent("informational", "10.0.0.2:514", at().Add(time.Second), "fire-now"),
	}
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Errorf("after reload, informational event should fire via DB rule, got %d alerts", len(alerts.created))
	}
}

func TestScanOnce_ReloadErrorKeepsPreviousRuleset(t *testing.T) {
	t.Parallel()
	reader := &fakeAlertRulesReader{rows: []*alertmodel.Rule{
		dbRule("starter", "syslog-udp", "informational", "", true),
	}}
	events := &fakeEvents{rows: []*listener.EventRecord{
		syslogEvent("informational", "10.0.0.1:514", at(), "should-fire"),
	}}
	alerts := &fakeAlerts{}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
		AlertRules: reader,
	})
	// Now break the reader and call reload — pipeline should retain the
	// starter rule rather than blanking the ruleset.
	reader.mu.Lock()
	reader.listErr = errAlertRulesReadFailed
	reader.mu.Unlock()
	if err := p.ReloadRules(context.Background()); err == nil {
		t.Error("expected reload error to surface")
	}
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Errorf("after failed reload, prior rule should still fire, got %d alerts", len(alerts.created))
	}
}

var errAlertRulesReadFailed = &alertRulesReadError{}

type alertRulesReadError struct{}

func (*alertRulesReadError) Error() string { return "alert_rules read failed" }
