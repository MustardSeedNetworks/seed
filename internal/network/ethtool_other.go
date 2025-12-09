//go:build !linux

package network

// getEthtoolSettings is a stub for non-Linux platforms.
// Ethtool functionality is only available on Linux.
func getEthtoolSettings(name string) (autoNeg bool, advertised []string) {
	return false, nil
}
