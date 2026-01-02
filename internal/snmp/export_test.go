// Package snmp exports internal functions for testing.
package snmp

import "github.com/gosnmp/gosnmp"

// FormatSNMPValue exports formatSNMPValue for testing.
var FormatSNMPValue = formatSNMPValue

// GetAuthProtocol exports getAuthProtocol for testing.
var GetAuthProtocol = func(protocol string) gosnmp.SnmpV3AuthProtocol {
	return getAuthProtocol(protocol)
}

// GetPrivProtocol exports getPrivProtocol for testing.
var GetPrivProtocol = func(protocol string) gosnmp.SnmpV3PrivProtocol {
	return getPrivProtocol(protocol)
}
