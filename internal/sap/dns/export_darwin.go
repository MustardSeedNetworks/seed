//go:build darwin

package dns

// ExportParseResolvConfDarwin is exported for testing (darwin only).
func ExportParseResolvConfDarwin(path string) []string {
	return parseResolvConfDarwin(path)
}

// ExportGetDNSFromInterfaces is exported for testing (darwin only).
func ExportGetDNSFromInterfaces() []string {
	return getDNSFromInterfaces()
}
