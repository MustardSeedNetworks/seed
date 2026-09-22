//go:build darwin

package dns

// ExportParseScutilScopedResolvers is exported for testing (darwin only).
func ExportParseScutilScopedResolvers(out, iface string) ([]string, bool) {
	return parseScutilScopedResolvers(out, iface)
}
