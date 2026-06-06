package wifianomaly

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/krisarmstrong/seed/internal/anomaly"
	"github.com/krisarmstrong/seed/internal/wifi/airspace"
)

// defaultCoChannelThreshold is the number of BSSes sharing one channel at or
// above which co-channel contention is reported. Four+ radios on a channel is
// where CSMA/CA airtime division becomes a practical problem.
const defaultCoChannelThreshold = 4

// minCoChannelThreshold is the floor for a configured co-channel threshold (a
// single radio cannot contend with itself).
const minCoChannelThreshold = 2

// minDistinctForMismatch is the number of distinct values (vendors, standards)
// across one SSID that constitutes a mismatch.
const minDistinctForMismatch = 2

// Detector evaluates the airspace tree against the Wi-Fi rule set and returns
// the detections to feed an [anomaly.Engine]. It is stateless apart from its
// tuning thresholds; the engine owns coalescing, escalation, and ageing.
type Detector struct {
	coChannelThreshold int
}

// Option tunes a Detector.
type Option func(*Detector)

// WithCoChannelThreshold sets the co-channel-contention reporting threshold
// (BSSes per channel). Values below 2 are clamped to 2.
func WithCoChannelThreshold(n int) Option {
	return func(d *Detector) {
		if n < minCoChannelThreshold {
			n = minCoChannelThreshold
		}
		d.coChannelThreshold = n
	}
}

// NewDetector returns a Detector with the given options applied.
func NewDetector(opts ...Option) *Detector {
	d := &Detector{coChannelThreshold: defaultCoChannelThreshold}
	for _, o := range opts {
		o(d)
	}
	return d
}

// Detect runs every Wi-Fi rule over the airspace tree and returns the resulting
// detections in a deterministic order (by def, then subject), so identical input
// yields identical output. The engine, not Detect, decides severity escalation
// and deduplication.
func (d *Detector) Detect(tree []airspace.SSIDGroup) []anomaly.Detection {
	var out []anomaly.Detection
	out = append(out, perBSSDetections(tree)...)
	out = append(out, ssidGroupDetections(tree)...)
	out = append(out, d.coChannelDetections(tree)...)
	out = append(out, countryConflictDetections(tree)...)

	sort.Slice(out, func(i, j int) bool {
		if out[i].DefKey != out[j].DefKey {
			return out[i].DefKey < out[j].DefKey
		}
		if out[i].Subject.Kind != out[j].Subject.Kind {
			return out[i].Subject.Kind < out[j].Subject.Kind
		}
		return out[i].Subject.ID < out[j].Subject.ID
	})
	return out
}

// forEachBSS visits every BSS in the tree with its AP and SSID-group context.
func forEachBSS(tree []airspace.SSIDGroup, fn func(g airspace.SSIDGroup, ap airspace.APGroup, b airspace.BSSView)) {
	for _, g := range tree {
		for _, ap := range g.APs {
			for _, b := range ap.BSSes {
				fn(g, ap, b)
			}
		}
	}
}

// perBSSDetections runs the rules that judge a single BSS in isolation:
// open/WEP/WPS, PMF-not-required, adjacent-channel overlap, and hidden SSID.
func perBSSDetections(tree []airspace.SSIDGroup) []anomaly.Detection {
	var out []anomaly.Detection
	forEachBSS(tree, func(_ airspace.SSIDGroup, _ airspace.APGroup, b airspace.BSSView) {
		subject := anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: b.BSSID}
		ev := bssEvidence(b)

		switch b.Security {
		case secOpen:
			out = append(out, anomaly.Detection{DefKey: DefOpenNetwork, Subject: subject, Evidence: ev})
		case secWEP:
			out = append(out, anomaly.Detection{DefKey: DefWEPInUse, Subject: subject, Evidence: ev})
		}

		if b.WPSEnabled {
			out = append(out, anomaly.Detection{DefKey: DefWPSEnabled, Subject: subject, Evidence: ev})
		}

		// PMF (802.11w) is only defined for RSN suites; flag an RSN BSS that
		// does not require it. Open/WEP/legacy-WPA carry no RSN, so PMF does
		// not apply and must not be reported.
		if securityTier(b.Security) == tierStrong && !b.PMFRequired {
			out = append(out, anomaly.Detection{DefKey: DefPMFNotRequired, Subject: subject, Evidence: ev})
		}

		if b.Band == band24GHz && b.Channel != 0 && !is24NonOverlapping(b.Channel) {
			out = append(out, anomaly.Detection{DefKey: DefAdjacentChannelOverlap, Subject: subject, Evidence: ev})
		}

		if b.Hidden {
			out = append(out, anomaly.Detection{DefKey: DefHiddenSSID, Subject: subject, Evidence: ev})
		}
	})
	return out
}

