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
// V1.0 keeps titles and messages as literal strings. Template support
// lands in #1378 by extending CompileRulesFromDB to parse AlertTitle/
// AlertMessage as text/template before returning the Rule.

import (
	"context"
	"strconv"
	"strings"

	"github.com/krisarmstrong/seed/internal/database"
)

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
	alertTitle := row.AlertTitle
	alertMessage := row.AlertMessage

	return Rule{
		ID: "db." + strconv.FormatInt(row.ID, 10),
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
			return &database.Alert{
				Type:     alertType,
				Severity: alertSeverity,
				Title:    alertTitle,
				Message:  alertMessage,
				Source:   evt.SourceAddr,
				Metadata: evt.PayloadJSON,
			}
		},
	}
}

// alertRulesReader is the narrow surface the listener pipeline reads
// from the alert_rules repo. Tests inject a fake. Implemented by
// *database.AlertRulesRepository.
type alertRulesReader interface {
	List(ctx context.Context, enabledOnly bool) ([]*database.AlertRule, error)
}
