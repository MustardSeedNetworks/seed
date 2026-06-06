package wifianomaly

// 802.11d regulatory checks for the 2.4 GHz band, where the channel plan is
// stable and the well-known cross-domain deltas are unambiguous. The 5/6 GHz
// plans (DFS, country-specific sub-bands) are deliberately out of scope here —
// encoding them wrong would produce false positives — so a 5/6 GHz channel is
// never evaluated by these helpers.
//
// The table is intentionally conservative: it returns chanForbidden only for the
// cases that are universally true, so a flagged channel is a real regulatory
// problem, not a guess. Anything we cannot assert confidently is chanUnknown.

// chanStatus is the regulatory verdict for a channel in a declared domain.
type chanStatus int

const (
	// chanUnknown means no confident rule applies (the caller emits nothing).
	chanUnknown chanStatus = iota
	// chanAllowed means the channel is permitted in the domain.
	chanAllowed
	// chanForbidden means the channel is not permitted in the domain.
	chanForbidden
)

// channel2GHz14 is JP-only worldwide (and only for 802.11b). Channels 12-13 are
// permitted across most of the world but not in the US/Canada FCC domain.
const channel2GHz14 = 14

// usFCCDomains are the regulatory domains limited to 2.4 GHz channels 1-11.
func usFCCDomains() map[string]struct{} {
	return map[string]struct{}{"US": {}, "CA": {}}
}

// channelStatus2GHz returns the regulatory verdict for 2.4 GHz channel ch under
// the ISO-3166 alpha-2 domain `country`.
func channelStatus2GHz(country string, ch int) chanStatus {
	switch {
	case country == "JP":
		// Japan permits 1-14 (14 for 11b only; we cannot see modulation, so we
		// do not flag it here).
		if ch >= 1 && ch <= channel2GHz14 {
			return chanAllowed
		}
		return chanForbidden
	case isUSFCC(country):
		// FCC: channels 1-11 only; 12-14 are violations.
		if ch >= 1 && ch <= 11 {
			return chanAllowed
		}
		return chanForbidden
	case ch == channel2GHz14:
		// Channel 14 is JP-only everywhere else — a confident violation.
		return chanForbidden
	default:
		// No country-specific rule we are confident about; do not guess.
		return chanUnknown
	}
}

func isUSFCC(country string) bool {
	_, ok := usFCCDomains()[country]
	return ok
}
