//go:build linux

package dns

// ExportParseResolvedLinkDNS is exported for testing (linux only).
func ExportParseResolvedLinkDNS(content string) []string {
	return parseResolvedLinkDNS(content)
}

// ExportParseDNSConfiguration is exported for testing (linux only).
func ExportParseDNSConfiguration(reply []byte, iface string) ([]string, error) {
	return parseDNSConfiguration(reply, iface)
}
