package pipeline

// listener_rules_loader compiles operator-configured alert_rules rows
// into the engine.Rule shape the listener pipeline already consumes.
//
// Semantics (#1377): when alert_rules has at least one enabled row,
// the DB ruleset replaces DefaultListenerRules entirely. When the
// table is empty (or every row is disabled), the pipeline falls back
// to DefaultListenerRules so a fresh install keeps emitting alerts
// without any operator action. Mixing the two ("additive merging")
// was rejected because it makes "why did this fire twice?" harder to
// answer than "my rules win or yours do".
//
// Template support (#1378): AlertTitle / AlertMessage are parsed as
// text/template at compile time. The runtime context (exposed to
// templates) is the ListenerEventContext struct below — operators
// can interpolate SourceAddr, Kind, Severity, ReceivedAt, MatchedRuleName,
// and the decoded payload map via .Payload.<field>. Strings without
// template syntax take a fast path that skips template execution
// entirely (zero allocs). A broken template (parse error) is logged
// elsewhere and falls back to the literal string so a typo never
// drops a rule from rotation.

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// ListenerEventContext is the data passed to AlertTitle / AlertMessage
// templates. Fields mirror the event the pipeline matched, plus the
// rule's own name so templates can self-identify.
//
// Operators interpolate via the standard text/template syntax:
//
//	{{.SourceAddr}}             - "10.0.0.1:514"
//	{{.Severity}}               - "error"
//	{{.Kind}}                   - "syslog-udp"
//	{{.ReceivedAt}}             - the event ObservedAt timestamp
//	{{.MatchedRuleName}}        - the rule's Name column
//	{{.Payload.message}}        - decoded JSON payload fields
type ListenerEventContext struct {
	SourceAddr      string
	Kind            string
	Severity        string
	ReceivedAt      time.Time
	Payload         map[string]any
	MatchedRuleName string
}

// CompileRulesFromDB converts every enabled alert_rules row into a
// runtime Rule. Disabled rows are dropped. Returns an empty slice
// when no enabled rows are present so callers can decide to fall
// back to DefaultListenerRules.
func CompileRulesFromDB(rows []*database.AlertRule) []Rule {
	out := make([]Rule, 0, len(rows))
	for _, row := range rows {
		if row == nil || !row.Enabled {
			continue
		}
		out = append(out, compileOne(row))
	}
	return out
}

func compileOne(row *database.AlertRule) Rule {
	matchKind := row.MatchKind
	matchSev := row.MatchSeverity
	matchSubstring := row.MatchPayloadContains

	// Snapshot the alert-construction inputs so Build is a closure
	// that doesn't accidentally pick up a future row mutation.
	alertType := row.AlertType
	alertSeverity := row.AlertSeverity
	ruleName := row.Name

	titleRenderer := compileTemplate(row.AlertTitle)
	messageRenderer := compileTemplate(row.AlertMessage)

	threshold := max(row.ThresholdCount, 1)
	window := time.Duration(row.WindowSeconds) * time.Second
	var counter *windowCounter
	if threshold > 1 && window > 0 {
		counter = newWindowCounter(window, threshold)
	}

	return Rule{
		ID:        "db." + strconv.FormatInt(row.ID, 10),
		Threshold: threshold,
		Window:    window,
		counter:   counter,
		Match: func(evt *database.ListenerEvent) bool {
			if matchKind != "" && evt.Kind != matchKind {
				return false
			}
			if matchSev != "" && evt.Severity != matchSev {
				return false
			}
			if matchSubstring != "" && !strings.Contains(evt.PayloadJSON, matchSubstring) {
				return false
			}
			return true
		},
		Build: func(evt *database.ListenerEvent) *database.Alert {
			ctx := buildEventContext(evt, ruleName)
			return &database.Alert{
				Type:     alertType,
				Severity: alertSeverity,
				Title:    titleRenderer(ctx),
				Message:  messageRenderer(ctx),
				Source:   evt.SourceAddr,
				Metadata: evt.PayloadJSON,
			}
		},
	}
}

// templateRenderer is a closure that takes the event context and
// returns the rendered string. Literal-string rules use a no-op
// closure that just returns the original.
type templateRenderer func(ListenerEventContext) string

// compileTemplate returns a renderer for s. When s contains no
// template syntax ("{{"), the renderer is a fast literal-pass-
// through that allocates nothing. When s parses cleanly, the
// renderer executes the template against the event context. A
// parse failure leaves the literal in place so a typo never drops
// a rule.
func compileTemplate(s string) templateRenderer {
	if !strings.Contains(s, "{{") {
		return func(ListenerEventContext) string { return s }
	}
	tmpl, err := template.New("alert").Parse(s)
	if err != nil {
		return func(ListenerEventContext) string { return s }
	}
	return func(ctx ListenerEventContext) string {
		var buf bytes.Buffer
		if execErr := tmpl.Execute(&buf, ctx); execErr != nil {
			return s
		}
		return buf.String()
	}
}

// buildEventContext converts a ListenerEvent into the template-
// visible struct. Payload JSON is best-effort decoded into a map
// so templates can reach inside it via .Payload.<field>; a bad
// payload renders as an empty map.
func buildEventContext(evt *database.ListenerEvent, ruleName string) ListenerEventContext {
	payload := map[string]any{}
	if evt.PayloadJSON != "" {
		_ = json.Unmarshal([]byte(evt.PayloadJSON), &payload)
	}
	return ListenerEventContext{
		SourceAddr:      evt.SourceAddr,
		Kind:            evt.Kind,
		Severity:        evt.Severity,
		ReceivedAt:      evt.ObservedAt,
		Payload:         payload,
		MatchedRuleName: ruleName,
	}
}

// alertRulesReader is the narrow surface the listener pipeline reads
// from the alert_rules repo. Tests inject a fake. Implemented by
// *database.AlertRulesRepository.
type alertRulesReader interface {
	List(ctx context.Context, enabledOnly bool) ([]*database.AlertRule, error)
}
