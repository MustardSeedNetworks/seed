package wifianomaly

import (
	"sort"
	"strings"
)

// The airspace Tree exposes Security/Standard/Band as the strings produced by
// dot11's String() methods (those nouns are kept verbatim — DNT). The rules
// classify off those tokens, so the tokens live here as named constants and a
// drift guard (SecurityRecognized + TestSecurityTierCoversEveryDot11Suite)
// fails fast if dot11's vocabulary ever changes underneath us.
const (
	secOpen     = "Open"
	secWEP      = "WEP"
	secWPA      = "WPA"
	secWPA2     = "WPA2"
	secWPA3     = "WPA3"
	secWPA2WPA3 = "WPA2/WPA3"

	stdUnknown = "Unknown"

	band24GHz   = "2.4 GHz"
	bandUnknown = "Unknown"
)

// security strength tiers. tierUnknown is the zero value for an unclassified
// suite; weak suites (Open/WEP/legacy WPA) and strong suites (WPA2/WPA3) are
// the two that the mismatch rule compares.
const (
	tierUnknown = iota
	tierWeak
	tierStrong
)

// securityTier classifies a stringified security suite into weak/strong, or
// tierUnknown if the token is not recognised.
func securityTier(s string) int {
	switch s {
	case secOpen, secWEP, secWPA:
		return tierWeak
	case secWPA2, secWPA3, secWPA2WPA3:
		return tierStrong
	default:
		return tierUnknown
	}
}

// SecurityRecognized reports whether s is a security suite the detector knows
// how to classify. It exists so a test can assert every dot11.Security String()
// value is handled, catching drift between the decoder and these rules.
func SecurityRecognized(s string) bool { return securityTier(s) != tierUnknown }

// hasWeakAndStrong reports whether the set holds at least one weak and one
// strong security suite — the security-mismatch condition.
func hasWeakAndStrong(securities *stringSet) bool {
	weak, strong := false, false
	for _, s := range securities.sorted() {
		switch securityTier(s) {
		case tierWeak:
			weak = true
		case tierStrong:
			strong = true
		}
	}
	return weak && strong
}

// is24NonOverlapping reports whether ch is one of the 2.4 GHz non-overlapping
// channels (1, 6, 11). Any other 2.4 GHz channel partially overlaps a neighbour.
func is24NonOverlapping(ch int) bool {
	return ch == 1 || ch == 6 || ch == 11
}

// defaultSSIDPrefixes are well-known router/manufacturer default network-name
// prefixes. Matching is conservative (prefix on a lower-cased, separator-stripped
// SSID) to flag clearly-unconfigured devices without snaring intentional names.
func defaultSSIDPrefixes() []string {
	return []string{
		"linksys", "netgear", "dlink", "tplink", "belkin",
		"asus", "tendawifi", "default", "wireless",
	}
}

// isDefaultSSID reports whether ssid looks like an unconfigured manufacturer
// default. Empty/cloaked SSIDs are not considered defaults (handled elsewhere).
func isDefaultSSID(ssid string) bool {
	norm := strings.ToLower(ssid)
	norm = strings.NewReplacer("-", "", "_", "", " ", "").Replace(norm)
	if norm == "" {
		return false
	}
	for _, p := range defaultSSIDPrefixes() {
		if strings.HasPrefix(norm, p) {
			return true
		}
	}
	return false
}

// stringSet is a tiny ordered-output set used to collect distinct evidence
// values (securities, vendors, standards, countries) deterministically.
type stringSet struct {
	m map[string]struct{}
}

func newStringSet() *stringSet { return &stringSet{m: map[string]struct{}{}} }

func (s *stringSet) add(v string) { s.m[v] = struct{}{} }

func (s *stringSet) len() int { return len(s.m) }

func (s *stringSet) sorted() []string {
	out := make([]string, 0, len(s.m))
	for v := range s.m {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
