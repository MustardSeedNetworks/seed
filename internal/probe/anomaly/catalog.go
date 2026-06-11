// Package probeanomaly is the active-monitoring anomaly source (ADR-0025): it
// maps the probe engine's threshold breaches onto the unified anomaly engine's
// detections and runs the long-lived producer that persists them under
// source=probe. The probe engine (internal/probe) stays unaware of anomalies; it
// emits ResultEvents on a fan-out channel and this package consumes them, exactly
// as internal/wifi/visibility consumes the Wi-Fi airspace. CGO-free, no I/O of
// its own beyond the anomaly store.
package probeanomaly

import "github.com/MustardSeedNetworks/seed/internal/anomaly"

// Exported def IDs are the stable catalog keys for active-monitoring anomalies.
// A probe Breach selects one by its Field; the API/UI key off these and tests
// assert against them, so they live as constants rather than scattered literals.
const (
	// DefUnreachable is the probe failing outright (Breach.Field == "success").
	DefUnreachable = "probe-unreachable"
	// DefHighLatency is a latency threshold breach (Breach.Field == "latency_ms").
	DefHighLatency = "probe-high-latency"
	// DefCertExpiry is a certificate nearing or past expiry (Breach.Field ==
	// "cert_days_remaining"): a TLS-family probe's leaf certificate has fewer days
	// left than the configured warning/critical bound.
	DefCertExpiry = "probe-cert-expiry"
	// DefThresholdBreach is the generic fallback for any other breached field a
	// checker reports (e.g. a future BGP state field) before that field has a
	// dedicated def. It keeps a breach from being silently dropped.
	DefThresholdBreach = "probe-threshold-breach"
)

// Catalog builds and validates the probe anomaly catalog. It fails fast if a
// definition is malformed, so a typo in the data below cannot ship a blank card.
func Catalog() (*anomaly.Catalog, error) {
	return anomaly.NewCatalog(Defs()...)
}

// Defs returns the probe anomaly definitions. Copy is authored originally; the
// severities are catalog defaults — a Breach overrides per detection (warning vs
// critical threshold), and the engine may escalate on recurrence. All are
// CategoryNetHealth: active-monitoring observations of a target's reachability
// and responsiveness.
func Defs() []anomaly.Def {
	return []anomaly.Def{
		{
			ID:              DefUnreachable,
			Category:        anomaly.CategoryNetHealth,
			DefaultSeverity: anomaly.SeverityCritical,
			Title:           "Monitored target unreachable",
			Description: "A scheduled probe failed: the target did not respond, or the " +
				"observation could not complete. The service this probe checks is unavailable " +
				"from this vantage point.",
			Impact: "The monitored service is down or unreachable from the appliance. " +
				"Dependent users or systems cannot reach it until it recovers.",
			Recommendation: "Confirm the target is up and the path to it (DNS, routing, " +
				"firewall) is intact. Check the probe's recent results for when it last " +
				"succeeded, and whether other probes to the same target or segment also fail.",
		},
		{
			ID:              DefHighLatency,
			Category:        anomaly.CategoryNetHealth,
			DefaultSeverity: anomaly.SeverityWarning,
			Title:           "Monitored target latency over threshold",
			Description: "A scheduled probe's measured latency exceeded its configured " +
				"warning or critical threshold. The target is reachable but responding more " +
				"slowly than the operator's defined bound.",
			Impact: "Elevated response time degrades the user or system experience of the " +
				"monitored service and can foreshadow saturation or an upstream problem.",
			Recommendation: "Compare current latency against the probe's baseline history. " +
				"Check for path congestion, target load, or an intermediate hop adding delay; " +
				"correlate with other probes sharing the path.",
		},
		{
			ID:              DefCertExpiry,
			Category:        anomaly.CategoryNetHealth,
			DefaultSeverity: anomaly.SeverityWarning,
			Standards:       []string{"RFC 5280 §4.1.2.5"},
			Title:           "TLS certificate nearing expiry",
			Description: "A TLS probe's leaf certificate has fewer days remaining than its " +
				"configured warning or critical bound (a negative value means the certificate " +
				"has already expired). The handshake still succeeds, so the target is reachable, " +
				"but the certificate needs renewal.",
			Impact: "Once the certificate expires, clients that validate it will refuse the " +
				"connection, causing an outage of the service this probe checks. Renewing late " +
				"risks the gap between expiry and replacement.",
			Recommendation: "Renew or rotate the certificate before it expires. Confirm the " +
				"automated renewal pipeline (for example ACME) is healthy, and check the " +
				"certificate's issuer and chain in the anomaly evidence.",
		},
		{
			ID:              DefThresholdBreach,
			Category:        anomaly.CategoryNetHealth,
			DefaultSeverity: anomaly.SeverityWarning,
			Title:           "Monitored target threshold breach",
			Description: "A scheduled probe reported a value outside its configured " +
				"threshold for a check-specific field (for example a protocol-state field) " +
				"that does not yet have a dedicated anomaly type. The exact field is in the " +
				"anomaly evidence.",
			Impact: "A monitored condition has crossed the operator's defined bound and " +
				"needs attention before it escalates to an outage.",
			Recommendation: "Read the breached field, threshold, and actual value in the " +
				"evidence, then remediate that specific condition on the target.",
		},
	}
}
