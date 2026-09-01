//go:build linux || darwin

package dns

// ExportParseResolvConf is exported for testing (linux and darwin only).
func ExportParseResolvConf(path string) []string {
	return parseResolvConf(path)
}
