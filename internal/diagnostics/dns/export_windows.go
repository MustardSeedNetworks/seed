//go:build windows

package dns

// ExportParseIPConfigAdapterDNS is exported for testing (windows only).
func ExportParseIPConfigAdapterDNS(out, iface string) ([]string, bool) {
	return parseIPConfigAdapterDNS(out, iface)
}
