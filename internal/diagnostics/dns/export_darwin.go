//go:build darwin

package dns

// ExportGetDNSFromInterfaces is exported for testing (darwin only).
func ExportGetDNSFromInterfaces() []string {
	return getDNSFromInterfaces()
}
