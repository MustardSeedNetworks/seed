// Package snmp exports internal functions for testing.
package snmp

import "github.com/gosnmp/gosnmp"

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
