package pipeline_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

func TestCompileRulesFromDB_LiteralTitlePassesThrough(t *testing.T) {
	t.Parallel()
	row := &database.AlertRule{
		ID: 1, Enabled: true,
		AlertType: database.AlertTypeSystem, AlertSeverity: database.AlertSeverityError,
		AlertTitle:   "Static title with no template syntax",
		AlertMessage: "Static message",
	}
	rule := pipeline.CompileRulesFromDB([]*database.AlertRule{row})[0]
	alert := rule.Build(&database.ListenerEvent{SourceAddr: "10.0.0.1"})
	if alert.Title != "Static title with no template syntax" {
		t.Errorf("literal title was mutated: %q", alert.Title)
	}
}

func TestCompileRulesFromDB_TemplateRendersSourceAddr(t *testing.T) {
	t.Parallel()
	row := &database.AlertRule{
		ID: 1, Enabled: true,
		AlertType: database.AlertTypeSystem, AlertSeverity: database.AlertSeverityError,
		AlertTitle:   "{{.SourceAddr}} reported {{.Severity}}",
		AlertMessage: "Event arrived from {{.SourceAddr}}",
	}
	rule := pipeline.CompileRulesFromDB([]*database.AlertRule{row})[0]
	alert := rule.Build(&database.ListenerEvent{
		SourceAddr:  "10.0.0.1:514",
		Severity:    "error",
		Kind:        "syslog-udp",
		PayloadJSON: `{}`,
	})
	if alert.Title != "10.0.0.1:514 reported error" {
		t.Errorf("title = %q, want \"10.0.0.1:514 reported error\"", alert.Title)
	}
	if alert.Message != "Event arrived from 10.0.0.1:514" {
		t.Errorf("message = %q", alert.Message)
	}
}

func TestCompileRulesFromDB_TemplateAccessesPayload(t *testing.T) {
	t.Parallel()
	row := &database.AlertRule{
		ID: 1, Enabled: true,
		AlertType: database.AlertTypeSystem, AlertSeverity: database.AlertSeverityError,
		AlertTitle: "Saw {{.Payload.message}}",
	}
	rule := pipeline.CompileRulesFromDB([]*database.AlertRule{row})[0]
	payload, _ := json.Marshal(map[string]any{"message": "interface eth0 down"})
	alert := rule.Build(&database.ListenerEvent{
		SourceAddr:  "10.0.0.1",
		PayloadJSON: string(payload),
	})
	if !strings.Contains(alert.Title, "interface eth0 down") {
		t.Errorf("title should include payload.message, got %q", alert.Title)
	}
}

func TestCompileRulesFromDB_BrokenTemplateFallsBackToLiteral(t *testing.T) {
	t.Parallel()
	// "{{" without close — parse error at compile time.
	row := &database.AlertRule{
		ID: 1, Enabled: true,
		AlertType: database.AlertTypeSystem, AlertSeverity: database.AlertSeverityError,
		AlertTitle: "Unclosed {{ template",
	}
	// Should not panic + rule should still load.
	rules := pipeline.CompileRulesFromDB([]*database.AlertRule{row})
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1 (broken template should not drop the rule)", len(rules))
	}
	alert := rules[0].Build(&database.ListenerEvent{SourceAddr: "10.0.0.1"})
	if alert.Title != "Unclosed {{ template" {
		t.Errorf("broken template should fall back to literal, got %q", alert.Title)
	}
}

func TestCompileRulesFromDB_MissingPayloadFieldRendersZero(t *testing.T) {
	t.Parallel()
	// text/template's default behavior on missing map keys is to
	// render "<no value>" — confirm we don't crash on it and that
	// the output is something deterministic.
	row := &database.AlertRule{
		ID: 1, Enabled: true,
		AlertType: database.AlertTypeSystem, AlertSeverity: database.AlertSeverityError,
		AlertTitle: "Saw {{.Payload.nonexistent}}",
	}
	rule := pipeline.CompileRulesFromDB([]*database.AlertRule{row})[0]
	alert := rule.Build(&database.ListenerEvent{
		SourceAddr:  "10.0.0.1",
		PayloadJSON: `{"message":"hello"}`,
	})
	// Just confirm we got a non-empty title and didn't crash. The
	// exact rendering of missing keys is text/template's contract
	// (`<no value>`), not ours.
	if alert.Title == "" {
		t.Error("missing payload field should still render a title")
	}
}

func TestCompileRulesFromDB_PayloadDecodeFailureRendersEmptyMap(t *testing.T) {
	t.Parallel()
	row := &database.AlertRule{
		ID: 1, Enabled: true,
		AlertType: database.AlertTypeSystem, AlertSeverity: database.AlertSeverityError,
		AlertTitle: "Got {{.Kind}}",
	}
	rule := pipeline.CompileRulesFromDB([]*database.AlertRule{row})[0]
	alert := rule.Build(&database.ListenerEvent{
		SourceAddr:  "10.0.0.1",
		Kind:        "syslog-udp",
		PayloadJSON: `not valid json`, // garbage
	})
	if !strings.Contains(alert.Title, "syslog-udp") {
		t.Errorf("non-payload templates should still render with bad payload, got %q", alert.Title)
	}
}

func TestCompileRulesFromDB_TemplateAccessesMatchedRuleName(t *testing.T) {
	t.Parallel()
	row := &database.AlertRule{
		ID: 1, Enabled: true, Name: "my-rule",
		AlertType: database.AlertTypeSystem, AlertSeverity: database.AlertSeverityError,
		AlertTitle: "Rule {{.MatchedRuleName}} fired",
	}
	rule := pipeline.CompileRulesFromDB([]*database.AlertRule{row})[0]
	alert := rule.Build(&database.ListenerEvent{SourceAddr: "10.0.0.1"})
	if alert.Title != "Rule my-rule fired" {
		t.Errorf("title = %q, want \"Rule my-rule fired\"", alert.Title)
	}
}
