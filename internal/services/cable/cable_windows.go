//go:build windows

// Windows-specific cable diagnostics implementation.
// TDR (Time Domain Reflectometry) cable testing on Windows requires vendor-specific
// drivers and tools. This file provides stubs with informative error messages.
//
// Platform limitations:
//   - Windows doesn't expose TDR through standard APIs
//   - Intel NICs: Use Intel PROSet/Wireless Software for cable diagnostics
//   - Broadcom NICs: Use Broadcom Advanced Control Suite (BACS)
//   - Realtek NICs: Limited/no TDR support
//   - Most consumer NICs don't support cable diagnostics at all
package cable

// isSupportedPlatform checks if cable diagnostics are supported on Windows.
// Returns false as Windows doesn't have a standard API for TDR.
func isSupportedPlatform(iface string) bool {
	// Windows doesn't provide standard TDR API access
	// Enterprise NICs from Intel/Broadcom have proprietary tools
	return false
}

// testPlatform performs cable test on Windows.
// This returns a result indicating TDR is not supported through standard APIs.
func testPlatform(iface string) *TestResult {
	return &TestResult{
		Supported: false,
		Status:    StatusUnknown,
		Faults: []string{
			"Cable diagnostics (TDR) on Windows are not available through standard APIs.",
			"To perform cable testing, use vendor-specific tools:",
			"  - Intel NICs: Intel PROSet/Wireless Software",
			"  - Broadcom NICs: Broadcom Advanced Control Suite (BACS)",
			"  - Marvell NICs: Marvell Yukon Device Manager",
			"Most consumer NICs (Realtek, etc.) don't support TDR cable testing.",
			"See HARDWARE.md for compatible hardware recommendations.",
		},
		WiringStd: Wiring568B,
		Pinout:    Get568BPinout(),
	}
}

// GetSupportedNICs returns a list of NICs known to support TDR on Windows.
// This is informational only - actual support depends on driver version.
func GetSupportedNICs() []string {
	return []string{
		"Intel I210/I211 (with Intel PROSet)",
		"Intel I350 (with Intel PROSet)",
		"Intel I225-V/LM (with Intel PROSet)",
		"Broadcom BCM5719/5720 (with BACS)",
		"Broadcom BCM57810 (with BACS)",
		"Marvell Yukon 88E8056/8053 (with Yukon Tools)",
	}
}
