package arp

// IP-MIB::ipNetToPhysicalTable (1.3.6.1.2.1.4.35.1, RFC 4293).
//
// ipNetToMediaTable is IPv4-only: its row index ends in four dotted octets
// with nowhere to put a longer address. RFC 4293 replaced it with a table
// indexed by an InetAddressType and a length-prefixed InetAddress, which is
// what makes an IPv6 neighbour expressible at all (#1371).
//
// The two tables answer the same question, so a device that implements both
// would otherwise report every IPv4 binding twice.

import (
	"net/netip"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// ipNetToPhysicalTable column OIDs. The first three columns are index
// components and are not-accessible, so a walk returns columns 4 and up.
const (
	physicalTablePrefix = "1.3.6.1.2.1.4.35.1"
	colPhysPhysAddress  = "4"
	colPhysLastUpdated  = "5"
	colPhysType         = "6"
	colPhysState        = "7"
)

// InetAddressType values used in the row index (RFC 4001).
const (
	inetAddressTypeIPv4 = 1
	inetAddressTypeIPv6 = 2
)

// Address lengths for the two InetAddressType values this table can carry.
const (
	ipv4AddressBytes = 4
	ipv6AddressBytes = 16
)

// MediaTypeLocal is the address the device itself owns. ipNetToPhysicalType
// adds it to the four values ipNetToMediaTable's MediaType already had and
// keeps the rest at the same numbers, so the existing MediaType constants stay
// correct for rows from either table.
const MediaTypeLocal = 5

// ipNetToPhysicalState values (RFC 4293). Zero means the agent did not report
// one, which is the case for every row that came from ipNetToMediaTable --
// that table has no equivalent column, so state is genuinely unknown rather
// than "unknown" in the enum's sense.
const (
	StateReachable  = 1
	StateStale      = 2
	StateDelay      = 3
	StateProbe      = 4
	StateInvalid    = 5
	StateUnknown    = 6
	StateIncomplete = 7
)

// buildPhysicalEntries folds an ipNetToPhysicalTable walk into Entries.
func buildPhysicalEntries(vbs []snmp.Varbind) []Entry {
	rows := make(map[rowKey]*Entry)

	for _, vb := range vbs {
		col, key, ok := parsePhysicalRowOID(vb.OID)
		if !ok {
			continue
		}
		entry := rows[key]
		if entry == nil {
			entry = &Entry{IfIndex: key.ifIndex, IPAddress: key.ip}
			rows[key] = entry
		}
		applyPhysicalColumn(entry, col, vb.Value)
	}

	out := make([]Entry, 0, len(rows))
	for _, entry := range rows {
		out = append(out, *entry)
	}
	sortEntries(out)
	return out
}

// parsePhysicalRowOID splits an OID under physicalTablePrefix into its column
// number and row key.
//
// The index is ifIndex, then an InetAddressType, then an InetAddress carried
// as an explicit length followed by that many octets. The length is what makes
// the index parseable at all: without it a 16-octet IPv6 address and a
// four-octet IPv4 one would be indistinguishable from a longer OID suffix.
func parsePhysicalRowOID(oid string) (string, rowKey, bool) {
	if !strings.HasPrefix(oid, physicalTablePrefix+".") {
		return "", rowKey{}, false
	}
	parts := strings.Split(strings.TrimPrefix(oid, physicalTablePrefix+"."), ".")

	// column, ifIndex, addressType, addressLength, then the address itself.
	const fixedFields = 4
	if len(parts) < fixedFields+1 {
		return "", rowKey{}, false
	}

	ifIndex, err := parseUint32(parts[1])
	if err != nil {
		return "", rowKey{}, false
	}
	addrType, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", rowKey{}, false
	}
	addrLen, err := strconv.Atoi(parts[3])
	if err != nil {
		return "", rowKey{}, false
	}
	if !addressLengthMatchesType(addrType, addrLen) {
		return "", rowKey{}, false
	}
	if len(parts) != fixedFields+addrLen {
		return "", rowKey{}, false
	}

	address, ok := addressFromOIDOctets(parts[fixedFields:])
	if !ok {
		return "", rowKey{}, false
	}
	return parts[0], rowKey{ifIndex: ifIndex, ip: address}, true
}

// addressLengthMatchesType rejects a row whose declared length disagrees with
// its address family. An agent that emits one is producing an index nothing
// can index by, and guessing which field to believe would invent a binding.
func addressLengthMatchesType(addrType, addrLen int) bool {
	switch addrType {
	case inetAddressTypeIPv4:
		return addrLen == ipv4AddressBytes
	case inetAddressTypeIPv6:
		return addrLen == ipv6AddressBytes
	default:
		// DNS names and zone-scoped forms are legal InetAddressTypes but
		// cannot appear as a neighbour's address here.
		return false
	}
}

// addressFromOIDOctets rebuilds an address from its per-octet OID components.
func addressFromOIDOctets(octets []string) (string, bool) {
	raw := make([]byte, len(octets))
	for i, part := range octets {
		value, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return "", false
		}
		raw[i] = byte(value)
	}

	address, ok := netip.AddrFromSlice(raw)
	if !ok {
		return "", false
	}
	return address.String(), true
}

// applyPhysicalColumn records one column value on a row.
func applyPhysicalColumn(entry *Entry, col string, value any) {
	switch col {
	case colPhysPhysAddress:
		entry.MACAddress = macAddressString(value)
	case colPhysType:
		entry.MediaType = intValue(value)
	case colPhysState:
		entry.State = intValue(value)
	case colPhysLastUpdated:
		// A sysUpTime-relative timestamp, meaningless without the target's
		// uptime, which this collector does not fetch. Deliberately dropped
		// rather than stored as a number that reads like a wall clock.
	}
}

// mergeEntries combines rows from the two tables, preferring
// ipNetToPhysicalTable where both describe the same binding.
//
// A device may implement both, and they answer the same question, so the
// bindings would otherwise appear twice. RFC 4293 supersedes RFC 1213 here and
// carries strictly more -- the neighbour state, and IPv6 at all -- so it wins.
func mergeEntries(physical, media []Entry) []Entry {
	seen := make(map[rowKey]bool, len(physical))
	out := make([]Entry, 0, len(physical)+len(media))

	for _, entry := range physical {
		seen[rowKey{ifIndex: entry.IfIndex, ip: entry.IPAddress}] = true
		out = append(out, entry)
	}
	for _, entry := range media {
		if seen[rowKey{ifIndex: entry.IfIndex, ip: entry.IPAddress}] {
			continue
		}
		out = append(out, entry)
	}

	sortEntries(out)
	return out
}

// hasIPv4 reports whether any row carries an IPv4 address, which decides
// whether the legacy table is worth walking.
func hasIPv4(entries []Entry) bool {
	for _, entry := range entries {
		if address, err := netip.ParseAddr(entry.IPAddress); err == nil && address.Is4() {
			return true
		}
	}
	return false
}
