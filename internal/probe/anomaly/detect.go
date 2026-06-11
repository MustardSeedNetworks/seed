package probeanomaly

import (
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// Detections maps one probe ResultEvent onto anomaly detections — one per
// threshold breach (ADR-0025). A clean run (no breaches) yields none, so the
// probe's instances age out on silence (the producer's Prune). The coalescing
// subject is the probe itself (SubjectProbe keyed by ProbeID), so repeated
// breaches of the same probe update one instance rather than piling up.
func Detections(re probe.ResultEvent) []anomaly.Detection {
	if len(re.Breaches) == 0 {
		return nil
	}
	out := make([]anomaly.Detection, 0, len(re.Breaches))
	for _, b := range re.Breaches {
		out = append(out, anomaly.Detection{
			DefKey:   defKeyForField(b.Field),
			Subject:  anomaly.SubjectRef{Kind: anomaly.SubjectProbe, ID: b.ProbeID},
			Severity: severityFor(b.Severity),
			Evidence: evidence(re.Result, b),
		})
	}
	return out
}

// defKeyForField selects the catalog def for a breached field, falling back to
// the generic threshold-breach def so a new checker-specific field is surfaced
// rather than dropped.
func defKeyForField(field string) string {
	switch field {
	case "success":
		return DefUnreachable
	case "latency_ms":
		return DefHighLatency
	case "cert_days_remaining":
		return DefCertExpiry
	default:
		return DefThresholdBreach
	}
}

// severityFor maps a probe breach severity string onto an anomaly severity. An
// unrecognized value returns "" so the engine applies the catalog default rather
// than rejecting the detection.
func severityFor(s string) anomaly.Severity {
	switch s {
	case "info":
		return anomaly.SeverityInfo
	case "warning":
		return anomaly.SeverityWarning
	case "critical":
		return anomaly.SeverityCritical
	default:
		return ""
	}
}

// evidence captures the live measured values behind a breach so the anomaly card
// shows what tripped: the breached field with its threshold and actual, plus the
// probe kind and any error string. Values are stringified (the wire model is
// map[string]string).
func evidence(r probe.Result, b probe.Breach) map[string]string {
	ev := map[string]string{
		"field":     b.Field,
		"threshold": fmt.Sprint(b.Threshold),
		"actual":    fmt.Sprint(b.Actual),
		"kind":      r.Kind,
	}
	if r.Error != "" {
		ev["error"] = r.Error
	}
	return ev
}
