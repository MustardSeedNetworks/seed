package snmp

import (
	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
)

// ExportFormatSNMPValue exports formatSNMPValue for testing.
func ExportFormatSNMPValue(variable gosnmp.SnmpPDU) string {
	return formatSNMPValue(variable)
}

// ExportGetAuthProtocol exports getAuthProtocol for testing.
func ExportGetAuthProtocol(protocol string) gosnmp.SnmpV3AuthProtocol {
	return getAuthProtocol(protocol)
}

// ExportGetPrivProtocol exports getPrivProtocol for testing.
func ExportGetPrivProtocol(protocol string) gosnmp.SnmpV3PrivProtocol {
	return getPrivProtocol(protocol)
}

// ExportGetMaxRepetitions exports getMaxRepetitions for testing.
func ExportGetMaxRepetitions(cfg *config.SNMPConfig) uint32 {
	return getMaxRepetitions(cfg)
}

// ExportParseInterfaceStatus exports parseInterfaceStatus for testing.
func ExportParseInterfaceStatus(value string) string {
	return parseInterfaceStatus(value)
}

// ExportParseDuplexStatus exports parseDuplexStatus for testing.
func ExportParseDuplexStatus(value string) string {
	return parseDuplexStatus(value)
}

// ExportParseMACStatus exports parseMACStatus for testing.
func ExportParseMACStatus(value string) string {
	return parseMACStatus(value)
}

// ExportParseMACFromOID exports parseMACFromOID for testing.
func ExportParseMACFromOID(parts []string) string {
	return parseMACFromOID(parts)
}

// ExportParseTimeTicks exports parseTimeTicks for testing.
func ExportParseTimeTicks(value string) any {
	return parseTimeTicks(value)
}

// ExportPortListContainsPort exports portListContainsPort for testing.
func ExportPortListContainsPort(portList any, portNum int) bool {
	return portListContainsPort(portList, portNum)
}

// ExportNetmaskToPrefix exports netmaskToPrefix for testing.
func ExportNetmaskToPrefix(mask string) int {
	return netmaskToPrefix(mask)
}

// ExportParseIPAddressType exports parseIPAddressType for testing.
func ExportParseIPAddressType(value string) string {
	return parseIPAddressType(value)
}

// ExportParseIPAddressOrigin exports parseIPAddressOrigin for testing.
func ExportParseIPAddressOrigin(value string) string {
	return parseIPAddressOrigin(value)
}

// ExportParseIPAddressStatus exports parseIPAddressStatus for testing.
func ExportParseIPAddressStatus(value string) string {
	return parseIPAddressStatus(value)
}

// ExportFormatIPv6FromOctets exports formatIPv6FromOctets for testing.
func ExportFormatIPv6FromOctets(octets []string) string {
	return formatIPv6FromOctets(octets)
}

// ExportParseIPAddressFromOID exports parseIPAddressFromOID for testing.
func ExportParseIPAddressFromOID(oid string) (string, string) {
	return parseIPAddressFromOID(oid)
}

// ExportParseRouteType exports parseRouteType for testing.
func ExportParseRouteType(value string) string {
	return parseRouteType(value)
}

// ExportParseRouteProtocol exports parseRouteProtocol for testing.
func ExportParseRouteProtocol(value string) string {
	return parseRouteProtocol(value)
}

// ExportParseIPCidrRouteIndex exports parseIPCidrRouteIndex for testing.
func ExportParseIPCidrRouteIndex(oid string) (string, string, string) {
	return parseIPCidrRouteIndex(oid)
}

// ExportParseInetCidrRouteIndex exports parseInetCidrRouteIndex for testing.
func ExportParseInetCidrRouteIndex(oid string) (string, int, string) {
	return parseInetCidrRouteIndex(oid)
}

// ExportExtractLLDPIndex exports extractLLDPIndex for testing.
func ExportExtractLLDPIndex(oid string) (int, int) {
	return extractLLDPIndex(oid)
}

// ExportFormatChassisID exports formatChassisID for testing.
func ExportFormatChassisID(value any) string {
	return formatChassisID(value)
}

// ExportIsPrintable exports isPrintable for testing.
func ExportIsPrintable(data []byte) bool {
	return isPrintable(data)
}

// ExportParseChassisIDSubtype exports parseChassisIDSubtype for testing.
func ExportParseChassisIDSubtype(value string) string {
	return parseChassisIDSubtype(value)
}

// ExportParsePortIDSubtype exports parsePortIDSubtype for testing.
func ExportParsePortIDSubtype(value string) string {
	return parsePortIDSubtype(value)
}

// ExportExtractVLANIndex exports extractVLANIndex for testing.
func ExportExtractVLANIndex(oid string) int {
	return extractVLANIndex(oid)
}

// ExportParsePortBitmap exports parsePortBitmap for testing.
func ExportParsePortBitmap(value any) []int {
	return parsePortBitmap(value)
}

// ExportParseRowStatus exports parseRowStatus for testing.
func ExportParseRowStatus(value string) string {
	return parseRowStatus(value)
}

// ExportExtractEntityIndex exports extractEntityIndex for testing.
func ExportExtractEntityIndex(oid string) int {
	return extractEntityIndex(oid)
}

// ExportParseEntityClass exports parseEntityClass for testing.
func ExportParseEntityClass(value string) string {
	return parseEntityClass(value)
}

// ExportParseVLANAndMAC exports parseVLANAndMAC for testing.
// Adds a guard for minimum OID parts to avoid panic.
func ExportParseVLANAndMAC(parts []string) (int, string, bool) {
	if len(parts) < minOIDPartsQBridge {
		return 0, "", false
	}
	return parseVLANAndMAC(parts)
}

// ExportParseBridgePort exports parseBridgePort for testing.
func ExportParseBridgePort(pdu gosnmp.SnmpPDU) (int, bool) {
	return parseBridgePort(pdu)
}

// ExportCollectMACEntries exports collectMACEntries for testing.
func ExportCollectMACEntries(macToEntry map[string]*MACEntry) []MACEntry {
	return collectMACEntries(macToEntry)
}
