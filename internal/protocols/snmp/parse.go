package snmp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseInterfaceStatus converts SNMP interface status value to string.
func parseInterfaceStatus(value string) string {
	switch value {
	case "1":
		return StatusUp
	case "2":
		return StatusDown
	case "3":
		return StatusTesting
	default:
		return StatusUnknown
	}
}

// parseDuplexStatus converts SNMP duplex status value to string.
func parseDuplexStatus(value string) string {
	switch value {
	case "1":
		return StatusUnknown
	case "2":
		return "half"
	case "3":
		return "full"
	default:
		return StatusUnknown
	}
}

// parseMACStatus converts SNMP MAC status value to entry type.
func parseMACStatus(value string) string {
	switch value {
	case "1":
		return MACTypeOther
	case "2":
		return MACTypeLearned // invalid
	case "3":
		return MACTypeLearned
	case "4":
		return MACTypeStatic
	case "5":
		return MACTypeStatic // mgmt
	default:
		return MACTypeOther
	}
}

// parseMACFromOID extracts MAC address from OID suffix.
// Expects 6 octets in the last parts of the OID.
func parseMACFromOID(parts []string) string {
	if len(parts) < macOctetCount {
		return ""
	}

	mac := make([]string, macOctetCount)
	for i := range macOctetCount {
		octet, err := strconv.Atoi(parts[i])
		if err != nil || octet < 0 || octet > 255 {
			return ""
		}
		mac[i] = fmt.Sprintf("%02x", octet) // #nosec G602 -- i is bounded by range [0,5]
	}

	return strings.Join(mac, ":")
}

// parseTimeTicks converts SNMP TimeTicks to a timestamp.
// TimeTicks are in hundredths of a second since system startup.
func parseTimeTicks(value string) time.Time {
	ticks, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}
	}

	// Convert hundredths of a second to duration.
	duration := time.Duration(ticks) * timeTicksToMilliseconds * time.Millisecond

	// Return time relative to now (approximation).
	return time.Now().Add(-duration)
}

// portListContainsPort checks if a port list bitmap contains the specified port.
// Port lists are byte arrays where each bit represents a port.
func portListContainsPort(portList any, portNum int) bool {
	bytes, ok := portList.([]byte)
	if !ok {
		return false
	}

	// Port numbering starts at 1, bitmap starts at 0.
	portIdx := portNum - 1
	if portIdx < 0 {
		return false
	}

	byteIdx := portIdx / bitsPerByte
	bitIdx := highBitIndex - (portIdx % bitsPerByte) // #nosec G115 -- portIdx%8 is always 0-7

	if byteIdx >= len(bytes) {
		return false
	}

	return (bytes[byteIdx] & (1 << bitIdx)) != 0
}

// Interface counter OIDs for bandwidth monitoring (IF-MIB).
const (
	OIDIfInOctets    = "1.3.6.1.2.1.2.2.1.10"    // ifInOctets (32-bit)
	OIDIfOutOctets   = "1.3.6.1.2.1.2.2.1.16"    // ifOutOctets (32-bit)
	OIDIfInErrors    = "1.3.6.1.2.1.2.2.1.14"    // ifInErrors
	OIDIfOutErrors   = "1.3.6.1.2.1.2.2.1.20"    // ifOutErrors
	OIDIfInDiscards  = "1.3.6.1.2.1.2.2.1.13"    // ifInDiscards
	OIDIfOutDiscards = "1.3.6.1.2.1.2.2.1.19"    // ifOutDiscards
	OIDIfHCInOctets  = "1.3.6.1.2.1.31.1.1.1.6"  // ifHCInOctets (64-bit, ifXTable)
	OIDIfHCOutOctets = "1.3.6.1.2.1.31.1.1.1.10" // ifHCOutOctets (64-bit, ifXTable)
)
