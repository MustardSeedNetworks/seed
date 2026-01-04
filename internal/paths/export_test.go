// Package paths exports internal functions for testing.
package paths

// ExportIsSystemdService exports isSystemdService for testing.
func ExportIsSystemdService() bool {
	return isSystemdService()
}
