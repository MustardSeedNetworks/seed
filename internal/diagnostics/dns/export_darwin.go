//go:build darwin

package dns

// ExportScopedResolversFor is exported for testing (darwin only).
func ExportScopedResolversFor(table, iface string) ([]string, bool) {
	return scopedResolversFor(table, iface)
}
