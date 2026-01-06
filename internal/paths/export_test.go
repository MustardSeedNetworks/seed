package paths

// ExportIsSystemdService exports isSystemdService for testing.
func ExportIsSystemdService() bool {
	return isSystemdService()
}
