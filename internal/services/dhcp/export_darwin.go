//go:build darwin

package dhcp

// ExportParseIPConfigOutput is exported for testing (darwin only).
func ExportParseIPConfigOutput(output string, result *TestResult) {
	parseIPConfigOutput(output, result)
}

// ExportParseLeaseTime is exported for testing (darwin only).
func ExportParseLeaseTime(val string) (int, error) {
	return parseLeaseTime(val)
}

// ExportParseDHCPLine is exported for testing (darwin only).
func ExportParseDHCPLine(line string, result *TestResult) {
	parseDHCPLine(line, result)
}