// ssidGroupDetections runs the rules that compare BSSes advertising the same
// SSID: security mismatch, evil-twin (vendor mismatch), and standard mismatch.
// The cloaked/empty-SSID bucket is skipped — its members are unrelated networks
// grouped only by their absent name, so cross-BSS comparison there is meaningless.
func ssidGroupDetections(tree []airspace.SSIDGroup) []anomaly.Detection {
	var out []anomaly.Detection
	for _, g := range tree {
		if g.SSID == "" {
			continue
		}
		subject := anomaly.SubjectRef{Kind: anomaly.SubjectSSID, ID: g.SSID}
		facts := collectSSIDFacts(g)
		securities, standards, vendors := facts.securities, facts.standards, facts.vendors

		if hasWeakAndStrong(securities) {
			out = append(out, anomaly.Detection{
				DefKey: DefSecurityMismatch, Subject: subject,
				Evidence: map[string]string{"securities": strings.Join(securities.sorted(), ", ")},
			})
		}
		if vendors.len() >= minDistinctForMismatch {
			out = append(out, anomaly.Detection{
				DefKey: DefEvilTwin, Subject: subject,
				Evidence: map[string]string{"vendors": strings.Join(vendors.sorted(), ", ")},
			})
		}
		if standards.len() >= minDistinctForMismatch {
			out = append(out, anomaly.Detection{
				DefKey: DefStandardMismatch, Subject: subject,
				Evidence: map[string]string{"standards": strings.Join(standards.sorted(), ", ")},
			})
		}
	}
	return out
}

// ssidFacts holds the distinct values advertised under one SSID group that the
// cross-BSS rules compare.
type ssidFacts struct {
	securities *stringSet
	standards  *stringSet
	vendors    *stringSet
}

// collectSSIDFacts gathers the distinct classified security suites, known
// 802.11 standards, and AP vendors advertised under one SSID group.
func collectSSIDFacts(g airspace.SSIDGroup) ssidFacts {
	f := ssidFacts{securities: newStringSet(), standards: newStringSet(), vendors: newStringSet()}
	for _, ap := range g.APs {
		if ap.Vendor != "" {
			f.vendors.add(ap.Vendor)
		}
		for _, b := range ap.BSSes {
			if t := securityTier(b.Security); t == tierWeak || t == tierStrong {
				f.securities.add(b.Security)
			}
			if b.Standard != "" && b.Standard != stdUnknown {
				f.standards.add(b.Standard)
			}
		}
	}
	return f
}

// coChannelDetections aggregates BSSes by (band, channel) across the whole
// airspace and reports each channel carrying at least the threshold count.
func (d *Detector) coChannelDetections(tree []airspace.SSIDGroup) []anomaly.Detection {
	byChannel := map[string][]string{} // "band ch N" -> bssids
	order := []string{}
	forEachBSS(tree, func(_ airspace.SSIDGroup, _ airspace.APGroup, b airspace.BSSView) {
		if b.Channel == 0 || b.Band == "" || b.Band == bandUnknown {
			return
		}
		key := fmt.Sprintf("%s ch %d", b.Band, b.Channel)
		if _, ok := byChannel[key]; !ok {
			order = append(order, key)
		}
		byChannel[key] = append(byChannel[key], b.BSSID)
	})

	var out []anomaly.Detection
	for _, key := range order {
		bssids := byChannel[key]
		if len(bssids) < d.coChannelThreshold {
			continue
		}
		sort.Strings(bssids)
		out = append(out, anomaly.Detection{
			DefKey:  DefCoChannelContention,
			Subject: anomaly.SubjectRef{Kind: anomaly.SubjectChannel, ID: key},
			Evidence: map[string]string{
				"count":  strconv.Itoa(len(bssids)),
				"bssids": strings.Join(bssids, ", "),
			},
		})
	}
	return out
}

// countryConflictDetections reports an 802.11d regulatory-domain disagreement:
// when more than one country code is advertised in the airspace, every BSS whose
// country differs from the reference domain is flagged. The reference is the
// lexicographically smallest code seen, chosen for determinism.
func countryConflictDetections(tree []airspace.SSIDGroup) []anomaly.Detection {
	countries := newStringSet()
	forEachBSS(tree, func(_ airspace.SSIDGroup, _ airspace.APGroup, b airspace.BSSView) {
		if b.CountryCode != "" {
			countries.add(b.CountryCode)
		}
	})
	if countries.len() < minDistinctForMismatch {
		return nil
	}
	reference := countries.sorted()[0]

	var out []anomaly.Detection
	forEachBSS(tree, func(_ airspace.SSIDGroup, _ airspace.APGroup, b airspace.BSSView) {
		if b.CountryCode == "" || b.CountryCode == reference {
			return
		}
		out = append(out, anomaly.Detection{
			DefKey:  DefCountryConflict,
			Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: b.BSSID},
			Evidence: map[string]string{
				"countryCode":     b.CountryCode,
				"referenceDomain": reference,
				"domainsSeen":     strings.Join(countries.sorted(), ", "),
			},
		})
	})
	return out
}

// bssEvidence captures the live values a per-BSS detection should carry.
func bssEvidence(b airspace.BSSView) map[string]string {
	ev := map[string]string{
		"ssid":     b.SSID,
		"security": b.Security,
		"band":     b.Band,
		"channel":  strconv.Itoa(b.Channel),
	}
	if b.SSID == "" {
		ev["ssid"] = "(hidden)"
	}
	return ev
}
