// Package indicators matches flow endpoints against the operator's list of
// hostile addresses: single IPv4 or IPv6 addresses and prefixes, as threat
// feeds publish them.
//
// Seed ships no list and fetches none. The operator supplies it, so an
// air-gapped install never reaches out because of this package.
//
// Only non-internal addresses are matched. A private, loopback, link-local,
// shared (CGNAT), multicast or reserved address is a host on some local
// network, never a hostile service on the internet, so the list refuses
// entries that reach into those ranges rather than accepting entries that
// can never match.
package indicators

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"
)

// maxEntries bounds a list. It is operator input stored in the database and
// checked against every flow; the common published blocklists hold a few
// thousand entries.
const maxEntries = 10000

// internalRanges is every range whose addresses belong to a local network
// rather than the internet. [New] parses it once per list.
func internalRanges() []netip.Prefix {
	ranges := []string{
		"0.0.0.0/8",      // "this network"
		"10.0.0.0/8",     // RFC 1918
		"100.64.0.0/10",  // shared address space (CGNAT)
		"127.0.0.0/8",    // loopback
		"169.254.0.0/16", // link-local
		"172.16.0.0/12",  // RFC 1918
		"192.168.0.0/16", // RFC 1918
		"224.0.0.0/3",    // multicast, reserved and broadcast
		"::/127",         // unspecified and loopback
		"fc00::/7",       // unique local
		"fe80::/10",      // link-local
		"ff00::/8",       // multicast
	}
	out := make([]netip.Prefix, len(ranges))
	for i, r := range ranges {
		out[i] = netip.MustParsePrefix(r)
	}
	return out
}

// List is a validated indicator list. It is immutable once built.
type List struct {
	entries []netip.Prefix
	// byBits indexes the entries by prefix length, and bits holds those
	// lengths longest first, so a lookup costs one map probe per distinct
	// length and the first hit is the most specific entry.
	byBits map[int]map[netip.Prefix]struct{}
	bits   []int
}

// New validates entries and builds a list. An entry is an address
// ("198.51.100.7") or a prefix ("203.0.113.0/24") with no host bits set.
//
// Errors identify an entry by position and never quote it: the list is
// operator input, and its text must not reach a log line or a response
// verbatim.
func New(entries []string) (*List, error) {
	if len(entries) > maxEntries {
		return nil, fmt.Errorf("list has %d entries, limit %d", len(entries), maxEntries)
	}
	internal := internalRanges()
	l := &List{byBits: map[int]map[netip.Prefix]struct{}{}}
	for i, s := range entries {
		p, err := parseEntry(s, internal)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
		if _, dup := l.byBits[p.Bits()][p]; dup {
			return nil, fmt.Errorf("entry %d repeats an earlier entry", i)
		}
		if l.byBits[p.Bits()] == nil {
			l.byBits[p.Bits()] = map[netip.Prefix]struct{}{}
			l.bits = append(l.bits, p.Bits())
		}
		l.byBits[p.Bits()][p] = struct{}{}
		l.entries = append(l.entries, p)
	}
	slices.SortFunc(l.bits, func(a, b int) int { return b - a })
	return l, nil
}

func parseEntry(s string, internal []netip.Prefix) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(s)
	if err != nil {
		a, addrErr := netip.ParseAddr(s)
		if addrErr != nil || a.Zone() != "" {
			return netip.Prefix{}, errors.New("must be an IPv4 or IPv6 address or prefix")
		}
		p = netip.PrefixFrom(a, a.BitLen())
	}
	if p.Addr().Is4In6() {
		return netip.Prefix{}, errors.New("must be written as IPv4, not as an IPv4-mapped IPv6 address")
	}
	if p.Masked() != p {
		return netip.Prefix{}, errors.New("prefix has host bits set")
	}
	for _, in := range internal {
		if in.Overlaps(p) {
			return netip.Prefix{}, errors.New(
				"reaches into a private, loopback, link-local, shared, multicast or reserved range, which is never matched",
			)
		}
	}
	return p, nil
}

// Len is the number of entries.
func (l *List) Len() int { return len(l.entries) }

// Entries returns the entries in the order they were given, each in its
// canonical form: a single address keeps its full-length prefix.
func (l *List) Entries() []string {
	out := make([]string, len(l.entries))
	for i, p := range l.entries {
		if p.IsSingleIP() {
			out[i] = p.Addr().String()
		} else {
			out[i] = p.String()
		}
	}
	return out
}

// Match returns the most specific entry containing a. An internal address
// never matches, because no entry reaches into internal space.
func (l *List) Match(a netip.Addr) (netip.Prefix, bool) {
	a = a.Unmap()
	for _, bits := range l.bits {
		if bits > a.BitLen() {
			continue
		}
		p := netip.PrefixFrom(a, bits).Masked()
		if _, ok := l.byBits[bits][p]; ok {
			return p, true
		}
	}
	return netip.Prefix{}, false
}

// document is the JSON shape of a stored or submitted list.
type document struct {
	Indicators []string `json:"indicators"`
}

// Parse decodes and validates a list in its JSON form.
func Parse(data []byte) (*List, error) {
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, errors.New("indicator list is not valid JSON of the documented shape")
	}
	return New(doc.Indicators)
}

// MarshalJSON writes the list in the form [Parse] reads.
func (l *List) MarshalJSON() ([]byte, error) {
	return json.Marshal(document{Indicators: l.Entries()})
}
