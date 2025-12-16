//go:build !linux

// Package network handles network interface management.
// ethtool stub implementation for non-Linux platforms provides no-op implementations
// as ethtool is Linux-specific. macOS and other platforms use alternative approaches.
package network

// getEthtoolSettings is a stub for non-Linux platforms.
// Ethtool functionality is only available on Linux.
func getEthtoolSettings(_ string) (autoNeg bool, advertised []string) {
	return false, nil
}
