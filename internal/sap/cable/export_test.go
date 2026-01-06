package cable

// TesterInterfaceName returns the interface name for testing.
func (t *Tester) TesterInterfaceName() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.interfaceName
}

// ExportIsSupportedPlatform is exported for testing.
func ExportIsSupportedPlatform(iface string) bool {
	return isSupportedPlatform(iface)
}

// ExportTestPlatform is exported for testing.
func ExportTestPlatform(iface string) *TestResult {
	return testPlatform(iface)
}
