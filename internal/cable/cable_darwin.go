//go:build darwin

// Package cable provides TDR cable testing functionality.
// macOS stub implementation provides no TDR support as macOS does not expose
// Time Domain Reflectometry functionality through standard APIs.
package cable

// isSupportedPlatform checks if the NIC supports TDR on macOS.
// macOS does not expose TDR functionality through standard APIs.
func isSupportedPlatform(iface string) bool {
	// TDR is not available on macOS
	return false
}

// testPlatform performs a cable test on macOS.
// Since macOS doesn't support TDR, return unsupported result.
func testPlatform(iface string) *TestResult {
	return &TestResult{
		Supported: false,
		Status:    StatusUnknown,
		Faults:    []string{"Cable testing not supported on macOS"},
	}
}
