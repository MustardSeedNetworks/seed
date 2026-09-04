//go:build windows

package phy

// Windows-specific PHY layer implementation.
//
// Platform limitations:
//   - Windows doesn't expose PoE or SFP/DDM information through standard APIs;
//     both require vendor-specific tools (Intel PROSet, Broadcom BACS, Mellanox
//     WinOF, Marvell Yukon Device Manager), so both stubs report "not present"
//     rather than attempting a query that cannot succeed.

// getPoEStatus detects PoE power status on Windows.
// Windows doesn't expose PoE information through standard APIs.
func getPoEStatus(_ string) *PoEStatus {
	// PoE detection on Windows requires vendor-specific tools
	// Most NICs don't expose this information to the OS
	return &PoEStatus{
		Detected: false,
	}
}

// getSFPInfo reads SFP module info on Windows.
// Windows doesn't expose SFP/DDM information through standard APIs.
func getSFPInfo(_ string) *SFPInfo {
	// SFP/DDM information requires vendor-specific tools on Windows:
	// - Intel: Intel PROSet
	// - Mellanox: WinOF
	// - Broadcom: BACS
	return &SFPInfo{
		Present:    false,
		DDMSupport: false,
	}
}
