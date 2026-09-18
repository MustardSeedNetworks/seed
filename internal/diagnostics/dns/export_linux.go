//go:build linux

package dns

// ExportParseResolvedLinkDNS is exported for testing (linux only).
func ExportParseResolvedLinkDNS(content string) []string {
	return parseResolvedLinkDNS(content)
}
