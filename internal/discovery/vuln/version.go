package vuln

import (
	"strings"
	"unicode"
)

// Version extraction for CPE lookups.
//
// Every parser in this package used to carry its own version regex, and each
// one was subtly different and subtly wrong: one character class held an
// uppercase X while its input had already been lowercased, another could not
// cross a second dot so "16.9.4" arrived as "16.9", a third left a trailing
// dot. Worse, each only fired on the one keyword its author had in mind, so a
// device whose vendor wrote "firmware" instead of "Version" produced no version
// at all — and an empty version means [VulnerabilityScanner.ScanDevice] gives
// up before it ever queries for CVEs.
//
// The rules below are derived from the 173 real sysDescr strings in
// testdata/sysdescr-corpus.txt, not invented, and TestExtractVersionOnCorpus
// measures them against it.

// versionKeywords returns the words vendors put immediately before a version.
// Ordered by how specific they are: a string containing both "junos" and
// "version" means the JunOS release, not whatever "version" introduces.
//
// A function rather than a package variable, matching getSNMPVendorMatchers.
func versionKeywords() []string {
	return []string{
		"junos",
		"arubaos-cx",
		"arubaos",
		"ironware version",
		"ros version",
		"revision",
		"firmware",
		"version",
		"release",
	}
}

// maxVersionTokenLen bounds what will be accepted as a version. Real versions
// in the corpus top out around twenty characters; anything much longer is a
// build string or a path that happens to contain dots.
const maxVersionTokenLen = 32

// maxVersionAlphaPrefix bounds the leading letters a version may carry. HP and
// Aruba use "FL.16.11.0009" and "E.10.74"; nothing in the corpus uses more than
// two, and allowing more would admit words like "build-1331820".
const maxVersionAlphaPrefix = 2

// minDateLen is the length of the shortest build date worth rejecting,
// "1990-1-1" — anything shorter cannot be one.
const minDateLen = 8

// isVersionToken reports whether tok has the shape of a software version.
//
// The test is deliberately structural rather than a pattern per vendor: at
// least one digit, at least one dot, nothing but characters versions actually
// use, and not a calendar date. That admits "17.6.3", "FL.10.10.1010",
// "12.1X47-D15.4", "08.0.10dT311" and "2.4.21-57.ELvmnix" — which between them
// cover every vendor in the corpus — without a rule for each.
func isVersionToken(tok string) bool {
	if tok == "" || len(tok) > maxVersionTokenLen {
		return false
	}

	var digits, dots int
	for _, r := range tok {
		switch {
		case unicode.IsDigit(r):
			digits++
		case r == '.':
			dots++
		case unicode.IsLetter(r), r == '-', r == '_', r == '(', r == ')', r == '/':
		default:
			return false
		}
	}
	if digits == 0 || dots == 0 {
		return false
	}

	// A version starts with a digit or a short alphabetic prefix.
	prefix := 0
	for _, r := range tok {
		if !unicode.IsLetter(r) {
			break
		}
		prefix++
	}
	if prefix > maxVersionAlphaPrefix {
		return false
	}

	return !looksLikeDate(tok)
}

// looksLikeDate rejects the build dates that sit next to versions in vendor
// prose — "2021-10-15" would otherwise satisfy every rule above.
func looksLikeDate(tok string) bool {
	if len(tok) < minDateLen {
		return false
	}
	year := tok[:4]
	for _, r := range year {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	// 1990-2099 is generous enough for a build date and narrow enough that no
	// real version number falls inside it.
	return (strings.HasPrefix(year, "19") || strings.HasPrefix(year, "20")) &&
		(tok[4] == '-' || tok[4] == '/')
}

// trimVersion strips the punctuation vendor prose leaves attached to a version:
// the comma in "Version 17.6.3, RELEASE SOFTWARE", the full stop ending a
// sentence, and any wrapping parentheses.
func trimVersion(tok string) string {
	tok = strings.Trim(tok, "(),;:\"'")
	return strings.TrimRight(tok, ".")
}

// extractVersion returns the software version named in s, or "" if none is.
//
// Anchored on a keyword when one is present, because that is where vendors put
// it and an anchored match beats the first dotted token in the string — which
// is frequently a model number. Falls back to the first version-shaped token so
// a vendor with no keyword at all still yields something.
func extractVersion(s string) string {
	lower := strings.ToLower(s)

	for _, kw := range versionKeywords() {
		idx := strings.Index(lower, kw)
		if idx < 0 {
			continue
		}
		if v := versionAfterKeyword(s[idx+len(kw):]); v != "" {
			return v
		}
	}

	return firstVersionToken(s)
}

// firstVersionToken returns the first token in s that is a version outright.
// Strict, because with no keyword in front of it there is nothing to say a
// dotted token is a version rather than a model number or an IP address.
func firstVersionToken(s string) string {
	for field := range strings.FieldsSeq(s) {
		if candidate := trimVersion(field); isVersionToken(candidate) {
			return candidate
		}
	}
	return ""
}

// versionAfterKeyword returns the version in s, which begins immediately after
// a version keyword.
//
// More forgiving than firstVersionToken, because the vendor has already told us
// what follows is a version. Cisco's development images append build metadata
// to the release — "15.2(CML_NIGHTLY_20151103)FLO_DSGS7", or
// "15.1(20130726:213425)" with a colon in it — which no reasonable token rule
// accepts whole. The release prefix is the part a CPE lookup matches on, so
// take that rather than returning nothing and skipping the CVE scan entirely.
func versionAfterKeyword(s string) string {
	fields := strings.Fields(s)
	for i, field := range fields {
		candidate := trimVersion(field)
		if isVersionToken(candidate) {
			return candidate
		}
		if core := leadingReleaseNumber(candidate); core != "" {
			return core
		}
		// Only the first couple of words after the keyword can be the version;
		// beyond that we are reading unrelated prose.
		if i >= 1 {
			break
		}
	}
	return ""
}

// leadingReleaseNumber returns the dotted numeric prefix of tok — "15.2" from
// "15.2(CML_NIGHTLY_20151103)FLO_DSGS7" — or "" if tok does not start with one.
func leadingReleaseNumber(tok string) string {
	end := 0
	for end < len(tok) && (tok[end] == '.' || (tok[end] >= '0' && tok[end] <= '9')) {
		end++
	}
	core := strings.TrimRight(tok[:end], ".")
	if !strings.Contains(core, ".") {
		return ""
	}
	return core
}

// containsWord reports whether word appears in s delimited by something other
// than a letter or digit.
//
// [strings.Contains] is wrong here and was actively harmful: it matched "hp"
// inside "ICX6430-24-HPOE", so every Brocade switch in the corpus was reported
// as an HP switch and looked up against the wrong vendor's CVEs.
func containsWord(s, word string) bool {
	if word == "" {
		return false
	}
	for offset := 0; ; {
		idx := strings.Index(s[offset:], word)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(word)
		if !isWordChar(s, start-1) && !isWordChar(s, end) {
			return true
		}
		offset = start + 1
	}
}

// isWordChar reports whether the byte at i is alphanumeric, treating positions
// outside s as delimiters.
func isWordChar(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return false
	}
	r := rune(s[i])
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
