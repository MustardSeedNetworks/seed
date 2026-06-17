package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

// listenerHighWaterKey is the settings key holding the latest
// ObservedAt the listener alert pipeline has already processed.
const listenerHighWaterKey = "alerts.listener.high_water"

// evaluate applies every rule to evt and writes any alerts that
// match + pass suppression. Returns the count of alerts emitted.
func (p *ListenerPipeline) evaluate(ctx context.Context, evt *listener.EventRecord) int {
	now := p.now()
	count := 0
	rules := p.snapshotRules()
	for _, rule := range rules {
		if !rule.Match(evt) {
			continue
		}
		fingerprint := fingerprintFor(rule.ID, evt.SourceAddr, evt.Kind)
		suppressed, suppErr := p.suppress.IsSuppressed(ctx, fingerprint, now)
		if suppErr != nil {
			p.logger.WarnContext(ctx, "suppression check failed",
				"rule", rule.ID, "error", suppErr)
		}
		if suppressed {
			continue
		}
		// Time-windowed threshold (#1379): only fire when the rule's
		// counter crosses Threshold inside Window. Counter==nil means
		// fire on first match (legacy path).
		if rule.counter != nil {
			if !rule.counter.Hit(evt.SourceAddr, now) {
				continue
			}
		}
		alert := rule.Build(evt)
		if alert == nil {
			continue
		}
		if writeErr := p.alerts.Create(ctx, alert); writeErr != nil {
			p.logger.WarnContext(ctx, "alert create failed",
				"rule", rule.ID, "source", evt.SourceAddr, "error", writeErr)
			continue
		}
		markErr := p.suppress.Mark(
			ctx, fingerprint, rule.ID, evt.SourceAddr, now.Add(p.suppression),
		)
		if markErr != nil {
			p.logger.WarnContext(ctx, "suppression mark failed",
				"rule", rule.ID, "error", markErr)
		}
		count++
	}
	return count
}

// fingerprintFor builds the suppression key.
func fingerprintFor(ruleID, source, kind string) string {
	h := sha256.New()
	h.Write([]byte(ruleID))
	h.Write([]byte{0x00})
	h.Write([]byte(source))
	h.Write([]byte{0x00})
	h.Write([]byte(kind))
	return hex.EncodeToString(h.Sum(nil))[:24]
}

func (p *ListenerPipeline) loadHighWater(ctx context.Context) (time.Time, error) {
	raw, err := p.settings.GetWithDefault(ctx, listenerHighWaterKey, "")
	if err != nil {
		return time.Time{}, err
	}
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse high-water %q: %w", raw, err)
	}
	return parsed, nil
}

func (p *ListenerPipeline) saveHighWater(ctx context.Context, t time.Time) error {
	return p.settings.Set(ctx, listenerHighWaterKey, t.UTC().Format(time.RFC3339Nano))
}

// DefaultListenerRules returns the V1.0 built-in rule set:
//
//   - syslog severity emergency/alert/critical/error -> alert
//   - snmp trap linkDown -> alert (warning)
//   - snmp trap authenticationFailure -> alert (error)
//   - snmp trap any other event -> alert (info) when an explicit
//     trap OID is present
//
// The function returns a fresh slice each call so a pipeline can
// safely append or filter without affecting other callers.
func DefaultListenerRules() []Rule {
	return []Rule{
		ruleSyslogSevereLogged(),
		ruleTrapLinkDown(),
		ruleTrapAuthFailure(),
	}
}

func ruleSyslogSevereLogged() Rule {
	severeSeverities := map[string]bool{
		"emergency": true,
		"alert":     true,
		"critical":  true,
		"error":     true,
	}
	return Rule{
		ID: "syslog.severe",
		Match: func(evt *listener.EventRecord) bool {
			return evt.Kind == "syslog-udp" && severeSeverities[evt.Severity]
		},
		Build: func(evt *listener.EventRecord) *alerts.Alert {
			return &alerts.Alert{
				Type:     alerts.TypeSystem,
				Severity: mapSyslogSeverity(evt.Severity),
				Title:    fmt.Sprintf("Syslog %s from %s", evt.Severity, evt.SourceAddr),
				Message:  summarize(evt.PayloadJSON, "message"),
				Source:   evt.SourceAddr,
				Metadata: evt.PayloadJSON,
			}
		},
	}
}

func ruleTrapLinkDown() Rule {
	return Rule{
		ID: "trap.linkdown",
		Match: func(evt *listener.EventRecord) bool {
			if evt.Kind != "snmp-trap-v2c" {
				return false
			}
			return strings.Contains(evt.PayloadJSON, `"1.3.6.1.6.3.1.1.5.3"`)
		},
		Build: func(evt *listener.EventRecord) *alerts.Alert {
			return &alerts.Alert{
				Type:     alerts.TypeConnectivity,
				Severity: alerts.SeverityWarning,
				Title:    "Link down trap from " + evt.SourceAddr,
				Message:  "SNMP linkDown trap received",
				Source:   evt.SourceAddr,
				Metadata: evt.PayloadJSON,
			}
		},
	}
}

func ruleTrapAuthFailure() Rule {
	return Rule{
		ID: "trap.authfail",
		Match: func(evt *listener.EventRecord) bool {
			if evt.Kind != "snmp-trap-v2c" {
				return false
			}
			return strings.Contains(evt.PayloadJSON, `"1.3.6.1.6.3.1.1.5.5"`)
		},
		Build: func(evt *listener.EventRecord) *alerts.Alert {
			return &alerts.Alert{
				Type:     alerts.TypeSecurity,
				Severity: alerts.SeverityError,
				Title:    "SNMP authentication failure from " + evt.SourceAddr,
				Message:  "Repeated authentication failures may indicate a credential probe",
				Source:   evt.SourceAddr,
				Metadata: evt.PayloadJSON,
			}
		},
	}
}

// mapSyslogSeverity bridges syslog severity names to the alerts
// table's enum.
func mapSyslogSeverity(s string) string {
	switch s {
	case "emergency", "alert", "critical":
		return alerts.SeverityCritical
	case "error":
		return alerts.SeverityError
	case "warning":
		return alerts.SeverityWarning
	default:
		return alerts.SeverityInfo
	}
}

// summarize pulls one string field out of a JSON payload without
// fully unmarshalling. Returns "" when the field is absent or the
// payload isn't valid JSON. Used to grab e.g. the syslog "message"
// for an alert title without forcing a struct definition per kind.
func summarize(payload, field string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return ""
	}
	raw, ok := m[field]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}
