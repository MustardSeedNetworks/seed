// Package wifianomaly is the Wi-Fi rule source for the general network-anomaly
// engine (internal/anomaly, ADR-0011). It contributes two things:
//
//   - a data-driven [anomaly.Catalog] of Wi-Fi anomaly definitions, each with
//     originally-authored copy, IEEE 802.11 citations, an impact statement, a
//     remediation recommendation, and capability-gated follow-ups; and
//   - a [Detector] whose rules evaluate the cross-referenced airspace tree
//     ([airspace.Airspace.Tree]) and emit [anomaly.Detection] values for the
//     engine to coalesce, escalate, and project.
//
// The rules read only the deterministic, stringified Tree view (no live capture,
// no packet handles), so the package is CGO-free and exercised entirely with
// synthetic airspace trees. The directory is internal/wifi/anomaly; the package
// is named wifianomaly so the general engine package can be referenced here as
// the unqualified anomaly.
package wifianomaly
